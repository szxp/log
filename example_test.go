package log_test

import (
	"fmt"
	"github.com/szxp/log"
	"io"
	"os"
)

func Example() {
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
