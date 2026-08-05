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

func (s stubLoader) LoadTemplate(_ string) (uimanifest.Manifest, error) {
	return s.mf, s.err
}

func sampleManifest() uimanifest.Manifest {
	return uimanifest.Manifest{
		Contract: uimanifest.Contract{Kind: "scenario-ui", Schema: "scenario-ui-manifest/v1", Template: "react-vite"},
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
			"hook": {
				Dir:         "ui/src/hooks",
				PathPattern: "{dir}/{camelName}.ts",
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

// findResolved returns the resolved file with the given library basename.
func findResolved(t *testing.T, files []ResolvedFile, libraryPath string) ResolvedFile {
	t.Helper()
	for _, f := range files {
		if f.LibraryPath == libraryPath {
			return f
		}
	}
	t.Fatalf("no resolved file for %q; got %+v", libraryPath, files)
	return ResolvedFile{}
}

// TestResolveVersion_DrawerShell_MultiFileAcrossSlots is the load-bearing case:
// the entry .tsx lands in the shared-component dir (its declared ui-pattern
// slot is absent from the manifest, so pickSlot falls to the default), while
// the two use*.ts hooks are heuristically routed to the hook slot — matching
// the real web-console DrawerShell adoption paths.
func TestResolveVersion_DrawerShell_MultiFileAcrossSlots(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	out, err := r.ResolveVersion(VersionResolveInput{
		ComponentName: "DrawerShell",
		ComponentSlot: "ui-pattern", // not in manifest -> default shared-component
		Template:      "react-vite",
		Files: []FileInput{
			{Path: "DrawerShell.tsx", IsEntry: true},
			{Path: "useFocusTrap.ts"},
			{Path: "useEscapeKey.ts"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if !out.ManifestResolved {
		t.Fatalf("expected manifestResolved=true")
	}
	if out.Template != "react-vite" {
		t.Fatalf("template: got %q", out.Template)
	}

	entry := findResolved(t, out.Files, "DrawerShell.tsx")
	if entry.TargetPath != "ui/src/components/DrawerShell.tsx" {
		t.Fatalf("entry path: got %q", entry.TargetPath)
	}
	if entry.SlotSource != SlotSourceEntry {
		t.Fatalf("entry slotSource: got %q", entry.SlotSource)
	}
	if entry.Source != SourceTemplateManifest {
		t.Fatalf("entry source: got %q", entry.Source)
	}

	focus := findResolved(t, out.Files, "useFocusTrap.ts")
	if focus.TargetPath != "ui/src/hooks/useFocusTrap.ts" {
		t.Fatalf("useFocusTrap path: got %q", focus.TargetPath)
	}
	if focus.Slot != "hook" || focus.SlotSource != SlotSourceHeuristic {
		t.Fatalf("useFocusTrap slot=%q slotSource=%q", focus.Slot, focus.SlotSource)
	}

	esc := findResolved(t, out.Files, "useEscapeKey.ts")
	if esc.TargetPath != "ui/src/hooks/useEscapeKey.ts" {
		t.Fatalf("useEscapeKey path: got %q", esc.TargetPath)
	}
	if esc.Slot != "hook" {
		t.Fatalf("useEscapeKey slot: got %q", esc.Slot)
	}
}

// TestResolveVersion_ExplicitSlotWins verifies authored fileSlots metadata
// overrides the extension heuristic.
func TestResolveVersion_ExplicitSlotWins(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	out, err := r.ResolveVersion(VersionResolveInput{
		ComponentName: "Widget",
		ComponentSlot: "shared-component",
		Template:      "react-vite",
		Files: []FileInput{
			{Path: "Widget.tsx", IsEntry: true},
			// useThing.ts would heuristically be a hook, but explicit metadata
			// pins it to layout-nav.
			{Path: "useThing.ts", Slot: "layout-nav"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	thing := findResolved(t, out.Files, "useThing.ts")
	if thing.Slot != "layout-nav" || thing.SlotSource != SlotSourceExplicit {
		t.Fatalf("explicit slot not honored: slot=%q slotSource=%q", thing.Slot, thing.SlotSource)
	}
	// layout-nav pattern is {dir}/{ComponentName}.tsx; the .ts extension is
	// preserved from the library file.
	if thing.TargetPath != "ui/src/layout/useThing.ts" {
		t.Fatalf("explicit slot path: got %q", thing.TargetPath)
	}
}

// TestResolveVersion_FlatFallback_WhenManifestMissing verifies the resolver
// reports manifestResolved=false without inventing a framework-specific
// directory layout.
func TestResolveVersion_FlatFallback_WhenManifestMissing(t *testing.T) {
	r := NewResolver(stubLoader{err: errors.New("manifest missing")}, "")
	out, err := r.ResolveVersion(VersionResolveInput{
		ComponentName: "DrawerShell",
		ComponentSlot: "ui-pattern",
		Scenario:      "demo",
		Files: []FileInput{
			{Path: "DrawerShell.tsx", IsEntry: true},
			{Path: "useFocusTrap.ts"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if out.ManifestResolved {
		t.Fatal("expected manifestResolved=false when manifest missing")
	}
	if len(out.Warnings) == 0 {
		t.Fatal("expected a top-level warning on flat fallback")
	}
	entry := findResolved(t, out.Files, "DrawerShell.tsx")
	if entry.TargetPath != "DrawerShell.tsx" || entry.Source != SourceFallback {
		t.Fatalf("entry fallback: path=%q source=%q", entry.TargetPath, entry.Source)
	}
	hook := findResolved(t, out.Files, "useFocusTrap.ts")
	if hook.TargetPath != "useFocusTrap.ts" {
		t.Fatalf("hook fallback path: got %q", hook.TargetPath)
	}
}

// TestResolveVersion_NoFilesSynthesizesEntry verifies an empty file set still
// yields the entry placement.
func TestResolveVersion_NoFilesSynthesizesEntry(t *testing.T) {
	r := NewResolver(stubLoader{mf: sampleManifest()}, "")
	out, err := r.ResolveVersion(VersionResolveInput{
		ComponentName: "WidgetCard",
		ComponentSlot: "",
		Template:      "react-vite",
	})
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if len(out.Files) != 1 {
		t.Fatalf("want 1 synthesized file, got %d", len(out.Files))
	}
	if out.Files[0].TargetPath != "ui/src/components/WidgetCard.tsx" {
		t.Fatalf("synthesized entry path: got %q", out.Files[0].TargetPath)
	}
}
