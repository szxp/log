// Package log is a structured logging library.
//
// Example:
//  package main
//
//  import (
// 	"fmt"
// 	"github.com/szxp/log"
// 	"os"
// 	"time"
//  )
//
//  func main() {
// 	// open logfile
// 	logfile, err := os.OpenFile("logfile", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(-1)
// 	}
// 	defer logfile.Close()
//
// 	// register stdout with filtering in the default Router
// 	// everything that is not a debug message will be written to stdout
// 	log.Output("stdout", os.Stdout, nil, log.Not(log.Eq("level", "debug")))
//
// 	// register a logfile in the default Router
// 	// everything without filtering goes into the logfile
// 	log.Output("mylogfile", logfile, nil, nil)
//
// 	// create a logger
// 	logger := log.NewLogger(log.Config{
// 		Name:       "loggername",      // optional, name of the logger
// 		TimeFormat: time.RFC3339,      // optional, format timestamp
// 		UTC:        true,              // optional, use UTC rather than local time zone
// 		FileLine:   log.ShortFileLine, // optional, include file and line number
// 		SortFields: true,              // optional, prevents keys to appear in any order
// 		Router:     nil,               // optional, defaults to log.DefaultRouter
// 	})
//
// 	// produce some log messages
// 	logger.Log(log.Fields{
// 		"level": "info",
// 		"user": log.Fields{
// 			"id":       1,
// 			"username": "admin",
// 		},
// 		"activated": true,
// 		"projects":  []string{"p1", "p2", "p3"},
// 	})
//
// 	logger.Log(log.Fields{
// 		"level":   "debug",
// 		"details": "...",
// 	})
//
// 	// update output configurations at runtime
// 	// for example disable filtering on Stdout
// 	log.Output("stdout", os.Stdout, nil, nil)
//
// 	// Output on Stdout:
// 	// {"activated":true,"file":"example.go:51","level":"info","logger":"loggername","projects":["p1","p2","p3"],"time":"2017-02-03T21:15:45Z","user":{"username":"admin","id":1}}
//
// 	// Output in logfile:
// 	// {"activated":true,"file":"example.go:51","level":"info","logger":"loggername","projects":["p1","p2","p3"],"time":"2017-02-03T21:15:45Z","user":{"id":1,"username":"admin"}}
// 	// {"details":"...","file":"example.go:56","level":"debug","logger":"loggername","time":"2017-02-03T21:15:45Z"}
//  }
package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// FieldTime is the name of the time field.
	FieldTime = "time"

	// FieldLogger is the name of the logger field.
	FieldLogger = "logger"

	// FieldFile is the name of the file field.
	FieldFile = "file"

	// FieldSort is the name of the field that indicates
	// if the keys should be sorted in the JSON encoded
	// log message.
	FieldSort = "_sort"
)

const (
	// ShortFileLine is a config option,
	// for the final file name element and line number.
	ShortFileLine = iota + 1

	// LongFileLine is a config option,
	// for the full file path and line number.
	LongFileLine
)

// DefaultRouter is used by those Loggers which are created
// without a Router. It can be used simultaneously from
// multiple goroutines.
var DefaultRouter Router = &router{}

// DefaultFormatter converts log message into JSON. It is
// used when there is no formatter associated with a Writer.
// It can be used simultaneously from multiple goroutines.
var DefaultFormatter Formatter = &jsonFormatter{}

// Fields represents a log message, a key-value map
// where keys are strings and a value can be a number,
// a string, a bool, an array, a slice, nil or another
// Fields object.
type Fields map[string]interface{}

// Value returns the value at the given path.
// The second return value indicates if the path exists.
func (f Fields) Value(path []string) (interface{}, bool) {
	for i, field := range path {
		v, ok := f[field]
		if !ok {
			return nil, false
		}
		if i == len(path)-1 {
			return v, true
		}

		f, ok = v.(Fields)
		if !ok {
			return nil, false
		}
	}
	return nil, false
}

// MarshalJSON marshals the fields into a JSON object.
//
// When iterating over the field keys, the iteration order
// is not specified and is not guaranteed to be the
// same from one iteration to the next. So field keys
// may appear in any order in the log message.
//
// Keys that begin with underscore will be skipped.
//
// If the Fields object contains a "_sort" key with a true
// bool value the keys will appear in increasing order
// in the JSON encoded string.
//
// Create a logger with the SortFields config option
// set to true if you want the keys in all log messages
// to be sorted.
func (f Fields) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteByte('{')

	keys := make([]string, 0, len(f))
	for k := range f {
		if len(k) > 0 && k[0] != '_' {
			keys = append(keys, k)
		}
	}

	if beSorted, ok := f[FieldSort]; ok && beSorted.(bool) {
		sort.Strings(keys)
	}

	size := len(keys)
	for i, k := range keys {
		v := f[k]
		b, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(b)
		buf.WriteByte(':')

		b, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
		buf.Write(b)
		if i < size-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// Logger writes a message.
type Logger interface {
	// Log writes a message.
	Log(fields Fields)
}

// Config can be used to configure a Logger.
type Config struct {
	// Name of the logger. A non empty name will be
	// added to the log message at the key FieldLogger.
	//
	// The value at the key FieldLogger can be overridden
	// by specifying a custom value at that key.
	Name string

	// TimeFormat specifies the format of the timestamp.
	// If non empty, a timestamp in local time zone
	// according to the specified format will be
	// added to the log message at the key FieldTime.
	// Set UTC to true to use UTC rather than the
	// local time zone.
	//
	// The value at the key FieldTime can be overridden
	// by specifying a custom value at that key.
	//
	// See the standard time package for details on how
	// to define time formats:
	// https://golang.org/pkg/time/#pkg-constants
	TimeFormat string

	// UTC configures a logger to use UTC rather than the
	// local time zone. Assumes a non empty TimeFormat.
	UTC bool

	// FileLine, if not zero, adds the file name and line
	// number to the log message at the key FieldFile.
	//
	// Use LongFileLine for the full file path and line number.
	// Use ShortFileLine for the final file name element and
	// line number.
	//
	// The value at the key FieldFile can be overridden
	// by specifying a custom value at that key.
	FileLine int

	// SortFields indicates if the keys in the Fields
	// object should be sorted when marshaling the Fields
	// object into JSON. It prevents the field keys to
	// appear in any order in the final JSON encoded log
	// message.
	//
	// The option can be overridden by providing a "_sort"
	// key with a bool value in the Fields object.
	SortFields bool

	// Router will forward the log messages to the registered
	// Writers. If not specified the default router will
	// be used.
	Router Router
}

// NewLogger creates and returns a new logger.
//
// The returned Logger can be used simultaneously from
// multiple goroutines if and only if the Router associated
// with the Logger can be used simultaneously from multiple
// goroutines.
func NewLogger(config Config) Logger {
	// config is a copy, can be stored safely
	return &logger{config}
}

type logger struct {
	config Config
}

// Log forwards the fields to the router associated with the
// logger. If the Router is not specified in the Logger
// the DefaultRouter will be used.
func (l *logger) Log(fields Fields) {
	t := time.Now() // get this early

	if fields == nil {
		fields = Fields{}
	}

	l.addTime(fields, t)
	l.addLogger(fields)
	l.addFile(fields, 2)
	l.sortFields(fields)

	r := l.config.Router
	if r == nil {
		r = DefaultRouter
	}
	r.Log(fields)
}

func (l *logger) addTime(fields Fields, t time.Time) {
	// don't override the user's custom "time" field
	_, ok := fields[FieldTime]
	if ok || l.config.TimeFormat == "" {
		return
	}

	if l.config.UTC {
		t = t.UTC()
	}
	fields[FieldTime] = t.Format(l.config.TimeFormat)
}

func (l *logger) addLogger(fields Fields) {
	// don't override the user's custom "logger" field
	_, ok := fields[FieldLogger]
	if ok || l.config.Name == "" {
		return
	}
	fields[FieldLogger] = l.config.Name
}

func (l *logger) addFile(fields Fields, calldepth int) {
	// don't override the user's custom "file" field
	_, ok := fields[FieldFile]
	if ok || (l.config.FileLine != ShortFileLine && l.config.FileLine != LongFileLine) {
		return
	}

	buf := &bytes.Buffer{}
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	if l.config.FileLine == ShortFileLine {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
	}
	buf.WriteString(fmt.Sprintf("%s:%d", file, line))
	fields[FieldFile] = buf.String()
}

func (l *logger) sortFields(fields Fields) {
	key := "_sort"
	// don't override the user's custom "_sort" config
	_, ok := fields[key]
	if ok {
		return
	}
	fields[key] = l.config.SortFields
}

// Router generates lines of output to registered Writers.
type Router interface {
	// Output registers a Writer where formatted log
	// messages should be written to.
	// Filter can be used to specify which messages
	// should be written.
	Output(id string, w io.Writer, formatter Formatter, filter Filter)

	// Log writes the message to the registered Writers.
	Log(fields Fields)

	// OnError registers an error handler callback.
	// The callback will be called if an error occurs while writing a log message.
	// The callback will be passed the id of the Writer, the Writer, and the error.
	OnError(f func(id string, w io.Writer, err error))
}

// NewRouter returns a new Router.
//
// The returned Router can be used simultaneously from
// multiple goroutines. It guarantees to serialize access
// to the Writer.
func NewRouter() Router {
	return &router{}
}

type router struct {
	mu      sync.RWMutex
	outputs map[string]*output
	onError func(id string, w io.Writer, err error)
}

// Output registers a Writer where lines should be written to.
//
// The formatter must be safe for concurrent use by multiple
// goroutines. If the formatter is nil the DefaultFormatter
// will be used that converts log messages into JSON.
//
// Optional filter can be used to specify which messages
// should be written.
//
// This method can be called with the same id to update the
// configuration of a registered Writer.
func (l *router) Output(id string, w io.Writer, formatter Formatter, filter Filter) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.outputs == nil {
		l.outputs = make(map[string]*output)
	}

	r, ok := l.outputs[id]
	if !ok {
		r = &output{}
	}

	r.w = w
	r.formatter = formatter
	if r.formatter == nil {
		r.formatter = DefaultFormatter
	}
	r.filter = filter
	l.outputs[id] = r
}

type output struct {
	w         io.Writer
	formatter Formatter
	filter    Filter
}

// Output registers a Writer in the default Router.
// See Router's Output method for details on how to
// register a Writer.
func Output(id string, w io.Writer, formatter Formatter, filter Filter) {
	DefaultRouter.Output(id, w, formatter, filter)
}

// Log marshals the fields into a JSON object and
// writes it to the registered Writers.
//
// When iterating over the field keys, the iteration order
// is not specified and is not guaranteed to be the
// same from one iteration to the next. So field keys
// may appear in any order in the log message.
func (l *router) Log(fields Fields) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for id, o := range l.outputs {
		if o.w != nil {
			if o.filter != nil {
				match, err := o.filter.Match(fields)
				if err != nil {
					l.reportError(id, o.w, err)
				}
				if !match {
					continue
				}
			}

			b, err := o.formatter.Format(fields)
			if err != nil {
				l.reportError(id, o.w, err)
				continue
			}

			writer := &writer{out: o.w}
			writer.write(b)
			writer.write([]byte{'\n'})
			if writer.err != nil {
				l.reportError(id, o.w, writer.err)
				continue
			}
		}
	}
}

// OnError registers an error handler callback.
// The callback will be called if an error occurs while writing a log message.
// The callback will be passed the id of the Writer, the Writer, and the error.
func (l *router) OnError(f func(id string, w io.Writer, err error)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onError = f
}

func (l *router) reportError(id string, w io.Writer, err error) {
	if l.onError != nil {
		l.onError(id, w, err)
	}
}

type writer struct {
	out io.Writer
	err error
}

func (w *writer) write(b []byte) {
	if w.err != nil {
		return
	}
	_, err := w.out.Write(b)
	if err != nil {
		w.err = err
	}
}

// Formatter converts a log message into a []byte.
type Formatter interface {
	// Format returns a textual representation of the fields as a []byte.
	Format(fields Fields) ([]byte, error)
}

type jsonFormatter struct{}

// NewJSONFormatter returns a new Formatter that converts
// a log message into JSON.
//
// The returned Formatter is safe for concurrent use by
// multiple goroutines.
func NewJSONFormatter() Formatter {
	return &jsonFormatter{}
}

// Format returns the fields as a valid JSON.
func (f *jsonFormatter) Format(fields Fields) ([]byte, error) {
	return json.Marshal(fields)
}

// Filter represents a filter condition.
type Filter interface {
	// Match evaluates the filter.
	Match(fields Fields) (bool, error)
}

// FieldExist returns a filter that checks if the given path
// exists in the log message. Path is a dot-separated field names.
//
// Example:
//  log.FieldExist("user")
func FieldExist(path string) Filter {
	return &fieldExist{strings.Split(path, ".")}
}

type fieldExist struct {
	path []string
}

// Match returns true if the path exists in the log message.
// Otherwise returns false.
func (e *fieldExist) Match(fields Fields) (bool, error) {
	_, ok := fields.Value(e.path)
	if !ok {
		return false, nil
	}
	return true, nil
}

// Eq returns a filter that checks if the value at the
// given path is equal to the given value.
// Path is a dot-separated field names.
//
// Example:
//  log.Eq("user.id", 1)
//  log.Not(log.Eq("level", "debug"))
func Eq(path string, value interface{}) Filter {
	return &eq{strings.Split(path, "."), value}
}

type eq struct {
	path  []string
	value interface{}
}

// Match returns true if the path exists and the value at
// that path is equal to the value in this filter.
func (e *eq) Match(fields Fields) (bool, error) {
	v, ok := fields.Value(e.path)
	if !ok {
		return false, nil
	}
	//fmt.Printf("%T %+v\n", e.value, e.value)
	//fmt.Printf("%T %+v\n", v, v)
	return e.value == v, nil
}

// And returns a composite filter consisting of multiple
// filters and-ed together.
//
// Filters are evaluated left to right, they are tested
// for possible "short-circuit" evaluation using the following
// rules: false && (anything) is short-circuit evaluated to false.
//
// Example:
//  log.And(
//    log.Eq("user.id", 1),
//    log.Eq("level", "trace")
//  )
func And(filters ...Filter) Filter {
	return &and{filters}
}

type and struct {
	filters []Filter
}

// Match returns true if all of the filters in this composite
// filter evaluate to true. Otherwise returns false.
func (a *and) Match(fields Fields) (bool, error) {
	for _, f := range a.filters {
		match, err := f.Match(fields)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}

// Or returns a composite filter consisting of multiple
// filters or-ed together.
//
// Filters are evaluated left to right, they are tested
// for possible "short-circuit" evaluation using the following
// rules: true || (anything) is short-circuit evaluated to true.
//
// Example:
//  log.Or(
//    log.Eq("username", "admin"),
//    log.Eq("username", "bot")
//  )
func Or(filters ...Filter) Filter {
	return &or{filters}
}

type or struct {
	filters []Filter
}

// Match returns true if one of the filters in this composite
// filter evaluates to true. Otherwise returns false.
func (o *or) Match(fields Fields) (bool, error) {
	for _, f := range o.filters {
		match, err := f.Match(fields)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

// Not returns a composite filter inverting the given filter.
//
// Example:
//  log.Not(log.Eq("user.id", 1))
func Not(filter Filter) Filter {
	return &not{filter}
}

type not struct {
	filter Filter
}

// Match returns true if the filter in this composite filter
// evaluates to false. Otherwise returns false.
func (n *not) Match(fields Fields) (bool, error) {
	match, err := n.filter.Match(fields)
	if err != nil {
		return false, err
	}
	return !match, nil
}
