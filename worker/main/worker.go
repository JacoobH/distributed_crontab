package main

import (
	"flag"
	"fmt"
	"github.com/JacoobH/crontab/worker"
	"runtime"
	"time"
)

var (
	confFile string // Configuration file path
)

// Parses command line arguments
func initArgs() {
	// worker -config ./worker.json
	// worker -h
	flag.StringVar(&confFile, "config", "./worker.json", "Specify the worker.json")
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

	// Initializing the thread
	initEnv()

	// Load the configuration
	if err = worker.InitConfig(confFile); err != nil {
		goto ERR
	}

	if err = worker.InitRegister(); err != nil {
		goto ERR
	}

	if err = worker.InitLogSink(); err != nil {
		goto ERR
	}

	// Initializing executor
	if err = worker.InitExecutor(); err != nil {
		goto ERR
	}

	// Initializing scheduler
	if err = worker.InitScheduler(); err != nil {
		goto ERR
	}

	if err = worker.InitJobMgr(); err != nil {
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
