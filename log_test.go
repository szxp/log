package log

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"testing"
	"time"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		fields   Fields
		expected string
	}{
		{"empty", nil, `{}` + "\n"},
		{"nil", Fields{"undefined": nil}, `{"undefined":null}` + "\n"},
		{"string", Fields{"string": "message1"}, `{"string":"message1"}` + "\n"},
		{"int", Fields{"number": 42}, `{"number":42}` + "\n"},
		{"float", Fields{"number": 99.1}, `{"number":99.1}` + "\n"},
		{"bool", Fields{"bool": true}, `{"bool":true}` + "\n"},
		{"object", Fields{"object": Fields{"key1": "value1"}}, `{"object":{"key1":"value1"}}` + "\n"},
		{"array_of_nils", Fields{"x": [2]interface{}{nil, nil}}, `{"x":[null,null]}` + "\n"},
		{"array_of_strings", Fields{"x": [2]interface{}{"msg1", "msg2"}}, `{"x":["msg1","msg2"]}` + "\n"},
		{"array_of_ints", Fields{"x": [2]interface{}{42, 82}}, `{"x":[42,82]}` + "\n"},
		{"array_of_floats", Fields{"x": [2]interface{}{99.1, 33.1}}, `{"x":[99.1,33.1]}` + "\n"},
		{"array_of_bools", Fields{"x": [2]interface{}{true, false}}, `{"x":[true,false]}` + "\n"},
		{"array_of_objects", Fields{"x": [2]interface{}{Fields{"key1": "msg1"}, Fields{"key2": "msg2"}}},
			`{"x":[{"key1":"msg1"},{"key2":"msg2"}]}` + "\n"},
		{"array_of_mixed", Fields{"x": [6]interface{}{nil, "msg", 42, 33.6, true, Fields{"key1": "msg1"}}},
			`{"x":[null,"msg",42,33.6,true,{"key1":"msg1"}]}` + "\n"},
		{"slice_of_nils", Fields{"x": []interface{}{nil, nil}}, `{"x":[null,null]}` + "\n"},
		{"slice_of_strings", Fields{"x": []interface{}{"msg1", "msg2"}}, `{"x":["msg1","msg2"]}` + "\n"},
		{"slice_of_ints", Fields{"x": []interface{}{42, 82}}, `{"x":[42,82]}` + "\n"},
		{"slice_of_floats", Fields{"x": []interface{}{99.1, 33.1}}, `{"x":[99.1,33.1]}` + "\n"},
		{"slice_of_bools", Fields{"x": []interface{}{true, false}}, `{"x":[true,false]}` + "\n"},
		{"slice_of_objects", Fields{"x": []interface{}{Fields{"key1": "msg1"}, Fields{"key2": "msg2"}}},
			`{"x":[{"key1":"msg1"},{"key2":"msg2"}]}` + "\n"},
		{"slice_of_mixed", Fields{"x": []interface{}{nil, "msg", 42, 33.6, true, Fields{"key1": "msg1"}}},
			`{"x":[null,"msg",42,33.6,true,{"key1":"msg1"}]}` + "\n"},
		{"ignore underscore", Fields{"_ignoreit": "abc"}, `{}` + "\n"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := NewRouter()
			buf := &bytes.Buffer{}
			r.Output("output1", buf, nil, nil)
			r.Log(tc.fields)

			actual := buf.String()
			if actual != tc.expected {
				t.Fatalf("expected %q, but got: %q", tc.expected, actual)
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
		config   Config
		fields   Fields
		expected []*e
	}{
		{"rfc3339", Config{TimeFormat: time.RFC3339}, nil, []*e{{"time", rfc3339Re}}},
		{"rfc3339 utc", Config{TimeFormat: time.RFC3339, UTC: true}, nil, []*e{{"time", rfc3339Re}}},
		{"logger name", Config{Name: "duck"}, nil, []*e{{"logger", "duck"}}},
		{"short file line", Config{FileLine: ShortFileLine}, nil, []*e{{"file", shortfileRe}}},
		{"long file line", Config{FileLine: LongFileLine}, nil, []*e{{"file", longfileRe}}},
		{"rfc3339 logger", Config{Name: "logger", TimeFormat: time.RFC3339}, nil, []*e{{"time", rfc3339Re}, {"logger", "logger"}}},
		{"sort fields1", Config{SortFields: true}, nil, []*e{{"_sort", true}}},
		{"sort fields2", Config{SortFields: false}, nil, []*e{{"_sort", false}}},
		{"custom time", Config{TimeFormat: time.RFC3339}, Fields{"time": "now1"}, []*e{{"time", "now1"}}},
		{"custom logger name", Config{Name: "monkey"}, Fields{"logger": "elephant"}, []*e{{"logger", "elephant"}}},
		{"custom short file line", Config{FileLine: ShortFileLine}, Fields{"file": "line1"}, []*e{{"file", "line1"}}},
		{"custom long file line", Config{FileLine: LongFileLine}, Fields{"file": "line2"}, []*e{{"file", "line2"}}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			spy := &routerSpy{}
			tc.config.Router = spy
			l := NewLogger(tc.config)
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
	fields Fields
}

func (r *routerSpy) Output(id string, w io.Writer, formatter Formatter, filter Filter) {}

func (r *routerSpy) Log(fields Fields) {
	r.fields = fields
}

func (r *routerSpy) OnError(f func(id string, w io.Writer, err error)) {}

func TestFiltersComposite(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testName string
		filter   Filter
		expected bool
	}{
		{"and1", And(&mockFilter{r: false}, &mockFilter{r: false}), false},
		{"and2", And(&mockFilter{r: false}, &mockFilter{r: true}), false},
		{"and3", And(&mockFilter{r: true}, &mockFilter{r: false}), false},
		{"and4", And(&mockFilter{r: true}, &mockFilter{r: true}), true},

		{"or1", Or(&mockFilter{r: false}, &mockFilter{r: false}), false},
		{"or2", Or(&mockFilter{r: false}, &mockFilter{r: true}), true},
		{"or3", Or(&mockFilter{r: true}, &mockFilter{r: false}), true},
		{"or4", Or(&mockFilter{r: true}, &mockFilter{r: true}), true},

		{"not1", Not(&mockFilter{r: true}), false},
		{"not2", Not(&mockFilter{r: false}), true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			match, err := tc.filter.Match(Fields{})
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

func (m *mockFilter) Match(fields Fields) (bool, error) {
	if m.buf != nil {
		m.buf.Write([]byte(m.n))
	}
	return m.r, nil
}

func TestFilters(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testName string
		filter   Filter
		fields   Fields
		expected bool
	}{
		{"field exist", FieldExist("time"), Fields{"time": 123}, true},
		{"field not exist", FieldExist("time2"), Fields{"time": 123}, false},
		{"field exist dotpath", FieldExist("user.id"), Fields{"user": Fields{"id": 1}}, true},
		{"field not exist dotpath", FieldExist("user.username"), Fields{"user": Fields{"id": 1}}, false},

		{"eq string", Eq("logger", "requestLogger"), Fields{"logger": "requestLogger"}, true},
		{"not eq string", Eq("logger", "requestLogger2"), Fields{"logger": "requestLogger"}, false},
		{"eq string dotpath", Eq("user.id", 1), Fields{"user": Fields{"id": 1}}, true},
		{"not eq string dotpath", Eq("user.id", 2), Fields{"user": Fields{"id": 1}}, false},
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
		filters       []Filter
		expectedMatch bool
		expectedOrder string
	}{
		{"and1", []Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: false, n: "B"}}, false, "A"},
		{"and2", []Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: true, n: "B"}}, false, "A"},
		{"and3", []Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: false, n: "B"}}, false, "AB"},
		{"and4", []Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: true, n: "B"}}, true, "AB"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			buf := &bytes.Buffer{}
			for _, f := range tc.filters {
				(f.(*mockFilter)).buf = buf
			}

			match, err := And(tc.filters...).Match(Fields{})
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
		filters       []Filter
		expectedMatch bool
		expectedOrder string
	}{
		{"or1", []Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: false, n: "B"}}, false, "AB"},
		{"or2", []Filter{&mockFilter{r: false, n: "A"}, &mockFilter{r: true, n: "B"}}, true, "AB"},
		{"or3", []Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: false, n: "B"}}, true, "A"},
		{"or4", []Filter{&mockFilter{r: true, n: "A"}, &mockFilter{r: true, n: "B"}}, true, "A"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			buf := &bytes.Buffer{}
			for _, f := range tc.filters {
				(f.(*mockFilter)).buf = buf
			}

			match, err := Or(tc.filters...).Match(Fields{})
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
