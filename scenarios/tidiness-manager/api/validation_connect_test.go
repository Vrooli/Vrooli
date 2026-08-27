package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/maturity-go/assessment"

	"connectrpc.com/connect"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tidiness-manager/v1/validation"
)

func TestScenarioValidationHandlerPacksTidinessNativeDetail(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	var content strings.Builder
	content.WriteString("package main\n\nfunc main() {}\n")
	for i := 0; i < 520; i++ {
		content.WriteString("// filler line\n")
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(content.String()), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	handler := newScenarioValidationHandler(&Server{})
	resp, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
		Path:     tmpDir,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario() error = %v", err)
	}
	if resp.Msg.GetScenario() != "demo" || resp.Msg.GetAssessment() == nil {
		t.Fatalf("shared response missing scenario/assessment: %+v", resp.Msg)
	}
	var native validationv1.TidinessScanResponse
	if err := resp.Msg.GetNativeDetail().UnmarshalTo(&native); err != nil {
		t.Fatalf("native_detail unmarshal failed: %v", err)
	}
	if native.GetSummary().GetLongFiles() == 0 || len(native.GetFindings()) == 0 {
		t.Fatalf("native detail missing tidiness findings: %+v", native.GetSummary())
	}
}

func TestTidinessNativeDetailCarriesDuplicationLineDebt(t *testing.T) {
	result, err := tidinessScanToProto(&TidinessScanResponse{
		Scenario: "demo",
		Summary:  TidinessScanSummary{DuplicationLineDebt: 37},
	})
	if err != nil {
		t.Fatalf("tidinessScanToProto() error = %v", err)
	}
	if got := result.GetSummary().GetDuplicationLineDebt(); got != 37 {
		t.Fatalf("duplication_line_debt = %d, want 37", got)
	}
}

func TestValidateTargetHonorsContractExcludesWhileToolTargetReportsFindings(t *testing.T) {
	repoRoot := t.TempDir()
	writeLongGoFile := func(path string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		content := "package main\n\nfunc main() {}\n"
		for i := 0; i < 520; i++ {
			content += "// filler line\n"
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeLongGoFile(filepath.Join(repoRoot, "internal", "tools", "vault", "main.go"))
	writeLongGoFile(filepath.Join(repoRoot, "internal", "safeguards", "secret", "main.go"))

	handler := newScenarioValidationHandler(&Server{
		scenarioLocator: &ScenarioLocator{repoRoot: repoRoot},
	})
	controlPlane, err := handler.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE,
			Id:   "internal",
			Root: "internal",
		},
		Path:    filepath.Join(repoRoot, "internal"),
		Exclude: []string{"internal/tools/*", "internal/safeguards/*"},
	}))
	if err != nil {
		t.Fatalf("control-plane ValidateTarget() error = %v", err)
	}
	var controlDetail validationv1.TidinessScanResponse
	if err := controlPlane.Msg.GetNativeDetail().UnmarshalTo(&controlDetail); err != nil {
		t.Fatalf("control-plane native_detail unmarshal failed: %v", err)
	}
	for _, finding := range controlDetail.GetFindings() {
		if strings.HasPrefix(finding.GetFilePath(), "tools/") || strings.HasPrefix(finding.GetFilePath(), "safeguards/") {
			t.Fatalf("control-plane finding escaped contract exclude: %q", finding.GetFilePath())
		}
	}

	tool, err := handler.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL,
			Id:   "vault",
			Root: "internal/tools/vault",
		},
		Path: filepath.Join(repoRoot, "internal", "tools", "vault"),
	}))
	if err != nil {
		t.Fatalf("tool ValidateTarget() error = %v", err)
	}
	var toolDetail validationv1.TidinessScanResponse
	if err := tool.Msg.GetNativeDetail().UnmarshalTo(&toolDetail); err != nil {
		t.Fatalf("tool native_detail unmarshal failed: %v", err)
	}
	if toolDetail.GetSummary().GetLongFiles() == 0 {
		t.Fatalf("tool target should report its long file: %+v", toolDetail.GetSummary())
	}
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	handler := newScenarioValidationHandler(&Server{})
	resp, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
		Path:     tmpDir,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario() error = %v", err)
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

func TestTidinessFindingsToProtoNormalizesTypedEvidence(t *testing.T) {
	findings, err := tidinessFindingsToProto([]TidinessFinding{{
		RuleID:   "DUPLICATED_CODE",
		Evidence: map[string]any{"locations": []DuplicateLocation{{Path: "api/main.go", StartLine: 10, EndLine: 20}}},
	}})
	if err != nil {
		t.Fatalf("tidinessFindingsToProto() error = %v", err)
	}
	locations := findings[0].GetEvidence().GetFields()["locations"].GetListValue().GetValues()
	if len(locations) != 1 {
		t.Fatalf("locations length = %d, want 1", len(locations))
	}
	if got := locations[0].GetStructValue().GetFields()["path"].GetStringValue(); got != "api/main.go" {
		t.Fatalf("location path = %q, want api/main.go", got)
	}
}

func TestTidinessScanRESTEndpointRemoved(t *testing.T) {
	router := http.NewServeMux()
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(newScenarioValidationHandler(&Server{}), assessment.Describer{}))
	router.Handle(path, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan/tidiness", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("/api/v1/scan/tidiness status = %d, want 404", rec.Code)
	}
}
