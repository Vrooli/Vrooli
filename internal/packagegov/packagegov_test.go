package packagegov

import (
	"os"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func TestLoadAllReportsMissingManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	packagesDir := filepath.Join(fixture.Root, "packages")
	if err := os.MkdirAll(filepath.Join(packagesDir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}

	items, issues, err := LoadAll(fixture.Root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
	if len(issues) != 1 || issues[0].Code != "missing-package-manifest" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateFlagsWorkspaceDepsAndPostinstallDebt(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "alpha", packageManifestFixture("alpha", func(manifest *Manifest) {
		manifest.Package.Refresh = RefreshPolicy{
			Strategy:                RefreshScenarioSetup,
			RestartRunningConsumers: true,
		}
	}))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo"))
	writeScenarioUIPackageManifestFixture(t, fixture.Root, "demo", map[string]string{"@vrooli/alpha": "workspace:*"}, map[string]string{
		"postinstall": "mkdir -p node_modules/@vrooli && cp -a ../../../packages/alpha node_modules/@vrooli/alpha",
	})

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(report.Issues) < 2 {
		t.Fatalf("issues = %#v", report.Issues)
	}
}

func TestValidateFlagsScenarioAdoptionOfInternalOnlyPackage(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteJSON(t, filepath.Join(fixture.Root, "packages", "testkit-go", ".vrooli", "package.json"), map[string]any{
		"$schema": "schemas/package.schema.json",
		"version": "1.0.0",
		"package": map[string]any{
			"name":               "testkit-go",
			"display_name":       "github.com/example/testkit-go",
			"kind":               "go_testkit",
			"module_identifiers": []string{"github.com/example/testkit-go"},
			"adoption": map[string]any{
				"scenario_adoptable": false,
				"allowed_consumers":  []string{"internal_platform"},
				"adoption_modes":     []string{},
			},
			"lifecycle": map[string]any{},
			"refresh": map[string]any{
				"strategy":                  "none",
				"restart_running_consumers": false,
			},
		},
	})
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo"))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "api", "go.mod"), `module example.com/demo/api

go 1.25.0

require github.com/example/testkit-go v0.0.0

replace github.com/example/testkit-go => ../../../packages/testkit-go
`)

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Code == "package-adoption-supported" && issue.PackageName == "testkit-go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected package-adoption-supported issue, got %#v", report.Issues)
	}
}

func TestLoadAllRejectsUnknownManifestFields(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteJSON(t, filepath.Join(fixture.Root, "packages", "alpha", ".vrooli", "package.json"), map[string]any{
		"$schema": "schemas/package.schema.json",
		"version": "1.0.0",
		"package": map[string]any{
			"name":               "alpha",
			"display_name":       "@vrooli/alpha",
			"kind":               "js_runtime",
			"module_identifiers": []string{"@vrooli/alpha"},
			"unexpected":         true,
			"adoption": map[string]any{
				"scenario_adoptable": true,
				"allowed_consumers":  []string{"scenario_ui"},
				"adoption_modes":     []string{"file_dependency"},
			},
			"lifecycle": map[string]any{},
			"refresh": map[string]any{
				"strategy":                  "scenario_setup",
				"restart_running_consumers": true,
			},
		},
	})

	_, issues, err := LoadAll(fixture.Root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(issues) != 1 || issues[0].Code != "invalid-package-manifest" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestLoadAllRejectsSchemaViolations(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "alpha", packageManifestFixture("alpha", func(manifest *Manifest) {
		manifest.Package.DisplayName = ""
	}))

	_, issues, err := LoadAll(fixture.Root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(issues) != 1 || issues[0].Code != "invalid-package-manifest" {
		t.Fatalf("issues = %#v", issues)
	}
	if issues[0].Message == "" || issues[0].Message == "schema validation failed" {
		t.Fatalf("expected specific schema validation error, got %#v", issues)
	}
}

func TestDiscoverDependentsClassifiesGeneratedArtifacts(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "proto", packageManifestFixture("proto", func(manifest *Manifest) {
		manifest.Package.DisplayName = "github.com/example/proto"
		manifest.Package.Kind = KindSchemaOrContract
		manifest.Package.ModuleIdentifiers = []string{"github.com/example/proto"}
		manifest.Package.GeneratedOutputs = []GeneratedOutput{{
			Name:        "proto-types",
			Identifiers: []string{"@vrooli/proto-types"},
			Consumers:   []ConsumerClass{ConsumerScenarioUI},
		}}
		manifest.Package.Adoption.AllowedConsumers = []ConsumerClass{ConsumerScenarioUI}
		manifest.Package.Adoption.AdoptionModes = []AdoptionMode{ModeGeneratedArtifact}
		manifest.Package.Refresh = RefreshPolicy{
			Strategy:                RefreshGenerateThenSetup,
			RestartRunningConsumers: true,
		}
	}))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo"))
	writeScenarioUIPackageManifestFixture(t, fixture.Root, "demo", map[string]string{"@vrooli/proto-types": "file:../../../packages/proto/gen/typescript"}, nil)

	items, issues, err := LoadAll(fixture.Root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("unexpected load issues: %#v", issues)
	}
	item, ok := FindByName(items, "proto")
	if !ok {
		t.Fatal("proto package not loaded")
	}
	report, err := DiscoverDependents(fixture.Root, item)
	if err != nil {
		t.Fatalf("DiscoverDependents: %v", err)
	}
	if len(report.Dependents) != 1 {
		t.Fatalf("dependents = %#v", report.Dependents)
	}
	if report.Dependents[0].AdoptionMode != ModeGeneratedArtifact {
		t.Fatalf("adoption mode = %q, want %q", report.Dependents[0].AdoptionMode, ModeGeneratedArtifact)
	}
}

func TestValidateAllowsResourceRuntimeConsumers(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "cli-core", packageManifestFixture("cli-core", func(manifest *Manifest) {
		manifest.Package.DisplayName = "github.com/example/cli-core"
		manifest.Package.Kind = KindGoCLI
		manifest.Package.ModuleIdentifiers = []string{"github.com/example/cli-core"}
		manifest.Package.Adoption.AllowedConsumers = []ConsumerClass{ConsumerResourceRuntime}
		manifest.Package.Adoption.AdoptionModes = []AdoptionMode{ModeGoModuleReplace}
		manifest.Package.Refresh = RefreshPolicy{Strategy: RefreshRebuildCLI}
	}))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "sqlite", "go.mod"), `module github.com/example/resources/sqlite

go 1.25.0

require github.com/example/cli-core v0.0.0

replace github.com/example/cli-core => ../../packages/cli-core
`)

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for _, issue := range report.Issues {
		if issue.PackageName == "cli-core" {
			t.Fatalf("unexpected issue for resource consumer: %#v", report.Issues)
		}
	}
}

func TestValidateFlagsLeafGoPackagesWithLocalDependencies(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "cli-core", packageManifestFixture("cli-core", func(manifest *Manifest) {
		manifest.Package.DisplayName = "github.com/example/cli-core"
		manifest.Package.Kind = KindGoCLI
		manifest.Package.ModuleIdentifiers = []string{"github.com/example/cli-core"}
		manifest.Package.Adoption.AllowedConsumers = []ConsumerClass{ConsumerScenarioCLI}
		manifest.Package.Adoption.AdoptionModes = []AdoptionMode{ModeGoModuleReplace}
		manifest.Package.Refresh = RefreshPolicy{Strategy: RefreshRebuildCLI}
	}))
	writePackageManifestFixture(t, fixture.Root, "api-core", packageManifestFixture("api-core", func(manifest *Manifest) {
		manifest.Package.DisplayName = "github.com/example/api-core"
		manifest.Package.Kind = KindGoRuntime
		manifest.Package.ModuleIdentifiers = []string{"github.com/example/api-core"}
		manifest.Package.Adoption.AllowedConsumers = []ConsumerClass{ConsumerScenarioAPI}
		manifest.Package.Adoption.AdoptionModes = []AdoptionMode{ModeGoModuleReplace}
		manifest.Package.Refresh = RefreshPolicy{Strategy: RefreshRestartConsumers}
	}))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "go.mod"), `module github.com/example/cli-core

go 1.25.0

require github.com/example/api-core v0.0.0

replace github.com/example/api-core => ../api-core
`)

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.PackageName == "cli-core" && issue.Code == "package-go-leaf-local-dependency" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected package-go-leaf-local-dependency issue, got %#v", report.Issues)
	}
}

func TestValidateRequiresGoReplaceForGovernedAdoption(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "package-governance.md"), "# ok\n")
	writePackageManifestFixture(t, fixture.Root, "alpha", packageManifestFixture("alpha", func(manifest *Manifest) {
		manifest.Package.DisplayName = "github.com/example/alpha"
		manifest.Package.Kind = KindGoRuntime
		manifest.Package.ModuleIdentifiers = []string{"github.com/example/alpha"}
		manifest.Package.Adoption.AllowedConsumers = []ConsumerClass{ConsumerScenarioAPI}
		manifest.Package.Adoption.AdoptionModes = []AdoptionMode{ModeGoModuleReplace}
		manifest.Package.Refresh = RefreshPolicy{
			Strategy:                RefreshRestartConsumers,
			RestartRunningConsumers: true,
		}
		manifest.Package.Docs = []string{"docs/package-governance.md"}
	}))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo"))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "api", "go.mod"), `module example.com/demo/api

go 1.25.0

require github.com/example/alpha v0.0.0
`)

	report, err := Validate(fixture.Root, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.PackageName == "alpha" && issue.Code == "package-go-module-replace-required" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected package-go-module-replace-required issue, got %#v", report.Issues)
	}
}

func TestAuditFlagsWorkspaceAndGoWorkGuidanceDrift(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "bad-package-guidance.md"), `# Bad Guidance

This package is linked via the pnpm workspace.

You can also use go.work to point local consumers at the package.
`)

	report, err := Audit(fixture.Root, "")
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}

	var foundWorkspace bool
	var foundGoWork bool
	for _, issue := range report.Issues {
		switch issue.Code {
		case "pnpm-workspace-guidance":
			foundWorkspace = true
		case "go-work-guidance":
			foundGoWork = true
		}
	}
	if !foundWorkspace || !foundGoWork {
		t.Fatalf("expected workspace and go.work guidance issues, got %#v", report.Issues)
	}
}

func packageManifestFixture(name string, opts ...func(*Manifest)) Manifest {
	manifest := Manifest{
		Schema:  "schemas/package.schema.json",
		Version: "1.0.0",
		Package: ManifestEntry{
			Name:              name,
			DisplayName:       "@vrooli/" + name,
			Kind:              KindJSRuntime,
			ModuleIdentifiers: []string{"@vrooli/" + name},
			Adoption: AdoptionPolicy{
				ScenarioAdoptable: true,
				AllowedConsumers:  []ConsumerClass{ConsumerScenarioUI},
				AdoptionModes:     []AdoptionMode{ModeFileDependency},
			},
			Lifecycle: LifecyclePolicy{},
			Refresh: RefreshPolicy{
				Strategy:                RefreshScenarioSetup,
				RestartRunningConsumers: true,
			},
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func writePackageManifestFixture(t *testing.T, root, name string, manifest Manifest) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(root, "packages", name, ".vrooli", "package.json"), manifest)
}

func writeScenarioUIPackageManifestFixture(t *testing.T, root, scenarioName string, dependencies, scripts map[string]string) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(root, "scenarios", scenarioName, "ui", "package.json"), map[string]any{
		"name":         scenarioName + "-ui",
		"private":      true,
		"dependencies": dependencies,
		"scripts":      scripts,
	})
}
