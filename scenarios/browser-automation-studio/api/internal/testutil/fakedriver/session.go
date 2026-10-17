// Package fakedriver provides protocol fakes for API tests that talk to the
// browser automation driver over HTTP.
package fakedriver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
)

// StartSessionServer serves the driver's standard session start and close
// endpoints for tests that only need the session lifecycle contract.
func StartSessionServer(t testing.TB, sessionID string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": sessionID,
			"actual_viewport": map[string]any{
				"width": 1280, "height": 720, "source": "requested",
				"reason": "UI-requested dimensions used",
			},
		})
	})
	mux.HandleFunc("/session/"+sessionID+"/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	return testutil.StartHTTPServer(t, mux)
}
