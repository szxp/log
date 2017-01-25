// Package log a small structured logging library for Golang.
package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// FlagRFC3339 adds the textual representation of the time
	// formatted according to RFC3339 to the message
	// at the FieldTime key.
	FlagRFC3339 = 1 << iota

	// FlagUTC configures a logger to use UTC rather than the
	// local time zone. Assumes FlagRFC3339.
	FlagUTC

	// FlagUnix adds Unix time (the number of seconds
	// elapsed since January 1, 1970 UTC) to the message
	// at the FieldTime key.
	FlagUnix

	// FlagUnixNano adds Unix time (the number of
	// nanoseconds elapsed since January 1, 1970 UTC)
	// to the message at the FieldTime key.
	// Overrides FlagUnix.
	FlagUnixNano

	// FlagLogger adds the logger's name to the message
	// at the FieldLogger key.
	FlagLogger

	// FlagLongfile adds the full file name and line number
	// to the message at the FieldFile key.
	FlagLongfile

	// FlagShortfile adds the final file name element and
	// line number to the message at the FieldFile key.
	// Overrides FlagLongfile.
	FlagShortfile

	// FlagStd is the initial value for logger created without flags.
	FlagStd = FlagRFC3339 | FlagUTC
)

const (
	// FieldTime is the name of the time field.
	FieldTime = "time"

	// FieldLogger is the name of the logger field.
	FieldLogger = "logger"

	// FieldFile is the name of the file field.
	FieldFile = "file"
)

// DefaultRouter is used by those Loggers which are created without a Router.
var DefaultRouter Router = &router{}

// Fields represents a log message. Think of it as a JSON object
// or a key-value map where keys are strings and value can be
// a number, a string, a bool, an array, a slice, nil or another
// Fields object.
type Fields map[string]interface{}

// value returns the value at the given path.
func (f Fields) value(path []string) (interface{}, bool) {
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
func (f Fields) MarshalJSON() ([]byte, error) {
	count := 0
	size := len(f)

	buf := &bytes.Buffer{}
	buf.WriteByte('{')

	for k, v := range f {
		count++
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
		if count < size {
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

// NewLogger returns a new named logger.
//
// Flags or'ed together to control what's printed.
//
// The messages will be forwarded to the router associated with the logger.
// The router will write the log messages to the registered Writers.
// If router is nil the default router will be used.
//
// The returned Logger can be used simultaneously from multiple goroutines
// if and only if the Router associated with the Logger
// can be used simultaneously from multiple goroutines.
func NewLogger(name string, flags int64, router Router) Logger {
	if flags == 0 {
		flags = FlagStd
	}
	return &logger{
		name:   name,
		flags:  flags,
		router: router,
	}
}

type logger struct {
	name   string
	flags  int64
	router Router
}

// Log forwards the fields to the router associated with the logger.
// If a Router is not set in the Logger then the DefaultRouter will be used.
func (l *logger) Log(fields Fields) {
	t := time.Now() // get this early

	if fields == nil {
		fields = Fields{}
	}

	l.addTime(fields, t)
	l.addLogger(fields)
	l.addFile(fields, 2)

	r := l.router
	if r == nil {
		r = DefaultRouter
	}
	r.Log(fields)
}

func (l *logger) addTime(fields Fields, t time.Time) {
	// don't override the user's custom "time" field
	_, ok := fields[FieldTime]
	if ok || l.flags&(FlagRFC3339|FlagUnix|FlagUnixNano) == 0 {
		return
	}

	if l.flags&FlagUnixNano != 0 {
		fields[FieldTime] = t.UnixNano()
		return
	} else if l.flags&FlagUnix != 0 {
		fields[FieldTime] = t.Unix()
		return
	}

	if l.flags&FlagUTC != 0 {
		t = t.UTC()
	}
	fields[FieldTime] = t.Format(time.RFC3339)
}

func (l *logger) addLogger(fields Fields) {
	// don't override the user's custom "logger" field
	_, ok := fields[FieldLogger]
	if ok || l.flags&FlagLogger == 0 {
		return
	}
	fields[FieldLogger] = l.name
}

func (l *logger) addFile(fields Fields, calldepth int) {
	// don't override the user's custom "file" field
	_, ok := fields[FieldFile]
	if ok || l.flags&(FlagShortfile|FlagLongfile) == 0 {
		return
	}

	buf := &bytes.Buffer{}
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	if l.flags&FlagShortfile != 0 {
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

// Router generates lines of output to registered Writers.
type Router interface {
	// Output registers a Writer where lines should be written to.
	Output(id string, w io.Writer, filter Filter)
	// Log writes the message to the registered Writers.
	Log(fields Fields)
}

// NewRouter returns a new Router.
//
// A Router can be used simultaneously from multiple goroutines.
// It guarantees to serialize access to the Writer.
func NewRouter() Router {
	return &router{}
}

type router struct {
	mu      sync.RWMutex
	outputs map[string]*output
}

// Output registers a Writer where lines should be written to.
func (l *router) Output(id string, w io.Writer, filter Filter) {
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
	r.filter = filter
	l.outputs[id] = r
}

type output struct {
	w      io.Writer
	filter Filter
}

// Output registers a Writer in the default Router.
func Output(id string, w io.Writer, filter Filter) {
	DefaultRouter.Output(id, w, filter)
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

	for _, o := range l.outputs {
		if o.w != nil {
			b, err := json.Marshal(fields)
			if err != nil {
				//if onError != nil {
				//  onError(err, id)
				//}
				continue
			}

			writer := &writer{out: o.w}
			writer.write(b)
			writer.write([]byte{'\n'})
			if writer.err != nil {
				//if onError != nil {
				//  onError(writer.err, id)
				//}
				continue
			}
		}
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

// Filter represents a filter condition.
type Filter interface {
	// Match evaluates the filter.
	Match(fields Fields) (bool, error)
}

// FieldExist returns a filter that checks if the given path
// exists in the log message. Path is a dot-separated field names.
func FieldExist(path string) Filter {
	return &fieldExist{strings.Split(path, ".")}
}

type fieldExist struct {
	path []string
}

// Match returns true if the path exists in the log message.
// Otherwise returns false.
func (e *fieldExist) Match(fields Fields) (bool, error) {
	_, ok := fields.value(e.path)
	if !ok {
		return false, nil
	}
	return true, nil
}

// Eq returns a filter that checks if the value at the
// given path is equal to the given value.
// Path is a dot-separated field names.
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
	v, ok := fields.value(e.path)
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
