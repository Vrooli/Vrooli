package main

import "net/http"

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
