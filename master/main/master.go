package main

import (
	"flag"
	"fmt"
	"github.com/k8svip/crontab/master"
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

	flag.StringVar(&confFile, "config", "./master.json", "指定master.json")
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
	if err = master.InitConfig(confFile); err != nil {
		goto ERR
	}

	// 初始化分组管理器
	if err = master.InitGroupMgr();err != nil {
		fmt.Println("====")
		goto ERR
	}

	// 初始化服务发现模块
	if err = master.InitWorkerMgr();err != nil {
		goto ERR
	}

	// 初始化日志管理器；
	if err = master.InitLogMgr();err != nil {
		goto ERR
	}

	// 任务管理器：启动JobMgr服务，因为ApiServer服务需要 调用JobMgr服务
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	//启动Api HTTP服务
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	//正常退出
	for {
		time.Sleep(1* time.Second)
	}

	return

ERR:
	fmt.Println(err)
}
