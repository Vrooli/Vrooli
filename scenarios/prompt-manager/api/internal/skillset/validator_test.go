package skillset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateReportsPresentWaivedAndAbsentRoles(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(filepath.Join(root, "scenarios", scenario, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", scenario, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", scenario, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", scenario, "cli", "manifest.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", scenario, "skills", "usage.md"), []byte("---\nname: demo\nmetadata:\n  kind: tools\n  modes: [tools]\n---\nusage"), 0o644); err != nil {
		t.Fatal(err)
	}
	service, err := json.Marshal(map[string]any{"skills": map[string]any{
		"usage":   map[string]any{"source": "skills/usage.md"},
		"waivers": map[string]any{"improve": map[string]any{"reason": "not owed", "declared_at": "2026-09-03"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), service, 0o644); err != nil {
		t.Fatal(err)
	}
	result := Validate(root, scenario)
	if result.Status != "ok" || len(result.Roles) != 3 {
		t.Fatalf("result=%+v", result)
	}
	if result.Roles[0].Status != "present" || result.Roles[1].Status != "waived" || result.Roles[2].Status != "not_applicable" {
		t.Fatalf("roles=%+v", result.Roles)
	}
}

func TestValidateReportsMissingDeclaredSource(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "demo", ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "demo", "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "demo", "cli", "manifest.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	service, err := json.Marshal(map[string]any{"skills": map[string]any{"usage": map[string]any{"source": "skills/missing.md"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), service, 0o644); err != nil {
		t.Fatal(err)
	}
	result := Validate(root, "demo")
	if result.Status != "findings" || len(result.Findings) != 1 || result.Findings[0] != "skill-set.source_missing:usage" {
		t.Fatalf("result=%+v", result)
	}
}
