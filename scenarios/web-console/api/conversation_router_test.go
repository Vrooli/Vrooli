package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestAppendConversationEvent_MissingSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.appendConversationEvent("hello", "", "test")
	if result.Appended {
		t.Error("expected failure for empty session")
	}
	if result.Code != "conversation_target_missing" {
		t.Errorf("expected code conversation_target_missing, got %s", result.Code)
	}
}

func TestAppendConversationEvent_UnknownSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.appendConversationEvent("hello", "nonexistent", "test")
	if result.Appended {
		t.Error("expected failure for unknown session")
	}
}

func TestAppendUserConversationEvent_MissingSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.appendUserConversationEvent("hello", "", "test")
	if result.Appended {
		t.Error("expected failure for empty session")
	}
}

func TestAppendUserConversationEvent_UnknownSession(t *testing.T) {
	srv := newFakeTestServer()
	result := srv.appendUserConversationEvent("hello", "nonexistent", "test")
	if result.Appended {
		t.Error("expected failure for unknown session")
	}
}

// newSummarizeTestServer wires up the server fields exercised by
// asyncSummarizeAndNotify and the on-demand summarize handler.
func newSummarizeTestServer(t *testing.T, ollama *httptest.Server) (*Server, *Session, string, string) {
	t.Helper()
	srv := newFakeTestServer()
	srv.conversations = NewConversationStore()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	srv.ttsSummarizer = NewTTSSummarizer(ollama.URL)
	srv.ttsSummarizeConfig = TTSSummarizeConfig{
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
	rawKey := TTSCacheKey{EventID: event.ID, Voice: "af_heart", Speed: 1.0, Version: "active"}
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

	srv.asyncSummarizeAndNotify(event, sess.ID, sess)

	// Cache for the raw audio must be gone.
	rawKey := TTSCacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
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

	srv.asyncSummarizeAndNotify(event, sess.ID, sess)

	// Cache must still hold the seeded raw audio — summary never succeeded.
	rawKey := TTSCacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
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
	srv.asyncSummarizeAndNotify(event, sess.ID, sess)

	rawKey := TTSCacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
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

	req := httptest.NewRequest("POST",
		"/api/v1/sessions/"+sess.ID+"/conversation/"+eventID+"/summarize",
		strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID, "eventId": eventID})
	rec := httptest.NewRecorder()

	srv.handleSummarizeEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rawKey := TTSCacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(rawKey); ok {
		t.Error("on-demand summarize should evict the raw-text cache entry")
	}
	updated, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if !updated.Summarized {
		t.Error("event should be marked summarized after on-demand endpoint")
	}
}

func TestHandleSummarizeEvent_FailurePreservesCache(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	req := httptest.NewRequest("POST",
		"/api/v1/sessions/"+sess.ID+"/conversation/"+eventID+"/summarize",
		strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID, "eventId": eventID})
	rec := httptest.NewRecorder()

	srv.handleSummarizeEvent(rec, req)

	rawKey := TTSCacheKey{EventID: eventID, Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(rawKey); !ok {
		t.Error("on-demand failure must not evict the cache")
	}
	updated, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if updated.Summarized {
		t.Error("event should not be marked summarized after on-demand failure")
	}
}
