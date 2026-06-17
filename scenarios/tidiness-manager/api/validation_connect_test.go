package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

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
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(newScenarioValidationHandler(&Server{}))
	router.Handle(path, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan/tidiness", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("/api/v1/scan/tidiness status = %d, want 404", rec.Code)
	}
}
