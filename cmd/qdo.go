package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/borgenk/qdo/http"
	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/log/stdout"
	"github.com/borgenk/qdo/log/syslog"
	"github.com/borgenk/qdo/queue"
	"github.com/borgenk/qdo/store"
	_ "github.com/borgenk/qdo/store/leveldb"
)

const Version = "0.3.0"

const defaultOptHttpPort = 8080
const defaultOptDbFilepath = "qdo.db"
const defaultOptSyslog = false
const defaultOptStore = "leveldb"

func main() {
	optHttpPort := flag.Int("p", defaultOptHttpPort, "HTTP port")
	optDbFilepath := flag.String("f", defaultOptDbFilepath, "Database file path")
	optSyslog := flag.Bool("s", defaultOptSyslog, "Log to syslog")
	flag.Parse()

	// Setup logging method.
	if *optSyslog {
		w, err := syslog.New(syslog.LOG_LOCAL0, "qdo")
		if err != nil {
			panic("Unable to connect to syslog")
		}
		log.InitLog(w)
	} else {
		w := stdout.New()
		log.InitLog(w)
	}
	log.Infof("starting QDo %s", Version)

	// Launch web admin interface server.
	go http.Run(*optHttpPort)

	// Launch queue manager.
	store, _ := store.GetStoreConstructor(defaultOptStore, *optDbFilepath)

	manager, err := queue.StartManager(store)
	if err != nil {
		fmt.Printf("%s", err)
		panic("Unable to start manager")
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	<-exit

	log.Info("stopping queues..")
	manager.Stop()
}
