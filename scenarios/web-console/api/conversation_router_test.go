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
	srv.ttsSummarization = NewTTSSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)
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
	req1 := httptest.NewRequest("POST",
		"/api/v1/sessions/"+sess.ID+"/conversation/"+eventID+"/summarize",
		strings.NewReader(""))
	req1 = mux.SetURLVars(req1, map[string]string{"id": sess.ID, "eventId": eventID})
	rec1 := httptest.NewRecorder()
	srv.handleSummarizeEvent(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d", rec1.Code)
	}

	event, _ := srv.conversations.GetEvent(sess.ID, eventID)
	if !event.Summarized {
		t.Fatal("event should be marked summarized after first call")
	}

	// Second on-demand call on the already-summarized event.
	req2 := httptest.NewRequest("POST",
		"/api/v1/sessions/"+sess.ID+"/conversation/"+eventID+"/summarize",
		strings.NewReader(""))
	req2 = mux.SetURLVars(req2, map[string]string{"id": sess.ID, "eventId": eventID})
	rec2 := httptest.NewRecorder()
	srv.handleSummarizeEvent(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second call: expected 200, got %d", rec2.Code)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls to summarizer (fresh re-summarize each time), got %d", callCount)
	}

	var resp summarizeEventResponse
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp)
	if !resp.Summarized {
		t.Error("second call response should be summarized=true")
	}
	if len(resp.SpeechParagraphs) == 0 || resp.SpeechParagraphs[0] == "First summary version." {
		t.Errorf("second call should return the fresh summary, got %v", resp.SpeechParagraphs)
	}
}

func TestAsyncSummarizeAndNotify_EmitsErrorOnFailure(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollama.Close()

	srv, sess, eventID, _ := newSummarizeTestServer(t, ollama)

	// Subscribe a client to observe the conversation_event_update on failure.
	conversationCh := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(conversationCh)

	event, _ := srv.conversations.GetEvent(sess.ID, eventID)
	// Drain any initial events (e.g., the append event) before the assertion.
	for len(conversationCh) > 0 {
		<-conversationCh
	}
	srv.asyncSummarizeAndNotify(event, sess.ID, sess)

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
