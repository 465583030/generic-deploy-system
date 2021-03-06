package server

import "flag"
import "log"
import "net"
import "sync"
import "time"
import "net/url"
import "../pipe"
import "os"
import "golang.org/x/net/websocket"
import "github.com/yuin/gopher-lua"
import "path/filepath"

var service = flag.String("service", "127.0.0.1:8888", "for remote client connect")
var webservice = flag.String("web", "127.0.0.1:8080", "http server port")

type sessionInfo struct {
        conn net.Conn
        quitC chan bool
        id int
        client *ClientT
}

type ClientT struct {
        sessionTbl map[int]*sessionInfo
        sessionLock sync.RWMutex
}

func (c *ClientT) Init() {
        c.sessionTbl = make(map[int]*sessionInfo)
}

func (c *ClientT) AddSession(conn net.Conn) *sessionInfo {
        id := genId()
        s := &sessionInfo{conn, make(chan bool), id, c}
        c.sessionLock.Lock()
        c.sessionTbl[id] = s
        c.sessionLock.Unlock()
        return s
}

func (c *ClientT) DelSession(id int) {
        c.sessionLock.Lock()
        s, b := c.sessionTbl[id]
        if b {
                close(s.quitC)
                delete(c.sessionTbl, id)
        }
        c.sessionLock.Unlock()
}

func (c *ClientT) OnRemove() {
        for id, _ := range c.sessionTbl {
                c.DelSession(id)
        }
}

type ClientTblT struct {
	tbl map[*websocket.Conn]*ClientT
	action2Session map[string](map[*sessionInfo]bool)
	actionLock sync.RWMutex
	sync.RWMutex
}

func (c *ClientTblT) CancelAction(action string) {
	c.actionLock.RLock()
	tbl, b := c.action2Session[action]
        if b {
                for session, _ := range tbl {
                        log.Println("cancel", action, session.id)
                        pipe.Send(session.conn, pipe.CancelRequest, pipe.RequestCmd{session.id, uint(0), ""})
                        session.client.DelSession(session.id)
                }
        }
	c.actionLock.RUnlock()
}

func (c *ClientTblT) AddAction(action string, session *sessionInfo) {
	c.actionLock.Lock()
	tbl, b := c.action2Session[action]
        if !b {
                tbl = make(map[*sessionInfo]bool)
                c.action2Session[action] = tbl
        }
        tbl[session] = true
	c.actionLock.Unlock()
}

func (c *ClientTblT) RemoveAction(action string, session *sessionInfo) {
	c.actionLock.Lock()
	tbl, b := c.action2Session[action]
        if b {
                delete(tbl, session)
        }
	if b && len(tbl) <= 0 {
		delete(c.action2Session, action)
	}
	c.actionLock.Unlock()
}

func (c *ClientTblT) HasActionSession(action string) int {
        n := 0
        c.actionLock.RLock()
	tbl, b := c.action2Session[action]
        if b {
                n = len(tbl)
        }
        c.actionLock.RUnlock()
	return n
}


func (c *ClientTblT) Broadcast(head, b []byte) {
        log.Println("begin Broadcast");
        c.RLock()
        for c, _ :=range c.tbl {
                WSWrite(c, head, b)
        }
        c.RUnlock()
        log.Println("end Broadcast");
}
func (c *ClientTblT) Add(conn *websocket.Conn) *ClientT{
        c.Lock()
        client := &ClientT{}
        client.Init()
        c.tbl[conn] = client
        c.Unlock()
        return client
}
func (c *ClientTblT) Get(conn *websocket.Conn) *ClientT{
        c.RLock()
        _m, _ := c.tbl[conn]
        c.RUnlock()
        return _m
}
func (c *ClientTblT) Del(conn *websocket.Conn) {
        _m := c.Get(conn)
        if _m != nil {
                _m.OnRemove()
        }
        c.Lock()
        delete(c.tbl, conn)
        c.Unlock()
}

type Machine struct {
        Group string
        Nick string
        conn net.Conn
}

type Machines struct {
        Name string
        Tbl map[string]*Machine
        sync.RWMutex
}
func (m *Machines) Add(nick string, conn net.Conn) {
        m.Lock()
        _m := &Machine{Group:m.Name, Nick:nick, conn:conn}
        m.Tbl[nick] = _m
        m.Unlock()
}
func (m *Machines) Empty() bool {
        m.RLock()
        empty := len(m.Tbl)
        m.RUnlock()
        return empty == 0
}
func (m *Machines) Get(nick string) *Machine {
        m.RLock()
        mm, _ := m.Tbl[nick]
        m.RUnlock()
        return mm
}
func (m *Machines) GetAll() []*Machine {
	t:=[]*Machine{}
        m.RLock()
	for _, mm:= range m.Tbl {
		t=append(t, mm)
	}
        m.RUnlock()
        return t
}
func (m *Machines) Del(nick string) {
        m.Lock()
        delete(m.Tbl, nick)
        m.Unlock()
}
type RemoteTblT struct {
        Tbl map[string]*Machines
        conntbl map[net.Conn]*Machine
        sync.RWMutex
}
func (c *RemoteTblT) Add(group, nick string, conn net.Conn) {
        c.Lock()
        g, have := c.Tbl[group]
        if !have {
                g = &Machines{Tbl:make(map[string]*Machine), Name:group}
                c.Tbl[group] = g
        }
        g.Add(nick, conn)
        m:=g.Get(nick)
        c.conntbl[conn]  = m
        c.Unlock()
}
func (c *RemoteTblT) GetMachines(g string) *Machines {
        c.RLock()
        m, _ := c.Tbl[g]
        c.RUnlock()
	return m
}
func (c *RemoteTblT) Get(conn net.Conn) *Machine {
        c.RLock()
        m, _ := c.conntbl[conn]
        c.RUnlock()
	return m
}
func (c *RemoteTblT) Del(conn net.Conn) {
        c.Lock()
        m, have := c.conntbl[conn]
        if have {
                g, _have := c.Tbl[m.Group]
                if _have {
                        g.Del(m.Nick)
                }
		if g.Empty() {
			delete(c.Tbl, m.Group)
		}
                delete(c.conntbl, conn)
        }
        c.Unlock()
}

var ClientTbl *ClientTblT
var RemoteTbl *RemoteTblT
type buttonConfig struct {
        Name string
	Hide bool
}
var LuaActionTbl map[string](map[string]*buttonConfig)
var LuaActionTblLock sync.RWMutex
func Init() {
	ClientTbl = &ClientTblT{}
	ClientTbl.tbl = make(map[*websocket.Conn]*ClientT)
	ClientTbl.action2Session = make(map[string](map[*sessionInfo]bool))
        RemoteTbl  = &RemoteTblT{Tbl:make(map[string]*Machines), conntbl:make(map[net.Conn]*Machine)}
	flag.Parse()
        er:=InitAdminPort(*webservice)
        if er !=nil {
                log.Println("web server fail", er.Error())
                return
	}
	ScanButtons()
        go func() {
                t:=time.NewTicker(time.Minute*time.Duration(10))
                for {
                        select {
                        case <-t.C:
                                ScanButtons()
                        }
                }
        }()
	Listen(*service)
}

func ScanButtons() {
	LuaActionTblLock.Lock()
	defer LuaActionTblLock.Unlock()
	LuaActionTbl = make(map[string](map[string]*buttonConfig))
	filepath.Walk("./logic", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if err != nil {
				log.Println("search button fail", err.Error())
			} else {
				if filepath.Dir(filepath.Dir(path)) == "logic" && filepath.Ext(path) == ".lua" && filepath.Base(filepath.Dir(path)) != "internal" {
					name := info.Name()
					l := len(name)
					groupName := filepath.Base(filepath.Dir(path))
					buttonName := string([]byte(name)[:l-4])
					g, h:=LuaActionTbl[groupName]
					if !h {
						g = make(map[string]*buttonConfig)
						LuaActionTbl[groupName] = g
					}
					config := &buttonConfig{Name:buttonName, Hide:false}
                                        g[buttonName] = config
					log.Println("find button:", groupName, buttonName)
                                        ls := lua.NewState()
                                        ls.SetGlobal("bInit", lua.LBool(true))
                                        if err := ls.DoFile(path); err != nil {
                                                panic(err)
                                        }
                                        if t:= ls.Get(-1); t.Type()==lua.LTTable {
                                                v := t.(*lua.LTable).RawGetString("name")
                                                if v.Type() == lua.LTString {
                                                        s, _:= v.(lua.LString)
                                                        config.Name = url.QueryEscape(string(s))
                                                        log.Println("name", s)
                                                }
                                                v = t.(*lua.LTable).RawGetString("hide")
						if v.Type() == lua.LTBool {
							b, _ := v.(lua.LBool)
							if b {
								config.Hide = bool(b)
							}
						}
                                        }
				}
			}
		}
		return nil
	})
}
/*l := lua.NewState()
lua.OpenLibraries(l)
//RegLuaFunc(l, "add_button", add_button)
if err := lua.DoFile(l, "logic/lua/init.lua"); err != nil {
	panic(err)
}
//callButtonAction(l, "qa2test1")
*/
/*func RegLuaFunc(l *lua.State, name string, f func(l *lua.State) int) {
	l.PushGoFunction(f)
	l.SetGlobal(name)
}

func callButtonAction(l *lua.State, groupNick string) {
	l.Field(lua.RegistryIndex, groupNick)
	if l.IsFunction(-1) {
		l.Call(0, 0)
	} else {
		log.Println("call button action fail:", groupNick)
	}
}

func add_button(l *lua.State) int {
	n := l.Top()
	if n == 3 {
		groupName, _ := l.ToString(1)
		nick, _ := l.ToString(2)

		l.SetField(lua.RegistryIndex, groupName+nick)
		log.Println("add button for", groupName, nick)
	} else {
		log.Println("warning, not enough arg for add_button")
	}
	return 0
}*/
