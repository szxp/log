
[![Build Status](https://travis-ci.org/szxp/log.svg?branch=master)](https://travis-ci.org/szxp/log)
[![Build Status](https://ci.appveyor.com/api/projects/status/github/szxp/log?branch=master&svg=true)](https://ci.appveyor.com/project/szxp/log)
[![GoDoc](https://godoc.org/github.com/szxp/log?status.svg)](https://godoc.org/github.com/szxp/log)
[![Go Report Card](https://goreportcard.com/badge/github.com/szxp/log)](https://goreportcard.com/report/github.com/szxp/log)

# log
A small structured logging library for Golang.
[Documentation is available at GoDoc](https://godoc.org/github.com/szxp/log).

## Releases
Master branch is still unstable, but it will be considered stable and production ready after the 1.0 release. Releases after the 1.0 version will always be backward compatible. 

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
	"io"
	"os"
)

func main() {
	// register an io.Writer
	// everything that is not a debug message will be written to stdout
	log.DefaultRouter.Output("stdout", os.Stdout, nil, log.Not(log.Eq("level", "debug")))

	// you can register as many io.Writers as you want
	// log.DefaultRouter.Output("mylogfile", myFile, nil, nil)

	// optional error callback function for debugging purposes
	log.DefaultRouter.OnError(func(id string, w io.Writer, err error) {
		fmt.Println(id, err)
	})

	// create a logger
	logger := log.NewLogger(log.Config{
		// TimeFormat: time.RFC3339,      // optional, see standard time package for custom formats

		Name:       "loggername",      // optional, name of the logger
		UTC:        true,              // optional, use UTC rather than local time zone
		FileLine:   log.ShortFileLine, // optional, include file and line number
		SortFields: true,              // optional, sort field keys in increasing order
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
	log.DefaultRouter.Output("stdout", os.Stdout, nil, nil)

	// Output:
	// {"activated":true,"file":"example_test.go:43","level":"info","logger":"loggername","projects":["p1","p2","p3"],"user":{"id":1,"username":"admin"}}

}
```

