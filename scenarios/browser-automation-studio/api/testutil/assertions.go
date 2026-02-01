// Package testutil provides testing utilities and assertions.
package testutil

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// AssertEqual checks if two values are equal.
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected: %v\nActual:   %v", msg, expected, actual)
	}
}

// AssertNotEqual checks if two values are not equal.
func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected value to not equal: %v", msg, actual)
	}
}

// AssertNil checks if a value is nil.
func AssertNil(t *testing.T, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if actual != nil && !isNilValue(actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected nil, got: %v", msg, actual)
	}
}

// AssertNotNil checks if a value is not nil.
func AssertNotNil(t *testing.T, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if actual == nil || isNilValue(actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected non-nil value", msg)
	}
}

// AssertTrue checks if a value is true.
func AssertTrue(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !condition {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected true, got false", msg)
	}
}

// AssertFalse checks if a value is false.
func AssertFalse(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if condition {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected false, got true", msg)
	}
}

// AssertNoError checks if an error is nil.
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nUnexpected error: %v", msg, err)
	}
}

// AssertError checks if an error is not nil.
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected an error but got nil", msg)
	}
}

// AssertErrorContains checks if an error message contains a substring.
func AssertErrorContains(t *testing.T, err error, contains string, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected an error but got nil", msg)
		return
	}
	if !strings.Contains(err.Error(), contains) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nError %q does not contain %q", msg, err.Error(), contains)
	}
}

// AssertContains checks if a string contains a substring.
func AssertContains(t *testing.T, s, contains string, msgAndArgs ...interface{}) {
	t.Helper()
	if !strings.Contains(s, contains) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nString %q does not contain %q", msg, s, contains)
	}
}

// AssertLen checks if a slice/map/string has expected length.
func AssertLen(t *testing.T, object interface{}, length int, msgAndArgs ...interface{}) {
	t.Helper()
	actualLen := getLen(object)
	if actualLen != length {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected length %d, got %d", msg, length, actualLen)
	}
}

// AssertEmpty checks if a slice/map/string is empty.
func AssertEmpty(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	actualLen := getLen(object)
	if actualLen != 0 {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected empty, got length %d", msg, actualLen)
	}
}

// AssertNotEmpty checks if a slice/map/string is not empty.
func AssertNotEmpty(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	actualLen := getLen(object)
	if actualLen == 0 {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected non-empty", msg)
	}
}

// AssertTimeEqual checks if two times are equal (within 1 second tolerance for test stability).
func AssertTimeEqual(t *testing.T, expected, actual time.Time, msgAndArgs ...interface{}) {
	t.Helper()
	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected time: %v\nActual time:   %v\nDifference:    %v", msg, expected, actual, diff)
	}
}

// AssertTimeApprox checks if two times are within a given tolerance.
func AssertTimeApprox(t *testing.T, expected, actual time.Time, tolerance time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected time: %v\nActual time:   %v\nDifference:    %v (tolerance: %v)", msg, expected, actual, diff, tolerance)
	}
}

// AssertJSONEqual checks if two JSON values are equivalent.
func AssertJSONEqual(t *testing.T, expected, actual string, msgAndArgs ...interface{}) {
	t.Helper()
	var expectedVal, actualVal interface{}
	if err := json.Unmarshal([]byte(expected), &expectedVal); err != nil {
		t.Errorf("Failed to parse expected JSON: %v", err)
		return
	}
	if err := json.Unmarshal([]byte(actual), &actualVal); err != nil {
		t.Errorf("Failed to parse actual JSON: %v", err)
		return
	}
	if !reflect.DeepEqual(expectedVal, actualVal) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%s\nExpected JSON: %s\nActual JSON:   %s", msg, expected, actual)
	}
}

// RequireNoError checks if an error is nil and fails the test immediately if not.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		msg := formatMessage(msgAndArgs...)
		t.Fatalf("%s\nUnexpected error: %v", msg, err)
	}
}

// RequireNotNil checks if a value is not nil and fails the test immediately if it is.
func RequireNotNil(t *testing.T, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if actual == nil || isNilValue(actual) {
		msg := formatMessage(msgAndArgs...)
		t.Fatalf("%s\nExpected non-nil value", msg)
	}
}

// RequireEqual checks if two values are equal and fails the test immediately if not.
func RequireEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		msg := formatMessage(msgAndArgs...)
		t.Fatalf("%s\nExpected: %v\nActual:   %v", msg, expected, actual)
	}
}

// formatMessage formats optional message and args for test output.
func formatMessage(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return "Assertion failed"
	}
	if len(msgAndArgs) == 1 {
		if msg, ok := msgAndArgs[0].(string); ok {
			return msg
		}
		return "Assertion failed"
	}
	format, ok := msgAndArgs[0].(string)
	if !ok {
		return "Assertion failed"
	}
	// Safely handle formatting
	return format
}

// isNilValue checks if a value is nil using reflection.
func isNilValue(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

// getLen returns the length of a slice, map, string, or array.
func getLen(object interface{}) int {
	if object == nil {
		return 0
	}
	rv := reflect.ValueOf(object)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.String, reflect.Array, reflect.Chan:
		return rv.Len()
	}
	return 0
}
