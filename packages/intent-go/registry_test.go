package intent

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRegistry(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestExtractRequirementsRegistryMissing(t *testing.T) {
	reg, err := ExtractRequirementsRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if reg.Present {
		t.Fatal("expected Present=false")
	}
}

func TestExtractRequirementsRegistryFull(t *testing.T) {
	dir := writeRegistry(t, map[string]string{
		"requirements/index.json": `{"imports":["01-core/module.json"]}`,
		"requirements/01-core/module.json": `{"requirements":[
			{"id":"X-001","title":"First","status":"planned","criticality":"P0","prd_ref":"OT-P0-001",
			 "tags":["core"],"children":["X-002"],"depends_on":[],
			 "validation":[{"type":"test","ref":"api/x_test.go","status":"planned","phase":"unit","notes":"n"}]},
			{"id":"X-002","title":"Second","status":"weird","prd_ref":"OT-P0-001","validation":[]}
		]}`,
		"requirements/02-orphan/module.json": `{"requirements":[{"id":"Y-001","title":"Orphan module record","status":"planned","prd_ref":"OT-P0-001"}]}`,
	})
	reg, err := ExtractRequirementsRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reg.Present {
		t.Fatal("expected Present")
	}
	if len(reg.Modules) != 2 {
		t.Fatalf("modules = %d, want 2 (declared + walked orphan)", len(reg.Modules))
	}
	reqs := reg.Requirements()
	if len(reqs) != 3 {
		t.Fatalf("requirements = %d, want 3", len(reqs))
	}
	first := reqs[0]
	if first.ID != "X-001" || first.Status != "planned" || first.Criticality != "P0" {
		t.Fatalf("unexpected first record: %+v", first)
	}
	if len(first.Validations) != 1 || first.Validations[0].Ref != "api/x_test.go" || first.Validations[0].Phase != "unit" {
		t.Fatalf("validations not extracted: %+v", first.Validations)
	}
	if len(first.Children) != 1 || first.Children[0] != "X-002" {
		t.Fatalf("children not extracted: %+v", first.Children)
	}
	if !first.HasTag("CORE") {
		t.Fatal("HasTag should be case-insensitive")
	}
	if first.Module != "requirements/01-core/module.json" {
		t.Fatalf("module path = %q", first.Module)
	}
}

func TestExtractRequirementsRegistryParseErrors(t *testing.T) {
	dir := writeRegistry(t, map[string]string{
		"requirements/index.json":         `{"imports":["01-a/module.json","02-missing/module.json"]}`,
		"requirements/01-a/module.json":   `{not json`,
		"requirements/03-bad/module.json": `also not json`,
	})
	reg, err := ExtractRequirementsRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reg.Present {
		t.Fatal("expected Present")
	}
	if len(reg.ParseErrors) != 3 {
		t.Fatalf("parse errors = %d (%+v), want 3 (bad declared, missing declared, bad walked)", len(reg.ParseErrors), reg.ParseErrors)
	}
}
