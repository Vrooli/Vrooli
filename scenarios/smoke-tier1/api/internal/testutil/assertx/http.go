// Package assertx holds domain-aware test assertions.
//
// Two helpers cover the canonical handler test; scenarios add focused
// helpers (AssertHealthDependency, AssertTaskStatus, ...) as their
// domain grows. Resist generalising into a god-helper grab bag — small
// focused helpers stay readable; one mega-`AssertResponse` does not.
package assertx

import (
	"encoding/json"
	"net/http"
	"testing"
)

// failer is the minimum surface AssertStatus / MustDecodeJSON need from
// *testing.T. It exists so the helpers' fail paths can be exercised in
// this package's own tests with a recording stub — *testing.T can't be
// faked directly because testing.TB carries an unexported method.
type failer interface {
	Helper()
	Fatalf(format string, args ...any)
}

// AssertStatus fails the test if resp.StatusCode != want. On mismatch,
// it reports the actual code so the failure message is self-contained.
// Body inspection belongs in a separate assertion (e.g., MustDecodeJSON
// + field checks) — keeping AssertStatus focused makes its intent
// obvious at the callsite.
func AssertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	assertStatus(t, resp, want)
}

func assertStatus(t failer, resp *http.Response, want int) {
	t.Helper()
	if resp == nil {
		t.Fatalf("AssertStatus: response is nil (want %d)", want)
		return
	}
	if resp.StatusCode != want {
		t.Fatalf("AssertStatus: got %d, want %d", resp.StatusCode, want)
	}
}

// MustDecodeJSON decodes body into a value of type T or fails the test.
// Generic so callers get a typed result without an extra type
// assertion: `got := MustDecodeJSON[map[string]any](t, body)`.
func MustDecodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	return mustDecodeJSON[T](t, body)
}

func mustDecodeJSON[T any](t failer, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("MustDecodeJSON: %v\nbody: %s", err, string(body))
	}
	return v
}
