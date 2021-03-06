package main

import (
	"flag"
	"fmt"
	"github.com/k8svip/crontab/worker"
	"runtime"
	"time"
)

// 初始化线程数量
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

var (
	confFile string //配置文件路径
)

// 解析命令行参数
func initArgs() {

	flag.StringVar(&confFile, "config", "./worker.json", "指定worker.json")
	flag.Parse()
}

func main() {
	var (
		err error
	)

	// 初始化命令行参数
	initArgs()

	// 初始化线程
	initEnv()

	// 加载配置
	if err = worker.InitConfig(confFile); err != nil {
		goto ERR
	}

	// 判断本地IP是否属于此APPID
	if err = worker.InitCheckGroupMgr();err != nil {
		goto ERR
	}

	// 服务注册
	if err = worker.InitRegister();err != nil {
		goto ERR
	}

	// 启动日志协程；
	if err = worker.InitLogSink();err != nil {
		goto ERR
	}

	// 启动执行器
	if err = worker.InitExecutor();err != nil {
		goto ERR
	}

	// 启动调度器
	if err = worker.InitScheduler();err != nil {
		goto ERR
	}

	// 初始化任务管理器
	if err = worker.InitJobMgr();err != nil {
		goto ERR

	}
	//正常退出
	for {
		time.Sleep(1 * time.Second)
	}

	return

ERR:
	fmt.Println(err)

}

