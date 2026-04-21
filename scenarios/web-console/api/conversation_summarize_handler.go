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

	if s.ttsSummarizer == nil {
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

	// If already summarized, return existing summary
	if event.Summarized {
		writeJSON(w, http.StatusOK, summarizeEventResponse{
			Summarized:       true,
			SpeechParagraphs: event.SpeechParagraphs,
		})
		return
	}

	cfg := s.getTTSSummarizeConfig()

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

	started := time.Now()
	summary, err := s.ttsSummarizer.Summarize(ctx, normalized, cfg.Model, cfg.Level)
	elapsedMs := time.Since(started).Milliseconds()
	if err != nil {
		logSummarizeResult("on-demand", cfg, eventID, len(normalized), 0, elapsedMs, err)
		writeJSON(w, http.StatusOK, summarizeEventResponse{
			Error: "Summarization failed: " + err.Error(),
		})
		return
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		logSummarizeResult("on-demand", cfg, eventID, len(normalized), 0, elapsedMs,
			emptySummaryErr)
		writeJSON(w, http.StatusOK, summarizeEventResponse{
			Error: "Summarizer returned empty result",
		})
		return
	}

	logSummarizeResult("on-demand", cfg, eventID, len(normalized), len(summary), elapsedMs, nil)

	newParagraphs := SplitIntoSpeechParagraphs(summary)
	s.conversations.UpdateSpeechParagraphs(sessionID, eventID, newParagraphs)
	s.invalidateTTSCacheForEvent(eventID)

	writeJSON(w, http.StatusOK, summarizeEventResponse{
		Summarized:       true,
		SpeechParagraphs: newParagraphs,
	})
}

// emptySummaryErr is a sentinel used by the on-demand handler so the unified
// log line has a grep-friendly error token for the empty-output case.
var emptySummaryErr = summarizeError("empty summary returned")

type summarizeError string

func (e summarizeError) Error() string { return string(e) }
