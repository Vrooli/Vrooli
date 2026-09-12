package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/instance"
)

// TestInstanceReadinessReportsUnavailableWithSetupNextAction [REQ:STC-P0-041]
// GET /instances/readiness answers 200 with a typed unavailable report naming
// vrooli setup when the declared host tools are absent (EXT-06).
func TestInstanceReadinessReportsUnavailableWithSetupNextAction(t *testing.T) {
	srv := newTestServer()
	srv.instanceProvider = instance.LocalQEMUProvider{
		LookPath:      func(string) (string, error) { return "", errors.New("not installed") },
		ImageManifest: filepath.Join(t.TempDir(), "absent.json"),
	}
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/instances/readiness", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var report instance.LaneReadiness
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if report.Ready || report.State != instance.ReadinessUnavailable {
		t.Fatalf("report = %+v, want unavailable", report)
	}
	if !strings.Contains(report.NextAction.Reference, "vrooli setup") {
		t.Fatalf("next action = %+v, want vrooli setup", report.NextAction)
	}
	if report.Architectures["arm64"] != instance.ArchStateUnavailable {
		t.Fatalf("arm64 = %s, want unavailable", report.Architectures["arm64"])
	}
}
