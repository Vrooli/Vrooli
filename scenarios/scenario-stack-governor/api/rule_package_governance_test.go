package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type fakeStructureHealth struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	findings []*commonv1.AssessmentFinding
}

func (f fakeStructureHealth) ValidateTarget(context.Context, *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Assessment: &commonv1.MaturityAssessment{Findings: f.findings},
	}), nil
}

func TestPackageGovernanceRule_FiltersToScenarioPaths(t *testing.T) {
	root := setupPackageGovernanceTestRepo(t)
	server := structureHealthTestServer(t, []*commonv1.AssessmentFinding{
		{Severity: "error", Code: "SCENARIO_WORKSPACE_DEPENDENCY", Message: "workspace protocol is not allowed", Location: "scenarios/alpha/ui/package.json"},
		{Severity: "error", Code: "PACKAGE_MANIFEST_MISSING", Message: "package manifest is missing", Location: "packages/beta/.vrooli/package.json"},
	})
	defer server.Close()
	testStructureHealthURL(t, server.URL)

	result := RunPackageGovernanceScenarioAdoption(t.Context(), root, "alpha")
	if result.Passed || len(result.Findings) != 1 {
		t.Fatalf("expected one filtered finding, got %+v", result)
	}
	if result.Findings[0].ScenarioName != "alpha" || !strings.Contains(result.Findings[0].Message, "workspace") {
		t.Fatalf("unexpected finding: %+v", result.Findings[0])
	}
}

func TestPackageGovernanceRule_ScansAllScenariosWhenUnscoped(t *testing.T) {
	root := setupPackageGovernanceTestRepo(t)
	server := structureHealthTestServer(t, []*commonv1.AssessmentFinding{
		{Severity: "warning", Code: "SCENARIO_UI_LOCKFILE_MISSING", Message: "lockfile missing", Location: "scenarios/alpha/ui/pnpm-lock.yaml"},
		{Severity: "error", Code: "PACKAGE_MANIFEST_INVALID", Message: "manifest invalid", Location: "scenarios/beta/api/.vrooli/package.json"},
	})
	defer server.Close()
	testStructureHealthURL(t, server.URL)

	result := RunPackageGovernanceScenarioAdoption(t.Context(), root, "")
	if len(result.Findings) != 2 || result.Findings[0].ScenarioName == result.Findings[1].ScenarioName {
		t.Fatalf("expected findings for different scenarios: %+v", result.Findings)
	}
}

func TestPackageGovernanceRule_ReportsProviderFailure(t *testing.T) {
	t.Setenv("VROOLI_STRUCTURE_HEALTH_TEST", "")
	original := resolveStructureHealthURL
	resolveStructureHealthURL = func(context.Context, string) (string, error) { return "", errors.New("provider unavailable") }
	t.Cleanup(func() { resolveStructureHealthURL = original })

	result := RunPackageGovernanceScenarioAdoption(t.Context(), setupPackageGovernanceTestRepo(t), "alpha")
	if result.Passed || len(result.Findings) != 1 || result.Findings[0].Level != "error" {
		t.Fatalf("expected provider failure finding: %+v", result)
	}
}

func structureHealthTestServer(t *testing.T, findings []*commonv1.AssessmentFinding) *httptest.Server {
	t.Helper()
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(fakeStructureHealth{findings: findings})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return httptest.NewServer(mux)
}

func testStructureHealthURL(t *testing.T, url string) {
	t.Helper()
	originalURL := resolveStructureHealthURL
	originalClient := structureHealthHTTPClient
	resolveStructureHealthURL = func(context.Context, string) (string, error) { return url, nil }
	structureHealthHTTPClient = http.DefaultClient
	t.Cleanup(func() {
		resolveStructureHealthURL = originalURL
		structureHealthHTTPClient = originalClient
	})
}

func setupPackageGovernanceTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".vrooli", "scenarios", "resources"} {
		mkdirAll(t, filepath.Join(root, dir))
	}
	return root
}
