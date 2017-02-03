
[![Build Status](https://travis-ci.org/szxp/log.svg?branch=master)](https://travis-ci.org/szxp/log)
[![Build Status](https://ci.appveyor.com/api/projects/status/github/szxp/log?branch=master&svg=true)](https://ci.appveyor.com/project/szxp/log)
[![GoDoc](https://godoc.org/github.com/szxp/log?status.svg)](https://godoc.org/github.com/szxp/log)
[![Go Report Card](https://goreportcard.com/badge/github.com/szxp/log)](https://goreportcard.com/report/github.com/szxp/log)

# log
A small structured logging library for Golang.
[Documentation is available at GoDoc](https://godoc.org/github.com/szxp/log).

## Features
* Only standard library dependencies
* Output configurations can be modified at runtime
* Default formatter formats log messages as JSON encoded string. Custom formatters can be used. 

## Example
```go
package main

import (
	"fmt"
	"github.com/szxp/log"
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
```

