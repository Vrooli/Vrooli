package main

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleCandidateProposals(w http.ResponseWriter, r *http.Request) {
	var request ProposalRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}
	result := proposeInventoryDispositions(request)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
