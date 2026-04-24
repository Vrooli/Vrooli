package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestGetConversationSession_SinceSequence_FiltersEvents verifies that
// ?since_sequence=N returns only events with sequence > N, so reconnect /
// view-open refresh can close gaps cheaply.
func TestGetConversationSession_SinceSequence_FiltersEvents(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	srv.conversations = NewConversationStore()

	for _, text := range []string{"one", "two", "three", "four"} {
		res := srv.appendConversationEvent(text, sess.ID, "unit-test")
		if !res.Appended {
			t.Fatalf("append %q failed: %+v", text, res)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sess.ID+"/conversation?since_sequence=2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	rec := httptest.NewRecorder()
	srv.handleGetConversationSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp conversationSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("expected 2 events after since_sequence=2, got %d", len(resp.Events))
	}
	for _, ev := range resp.Events {
		if ev.Sequence <= 2 {
			t.Errorf("expected every event sequence > 2, got %d (%q)", ev.Sequence, ev.Text)
		}
	}
}

// TestGetConversationSession_SinceSequence_IgnoresInvalid verifies that a
// malformed or negative since_sequence falls back to the full history —
// crucial so a bad query never silently hides messages.
func TestGetConversationSession_SinceSequence_IgnoresInvalid(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	srv.conversations = NewConversationStore()
	srv.appendConversationEvent("only", sess.ID, "unit-test")

	for _, q := range []string{"?since_sequence=abc", "?since_sequence=-5", "?since_sequence=0"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sess.ID+"/conversation"+q, nil)
		req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
		rec := httptest.NewRecorder()
		srv.handleGetConversationSession(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", q, rec.Code)
		}
		var resp conversationSessionResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("%s: decode: %v", q, err)
		}
		if len(resp.Events) != 1 {
			t.Errorf("%s: expected 1 event (full fallback), got %d", q, len(resp.Events))
		}
	}
}

// TestAppendConversation_DedupAcrossSources verifies the cross-source dedup
// fix: the same text from two different sources in the dedup window should
// be treated as the same semantic event (second call reports duplicate) so
// downstream subscribers don't get double-render.
func TestAppendConversation_DedupAcrossSources(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	srv.conversations = NewConversationStore()

	first := srv.appendConversationEvent("identical text", sess.ID, "codex_tailer")
	if !first.Appended || first.Duplicate {
		t.Fatalf("first append should be appended and non-duplicate: %+v", first)
	}

	second := srv.appendConversationEvent("identical text", sess.ID, "claude_hook")
	if !second.Duplicate {
		t.Fatalf("second append from different source should be flagged duplicate, got: %+v", second)
	}
	if second.EventID != first.EventID {
		t.Errorf("expected duplicate to resolve to original event %s, got %s", first.EventID, second.EventID)
	}

	// And only one event should actually be stored.
	state := srv.conversations.ListSession(sess.ID)
	if len(state.Events) != 1 {
		t.Errorf("expected exactly one stored event, got %d", len(state.Events))
	}
}
