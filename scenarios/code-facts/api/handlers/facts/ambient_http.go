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

const maxAmbientSearchRequestBytes = 8 << 10

type ambientSeamService interface {
	SearchAmbientSeams(context.Context, string, string, int) (*internalfacts.AmbientSeamResponse, error)
}

type ambientSeamRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario"`
	Limit    int    `json:"limit"`
}

// SearchAmbientSeamsHTTPHandler exposes bounded source-call-site evidence. It
// reports static file suffix and declared build-tag facts without claiming
// target build inclusion, binary membership, runtime wiring, or test substitution.
func SearchAmbientSeamsHTTPHandler(service ambientSeamService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, maxAmbientSearchRequestBytes))
		decoder.DisallowUnknownFields()
		var request ambientSeamRequest
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
		if request.Query == "" || !invariantScenarioPattern.MatchString(request.Scenario) || request.Limit < 0 || request.Limit > 12 {
			http.Error(w, "query and scenario are required; limit must be from 0 to 12", http.StatusBadRequest)
			return
		}
		if service == nil {
			http.Error(w, "ambient seam source service unavailable", http.StatusServiceUnavailable)
			return
		}
		response, err := service.SearchAmbientSeams(r.Context(), request.Query, request.Scenario, request.Limit)
		if err != nil {
			http.Error(w, "bounded scenario source evidence unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
