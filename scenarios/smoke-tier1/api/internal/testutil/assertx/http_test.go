package assertx

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// recordingT is the spy *testing.T-compatible target the failer-using
// helpers fail into. testing.TB cannot be implemented externally (the
// private() method is the gate); the failer interface in http.go is
// the seam that lets us record fatal calls in-process.
type recordingT struct {
	helperCalls int
	fatalMsg    string
	fatalCalled bool
}

func (r *recordingT) Helper() { r.helperCalls++ }

func (r *recordingT) Fatalf(format string, args ...any) {
	r.fatalCalled = true
	r.fatalMsg = fmt.Sprintf(format, args...)
}

func TestAssertStatus_Match(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusOK}
	r := &recordingT{}
	assertStatus(r, resp, http.StatusOK)
	if r.fatalCalled {
		t.Fatalf("AssertStatus on matching code unexpectedly fataled: %s", r.fatalMsg)
	}
	if r.helperCalls == 0 {
		t.Error("AssertStatus must call Helper()")
	}
}

func TestAssertStatus_Mismatch(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError}
	r := &recordingT{}
	assertStatus(r, resp, http.StatusOK)
	if !r.fatalCalled {
		t.Fatal("AssertStatus on mismatch should fatal")
	}
	if !strings.Contains(r.fatalMsg, "got 500") || !strings.Contains(r.fatalMsg, "want 200") {
		t.Errorf("Fatalf message missing actual/expected codes: %q", r.fatalMsg)
	}
}

func TestAssertStatus_NilResponseFails(t *testing.T) {
	r := &recordingT{}
	assertStatus(r, nil, http.StatusOK)
	if !r.fatalCalled {
		t.Fatal("AssertStatus on nil response should fatal")
	}
	if !strings.Contains(r.fatalMsg, "nil") {
		t.Errorf("Fatalf message should mention nil response: %q", r.fatalMsg)
	}
}

func TestMustDecodeJSON_Success(t *testing.T) {
	r := &recordingT{}
	got := mustDecodeJSON[map[string]any](r, []byte(`{"a":1}`))
	if r.fatalCalled {
		t.Fatalf("decode unexpectedly fataled: %s", r.fatalMsg)
	}
	if got["a"].(float64) != 1 {
		t.Errorf("decoded value = %v, want a=1", got)
	}
}

func TestMustDecodeJSON_FailureIncludesBody(t *testing.T) {
	r := &recordingT{}
	body := []byte(`{not json`)
	_ = mustDecodeJSON[map[string]any](r, body)
	if !r.fatalCalled {
		t.Fatal("malformed JSON should fatal")
	}
	if !strings.Contains(r.fatalMsg, string(body)) {
		t.Errorf("Fatalf message should include the body bytes for debugging; got %q", r.fatalMsg)
	}
}
