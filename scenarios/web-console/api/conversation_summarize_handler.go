package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type summarizeEventResponse struct {
	Summarized       bool     `json:"summarized"`
	SpeechParagraphs []string `json:"speechParagraphs"`
	Error            string   `json:"error,omitempty"`
}

// handleSummarizeEvent triggers on-demand TTS summarization for a specific
// conversation event. Returns the summarized speech paragraphs.
// POST /api/v1/sessions/{id}/conversation/{eventId}/summarize
func (s *Server) handleSummarizeEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	eventID := vars["eventId"]

	if sessionID == "" || eventID == "" {
		writeCatalogError(w, "invalid_body", "session ID and event ID are required")
		return
	}

	if s.ttsSummarization == nil {
		writeJSON(w, http.StatusOK, summarizeEventResponse{
			Error: "Summarizer not available — Ollama may not be running",
		})
		return
	}

	// Find the event in the conversation store
	state := s.conversations.ListSession(sessionID)
	var event *ConversationEvent
	for i := range state.Events {
		if state.Events[i].ID == eventID {
			event = &state.Events[i]
			break
		}
	}
	if event == nil {
		writeCatalogError(w, "not_found", "Event not found in session")
		return
	}

	if event.Role != ConversationRoleAssistant {
		writeCatalogError(w, "invalid_body", "Only assistant messages can be summarized")
		return
	}

	cfg := s.getTTSSummarizeConfig()

	// On-demand calls always re-summarize. Unlike the auto path, the user has
	// explicitly asked for a fresh summary (e.g. after changing level), so the
	// cached-summary short-circuit is counterproductive.
	normalized := NormalizeTextForSpeech(event.Text)
	if strings.TrimSpace(normalized) == "" {
		writeJSON(w, http.StatusOK, summarizeEventResponse{
			Error: "Event text is empty after normalization",
		})
		return
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	result, err := s.ttsSummarization.Summarize(ctx, TTSSummarizeRequest{
		EventID: eventID,
		Path:    "on-demand",
		Text:    normalized,
	})
	if err != nil {
		logSummarizeResult("on-demand", cfg, eventID, len(normalized), result, err)
		writeJSON(w, http.StatusOK, summarizeEventResponse{
			Error: summarizeErrorMessage(err),
		})
		return
	}
	logSummarizeResult("on-demand", cfg, eventID, len(normalized), result, nil)

	newParagraphs := result.Paragraphs
	s.conversations.UpdateSpeechParagraphs(sessionID, eventID, newParagraphs)
	s.invalidateTTSCacheForEvent(eventID)

	writeJSON(w, http.StatusOK, summarizeEventResponse{
		Summarized:       true,
		SpeechParagraphs: newParagraphs,
	})
}

// errSummarizeBudgetInThink / errSummarizeTruncated / errSummarizeEmptyAfterStrip
// / errSummarizeTrulyEmpty are the four categorized empty-result sentinels used
// by the service and error-message helpers. Each names a distinct failure mode
// so logs and user-facing banners can say something actionable.
var (
	errSummarizeBudgetInThink   = summarizeError("budget exhausted inside <think>")
	errSummarizeTruncated       = summarizeError("response truncated (done_reason=length)")
	errSummarizeEmptyAfterStrip = summarizeError("empty after stripping <think> block")
	errSummarizeTrulyEmpty      = summarizeError("truly empty response from model")
)

type summarizeError string

func (e summarizeError) Error() string { return string(e) }
