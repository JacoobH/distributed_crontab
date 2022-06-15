package main

import (
	"flag"
	"fmt"
	"github.com/JacoobH/crontab/worker"
	"runtime"
	"time"
)

var (
	// Set a global configuration file path, obtained from the command line
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
	// Set the maximum number of threads for the current GO program to the number of CPU cores on the host
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

	// Service registration
	if err = worker.InitRegister(); err != nil {
		goto ERR
	}

	// Start log
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

	//
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	//The normal exit
	for {
		time.Sleep(1 * time.Second)
	}

	return

ERR:
	fmt.Println(err)
}
