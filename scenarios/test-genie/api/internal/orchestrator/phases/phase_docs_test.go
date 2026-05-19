package phases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"test-genie/internal/orchestrator/workspace"

	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
)

type fakeConnectClient struct {
	resp *kov1.DocHealthResponse
	err  error
}

func (f fakeConnectClient) Do(_ *http.Request) (*http.Response, error) {
	// Not used: tests stub the URL resolver to point at a fake HTTP
	// server via httptest where possible. When tests need to inject a
	// canned response without an HTTP round-trip, they replace the
	// resolveDocHealthBaseURL+docHealthHTTPClient pair entirely below.
	return nil, errors.New("not implemented")
}

func swapDocsSeams(t *testing.T, baseURL string, client connect.HTTPClient) func() {
	t.Helper()
	prevResolve := resolveDocHealthBaseURL
	prevClient := docHealthHTTPClient
	resolveDocHealthBaseURL = func(_ context.Context) (string, error) { return baseURL, nil }
	docHealthHTTPClient = client
	return func() {
		resolveDocHealthBaseURL = prevResolve
		docHealthHTTPClient = prevClient
	}
}

// stubHTTPClient lets a test return a canned proto response without
// running an HTTP server. It hand-builds a Connect-framed body so the
// generated client decodes successfully.
type stubHTTPClient struct {
	respond func(*http.Request) (*http.Response, error)
}

func (s stubHTTPClient) Do(r *http.Request) (*http.Response, error) { return s.respond(r) }

func TestTranslateDocHealth_AllClean(t *testing.T) {
	resp := &kov1.DocHealthResponse{
		ScenarioName: "demo",
		Counts:       &kov1.DocHealthCounts{FilesChecked: 3, LocalLinks: 5},
	}
	out := translateDocHealth(resp)
	if !out.Success {
		t.Errorf("expected Success=true")
	}
	if out.Summary.FilesChecked != 3 || out.Summary.LocalLinks != 5 {
		t.Errorf("counts not translated: %+v", out.Summary)
	}
}

func TestTranslateDocHealth_FailureSeverityFailsPhase(t *testing.T) {
	path := "docs/foo.md"
	resp := &kov1.DocHealthResponse{
		ScenarioName: "demo",
		ContentFindings: []*kov1.DocHealthFinding{
			{Code: "broken_local_link", Severity: kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE, Message: "broken link", Path: &path},
		},
	}
	out := translateDocHealth(resp)
	if out.Success {
		t.Errorf("expected Success=false when a FAILURE finding is present")
	}
	if out.FailureClass == "" {
		t.Errorf("expected failure class to be set")
	}
	if out.Error == nil {
		t.Errorf("expected Error to be set")
	}
	if len(out.Observations) == 0 {
		t.Errorf("expected at least one observation")
	}
}

func TestTranslateDocHealth_WarningOnly_Succeeds(t *testing.T) {
	resp := &kov1.DocHealthResponse{
		ScenarioName: "demo",
		ContentFindings: []*kov1.DocHealthFinding{
			{Code: "absolute_path", Severity: kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_WARNING, Message: "absolute path detected"},
		},
	}
	out := translateDocHealth(resp)
	if !out.Success {
		t.Errorf("expected Success=true when only warnings are present")
	}
}

func TestRunDocsPhase_RPCErrorIsFailureClassMissingDependency(t *testing.T) {
	restore := swapDocsSeams(t, "http://127.0.0.1:1", stubHTTPClient{respond: func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}})
	defer restore()

	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var buf bytes.Buffer
	report := runDocsPhase(context.Background(), env, io.MultiWriter(&buf, io.Discard))
	if report.Err == nil {
		t.Fatalf("expected error when DocHealth RPC fails, got nil")
	}
	if string(report.FailureClassification) != "missing_dependency" {
		t.Errorf("FailureClassification = %q, want missing_dependency", report.FailureClassification)
	}
}
