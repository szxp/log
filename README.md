
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
		Name:       "loggername",      // optional
		TimeFormat: time.RFC3339,      // optional
		UTC:        true,              // optional
		FileLine:   log.ShortFileLine, // optional
		Router:     nil,               // optional, if nil the log.DefaultRouter will be used
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
	// for example disable logging by setting a nil Writer
	log.Output("stdout", nil, nil, nil)
	log.Output("mylogfile", nil, nil, nil)

	// this message will be never logged
	logger.Log(log.Fields{
		"neverLogged": 1,
	})

	// Output on Stdout:
	// {"activated":true,"projects":["p1","p2","p3"],"time":"2017-01-28T19:48:38Z","logger":"loggername","file":"example.go:45","level":"info","user":{"id":1,"username":"admin"}}

	// Output in logfile:
	// {"activated":true,"projects":["p1","p2","p3"],"time":"2017-01-28T19:49:16Z","logger":"loggername","file":"example.go:45","level":"info","user":{"id":1,"username":"admin"}}
	// {"level":"debug","details":"...","time":"2017-01-28T19:49:16Z","logger":"loggername","file":"example.go:50"}
}
```

