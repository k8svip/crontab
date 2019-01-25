package master

import (
	"net"
	"net/http"
	"time"
)

// 任务的http接口
type ApiServer struct {
	httpServer *http.Server
}

// 保存任务接口
func handleJobSave(w http.ResponseWriter, r *http.Request) {

}

var (
	// 单例对象
	G_apiServer *ApiServer
)

// 初始化服务
func InitApiServer(err error) {
	var (
		mux        *http.ServeMux
		listen     net.Listener
		httpServer *http.Server
	)

	// 配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)

	// 启动TCP监听
	if listen, err = net.Listen("tcp", ":8070"); err != nil {
		return
	}

	// 创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler:      mux, //为http服务配置路由，就是这里的handle字段，你会发现与我们这里定义的handleJobSave是一个类型的回调函数；它们的参数是一样的，所谓的路由也是一个handler;
		// 无非是帮我们转发一下，当http.Server收到请求之后，会回调给handler方法，Handler把路由传进去了，路由呢，会根据他内部的URL，遍历这里URL，找到匹配的回调函数，再帮我们转发一下流量；
		// 所以说mux是一个代理模式；
	}

	// 赋值单例

	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	// 启动服务端；
	go httpServer.Serve(listen)
	return
}

