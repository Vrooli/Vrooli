package discovery

import (
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

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
	if surface.Framework != "vite" {
		t.Errorf("framework = %q, want vite", surface.Framework)
	}
	if surface.PackageManager != "npm" {
		t.Errorf("package manager = %q, want npm", surface.PackageManager)
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
