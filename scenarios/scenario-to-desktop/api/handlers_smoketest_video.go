package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// smokeTestVideoHandler serves the recorded video for a smoke test.
// GET /api/v1/smoketest/{id}/video
// Supports Range headers for seeking in the browser video player.
func (s *Server) smokeTestVideoHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		http.Error(w, `{"error":"smoke test id required"}`, http.StatusBadRequest)
		return
	}

	status, ok := s.smokeTestStore.Get(id)
	if !ok {
		http.Error(w, `{"error":"smoke test not found"}`, http.StatusNotFound)
		return
	}

	if status.ScreenRecording == nil || !status.ScreenRecording.Recorded || status.ScreenRecording.VideoPath == "" {
		http.Error(w, `{"error":"no video recording available for this smoke test"}`, http.StatusNotFound)
		return
	}

	// ServeFile handles Range headers, Content-Type, and conditional requests.
	w.Header().Set("Content-Disposition", "inline")
	http.ServeFile(w, r, status.ScreenRecording.VideoPath)
}
