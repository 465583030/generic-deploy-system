通用部署系统

场合：局域网多台多组机器状态查询，环境部署，等

特点：
  二进制程序部署方便
  被管理机器只需部署一个二进制文件和一组内置功能的lua脚本(可自行扩展)
  服务终端提供web管理页面，用户无需编写页面，只需要写简单的lua逻辑脚本，放到终端程序目录下面，会自动在页面上以按钮等形式展现
  lua脚本可与被管理端或者浏览器交互
  h5管理页面，无刷新，实时反馈
  可自行扩展功能，扩展功能也无需重新编译二进制程序，所有内置功能均以lua脚本实现
  每个按钮每次点击均运行在独立线程，不会阻塞系统运行
概念：
  整个系统分为三大部分
      浏览器端  local 由server提供的http服务和web-socket服务
      服务端  server 监听http服务和控制服务端口
      被控制端 remote 连接控制服务端口，自动注册分组和昵称
  
  
  
