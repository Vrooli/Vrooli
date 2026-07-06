package maturity

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestScanEmptyFleetDoesNotCallProvider(t *testing.T) {
	root := makeRepo(t)
	client := &fakeValidationClient{}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, out := scanContext(root, true)
	if err := newHandlers(nil).scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}

	var report scanReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal scan JSON: %v\n%s", err, out.String())
	}
	if report.Summary.Total != 0 {
		t.Fatalf("total = %d, want 0", report.Summary.Total)
	}
	if client.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", client.calls)
	}
	if report.ProviderLiveness.State != "not_checked" {
		t.Fatalf("provider liveness = %q, want not_checked", report.ProviderLiveness.State)
	}
}

func TestScanReportsMixedMaturityAndGroupsFindings(t *testing.T) {
	root := makeRepo(t)
	writeSearchDescriptor(t, root, "clean")
	writeSearchDescriptor(t, root, "blocked")
	client := &fakeValidationClient{responses: map[string]*scenariovalidationv1.ValidateScenarioResponse{
		"clean": {
			Scenario:   "clean",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: cleanAssessment("clean"),
		},
		"blocked": {
			Scenario:   "blocked",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			Assessment: findingAssessment("blocked", "SEARCH_EVAL_CORPUS_MISSING", "SEVERITY_ERROR", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED),
		},
	}}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, out := scanContext(root, true)
	if err := newHandlers(nil).scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}

	var report scanReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal scan JSON: %v\n%s", err, out.String())
	}
	if report.Summary.Total != 2 || report.Summary.Passed != 1 || report.Summary.Failed != 1 {
		t.Fatalf("summary = %#v, want one passed and one failed", report.Summary)
	}
	if report.Summary.Blocking != 1 || report.Summary.Advisory != 0 {
		t.Fatalf("blocking/advisory = %d/%d, want 1/0", report.Summary.Blocking, report.Summary.Advisory)
	}
	if got := report.Groups.ByFinding["SEARCH_EVAL_CORPUS_MISSING"]; len(got) != 1 || got[0] != "blocked" {
		t.Fatalf("finding group = %#v, want [blocked]", got)
	}
	blocked := findResult(t, report, "blocked")
	if !strings.Contains(blocked.RecommendedNextAction, "SEARCH_EVAL_CORPUS_MISSING") {
		t.Fatalf("next action = %q, want finding code", blocked.RecommendedNextAction)
	}
	if blocked.Findings[0].CapabilityID != "search_eval_performance" {
		t.Fatalf("capability = %q, want search_eval_performance", blocked.Findings[0].CapabilityID)
	}
}

func TestScanKeepsAdvisoryDebtSeparateFromBlockingMaturity(t *testing.T) {
	root := makeRepo(t)
	writeSearchDescriptor(t, root, "advisory")
	client := &fakeValidationClient{responses: map[string]*scenariovalidationv1.ValidateScenarioResponse{
		"advisory": {
			Scenario:   "advisory",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: findingAssessment("advisory", "SEARCH_STATUS_ENDPOINT_MISSING", "SEVERITY_WARNING", commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY),
		},
	}}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, out := scanContext(root, true)
	if err := newHandlers(nil).scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}

	var report scanReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal scan JSON: %v\n%s", err, out.String())
	}
	if report.Summary.Passed != 1 || report.Summary.Blocking != 0 || report.Summary.Advisory != 1 {
		t.Fatalf("summary = %#v, want passed advisory-only result", report.Summary)
	}
	result := findResult(t, report, "advisory")
	if !result.Findings[0].Advisory || result.Findings[0].Gating {
		t.Fatalf("finding advisory/gating = %v/%v, want true/false", result.Findings[0].Advisory, result.Findings[0].Gating)
	}
}

func TestScanRequestsExecutionModeValidationByDefault(t *testing.T) {
	root := makeRepo(t)
	writeSearchDescriptor(t, root, "demo")
	client := &fakeValidationClient{}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, _ := scanContext(root, true)
	if err := newHandlers(nil).scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !client.lastIncludeExecution {
		t.Fatal("scan did not set ValidateScenarioRequest.include_execution")
	}
}

func TestScanFastModeSkipsExecutionModeValidation(t *testing.T) {
	root := makeRepo(t)
	writeSearchDescriptor(t, root, "demo")
	client := &fakeValidationClient{}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, _ := scanContextWithOptions(root, true, true, false)
	if err := newHandlers(nil).scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if client.lastIncludeExecution {
		t.Fatal("fast scan should not set ValidateScenarioRequest.include_execution")
	}
}

func TestScanReportsProviderUnavailableWithoutTargetFailureFindings(t *testing.T) {
	root := makeRepo(t)
	writeSearchDescriptor(t, root, "demo")
	client := &fakeValidationClient{err: errors.New("connection refused")}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, out := scanContext(root, true)
	if err := newHandlers(nil).scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}

	var report scanReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal scan JSON: %v\n%s", err, out.String())
	}
	if report.Summary.Unavailable != 1 || report.Summary.Failed != 0 || report.Summary.Findings != 0 {
		t.Fatalf("summary = %#v, want unavailable without target findings", report.Summary)
	}
	result := findResult(t, report, "demo")
	if result.ProviderLiveness.State != "unavailable" || result.Status != "unavailable" {
		t.Fatalf("result status/liveness = %s/%s, want unavailable/unavailable", result.Status, result.ProviderLiveness.State)
	}
}

func TestHumanScanOutputIncludesFindingsAndNextAction(t *testing.T) {
	root := makeRepo(t)
	writeSearchDescriptor(t, root, "blocked")
	client := &fakeValidationClient{responses: map[string]*scenariovalidationv1.ValidateScenarioResponse{
		"blocked": {
			Scenario:   "blocked",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			Assessment: findingAssessment("blocked", "SEARCH_EVAL_CORPUS_MISSING", "SEVERITY_ERROR", commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED),
		},
	}}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, out := scanContext(root, false)
	if err := newHandlers(nil).scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}
	got := out.String()
	for _, want := range []string{"Search maturity scan", "blocked", "SEARCH_EVAL_CORPUS_MISSING", "Resolve blocking"} {
		if !strings.Contains(got, want) {
			t.Fatalf("human output missing %q:\n%s", want, got)
		}
	}
}

func TestFixPreviewsCandidatesByDefault(t *testing.T) {
	client := &fakeValidationClient{fixResp: &scenariovalidationv1.FixResponse{
		Scenario: "demo",
		Candidates: []*scenariovalidationv1.FixCandidate{{
			RuleId:      "SEARCH_CONFIG_INVALID",
			FilePath:    "/tmp/demo/.vrooli/search.json",
			Description: "Set search descriptor version to 1.0.0.",
		}},
	}}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, out := fixContext("demo", false, false)
	if err := newHandlers(nil).fix(ctx); err != nil {
		t.Fatalf("fix preview: %v", err)
	}
	if client.previewCalls != 1 || client.applyCalls != 0 {
		t.Fatalf("preview/apply calls = %d/%d, want 1/0", client.previewCalls, client.applyCalls)
	}
	got := out.String()
	for _, want := range []string{"Search maturity fix preview", "SEARCH_CONFIG_INVALID", "Re-run with --apply"} {
		if !strings.Contains(got, want) {
			t.Fatalf("human fix output missing %q:\n%s", want, got)
		}
	}
}

func TestFixApplyUsesApplyRPCAndJSON(t *testing.T) {
	client := &fakeValidationClient{fixResp: &scenariovalidationv1.FixResponse{
		Scenario: "demo",
		Applied:  true,
		Candidates: []*scenariovalidationv1.FixCandidate{{
			RuleId:  "SEARCH_EVAL_CORPUS_INVALID",
			Applied: true,
		}},
	}}
	restore := replaceValidationClient(client)
	defer restore()

	ctx, out := fixContext("demo", true, true)
	if err := newHandlers(nil).fix(ctx); err != nil {
		t.Fatalf("fix apply: %v", err)
	}
	if client.previewCalls != 0 || client.applyCalls != 1 {
		t.Fatalf("preview/apply calls = %d/%d, want 0/1", client.previewCalls, client.applyCalls)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal fix JSON: %v\n%s", err, out.String())
	}
	candidates, _ := payload["candidates"].([]any)
	if payload["applied"] != true || len(candidates) != 1 {
		t.Fatalf("JSON fix output missing applied candidate:\n%s", out.String())
	}
	candidate, _ := candidates[0].(map[string]any)
	if candidate["rule_id"] != "SEARCH_EVAL_CORPUS_INVALID" || candidate["applied"] != true {
		t.Fatalf("candidate JSON = %#v, want applied SEARCH_EVAL_CORPUS_INVALID", candidate)
	}
	if len(client.lastRules) != 1 || client.lastRules[0] != "SEARCH_EVAL_CORPUS_INVALID" {
		t.Fatalf("rule filter = %#v, want SEARCH_EVAL_CORPUS_INVALID", client.lastRules)
	}
}

type fakeValidationClient struct {
	responses            map[string]*scenariovalidationv1.ValidateScenarioResponse
	fixResp              *scenariovalidationv1.FixResponse
	err                  error
	calls                int
	previewCalls         int
	applyCalls           int
	lastRules            []string
	lastIncludeExecution bool
}

func (f *fakeValidationClient) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	f.calls++
	f.lastIncludeExecution = req.Msg.GetIncludeExecution()
	if f.err != nil {
		return nil, f.err
	}
	resp := f.responses[req.Msg.GetScenario()]
	if resp == nil {
		resp = &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   req.Msg.GetScenario(),
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: cleanAssessment(req.Msg.GetScenario()),
		}
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeValidationClient) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	f.previewCalls++
	f.lastRules = append([]string(nil), req.Msg.GetRuleIds()...)
	if f.err != nil {
		return nil, f.err
	}
	if f.fixResp != nil {
		return connect.NewResponse(f.fixResp), nil
	}
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Scenario: req.Msg.GetScenario()}), nil
}

func (f *fakeValidationClient) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	f.applyCalls++
	f.lastRules = append([]string(nil), req.Msg.GetRuleIds()...)
	if f.err != nil {
		return nil, f.err
	}
	if f.fixResp != nil {
		return connect.NewResponse(f.fixResp), nil
	}
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Scenario: req.Msg.GetScenario(), Applied: true}), nil
}

func replaceValidationClient(client validationClient) func() {
	original := newValidationClient
	newValidationClient = func(*cliapp.ScenarioApp, time.Duration) validationClient {
		return client
	}
	return func() { newValidationClient = original }
}

func fixContext(scenario string, apply bool, asJSON bool) (cliapp.RunContext, interface {
	Bytes() []byte
	String() string
},
) {
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "path"},
			{Name: "rule"},
			{Name: "timeout", Default: "30s"},
			{Name: "apply", Bool: true},
		},
	}
	ctx, out := cliapptest.NewCapturedRunContext(nil, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": scenario},
		Flags: map[string]string{
			"timeout": "5s",
			"rule":    "SEARCH_EVAL_CORPUS_INVALID",
		},
		BoolFlags: map[string]bool{"apply": apply},
		JSON:      asJSON,
	})
	return ctx, out
}

func scanContext(root string, asJSON bool) (cliapp.RunContext, interface {
	Bytes() []byte
	String() string
},
) {
	return scanContextWithOptions(root, asJSON, false, false)
}

func scanContextWithOptions(root string, asJSON bool, fast bool, includeEvals bool) (cliapp.RunContext, interface {
	Bytes() []byte
	String() string
},
) {
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "root", Default: "."},
		{Name: "timeout", Default: "30s"},
		{Name: "fast", Bool: true},
		{Name: "include-evals", Bool: true},
	}}
	ctx, out := cliapptest.NewCapturedRunContext(nil, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"root":    root,
			"timeout": "5s",
		},
		BoolFlags: map[string]bool{"fast": fast, "include-evals": includeEvals},
		JSON:      asJSON,
	})
	return ctx, out
}

func makeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeSearchDescriptor(t *testing.T, root, scenario string) {
	t.Helper()
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "search.json"), []byte(`{"version":"1.0.0","providers":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func cleanAssessment(scenario string) *commonv1.MaturityAssessment {
	return &commonv1.MaturityAssessment{
		Scenario: scenario,
		Provider: "search-hub",
		Phase:    "search",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: "L4",
			Clean:        true,
		},
		Capabilities: []*commonv1.CapabilityMaturityAssessment{{
			Id:           "search_descriptor",
			Label:        "Search descriptor",
			CurrentLevel: "L4",
			Clean:        true,
		}},
	}
}

func findingAssessment(scenario, code, severity string, clean commonv1.CleanRequirement) *commonv1.MaturityAssessment {
	blocking := []string(nil)
	if clean == commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED {
		blocking = []string{code}
	}
	return &commonv1.MaturityAssessment{
		Scenario: scenario,
		Provider: "search-hub",
		Phase:    "search",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel:         "L2",
			NextLevel:            "L3",
			BlockingFindingCodes: blocking,
		},
		HighestPriorityCapability: &commonv1.PriorityFocus{
			CapabilityId:    "search_eval_performance",
			CapabilityLabel: "Search eval performance",
			CurrentLevel:    "L2",
			NextLevel:       "L3",
			Reason:          "finding debt",
		},
		Capabilities: []*commonv1.CapabilityMaturityAssessment{{
			Id:                   "search_eval_performance",
			Label:                "Search eval performance",
			CurrentLevel:         "L2",
			NextLevel:            "L3",
			BlockingFindingCodes: blocking,
			FindingsBySeverity:   map[string]int32{severity: 1},
		}},
		Findings: []*commonv1.AssessmentFinding{{
			Code:        code,
			Severity:    severity,
			Title:       "Search maturity finding",
			Message:     "finding message",
			Location:    ".vrooli/search.json",
			Remediation: "fix search maturity",
			Maturity: &commonv1.FindingMaturity{
				LocalLevel:       "L3",
				CapabilityId:     "search_eval_performance",
				CleanRequirement: clean,
			},
		}},
	}
}

func findResult(t *testing.T, report scanReport, scenario string) scenarioResult {
	t.Helper()
	for _, result := range report.Results {
		if result.Scenario == scenario {
			return result
		}
	}
	t.Fatalf("missing result %s in %#v", scenario, report.Results)
	return scenarioResult{}
}
