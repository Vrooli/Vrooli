package parsing

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/requirements/discovery"
	"test-genie/internal/requirements/types"
)

// memReader implements Reader for testing.
type memReader struct {
	files map[string][]byte
}

func newMemReader() *memReader {
	return &memReader{
		files: make(map[string][]byte),
	}
}

func (r *memReader) ReadFile(path string) ([]byte, error) {
	if data, ok := r.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func TestParser_Parse_ValidModule(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/module.json"] = []byte(`{
		"_metadata": {
			"module": "test-module",
			"description": "Test description"
		},
		"requirements": [
			{
				"id": "REQ-001",
				"title": "Test requirement",
				"status": "pending"
			}
		]
	}`)

	parser := New(reader)
	module, err := parser.Parse(context.Background(), "/test/module.json")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if module == nil {
		t.Fatal("expected non-nil module")
	}
	if module.Metadata.Module != "test-module" {
		t.Errorf("unexpected module name: %s", module.Metadata.Module)
	}
	if len(module.Requirements) != 1 {
		t.Errorf("expected 1 requirement, got: %d", len(module.Requirements))
	}
	if module.Requirements[0].ID != "REQ-001" {
		t.Errorf("unexpected requirement ID: %s", module.Requirements[0].ID)
	}
}

func TestParser_Parse_FileNotFound(t *testing.T) {
	reader := newMemReader()
	parser := New(reader)

	_, err := parser.Parse(context.Background(), "/nonexistent.json")

	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParser_Parse_InvalidJSON(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/invalid.json"] = []byte(`{invalid json`)

	parser := New(reader)
	_, err := parser.Parse(context.Background(), "/test/invalid.json")

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParser_Parse_WithValidations(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/module.json"] = []byte(`{
		"requirements": [
			{
				"id": "REQ-001",
				"title": "Test",
				"validation": [
					{"type": "test", "ref": "test/example.test.ts", "status": "implemented"},
					{"type": "automation", "ref": "automation/flow.json", "status": "planned"}
				]
			}
		]
	}`)

	parser := New(reader)
	module, err := parser.Parse(context.Background(), "/test/module.json")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(module.Requirements[0].Validations) != 2 {
		t.Errorf("expected 2 validations, got: %d", len(module.Requirements[0].Validations))
	}
	if module.Requirements[0].Validations[0].Type != types.ValTypeTest {
		t.Errorf("unexpected validation type: %s", module.Requirements[0].Validations[0].Type)
	}
}

func TestParser_Parse_NormalizesStatus(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/module.json"] = []byte(`{
		"requirements": [
			{"id": "REQ-001", "title": "Test", "status": "COMPLETE"},
			{"id": "REQ-002", "title": "Test 2", "status": "In_Progress"}
		]
	}`)

	parser := New(reader)
	module, err := parser.Parse(context.Background(), "/test/module.json")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if module.Requirements[0].Status != types.StatusComplete {
		t.Errorf("expected normalized status 'complete', got: %s", module.Requirements[0].Status)
	}
	if module.Requirements[1].Status != types.StatusInProgress {
		t.Errorf("expected normalized status 'in_progress', got: %s", module.Requirements[1].Status)
	}
}

func TestParser_Parse_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/module.json"] = []byte(`{"requirements": []}`)

	parser := New(reader)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := parser.Parse(ctx, "/test/module.json")

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestParser_ParseAll_MultipleFiles(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/module1.json"] = []byte(`{
		"_metadata": {"module": "mod1"},
		"requirements": [{"id": "REQ-001", "title": "Test 1"}]
	}`)
	reader.files["/test/module2.json"] = []byte(`{
		"_metadata": {"module": "mod2"},
		"requirements": [{"id": "REQ-002", "title": "Test 2"}]
	}`)

	parser := New(reader)

	files := []discovery.DiscoveredFile{
		{AbsolutePath: "/test/module1.json", RelativePath: "module1.json"},
		{AbsolutePath: "/test/module2.json", RelativePath: "module2.json"},
	}

	index, err := parser.ParseAll(context.Background(), files)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if index == nil {
		t.Fatal("expected non-nil index")
	}
	if len(index.Modules) != 2 {
		t.Errorf("expected 2 modules, got: %d", len(index.Modules))
	}
	if index.ByID["REQ-001"] == nil {
		t.Error("expected REQ-001 in index")
	}
	if index.ByID["REQ-002"] == nil {
		t.Error("expected REQ-002 in index")
	}
}

func TestParser_ParseAll_EmptyFiles(t *testing.T) {
	reader := newMemReader()
	parser := New(reader)

	index, err := parser.ParseAll(context.Background(), nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if index == nil {
		t.Fatal("expected non-nil index")
	}
	if len(index.Modules) != 0 {
		t.Errorf("expected 0 modules, got: %d", len(index.Modules))
	}
}

func TestModuleIndex_AddModule(t *testing.T) {
	index := NewModuleIndex()

	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Metadata:   types.ModuleMetadata{Module: "test"},
		Requirements: []types.Requirement{
			{
				ID:       "REQ-001",
				Title:    "Test",
				Children: []string{"REQ-002"},
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test/example.test.ts"},
				},
			},
			{
				ID:    "REQ-002",
				Title: "Child",
			},
		},
	}

	err := index.AddModule(module)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(index.Modules) != 1 {
		t.Errorf("expected 1 module, got: %d", len(index.Modules))
	}
	if index.ByID["REQ-001"] == nil {
		t.Error("expected REQ-001 in ByID")
	}
	if index.ByFile["/test/module.json"] == nil {
		t.Error("expected module in ByFile")
	}
	if index.ByModule["test-module"] == nil {
		t.Error("expected module in ByModule")
	}
	// Validation should be indexed by ref
	if refs := index.ValidationsByRef["test/example.test.ts"]; len(refs) != 1 {
		t.Errorf("expected validation ref indexed, got: %d", len(refs))
	}
}

func TestModuleIndex_AddModule_NilModule(t *testing.T) {
	index := NewModuleIndex()

	err := index.AddModule(nil)

	if err != nil {
		t.Errorf("nil module should not error: %v", err)
	}
	if len(index.Modules) != 0 {
		t.Error("nil module should not be added")
	}
}

func TestNormalizer_NormalizeModule(t *testing.T) {
	normalizer := NewNormalizer()

	module := &types.RequirementModule{
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Status: "PENDING",
				Validations: []types.Validation{
					{Type: "test", Status: "IMPLEMENTED"},
				},
			},
		},
	}

	normalizer.NormalizeModule(module)

	if module.Requirements[0].Status != types.StatusPending {
		t.Errorf("expected normalized status, got: %s", module.Requirements[0].Status)
	}
	if module.Requirements[0].Validations[0].Status != types.ValStatusImplemented {
		t.Errorf("expected normalized validation status, got: %s", module.Requirements[0].Validations[0].Status)
	}
}

// Integration test using real filesystem
func TestParser_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "requirements")

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	moduleData := []byte(`{
		"_metadata": {
			"module": "integration-test",
			"description": "Integration test module"
		},
		"requirements": [
			{
				"id": "INT-001",
				"title": "Integration requirement",
				"status": "in_progress",
				"criticality": "P0",
				"validation": [
					{"type": "test", "ref": "test/integration.test.ts", "status": "planned"}
				]
			}
		]
	}`)
	modulePath := filepath.Join(moduleDir, "module.json")
	if err := os.WriteFile(modulePath, moduleData, 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	parser := NewDefault()
	module, err := parser.Parse(context.Background(), modulePath)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if module.Metadata.Module != "integration-test" {
		t.Errorf("unexpected module name: %s", module.Metadata.Module)
	}
	if len(module.Requirements) != 1 {
		t.Errorf("expected 1 requirement, got: %d", len(module.Requirements))
	}
	if module.Requirements[0].Criticality != types.CriticalityP0 {
		t.Errorf("unexpected criticality: %s", module.Requirements[0].Criticality)
	}

	t.Logf("Parsed module: %s with %d requirements", module.Metadata.Module, len(module.Requirements))
}
