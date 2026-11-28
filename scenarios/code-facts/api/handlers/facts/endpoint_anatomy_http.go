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

type endpointAnatomyService interface {
	SearchEndpointAnatomy(context.Context, string, string, int) (*internalfacts.EndpointAnatomyResponse, error)
}
type endpointAnatomyRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario"`
	Limit    int    `json:"limit"`
}

// SearchEndpointAnatomyHTTPHandler returns bounded declaration/route/handler source joins.
func SearchEndpointAnatomyHTTPHandler(service endpointAnatomyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, 8<<10))
		decoder.DisallowUnknownFields()
		var request endpointAnatomyRequest
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
		if request.Query == "" || !invariantScenarioPattern.MatchString(request.Scenario) || request.Limit < 0 || request.Limit > 10 {
			http.Error(w, "query and scenario are required; limit must be from 0 to 10", http.StatusBadRequest)
			return
		}
		if service == nil {
			http.Error(w, "endpoint anatomy source service unavailable", http.StatusServiceUnavailable)
			return
		}
		response, err := service.SearchEndpointAnatomy(r.Context(), request.Query, request.Scenario, request.Limit)
		if err != nil {
			http.Error(w, "bounded endpoint source evidence unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
