package support

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapptest"
)

func TestScanJobStatusReportRetainsOwnerProgressAndCommands(t *testing.T) {
	resp := map[string]any{"id": "job-1", "status": "running", "scenario": "example", "processed_scenarios": float64(2), "total_scenarios": float64(5), "current_scanner": "gosec", "current_file": "main.go"}
	security := ScanJobStatusReport("Security scan", "security", "current_scanner", resp)
	if len(security.Status) != 4 || security.Status[0] != "Security scan job: job-1" {
		t.Fatalf("security status = %#v", security.Status)
	}
	joined := strings.Join(security.Triage[0].Items, "\n")
	if !strings.Contains(joined, "Current Scanner: gosec") || strings.Contains(joined, "Current File: main.go") {
		t.Fatalf("security detail fields = %q", joined)
	}
	if len(security.NextSteps) != 2 || security.NextSteps[0] != "scenario-auditor security summary job-1" || security.NextSteps[1] != "scenario-auditor security cancel job-1" {
		t.Fatalf("security next steps = %#v", security.NextSteps)
	}

	standards := ScanJobStatusReport("Standards scan", "standards", "current_file", resp)
	joined = strings.Join(standards.Triage[0].Items, "\n")
	if !strings.Contains(joined, "Current File: main.go") || strings.Contains(joined, "Current Scanner: gosec") {
		t.Fatalf("standards detail fields = %q", joined)
	}
}

func TestCancelScanUsesOwnerEndpointAndReportsNextStep(t *testing.T) {
	var gotPath, gotMethod string
	core := cliapptest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "{}" {
			t.Errorf("request body = %q, error = %v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"cancelled"}`)
	}))
	output := cliapptest.CaptureStdout(t, func() error {
		return CancelScan(core, "security", "/scenarios/scan/jobs/job-1/cancel", "Security", "job-1", false)
	})
	if gotMethod != http.MethodPost || !strings.HasSuffix(gotPath, "/scenarios/scan/jobs/job-1/cancel") {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(output, "Security scan cancellation requested") || !strings.Contains(output, "scenario-auditor security status job-1") {
		t.Fatalf("cancellation report = %q", output)
	}
}
