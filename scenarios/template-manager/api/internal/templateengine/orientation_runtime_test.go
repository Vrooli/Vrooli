package templateengine

import (
	"os"
	"path/filepath"
	"testing"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func TestEvaluateOrientationCheckPrimitives(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "docs", "START-HERE.md"), "ready\n")
	mustWrite(t, filepath.Join(root, "requirements", "index.json"), `{"imports":["real/module.json"],"modules":{"core":{}}}`)
	mustWrite(t, filepath.Join(root, "api", "internal", "orders", "service.go"), "package orders\n")
	mustWrite(t, filepath.Join(root, "api", "internal", "billing", "service.go"), "package billing\n")

	checks := []templatecontracts.TemplateOrientationCheck{
		{Kind: "file_exists", Path: "docs/START-HERE.md"},
		{Kind: "file_absent", Path: "missing.txt"},
		{Kind: "directory_exists", Path: "requirements"},
		{Kind: "glob_present", Pattern: "api/internal/*/service.go"},
		{Kind: "glob_min_count", Pattern: "api/internal/*/service.go", MinCount: 2},
		{Kind: "glob_absent", Pattern: "api/internal/notes*"},
		{Kind: "json_path_exists", Path: "requirements/index.json", Query: "imports.0"},
		{Kind: "json_min_entries", Path: "requirements/index.json", Query: "imports", MinCount: 1},
		{Kind: "json_min_entries", Path: "requirements/index.json", Query: "modules", MinCount: 1},
		{Kind: "text_contains", Path: "docs/START-HERE.md", Text: "ready"},
		{Kind: "text_absent", Path: "docs/START-HERE.md", Text: "placeholder"},
	}
	for _, check := range checks {
		report := evaluateOrientationCheck(HandlerDeps[struct{}]{}, struct{}{}, orientationEval{scenarioRoot: root}, check)
		if !report.Passed {
			t.Fatalf("%s should pass: %#v", check.Kind, report)
		}
	}
}

func TestValidateOrientationSourceRejectsUnsafeCleanup(t *testing.T) {
	info := templatecontracts.TemplateInfo{
		Name: "demo",
		Manifest: templatecontracts.TemplateManifest{
			Version: "0.1.0",
			Orientation: &templatecontracts.TemplateOrientation{
				CopyTo: ".vrooli/orientation.json",
				Finalize: templatecontracts.TemplateOrientationFinalize{
					Cleanup: []string{"../outside", "docs"},
				},
				Steps: []templatecontracts.TemplateOrientationStep{{
					ID:     "start",
					Checks: []templatecontracts.TemplateOrientationCheck{{Kind: "file_exists", Path: "README.md"}},
				}},
			},
		},
	}
	issues := validateOrientationSource(info)
	if len(issues) != 2 {
		t.Fatalf("issues = %#v, want 2", issues)
	}
}

func TestValidateOrientationSourceAcceptsContentAdapted(t *testing.T) {
	info := templatecontracts.TemplateInfo{
		Name: "demo",
		Manifest: templatecontracts.TemplateManifest{
			Version: "0.1.0",
			Orientation: &templatecontracts.TemplateOrientation{
				CopyTo: ".vrooli/orientation.json",
				Steps: []templatecontracts.TemplateOrientationStep{{
					ID: "charter",
					Checks: []templatecontracts.TemplateOrientationCheck{
						{Kind: "content_adapted", Path: "README.md"},
						{Kind: "content_adapted", Path: "CHANGELOG.md", MinCount: 2},
					},
				}},
			},
		},
	}
	if issues := validateOrientationSource(info); len(issues) != 0 {
		t.Fatalf("valid content_adapted checks should not raise issues: %#v", issues)
	}

	bad := info
	bad.Manifest.Orientation.Steps[0].Checks = []templatecontracts.TemplateOrientationCheck{
		{Kind: "content_adapted", Path: "../escape.md"},
		{Kind: "content_adapted", Path: "ok.md", MinCount: -1},
	}
	if issues := validateOrientationSource(bad); len(issues) != 2 {
		t.Fatalf("invalid content_adapted checks should raise 2 issues, got %#v", issues)
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
