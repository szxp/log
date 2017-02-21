// Package log is a structured logging library.
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
var DefaultRouter = &defaultRouter{}

// DefaultFormatter converts a log message into JSON. It is
// used when there is no formatter associated with the io.Writer.
// It can be used simultaneously from multiple goroutines.
var DefaultFormatter Formatter = &JSONFormatter{}

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

	sorted := false
	if fieldSort, ok := f[FieldSort]; ok {
		sorted = fieldSort.(bool)
		if sorted {
			sort.Strings(keys)
		}
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

		if fv, ok := v.(Fields); ok {
			if _, ok = fv[FieldSort]; !ok {
				fv[FieldSort] = sorted
			}
		}

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

// LoggerConfig can be used to create a new Logger.
type LoggerConfig struct {
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
	// object should be sorted in increasing order
	// when marshaling the Fields object into JSON.
	//
	// The option can be overridden by providing a "_sort"
	// key with a bool value in the Fields object.
	SortFields bool

	// Router will forward the log messages to the registered
	// Writers. If not specified the default router will
	// be used.
	Router Router
}

// NewLogger creates and returns a new logger that forwards
// the fields to the router associated with the
// logger. If the Router is not specified in the Logger
// the DefaultRouter will be used.
//
// The returned Logger can be used simultaneously from
// multiple goroutines if and only if the Router associated
// with the Logger can be used simultaneously from multiple
// goroutines.
func (c LoggerConfig) NewLogger() Logger {
	return &logger{c}
}

type logger struct {
	config LoggerConfig
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
	// don't override the user's custom "_sort" config
	_, ok := fields[FieldSort]
	if ok {
		return
	}
	fields[FieldSort] = l.config.SortFields
}

// Router generates lines of output to registered Writers.
type Router interface {
	// Log writes the message to the registered Writers.
	Log(fields Fields)
}

// Output describes an output configuration in the DefaultRouter that
// formatted log messages will be written to.
type Output struct {
	// Id identifies the output configuration. It can
	// be used to update the configuration later.
	Id string

	// Writer represents the storage backend that formatted
	// log messages will be written to.
	Writer io.Writer

	// Formatter converts a log message into a []byte.
	// It is optional.
	//
	// The formatter must be safe for concurrent use by multiple
	// goroutines. If the formatter is nil the DefaultFormatter
	// will be used that converts log messages into a
	// JSON encoded string.
	Formatter Formatter

	// Filter specifies which messages should be
	// written to the io.Writer. It is optional.
	Filter Filter
}

// Register registers the output configuration in the DefaultRouter.
func (o Output) Register() {
	DefaultRouter.output(&o)
}

type defaultRouter struct {
	mu           sync.Mutex
	outputs      map[string]*Output
	errorHandler func(err error, fields Fields, o Output)
}

func (l *defaultRouter) output(o *Output) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.outputs == nil {
		l.outputs = make(map[string]*Output)
	}

	out, ok := l.outputs[o.Id]
	if !ok {
		out = o
	}

	out.Writer = o.Writer
	out.Formatter = o.Formatter
	if out.Formatter == nil {
		out.Formatter = DefaultFormatter
	}
	out.Filter = o.Filter
	l.outputs[out.Id] = out
}

// Log marshals the fields into a JSON object and
// writes it to the registered Writers.
func (l *defaultRouter) Log(fields Fields) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, o := range l.outputs {
		if o.Writer != nil {
			if o.Filter != nil {
				match, err := o.Filter.Match(fields)
				if err != nil {
					l.reportError(err, fields, o)
				}
				if !match {
					continue
				}
			}

			b, err := o.Formatter.Format(fields)
			if err != nil {
				l.reportError(err, fields, o)
				continue
			}

			writer := &writer{out: o.Writer}
			writer.write(b)
			writer.write([]byte{'\n'})
			if writer.err != nil {
				l.reportError(writer.err, fields, o)
				continue
			}
		}
	}
}

// OnError registers an error handler callback in the DefaultRouter.
//
// The callback will be called if an error occurs while writing
// a log message to an io.Writer.
func OnError(f func(err error, fields Fields, o Output)) {
	DefaultRouter.onError(f)
}

func (l *defaultRouter) onError(f func(err error, fields Fields, o Output)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorHandler = f
}

func (l *defaultRouter) reportError(err error, fields Fields, o *Output) {
	if l.onError != nil {
		l.errorHandler(err, fields, *o)
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

// JSONFormatter converts a log message into JSON encoded string.
//
// JSONFormatter is safe for concurrent use by multiple goroutines.
type JSONFormatter struct{}

// Format returns the fields as a valid JSON.
func (f *JSONFormatter) Format(fields Fields) ([]byte, error) {
	return json.Marshal(fields)
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
	_, ok := fields.Value(e.path)
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
