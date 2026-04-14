package packageapp

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/bootstrap"
	"github.com/vrooli/vrooli/internal/lifecycle"
	packagegov "github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testkitvrooli "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

func TestListInfoDependentsValidateAndAudit(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "alpha", testkitvrooli.PackageManifest(
		"alpha",
		testkitvrooli.WithPackageDocs("docs/package-governance.md"),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshScenarioSetup, true),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "package-governance.md"), "# ok\n")
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo"))
	writeScenarioUIDependenciesFixture(t, fixture.Root, "demo", map[string]string{
		"@vrooli/alpha": "file:../../../packages/alpha",
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

	validateReport, err := svc.Validate("")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(validateReport.Issues) != 0 {
		t.Fatalf("validate report = %#v", validateReport)
	}

	auditReport, err := svc.Audit("")
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if len(auditReport.Issues) != 0 {
		t.Fatalf("audit report = %#v", auditReport)
	}
}

func TestRefreshScenarioSetupRunsBuildAndSetup(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "alpha", testkitvrooli.PackageManifest(
		"alpha",
		testkitvrooli.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshScenarioSetup, false),
	))
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup > build/setup.txt",
			}}},
		}),
	))
	writeScenarioUIDependenciesFixture(t, fixture.Root, "demo", map[string]string{
		"@vrooli/alpha": "file:../../../packages/alpha",
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
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "proto", testkitvrooli.PackageManifest(
		"proto",
		testkitvrooli.WithPackageDisplayName("@vrooli/proto"),
		testkitvrooli.WithPackageKind(packagegov.KindSchemaOrContract),
		testkitvrooli.WithPackageModuleIdentifiers("@vrooli/proto-types"),
		testkitvrooli.WithPackageGenerateCommands(commandSpec("generate", "mkdir -p build && printf generate > build/generate.txt")),
		testkitvrooli.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshGenerateThenSetup, false),
	))
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup > build/setup.txt",
			}}},
		}),
	))
	writeScenarioUIDependenciesFixture(t, fixture.Root, "demo", map[string]string{
		"@vrooli/proto-types": "file:../../../packages/proto",
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
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "cli-core", testkitvrooli.PackageManifest(
		"cli-core",
		testkitvrooli.WithPackageDisplayName("example.com/cli-core"),
		testkitvrooli.WithPackageKind(packagegov.KindGoCLI),
		testkitvrooli.WithPackageModuleIdentifiers("example.com/cli-core"),
		testkitvrooli.WithPackageAllowedConsumers(packagegov.ConsumerScenarioCLI),
		testkitvrooli.WithPackageAdoptionModes(packagegov.ModeGoModuleReplace),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshRebuildCLI, false),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "go.mod"), "module example.com/cli-core\n\ngo 1.25.0\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "cli_core.go"), "package clicore\n\nfunc Name() string { return \"ok\" }\n")
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo", testkitvrooli.WithLifecycle(scenario.Lifecycle{Version: "2.0.0"})))
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
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "alpha", testkitvrooli.PackageManifest(
		"alpha",
		testkitvrooli.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshScenarioSetup, false),
	))
	for _, name := range []string{"alpha-ui", "beta-ui"} {
		testkitvrooli.WriteScenarioService(t, fixture.Root, name, testkitvrooli.ScenarioServiceManifest(name,
			testkitvrooli.WithLifecycle(scenario.Lifecycle{
				Version: "2.0.0",
				Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
					Name: "capture-setup",
					Run:  "mkdir -p build && printf setup > build/setup.txt",
				}}},
			}),
		))
		writeScenarioUIDependenciesFixture(t, fixture.Root, name, map[string]string{
			"@vrooli/alpha": "file:../../../packages/alpha",
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

func TestRefreshIncludesTemplateConsumersExplicitly(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "alpha", testkitvrooli.PackageManifest(
		"alpha",
		testkitvrooli.WithPackageAllowedConsumers(packagegov.ConsumerScenarioUI, packagegov.ConsumerTemplateUI),
		testkitvrooli.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshScenarioSetup, false),
	))
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup > build/setup.txt",
			}}},
		}),
	))
	writeScenarioUIDependenciesFixture(t, fixture.Root, "demo", map[string]string{
		"@vrooli/alpha": "file:../../../packages/alpha",
	})
	testkitvrooli.WriteTemplateScenarioUIPackageManifest(t, fixture.Root, "react-vite", testkitvrooli.NodePackageManifest{
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
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "cli-core", testkitvrooli.PackageManifest(
		"cli-core",
		testkitvrooli.WithPackageDisplayName("example.com/cli-core"),
		testkitvrooli.WithPackageKind(packagegov.KindGoCLI),
		testkitvrooli.WithPackageModuleIdentifiers("example.com/cli-core"),
		testkitvrooli.WithPackageAllowedConsumers(packagegov.ConsumerResourceRuntime),
		testkitvrooli.WithPackageAdoptionModes(packagegov.ModeGoModuleReplace),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshRebuildCLI, false),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "go.mod"), "module example.com/cli-core\n\ngo 1.25.0\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "cli_core.go"), "package clicore\n\nfunc Name() string { return \"ok\" }\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "sqlite", "go.mod"), `module example.com/resources/sqlite

go 1.25.0

require example.com/cli-core v0.0.0

replace example.com/cli-core => ../../packages/cli-core
`)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "sqlite", "main.go"), `package sqlite

import core "example.com/cli-core"

func Name() string {
	return core.Name()
}
`)

	resp, err := newIntegrationPackageService(fixture, false).Refresh(RefreshRequest{PackageName: "cli-core", Target: "all"})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Consumer != "sqlite" || resp.Items[0].Status != "rebuilt" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
}

func TestRefreshDedupesMultiSurfaceScenarioSetup(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "proto", testkitvrooli.PackageManifest(
		"proto",
		testkitvrooli.WithPackageDisplayName("@vrooli/proto"),
		testkitvrooli.WithPackageKind(packagegov.KindSchemaOrContract),
		testkitvrooli.WithPackageModuleIdentifiers("github.com/example/proto", "@vrooli/proto-types"),
		testkitvrooli.WithPackageGeneratedOutputs(packagegov.GeneratedOutput{
			Name:        "proto-types",
			Identifiers: []string{"@vrooli/proto-types"},
			Consumers:   []packagegov.ConsumerClass{packagegov.ConsumerScenarioUI},
		}),
		testkitvrooli.WithPackageAllowedConsumers(packagegov.ConsumerScenarioUI, packagegov.ConsumerScenarioAPI),
		testkitvrooli.WithPackageAdoptionModes(packagegov.ModeGoModuleReplace, packagegov.ModeGeneratedArtifact),
		testkitvrooli.WithPackageGenerateCommands(commandSpec("generate", "mkdir -p build && printf generate > build/generate.txt")),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshGenerateThenSetup, false),
	))
	testkitvrooli.WriteScenarioService(t, fixture.Root, "desktop", testkitvrooli.ScenarioServiceManifest("desktop",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup >> build/setup.txt",
			}}},
		}),
	))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "desktop", "api", "go.mod"), `module example.com/desktop/api

go 1.25.0

require github.com/example/proto v0.0.0

replace github.com/example/proto => ../../../packages/proto
`)
	writeScenarioUIDependenciesFixture(t, fixture.Root, "desktop", map[string]string{
		"@vrooli/proto-types": "file:../../../packages/proto/gen/typescript",
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
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "alpha", testkitvrooli.PackageManifest(
		"alpha",
		testkitvrooli.WithPackageBuildCommands(commandSpec("build", "mkdir -p build && printf build > build/build.txt")),
		testkitvrooli.WithPackageRefresh(packagegov.RefreshScenarioSetup, true),
	))
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup > build/setup.txt",
			}}},
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name:       "stay-running",
				Run:        "sleep 30",
				Background: true,
			}}},
		}),
	))
	writeScenarioUIDependenciesFixture(t, fixture.Root, "demo", map[string]string{
		"@vrooli/alpha": "file:../../../packages/alpha",
	})

	svc := newIntegrationPackageService(fixture, false)
	services := bootstrap.New(fixture.Root, fixture.Home, &bytes.Buffer{}, &bytes.Buffer{}, nil)
	scenarios := services.Orchestrator()
	if _, err := scenarios.StartDetailed("demo", lifecycle.StartOptions{}); err != nil {
		t.Fatalf("StartDetailed: %v", err)
	}
	t.Cleanup(func() {
		runner, runErr := services.LifecycleRunner()
		if runErr == nil {
			_ = runner.Stop("demo", lifecycle.StopOptions{})
		}
	})

	resp, err := svc.Refresh(RefreshRequest{PackageName: "alpha", Target: "demo"})
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
	if detail.Runtime.ProcessCount == 0 {
		t.Fatalf("detail.Runtime = %#v", detail.Runtime)
	}
}

func TestRefreshNoRestartLeavesScenarioStopped(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitvrooli.WriteScenarioPortRegistryFixture(t, fixture.Root)
	writePackageManifestFixture(t, fixture.Root, "alpha", testkitvrooli.PackageManifest(
		"alpha",
		testkitvrooli.WithPackageRefresh(packagegov.RefreshScenarioSetup, true),
	))
	testkitvrooli.WriteScenarioService(t, fixture.Root, "demo", testkitvrooli.ScenarioServiceManifest("demo",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name: "capture-setup",
				Run:  "mkdir -p build && printf setup > build/setup.txt",
			}}},
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name:       "stay-running",
				Run:        "sleep 30",
				Background: true,
			}}},
		}),
	))
	writeScenarioUIDependenciesFixture(t, fixture.Root, "demo", map[string]string{
		"@vrooli/alpha": "file:../../../packages/alpha",
	})

	svc := newIntegrationPackageService(fixture, false)
	services := bootstrap.New(fixture.Root, fixture.Home, &bytes.Buffer{}, &bytes.Buffer{}, nil)
	scenarios := services.Orchestrator()
	if _, err := scenarios.StartDetailed("demo", lifecycle.StartOptions{}); err != nil {
		t.Fatalf("StartDetailed: %v", err)
	}
	t.Cleanup(func() {
		runner, runErr := services.LifecycleRunner()
		if runErr == nil {
			_ = runner.Stop("demo", lifecycle.StopOptions{})
		}
	})

	resp, err := svc.Refresh(RefreshRequest{PackageName: "alpha", Target: "demo", NoRestart: true})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Status != "stopped_after_setup" {
		t.Fatalf("resp.Items = %#v", resp.Items)
	}
	detail, _, err := scenarios.Lookup("demo")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if detail.Runtime.ProcessCount != 0 {
		t.Fatalf("detail.Runtime = %#v", detail.Runtime)
	}
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

func writePackageManifestFixture(t *testing.T, root, name string, manifest packagegov.Manifest) {
	t.Helper()
	testkitvrooli.WritePackageManifest(t, root, name, manifest)
}

func writeScenarioUIDependenciesFixture(t *testing.T, root, name string, dependencies map[string]string) {
	t.Helper()
	testkitvrooli.WriteScenarioUIPackageManifest(t, root, name, testkitvrooli.NodePackageManifest{
		Dependencies: dependencies,
	})
}

func commandSpec(name, shellCommand string) packagegov.CommandSpec {
	return packagegov.CommandSpec{
		Name: name,
		Run:  []string{"bash", "-lc", shellCommand},
	}
}
