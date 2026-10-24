package workspace

import (
	"encoding/json"
	"testing"
)

// A body that names its own session used to be parsed and silently dropped,
// so a complete-looking request failed with "missing required positional".
// The body form is now a valid way to express the whole request.
func TestPaneUpdateBodyCarriesSessionID(t *testing.T) {
	const raw = `{"session_id":"83f2b8ad-a8e3-4f0f-bfcb-f869bb180dd8","name":"Route solver","group_id":"g1"}`
	var body paneUpdateBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	if body.SessionID == nil {
		t.Fatal("session_id in the body was discarded; it must be decoded")
	}
	if *body.SessionID != "83f2b8ad-a8e3-4f0f-bfcb-f869bb180dd8" {
		t.Fatalf("session id = %q", *body.SessionID)
	}
	if body.Name == nil || *body.Name != "Route solver" {
		t.Fatal("name did not decode alongside session_id")
	}
}

// Absent session_id stays absent, so the positional remains required and the
// has_* semantics of every other field are unaffected.
func TestPaneUpdateBodyWithoutSessionID(t *testing.T) {
	var body paneUpdateBody
	if err := json.Unmarshal([]byte(`{"name":"Route solver"}`), &body); err != nil {
		t.Fatal(err)
	}
	if body.SessionID != nil {
		t.Fatalf("session id = %v, want nil", *body.SessionID)
	}
}
