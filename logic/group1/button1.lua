if bInit then
        return {name="测试1"}
end
if not Single() then return end
--single() check lock, is locked quit
--remote_cmd("ls -l", function(status, recv)
--	print(status, recv)
--end)
--server_upload("/tmp/a.txt", "/home/pangu/a.txt")
--server_download("/home/pangu/a.txt", "/tmp/a.txt")
--
--local_server_download("/tmp/a.txt")
--local_remote_download("/tmp/a.txt", targetNick)
--local_remote_upload("/tmp/a.txt", targetNick) arg1 is target file.this will choose a file from broswer
--local_server_upload("/tmp/a.txt", targetNick)
--local_getinput("ok?")
--local_confirm("ok?")
--local_output("ssss")
--
--global_setmap(key, value)
--global_setmap_ifeq(key, value, old)
--global_getmap(key)
--
--dieafter(10)
--try_dialtimeout(ip+port, 10)
--sleep(10)
--

remote_cmd("sleep 1", function() end)
local endStr = remote_cmd("ls -r", function(aa) 
	local tbl = MsgPack.unpack(aa)
	for a,b in pairs(tbl) do 
                if type(b) == "string" then
                        local_msg(b)
                end
        end
end)

--print("test over", endStr)
