package phases

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func withTidinessSeams(t *testing.T, resolve func(ctx context.Context) (string, error)) {
	t.Helper()
	origResolve := resolveTidinessBaseURL
	resolveTidinessBaseURL = resolve
	t.Cleanup(func() { resolveTidinessBaseURL = origResolve })
}

func TestTidinessPhaseSkipsWhenManagerDown(t *testing.T) {
	withTidinessSeams(t, func(ctx context.Context) (string, error) {
		return "", errors.New("tidiness-manager not running")
	})
	rep := runTidinessPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if rep.Err != nil {
		t.Fatalf("manager-down must not fail the phase, got err: %v", rep.Err)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("manager-down must emit no findings, got %d", len(rep.Findings))
	}
}

func TestTidinessPhaseMapsViolations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scan/tidiness" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, `{"findings":[
			{"rule_id":"LONG_FILE","category":"length","severity":"medium","title":"large file","file_path":"api/server.go","line_number":1,"why_it_matters":"large files slow review","recommended_remediation":"split cohesive responsibilities"},
			{"rule_id":"TECH_DEBT_MARKERS","category":"technical_debt","severity":"low","title":"many TODOs","file_path":"ui/App.tsx","recommended_remediation":"resolve stale markers"}
		]}`)
	}))
	t.Cleanup(srv.Close)
	withTidinessSeams(t, func(ctx context.Context) (string, error) { return srv.URL, nil })

	rep := runTidinessPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if rep.Err != nil {
		t.Fatalf("unexpected err: %v", rep.Err)
	}
	if len(rep.Findings) != 2 {
		t.Fatalf("want 2 findings, got %d", len(rep.Findings))
	}
	for _, f := range rep.Findings {
		if f.Source != architecturev1.FindingSource_FINDING_SOURCE_TIDINESS {
			t.Errorf("finding %q source = %v, want TIDINESS", f.Code, f.Source)
		}
	}
	if rep.Findings[0].Code != "LONG_FILE" {
		t.Fatalf("expected tidiness rule code, got %q", rep.Findings[0].Code)
	}
}

func TestTidinessPhaseCleanScenario(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"violations":[]}`)
	}))
	t.Cleanup(srv.Close)
	withTidinessSeams(t, func(ctx context.Context) (string, error) { return srv.URL, nil })

	rep := runTidinessPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if rep.Err != nil || len(rep.Findings) != 0 {
		t.Fatalf("clean scenario: err=%v findings=%d", rep.Err, len(rep.Findings))
	}
}
