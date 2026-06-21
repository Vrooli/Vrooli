package dochealth

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"

	"knowledge-observatory/internal/services/dochealing"
	"knowledge-observatory/internal/services/dochealth"

	"github.com/vrooli/maturity-go/assessment"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// stubFixer satisfies DocFixer for the shared Fix RPC tests.
type stubFixer struct {
	dryRunResult *dochealing.AutoFixResult
	applyResult  *dochealing.AutoFixResult
}

func (s *stubFixer) AutoFix(_ context.Context, scenario string, dryRun bool) (*dochealing.AutoFixResult, error) {
	if dryRun {
		return s.dryRunResult, nil
	}
	return s.applyResult, nil
}

func TestPreviewFixMapsMovesToCandidates(t *testing.T) {
	h := NewWithDeps(Deps{Fixer: &stubFixer{dryRunResult: &dochealing.AutoFixResult{
		ScenarioName: "demo",
		Moved:        []dochealing.MovedDoc{{FromPath: "X.md", ToPath: "docs/X.md", DocType: "guide"}},
		Skipped:      []dochealing.SkippedDoc{{FromPath: "Y.md", ToPath: "docs/Y.md", Reason: "destination already exists"}},
	}}})
	resp, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("PreviewFix: %v", err)
	}
	if resp.Msg.GetApplied() {
		t.Fatalf("preview applied = true, want false")
	}
	cands := resp.Msg.GetCandidates()
	if len(cands) != 1 || cands[0].GetRuleId() != "misplaced_doc" || cands[0].GetFilePath() != "docs/X.md" {
		t.Fatalf("unexpected candidates: %+v", cands)
	}
	if cands[0].GetApplied() {
		t.Fatalf("preview candidate must not be applied")
	}
	if len(resp.Msg.GetMessages()) != 1 {
		t.Fatalf("expected one skipped message, got %v", resp.Msg.GetMessages())
	}
}

func TestFixUnimplementedWithoutFixer(t *testing.T) {
	h := NewWithDeps(Deps{})
	if _, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo"})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("code = %v, want Unimplemented", connect.CodeOf(err))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newServiceFixture(t *testing.T) (*dochealth.Service, string) {
	t.Helper()
	root := t.TempDir()
	scenario := filepath.Join(root, "demo")
	writeFile(t, filepath.Join(scenario, "README.md"), "# demo\n")
	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), `{"version":"1","docs":[]}`)
	svc, err := dochealth.NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, "demo"
}

func testMaturitySpec(t *testing.T) *assessment.Spec {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatalf("read maturity spec: %v", err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatalf("parse maturity spec: %v", err)
	}
	return spec
}

func TestHandler_DocHealth_ReturnsResponse(t *testing.T) {
	svc, name := newServiceFixture(t)
	h := NewWithDeps(Deps{Service: svc, MaturitySpec: testMaturitySpec(t)})

	resp, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: name,
	}))
	if err != nil {
		t.Fatalf("DocHealth: %v", err)
	}
	if resp == nil || resp.Msg == nil {
		t.Fatalf("nil response")
	}
	if resp.Msg.GetScenarioName() != name {
		t.Errorf("scenario_name = %q, want %q", resp.Msg.GetScenarioName(), name)
	}
	if resp.Msg.GetCounts() == nil {
		t.Errorf("counts is nil")
	}
	if resp.Msg.GetTimestamp() == "" {
		t.Errorf("timestamp empty")
	}
	if resp.Msg.GetAssessment() == nil {
		t.Fatalf("assessment is nil")
	}
	if got := resp.Msg.GetAssessment().GetProvider(); got != "knowledge-observatory" {
		t.Errorf("assessment.provider = %q, want knowledge-observatory", got)
	}
	if got := resp.Msg.GetAssessment().GetPhase(); got != "docs" {
		t.Errorf("assessment.phase = %q, want docs", got)
	}
	if got := resp.Msg.GetAssessment().GetLocal().GetCurrentLevel(); got == "" {
		t.Errorf("assessment.local.current_level empty")
	}
}

func TestDocHealthMaturitySpecCoversKnownFindingCodes(t *testing.T) {
	spec := testMaturitySpec(t)
	codes := []string{
		"schema_load_error",
		"schema_violation",
		"manifest_missing",
		"duplicate_identifier",
		"invalid_maturity",
		"invalid_required_by",
		"invalid_contract_kind",
		"invalid_contract_schema",
		"missing_maturity_values",
		"missing_stages",
		"missing_section_id",
		"missing_section_title",
		"missing_document_path",
		"missing_doc_type",
		"missing_document_title",
		"missing_maturity",
		"missing_required_by",
		"missing_completion",
		"invalid_completion",
		"append_log_missing_heading",
		"append_log_invalid_fields",
		"append_log_invalid_date_source",
		"append_log_invalid_format",
		"misplaced_doc",
		"missing_doc",
		"extra_doc",
		"temporary_doc",
		"manifest_missing_doc",
		"manifest_orphaned_doc",
		"broken_link_parse",
		"broken_local_link",
		"external_link_warning",
		"broken_external_link",
		"mermaid_invalid",
		"broken_code_ref",
		"unknown_marked_ref",
		"broken_marked_ref",
		"broken_doc_ref",
		"absolute_path",
		"markdown_unclosed_fence",
		"number_marker_without_reason",
		"unmarked_number",
		"content_issue",
		"file_read_error",
	}
	for _, code := range codes {
		if _, ok := spec.Findings[code]; !ok {
			t.Fatalf("maturity spec missing finding code %q", code)
		}
	}
}

func TestHandler_DocHealth_InvalidName(t *testing.T) {
	svc, _ := newServiceFixture(t)
	h := NewWithDeps(Deps{Service: svc, MaturitySpec: testMaturitySpec(t)})

	_, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: "../escape",
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestHandler_DocHealth_NotFound(t *testing.T) {
	svc, _ := newServiceFixture(t)
	h := NewWithDeps(Deps{Service: svc, MaturitySpec: testMaturitySpec(t)})

	_, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: "no-such-scenario",
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestHandler_DocHealth_UnavailableWhenServiceNil(t *testing.T) {
	h := New(nil)
	_, err := h.DocHealth(context.Background(), connect.NewRequest(&kov1.DocHealthRequest{
		ScenarioName: "demo",
	}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("code = %v, want Unavailable", connect.CodeOf(err))
	}
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	svc, name := newServiceFixture(t)
	h := NewWithDeps(Deps{Service: svc, MaturitySpec: testMaturitySpec(t)})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: name,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	m := resp.Msg.GetMetrics()
	if m == nil {
		t.Fatal("metrics must be attached to the response")
	}
	if m.GetWallClockMs() < 0 {
		t.Fatalf("wall clock must be non-negative, got %d", m.GetWallClockMs())
	}
	env := m.GetEnvironment()
	if env == nil {
		t.Fatal("metrics environment must be populated with the stdlib baseline")
	}
	if env.GetOs() != runtime.GOOS {
		t.Fatalf("env os = %q, want %q", env.GetOs(), runtime.GOOS)
	}
	if env.GetArch() != runtime.GOARCH {
		t.Fatalf("env arch = %q, want %q", env.GetArch(), runtime.GOARCH)
	}
	if env.GetNumCpu() != int32(runtime.NumCPU()) {
		t.Fatalf("env num_cpu = %d, want %d", env.GetNumCpu(), runtime.NumCPU())
	}
}
