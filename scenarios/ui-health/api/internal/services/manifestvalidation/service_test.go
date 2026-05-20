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
	mustMkdirAll(t, filepath.Join(root, "scenarios", "demo", ".vrooli"))
	mustWriteJSON(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]any{
		"generation": map[string]any{"template": map[string]any{"id": "nonexistent-template"}},
	})

	svc := New(root, nil)
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := findingCount(rep, "template_manifest_missing"); got != 1 {
		t.Fatalf("expected 1 template_manifest_missing finding, got %d (findings: %+v)", got, rep.Findings)
	}
}

type uiManifestFix struct {
	Dir         string `json:"dir"`
	PathPattern string `json:"pathPattern,omitempty"`
	MultiFile   bool   `json:"multiFile,omitempty"`
}

func setupFixture(t *testing.T, root, scenario, template string, templateSlots, overlaySlots map[string]uiManifestFix) {
	t.Helper()
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
