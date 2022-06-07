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

	//Initializing the thread
	initEnv()

	// Load the configuration
	if err = master.InitConfig(confFile); err != nil {
		goto ERR
	}

	//service registration
	if err = master.InitWorkerMgr(); err != nil {
		goto ERR
	}

	//Start logging (connect to mongodb)
	if err = master.InitLogMgr(); err != nil {
		goto ERR
	}

	// Start etcd (Connect etcd)
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	//Start the Api HTTP service
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	//The normal exit
	return

ERR:
	fmt.Println(err)
}
