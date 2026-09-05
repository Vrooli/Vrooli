package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateScenarioReportsMissingTarget(t *testing.T) {
	root := t.TempDir()
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "missing", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	require.Equal(t, ResolutionMissing, report.Target.Resolution)
	require.Len(t, report.Findings, 1)
	require.Equal(t, CodeTargetUnresolved, report.Findings[0].Code)
	require.Equal(t, SeverityError, report.Findings[0].Severity)
}

func TestValidateScenarioReportsNoAPISurface(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "docs-only", ".vrooli", "service.json"), `{"service":{"name":"docs-only"}}`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "docs-only", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.Equal(t, ResolutionResolved, report.Target.Resolution)
	require.Equal(t, APIKindAbsent, report.Target.APIKind)
	require.Len(t, report.Findings, 1)
	require.Equal(t, CodeAPISurfaceAbsent, report.Findings[0].Code)
	require.Equal(t, SeverityInfo, report.Findings[0].Severity)
}

func TestValidateScenarioClassifiesGoAPI(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.Empty(t, report.Findings)
	require.Equal(t, APIKindGo, report.Target.APIKind)
	require.True(t, report.Target.HasAPIDir)
	require.True(t, report.Target.Service.PortsAPI)
	require.Equal(t, "/health", report.Target.Service.HealthAPIPath)
	require.True(t, report.Target.Service.HealthAPICheck)
	require.True(t, report.Target.Lifecycle.ManifestHealthy)
	require.True(t, report.Target.Lifecycle.PreflightHealthy)
	require.True(t, report.Target.Lifecycle.ServerRunnerHealthy)
}

func TestValidateScenarioUsesExplicitPath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "external")
	writeFile(t, filepath.Join(target, ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(target, "api", "main.go"), compliantMain("external"))
	svc := New(Deps{RepoRoot: t.TempDir()})

	report, err := svc.ValidateScenario(context.Background(), "", target, false)
	require.NoError(t, err)
	require.Equal(t, "external", report.Scenario)
	require.Equal(t, APIKindGo, report.Target.APIKind)
	require.Empty(t, report.Findings)
}

func validServiceJSON() string {
	return `{
		"ports":{"api":{}},
		"lifecycle":{"health":{
			"endpoints":{"api":"/health"},
			"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:${API_PORT}/health"}]
		}}
	}`
}

func compliantMain(scenario string) string {
	return `package main

import (
	"log"
	"net/http"

	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "` + scenario + `"}) {
		return
	}
	handler := http.NewServeMux()
	if err := apiserver.Run(apiserver.Config{Handler: handler}); err != nil {
		log.Fatal(err)
	}
}
`
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
