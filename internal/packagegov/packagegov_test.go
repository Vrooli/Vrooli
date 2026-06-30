package packagegov

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "resources", "fixturecli", "go.mod"), `module github.com/example/resources/fixturecli

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

// TestValidateFlagsLeafGoLocalDependencyUnderSinglePackageFilter mirrors the
// cli-core -> proto regression: a governed leaf requiring a non-allowlisted
// governed module, declared in a SECOND require block, must be flagged even when
// validation is scoped to just the leaf package. Before the fix the single-
// package filter reduced the discovered set so the governed-module map no longer
// contained proto, and the forbidden dependency went unreported.
func TestValidateFlagsLeafGoLocalDependencyUnderSinglePackageFilter(t *testing.T) {
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
	writePackageManifestFixture(t, fixture.Root, "proto", packageManifestFixture("proto", func(manifest *Manifest) {
		manifest.Package.DisplayName = "github.com/example/proto"
		manifest.Package.Kind = KindSchemaOrContract
		manifest.Package.ModuleIdentifiers = []string{"github.com/example/proto"}
		manifest.Package.Adoption.AllowedConsumers = []ConsumerClass{ConsumerScenarioCLI}
		manifest.Package.Adoption.AdoptionModes = []AdoptionMode{ModeGoModuleReplace}
		manifest.Package.Refresh = RefreshPolicy{Strategy: RefreshRestartConsumers}
	}))
	// proto declared in a SECOND require block alongside an indirect dep, exactly
	// like the real cli-core/go.mod that slipped through.
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "cli-core", "go.mod"), `module github.com/example/cli-core

go 1.25.0

require github.com/example/repo-contract-go v0.0.0

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/example/proto v0.0.0
)

replace github.com/example/repo-contract-go => ../repo-contract-go

replace github.com/example/proto => ../proto
`)

	report, err := Validate(fixture.Root, "cli-core")
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
		t.Fatalf("expected package-go-leaf-local-dependency issue under single-package filter, got %#v", report.Issues)
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

func TestAuditSkipsRuntimeDataAndElectronOutputsBeforeOpen(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	writePackageManifestFixture(t, fixture.Root, "alpha", packageManifestFixture("alpha"))
	testscenario.WriteScenarioService(t, fixture.Root, "demo", testscenario.ScenarioServiceManifest("demo"))
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "docs", "package-governance.md"), "# package governance\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "README.md"), "# demo\n")
	mustSparseFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "data", "demo.db"), 8<<30)
	mustSparseFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "data", "demo.db-wal"), 1<<20)
	mustSparseFile(t, filepath.Join(fixture.Root, "scenarios", "demo", "platforms", "electron", "dist-electron", "demo.AppImage"), 256<<20)

	originalOpen := scanOpenFile
	t.Cleanup(func() { scanOpenFile = originalOpen })
	scanOpenFile = func(path string) (*os.File, error) {
		slash := filepath.ToSlash(path)
		if strings.Contains(slash, "/data/") || strings.Contains(slash, "/dist-electron/") {
			t.Fatalf("skipped heavyweight path was opened for content scan: %s", slash)
		}
		return originalOpen(path)
	}

	report, err := Audit(fixture.Root, "")
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.ScanStats.FilesScanned == 0 {
		t.Fatalf("expected some eligible files to be scanned, stats = %#v", report.ScanStats)
	}
	if got := report.ScanStats.SkippedByReason["runtime-data-dir"]; got == 0 {
		t.Fatalf("expected runtime data dir skip, stats = %#v", report.ScanStats)
	}
	if got := report.ScanStats.SkippedByReason["output-dir"]; got == 0 {
		t.Fatalf("expected output dir skip, stats = %#v", report.ScanStats)
	}
}

func TestScanDocsDriftBudgetSkipsOversizedEligibleText(t *testing.T) {
	root := t.TempDir()
	testkitgo.WriteFile(t, filepath.Join(root, "docs", "small.md"), "workspace:*\n")
	testkitgo.WriteFile(t, filepath.Join(root, "docs", "large.md"), strings.Repeat("a", 128))

	issues, stats, err := scanDocsDriftWithPolicy(filepath.Join(root, "docs"), scanPolicy{MaxFileBytes: 64, MaxTotalBytes: 1024})
	if err != nil {
		t.Fatalf("scanDocsDriftWithPolicy: %v", err)
	}
	if !stats.BudgetExceeded {
		t.Fatalf("expected file budget to be exceeded, stats = %#v", stats)
	}
	if got := stats.SkippedByReason["file-byte-budget"]; got != 1 {
		t.Fatalf("file-byte-budget skips = %d, stats = %#v", got, stats)
	}
	if len(issues) != 1 || issues[0].Code != "workspace-star-guidance" {
		t.Fatalf("issues = %#v", issues)
	}
}

func BenchmarkAuditAllSyntheticHeavyFiles(b *testing.B) {
	root := newBenchmarkRepoFixture(b)
	for i := range 24 {
		name := fmt.Sprintf("pkg-%02d", i)
		writeBenchmarkPackageManifestFixture(b, root, name, packageManifestFixture(name, func(manifest *Manifest) {
			manifest.Package.ModuleIdentifiers = []string{"@vrooli/" + name}
		}))
	}
	for i := range 200 {
		name := fmt.Sprintf("demo-%03d", i)
		writeBenchmarkScenarioService(b, root, name)
		dep := fmt.Sprintf("@vrooli/pkg-%02d", i%24)
		writeBenchmarkScenarioUIPackageManifestFixture(b, root, name, map[string]string{dep: "file:../../../packages/" + strings.TrimPrefix(dep, "@vrooli/")}, nil)
		writeBenchmarkFile(b, filepath.Join(root, "scenarios", name, "README.md"), "# "+name+"\n")
	}
	mustSparseFile(b, filepath.Join(root, "scenarios", "demo-000", "data", "demo.db"), 8<<30)
	mustSparseFile(b, filepath.Join(root, "scenarios", "demo-001", "platforms", "electron", "dist-electron", "demo.AppImage"), 256<<20)

	b.ResetTimer()
	for range b.N {
		if _, err := Audit(root, ""); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateAllManyPackagesManyConsumers(b *testing.B) {
	root := newBenchmarkRepoFixture(b)
	for i := range 32 {
		name := fmt.Sprintf("pkg-%02d", i)
		writeBenchmarkPackageManifestFixture(b, root, name, packageManifestFixture(name, func(manifest *Manifest) {
			manifest.Package.ModuleIdentifiers = []string{"@vrooli/" + name}
		}))
	}
	for i := range 500 {
		name := fmt.Sprintf("demo-%03d", i)
		writeBenchmarkScenarioService(b, root, name)
		dep := fmt.Sprintf("@vrooli/pkg-%02d", i%32)
		writeBenchmarkScenarioUIPackageManifestFixture(b, root, name, map[string]string{dep: "file:../../../packages/" + strings.TrimPrefix(dep, "@vrooli/")}, nil)
	}

	b.ResetTimer()
	for range b.N {
		if _, err := Validate(root, ""); err != nil {
			b.Fatal(err)
		}
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

type testHelper interface {
	Helper()
	Fatalf(format string, args ...any)
}

func mustSparseFile(t testHelper, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir sparse file parent: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create sparse file: %v", err)
	}
	if err := file.Truncate(size); err != nil {
		_ = file.Close()
		t.Fatalf("truncate sparse file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close sparse file: %v", err)
	}
}

func newBenchmarkRepoFixture(b *testing.B) string {
	b.Helper()
	root := b.TempDir()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	contract, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		b.Fatalf("read live repo contract: %v", err)
	}
	writeBenchmarkFile(b, filepath.Join(root, ".vrooli", "repo-contract.json"), string(contract))
	writeBenchmarkFile(b, filepath.Join(root, "go.mod"), "module benchmark\n")
	for _, dir := range []string{"packages", "scenarios", "templates", "resources", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			b.Fatalf("mkdir benchmark dir: %v", err)
		}
	}
	return root
}

func writeBenchmarkPackageManifestFixture(b *testing.B, root, name string, manifest Manifest) {
	b.Helper()
	writeBenchmarkJSON(b, filepath.Join(root, "packages", name, ".vrooli", "package.json"), manifest)
}

func writeBenchmarkScenarioService(b *testing.B, root, name string) {
	b.Helper()
	writeBenchmarkJSON(b, filepath.Join(root, "scenarios", name, ".vrooli", "service.json"), map[string]any{
		"id":      name,
		"name":    name,
		"version": "1.0.0",
	})
}

func writeBenchmarkScenarioUIPackageManifestFixture(b *testing.B, root, scenarioName string, dependencies, scripts map[string]string) {
	b.Helper()
	writeBenchmarkJSON(b, filepath.Join(root, "scenarios", scenarioName, "ui", "package.json"), map[string]any{
		"name":         scenarioName + "-ui",
		"private":      true,
		"dependencies": dependencies,
		"scripts":      scripts,
	})
}

func writeBenchmarkJSON(b *testing.B, path string, value any) {
	b.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		b.Fatalf("marshal benchmark JSON: %v", err)
	}
	writeBenchmarkFile(b, path, string(data))
}

func writeBenchmarkFile(b *testing.B, path, content string) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatalf("mkdir benchmark file parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("write benchmark file: %v", err)
	}
}
