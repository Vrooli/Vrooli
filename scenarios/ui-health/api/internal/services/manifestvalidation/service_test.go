package manifestvalidation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateScenario_CleanFixture(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"layout-shell": {Dir: "ui/src/layout"},
		"page":         {Dir: "ui/src/pages", PathPattern: "{dir}/{ComponentName}.tsx"},
	}, nil)
	// Create the slot dirs on disk so on-disk reconciliation passes.
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "layout"))
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "pages"))

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if !rep.Passed {
		t.Fatalf("expected passed; got findings: %+v", rep.Findings)
	}
	if rep.Summary.Errors != 0 {
		t.Fatalf("expected 0 errors, got %d", rep.Summary.Errors)
	}
}

func TestValidateScenario_MissingSlotDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"page": {Dir: "ui/src/pages"},
	}, nil)
	// Intentionally do NOT create the page dir.

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "slot_dir_missing"); got != 1 {
		t.Fatalf("expected 1 slot_dir_missing finding, got %d (findings: %+v)", got, rep.Findings)
	}
}

func TestValidateScenario_PredatesTemplateLayoutCollapses(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Template declares 4 leaf slots; none of their dirs exist on disk.
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"layout-shell": {Dir: "ui/src/layout"},
		"layout-nav":   {Dir: "ui/src/layout-nav"},
		"page":         {Dir: "ui/src/pages"},
		"theme-token":  {Dir: "ui/src/theme"},
	}, nil)
	// Scenario has ui/package.json (so it counts as having a UI) but none
	// of the slot directories — i.e. it predates the slot layout.

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "slot_dir_missing"); got != 0 {
		t.Fatalf("expected per-slot warnings to be collapsed, got %d slot_dir_missing", got)
	}
	if got := findingCount(rep, "ui_predates_template_layout"); got != 1 {
		t.Fatalf("expected 1 ui_predates_template_layout finding, got %d (findings: %+v)", got, rep.Findings)
	}
	if !rep.Passed {
		t.Fatalf("predates-template should warn but pass; got %+v", rep.Findings)
	}
}

func TestValidateScenario_FewMissingSlotsAreNotCollapsed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// 4 slots, only 1 missing — per-slot signal is more useful than summary.
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"layout-shell": {Dir: "ui/src/layout"},
		"layout-nav":   {Dir: "ui/src/layout-nav"},
		"page":         {Dir: "ui/src/pages"},
		"theme-token":  {Dir: "ui/src/theme"},
	}, nil)
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "layout"))
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "layout-nav"))
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "pages"))
	// theme is missing.

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "slot_dir_missing"); got != 1 {
		t.Fatalf("expected 1 slot_dir_missing (no collapse), got %d (findings: %+v)", got, rep.Findings)
	}
	if got := findingCount(rep, "ui_predates_template_layout"); got != 0 {
		t.Fatalf("expected 0 ui_predates_template_layout (not enough missing), got %d", got)
	}
}

func TestValidateScenario_UnknownOverlaySlot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	setupFixture(t, root, "demo", "demo-template",
		map[string]uiManifestFix{"page": {Dir: "ui/src/pages"}},
		map[string]uiManifestFix{"bogus-slot": {Dir: "ui/src/bogus"}},
	)
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "pages"))

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "overlay_unknown_slot"); got != 1 {
		t.Fatalf("expected 1 overlay_unknown_slot finding, got %d (findings: %+v)", got, rep.Findings)
	}
}

func TestValidateScenario_UnknownPathPatternToken(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"page": {Dir: "ui/src/pages", PathPattern: "{dir}/{NotAToken}.tsx"},
	}, nil)
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "pages"))

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "path_pattern_unknown_token"); got != 1 {
		t.Fatalf("expected 1 path_pattern_unknown_token finding, got %d (findings: %+v)", got, rep.Findings)
	}
}

func TestValidateScenario_MissingTemplate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui"))
	mustWriteFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"), "{}")
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", ".vrooli"))
	mustWriteJSON(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]any{
		"generation": map[string]any{"template": map[string]any{"id": "nonexistent-template"}},
	})

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "template_unknown"); got != 1 {
		t.Fatalf("expected 1 template_unknown finding, got %d (findings: %+v)", got, rep.Findings)
	}
	if rep.Passed {
		t.Fatalf("expected failed; got passed: %+v", rep.Findings)
	}
}

func TestValidateScenario_NoUISurface(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "cli-only"))

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "cli-only")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if !rep.Passed {
		t.Fatalf("expected passed for no-UI scenario; got findings: %+v", rep.Findings)
	}
	if got := findingCount(rep, "no_ui_surface"); got != 1 {
		t.Fatalf("expected 1 no_ui_surface finding, got %d", got)
	}
}

func TestValidateScenario_TemplateIDMissingIsWarning(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui"))
	mustWriteFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"), "{}")
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", ".vrooli"))
	mustWriteJSON(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]any{})

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "template_id_missing"); got != 1 {
		t.Fatalf("expected 1 template_id_missing finding, got %d", got)
	}
	// Pre-template-tracking is a "validation impossible" state, not a defect —
	// the scenario still has a working UI. Surfacing as a warning lets gates
	// like phase_ui_health pass across the legacy fleet until each scenario
	// backfills generation.template.id.
	if !rep.Passed {
		t.Fatalf("template_id_missing must not fail validation; got findings: %+v", rep.Findings)
	}
	for _, f := range rep.Findings {
		if f.Code == "template_id_missing" && f.Severity != SeverityWarning {
			t.Fatalf("template_id_missing severity = %q, want %q", f.Severity, SeverityWarning)
		}
	}
}

func TestValidateScenario_PlaceholderSlot_NoInstances(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"feature": {Dir: "ui/src/features/{feature}", MultiFile: true},
	}, nil)
	mustWriteFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"), "{}")
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "features"))

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "slot_dir_missing"); got != 0 {
		t.Fatalf("placeholder slot must not produce slot_dir_missing, got %d (findings: %+v)", got, rep.Findings)
	}
	if got := findingCount(rep, "slot_instances_empty"); got != 1 {
		t.Fatalf("expected 1 slot_instances_empty finding, got %d (findings: %+v)", got, rep.Findings)
	}
	if !rep.Passed {
		t.Fatalf("info findings must not fail validation; got %+v", rep.Findings)
	}
}

func TestValidateScenario_PlaceholderSlot_WithInstances(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"feature": {Dir: "ui/src/features/{feature}", MultiFile: true},
	}, nil)
	mustWriteFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"), "{}")
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "features", "inbox"))
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", "ui", "src", "features", "settings"))

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "slot_instances_empty"); got != 0 {
		t.Fatalf("expected 0 slot_instances_empty with features present, got %d", got)
	}
	if !rep.Passed {
		t.Fatalf("expected passed; got %+v", rep.Findings)
	}
}

func TestValidateScenario_PlaceholderSlot_ParentMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	setupFixture(t, root, "demo", "demo-template", map[string]uiManifestFix{
		"feature": {Dir: "ui/src/features/{feature}", MultiFile: true},
	}, nil)
	mustWriteFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"), "{}")
	// Intentionally do NOT create ui/src/features.

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "slot_parent_dir_missing"); got != 1 {
		t.Fatalf("expected 1 slot_parent_dir_missing finding, got %d (findings: %+v)", got, rep.Findings)
	}
}

type uiManifestFix struct {
	Dir         string `json:"dir"`
	PathPattern string `json:"pathPattern,omitempty"`
	MultiFile   bool   `json:"multiFile,omitempty"`
}

// TestValidateScenario_ExplicitOutOfTreePath proves WithScenarioPath lets the
// validator inspect a scenario generated outside the repo scenarios/ tree (deep
// template validation's temp dir) while still resolving its template from the
// repo. Without the path the same name is correctly not found under scenarios/.
func TestValidateScenario_ExplicitOutOfTreePath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Template lives under the repo root (templates are always repo-rooted).
	mustMkdirAll(t, filepath.Join(root, "templates", "scenarios", "demo-template", "ui"))
	mustWriteJSON(t, filepath.Join(root, "templates", "scenarios", "demo-template", "ui", "manifest.json"), map[string]any{
		"contract": map[string]any{"kind": "scenario-ui", "schema": "scenario-ui-manifest/v1"},
		"slots":    map[string]uiManifestFix{"layout-shell": {Dir: "ui/src/layout"}},
	})
	// Scenario generated OUTSIDE root/scenarios.
	outDir := filepath.Join(t.TempDir(), "scenarios", "generated-demo")
	mustMkdirAll(t, filepath.Join(outDir, "ui", "src", "layout"))
	mustWriteFile(t, filepath.Join(outDir, "ui", "package.json"), "{}")
	mustMkdirAll(t, filepath.Join(outDir, ".vrooli"))
	mustWriteJSON(t, filepath.Join(outDir, ".vrooli", "service.json"), map[string]any{
		"generation": map[string]any{"template": map[string]any{"id": "demo-template"}},
	})

	svc := New(root, nil)

	if _, err := svc.ValidateScenario(context.Background(), "generated-demo"); err == nil {
		t.Fatal("expected not-found without explicit path")
	}

	rep, err := svc.ValidateScenario(WithScenarioPath(context.Background(), outDir), "generated-demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if !rep.Passed || rep.Summary.Errors != 0 {
		t.Fatalf("expected clean pass for out-of-tree scenario, got %+v", rep.Findings)
	}
}

func setupFixture(t *testing.T, root, scenario, template string, templateSlots, overlaySlots map[string]uiManifestFix) {
	t.Helper()
	mustMkdirAll(t, filepath.Join(root, "scenarios", scenario, "ui"))
	mustWriteFile(t, filepath.Join(root, "scenarios", scenario, "ui", "package.json"), "{}")
	mustMkdirAll(t, filepath.Join(root, "scenarios", scenario, ".vrooli"))
	mustMkdirAll(t, filepath.Join(root, "templates", "scenarios", template, "ui"))
	mustWriteJSON(t, filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json"), map[string]any{
		"generation": map[string]any{"template": map[string]any{"id": template}},
	})
	mustWriteJSON(t, filepath.Join(root, "templates", "scenarios", template, "ui", "manifest.json"), map[string]any{
		"contract": map[string]any{"kind": "scenario-ui", "schema": "scenario-ui-manifest/v1"},
		"slots":    templateSlots,
	})
	if overlaySlots != nil {
		mustWriteJSON(t, filepath.Join(root, "scenarios", scenario, ".vrooli", "ui-manifest.json"), map[string]any{
			"contract": map[string]any{"kind": "scenario-ui", "schema": "scenario-ui-manifest/v1"},
			"slots":    overlaySlots,
		})
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustWriteJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func findingCount(rep Report, code string) int {
	n := 0
	for _, f := range rep.Findings {
		if f.Code == code {
			n++
		}
	}
	return n
}
