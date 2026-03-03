package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupModuleFixture creates a requirements dir with an index.json and one module.
func setupModuleFixture(t *testing.T) string {
	t.Helper()
	itemDir := t.TempDir()
	reqDir := filepath.Join(itemDir, "requirements")
	modDir := filepath.Join(reqDir, "01-route-manifest")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}

	idx := map[string]any{
		"_metadata": map[string]any{
			"description":    "Test index",
			"schema_version": "1.0.0",
		},
		"imports":      []string{"01-route-manifest/module.json"},
		"requirements": []any{},
	}
	writeTestJSON(t, filepath.Join(reqDir, "index.json"), idx)

	mod := map[string]any{
		"module":      "route-manifest",
		"description": "Route manifest module",
		"prd_refs":    []string{"OT-P0-001"},
		"requirements": []map[string]any{
			{
				"id":          "ROUTE-001",
				"title":       "Route manifest schema",
				"description": "Schema definition",
				"prd_ref":     "OT-P0-001",
				"criticality": "P0",
				"status":      "pending",
				"validation": []map[string]any{
					{"type": "test", "phase": "api", "status": "planned", "ref": "test.go"},
				},
			},
		},
	}
	writeTestJSON(t, filepath.Join(modDir, "module.json"), mod)

	return itemDir
}

func writeTestJSON(t *testing.T, path string, data any) {
	t.Helper()
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestArchiveWriter_WriteModuleRequirements_RoundTrip(t *testing.T) {
	itemDir := setupModuleFixture(t)

	// Replace requirements with new ones.
	newReqs := []json.RawMessage{
		json.RawMessage(`{"id":"ROUTE-NEW","title":"New req","description":"desc","status":"done","criticality":"P1","validation":[{"type":"manual"}]}`),
	}

	if err := WriteModuleRequirements(itemDir, "route-manifest", newReqs); err != nil {
		t.Fatalf("WriteModuleRequirements: %v", err)
	}

	// Read back and verify requirements changed but other fields preserved.
	modPath := filepath.Join(itemDir, "requirements", "01-route-manifest", "module.json")
	doc := readTestJSON(t, modPath)

	// Module name preserved.
	if doc["module"] != "route-manifest" {
		t.Errorf("module = %v, want route-manifest", doc["module"])
	}
	// Description preserved.
	if doc["description"] != "Route manifest module" {
		t.Errorf("description = %v, want Route manifest module", doc["description"])
	}
	// prd_refs preserved.
	refs, ok := doc["prd_refs"].([]any)
	if !ok || len(refs) != 1 || refs[0] != "OT-P0-001" {
		t.Errorf("prd_refs = %v, want [OT-P0-001]", doc["prd_refs"])
	}
	// Requirements replaced.
	reqs, ok := doc["requirements"].([]any)
	if !ok || len(reqs) != 1 {
		t.Fatalf("requirements length = %d, want 1", len(reqs))
	}
	req := reqs[0].(map[string]any)
	if req["id"] != "ROUTE-NEW" {
		t.Errorf("req.id = %v, want ROUTE-NEW", req["id"])
	}
	// Validation array preserved in new requirement.
	val, ok := req["validation"].([]any)
	if !ok || len(val) != 1 {
		t.Errorf("validation = %v, want [{type:manual}]", req["validation"])
	}
}

func TestArchiveWriter_CreateModule_NoExistingDir(t *testing.T) {
	itemDir := t.TempDir()

	input := CreateModuleInput{
		ID:          "auth-module",
		Title:       "Authentication",
		Description: "Handles auth",
	}

	if err := CreateModule(itemDir, input, 0); err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	// Verify directory structure created.
	reqDir := filepath.Join(itemDir, "requirements")
	if _, err := os.Stat(reqDir); err != nil {
		t.Fatalf("requirements dir not created: %v", err)
	}

	// Verify index.json.
	idxDoc := readTestJSON(t, filepath.Join(reqDir, "index.json"))
	imports, ok := idxDoc["imports"].([]any)
	if !ok || len(imports) != 1 {
		t.Fatalf("imports = %v, want 1 entry", idxDoc["imports"])
	}
	if imports[0] != "01-auth-module/module.json" {
		t.Errorf("import = %v, want 01-auth-module/module.json", imports[0])
	}

	// Verify module.json.
	modDoc := readTestJSON(t, filepath.Join(reqDir, "01-auth-module", "module.json"))
	if modDoc["module"] != "auth-module" {
		t.Errorf("module = %v, want auth-module", modDoc["module"])
	}
	if modDoc["description"] != "Handles auth" {
		t.Errorf("description = %v, want Handles auth", modDoc["description"])
	}
	reqs, ok := modDoc["requirements"].([]any)
	if !ok || len(reqs) != 0 {
		t.Errorf("requirements = %v, want empty", modDoc["requirements"])
	}
}

func TestArchiveWriter_CreateModule_ArchiveDir(t *testing.T) {
	// When archive/requirements/ exists, new modules go there.
	itemDir := t.TempDir()
	archiveReq := filepath.Join(itemDir, "archive", "requirements")
	if err := os.MkdirAll(archiveReq, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a minimal index.
	writeTestJSON(t, filepath.Join(archiveReq, "index.json"), map[string]any{
		"imports":      []string{},
		"requirements": []any{},
	})

	input := CreateModuleInput{ID: "new-mod", Description: "test"}
	if err := CreateModule(itemDir, input, 1); err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	// Module should be under archive/requirements/, not requirements/.
	modPath := filepath.Join(archiveReq, "01-new-mod", "module.json")
	if _, err := os.Stat(modPath); err != nil {
		t.Fatalf("module not created in archive dir: %v", err)
	}
	// Root requirements/ should not exist.
	if _, err := os.Stat(filepath.Join(itemDir, "requirements")); !os.IsNotExist(err) {
		t.Error("root requirements/ should not have been created")
	}
}

func TestArchiveWriter_DeleteModule(t *testing.T) {
	itemDir := setupModuleFixture(t)

	if err := DeleteModule(itemDir, "route-manifest"); err != nil {
		t.Fatalf("DeleteModule: %v", err)
	}

	// Directory should be gone.
	modDir := filepath.Join(itemDir, "requirements", "01-route-manifest")
	if _, err := os.Stat(modDir); !os.IsNotExist(err) {
		t.Error("module directory should have been removed")
	}

	// Index should have no imports.
	idxDoc := readTestJSON(t, filepath.Join(itemDir, "requirements", "index.json"))
	imports, ok := idxDoc["imports"].([]any)
	if !ok || len(imports) != 0 {
		t.Errorf("imports = %v, want empty", idxDoc["imports"])
	}

	// Metadata should be preserved.
	meta, ok := idxDoc["_metadata"].(map[string]any)
	if !ok || meta["description"] != "Test index" {
		t.Errorf("_metadata not preserved after delete: %v", idxDoc["_metadata"])
	}
}

func TestArchiveWriter_DeleteModule_NotFound(t *testing.T) {
	itemDir := setupModuleFixture(t)

	err := DeleteModule(itemDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}
}

func TestArchiveWriter_ResolveModulePath(t *testing.T) {
	itemDir := setupModuleFixture(t)
	reqDir := filepath.Join(itemDir, "requirements")

	path, err := resolveModulePath(reqDir, "route-manifest")
	if err != nil {
		t.Fatalf("resolveModulePath: %v", err)
	}

	expected := filepath.Join(reqDir, "01-route-manifest", "module.json")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

func TestArchiveWriter_ResolveModulePath_NotFound(t *testing.T) {
	itemDir := setupModuleFixture(t)
	reqDir := filepath.Join(itemDir, "requirements")

	_, err := resolveModulePath(reqDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}
}

func TestArchiveWriter_UpdateModuleMeta(t *testing.T) {
	itemDir := setupModuleFixture(t)

	if err := UpdateModuleMeta(itemDir, "route-manifest", "new-title", "new-desc"); err != nil {
		t.Fatalf("UpdateModuleMeta: %v", err)
	}

	modPath := filepath.Join(itemDir, "requirements", "01-route-manifest", "module.json")
	doc := readTestJSON(t, modPath)

	if doc["module"] != "new-title" {
		t.Errorf("module = %v, want new-title", doc["module"])
	}
	if doc["description"] != "new-desc" {
		t.Errorf("description = %v, want new-desc", doc["description"])
	}

	// Requirements should be preserved.
	reqs, ok := doc["requirements"].([]any)
	if !ok || len(reqs) != 1 {
		t.Fatalf("requirements should be preserved, got %v", doc["requirements"])
	}
	req := reqs[0].(map[string]any)
	if req["id"] != "ROUTE-001" {
		t.Errorf("requirement id = %v, want ROUTE-001", req["id"])
	}
	// Validation should be preserved within requirements.
	val, ok := req["validation"].([]any)
	if !ok || len(val) != 1 {
		t.Errorf("validation should be preserved, got %v", req["validation"])
	}
}

func TestArchiveWriter_UpdateModuleMeta_PartialUpdate(t *testing.T) {
	itemDir := setupModuleFixture(t)

	// Only update description, leave title empty.
	if err := UpdateModuleMeta(itemDir, "route-manifest", "", "only-desc"); err != nil {
		t.Fatalf("UpdateModuleMeta: %v", err)
	}

	modPath := filepath.Join(itemDir, "requirements", "01-route-manifest", "module.json")
	doc := readTestJSON(t, modPath)

	// Module name should be unchanged.
	if doc["module"] != "route-manifest" {
		t.Errorf("module = %v, want route-manifest (unchanged)", doc["module"])
	}
	if doc["description"] != "only-desc" {
		t.Errorf("description = %v, want only-desc", doc["description"])
	}
}
