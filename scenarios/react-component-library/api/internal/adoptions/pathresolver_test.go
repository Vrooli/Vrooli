package adoptions

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"react-component-library/internal/uimanifest"
)

// stubLoader returns the given manifest (or error) regardless of scenario.
type stubLoader struct {
	mf  uimanifest.Manifest
	err error
}

func (s stubLoader) Load(_ string) (uimanifest.Manifest, error) {
	return s.mf, s.err
}

func sampleManifest() uimanifest.Manifest {
	return uimanifest.Manifest{
		Contract: uimanifest.Contract{Kind: "scenario-ui", Schema: "scenario-ui-manifest/v1"},
		Slots: map[string]uimanifest.Slot{
			"ui-primitive": {
				Dir:         "ui/src/components/ui",
				PathPattern: "{dir}/{kebab-name}.tsx",
			},
			"layout-nav": {
				Dir:         "ui/src/layout",
				PathPattern: "{dir}/{ComponentName}.tsx",
			},
			"shared-component": {
				Dir: "ui/src/components",
			},
			"feature-component": {
				Dir:             "ui/src/features/{feature}",
				PathPattern:     "{dir}/{ComponentName}.tsx",
				RequiresFeature: true,
			},
		},
		Defaults: uimanifest.Defaults{Slot: "shared-component"},
	}
}

func TestResolve_ExplicitOverrideWins(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	got, err := r.Resolve(ResolveInput{
		ComponentSlot: "layout-nav",
		ComponentName: "SidebarShell",
		Scenario:      "demo",
		Override:      "ui/src/custom/Sidebar.tsx",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != SourceExplicit {
		t.Fatalf("source: want explicit, got %s", got.Source)
	}
	if got.Path != "ui/src/custom/Sidebar.tsx" {
		t.Fatalf("path: got %q", got.Path)
	}
}

func TestResolve_TemplateManifest_LayoutNav(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	got, err := r.Resolve(ResolveInput{
		ComponentSlot: "layout-nav",
		ComponentName: "SidebarShell",
		Scenario:      "demo",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != SourceTemplateManifest {
		t.Fatalf("source: want template-manifest, got %s", got.Source)
	}
	if got.Path != "ui/src/layout/SidebarShell.tsx" {
		t.Fatalf("path: got %q", got.Path)
	}
	if got.Slot != "layout-nav" {
		t.Fatalf("slot: got %q", got.Slot)
	}
}

func TestResolve_TemplateManifest_UIPrimitive_KebabCase(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	got, err := r.Resolve(ResolveInput{
		ComponentSlot: "ui-primitive",
		ComponentName: "SidebarShell",
		Scenario:      "demo",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Path != "ui/src/components/ui/sidebar-shell.tsx" {
		t.Fatalf("path: got %q", got.Path)
	}
}

func TestResolve_TemplateManifest_DefaultSlot(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	got, err := r.Resolve(ResolveInput{
		ComponentSlot: "", // no hint -> falls to defaults.slot ("shared-component")
		ComponentName: "WidgetCard",
		Scenario:      "demo",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Slot != "shared-component" {
		t.Fatalf("slot: want shared-component, got %q", got.Slot)
	}
	if got.Path != "ui/src/components/WidgetCard.tsx" {
		t.Fatalf("path: got %q", got.Path)
	}
}

func TestResolve_FeatureSlot_RequiresFeature(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	_, err := r.Resolve(ResolveInput{
		ComponentSlot: "feature-component",
		ComponentName: "NoteRow",
		Scenario:      "demo",
	})
	if err == nil {
		t.Fatal("expected error when feature is required but missing")
	}
}

func TestResolve_FeatureSlot_Substitutes(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	got, err := r.Resolve(ResolveInput{
		ComponentSlot: "feature-component",
		ComponentName: "NoteRow",
		Scenario:      "demo",
		Feature:       "notes",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Path != "ui/src/features/notes/NoteRow.tsx" {
		t.Fatalf("path: got %q", got.Path)
	}
}

func TestResolve_Heuristic_WhenManifestMissing(t *testing.T) {
	repo := t.TempDir()
	uiSrc := filepath.Join(repo, "scenarios", "demo", "ui", "src", "layout")
	if err := os.MkdirAll(uiSrc, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	r := NewResolver(stubLoader{err: errors.New("manifest missing")}, repo)
	got, err := r.Resolve(ResolveInput{
		ComponentSlot: "layout-nav",
		ComponentName: "BottomNav",
		Scenario:      "demo",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != SourceHeuristic {
		t.Fatalf("source: want heuristic, got %s", got.Source)
	}
	if got.Path != "ui/src/layout/BottomNav.tsx" {
		t.Fatalf("path: got %q", got.Path)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected at least one warning on heuristic")
	}
}

func TestResolve_Fallback_WhenScenarioDirAbsent(t *testing.T) {
	r := NewResolver(stubLoader{err: errors.New("manifest missing")}, t.TempDir())
	got, err := r.Resolve(ResolveInput{
		ComponentSlot: "layout-nav",
		ComponentName: "BottomNav",
		Scenario:      "demo",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != SourceFallback {
		t.Fatalf("source: want fallback, got %s", got.Source)
	}
	if got.Path != "ui/src/components/BottomNav.tsx" {
		t.Fatalf("path: got %q", got.Path)
	}
}

func TestResolve_RejectsTraversalOverride(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	_, err := r.Resolve(ResolveInput{
		ComponentSlot: "layout-nav",
		ComponentName: "X",
		Scenario:      "demo",
		Override:      "../../etc/passwd",
	})
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestResolve_RejectsAbsoluteOverride(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	_, err := r.Resolve(ResolveInput{
		ComponentSlot: "layout-nav",
		ComponentName: "X",
		Scenario:      "demo",
		Override:      "/tmp/Sidebar.tsx",
	})
	if err == nil {
		t.Fatal("expected absolute-path rejection")
	}
}

func TestResolve_MissingComponentName(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	_, err := r.Resolve(ResolveInput{Scenario: "demo"})
	if err == nil {
		t.Fatal("expected error when componentName missing")
	}
}
