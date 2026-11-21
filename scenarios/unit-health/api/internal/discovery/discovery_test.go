package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type failingResolver struct{}

func (failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", os.ErrNotExist
}

func TestDiscoveryHelpersCoverLocatorFallbackAndObservedToolchains(t *testing.T) {
	root := t.TempDir()
	if name, kind, got, err := (DefaultLocator{}).Locate(context.Background(), "demo", root); err != nil || name != "demo" || kind != "path" || got == "" {
		t.Fatalf("path locate = %q,%q,%q,%v", name, kind, got, err)
	}
	if _, _, _, err := (DefaultLocator{RepoRoot: root}).Locate(context.Background(), "", ""); err == nil {
		t.Fatal("empty locator accepted")
	}
	for _, kind := range []string{"scenario", "package", "control-plane", "resource", "project", "other"} {
		if targetKindToProto(kind).String() == "" {
			t.Fatalf("empty target kind for %s", kind)
		}
	}
	for _, observation := range []ToolchainObservation{{RunnerIndicators: []string{"dependency:vitest"}}, {RunnerIndicators: []string{"devDependency:jest"}}, {RunnerIndicators: []string{"config:pytest"}}, {Ecosystem: "go"}, {Ecosystem: "rust"}, {Ecosystem: "bash", RunnerIndicators: []string{"source:.bats"}}} {
		if frameworkFromToolchain(observation) == "" {
			t.Fatalf("toolchain not recognized: %+v", observation)
		}
	}
	if confidence(nil) != 0.8 || confidence([]*factsv1.Evidence{{Confidence: 0.9}}) != 0.9 {
		t.Fatal("confidence calculation failed")
	}
	for _, name := range []string{"pyproject.toml", "setup.py", "requirements.txt"} {
		dir := t.TempDir()
		write(t, filepath.Join(dir, name), "")
		if !hasPythonIndicators(dir) {
			t.Fatalf("python marker %s not recognized", name)
		}
	}
	shell := t.TempDir()
	write(t, filepath.Join(shell, "run.sh"), "#!/bin/sh\n")
	if !hasShellIndicators(shell) {
		t.Fatal("shell indicator not recognized")
	}
	for _, want := range []struct{ name, file, language string }{{"go", "go.mod", "go"}, {"typescript", "tsconfig.json", "typescript"}, {"javascript", "package.json", "javascript"}} {
		dir := t.TempDir()
		write(t, filepath.Join(dir, want.file), "{}")
		if languageFromRoot(dir) != want.language {
			t.Fatalf("language for %s = %q", want.name, languageFromRoot(dir))
		}
	}
	if languageFromRoot(t.TempDir()) != "unknown" {
		t.Fatal("empty root was not unknown")
	}
	nested := t.TempDir()
	mkdir(t, filepath.Join(nested, "cli"))
	write(t, filepath.Join(nested, "cli", "package.json"), `{"devDependencies":{"vitest":"1"}}`)
	inv := fallbackInventory("demo", "scenario", nested)
	if len(inv.Surfaces) != 1 || inv.Surfaces[0].ID != "cli" {
		t.Fatalf("nested fallback = %+v", inv)
	}
	if report := fromCodeFacts(nil, "demo", "scenario", root); report.DegradedReason == "" {
		t.Fatal("nil Code Facts report was not degraded")
	}
}

func TestCodeFactsDiscoverDegradesWhenOwnerUnavailable(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module demo\n")
	client := CodeFactsClient{Resolver: failingResolver{}, Locator: DefaultLocator{}}
	inv, err := client.Discover(context.Background(), "demo", "scenario", root, false)
	if err != nil || inv.DegradedReason == "" || len(inv.Surfaces) != 1 {
		t.Fatalf("degraded discovery = %+v, %v", inv, err)
	}
}

func TestFromCodeFactsMapsSurfacesAndLanguages(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "api"))
	write(t, filepath.Join(root, "api", "go.mod"), "module x\n")
	mkdir(t, filepath.Join(root, "ui"))
	write(t, filepath.Join(root, "ui", "package.json"), `{"dependencies":{"react":"18","vite":"5"}}`)
	write(t, filepath.Join(root, "ui", "pnpm-lock.yaml"), "lockfileVersion: 9\n")

	report := &factsv1.CodeFactsReport{
		Target: &factsv1.TargetContext{Scenario: "demo", RootPath: root},
		Surfaces: []*factsv1.Surface{
			{Id: "api", Kind: factsv1.SurfaceKind_SURFACE_KIND_API, Path: filepath.Join(root, "api")},
			{Id: "ui", Kind: factsv1.SurfaceKind_SURFACE_KIND_UI, Path: filepath.Join(root, "ui")},
		},
		ParseUnits: []*factsv1.ParseUnit{
			{Language: "go", RootPath: filepath.Join(root, "api")},
			{Language: "typescript", RootPath: filepath.Join(root, "ui")},
		},
	}

	inv := fromCodeFacts(report, "demo", "scenario", root)
	if inv.DegradedReason != "" {
		t.Fatalf("unexpected degraded reason: %q", inv.DegradedReason)
	}
	if len(inv.Surfaces) != 2 {
		t.Fatalf("expected 2 surfaces, got %d", len(inv.Surfaces))
	}
	byID := map[string]Surface{}
	for _, s := range inv.Surfaces {
		byID[s.ID] = s
	}
	if byID["api"].Language != "go" {
		t.Errorf("api language = %q, want go", byID["api"].Language)
	}
	if byID["ui"].Language != "typescript" {
		t.Errorf("ui language = %q, want typescript", byID["ui"].Language)
	}
	if byID["ui"].Framework != "react-vite" {
		t.Errorf("ui framework = %q, want react-vite", byID["ui"].Framework)
	}
	if byID["ui"].PackageManager != "pnpm" {
		t.Errorf("ui package manager = %q, want pnpm", byID["ui"].PackageManager)
	}
}

func TestFromCodeFactsEmptyFallsBackToFilesystem(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "api"))
	write(t, filepath.Join(root, "api", "go.mod"), "module x\n")

	report := &factsv1.CodeFactsReport{Target: &factsv1.TargetContext{Scenario: "demo", RootPath: root}}
	inv := fromCodeFacts(report, "demo", "scenario", root)
	if inv.DegradedReason == "" {
		t.Errorf("expected degraded reason when Code Facts returns no surfaces")
	}
	if len(inv.Surfaces) != 1 || inv.Surfaces[0].ID != "api" {
		t.Fatalf("expected filesystem fallback to find api surface, got %+v", inv.Surfaces)
	}
	if inv.Surfaces[0].Language != "go" {
		t.Errorf("fallback api language = %q, want go", inv.Surfaces[0].Language)
	}
}

func TestFromCodeFactsUsesNeutralToolchainObservation(t *testing.T) {
	root := t.TempDir()
	uiRoot := filepath.Join(root, "ui")
	report := &factsv1.CodeFactsReport{
		Target:   &factsv1.TargetContext{Scenario: "demo", RootPath: root},
		Surfaces: []*factsv1.Surface{{Id: "ui", Kind: factsv1.SurfaceKind_SURFACE_KIND_UI, Path: uiRoot}},
		ParseUnits: []*factsv1.ParseUnit{{
			Language: "typescript", RootPath: uiRoot,
			Toolchain: &factsv1.ToolchainObservation{
				Ecosystem: "node", PackageManager: "pnpm@9", Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
				RunnerIndicators: []string{"devDependency:vitest"},
			},
		}},
	}
	inv := fromCodeFacts(report, "demo", "scenario", root)
	if len(inv.Surfaces) != 1 || inv.Surfaces[0].Framework != "vitest" {
		t.Fatalf("surface did not use observed runner indicator: %+v", inv.Surfaces)
	}
	if inv.Surfaces[0].PackageManager != "pnpm@9" {
		t.Fatalf("package manager = %q, want versioned observed value", inv.Surfaces[0].PackageManager)
	}
}

func TestFromCodeFactsPrefersSpecificLanguageWhenParseUnitsShareRoot(t *testing.T) {
	root := t.TempDir()
	uiRoot := filepath.Join(root, "ui")
	report := &factsv1.CodeFactsReport{
		Target:   &factsv1.TargetContext{Scenario: "demo", RootPath: root},
		Surfaces: []*factsv1.Surface{{Id: "ui", Kind: factsv1.SurfaceKind_SURFACE_KIND_UI, Path: uiRoot}},
		ParseUnits: []*factsv1.ParseUnit{
			{Language: "typescript", RootPath: uiRoot},
			{Language: "node", RootPath: uiRoot},
		},
	}

	inv := fromCodeFacts(report, "demo", "scenario", root)
	if len(inv.Surfaces) != 1 || inv.Surfaces[0].Language != "typescript" {
		t.Fatalf("language = %q, want stable specific typescript selection", inv.Surfaces[0].Language)
	}
}

func TestFallbackInventoryDiscoversRootNodeSurface(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "package.json"), `{"devDependencies":{"vitest":"latest","vite":"latest"}}`)
	write(t, filepath.Join(root, "package-lock.json"), "{}\n")

	inv := fallbackInventory("simple-test", "scenario", root)
	if len(inv.Surfaces) != 1 {
		t.Fatalf("expected one root surface, got %+v", inv.Surfaces)
	}
	surface := inv.Surfaces[0]
	if surface.ID != "node" || surface.RootPath != root {
		t.Fatalf("unexpected root surface: %+v", surface)
	}
	if surface.Language != "javascript" {
		t.Errorf("language = %q, want javascript", surface.Language)
	}
	if surface.Framework != "vitest" {
		t.Errorf("framework = %q, want declared vitest runner", surface.Framework)
	}
	if surface.PackageManager != "npm" {
		t.Errorf("package manager = %q, want npm", surface.PackageManager)
	}
}

func TestFallbackFrameworkUsesDeclaredRunnerNotApplicationFramework(t *testing.T) {
	for _, tc := range []struct{ name, manifest, want string }{
		{"vitest UI", `{"dependencies":{"react":"18"},"devDependencies":{"vite":"5","vitest":"2.1.9"}}`, "vitest"},
		{"jest UI", `{"dependencies":{"react":"18"},"devDependencies":{"vite":"5","jest":"29"}}`, "jest"},
		{"ambiguous runners", `{"devDependencies":{"vitest":"2","jest":"29"}}`, ""},
		{"text is not a dependency", `{"description":"vitest","scripts":{"vitest":"echo not a runner"}}`, "node"},
		{"malformed manifest", `{"devDependencies":`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, filepath.Join(root, "package.json"), tc.manifest)
			if got := frameworkFromRoot(root); got != tc.want {
				t.Fatalf("framework=%q, want %q", got, tc.want)
			}
		})
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
