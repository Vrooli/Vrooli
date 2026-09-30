//nolint:goconst // test data deliberately reuses stable package fixtures.
package packageapp

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/bootstrap"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	packagegov "github.com/vrooli/vrooli/internal/packagegov"
	testpackage "github.com/vrooli/vrooli/internal/packagegov/packagegovtest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func TestListInfoAndDependents(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testpackage.WritePackageManifest(t, fixture.Root, "alpha", testpackage.PackageManifest(
		"alpha",
		testpackage.WithPackageDocs("docs/package-governance.md"),
		testpackage.WithPackageRefresh(packagegov.RefreshScenarioSetup, true),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "package-governance.md"), "# ok\n")
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo"))
	testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, "demo", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/alpha": "file:../../../packages/alpha",
		},
	})

	svc := newIntegrationPackageService(fixture, false)

	packages, issues, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
	if len(packages) != 1 || packages[0].Name != "alpha" {
		t.Fatalf("packages = %#v", packages)
	}

	item, err := svc.Info("alpha")
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if item.Name != "alpha" {
		t.Fatalf("item = %#v", item)
	}

	owner, report, err := svc.Dependents("alpha")
	if err != nil {
		t.Fatalf("Dependents: %v", err)
	}
	if owner.Name != "alpha" {
		t.Fatalf("owner = %#v", owner)
	}
	if len(report.Dependents) != 1 || report.Dependents[0].ConsumerName != "demo" {
		t.Fatalf("dependents = %#v", report.Dependents)
	}
}

func TestDependentsReportsGoConsumerWhenGovernedReplaceIsMissing(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testpackage.WritePackageManifest(t, fixture.Root, "alpha", testpackage.PackageManifest(
		"alpha",
		testpackage.WithPackageDisplayName("example.com/alpha"),
		testpackage.WithPackageKind(packagegov.KindGoRuntime),
		testpackage.WithPackageModuleIdentifiers("example.com/alpha"),
		testpackage.WithPackageAllowedConsumers(packagegov.ConsumerScenarioCLI),
		testpackage.WithPackageAdoptionModes(packagegov.ModeGoModuleReplace),
	))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo"))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "cli", "go.mod"), `module example.com/demo/cli

go 1.25.0

require example.com/alpha v0.0.0
`)

	_, report, err := newIntegrationPackageService(fixture, false).Dependents("alpha")
	if err != nil {
		t.Fatalf("Dependents: %v", err)
	}
	if len(report.Dependents) != 1 || report.Dependents[0].ConsumerName != "demo" {
		t.Fatalf("dependents = %#v", report.Dependents)
	}
	if len(report.Issues) != 1 || report.Issues[0].Code != "package-go-module-replace-required" {
		t.Fatalf("issues = %#v", report.Issues)
	}
}

func TestRefreshScenarioSetupRunsBuildAndSetup(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "alpha", testpackage.PackageManifest(
		"alpha",
		testpackage.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testpackage.WithPackageRefresh(packagegov.RefreshScenarioSetup, false),
	))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo",
		testscenario.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Exec: []string{"bash", "-c", "mkdir -p build && printf setup > build/setup.txt"},
			}}},
		}),
	))
	testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, "demo", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/alpha": "file:../../../packages/alpha",
		},
	})

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "alpha", Target: "all"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "setup_only" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "packages", "alpha", "build", "build.txt")); err != nil {
		t.Fatal("expected package build marker")
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt")); err != nil {
		t.Fatal("expected scenario setup marker")
	}
}

func TestRefreshGenerateThenSetupRunsGenerateBuildAndSetup(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "proto", testpackage.PackageManifest(
		"proto",
		testpackage.WithPackageDisplayName("@vrooli/proto"),
		testpackage.WithPackageKind(packagegov.KindSchemaOrContract),
		testpackage.WithPackageModuleIdentifiers("@vrooli/proto-types"),
		testpackage.WithPackageGenerateCommands(commandSpec("generate", "mkdir -p build && printf generate > build/generate.txt")),
		testpackage.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testpackage.WithPackageRefresh(packagegov.RefreshGenerateThenSetup, false),
	))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo",
		testscenario.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Exec: []string{"bash", "-c", "mkdir -p build && printf setup > build/setup.txt"},
			}}},
		}),
	))
	testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, "demo", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/proto-types": "file:../../../packages/proto",
		},
	})

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "proto", Target: "all"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "setup_only" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	for _, path := range []string{
		filepath.Join(fixture.Root, "packages", "proto", "build", "generate.txt"),
		filepath.Join(fixture.Root, "packages", "proto", "build", "build.txt"),
		filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
}

func TestRefreshRebuildCLIConsumers(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "cli-core", testpackage.PackageManifest(
		"cli-core",
		testpackage.WithPackageDisplayName("example.com/cli-core"),
		testpackage.WithPackageKind(packagegov.KindGoCLI),
		testpackage.WithPackageModuleIdentifiers("example.com/cli-core"),
		testpackage.WithPackageAllowedConsumers(packagegov.ConsumerScenarioCLI),
		testpackage.WithPackageAdoptionModes(packagegov.ModeGoModuleReplace),
		testpackage.WithPackageRefresh(packagegov.RefreshRebuildCLI, false),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "go.mod"), "module example.com/cli-core\n\ngo 1.25.0\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "cli_core.go"), "package clicore\n\nfunc Name() string { return \"ok\" }\n")
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo", testscenario.WithLifecycle(scenario.Lifecycle{Version: "2.0.0"})))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "cli", "go.mod"), `module example.com/demo/cli

go 1.25.0

require example.com/cli-core v0.0.0

replace example.com/cli-core => ../../../packages/cli-core
`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "cli", "main.go"), `package main

import core "example.com/cli-core"

func main() {
	_ = core.Name()
}
`)

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "cli-core", Target: "all"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "rebuilt" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
}

func TestRefreshTargetFiltersAffectedScenario(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "alpha", testpackage.PackageManifest(
		"alpha",
		testpackage.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testpackage.WithPackageRefresh(packagegov.RefreshScenarioSetup, false),
	))
	for _, name := range []string{"alpha-ui", "beta-ui"} {
		testscenario.WriteScenarioService(t, fixture.Root, name, testscenario.ScenarioServiceManifest(name,
			testscenario.WithLifecycle(scenario.Lifecycle{
				Version: "2.0.0",
				Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
					Name: "capture-setup",
					Exec: []string{"bash", "-c", "mkdir -p build && printf setup > build/setup.txt"},
				}}},
			}),
		))
		testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, name, testpackage.NodePackageManifest{
			Dependencies: map[string]string{
				"@vrooli/alpha": "file:../../../packages/alpha",
			},
		})
	}

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "alpha", Target: "beta-ui"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Consumer != "beta-ui" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "alpha-ui", "build", "setup.txt")); !os.IsNotExist(err) {
		t.Fatalf("alpha-ui should not have been refreshed, err=%v", err)
	}
}

func TestRefreshArtifactDryRunUsesExactExportImpact(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "react-component-library", testpackage.PackageManifest(
		"react-component-library",
		testpackage.WithPackageDisplayName("@vrooli/react-component-library"),
		testpackage.WithPackageModuleIdentifiers("@vrooli/react-component-library"),
		testpackage.WithPackageRefresh(packagegov.RefreshScenarioSetup, false),
	))
	for _, name := range []string{"affected", "unaffected"} {
		testscenario.WriteScenarioService(t, fixture.Root, name, testscenario.ScenarioServiceManifest(name))
		testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, name, testpackage.NodePackageManifest{
			Dependencies: map[string]string{"@vrooli/react-component-library": "file:../../../packages/react-component-library"},
		})
	}
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "affected", "ui", "src", "App.tsx"), `import { Button } from "@vrooli/react-component-library/Button/1.0.0"; export const App = Button;`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "unaffected", "ui", "src", "App.tsx"), `import { Card } from "@vrooli/react-component-library/Card/1.0.0"; export const App = Card;`)

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{
		PackageName: "react-component-library", Target: "all", ChangedExports: []string{"./Button/1.0.0"}, DryRun: true,
	})
	if err != nil {
		t.Fatalf("Refresh dry run: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Consumer != "affected" || resp.Items[0].Status != "planned" {
		t.Fatalf("planned items = %#v", resp.Items)
	}
	if len(resp.Impacts) != 2 || resp.Impacts[0].Status != packagegov.ImpactAffected || resp.Impacts[1].Status != packagegov.ImpactUnaffected {
		t.Fatalf("impact report = %#v", resp.Impacts)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, "scenarios", "affected", "build")); !os.IsNotExist(err) {
		t.Fatalf("dry run mutated affected scenario, err=%v", err)
	}
}

func TestRefreshIncludesTemplateConsumersExplicitly(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "alpha", testpackage.PackageManifest(
		"alpha",
		testpackage.WithPackageAllowedConsumers(packagegov.ConsumerScenarioUI, packagegov.ConsumerTemplateUI),
		testpackage.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testpackage.WithPackageRefresh(packagegov.RefreshScenarioSetup, false),
	))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo",
		testscenario.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Exec: []string{"bash", "-c", "mkdir -p build && printf setup > build/setup.txt"},
			}}},
		}),
	))
	testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, "demo", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/alpha": "file:../../../packages/alpha",
		},
	})
	testpackage.WriteTemplateScenarioUIPackageManifest(t, fixture.Root, "react-vite", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/alpha": "file:../../../packages/alpha",
		},
	})

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "alpha", Target: "all"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	if resp.Items[0].Consumer != "demo" || resp.Items[0].Status != "setup_only" {
		t.Fatalf("resp.Items[0] = %#v", resp.Items[0])
	}
	if resp.Items[1].Consumer != "react-vite" || resp.Items[1].Status != "no_runtime_refresh" {
		t.Fatalf("resp.Items[1] = %#v", resp.Items[1])
	}
}

func TestRefreshRebuildsResourceConsumers(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "cli-core", testpackage.PackageManifest(
		"cli-core",
		testpackage.WithPackageDisplayName("example.com/cli-core"),
		testpackage.WithPackageKind(packagegov.KindGoCLI),
		testpackage.WithPackageModuleIdentifiers("example.com/cli-core"),
		testpackage.WithPackageAllowedConsumers(packagegov.ConsumerResourceRuntime),
		testpackage.WithPackageAdoptionModes(packagegov.ModeGoModuleReplace),
		testpackage.WithPackageRefresh(packagegov.RefreshRebuildCLI, false),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "go.mod"), "module example.com/cli-core\n\ngo 1.25.0\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "cli_core.go"), "package clicore\n\nfunc Name() string { return \"ok\" }\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "fixturecli", "go.mod"), `module example.com/resources/fixturecli

go 1.25.0

require example.com/cli-core v0.0.0

replace example.com/cli-core => ../../packages/cli-core
`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "fixturecli", "main.go"), `package fixturecli

import core "example.com/cli-core"

func Name() string {
	return core.Name()
}
`)

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "cli-core", Target: "all"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Consumer != "fixturecli" || resp.Items[0].Status != "rebuilt" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
}

func TestRefreshDedupesMultiSurfaceScenarioSetup(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "proto", testpackage.PackageManifest(
		"proto",
		testpackage.WithPackageDisplayName("@vrooli/proto"),
		testpackage.WithPackageKind(packagegov.KindSchemaOrContract),
		testpackage.WithPackageModuleIdentifiers("github.com/example/proto", "@vrooli/proto-types"),
		testpackage.WithPackageGeneratedOutputs(packagegov.GeneratedOutput{
			Name:        "proto-types",
			Identifiers: []string{"@vrooli/proto-types"},
			Consumers:   []packagegov.ConsumerClass{packagegov.ConsumerScenarioUI},
		}),
		testpackage.WithPackageAllowedConsumers(packagegov.ConsumerScenarioUI, packagegov.ConsumerScenarioAPI),
		testpackage.WithPackageAdoptionModes(packagegov.ModeGoModuleReplace, packagegov.ModeGeneratedArtifact),
		testpackage.WithPackageGenerateCommands(commandSpec("generate", "mkdir -p build && printf generate > build/generate.txt")),
		testpackage.WithPackageRefresh(packagegov.RefreshGenerateThenSetup, false),
	))
	testscenario.WriteScenarioService(t, fixture.Root, "desktop", testscenario.ScenarioServiceManifest("desktop",
		testscenario.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Exec: []string{"bash", "-c", "mkdir -p build && printf setup >> build/setup.txt"},
			}}},
		}),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "desktop", "api", "go.mod"), `module example.com/desktop/api

go 1.25.0

require github.com/example/proto v0.0.0

replace github.com/example/proto => ../../../packages/proto
`)
	testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, "desktop", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/proto-types": "file:../../../packages/proto/gen/typescript",
		},
	})

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "proto", Target: "desktop"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	if len(resp.Items[0].Classes) != 2 {
		t.Fatalf("resp.Items[0] = %#v", resp.Items[0])
	}
	data, err := os.ReadFile(filepath.Join(fixture.Root, "scenarios", "desktop", "build", "setup.txt"))
	if err != nil {
		t.Fatalf("read setup marker: %v", err)
	}
	if strings.Count(string(data), "setup") != 1 {
		t.Fatalf("setup marker = %q", string(data))
	}
}

func TestRefreshRestartsRunningScenario(t *testing.T) {
	t.Setenv("VROOLI_RUNTIME_SUPERVISOR", "off")
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectService(t, fixture.Root, testscenario.ProjectServiceManifest())
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "alpha", testpackage.PackageManifest(
		"alpha",
		testpackage.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testpackage.WithPackageRefresh(packagegov.RefreshScenarioSetup, true),
	))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo",
		testscenario.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Exec: []string{"bash", "-c", "mkdir -p build && printf setup >> build/setup.txt"},
			}}},
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name:       "stay-running",
				Exec:       []string{"sleep", "30"},
				Background: true,
			}}},
		}),
	))
	testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, "demo", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/alpha": "file:../../../packages/alpha",
		},
	})

	svc := newIntegrationPackageService(fixture, false)
	services := bootstrap.New(fixture.Root, fixture.Home, &bytes.Buffer{}, &bytes.Buffer{}, nil)
	scenarios := services.Orchestrator()
	if _, err := scenarios.StartDetailed("demo", lifecycle.StartOptions{ForceSetup: true}); err != nil {
		t.Fatalf("StartDetailed: %v", err)
	}
	t.Cleanup(func() {
		runner, runErr := services.LifecycleRunner()
		if runErr == nil {
			_ = runner.Stop("demo", lifecycle.StopOptions{})
		}
	})

	before, _, err := scenarios.Lookup("demo")
	if err != nil || len(before.Runtime.Records) != 1 {
		t.Fatalf("initial runtime = %#v, err = %v", before.Runtime, err)
	}
	runtime := &refreshLookupBarrier{ScenarioRuntime: scenarios}
	svc.ScenarioService = func() (ScenarioRuntime, error) { return runtime, nil }
	runner, err := services.LifecycleRunner()
	if err != nil {
		t.Fatal(err)
	}
	recording := &refreshRecordingRunner{ScenarioPhaseRunner: runner}
	svc.ScenarioRunner = func() (ScenarioPhaseRunner, error) { return recording, nil }
	resp, err := svc.Refresh(RefreshRequest{PackageName: "alpha", Target: "demo", Interactive: true})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "restarted" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	detail, _, err := scenarios.Lookup("demo")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if detail.Runtime.ProcessCount != 1 || len(detail.Runtime.Records) != 1 || detail.Runtime.Records[0].PID == before.Runtime.Records[0].PID {
		t.Fatalf("expected replacement process: before=%#v after=%#v", before.Runtime, detail.Runtime)
	}
	data, err := os.ReadFile(filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt"))
	if err != nil || string(data) != "setupsetup" {
		t.Fatalf("setup count = %q, err = %v", data, err)
	}
	if recording.stops != 1 || recording.admissions != 1 || runtime.starts != 1 {
		t.Fatalf("restart calls: stop=%d setup=%d start=%d", recording.stops, recording.admissions, runtime.starts)
	}
}

func TestRefreshNoRestartPreservesRunningConsumer(t *testing.T) {
	for _, tc := range []struct {
		name           string
		interactive    bool
		noRestart      bool
		packageRestart bool
		startAtLookup  bool
	}{
		{name: "noninteractive", packageRestart: true},
		{name: "explicit_no_restart", interactive: true, noRestart: true, packageRestart: true},
		{name: "package_prohibits_restart", interactive: true},
		{name: "start_after_stopped_lookup", packageRestart: true, startAtLookup: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testRefreshPreservesConsumer(t, tc.interactive, tc.noRestart, tc.packageRestart, tc.startAtLookup)
		})
	}
}

func testRefreshPreservesConsumer(t *testing.T, interactive, noRestart, packageRestart, startAtLookup bool) {
	t.Helper()
	t.Setenv("VROOLI_RUNTIME_SUPERVISOR", "off")
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectService(t, fixture.Root, testscenario.ProjectServiceManifest())
	testresource.WritePortRegistry(t, fixture.Root, nil)
	testpackage.WritePackageManifest(t, fixture.Root, "alpha", testpackage.PackageManifest(
		"alpha",
		testpackage.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testpackage.WithPackageRefresh(packagegov.RefreshScenarioSetup, packageRestart),
	))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo",
		testscenario.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Exec: []string{"bash", "-c", "mkdir -p build && printf setup >> build/setup.txt"},
			}}},
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name:       "stay-running",
				Exec:       []string{"sleep", "30"},
				Background: true,
			}}},
		}),
	))
	testpackage.WriteScenarioUIPackageManifest(t, fixture.Root, "demo", testpackage.NodePackageManifest{
		Dependencies: map[string]string{
			"@vrooli/alpha": "file:../../../packages/alpha",
		},
	})

	svc := newIntegrationPackageService(fixture, false)
	services := bootstrap.New(fixture.Root, fixture.Home, &bytes.Buffer{}, &bytes.Buffer{}, nil)
	scenarios := services.Orchestrator()
	t.Cleanup(func() {
		runner, runErr := services.LifecycleRunner()
		if runErr == nil {
			_ = runner.Stop("demo", lifecycle.StopOptions{})
		}
	})

	var before orchestrator.Detail
	var setupBefore []byte
	start := func() {
		if _, err := scenarios.StartDetailed("demo", lifecycle.StartOptions{ForceSetup: true}); err != nil {
			t.Fatalf("StartDetailed: %v", err)
		}
		var err error
		before, _, err = scenarios.Lookup("demo")
		if err != nil || before.Runtime.ProcessCount != 1 || len(before.Runtime.Records) != 1 {
			t.Fatalf("initial runtime = %#v, err = %v", before.Runtime, err)
		}
		setupBefore, err = os.ReadFile(filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt"))
		if err != nil || string(setupBefore) != "setup" {
			t.Fatalf("initial setup = %q, err = %v", setupBefore, err)
		}
	}
	if !startAtLookup {
		start()
	}
	runtime := &refreshLookupBarrier{ScenarioRuntime: scenarios}
	if startAtLookup {
		// Complete a fixture start after observing stopped state, before
		// refresh receives that observation. No scheduler timing is involved.
		runtime.afterLookup = func(detail orchestrator.Detail) {
			if detail.Runtime.ProcessCount != 0 {
				t.Fatalf("expected stopped lookup, got %#v", detail.Runtime)
			}
			start()
		}
	}
	svc.ScenarioService = func() (ScenarioRuntime, error) { return runtime, nil }
	runner, err := services.LifecycleRunner()
	if err != nil {
		t.Fatal(err)
	}
	recording := &refreshRecordingRunner{ScenarioPhaseRunner: runner}
	svc.ScenarioRunner = func() (ScenarioPhaseRunner, error) { return recording, nil }
	resp, err := svc.Refresh(RefreshRequest{PackageName: "alpha", Target: "demo", NoRestart: noRestart, Interactive: interactive})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "running_setup_deferred" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	detail, _, err := scenarios.Lookup("demo")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if detail.Runtime.ProcessCount != before.Runtime.ProcessCount || !reflect.DeepEqual(detail.Runtime.Records, before.Runtime.Records) {
		t.Fatalf("runtime identity changed: before=%#v after=%#v", before.Runtime, detail.Runtime)
	}
	setupAfter, err := os.ReadFile(filepath.Join(fixture.Root, "scenarios", "demo", "build", "setup.txt"))
	if err != nil || !bytes.Equal(setupAfter, setupBefore) {
		t.Fatalf("setup repeated: before=%q after=%q err=%v", setupBefore, setupAfter, err)
	}
	if recording.stops != 0 || recording.phases != 0 || runtime.starts != 0 || runtime.lookups != 1 {
		t.Fatalf("refresh calls: stop=%d phase=%d start=%d lookup=%d", recording.stops, recording.phases, runtime.starts, runtime.lookups)
	}
	wantAdmissions := 0
	if startAtLookup {
		wantAdmissions = 1
	}
	if recording.admissions != wantAdmissions {
		t.Fatalf("stopped-only admission calls = %d, want %d", recording.admissions, wantAdmissions)
	}
}

type refreshLookupBarrier struct {
	ScenarioRuntime
	afterLookup func(orchestrator.Detail)
	starts      int
	lookups     int
}

func (f *fakeScenarioRunner) RunSetupIfStopped(name string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	return f.RunPhaseDetailed(name, "setup", opts)
}

func (r *refreshLookupBarrier) Lookup(name string) (orchestrator.Detail, bool, error) {
	r.lookups++
	detail, found, err := r.ScenarioRuntime.Lookup(name)
	if err == nil && r.afterLookup != nil {
		barrier := r.afterLookup
		r.afterLookup = nil
		barrier(detail)
	}
	return detail, found, err
}

func (r *refreshLookupBarrier) StartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	r.starts++
	return r.ScenarioRuntime.StartDetailed(name, opts)
}

type refreshRecordingRunner struct {
	ScenarioPhaseRunner
	stops      int
	phases     int
	admissions int
}

func (r *refreshRecordingRunner) RunSetupIfStopped(name string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	r.admissions++
	return r.ScenarioPhaseRunner.RunSetupIfStopped(name, opts)
}

func (r *refreshRecordingRunner) Stop(name string, opts lifecycle.StopOptions) error {
	r.stops++
	return r.ScenarioPhaseRunner.Stop(name, opts)
}

func (r *refreshRecordingRunner) RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	r.phases++
	return r.ScenarioPhaseRunner.RunPhaseDetailed(name, phase, opts)
}

func newIntegrationPackageService(fixture testkitgo.RepoFixture, json bool) Service {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	services := bootstrap.New(fixture.Root, fixture.Home, &stdout, &stderr, nil)
	commandStdout := &stdout
	if json {
		commandStdout = &stderr
	}
	return Service{
		Root:   fixture.Root,
		Stdout: commandStdout,
		Stderr: &stderr,
		ScenarioService: func() (ScenarioRuntime, error) {
			return services.Orchestrator(), nil
		},
		ScenarioRunner: func() (ScenarioPhaseRunner, error) {
			return services.LifecycleRunner()
		},
	}
}

func commandSpec(name, shellCommand string) packagegov.CommandSpec {
	return packagegov.CommandSpec{
		Name:    name,
		Run:     []string{"bash", "-lc", shellCommand},
		Outputs: []string{"build/**"},
	}
}
