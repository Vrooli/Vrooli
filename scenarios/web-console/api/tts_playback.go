package main

import "net/http"

func (s *Server) handlePostTTSEvent(w http.ResponseWriter, r *http.Request) {
	var event TTSPlaybackEvent
	if !decodeJSON(w, r, &event) {
		return
	}
	if event.Source == "" || event.Stage == "" {
		writeCatalogError(w, "invalid_body", "source and stage are required")
		return
	}
	s.recordTTSPlaybackEvent(event)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
