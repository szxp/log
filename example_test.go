// Copyright 2017 Szakszon Péter. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log_test

import (
	"fmt"
	"github.com/szxp/log"
	"os"
)

func Example() {
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

	// Output is something like this:
	// {"activated":true,"file":"example_test.go:44","level":"info","logger":"loggername","projects":["p1","p2","p3"],"user":{"id":1,"username":"admin"}}

}
