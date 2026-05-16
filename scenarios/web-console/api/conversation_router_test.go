package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web-console/session"

	"connectrpc.com/connect"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"

	inttts "web-console/internal/tts"
)

func callSummarizeEvent(t *testing.T, srv *Server, sessID, eventID string) *conversationv1.SummarizeEventResponse {
	t.Helper()
	resp, err := newConversationConnectHandlerForServer(srv).SummarizeEvent(
		context.Background(),
		connect.NewRequest(&conversationv1.SummarizeEventRequest{SessionId: sessID, EventId: eventID}),
	)
	if err != nil {
		t.Fatalf("SummarizeEvent: %v", err)
	}
	return resp.Msg
}

func TestAppendConversationEvent_MissingSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendAssistant("hello", "", "test")
	if result.Appended {
		t.Error("expected failure for empty session")
	}
	if result.Code != "conversation_target_missing" {
		t.Errorf("expected code conversation_target_missing, got %s", result.Code)
	}
}

func TestAppendConversationEvent_UnknownSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendAssistant("hello", "nonexistent", "test")
	if result.Appended {
		t.Error("expected failure for unknown session")
	}
}

func TestAppendUserConversationEvent_MissingSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendUser("hello", "", "test")
	if result.Appended {
		t.Error("expected failure for empty session")
	}
}

func TestAppendUserConversationEvent_UnknownSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.AppendUser("hello", "nonexistent", "test")
	if result.Appended {
		t.Error("expected failure for unknown session")
	}
}

// newSummarizeTestServer wires up the server fields exercised by
// asyncSummarizeAndNotify and the on-demand summarize handler.
func newSummarizeTestServer(t *testing.T, ollama *httptest.Server) (*Server, *session.Session, string, string) {
	t.Helper()
	srv := newFakeTestServer()
	srv.conversations = NewConversationStore()
	srv.ttsCache = inttts.NewCache(1024 * 1024)
	srv.ttsSummarizer = inttts.NewSummarizer(ollama.URL)
	srv.ttsSummarization = inttts.NewSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)
	srv.ttsSummarizeConfig = inttts.SummarizeConfig{
		Enabled:        true,
		CharThreshold:  20,
		Level:          "moderate",
		Model:          "test-model",
		TimeoutSeconds: 5,
	}

	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(sess.ID) })

	longText := strings.Repeat("This is some assistant output that should be summarized. ", 5)
	event, result := srv.conversations.AppendAssistantEvent(sess.ID, "test", longText)
	if !result.Appended {
		t.Fatalf("failed to append event: %+v", result)
	}

	// Seed the cache as if pre-synthesis already ran against the raw text.
	rawKey := inttts.CacheKey{EventID: event.ID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	srv.ttsCache.Put(rawKey, []byte("raw-audio"), "audio/mpeg")

	return srv, sess, event.ID, longText
}

func TestAsyncSummarizeAndNotify_EvictsCacheOnSuccess(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "Short summary."},
		})
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	// Grab the event to hand to asyncSummarizeAndNotify.
	event, ok := srv.conversations.GetEvent(sess.ID, eventID)
	if !ok {
		t.Fatal("event missing from store")
	}

	srv.asyncSummarizeAndNotify(event, sess.ID)

	// Cache for the raw audio must be gone.
	rawKey := inttts.CacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(rawKey); ok {
		t.Error("expected raw-text cache entry to be evicted after summarization")
	}

	// Paragraphs on the stored event should reflect the summary.
	updated, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if !updated.Summarized {
		t.Error("expected Summarized=true after successful summarization")
	}
	if len(updated.SpeechParagraphs) == 0 {
		t.Fatal("expected summary paragraphs to be written")
	}
	if !strings.Contains(updated.SpeechParagraphs[0], "Short summary") {
		t.Errorf("expected summary paragraph to contain 'Short summary', got %q", updated.SpeechParagraphs[0])
	}
}

func TestAsyncSummarizeAndNotify_PreservesCacheOnError(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	event, _ := srv.conversations.GetEvent(sess.ID, eventID)
	before := append([]string(nil), event.SpeechParagraphs...)

	srv.asyncSummarizeAndNotify(event, sess.ID)

	// Cache must still hold the seeded raw audio — summary never succeeded.
	rawKey := inttts.CacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(rawKey); !ok {
		t.Error("raw-text cache entry should survive when summarization fails")
	}

	updated, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if updated.Summarized {
		t.Error("event should not be marked summarized after failure")
	}
	if len(updated.SpeechParagraphs) != len(before) {
		t.Errorf("paragraphs changed despite summarization failure: got %d, want %d",
			len(updated.SpeechParagraphs), len(before))
	}
}

func TestAsyncSummarizeAndNotify_PreservesCacheOnEmpty(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "   "}, // whitespace collapses to empty
		})
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	event, _ := srv.conversations.GetEvent(sess.ID, eventID)
	srv.asyncSummarizeAndNotify(event, sess.ID)

	rawKey := inttts.CacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(rawKey); !ok {
		t.Error("empty summary should not evict the cache")
	}
	updated, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if updated.Summarized {
		t.Error("empty summary should not mark event as summarized")
	}
}

func TestHandleSummarizeEvent_EvictsCache(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "Short on-demand summary."},
		})
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	callSummarizeEvent(t, srv, sess.ID, eventID)

	rawKey := inttts.CacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(rawKey); ok {
		t.Error("on-demand summarize should evict the raw-text cache entry")
	}
	updated, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if !updated.Summarized {
		t.Error("event should be marked summarized after on-demand endpoint")
	}
}

func TestHandleSummarizeEvent_AlreadySummarized_ReSummarizes(t *testing.T) {
	callCount := 0
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		content := "First summary version."
		if callCount > 1 {
			content = "Second summary version."
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": content},
		})
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	// First on-demand call.
	callSummarizeEvent(t, srv, sess.ID, eventID)

	event, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if !event.Summarized {
		t.Fatal("event should be marked summarized after first call")
	}

	// Second on-demand call on the already-summarized event.
	resp := callSummarizeEvent(t, srv, sess.ID, eventID)

	if callCount != 2 {
		t.Errorf("expected 2 calls to summarizer (fresh re-summarize each time), got %d", callCount)
	}

	if !resp.GetSummarized() {
		t.Error("second call response should be summarized=true")
	}
	if len(resp.GetSpeechParagraphs()) == 0 || resp.GetSpeechParagraphs()[0] == "First summary version." {
		t.Errorf("second call should return the fresh summary, got %v", resp.GetSpeechParagraphs())
	}
}

func TestAsyncSummarizeAndNotify_EmitsErrorOnFailure(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	// Subscribe a client to observe the conversation_event_update on failure.
	fanout := srv.fanouts.Get(sess.ID)
	conversationCh := fanout.Subscribe()
	defer fanout.Unsubscribe(conversationCh)

	event, _ := srv.conversations.GetEvent(sess.ID, eventID)
	// Drain any initial events (e.g., the append event) before the assertion.
	for len(conversationCh) > 0 {
		<-conversationCh
	}
	srv.asyncSummarizeAndNotify(event, sess.ID)

	select {
	case emitted := <-conversationCh:
		if emitted.ID != eventID {
			t.Fatalf("expected error event for %s, got %s", eventID, emitted.ID)
		}
		if !emitted.IsUpdate {
			t.Error("error notification should be flagged IsUpdate")
		}
		if emitted.SummarizeError == "" {
			t.Error("error notification should carry a non-empty SummarizeError")
		}
	default:
		t.Fatal("expected a conversation event update carrying SummarizeError, got none")
	}
}

func TestHandleSummarizeEvent_FailurePreservesCache(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	callSummarizeEvent(t, srv, sess.ID, eventID)

	rawKey := inttts.CacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(rawKey); !ok {
		t.Error("on-demand failure must not evict the cache")
	}
	updated, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if updated.Summarized {
		t.Error("event should not be marked summarized after on-demand failure")
	}
}
