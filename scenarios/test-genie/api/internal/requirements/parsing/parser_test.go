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

// --- Normalizer tests ---

func TestNormalizer_NormalizeModule_NilModule(t *testing.T) {
	normalizer := NewNormalizer()
	// Should not panic
	normalizer.NormalizeModule(nil)
}

func TestNormalizer_NormalizeRequirement_NilRequirement(t *testing.T) {
	normalizer := NewNormalizer()
	// Should not panic
	normalizer.NormalizeRequirement(nil)
}

func TestNormalizer_NormalizeValidation_NilValidation(t *testing.T) {
	normalizer := NewNormalizer()
	// Should not panic
	normalizer.NormalizeValidation(nil)
}

func TestNormalizer_NormalizeRequirement_TrimWhitespace(t *testing.T) {
	normalizer := NewNormalizer()
	req := &types.Requirement{
		ID:          "  REQ-001  ",
		Title:       "  Test Title  ",
		PRDRef:      "  prd-ref  ",
		Category:    "  category  ",
		Description: "  description  ",
	}

	normalizer.NormalizeRequirement(req)

	if req.ID != "REQ-001" {
		t.Errorf("ID not trimmed: %q", req.ID)
	}
	if req.Title != "Test Title" {
		t.Errorf("Title not trimmed: %q", req.Title)
	}
	if req.PRDRef != "prd-ref" {
		t.Errorf("PRDRef not trimmed: %q", req.PRDRef)
	}
}

func TestNormalizer_NormalizeValidation_TrimWhitespace(t *testing.T) {
	normalizer := NewNormalizer()
	val := &types.Validation{
		Ref:        "  test/file.test.ts  ",
		WorkflowID: "  workflow-123  ",
		Phase:      "  UNIT  ",
		Notes:      "  some notes  ",
		Scenario:   "  my-scenario  ",
		Folder:     "  folder  ",
	}

	normalizer.NormalizeValidation(val)

	if val.Ref != "test/file.test.ts" {
		t.Errorf("Ref not trimmed: %q", val.Ref)
	}
	if val.WorkflowID != "workflow-123" {
		t.Errorf("WorkflowID not trimmed: %q", val.WorkflowID)
	}
	if val.Phase != "unit" {
		t.Errorf("Phase not normalized: %q", val.Phase)
	}
}

func TestNormalizePhase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UNIT", "unit"},
		{"unit-test", "unit"},
		{"unit_test", "unit"},
		{"unittest", "unit"},
		{"integration-test", "integration"},
		{"integration_test", "integration"},
		{"e2e", "integration"},
		{"business-logic", "business"},
		{"struct", "structure"},
		{"deps", "dependencies"},
		{"perf", "performance"},
		{"playbook", "playbooks"},
		{"custom-phase", "custom-phase"},
		{"  spaced  ", "spaced"},
	}

	for _, tt := range tests {
		result := normalizePhase(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePhase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeStringSlice(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{nil, nil},
		{[]string{}, nil},
		{[]string{"  a  ", "  b  "}, []string{"a", "b"}},
		{[]string{"", "  ", "valid"}, []string{"valid"}},
		{[]string{"", ""}, nil},
	}

	for _, tt := range tests {
		result := normalizeStringSlice(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("normalizeStringSlice(%v) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("normalizeStringSlice(%v)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"req-001", "REQ-001"},
		{"  REQ-002  ", "REQ-002"},
		{"lower", "LOWER"},
	}

	for _, tt := range tests {
		result := NormalizeID(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test/file.ts", "test/file.ts"},
		{"  spaced  ", "spaced"},
		{"windows\\path\\file.ts", "windows/path/file.ts"},
	}

	for _, tt := range tests {
		result := NormalizeRef(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeRef(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractPhaseFromRef(t *testing.T) {
	// Note: The function checks patterns in order - test file suffixes first
	tests := []struct {
		ref      string
		expected string
	}{
		{"src/app.test.ts", "unit"},
		{"src/app.test.tsx", "unit"},
		{"src/app.test.js", "unit"},
		{"src/app_test.go", "unit"},
		{"src/app.spec.ts", "unit"},
		// These have test file suffixes so they match "unit" first
		{"test/integration/api.test.ts", "unit"},
		{"tests/integration_test.go", "unit"},
		// Integration detection via path contains "integration"
		{"integration/something.go", "integration"},
		{"test_integration_helpers.go", "integration"},
		{"bas/cases/workflow.json", "playbooks"},
		{"playbook/test.yaml", "playbooks"},
		{"scripts/test.bats", "business"},
		{"scripts/integration.bats", "integration"},
		{"random/file.ts", ""},
	}

	for _, tt := range tests {
		result := ExtractPhaseFromRef(tt.ref)
		if result != tt.expected {
			t.Errorf("ExtractPhaseFromRef(%q) = %q, want %q", tt.ref, result, tt.expected)
		}
	}
}

func TestExtractTypeFromRef(t *testing.T) {
	tests := []struct {
		ref      string
		expected types.ValidationType
	}{
		{"src/app.test.ts", types.ValTypeTest},
		{"src/app_test.go", types.ValTypeTest},
		{"scripts/test.bats", types.ValTypeTest},
		{"workflows/deploy.yaml", types.ValTypeAutomation},
		{"playbooks/e2e.yml", types.ValTypeAutomation},
		{"src/main.ts", types.ValTypeTest}, // default
	}

	for _, tt := range tests {
		result := ExtractTypeFromRef(tt.ref)
		if result != tt.expected {
			t.Errorf("ExtractTypeFromRef(%q) = %q, want %q", tt.ref, result, tt.expected)
		}
	}
}

func TestDeduplicateValidations(t *testing.T) {
	validations := []types.Validation{
		{Ref: "test/a.test.ts"},
		{Ref: "test/b.test.ts"},
		{Ref: "test/a.test.ts"}, // duplicate
		{WorkflowID: "wf-1"},
		{WorkflowID: "wf-1"}, // duplicate
		{Notes: "no key"},    // kept even without unique key
	}

	result := DeduplicateValidations(validations)

	if len(result) != 4 {
		t.Errorf("expected 4 unique validations, got: %d", len(result))
	}
}

func TestDeduplicateValidations_EdgeCases(t *testing.T) {
	// Empty slice
	result := DeduplicateValidations(nil)
	if result != nil {
		t.Error("nil input should return nil")
	}

	// Single item
	single := []types.Validation{{Ref: "test.ts"}}
	result = DeduplicateValidations(single)
	if len(result) != 1 {
		t.Error("single item should be unchanged")
	}
}

// --- ModuleIndex tests ---

func TestModuleIndex_GetRequirement_NilIndex(t *testing.T) {
	var idx *ModuleIndex
	if idx.GetRequirement("REQ-001") != nil {
		t.Error("nil index should return nil")
	}
}

func TestModuleIndex_GetModule_NilIndex(t *testing.T) {
	var idx *ModuleIndex
	if idx.GetModule("/path") != nil {
		t.Error("nil index should return nil")
	}
}

func TestModuleIndex_GetParent(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "PARENT", Children: []string{"CHILD"}},
			{ID: "CHILD"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	parent, ok := idx.GetParent("CHILD")
	if !ok {
		t.Error("expected to find parent")
	}
	if parent != "PARENT" {
		t.Errorf("unexpected parent: %s", parent)
	}

	_, ok = idx.GetParent("PARENT")
	if ok {
		t.Error("PARENT should have no parent")
	}
}

func TestModuleIndex_GetChildren(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "PARENT", Children: []string{"CHILD1", "CHILD2"}},
			{ID: "CHILD1"},
			{ID: "CHILD2"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	children := idx.GetChildren("PARENT")
	if len(children) != 2 {
		t.Errorf("expected 2 children, got: %d", len(children))
	}
}

func TestModuleIndex_GetDependencies(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", DependsOn: []string{"DEP-001", "DEP-002"}},
			{ID: "DEP-001"},
			{ID: "DEP-002"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	deps := idx.GetDependencies("REQ-001")
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got: %d", len(deps))
	}
}

func TestModuleIndex_GetBlockedBy(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "BLOCKER", Blocks: []string{"BLOCKED-1", "BLOCKED-2"}},
			{ID: "BLOCKED-1"},
			{ID: "BLOCKED-2"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	blocked := idx.GetBlockedBy("BLOCKER")
	if len(blocked) != 2 {
		t.Errorf("expected 2 blocked, got: %d", len(blocked))
	}
}

func TestModuleIndex_AllRequirements(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
			{ID: "REQ-002"},
		},
	}
	idx.AddModule(module)

	all := idx.AllRequirements()
	if len(all) != 2 {
		t.Errorf("expected 2 requirements, got: %d", len(all))
	}
}

func TestModuleIndex_AllRequirementIDs(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
			{ID: "REQ-002"},
		},
	}
	idx.AddModule(module)

	ids := idx.AllRequirementIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got: %d", len(ids))
	}
}

func TestModuleIndex_Counts(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test.json",
		ModuleName: "test",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
		},
	}
	idx.AddModule(module)

	if idx.RequirementCount() != 1 {
		t.Errorf("expected 1 requirement, got: %d", idx.RequirementCount())
	}
	if idx.ModuleCount() != 1 {
		t.Errorf("expected 1 module, got: %d", idx.ModuleCount())
	}
}

func TestModuleIndex_HasErrors(t *testing.T) {
	idx := NewModuleIndex()
	if idx.HasErrors() {
		t.Error("new index should not have errors")
	}

	// Add module with missing ID to trigger error
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: ""},
		},
	}
	idx.AddModule(module)

	if !idx.HasErrors() {
		t.Error("index should have errors after adding invalid requirement")
	}
}

func TestModuleIndex_IsAncestor(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "ROOT", Children: []string{"CHILD"}},
			{ID: "CHILD", Children: []string{"GRANDCHILD"}},
			{ID: "GRANDCHILD"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	if !idx.IsAncestor("ROOT", "GRANDCHILD") {
		t.Error("ROOT should be ancestor of GRANDCHILD")
	}
	if !idx.IsAncestor("ROOT", "CHILD") {
		t.Error("ROOT should be ancestor of CHILD")
	}
	if idx.IsAncestor("GRANDCHILD", "ROOT") {
		t.Error("GRANDCHILD should not be ancestor of ROOT")
	}
}

func TestModuleIndex_GetRootRequirements(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "ROOT1", Children: []string{"CHILD"}},
			{ID: "ROOT2"},
			{ID: "CHILD"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	roots := idx.GetRootRequirements()
	if len(roots) != 2 {
		t.Errorf("expected 2 roots, got: %d", len(roots))
	}
}

func TestModuleIndex_GetLeafRequirements(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "ROOT", Children: []string{"CHILD"}},
			{ID: "CHILD"},
			{ID: "LEAF"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	leaves := idx.GetLeafRequirements()
	if len(leaves) != 2 {
		t.Errorf("expected 2 leaves, got: %d", len(leaves))
	}
}

func TestModuleIndex_DetectCycles(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "A", Children: []string{"B"}},
			{ID: "B", Children: []string{"C"}},
			{ID: "C", Children: []string{"A"}}, // cycle: A -> B -> C -> A
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	cycles := idx.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected to detect cycle")
	}
}

func TestModuleIndex_DetectCycles_NoCycle(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "A", Children: []string{"B"}},
			{ID: "B", Children: []string{"C"}},
			{ID: "C"},
		},
	}
	idx.AddModule(module)
	idx.BuildHierarchy()

	cycles := idx.DetectCycles()
	if len(cycles) != 0 {
		t.Error("expected no cycles")
	}
}

func TestModuleIndex_FilterByStatus(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusPending},
			{ID: "REQ-002", Status: types.StatusComplete},
			{ID: "REQ-003", Status: types.StatusPending},
		},
	}
	idx.AddModule(module)

	pending := idx.FilterByStatus(types.StatusPending)
	if len(pending) != 2 {
		t.Errorf("expected 2 pending, got: %d", len(pending))
	}
}

func TestModuleIndex_FilterByCriticality(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Criticality: types.CriticalityP0},
			{ID: "REQ-002", Criticality: types.CriticalityP1},
			{ID: "REQ-003", Criticality: types.CriticalityP0},
		},
	}
	idx.AddModule(module)

	p0 := idx.FilterByCriticality(types.CriticalityP0)
	if len(p0) != 2 {
		t.Errorf("expected 2 P0, got: %d", len(p0))
	}
}

func TestModuleIndex_FilterCritical(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Criticality: types.CriticalityP0},
			{ID: "REQ-002", Criticality: types.CriticalityP1},
			{ID: "REQ-003", Criticality: types.CriticalityP2},
		},
	}
	idx.AddModule(module)

	critical := idx.FilterCritical()
	if len(critical) != 2 {
		t.Errorf("expected 2 critical (P0+P1), got: %d", len(critical))
	}
}

func TestModuleIndex_FindValidationsByRef(t *testing.T) {
	idx := NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Ref: "test/app.test.ts"},
				},
			},
		},
	}
	idx.AddModule(module)

	refs := idx.FindValidationsByRef("test/app.test.ts")
	if len(refs) != 1 {
		t.Errorf("expected 1 validation ref, got: %d", len(refs))
	}
}

func TestModuleIndex_DuplicateID_Error(t *testing.T) {
	idx := NewModuleIndex()
	module1 := &types.RequirementModule{
		FilePath: "/test1.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", SourceFile: "/test1.json"},
		},
	}
	module2 := &types.RequirementModule{
		FilePath: "/test2.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", SourceFile: "/test2.json"}, // duplicate
		},
	}

	idx.AddModule(module1)
	idx.AddModule(module2)

	if !idx.HasErrors() {
		t.Error("expected error for duplicate ID")
	}
}

// --- ParseFlexible tests ---

func TestParseFlexible_ValidModule(t *testing.T) {
	data := []byte(`{
		"_metadata": {"module": "test"},
		"requirements": [
			{
				"id": "REQ-001",
				"title": "Test",
				"status": "pending",
				"validation": [
					{"type": "test", "ref": "test.ts", "status": "implemented"}
				]
			}
		]
	}`)

	module, err := ParseFlexible(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(module.Requirements) != 1 {
		t.Errorf("expected 1 requirement, got: %d", len(module.Requirements))
	}
	if len(module.Requirements[0].Validations) != 1 {
		t.Errorf("expected 1 validation, got: %d", len(module.Requirements[0].Validations))
	}
}

func TestParseFlexible_ValidationsField(t *testing.T) {
	// Test that "validations" (plural) also works
	data := []byte(`{
		"requirements": [
			{
				"id": "REQ-001",
				"validations": [
					{"type": "test", "ref": "test.ts", "status": "planned"}
				]
			}
		]
	}`)

	module, err := ParseFlexible(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(module.Requirements[0].Validations) != 1 {
		t.Errorf("expected 1 validation from 'validations' field, got: %d", len(module.Requirements[0].Validations))
	}
}

func TestParseFlexible_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json`)

	_, err := ParseFlexible(data)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseFlexible_WithImports(t *testing.T) {
	data := []byte(`{
		"imports": ["01-core/module.json", "02-features/module.json"],
		"requirements": []
	}`)

	module, err := ParseFlexible(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(module.Imports) != 2 {
		t.Errorf("expected 2 imports, got: %d", len(module.Imports))
	}
}

// --- Parser edge cases ---

func TestParser_ParseAll_PartialErrors(t *testing.T) {
	reader := newMemReader()
	reader.files["/valid.json"] = []byte(`{"requirements": [{"id": "REQ-001"}]}`)
	reader.files["/invalid.json"] = []byte(`{invalid}`)

	parser := New(reader)

	files := []discovery.DiscoveredFile{
		{AbsolutePath: "/valid.json", RelativePath: "valid.json"},
		{AbsolutePath: "/invalid.json", RelativePath: "invalid.json"},
	}

	index, err := parser.ParseAll(context.Background(), files)

	// Should not return error - errors are collected in index
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 1 valid module and 1 error
	if len(index.Modules) != 1 {
		t.Errorf("expected 1 module, got: %d", len(index.Modules))
	}
	if len(index.Errors) != 1 {
		t.Errorf("expected 1 error, got: %d", len(index.Errors))
	}
}

func TestParser_ParseAll_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	reader.files["/test1.json"] = []byte(`{"requirements": []}`)
	reader.files["/test2.json"] = []byte(`{"requirements": []}`)

	parser := New(reader)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files := []discovery.DiscoveredFile{
		{AbsolutePath: "/test1.json"},
		{AbsolutePath: "/test2.json"},
	}

	_, err := parser.ParseAll(ctx, files)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}
