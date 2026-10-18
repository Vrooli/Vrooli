package validation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateScenarioReportsMissingLifecycleHealthMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), `{"ports":{"ui":{}},"lifecycle":{"health":{"endpoints":{"api":"/healthz"}}}}`)
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeServiceHealthMissing)
	require.False(t, report.Target.Lifecycle.ManifestHealthy)
}

func TestValidateScenarioReportsPreflightScenarioMismatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("wrong-app"))
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodePreflightMissingLate)
	require.False(t, report.Target.Lifecycle.PreflightHealthy)
}

func TestValidateScenarioReportsLatePreflight(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

func main() {
	fmt.Println("starting")
	if preflight.Run(preflight.Config{ScenarioName: "api-app"}) {
		return
	}
	if err := apiserver.Run(apiserver.Config{Handler: http.NewServeMux()}); err != nil {
		log.Fatal(err)
	}
}
`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	finding := requireFinding(t, report, CodePreflightMissingLate)
	require.Contains(t, finding.Message, "before api-core preflight")
}

func TestValidateScenarioAllowsLogFlagsBeforePreflight(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"log"
	"net/http"

	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	if preflight.Run(preflight.Config{ScenarioName: "api-app"}) {
		return
	}
	if err := apiserver.Run(apiserver.Config{Handler: http.NewServeMux()}); err != nil {
		log.Fatal(err)
	}
}
`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.Empty(t, report.Findings)
}

func TestValidateScenarioReportsDirectListenAndServe(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"log"
	"net/http"

	"github.com/vrooli/api-core/preflight"
)

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "api-app"}) {
		return
	}
	log.Fatal(http.ListenAndServe(":8080", http.NewServeMux()))
}
`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeServerRunnerMissing)
	require.True(t, report.Target.Lifecycle.DirectListenAndServe)
}

func TestValidateScenarioAllowsServerRunStartServerCallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main
import ("net/http"; "github.com/vrooli/api-core/preflight"; apiserver "github.com/vrooli/api-core/server")
func main() { if preflight.Run(preflight.Config{ScenarioName:"api-app"}) { return }; _ = apiserver.Run(apiserver.Config{StartServer: func(*http.Server) error { return http.ListenAndServe(":8080", nil) }}) }`)
	report, err := New(Deps{RepoRoot: root}).ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.NotEqual(t, CodeServerRunnerMissing, finding.Code)
	}
}

func requireFinding(t *testing.T, report Report, code string) Finding {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code {
			return finding
		}
	}
	require.Failf(t, "finding not found", "missing finding code %s in %#v", code, report.Findings)
	return Finding{}
}

func TestValidateScenarioResolvesSameFileLifecycleAliasesAndInitialHelper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"net/http"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

var runApplication = startApplication
var preflightRun = preflight.Run
var runAPIServer = apiserver.Run

func main() { if err := runApplication(); err != nil { panic(err) } }

func startApplication() error {
	if preflightRun(preflight.Config{ScenarioName: "api-app"}) { return nil }
	return runAPIServer(apiserver.Config{Handler: http.NewServeMux()})
}
`)

	report, err := New(Deps{RepoRoot: root}).ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed, "%#v", report.Findings)
	require.Empty(t, report.Findings)
	require.True(t, report.Target.Lifecycle.PreflightHealthy)
	require.True(t, report.Target.Lifecycle.ServerRunnerHealthy)
}

func TestValidateScenarioStillRejectsPreflightAfterWorkInResolvedHelper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"fmt"
	"github.com/vrooli/api-core/preflight"
)

var runApplication = startApplication
var preflightRun = preflight.Run

func main() { if err := runApplication(); err != nil { panic(err) } }

func startApplication() error {
	fmt.Println("opening database")
	if preflightRun(preflight.Config{ScenarioName: "api-app"}) { return nil }
	return nil
}
`)

	report, err := New(Deps{RepoRoot: root}).ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	finding := requireFinding(t, report, CodePreflightMissingLate)
	require.Contains(t, finding.Message, "business logic executes before")
}

func TestValidateScenarioDoesNotAcceptUnrelatedFunctionAliases(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

var preflightRun = other.Run
var runAPIServer = other.Run

func main() {
	if preflightRun(preflight.Config{ScenarioName: "api-app"}) { return }
	_ = runAPIServer(nil)
}
`)

	report, err := New(Deps{RepoRoot: root}).ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodePreflightMissingLate)
	requireFinding(t, report, CodeServerRunnerMissing)
}

func TestValidateScenarioDoesNotCreditLocallyShadowedAliases(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

var preflightRun = preflight.Run
var runAPIServer = apiserver.Run

func main() { startApplication() }

func startApplication() {
	var preflightRun = other.Run
	if preflightRun(preflight.Config{ScenarioName: "api-app"}) { return }
	var runAPIServer = other.Run
	_ = runAPIServer(nil)
}
`)

	report, err := New(Deps{RepoRoot: root}).ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodePreflightMissingLate)
	requireFinding(t, report, CodeServerRunnerMissing)
}

func TestValidateScenarioDoesNotCreditServerAliasShadowedByParameter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), `package main

import (
	"net/http"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
)

var runAPIServer = apiserver.Run

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "api-app"}) { return }
	startApplication(func(apiserver.Config) error { return nil })
}

func startApplication(runAPIServer func(apiserver.Config) error) {
	_ = runAPIServer(apiserver.Config{Handler: http.NewServeMux()})
}
`)

	report, err := New(Deps{RepoRoot: root}).ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeServerRunnerMissing)
}
