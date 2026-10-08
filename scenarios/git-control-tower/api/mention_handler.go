package main

import (
	"net/http"

	"git-control-tower/internal/mentions"
)

type mentionResponse struct {
	Status  string `json:"status"`
	Key     string `json:"key,omitempty"`
	Command string `json:"command,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

func (s *Server) handleAdvisoryMention(w http.ResponseWriter, r *http.Request) {
	var delivery mentions.Delivery
	if !ParseJSONBody(w, r, &delivery) {
		return
	}
	request, err := mentions.Parse(delivery)
	if err != nil {
		NewResponse(w).OK(mentionResponse{Status: "refused", Reason: err.Error()})
		return
	}
	store, err := mentions.NewStore(s.db)
	if err != nil {
		NewResponse(w).OK(mentionResponse{Status: "unavailable", Reason: err.Error()})
		return
	}
	created, err := store.Record(r.Context(), request)
	if err != nil {
		NewResponse(w).OK(mentionResponse{Status: "failed", Reason: err.Error()})
		return
	}
	status := "duplicate"
	if created {
		status = "queued"
	}
	NewResponse(w).OK(mentionResponse{Status: status, Key: request.Key, Command: string(request.Command)})
}
