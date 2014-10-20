package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/borgenk/qdo/core"
	"github.com/borgenk/qdo/http"
	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/log/stdout"
	"github.com/borgenk/qdo/log/syslog"
	"github.com/borgenk/qdo/store"
	_ "github.com/borgenk/qdo/store/leveldb"
)

const Version = "0.3.0"

const defaultOptHTTPPort = 7999
const defaultOptDBFilepath = "/var/qdo/"
const defaultOptSyslog = false
const defaultOptStore = "leveldb"

func main() {
	optHTTPPort := flag.Int("p", defaultOptHTTPPort, "HTTP port")
	optDBFilepath := flag.String("f", defaultOptDBFilepath, "Database filepath")
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
	go http.Run(*optHTTPPort)

	// Launch queue manager.
	store, _ := store.GetStoreConstructor(defaultOptStore, *optDBFilepath)

	manager, err := core.StartController(store)
	if err != nil {
		fmt.Printf("%s", err)
		panic("Unable to start controller")
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	<-exit

	log.Info("stopping..")
	manager.Stop()
}
