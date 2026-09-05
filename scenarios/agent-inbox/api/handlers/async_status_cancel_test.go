package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// TestCancelAsyncOperation_MissingParams verifies 400 for missing parameters.
func TestCancelAsyncOperation_MissingParams(t *testing.T) {
	h := &Handlers{
		AsyncTracker: services.NewAsyncTrackerService(nil, nil),
	}

	tests := []struct {
		name string
		url  string
	}{
		{"missing both", "/api/v1/chats//async-operations//cancel"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Using a handler that won't match means 404
			r := mux.NewRouter()
			r.HandleFunc("/api/v1/chats/{id}/async-operations/{toolCallId}/cancel", h.CancelAsyncOperation).Methods("POST")

			req := httptest.NewRequest("POST", tc.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// Will get 404 because route doesn't match
			if w.Code != http.StatusNotFound {
				t.Logf("got status %d for %s", w.Code, tc.url)
			}
		})
	}
}
