package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// getVrooliRoot returns the Vrooli installation root directory.
// It checks VROOLI_ROOT environment variable first, then falls back to ~/Vrooli.
func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "Vrooli")
	}
	return filepath.Join(home, "Vrooli")
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil && logger != nil {
		logger.Error("failed to write JSON response: %v", err)
	}
}

func stringPtr(s string) *string {
	return &s
}
