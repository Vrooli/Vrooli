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

type protoHandlerConformanceService interface {
	ProtoHandlerConformance(context.Context, string) (*internalfacts.ProtoHandlerConformanceResponse, error)
}

type protoHandlerConformanceRequest struct {
	Scenario string `json:"scenario"`
}

// ProtoHandlerConformanceHTTPHandler returns bounded signature comparisons
// between concrete Go handlers and imported generated Connect interfaces.
func ProtoHandlerConformanceHTTPHandler(service protoHandlerConformanceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
		decoder.DisallowUnknownFields()
		var request protoHandlerConformanceRequest
		if err := decoder.Decode(&request); err != nil {
			var maxBytes *http.MaxBytesError
			if errors.As(err, &maxBytes) {
				http.Error(w, "request body exceeds limit", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "invalid request body", http.StatusBadRequest)
			}
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "request body must contain one JSON object", http.StatusBadRequest)
			return
		}
		request.Scenario = strings.TrimSpace(request.Scenario)
		if !invariantScenarioPattern.MatchString(request.Scenario) {
			http.Error(w, "one scenario slug is required", http.StatusBadRequest)
			return
		}
		if service == nil {
			http.Error(w, "proto handler conformance service unavailable", http.StatusServiceUnavailable)
			return
		}
		response, err := service.ProtoHandlerConformance(r.Context(), request.Scenario)
		if err != nil {
			http.Error(w, "bounded proto handler conformance source unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
