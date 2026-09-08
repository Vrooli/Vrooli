package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
)

func (s *Server) handleCandidateInventory(w http.ResponseWriter, r *http.Request) {
	var request InventoryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}
	scenariosRoot := s.config.ScenariosRoot
	if scenariosRoot == "" {
		scenariosRoot = resolveScenariosRoot()
	}
	if scenariosRoot == "" {
		respondWithError(w, http.StatusServiceUnavailable, fmt.Errorf("scenarios root is unavailable"))
		return
	}
	result, err := buildCandidateInventory(filepath.Dir(scenariosRoot), request)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
