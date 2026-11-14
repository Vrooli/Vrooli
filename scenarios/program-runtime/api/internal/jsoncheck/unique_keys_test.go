package jsoncheck

import "testing"

func TestUniqueKeys(t *testing.T) {
	for _, test := range []struct {
		name, input string
		valid       bool
	}{
		{"nested objects and independent keys", `{"a":[{"x":1},{"x":2}],"b":{"x":3}}`, true},
		{"large numbers retain JSON syntax", `{"n":9007199254740993,"huge":1e10000}`, true},
		{"top level duplicate", `{"status":"failed","status":"ok"}`, false},
		{"escaped duplicate", `{"name":1,"na\u006de":2}`, false},
		{"nested duplicate", `{"items":[{"x":1,"x":2}]}`, false},
		{"trailing value", `{"a":1} {}`, false},
		{"truncated", `{"a":[1`, false},
		{"empty", ``, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := UniqueKeys([]byte(test.input)); (err == nil) != test.valid {
				t.Fatalf("UniqueKeys(%s) = %v; valid = %t", test.input, err, test.valid)
			}
		})
	}
}
