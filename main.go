package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const version = "2.9.0"

type ctxKey string

func initialize() {
	log.SetOutput(os.Stdout)

	initLog()
	configure()
	initNewrelic()
	initPrometheus()
	initDownloading()
	initErrorsReporting()
	initVips()

	if err := checkPresets(conf.Presets); err != nil {
		shutdownVips()
		logFatal(err.Error())
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "health" {
		healthcheck()
	}

	initialize()
	defer shutdownVips()

	go func() {
		var logMemStats = len(os.Getenv("IMGPROXY_LOG_MEM_STATS")) > 0

		for range time.Tick(time.Duration(conf.FreeMemoryInterval) * time.Second) {
			freeMemory()

			if logMemStats {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				logDebug("MEMORY USAGE: Sys=%d HeapIdle=%d HeapInuse=%d", m.Sys/1024/1024, m.HeapIdle/1024/1024, m.HeapInuse/1024/1024)
			}
		}
	}()

	s := startServer()
	defer shutdownServer(s)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
}
