package main

import (
	"fmt"
	"github.com/szxp/log"
	"io"
	"os"
	"time"
)

func main() {
	// open logfile
	logfile, err := os.OpenFile("logfile", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer logfile.Close()

	// register stdout with filtering in the default Router
	// everything that is not a debug message will be written to stdout
	log.Output("stdout", os.Stdout, nil, log.Not(log.Eq("level", "debug")))

	// register a logfile in the default Router
	// everything without filtering goes into the logfile
	log.Output("mylogfile", logfile, nil, nil)

	log.DefaultRouter.OnError(func(id string, w io.Writer, err error) {
		fmt.Println(id, " ", err)
	})

	// create a logger
	logger := log.NewLogger(log.Config{
		Name:       "loggername",      // optional, name of the logger
		TimeFormat: time.RFC3339,      // optional, format timestamp
		UTC:        true,              // optional, use UTC rather than local time zone
		FileLine:   log.ShortFileLine, // optional, include file and line number
		SortFields: true,              // optional, prevents keys to appear in any order
		Router:     nil,               // optional, defaults to log.DefaultRouter
	})

	// produce some log messages
	logger.Log(log.Fields{
		"level": "info",
		"user": log.Fields{
			"id":       1,
			"username": "admin",
		},
		"activated": true,
		"projects":  []string{"p1", "p2", "p3"},
	})

	logger.Log(log.Fields{
		"level":   "debug",
		"details": "...",
	})

	// update output configurations at runtime
	// for example disable filtering on Stdout
	log.Output("stdout", os.Stdout, nil, nil)

	// Output on Stdout:
	// {"activated":true,"file":"example.go:51","level":"info","logger":"loggername","projects":["p1","p2","p3"],"time":"2017-02-03T21:43:23Z","user":{"id":1,"username":"admin"}}

	// Output in logfile:
	// {"activated":true,"file":"example.go:51","level":"info","logger":"loggername","projects":["p1","p2","p3"],"time":"2017-02-03T21:43:23Z","user":{"id":1,"username":"admin"}}
	// {"details":"...","file":"example.go:56","level":"debug","logger":"loggername","time":"2017-02-03T21:43:23Z"}
}
