
[![Build Status](https://travis-ci.org/szxp/log.svg?branch=master)](https://travis-ci.org/szxp/log)
[![Build Status](https://ci.appveyor.com/api/projects/status/github/szxp/log?branch=master&svg=true)](https://ci.appveyor.com/project/szxp/log)
[![GoDoc](https://godoc.org/github.com/szxp/log?status.svg)](https://godoc.org/github.com/szxp/log)
[![Go Report Card](https://goreportcard.com/badge/github.com/szxp/log)](https://goreportcard.com/report/github.com/szxp/log)

# log
A small structured logging library for Golang.
[Documentation is available at GoDoc](https://godoc.org/github.com/szxp/log).

## Releases
Master branch is the stable production-ready branch. 

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
)

func main() {
	// register an io.Writer in the DefaultRouter
	// everything that is not a debug message will be written to stdout
	log.Output{
		Id:        "stdout1",
		Writer:    os.Stdout,
		Formatter: nil,
		Filter:    log.Not(log.Eq("level", "debug")),
	}.Register()

	// optional error callback in the DefaultRouter for debugging purposes
	log.OnError(func(err error, fields log.Fields, o log.Output) {
		fmt.Printf("%v: %+v: %+v", err, fields, o)
	})

	// create a logger
	logger := log.LoggerConfig{
		// TimeFormat: time.RFC3339,      // optional, see standard time package for custom formats

		Name:       "loggername",      // optional, name of the logger
		UTC:        true,              // optional, use UTC rather than local time zone
		FileLine:   log.ShortFileLine, // optional, include file and line number
		SortFields: true,              // optional, sort field keys in increasing order
		Router:     nil,               // optional, defaults to log.DefaultRouter
	}.NewLogger()

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

	// output reconfiguration in the DefaultRouter
	// for example disable filtering on Stdout
	log.Output{
		Id:        "stdout1",
		Writer:    os.Stdout,
		Formatter: nil,
		Filter:    nil,
	}.Register()

	// Output:
	// {"activated":true,"file":"example_test.go:44","level":"info","logger":"loggername","projects":["p1","p2","p3"],"user":{"id":1,"username":"admin"}}

}
```

