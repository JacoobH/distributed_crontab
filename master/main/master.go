package main

import (
	"flag"
	"fmt"
	"github.com/JacoobH/crontab/master"
	"runtime"
)

var (
	confFile string // Configuration file path
)

// Parses command line arguments
func initArgs() {
	// master -config ./master.json
	// master -h
	flag.StringVar(&confFile, "config", "./master.json", "Specify the master.json")
	flag.Parse()
}

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)

	// Initialize command line arguments
	initArgs()

	//初始化线程
	initEnv()

	// Load the configuration
	if err = master.InitConfig(confFile); err != nil {
		goto ERR
	}

	if err = master.InitWorkerMgr(); err != nil {
		goto ERR
	}

	if err = master.InitLogMgr(); err != nil {
		goto ERR
	}

	// Job manager
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	//启动API HTTP服务
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	//正常退出
	return

ERR:
	fmt.Println(err)
}
