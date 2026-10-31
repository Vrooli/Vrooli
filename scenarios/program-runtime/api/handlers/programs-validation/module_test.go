package programsvalidation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	"program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
	programsinternal "program-runtime/internal/programs"
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

func TestProgramDependencyFindingNamesBindingAndManifestFix(t *testing.T) {
	bindings := []struct {
		ID     string `json:"id"`
		Effect string `json:"effect"`
	}{{ID: "vrooli-memory/learning/record", Effect: "write"}}
	details := []any{}
	findings := programDependencyFindings("demo", "learned", "", bindings, nil, map[string]struct{}{}, &details)
	if !hasFinding(findings, "programs.dependency_undeclared") {
		t.Fatalf("findings = %v, want undeclared dependency", findings)
	}
	if len(details) != 1 || !strings.Contains(details[0].(map[string]any)["message"].(string), "vrooli-memory") {
		t.Fatalf("details = %v, want prose dependency fix", details)
	}

	if findings := programDependencyFindings("demo", "learned", "", bindings, nil, map[string]struct{}{"vrooli-memory": {}}, nil); len(findings) != 0 {
		t.Fatalf("declared dependency still reported: %v", findings)
	}
}

func TestProgramLibraryAndLearnUsesAreDependencies(t *testing.T) {
	findings := programDependencyFindings("demo", "learned", "lib.vrooli_memory.choose_option()\nlearn.note('preference', {})", nil, nil, map[string]struct{}{}, nil)
	if !hasFinding(findings, "programs.dependency_undeclared") {
		t.Fatalf("findings = %v, want undeclared dependency", findings)
	}
}

func TestProjectControlPlaneBindingIsNotAScenarioDependency(t *testing.T) {
	declared := []struct {
		ID     string `json:"id"`
		Effect string `json:"effect"`
	}{{ID: "vrooli/scenario/list", Effect: "read"}}
	if findings := programDependencyFindings("demo", "fleet", "", declared, nil, map[string]struct{}{}); len(findings) != 0 {
		t.Fatalf("control-plane binding must not require a fictional vrooli scenario dependency: %v", findings)
	}
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := bindings.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Binding("vrooli/scenario/list"); !ok {
		t.Fatal("control-plane binding absent from registry")
	}
	if findings := programDependencyFindings("demo", "fleet", "", declared, registry, map[string]struct{}{}); len(findings) != 0 {
		t.Fatalf("registered control-plane binding must not require a fictional scenario dependency: %v", findings)
	}
}

func TestValidationResponseRetainsNativeFailureEvidence(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	h := &handler{repoRoot: root}
	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "missing"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("status = %s, want FAILED for a severe finding", response.Msg.GetStatus())
	}
	if response.Msg.GetFailureClassification() != "test_failure" {
		t.Fatalf("failure classification = %q, want test_failure for failed validation findings", response.Msg.GetFailureClassification())
	}
	assessment := response.Msg.GetAssessment()
	if assessment.GetFindingsBySeverity()["FINDING_SEVERITY_ERROR"] != 1 || len(assessment.GetFindings()) != 1 || assessment.GetFindings()[0].GetCode() != "programs.scenario_missing" {
		t.Fatalf("handler assessment lost severe finding evidence: %+v", assessment)
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

func TestValidationResponseClassifiesAssessmentConstructionErrorsAsSystem(t *testing.T) {
	h := &handler{repoRoot: t.TempDir()}
	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "missing"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR {
		t.Fatalf("status = %s, want ERROR when the provider cannot load its assessment contract", response.Msg.GetStatus())
	}
	if response.Msg.GetFailureClassification() != "system" {
		t.Fatalf("failure classification = %q, want system for a provider assessment error", response.Msg.GetFailureClassification())
	}
}

func TestValidateScenarioProjectsFindingSeverityToAssessment(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := bindings.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	h := &handler{repoRoot: root, registry: registry, portfolio: warningOnlyPortfolio{}}
	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "tech-tree-designer"}))
	if err != nil {
		t.Fatal(err)
	}
	warningAssessment := response.Msg.GetAssessment()
	if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("warning-only handler status = %s, want PASSED", response.Msg.GetStatus())
	}
	if warningAssessment.GetFindingsBySeverity()["FINDING_SEVERITY_WARNING"] != 1 || len(warningAssessment.GetFindings()) != 1 || warningAssessment.GetFindings()[0].GetCode() != "programs.never_exercised" {
		t.Fatalf("warning assessment lost normalized severity evidence: %+v", warningAssessment)
	}
	if warningAssessment.GetLocal().GetClean() || warningAssessment.GetLocal().GetCurrentLevel() != "L0" {
		t.Fatalf("warning changed stricter local maturity semantics: %+v", warningAssessment.GetLocal())
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

func TestFixtureExpectationChecksPartialObjectsWithinArrays(t *testing.T) {
	actual := map[string]any{"errors": []any{map[string]any{"class": "invalid_input", "detail": "missing source", "where": "validate"}}}
	if !matchesExpectation(actual, map[string]any{"errors": []any{map[string]any{"class": "invalid_input"}}}) {
		t.Fatal("partial error assertion rejected additional diagnostic fields")
	}
	for _, expected := range []any{
		map[string]any{"errors": []any{map[string]any{"class": "remote_error"}}},
		map[string]any{"errors": []any{}},
		map[string]any{"errors": []any{map[string]any{"class": "invalid_input"}, map[string]any{"class": "invalid_input"}}},
	} {
		if matchesExpectation(actual, expected) {
			t.Fatalf("incorrect error class/count passed: %#v", expected)
		}
	}
}

func TestPortfolioFindingsCoverMaturityGaps(t *testing.T) { // [REQ:PRT-P0-009]
	bad := contracts.Contract{
		ID: "demo.bad", Scenario: "demo", SourceMissing: true, WallMS: 100,
		Bindings: []contracts.BindingRef{{ID: "demo/read/one"}, {ID: "demo/read/two"}},
		Fixtures: []contracts.Fixture{{ID: "synthetic"}},
	}
	response := &programsv1.PortfolioStatsResponse{
		Rows:          []*programsv1.ProgramPortfolioRow{{Name: "demo.bad", P95Millis: 101}},
		NeverExecuted: []string{"demo.bad"},
	}
	findings := portfolioFindings([]contracts.Contract{bad}, response)
	for _, want := range []string{"programs.source_missing_for_contract", "programs.verbs_absent", "programs.no_optional_binding", "programs.no_live_fixture", "programs.budget_exceeded", "programs.never_exercised"} {
		if !containsString(findings, want) {
			t.Fatalf("missing finding %q in %v", want, findings)
		}
	}

	clean := contracts.Contract{
		ID: "demo.clean", Scenario: "demo", Source: "print(1)", WallMS: 100,
		Verbs:    []string{"gather"},
		Bindings: []contracts.BindingRef{{ID: "demo/read/one"}, {ID: "demo/read/two", Optional: true}},
		Fixtures: []contracts.Fixture{{ID: "live", Requires: []string{"demo"}}},
	}
	cleanResponse := &programsv1.PortfolioStatsResponse{Rows: []*programsv1.ProgramPortfolioRow{{Name: "demo.clean", P95Millis: 50}}}
	if got := portfolioFindings([]contracts.Contract{clean}, cleanResponse); len(got) != 0 {
		t.Fatalf("clean contract findings = %v", got)
	}
}

func TestBudgetFindingExplainsHistoricalNameAggregation(t *testing.T) {
	contract := contracts.Contract{ID: "demo.example", WallMS: 100, Digest: "current-digest"}
	response := &programsv1.PortfolioStatsResponse{
		WindowStart: "2026-10-01T00:00:00Z", WindowEnd: "2026-10-30T00:00:00Z",
		Rows: []*programsv1.ProgramPortfolioRow{{Name: contract.ID, P95Millis: 150, Runs: 4}},
	}
	var details []any
	findings := portfolioFindings([]contracts.Contract{contract}, response, &details)
	if !hasFinding(findings, "programs.budget_exceeded") || len(details) != 1 {
		t.Fatalf("missing budget finding and explanation: %v %+v", findings, details)
	}
	detail, ok := details[0].(map[string]any)
	if !ok || detail["program"] != contract.ID || detail["current_digest"] != contract.Digest || detail["runs"] != int64(4) || detail["p95_millis"] != int64(150) || !strings.Contains(detail["message"].(string), "across all digests") {
		t.Fatalf("misleading budget diagnostic: %+v", details[0])
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type fixtureTestRunner struct{ calls int }

type historicalBudgetPortfolio struct{}

type warningOnlyPortfolio struct{}

type unavailablePortfolio struct{ nilResponse bool }

func (p unavailablePortfolio) PortfolioStats(context.Context, *programsv1.PortfolioStatsRequest) (*programsv1.PortfolioStatsResponse, error) {
	if p.nilResponse {
		return nil, nil
	}
	return nil, fmt.Errorf("portfolio offline")
}

func TestUnavailablePortfolioCannotProduceCleanValidation(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, nilResponse := range []bool{false, true} {
		h := &handler{repoRoot: root, portfolio: unavailablePortfolio{nilResponse: nilResponse}}
		response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "program-runtime"}))
		if err != nil {
			t.Fatal(err)
		}
		if response.Msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED || !containsString(response.Msg.GetAssessment().GetLocal().GetBlockingFindingCodes(), "programs.portfolio_unavailable") {
			t.Fatalf("unavailable portfolio produced clean result: %v", response.Msg)
		}
	}
}

func (warningOnlyPortfolio) PortfolioStats(context.Context, *programsv1.PortfolioStatsRequest) (*programsv1.PortfolioStatsResponse, error) {
	return &programsv1.PortfolioStatsResponse{NeverExecuted: []string{"tech-tree-designer.setpoint-read"}}, nil
}

func (historicalBudgetPortfolio) PortfolioStats(context.Context, *programsv1.PortfolioStatsRequest) (*programsv1.PortfolioStatsResponse, error) {
	return &programsv1.PortfolioStatsResponse{Rows: []*programsv1.ProgramPortfolioRow{{Name: "tech-tree-designer.setpoint-read", P95Millis: 60004}}}, nil
}

type recoveryFixtureRunner struct {
	calls int
	fail  bool
}

func (r *recoveryFixtureRunner) RunDeclaredProgram(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	r.calls++
	if req.Msg.GetProvenance() != programsv1.Provenance_PROVENANCE_TEST || req.Msg.GetExpectedDigest() == "" {
		return nil, fmt.Errorf("fixture must be pinned and test-provenance")
	}
	if req.Msg.GetInputs().GetFields()["collect_diagnostics"].GetStringValue() == "false" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid boolean"))
	}
	if r.fail {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("fixture unavailable"))
	}
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: `{"status":"ok","signals":{"acceptance":"unknown","unresolved_outcomes":18,"diagnostics":[]}}`}}), nil
}

// Exercise the shipped recovery case through admission, not just the fixture
// helper: historical debt must neither starve fresh evidence nor become PASS.
func TestHistoricalBudgetDebtDoesNotSuppressFreshFixtures(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := bindings.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		scenario      string
		execute, fail bool
		wantCalls     int
	}{
		{"recovery", "tech-tree-designer", true, false, 3},
		{"new-failure", "tech-tree-designer", true, true, 3},
		{"static-only", "tech-tree-designer", false, false, 0},
		{"invalid-contract", "missing-fixture-scenario", true, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &recoveryFixtureRunner{fail: tc.fail}
			h := &handler{repoRoot: root, registry: registry, runner: runner, portfolio: historicalBudgetPortfolio{}}
			response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: tc.scenario, IncludeExecution: tc.execute}))
			if err != nil {
				t.Fatal(err)
			}
			findings := response.Msg.GetAssessment().GetLocal().GetBlockingFindingCodes()
			if runner.calls != tc.wantCalls {
				t.Fatalf("calls = %d, want %d; findings = %v", runner.calls, tc.wantCalls, findings)
			}
			if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
				t.Fatal("unresolved debt became a passing qualification")
			}
			if tc.scenario == "tech-tree-designer" && !containsString(findings, "programs.budget_exceeded") {
				t.Fatalf("historical debt lost: %v", findings)
			}
			fixtureFailure := false
			for _, finding := range findings {
				fixtureFailure = fixtureFailure || strings.HasPrefix(finding, "programs.fixture:")
			}
			if fixtureFailure != tc.fail {
				t.Fatalf("fresh evidence lost or invented: %v", findings)
			}
		})
	}
}

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
