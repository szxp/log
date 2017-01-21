// A small structured logging library for Golang.
package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"
)

const (
	// Bits or'ed together to control what's printed.
	FlagDate      = 1 << iota                       // the date in UTC
	FlagTime                                        // the time in UTC
	FlagMicro                                       // microsecond resolution, assumes FlagTime
	FlagLogger                                      // include logger's name
	FlagLongfile                                    // full file name and line number
	FlagShortfile                                   // final file name element and line number, overrides FlagLongfile
	FlagLocaltime                                   // if FlagDate or FlagTime is set, use local time rather than UTC
	FlagStd       = FlagDate | FlagTime | FlagMicro // initial values for the standard logger
)

const (
	FieldTime   = "time"   // time field
	FieldLogger = "logger" // logger field
	FieldFile   = "file"   // file field
)

// DefaultRouter is used by those Loggers which are created without a Router.
var DefaultRouter = &Router{}

// Fields represents a log message that consists of key-value pairs.
type Fields map[string]interface{}

// MarshalJSON marshals this fields into valid JSON.
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

// NewLogger returns a new logger.
//
// The router will be used to forward the log messages to output Writers.
// If router is nil the default router will be used instead.
func NewLogger(name string, flags int, router *Router) Logger {
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
	flags  int
	router *Router
}

// Log writes a message.
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
	if ok || l.flags&(FlagDate|FlagTime) == 0 {
		return
	}

	buf := &bytes.Buffer{}

	if l.flags&FlagLocaltime == 0 {
		t = t.UTC()
	}

	if l.flags&FlagDate != 0 {
		year, month, day := t.Date()
		buf.WriteString(fmt.Sprintf("%04d-", year))
		buf.WriteString(fmt.Sprintf("%02d-", int(month)))
		buf.WriteString(fmt.Sprintf("%02d", day))
	}

	if l.flags&(FlagTime) != 0 {
		if l.flags&FlagDate != 0 {
			buf.WriteByte(' ')
		}

		hour, min, sec := t.Clock()
		buf.WriteString(fmt.Sprintf("%02d:", hour))
		buf.WriteString(fmt.Sprintf("%02d:", min))
		buf.WriteString(fmt.Sprintf("%02d", sec))

		if l.flags&FlagMicro != 0 {
			buf.WriteString(fmt.Sprintf(".%06d", t.Nanosecond()/1e3))
		}
	}
	fields[FieldTime] = buf.String()
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

// Router generates lines of output to Writers.
//
// A Router can be used simultaneously from multiple goroutines.
// It guarantees to serialize access to the Writer.
type Router struct {
	mu      sync.RWMutex
	outputs map[string]*output
}

// Output registers a Writer where lines should be written to.
func (l *Router) Output(id string, w io.Writer) {
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
	l.outputs[id] = r
}

type output struct {
	w io.Writer
}

// Output registers a Writer in the default Router.
func Output(id string, w io.Writer) {
	DefaultRouter.Output(id, w)
}

// Log writes a line to a Writer.
func (l *Router) Log(fields Fields) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, r := range l.outputs {
		if r.w != nil {
			b, err := json.Marshal(fields)
			if err != nil {
				//if onError != nil {
				//  onError(err, id)
				//}
				continue
			}

			writer := &writer{out: r.w}
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
