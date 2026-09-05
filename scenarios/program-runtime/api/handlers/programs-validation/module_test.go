package programsvalidation

import (
	"connectrpc.com/connect"
	"context"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateScenarioAllowsNoDeclaredPrograms(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if findings := validateScenario(root, "demo", nil, true); len(findings) != 0 {
		t.Fatalf("findings = %v, want no finding for an earned no-program state", findings)
	}
}

func TestValidateScenarioStillRejectsMissingScenario(t *testing.T) {
	if findings := validateScenario(t.TempDir(), "missing", nil, false); len(findings) != 1 || findings[0] != "programs.scenario_missing" {
		t.Fatalf("findings = %v, want scenario-missing", findings)
	}
}

func TestFixtureExpectationChecksNestedSignals(t *testing.T) {
	actual := map[string]any{"status": "ok", "signals": map[string]any{"disposition": "terminal"}}
	if matchesExpectation(actual, map[string]any{"status": []any{"ok"}, "signals": map[string]any{"disposition": "quiet"}}) {
		t.Fatal("nested mismatch passed")
	}
	if !matchesExpectation(actual, map[string]any{"status": []any{"ok"}, "signals": map[string]any{"disposition": "terminal"}}) {
		t.Fatal("matching nested output rejected")
	}
}

type fixtureTestRunner struct{ calls int }

func (r *fixtureTestRunner) RunDeclaredProgram(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	r.calls++
	if req.Msg.GetProvenance() != programsv1.Provenance_PROVENANCE_TEST || req.Msg.GetExpectedDigest() == "" {
		panic("fixture execution must be pinned and test-provenance")
	}
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: `{"status":"invented"}`}}), nil
}
func TestExecutionValidationActuallyRunsShippedFixtures(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	runner := &fixtureTestRunner{}
	h := &handler{repoRoot: root, runner: runner}
	findings := h.executeFixtures(context.Background(), "agent-manager")
	if runner.calls == 0 {
		t.Fatalf("fixtures were not executed: %v", findings)
	}
	mismatch := false
	for _, finding := range findings {
		mismatch = mismatch || strings.HasSuffix(finding, "expectation_mismatch")
	}
	if !mismatch {
		t.Fatalf("invented result passed: %v", findings)
	}
}
