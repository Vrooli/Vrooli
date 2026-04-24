package main

import (
	"net/http"
	"strconv"
)

type conversationCursorRequest struct {
	LastSeenSequence     *int64 `json:"lastSeenSequence,omitempty"`
	LastListenedSequence *int64 `json:"lastListenedSequence,omitempty"`
}

type conversationSessionResponse struct {
	SessionID string              `json:"sessionId"`
	Events    []ConversationEvent `json:"events"`
	Cursor    ConversationCursor  `json:"cursor"`
}

func (s *Server) handleGetConversationSession(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	state := s.conversations.ListSession(sess.ID)

	// Optional ?since_sequence=N: return only events with sequence > N so
	// reconnect / view-refresh can fetch the gap instead of the full history.
	if raw := r.URL.Query().Get("since_sequence"); raw != "" {
		if since, err := strconv.ParseInt(raw, 10, 64); err == nil && since > 0 {
			filtered := make([]ConversationEvent, 0, len(state.Events))
			for _, ev := range state.Events {
				if ev.Sequence > since {
					filtered = append(filtered, ev)
				}
			}
			state.Events = filtered
		}
	}

	writeJSON(w, http.StatusOK, conversationSessionResponse(state))
}

func (s *Server) handleUpdateConversationCursor(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	var req conversationCursorRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	cursor := s.conversations.UpdateCursor(sess.ID, conversationCursorPatch{
		seenSequence:     req.LastSeenSequence,
		listenedSequence: req.LastListenedSequence,
	})
	writeJSON(w, http.StatusOK, cursor)
}
