
function handle_cmd(info)
        local commond = info.Cmd
        local s, ok = cmd(commond)
        SendBack(MsgPack.pack({str=s, ok = ok}))
end

