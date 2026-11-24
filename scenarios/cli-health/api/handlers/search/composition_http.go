package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"cli-health/internal/clianatomy"
	repocontract "github.com/vrooli/repo-contract-go"
)

const maxCompositionRequestBytes = 4096

type compositionRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario"`
}

// CompositionSearchHTTPHandler exposes static ScenarioApp/command-group source
// evidence. It never invokes the target CLI or claims runtime dispatch.
func CompositionSearchHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(r.Body, maxCompositionRequestBytes+1))
		if readErr != nil || len(body) > maxCompositionRequestBytes {
			http.Error(w, "request body exceeds the size limit", http.StatusRequestEntityTooLarge)
			return
		}
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.DisallowUnknownFields()
		var req compositionRequest
		if dec.Decode(&req) != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}
		req.Query = strings.TrimSpace(req.Query)
		req.Scenario = strings.TrimSpace(req.Scenario)
		if req.Query == "" || req.Scenario == "" {
			http.Error(w, "query and scenario are required", http.StatusBadRequest)
			return
		}
		root, err := repocontract.ResolveRepoRoot()
		if err != nil {
			http.Error(w, "repository source root unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clianatomy.Search(root, req.Scenario, req.Query))
	}
}
