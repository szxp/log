package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
)

func TestLogRouter(t *testing.T) {
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
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := &Router{}
			buf := &bytes.Buffer{}
			r.Output("output1", buf)
			r.Log(tc.fields)

			actual := buf.String()
			if actual != tc.expected {
				t.Fatalf("expected %q, but got: %q", tc.expected, actual)
			}
		})
	}
}

func TestLoggerFlags(t *testing.T) {
	t.Parallel()

	dateRe := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	dateTimeRe := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$`)
	dateTimeMicroRe := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}$`)
	shortfileRe := regexp.MustCompile(`log_test.go:[0-9]+$`)
	longfileRe := regexp.MustCompile(`.+(\\|/)log_test.go:[0-9]+$`)

	type e struct {
		field   string
		pattern interface{}
	}

	testCases := []struct {
		testName   string
		loggerName string
		flags      int
		fields     Fields
		expected   []*e
	}{
		{"no flags", "logger", 0, nil, []*e{&e{"time", dateTimeMicroRe}}},
		{"std flags", "logger", FlagStd, nil, []*e{&e{"time", dateTimeMicroRe}}},
		{"date", "logger", FlagDate, nil, []*e{&e{"time", dateRe}}},
		{"date time", "logger", FlagDate | FlagTime, nil, []*e{&e{"time", dateTimeRe}}},
		{"date time micro", "logger", FlagDate | FlagTime | FlagMicro, nil, []*e{&e{"time", dateTimeMicroRe}}},
		{"logger name", "duck duck", FlagLogger, nil, []*e{&e{"logger", "duck duck"}}},
		{"short file line", "logger", FlagShortfile, nil, []*e{&e{"file", shortfileRe}}},
		{"long file line", "logger", FlagLongfile, nil, []*e{&e{"file", longfileRe}}},
		{"custom time1", "logger", 0, Fields{"time": "now1"}, []*e{&e{"time", "now1"}}},
		{"custom time2", "logger", FlagStd, Fields{"time": "now2"}, []*e{&e{"time", "now2"}}},
		{"custom time3", "logger", FlagDate, Fields{"time": "now3"}, []*e{&e{"time", "now3"}}},
		{"custom time4", "logger", FlagDate | FlagTime, Fields{"time": "now4"}, []*e{&e{"time", "now4"}}},
		{"custom time5", "logger", FlagDate | FlagTime | FlagMicro, Fields{"time": "now5"}, []*e{&e{"time", "now5"}}},
		{"custom logger name", "monkey", FlagLogger, Fields{"logger": "elephant"}, []*e{&e{"logger", "elephant"}}},
		{"custom short file line", "logger", FlagShortfile, Fields{"file": "line1"}, []*e{&e{"file", "line1"}}},
		{"custom long file line", "logger", FlagLongfile, Fields{"file": "line2"}, []*e{&e{"file", "line2"}}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			r := &Router{}
			buf := &bytes.Buffer{}
			r.Output("output1", buf)

			l := NewLogger(tc.loggerName, tc.flags, r)
			l.Log(tc.fields)

			obj := make(map[string]interface{})
			err := json.Unmarshal(buf.Bytes(), &obj)
			if err != nil {
				t.Fatal(err)
			}

			for _, e := range tc.expected {
				actual := obj[e.field]
				if actual == nil {
					t.Fatalf("field not found: %s", e.field)
				}

				if re, ok := e.pattern.(*regexp.Regexp); ok {
					if !re.MatchString(fmt.Sprintf("%s", actual)) {
						t.Fatalf("expected %s, but got %s", re.String(), actual)
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
