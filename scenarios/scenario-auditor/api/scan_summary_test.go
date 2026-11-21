package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func TestServeScanSummary(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		serveScanSummary(recorder, scanRequest("job-1", ""), "Security scan", errSecurityScanNotFound, func(string) (*ViolationSummary, bool) {
			return nil, false
		})
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})

	t.Run("running", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		serveScanSummary(recorder, scanRequest("job-1", ""), "Security scan", errSecurityScanNotFound, func(string) (*ViolationSummary, bool) {
			return nil, true
		})
		if recorder.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
		}
	})

	t.Run("filtered rule groups", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		summary := &ViolationSummary{
			Total:      2,
			BySeverity: map[string]int{"high": 1, "low": 1},
			TopViolations: []ViolationExcerpt{
				{ID: "high", Severity: "high"},
				{ID: "low", Severity: "low"},
			},
			ByRule: []RuleCount{
				{RuleID: "high-rule", Count: 1, Severity: "high"},
				{RuleID: "low-rule", Count: 1, Severity: "low"},
			},
		}
		serveScanSummary(recorder, scanRequest("job-1", "group_by=rule&min_severity=high"), "Standards scan", errStandardsScanNotFound, func(string) (*ViolationSummary, bool) {
			return summary, true
		})
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		var response struct {
			Summary ViolationSummary `json:"summary"`
			Groups  []RuleCount      `json:"groups"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Summary.TopViolations) != 1 || response.Summary.TopViolations[0].ID != "high" {
			t.Fatalf("filtered violations = %#v", response.Summary.TopViolations)
		}
		if len(response.Groups) != 2 || response.Groups[0].RuleID != "high-rule" {
			t.Fatalf("rule groups = %#v", response.Groups)
		}
	})
}

func TestServeScanArtifact(t *testing.T) {
	t.Run("serves retained artifact", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "summary.json")
		const want = `{"retained":true}`
		if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		serveScanArtifact(recorder, scanRequest("job-7", ""), "security", "Security scan not found", errSecurityScanNotFound, func(string) (string, bool, bool) {
			return path, true, true
		})
		if recorder.Code != http.StatusOK || recorder.Body.String() != want {
			t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Content-Disposition"); got != "attachment; filename=security-job-7.json" {
			t.Fatalf("Content-Disposition = %q", got)
		}
	})

	t.Run("rejects missing artifact", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		serveScanArtifact(recorder, scanRequest("job-7", ""), "standards", "Standards scan not found", errStandardsScanNotFound, func(string) (string, bool, bool) {
			return "", true, false
		})
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})
}

func scanRequest(jobID, query string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/scan?"+query, nil)
	return mux.SetURLVars(request, map[string]string{"jobId": jobID})
}
