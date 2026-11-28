package graph

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"architecture-cartographer/internal/domains"
)

const maxDocCodeDriftBodyBytes = 8 << 10

type docCodeDriftRequest struct {
	Query string `json:"query"`
	Scope string `json:"scope"`
	Limit int    `json:"limit"`
}

// SearchDocCodeDriftHTTP exposes one-scenario, static docs/API-source path comparisons.
func (h *Handler) SearchDocCodeDriftHTTP() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, maxDocCodeDriftBodyBytes))
		decoder.DisallowUnknownFields()
		var req docCodeDriftRequest
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}
		scenario, err := scenarioFromScope(req.Scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		query := strings.TrimSpace(req.Query)
		if query == "" {
			http.Error(w, "query is required", http.StatusBadRequest)
			return
		}
		limit := req.Limit
		if limit == 0 {
			limit = 10
		}
		if limit < 1 || limit > 10 {
			http.Error(w, "limit must be between 1 and 10", http.StatusBadRequest)
			return
		}
		if h.domainsSvc == nil || h.scenarioLocator == nil {
			http.Error(w, "scenario documentation/code source unavailable", http.StatusServiceUnavailable)
			return
		}
		domainMap, err := h.domainsSvc.GetDomainMap(r.Context(), scenario)
		if err != nil {
			http.Error(w, "domain map unavailable", http.StatusServiceUnavailable)
			return
		}
		root, err := h.scenarioLocator.Locate(scenario)
		if err != nil {
			http.Error(w, "scenario source unavailable", http.StatusServiceUnavailable)
			return
		}
		response, err := domains.SearchDocCodeDrift(root, domainMap, query, limit)
		if err != nil {
			http.Error(w, "bounded docs/code comparison unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
