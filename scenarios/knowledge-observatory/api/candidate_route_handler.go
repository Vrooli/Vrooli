package main

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleCandidateRoute(w http.ResponseWriter, r *http.Request) {
	var request RouteRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}
	result := routeDisposition(request)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
