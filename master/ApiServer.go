package master

import (
	"encoding/json"
	"github.com/k8svip/crontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 任务的http接口
type ApiServer struct {
	httpServer *http.Server
}

// 保存任务接口
// 我们让客户端Post的一个数据，这个数据是Json格式 ，
// POST job = {"name":"job1", "command":"echo hello", "cronExpr":"* * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	// 保存在ETCD中，1. 任务是什么，2. etcd，第一步定义任务结构，任务由ajx提交上来，我们保存到etcd当中；

	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)

	// 1. 解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 2. 取表单中的job字段；
	postJob = req.PostForm.Get("job")

	// 3. 反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	// 4. 保存到etcd
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}

	// 5. 返回正常应答({"error":0,"msg":"","data":{...}})
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	// 6. 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 删除任务接口
// 提交一个这样的请求 POST，/job/delete name = job1
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		name   string
		oldJob *common.Job
		bytes  []byte
	)

	// POST: a=1&b=2&c=3
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 删除的任务名
	name = req.PostForm.Get("name")

	// 去删除任务
	if oldJob, err = G_jobMgr.DeleteJob(name); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "sucesss", oldJob); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

//列举所有crontab任务
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		jobList []*common.Job
		err     error
		bytes   []byte
	)
	if jobList, err = G_jobMgr.ListJobs(); err != nil {
		goto ERR

	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "sucesss", jobList); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

}

// 强制杀死某个任务
// POST /job/kill name = job1
func handleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		err   error
		name  string
		bytes []byte
	)

	//解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	//杀死任务
	name = req.PostForm.Get("name")

	if err = G_jobMgr.KillJob(name); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "sucesss", nil); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 获取健康worker节点列表
func handleWorkerList(resp http.ResponseWriter, req *http.Request) {
	var (
		workerArr []string
		err error
		bytes []byte
	)
	if workerArr,err = G_workerMgr.ListWorkers();err != nil {
		goto ERR
	}
	// 正常应答
	if bytes, err = common.BuildResponse(0, "sucesss", workerArr); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

//查询任务日志
func handleJobLog(resp http.ResponseWriter, req *http.Request) {
	// 解析GET参数
	var (
		err        error
		name       string //任务名字
		skipParam string        //从第几条开始
		limitParam  string      //返回多少条
		skip       int
		limit      int
		logArr []*common.JobLog
		bytes []byte
	)

	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 获取请求参数/job/log?name=job10&skip=0&limit=10
	name = req.Form.Get("name")
	skipParam = req.Form.Get("skip")
	limitParam = req.Form.Get("limit")

	if skip, err = strconv.Atoi(skipParam); err != nil {
		skip = 0
	}
	if limit,err = strconv.Atoi(limitParam);err != nil {
		limit = 20
	}

	// 获取请求参数
	if logArr,err = G_logMgr.ListLog(name, skip, limit);err != nil {
		goto ERR

	}
	// 正常应答
	if bytes, err = common.BuildResponse(0, "sucesss", logArr); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

var (
	// 单例对象
	G_apiServer *ApiServer
)

// 初始化服务
func InitApiServer() (err error) {
	var (
		mux           *http.ServeMux
		listen        net.Listener
		httpServer    *http.Server
		staticDir     http.Dir // 静态文件的根目录
		staticHandler http.Handler //静态文件的HTTP回调
	)

	// 配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)
	mux.HandleFunc("/worker/list", handleWorkerList)

	// /index.html

	// 静态文件目录
	staticDir = http.Dir(G_config.Webroot)
	staticHandler = http.FileServer(staticDir)
	// mux.Handle("/", staticHandler) //
	mux.Handle("/", http.StripPrefix("/", staticHandler)) // ./webroot/index.html

	// 启动TCP监听
	if listen, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}

	// 创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
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
