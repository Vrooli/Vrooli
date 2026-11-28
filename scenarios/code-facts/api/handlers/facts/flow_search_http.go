package facts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	internalfacts "code-facts/internal/facts"
)

const maxFlowSearchRequestBytes = 8 << 10

type featureFlowService interface {
	SearchFeatureFlow(context.Context, string, string, string, int) (*internalfacts.FlowSearchResponse, error)
}

type flowSearchRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario"`
	Path     string `json:"path"`
	Limit    int    `json:"limit"`
}

// SearchFlowHTTPHandler exposes a small JSON adapter over Code Facts' existing
// semantic Go call graph. The caller must choose one scenario and one source
// subdirectory; the service applies file/byte and result-depth caps.
func SearchFlowHTTPHandler(service featureFlowService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, maxFlowSearchRequestBytes))
		decoder.DisallowUnknownFields()
		var req flowSearchRequest
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}
		req.Query = strings.TrimSpace(req.Query)
		req.Scenario = strings.TrimSpace(req.Scenario)
		req.Path = filepath.Clean(strings.TrimSpace(req.Path))
		if req.Query == "" || req.Scenario == "" || req.Path == "." || filepath.IsAbs(req.Path) || req.Path == ".." || strings.HasPrefix(req.Path, ".."+string(filepath.Separator)) || req.Limit < 0 || req.Limit > 5 {
			http.Error(w, "query, scenario, relative source path, or limit is invalid", http.StatusBadRequest)
			return
		}
		if service == nil {
			http.Error(w, "source graph service unavailable", http.StatusServiceUnavailable)
			return
		}
		result, err := service.SearchFeatureFlow(r.Context(), req.Query, req.Scenario, req.Path, req.Limit)
		if err != nil {
			http.Error(w, "source graph unavailable or outside the bounded scope", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}
