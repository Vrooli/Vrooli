package programsvalidation

import (
	"connectrpc.com/connect"
	"context"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"os"
	"path/filepath"
	programsinternal "program-runtime/internal/programs"
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

func TestValidationRecognizesRuntimeTaskNamespace(t *testing.T) {
	for _, name := range knownBindingNames(nil) {
		if name == "tasks" {
			return
		}
	}
	t.Fatal("validation must recognize the same tasks namespace as execution")
}

func TestDeclaredPreflightAcceptsInjectedInputsButRejectsUnknownNames(t *testing.T) {
	analyzer := filepath.Join("..", "..", "..", "kernel", "host", "analyze.py")
	if _, err := os.Stat(analyzer); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		source string
		want   string
	}{
		{`print(inputs.get("corpus", []))`, ""},
		{`print(missing_inputs.get("corpus", []))`, "missing_inputs"},
	} {
		diagnostics := programsinternal.ResolveSource(test.source, knownBindingNames(nil), analyzer)
		if test.want == "" {
			if len(diagnostics) != 0 {
				t.Fatalf("injected inputs rejected: %v", diagnostics)
			}
		} else if len(diagnostics) != 1 || diagnostics[0].GetName() != test.want || diagnostics[0].GetSeverity() != "error" {
			t.Fatalf("unknown input diagnostic = %v", diagnostics)
		}
	}
}

func TestValidateScenarioStillRejectsMissingScenario(t *testing.T) {
	if findings := validateScenario(t.TempDir(), "missing", nil, false); len(findings) != 1 || findings[0] != "programs.scenario_missing" {
		t.Fatalf("findings = %v, want scenario-missing", findings)
	}
}

func TestValidationResponseRetainsNativeFailureEvidence(t *testing.T) {
	h := &handler{repoRoot: t.TempDir()}
	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "missing"}))
	if err != nil {
		t.Fatal(err)
	}
	var detail structpb.Struct
	if err := response.Msg.GetNativeDetail().UnmarshalTo(&detail); err != nil {
		t.Fatal(err)
	}
	findings := detail.GetFields()["findings"].GetListValue().GetValues()
	if len(findings) != 1 || findings[0].GetStringValue() != "programs.scenario_missing" {
		t.Fatalf("failure evidence lost: %v", &detail)
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
