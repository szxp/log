package log

import (
	"bytes"
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
		t.Run(tc.name, func(t *testing.T) {
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
