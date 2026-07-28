package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"web-console/internal/audioports"

	conversationH "web-console/handlers/conversation"
)

// conversationAdapter implements conversationH.Service against the server's
// ConversationStore and TTS summarizer.
type conversationAdapter struct {
	srv *Server
}

func newConversationAdapter(s *Server) *conversationAdapter {
	return &conversationAdapter{srv: s}
}

func (a *conversationAdapter) Get(sessionID string, sinceSequence int64, limit int, beforeSequence int64) (conversationH.SessionState, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return conversationH.SessionState{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	state := a.srv.conversations.ListSession(sessionID)
	hasMore := false
	if limit > 0 {
		state, hasMore = a.srv.conversations.ListSessionPage(sessionID, limit, beforeSequence)
	}
	events := state.Events
	if sinceSequence > 0 {
		filtered := make([]ConversationEvent, 0, len(events))
		for _, ev := range events {
			if ev.Sequence > sinceSequence {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}
	result := conversationH.SessionState{
		SessionID: state.SessionID,
		Events:    transportEvents(events),
		Cursor: conversationH.Cursor{
			LastSeenSequence:     state.Cursor.LastSeenSequence,
			LastListenedSequence: state.Cursor.LastListenedSequence,
		},
		HasMore:    hasMore,
		TotalCount: a.srv.conversations.CountSessionEvents(sessionID),
	}
	if len(events) > 0 {
		result.OldestSequence = events[0].Sequence
		result.NewestSequence = events[len(events)-1].Sequence
	}
	return result, nil
}

func (a *conversationAdapter) Search(sessionID, query string, limit int) ([]conversationH.SearchMatch, bool, int64, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return nil, false, 0, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	matches, truncated, total, err := a.srv.conversations.SearchSession(sessionID, query, limit)
	if err != nil {
		return nil, false, 0, err
	}
	out := make([]conversationH.SearchMatch, 0, len(matches))
	for _, match := range matches {
		out = append(out, conversationH.SearchMatch{EventID: match.EventID, Sequence: match.Sequence, Excerpt: match.Excerpt})
	}
	return out, truncated, total, nil
}

func (a *conversationAdapter) GetRange(sessionID string, from, to int64) (conversationH.SessionState, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return conversationH.SessionState{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	events, err := a.srv.conversations.ListSessionRange(sessionID, from, to)
	if err != nil {
		return conversationH.SessionState{}, err
	}
	state := a.srv.conversations.ListSession(sessionID)
	return conversationH.SessionState{SessionID: sessionID, Events: transportEvents(events), Cursor: conversationH.Cursor{LastSeenSequence: state.Cursor.LastSeenSequence, LastListenedSequence: state.Cursor.LastListenedSequence}}, nil
}

func (a *conversationAdapter) UpdateCursor(sessionID string, patch conversationH.CursorPatch) (conversationH.Cursor, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return conversationH.Cursor{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	storePatch := conversationCursorPatch{}
	if patch.HasLastSeenSequence {
		v := patch.LastSeenSequence
		storePatch.seenSequence = &v
	}
	if patch.HasLastListenedSequence {
		v := patch.LastListenedSequence
		storePatch.listenedSequence = &v
	}
	cur := a.srv.conversations.UpdateCursor(sessionID, storePatch)
	return conversationH.Cursor{
		LastSeenSequence:     cur.LastSeenSequence,
		LastListenedSequence: cur.LastListenedSequence,
	}, nil
}

func (a *conversationAdapter) SummarizeEvent(ctx context.Context, sessionID, eventID string) (conversationH.SummarizeResult, error) {
	if _, ok := a.srv.sessions.Get(sessionID); !ok {
		return conversationH.SummarizeResult{}, fmt.Errorf("session %q: %w", sanitizeID(sessionID), conversationH.ErrSessionNotFound)
	}
	if a.srv.summarizer == nil {
		return conversationH.SummarizeResult{
			Error: "Summarizer not available — audio-tools is not reachable",
		}, nil
	}

	state := a.srv.conversations.ListSession(sessionID)
	var event *ConversationEvent
	for i := range state.Events {
		if state.Events[i].ID == eventID {
			event = &state.Events[i]
			break
		}
	}
	if event == nil {
		return conversationH.SummarizeResult{}, fmt.Errorf("event %q: %w", sanitizeID(eventID), conversationH.ErrNotFound)
	}
	if event.Role != ConversationRoleAssistant {
		return conversationH.SummarizeResult{}, fmt.Errorf("only assistant events can be summarized: %w", conversationH.ErrInvalidArgument)
	}

	policy := a.srv.getSummarizeAutoPolicy()
	if strings.TrimSpace(event.Text) == "" {
		return conversationH.SummarizeResult{
			Error: "Event text is empty",
		}, nil
	}

	timeout := time.Duration(policy.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, err := a.srv.summarizer.Summarize(cctx, audioports.SummarizeInput{
		Text:           event.Text,
		Level:          policy.Level,
		TimeoutSeconds: policy.TimeoutSeconds,
	})
	if err != nil {
		logSummarizeResult("on-demand", policy, eventID, len(event.Text), out, err)
		return conversationH.SummarizeResult{
			Error: summarizeErrorMessage(err),
		}, nil
	}
	logSummarizeResult("on-demand", policy, eventID, len(event.Text), out, nil)

	newParagraphs := splitIntoSpeechParagraphs(out.Text)
	if len(newParagraphs) == 0 {
		return conversationH.SummarizeResult{
			Error: "Summarization returned empty content",
		}, nil
	}
	a.srv.conversations.UpdateSpeechParagraphs(sessionID, eventID, newParagraphs)

	return conversationH.SummarizeResult{
		Summarized:       true,
		SpeechParagraphs: newParagraphs,
	}, nil
}

// transportEvents converts the internal ConversationEvent slice into the
// transport-neutral shape used by the handler package.
func transportEvents(in []ConversationEvent) []conversationH.Event {
	out := make([]conversationH.Event, 0, len(in))
	for _, e := range in {
		out = append(out, conversationH.Event{
			ID:                       e.ID,
			SessionID:                e.SessionID,
			Source:                   e.Source,
			Role:                     string(e.Role),
			Text:                     e.Text,
			SpeechParagraphs:         append([]string(nil), e.SpeechParagraphs...),
			OriginalSpeechParagraphs: append([]string(nil), e.OriginalSpeechParagraphs...),
			Summarized:               e.Summarized,
			CreatedAt:                e.CreatedAt.Format(time.RFC3339Nano),
			Sequence:                 e.Sequence,
			DeliveryState:            string(e.DeliveryState),
			TTSState:                 string(e.TTSState),
			ConsumptionState:         string(e.ConsumptionState),
		})
	}
	return out
}
