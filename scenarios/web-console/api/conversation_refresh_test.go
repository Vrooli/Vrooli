package main

import (
	"context"
	"testing"
	"time"

	"web-console/internal/sessionstore"

	"connectrpc.com/connect"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
)

// TestGetConversationSession_AllowsArchivedTranscript verifies that the
// read-only archive viewer can hydrate a transcript after the live session
// manager has intentionally released the PTY. Archived metadata is the
// authority for this read path; the session must not be made live again.
func TestGetConversationSession_AllowsArchivedTranscript(t *testing.T) { // [REQ:REQ-P0-003c]
	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	srv.conversations = NewConversationStore()

	const sessionID = "archived-transcript"
	if err := srv.sessionStore.Save(context.Background(), sessionstore.Metadata{
		ID:         sessionID,
		Status:     sessionstore.StatusDismissed,
		ArchivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("save archived metadata: %v", err)
	}
	if _, result := srv.conversations.AppendAssistantEvent(context.Background(), sessionID, "unit-test", "persisted archive message"); !result.Appended {
		t.Fatalf("append archived event: %+v", result)
	}

	resp, err := newConversationConnectHandlerForServer(srv).Get(
		context.Background(),
		connect.NewRequest(&conversationv1.GetRequest{SessionId: sessionID, Limit: 500}),
	)
	if err != nil {
		t.Fatalf("Get archived transcript: %v", err)
	}
	if got := resp.Msg.GetEvents(); len(got) != 1 || got[0].GetText() != "persisted archive message" {
		t.Fatalf("archived events = %+v, want persisted message", got)
	}
	if _, ok := srv.sessions.Get(sessionID); ok {
		t.Fatal("archive read unexpectedly restored a live session")
	}
}

// TestGetConversationSession_SinceSequence_FiltersEvents verifies that
// since_sequence=N returns only events with sequence > N, so reconnect /
// view-open refresh can close gaps cheaply.
func TestGetConversationSession_SinceSequence_FiltersEvents(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	srv.conversations = NewConversationStore()

	for _, text := range []string{"one", "two", "three", "four"} {
		res := srv.AppendAssistant(text, sess.ID, "unit-test")
		if !res.Appended {
			t.Fatalf("append %q failed: %+v", text, res)
		}
	}

	resp, err := newConversationConnectHandlerForServer(srv).Get(
		context.Background(),
		connect.NewRequest(&conversationv1.GetRequest{SessionId: sess.ID, SinceSequence: 2}),
	)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(resp.Msg.GetEvents()) != 2 {
		t.Fatalf("expected 2 events after since_sequence=2, got %d", len(resp.Msg.GetEvents()))
	}
	for _, ev := range resp.Msg.GetEvents() {
		if ev.GetSequence() <= 2 {
			t.Errorf("expected every event sequence > 2, got %d (%q)", ev.GetSequence(), ev.GetText())
		}
	}
}

// TestGetConversationSession_SinceSequence_IgnoresInvalid verifies that a
// non-positive since_sequence falls back to the full history — crucial so a
// bad value never silently hides messages. (Malformed string inputs are now
// rejected at the proto layer rather than silently ignored, so this test
// only exercises the documented 0 / negative fallback path.)
func TestGetConversationSession_SinceSequence_IgnoresInvalid(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	srv.conversations = NewConversationStore()
	srv.AppendAssistant("only", sess.ID, "unit-test")

	for _, since := range []int64{-5, 0} {
		resp, err := newConversationConnectHandlerForServer(srv).Get(
			context.Background(),
			connect.NewRequest(&conversationv1.GetRequest{SessionId: sess.ID, SinceSequence: since}),
		)
		if err != nil {
			t.Fatalf("since=%d: %v", since, err)
		}
		if len(resp.Msg.GetEvents()) != 1 {
			t.Errorf("since=%d: expected 1 event (full fallback), got %d", since, len(resp.Msg.GetEvents()))
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

	first := srv.AppendAssistant("identical text", sess.ID, "codex_tailer")
	if !first.Appended || first.Duplicate {
		t.Fatalf("first append should be appended and non-duplicate: %+v", first)
	}

	second := srv.AppendAssistant("identical text", sess.ID, "claude_hook")
	if !second.Duplicate {
		t.Fatalf("second append from different source should be flagged duplicate, got: %+v", second)
	}
	if second.EventID != first.EventID {
		t.Errorf("expected duplicate to resolve to original event %s, got %s", first.EventID, second.EventID)
	}

	state := srv.conversations.ListSession(context.Background(), sess.ID)
	if len(state.Events) != 1 {
		t.Errorf("expected exactly one stored event, got %d", len(state.Events))
	}
}
