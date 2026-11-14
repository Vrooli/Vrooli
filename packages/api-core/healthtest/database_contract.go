// Package healthtest contains reusable assertions for scenario database health
// handlers. The caller supplies the owner-specific live-server round trip.
package healthtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type DatabaseRoundTrip func(pingErr error) (statusCode int, body []byte, pingCalls int64)

type databaseResponse struct {
	Status       string `json:"status"`
	Service      string `json:"service"`
	Version      string `json:"version"`
	Readiness    bool   `json:"readiness"`
	Dependencies map[string]struct {
		Connected bool            `json:"connected"`
		Error     json.RawMessage `json:"error"`
	} `json:"dependencies"`
}

// AssertDatabaseContract verifies healthy and unreachable database responses
// through the owner's supplied live-server path.
func AssertDatabaseContract(t *testing.T, service, version string, roundTrip DatabaseRoundTrip) {
	t.Helper()
	cases := []struct {
		name          string
		pingErr       error
		wantHTTP      int
		wantStatus    string
		wantReadiness bool
		wantConnected bool
		wantErrSubstr string
	}{
		{name: "ok", wantHTTP: http.StatusOK, wantStatus: "healthy", wantReadiness: true, wantConnected: true},
		{name: "db_unreachable", pingErr: fmt.Errorf("connection refused"), wantHTTP: http.StatusServiceUnavailable, wantStatus: "unhealthy", wantErrSubstr: "connection refused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, calls := roundTrip(tc.pingErr)
			if status != tc.wantHTTP {
				t.Fatalf("HTTP status = %d, want %d; body: %s", status, tc.wantHTTP, body)
			}
			var got databaseResponse
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("decode health response: %v; body: %s", err, body)
			}
			if got.Status != tc.wantStatus || got.Service != service || got.Version != version || got.Readiness != tc.wantReadiness {
				t.Fatalf("health identity = status:%q service:%q version:%q readiness:%t", got.Status, got.Service, got.Version, got.Readiness)
			}
			dep, ok := got.Dependencies["database"]
			if !ok || dep.Connected != tc.wantConnected {
				t.Fatalf("database dependency = %+v (present=%t), want connected=%t", dep, ok, tc.wantConnected)
			}
			if tc.wantErrSubstr != "" && !strings.Contains(string(dep.Error), tc.wantErrSubstr) {
				t.Fatalf("database error %s does not contain %q", dep.Error, tc.wantErrSubstr)
			}
			if calls != 1 {
				t.Fatalf("pinger calls = %d, want 1", calls)
			}
		})
	}
}
