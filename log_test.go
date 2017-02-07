package log_test

import (
	"bytes"
	"fmt"
	"github.com/szxp/log"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestJSONFormatter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		fields   log.Fields
		expected string
	}{
		{"empty", nil, `{}`},
		{"nil", log.Fields{"undefined": nil}, `{"undefined":null}`},
		{"string", log.Fields{"string": "message1"}, `{"string":"message1"}`},
		{"int", log.Fields{"number": 42}, `{"number":42}`},
		{"float", log.Fields{"number": 99.1}, `{"number":99.1}`},
		{"bool", log.Fields{"bool": true}, `{"bool":true}`},
		{"object", log.Fields{"object": log.Fields{"key1": "value1"}}, `{"object":{"key1":"value1"}}`},
		{"array_of_nils", log.Fields{"x": [2]interface{}{nil, nil}}, `{"x":[null,null]}`},
		{"array_of_strings", log.Fields{"x": [2]interface{}{"msg1", "msg2"}}, `{"x":["msg1","msg2"]}`},
		{"array_of_ints", log.Fields{"x": [2]interface{}{42, 82}}, `{"x":[42,82]}`},
		{"array_of_floats", log.Fields{"x": [2]interface{}{99.1, 33.1}}, `{"x":[99.1,33.1]}`},
		{"array_of_bools", log.Fields{"x": [2]interface{}{true, false}}, `{"x":[true,false]}`},
		{"array_of_objects", log.Fields{"x": [2]interface{}{log.Fields{"key1": "msg1"}, log.Fields{"key2": "msg2"}}},
			`{"x":[{"key1":"msg1"},{"key2":"msg2"}]}`},
		{"array_of_mixed", log.Fields{"x": [6]interface{}{nil, "msg", 42, 33.6, true, log.Fields{"key1": "msg1"}}},
			`{"x":[null,"msg",42,33.6,true,{"key1":"msg1"}]}`},
		{"slice_of_nils", log.Fields{"x": []interface{}{nil, nil}}, `{"x":[null,null]}`},
		{"slice_of_strings", log.Fields{"x": []interface{}{"msg1", "msg2"}}, `{"x":["msg1","msg2"]}`},
		{"slice_of_ints", log.Fields{"x": []interface{}{42, 82}}, `{"x":[42,82]}`},
		{"slice_of_floats", log.Fields{"x": []interface{}{99.1, 33.1}}, `{"x":[99.1,33.1]}`},
		{"slice_of_bools", log.Fields{"x": []interface{}{true, false}}, `{"x":[true,false]}`},
		{"slice_of_objects", log.Fields{"x": []interface{}{log.Fields{"key1": "msg1"}, log.Fields{"key2": "msg2"}}},
			`{"x":[{"key1":"msg1"},{"key2":"msg2"}]}`},
		{"slice_of_mixed", log.Fields{"x": []interface{}{nil, "msg", 42, 33.6, true, log.Fields{"key1": "msg1"}}},
			`{"x":[null,"msg",42,33.6,true,{"key1":"msg1"}]}`},
		{"ignore underscore", log.Fields{"_ignoreit": "abc"}, `{}`},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := log.JSONFormatter{}
			b, err := f.Format(tc.fields)
			if err != nil {
				t.Fatalf("non-nil error: %v", err)
			}

			if !bytes.Equal(b, []byte(tc.expected)) {
				t.Fatalf("expected %q, but got: %q", tc.expected, string(b))
			}
		})
	}
}

func TestLogger(t *testing.T) {
	t.Parallel()

	rfc3339Re := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(Z|[+-][0-9]{2}:[0-9]{2})$`)
	shortfileRe := regexp.MustCompile(`log_test.go:[0-9]+$`)
	longfileRe := regexp.MustCompile(`.+(\\|/)log_test.go:[0-9]+$`)

	type e struct {
		field   string
		pattern interface{}
	}

	testCases := []struct {
		testName string
		config   log.LoggerConfig
		fields   log.Fields
		expected []*e
	}{
		{"rfc3339", log.LoggerConfig{TimeFormat: time.RFC3339}, nil, []*e{{"time", rfc3339Re}}},
		{"rfc3339 utc", log.LoggerConfig{TimeFormat: time.RFC3339, UTC: true}, nil, []*e{{"time", rfc3339Re}}},
		{"logger name", log.LoggerConfig{Name: "duck"}, nil, []*e{{"logger", "duck"}}},
		{"short file line", log.LoggerConfig{FileLine: log.ShortFileLine}, nil, []*e{{"file", shortfileRe}}},
		{"long file line", log.LoggerConfig{FileLine: log.LongFileLine}, nil, []*e{{"file", longfileRe}}},
		{"rfc3339 logger", log.LoggerConfig{Name: "logger", TimeFormat: time.RFC3339}, nil, []*e{{"time", rfc3339Re}, {"logger", "logger"}}},
		{"sort fields1", log.LoggerConfig{SortFields: true}, nil, []*e{{"_sort", true}}},
		{"sort fields2", log.LoggerConfig{SortFields: false}, nil, []*e{{"_sort", false}}},
		{"custom time", log.LoggerConfig{TimeFormat: time.RFC3339}, log.Fields{"time": "now1"}, []*e{{"time", "now1"}}},
		{"custom logger name", log.LoggerConfig{Name: "monkey"}, log.Fields{"logger": "elephant"}, []*e{{"logger", "elephant"}}},
		{"custom short file line", log.LoggerConfig{FileLine: log.ShortFileLine}, log.Fields{"file": "line1"}, []*e{{"file", "line1"}}},
		{"custom long file line", log.LoggerConfig{FileLine: log.LongFileLine}, log.Fields{"file": "line2"}, []*e{{"file", "line2"}}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			spy := &routerSpy{}
			tc.config.Router = spy
			l := tc.config.NewLogger()
			l.Log(tc.fields)

			for _, e := range tc.expected {
				actual, ok := spy.fields[e.field]
				if !ok {
					t.Fatalf("field not found: %s", e.field)
				}

				if re, ok := e.pattern.(*regexp.Regexp); ok {
					if !re.MatchString(fmt.Sprintf("%v", actual)) {
						t.Fatalf("expected %v, but got %v", re.String(), actual)
					}
				} else {
					if actual != e.pattern {
						t.Fatalf("expected %v, but got %v", e.pattern, actual)
					}
				}
			}
		})
	}
}

// no goroutine safe
type routerSpy struct {
	fields log.Fields
}

func (r *routerSpy) Log(fields log.Fields) {
	r.fields = fields
}

func TestFiltersComposite(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testName string
		filter   log.Filter
		expected bool
	}{
		{"and1", log.And(&mockFilter{r: false}, &mockFilter{r: false}), false},
		{"and2", log.And(&mockFilter{r: false}, &mockFilter{r: true}), false},
		{"and3", log.And(&mockFilter{r: true}, &mockFilter{r: false}), false},
		{"and4", log.And(&mockFilter{r: true}, &mockFilter{r: true}), true},

		{"or1", log.Or(&mockFilter{r: false}, &mockFilter{r: false}), false},
		{"or2", log.Or(&mockFilter{r: false}, &mockFilter{r: true}), true},
		{"or3", log.Or(&mockFilter{r: true}, &mockFilter{r: false}), true},
		{"or4", log.Or(&mockFilter{r: true}, &mockFilter{r: true}), true},

		{"not1", log.Not(&mockFilter{r: true}), false},
		{"not2", log.Not(&mockFilter{r: false}), true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			match, err := tc.filter.Match(log.Fields{})
			if err != nil {
				t.Fatalf("non nil error: %v", err)
			}
			if match != tc.expected {
				t.Fatalf("expected %v, but got %v", tc.expected, match)
			}
		})
	}
}

type mockFilter struct {
	r   bool
	n   string
	buf *bytes.Buffer
}

func (m *mockFilter) Match(fields log.Fields) (bool, error) {
	if m.buf != nil {
		m.buf.Write([]byte(m.n))
	}
	return m.r, nil
}

func TestFilters(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testName string
		filter   log.Filter
		fields   log.Fields
		expected bool
	}{
		{"field exist", log.FieldExist("time"), log.Fields{"time": 123}, true},
		{"field not exist", log.FieldExist("time2"), log.Fields{"time": 123}, false},
		{"field exist dotpath", log.FieldExist("user.id"), log.Fields{"user": log.Fields{"id": 1}}, true},
		{"field not exist dotpath", log.FieldExist("user.username"), log.Fields{"user": log.Fields{"id": 1}}, false},

		{"eq string", log.Eq("logger", "requestLogger"), log.Fields{"logger": "requestLogger"}, true},
		{"not eq string", log.Eq("logger", "requestLogger2"), log.Fields{"logger": "requestLogger"}, false},
		{"eq string dotpath", log.Eq("user.id", 1), log.Fields{"user": log.Fields{"id": 1}}, true},
		{"not eq string dotpath", log.Eq("user.id", 2), log.Fields{"user": log.Fields{"id": 1}}, false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			match, err := tc.filter.Match(tc.fields)
			if err != nil {
				t.Fatalf("non nil error: %v", err)
			}
			if match != tc.expected {
				t.Fatalf("expected %v, but got %v", tc.expected, match)
			}
		})
	}
}

func TestAndShortCircuit(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testName      string
		filters       []log.Filter
		expectedMatch bool
		expectedOrder string
	}{
		{"and1", []log.Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: false, n: "B"}}, false, "A"},
		{"and2", []log.Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: true, n: "B"}}, false, "A"},
		{"and3", []log.Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: false, n: "B"}}, false, "AB"},
		{"and4", []log.Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: true, n: "B"}}, true, "AB"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			buf := &bytes.Buffer{}
			for _, f := range tc.filters {
				(f.(*mockFilter)).buf = buf
			}

			match, err := log.And(tc.filters...).Match(log.Fields{})
			if err != nil {
				t.Fatalf("non nil error: %v", err)
			}
			if match != tc.expectedMatch {
				t.Fatalf("expected %v, but got %v", tc.expectedMatch, match)
			}
			if buf.String() != tc.expectedOrder {
				t.Fatalf("expected %v, but got %v", tc.expectedOrder, buf.String())
			}
		})
	}
}

func TestOrShortCircuit(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testName      string
		filters       []log.Filter
		expectedMatch bool
		expectedOrder string
	}{
		{"or1", []log.Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: false, n: "B"}}, false, "AB"},
		{"or2", []log.Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: true, n: "B"}}, true, "AB"},
		{"or3", []log.Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: false, n: "B"}}, true, "A"},
		{"or4", []log.Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: true, n: "B"}}, true, "A"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			buf := &bytes.Buffer{}
			for _, f := range tc.filters {
				(f.(*mockFilter)).buf = buf
			}

			match, err := log.Or(tc.filters...).Match(log.Fields{})
			if err != nil {
				t.Fatalf("non nil error: %v", err)
			}
			if match != tc.expectedMatch {
				t.Fatalf("expected %v, but got %v", tc.expectedMatch, match)
			}
			if buf.String() != tc.expectedOrder {
				t.Fatalf("expected %v, but got %v", tc.expectedOrder, buf.String())
			}
		})
	}
}

func ExampleOutput() {
	// register an io.Writer in the DefaultRouter
	// everything that is not a debug message will be written to stdout
	log.Output{
		Id:        "stdout1",
		Writer:    os.Stdout,
		Formatter: nil,
		Filter:    log.Not(log.Eq("level", "debug")),
	}.Register()
}

func ExampleOnError() {
	log.OnError(func(err error, fields log.Fields, o log.Output) {
		fmt.Printf("%v: %+v: %+v", err, fields, o)
	})
}

func ExampleLoggerConfig() {
	logger := log.LoggerConfig{
		TimeFormat: time.RFC3339,      // optional, see standard time package for custom formats
		Name:       "loggername",      // optional, name of the logger
		UTC:        true,              // optional, use UTC rather than local time zone
		FileLine:   log.ShortFileLine, // optional, include file and line number
		SortFields: true,              // optional, sort field keys in increasing order
		Router:     nil,               // optional, defaults to log.DefaultRouter
	}.NewLogger()

	logger.Log(log.Fields{
		"level": "info",
		"user": log.Fields{
			"id":       1,
			"username": "admin",
		},
		"activated": true,
		"projects":  []string{"p1", "p2", "p3"},
	})
}

func ExampleFields() {
	logger := log.LoggerConfig{
		TimeFormat: time.RFC3339,      // optional, see standard time package for custom formats
		Name:       "loggername",      // optional, name of the logger
		UTC:        true,              // optional, use UTC rather than local time zone
		FileLine:   log.ShortFileLine, // optional, include file and line number
		SortFields: true,              // optional, sort field keys in increasing order
		Router:     nil,               // optional, defaults to log.DefaultRouter
	}.NewLogger()

	logger.Log(log.Fields{
		"level": "info",
		"user": log.Fields{
			"id":       1,
			"username": "admin",
		},
		"activated": true,
		"projects":  []string{"p1", "p2", "p3"},
	})
}

func ExampleAnd() {
	log.Output{
		Id:     "stdout1",
		Writer: os.Stdout,
		Filter: log.And(
			log.Eq("user.id", 1),
			log.Eq("level", "trace"),
		),
	}.Register()
}

func ExampleOr() {
	log.Output{
		Id:     "stdout1",
		Writer: os.Stdout,
		Filter: log.Or(
			log.Eq("username", "admin"),
			log.Eq("username", "bot"),
		),
	}.Register()
}

func ExampleNot() {
	log.Output{
		Id:     "stdout1",
		Writer: os.Stdout,
		Filter: log.Not(log.Eq("user.id", 1)),
	}.Register()
}

func ExampleEq() {
	log.Output{
		Id:     "stdout1",
		Writer: os.Stdout,
		Filter: log.Eq("use.id", 1),
	}.Register()
}

func ExampleFieldExist() {
	log.Output{
		Id:     "stdout1",
		Writer: os.Stdout,
		Filter: log.FieldExist("user.id"),
	}.Register()
}

// Run only this benchmark: go test -bench Logger -run ^$
func BenchmarkLogger(b *testing.B) {
	log.Output{
		Id:        "buf1",
		Writer:    &bytes.Buffer{},
		Formatter: &log.JSONFormatter{},
		Filter:    log.FieldExist("bench"),
	}.Register()

	logger := log.LoggerConfig{
		TimeFormat: time.RFC3339,
		Name:       "loggername",
		UTC:        true,
		FileLine:   log.ShortFileLine,
		SortFields: true,
		Router:     nil,
	}.NewLogger()

	fields := log.Fields{
		"bench": true,
		"level": "info",
		"user": log.Fields{
			"id":       1,
			"username": "admin",
		},
		"activated": true,
		"projects":  []string{"p1", "p2", "p3"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(fields)

	}
}
