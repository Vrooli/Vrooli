package apihttptest

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

func TestDoerFuncAndResponse(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.test/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	doer := DoerFunc(func(got *http.Request) (*http.Response, error) {
		if got.URL.Path != "/health" {
			t.Errorf("request path = %q, want /health", got.URL.Path)
		}
		return Response(http.StatusAccepted, "ready"), nil
	})
	resp, err := doer.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ready" {
		t.Fatalf("body = %q, want ready", string(body))
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

// MustUnmarshalProto exercises the generic proto.Message decoder.
// wrapperspb.StringValue is used as a stable, scenario-independent
// proto.Message — assertx must stay decoupled from any per-scenario
// generated type, so a well-known wrapper is the right test target.

func TestMustUnmarshalProto_Success(t *testing.T) {
	r := &recordingT{}
	got := mustUnmarshalProto[wrapperspb.StringValue](r, []byte(`"hello"`))
	if r.fatalCalled {
		t.Fatalf("MustUnmarshalProto on valid body unexpectedly fataled: %s", r.fatalMsg)
	}
	if got == nil {
		t.Fatal("MustUnmarshalProto returned nil pointer")
	}
	if got.Value != "hello" {
		t.Errorf("got.Value = %q, want hello", got.Value)
	}
}

func TestMustUnmarshalProto_FailureIncludesBody(t *testing.T) {
	r := &recordingT{}
	body := []byte(`{not proto`)
	_ = mustUnmarshalProto[wrapperspb.StringValue](r, body)
	if !r.fatalCalled {
		t.Fatal("malformed JSON should fatal")
	}
	if !strings.Contains(r.fatalMsg, string(body)) {
		t.Errorf("Fatalf message should include the body bytes for debugging; got %q", r.fatalMsg)
	}
}

// TestMustUnmarshalProto_TolerantToUnknownFields pins the
// DiscardUnknown:true contract — handler tests should keep passing
// when the wire grows fields the proto hasn't caught up to. The
// reverse (failing on every wire-shape addition) would force
// unrelated test churn whenever api-core/health gets a new field.
//
// google.protobuf.Empty has no fields by definition, so every property
// in the JSON object below is "unknown" from the proto's perspective.
// With DiscardUnknown:true the decode succeeds; without it,
// protojson.Unmarshal would error on the first unknown field.
func TestMustUnmarshalProto_TolerantToUnknownFields(t *testing.T) {
	r := &recordingT{}
	bodyWithUnknowns := []byte(`{"unknown":"field","another":42}`)
	_ = mustUnmarshalProto[emptypb.Empty](r, bodyWithUnknowns)
	if r.fatalCalled {
		t.Fatalf("DiscardUnknown was not wired; mustUnmarshalProto fataled on unknown fields: %s", r.fatalMsg)
	}
}
