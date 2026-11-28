package facts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	internalfacts "code-facts/internal/facts"
)

type persistencePatternService interface {
	SearchPersistencePatterns(context.Context, string, string, int) (*internalfacts.PersistencePatternResponse, error)
}
type persistencePatternRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario"`
	Limit    int    `json:"limit"`
}

// SearchPersistencePatternsHTTPHandler returns bounded, static persistence-source joins.
func SearchPersistencePatternsHTTPHandler(service persistencePatternService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, 8<<10))
		decoder.DisallowUnknownFields()
		var request persistencePatternRequest
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}
		request.Query = strings.TrimSpace(request.Query)
		request.Scenario = strings.TrimSpace(request.Scenario)
		if request.Query == "" || !invariantScenarioPattern.MatchString(request.Scenario) || request.Limit < 0 || request.Limit > 5 {
			http.Error(w, "query and scenario are required; limit must be from 0 to 5", http.StatusBadRequest)
			return
		}
		if service == nil {
			http.Error(w, "persistence source service unavailable", http.StatusServiceUnavailable)
			return
		}
		response, err := service.SearchPersistencePatterns(r.Context(), request.Query, request.Scenario, request.Limit)
		if err != nil {
			http.Error(w, "bounded persistence evidence unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
