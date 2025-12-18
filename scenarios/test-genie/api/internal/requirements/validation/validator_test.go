package validation

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// memReader implements Reader for testing.
type memReader struct {
	files map[string][]byte
	dirs  map[string][]fs.DirEntry
}

func newMemReader() *memReader {
	return &memReader{
		files: make(map[string][]byte),
		dirs:  make(map[string][]fs.DirEntry),
	}
}

func (r *memReader) ReadFile(path string) ([]byte, error) {
	if data, ok := r.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) ReadDir(path string) ([]fs.DirEntry, error) {
	if entries, ok := r.dirs[path]; ok {
		return entries, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) Exists(path string) bool {
	_, hasFile := r.files[path]
	_, hasDir := r.dirs[path]
	return hasFile || hasDir
}

func (r *memReader) AddFile(path string, content []byte) {
	r.files[path] = content
}

// =============================================================================
// Validator Tests
// =============================================================================

func TestValidator_New(t *testing.T) {
	reader := newMemReader()
	v := New(reader)

	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestValidator_NewDefault(t *testing.T) {
	v := NewDefault()

	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestValidator_Validate_NilIndex(t *testing.T) {
	reader := newMemReader()
	v := New(reader)

	result := v.Validate(context.Background(), nil, "/test")

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected no issues for nil index, got: %d", len(result.Issues))
	}
}

func TestValidator_Validate_EmptyIndex(t *testing.T) {
	reader := newMemReader()
	v := New(reader)

	index := parsing.NewModuleIndex()
	result := v.Validate(context.Background(), index, "/test")

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected no issues for empty index, got: %d", len(result.Issues))
	}
}

func TestValidator_Validate_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	v := New(reader)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Test"},
		},
	}
	index.AddModule(module)

	result := v.Validate(ctx, index, "/test")

	// Result should still be valid, just may be incomplete
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestValidator_Validate_ValidRequirements(t *testing.T) {
	reader := newMemReader()
	v := New(reader)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "First Requirement", Status: types.StatusComplete},
			{ID: "REQ-002", Title: "Second Requirement", Status: types.StatusInProgress},
		},
	}
	index.AddModule(module)

	result := v.Validate(context.Background(), index, "/test")

	if result.HasErrors() {
		t.Errorf("expected no errors for valid requirements, got: %d", result.ErrorCount())
		for _, issue := range result.Errors() {
			t.Logf("  Issue: %s", issue.Message)
		}
	}
}

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()

	if len(rules) == 0 {
		t.Fatal("expected at least one default rule")
	}

	// Check that all expected rules are present
	ruleNames := make(map[string]bool)
	for _, rule := range rules {
		ruleNames[rule.Name()] = true
	}

	expectedRules := []string{
		"duplicate_id",
		"missing_id",
		"missing_title",
		"invalid_reference",
		"cycle_detection",
		"orphaned_child",
		"invalid_status",
	}

	for _, expected := range expectedRules {
		if !ruleNames[expected] {
			t.Errorf("expected rule %q in default rules", expected)
		}
	}
}

// =============================================================================
// DuplicateIDRule Tests
// =============================================================================

func TestDuplicateIDRule_Name(t *testing.T) {
	rule := &DuplicateIDRule{}
	if rule.Name() != "duplicate_id" {
		t.Errorf("expected name 'duplicate_id', got: %s", rule.Name())
	}
}

func TestDuplicateIDRule_Check_NoDuplicates(t *testing.T) {
	rule := &DuplicateIDRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
			{ID: "REQ-002"},
			{ID: "REQ-003"},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %d", len(issues))
	}
}

func TestDuplicateIDRule_Check_WithDuplicates(t *testing.T) {
	rule := &DuplicateIDRule{}

	index := parsing.NewModuleIndex()
	module1 := &types.RequirementModule{
		FilePath: "/test/module1.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
		},
	}
	module2 := &types.RequirementModule{
		FilePath: "/test/module2.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"}, // Duplicate
		},
	}
	index.AddModule(module1)
	index.AddModule(module2)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got: %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Severity != types.SeverityError {
		t.Errorf("expected error severity, got: %s", issues[0].Severity)
	}
}

func TestDuplicateIDRule_Check_CaseInsensitive(t *testing.T) {
	rule := &DuplicateIDRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
			{ID: "req-001"}, // Same ID different case
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue for case-insensitive duplicate, got: %d", len(issues))
	}
}

// =============================================================================
// MissingIDRule Tests
// =============================================================================

func TestMissingIDRule_Name(t *testing.T) {
	rule := &MissingIDRule{}
	if rule.Name() != "missing_id" {
		t.Errorf("expected name 'missing_id', got: %s", rule.Name())
	}
}

func TestMissingIDRule_Check_AllHaveIDs(t *testing.T) {
	rule := &MissingIDRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
			{ID: "REQ-002"},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %d", len(issues))
	}
}

func TestMissingIDRule_Check_MissingID(t *testing.T) {
	rule := &MissingIDRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
			{ID: ""},    // Missing ID
			{ID: "   "}, // Whitespace only
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 2 {
		t.Errorf("expected 2 issues for missing IDs, got: %d", len(issues))
	}
	for _, issue := range issues {
		if issue.Severity != types.SeverityError {
			t.Errorf("expected error severity, got: %s", issue.Severity)
		}
	}
}

// =============================================================================
// MissingTitleRule Tests
// =============================================================================

func TestMissingTitleRule_Name(t *testing.T) {
	rule := &MissingTitleRule{}
	if rule.Name() != "missing_title" {
		t.Errorf("expected name 'missing_title', got: %s", rule.Name())
	}
}

func TestMissingTitleRule_Check_AllHaveTitles(t *testing.T) {
	rule := &MissingTitleRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "First Requirement"},
			{ID: "REQ-002", Title: "Second Requirement"},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %d", len(issues))
	}
}

func TestMissingTitleRule_Check_MissingTitle(t *testing.T) {
	rule := &MissingTitleRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Has Title"},
			{ID: "REQ-002", Title: ""},    // Missing title
			{ID: "REQ-003", Title: "   "}, // Whitespace only
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 2 {
		t.Errorf("expected 2 issues for missing titles, got: %d", len(issues))
	}
	for _, issue := range issues {
		if issue.Severity != types.SeverityWarning {
			t.Errorf("expected warning severity, got: %s", issue.Severity)
		}
	}
}

// =============================================================================
// ParseRef Tests
// =============================================================================

func TestParseRef_SimpleFilePath(t *testing.T) {
	parsed := ParseRef("path/to/file.go")

	if parsed.FilePath != "path/to/file.go" {
		t.Errorf("expected FilePath 'path/to/file.go', got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "" {
		t.Errorf("expected empty Symbol, got: %s", parsed.Symbol)
	}
	if parsed.Raw != "path/to/file.go" {
		t.Errorf("expected Raw 'path/to/file.go', got: %s", parsed.Raw)
	}
}

func TestParseRef_WithSymbol(t *testing.T) {
	parsed := ParseRef("api/internal/handlers/handlers_test.go::TestHealthHandler")

	if parsed.FilePath != "api/internal/handlers/handlers_test.go" {
		t.Errorf("expected FilePath 'api/internal/handlers/handlers_test.go', got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "TestHealthHandler" {
		t.Errorf("expected Symbol 'TestHealthHandler', got: %s", parsed.Symbol)
	}
}

func TestParseRef_BatsTestWithSpaces(t *testing.T) {
	parsed := ParseRef("bas/cases/health-check/api/health_endpoint.bats::health endpoint returns 200 OK")

	if parsed.FilePath != "bas/cases/health-check/api/health_endpoint.bats" {
		t.Errorf("expected FilePath 'bas/cases/health-check/api/health_endpoint.bats', got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "health endpoint returns 200 OK" {
		t.Errorf("expected Symbol 'health endpoint returns 200 OK', got: %s", parsed.Symbol)
	}
}

func TestParseRef_EmptyRef(t *testing.T) {
	parsed := ParseRef("")

	if parsed.FilePath != "" {
		t.Errorf("expected empty FilePath, got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "" {
		t.Errorf("expected empty Symbol, got: %s", parsed.Symbol)
	}
	if parsed.Raw != "" {
		t.Errorf("expected empty Raw, got: %s", parsed.Raw)
	}
}

func TestParseRef_OnlySymbol(t *testing.T) {
	// Malformed: no file path before ::
	parsed := ParseRef("::TestFunction")

	if parsed.FilePath != "" {
		t.Errorf("expected empty FilePath for malformed ref, got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "TestFunction" {
		t.Errorf("expected Symbol 'TestFunction', got: %s", parsed.Symbol)
	}
}

func TestParseRef_EmptySymbol(t *testing.T) {
	// File path with trailing :: but no symbol
	parsed := ParseRef("file.go::")

	if parsed.FilePath != "file.go" {
		t.Errorf("expected FilePath 'file.go', got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "" {
		t.Errorf("expected empty Symbol, got: %s", parsed.Symbol)
	}
}

func TestParseRef_MultipleColonSeparators(t *testing.T) {
	// Multiple :: in the ref - should split at first occurrence
	parsed := ParseRef("file.go::Symbol::WithColons")

	if parsed.FilePath != "file.go" {
		t.Errorf("expected FilePath 'file.go', got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "Symbol::WithColons" {
		t.Errorf("expected Symbol 'Symbol::WithColons', got: %s", parsed.Symbol)
	}
}

func TestParseRef_SingleColon(t *testing.T) {
	// Single colon should NOT be treated as separator
	parsed := ParseRef("C:/path/to/file.go")

	if parsed.FilePath != "C:/path/to/file.go" {
		t.Errorf("expected FilePath 'C:/path/to/file.go', got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "" {
		t.Errorf("expected empty Symbol, got: %s", parsed.Symbol)
	}
}

func TestParseRef_WindowsPathWithSymbol(t *testing.T) {
	// Windows-style path with symbol
	parsed := ParseRef("C:/path/to/file.go::TestFunc")

	if parsed.FilePath != "C:/path/to/file.go" {
		t.Errorf("expected FilePath 'C:/path/to/file.go', got: %s", parsed.FilePath)
	}
	if parsed.Symbol != "TestFunc" {
		t.Errorf("expected Symbol 'TestFunc', got: %s", parsed.Symbol)
	}
}

func TestParseRef_LegacySingleColonFileSymbol(t *testing.T) {
	// Single colon after file extension SHOULD be treated as separator (legacy format)
	testCases := []struct {
		ref          string
		wantFilePath string
		wantSymbol   string
	}{
		{"api/internal/gc/gc_test.go:TestGCService", "api/internal/gc/gc_test.go", "TestGCService"},
		{"path/to/file.ts:someFunction", "path/to/file.ts", "someFunction"},
		{"test.bats:test_name", "test.bats", "test_name"}, // bats tests often use lowercase names
		{"file.py:TestClass", "file.py", "TestClass"},
		{"file.json:key", "file.json", "key"},
		// Multiple colons after extension - symbol includes the rest
		{"file.go:Test:SubTest", "file.go", "Test:SubTest"},
	}

	for _, tc := range testCases {
		t.Run(tc.ref, func(t *testing.T) {
			parsed := ParseRef(tc.ref)
			if parsed.FilePath != tc.wantFilePath {
				t.Errorf("FilePath: got %q, want %q", parsed.FilePath, tc.wantFilePath)
			}
			if parsed.Symbol != tc.wantSymbol {
				t.Errorf("Symbol: got %q, want %q", parsed.Symbol, tc.wantSymbol)
			}
		})
	}
}

func TestParseRef_SingleColonNotSeparator(t *testing.T) {
	// Cases where single colon should NOT be treated as file:symbol separator
	testCases := []struct {
		ref          string
		wantFilePath string
		wantSymbol   string
	}{
		// Windows drive letter
		{"C:/path/to/file.go", "C:/path/to/file.go", ""},
		// No extension before colon
		{"noextension:symbol", "noextension:symbol", ""},
		// Colon after something that doesn't look like file extension
		{"path/to/dir:something", "path/to/dir:something", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.ref, func(t *testing.T) {
			parsed := ParseRef(tc.ref)
			if parsed.FilePath != tc.wantFilePath {
				t.Errorf("FilePath: got %q, want %q", parsed.FilePath, tc.wantFilePath)
			}
			if parsed.Symbol != tc.wantSymbol {
				t.Errorf("Symbol: got %q, want %q", parsed.Symbol, tc.wantSymbol)
			}
		})
	}
}

// =============================================================================
// InvalidReferenceRule Tests
// =============================================================================

func TestInvalidReferenceRule_Name(t *testing.T) {
	rule := &InvalidReferenceRule{}
	if rule.Name() != "invalid_reference" {
		t.Errorf("expected name 'invalid_reference', got: %s", rule.Name())
	}
}

func TestInvalidReferenceRule_Check_NoScenarioRoot(t *testing.T) {
	rule := &InvalidReferenceRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "", Reader: nil}
	issues := rule.Check(context.Background(), rctx)

	// Should return no issues when no scenario root is provided
	if len(issues) != 0 {
		t.Errorf("expected no issues when scenario root is empty, got: %d", len(issues))
	}
}

func TestInvalidReferenceRule_Check_ValidReference(t *testing.T) {
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	reader.AddFile("/test/scenario/test.spec.ts", []byte("test content"))

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 0 {
		t.Errorf("expected no issues for valid reference, got: %d", len(issues))
	}
}

func TestInvalidReferenceRule_Check_InvalidReference(t *testing.T) {
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	// Don't add the file - simulating missing reference

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "nonexistent.spec.ts"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue for invalid reference, got: %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Severity != types.SeverityWarning {
		t.Errorf("expected warning severity, got: %s", issues[0].Severity)
	}
}

func TestInvalidReferenceRule_Check_ManualValidationSkipped(t *testing.T) {
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	// Don't add the file

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeManual, Ref: "some/path"}, // Manual - should be skipped
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Manual validations should be skipped
	if len(issues) != 0 {
		t.Errorf("expected no issues for manual validation, got: %d", len(issues))
	}
}

func TestInvalidReferenceRule_Check_EmptyRef(t *testing.T) {
	rule := &InvalidReferenceRule{}
	reader := newMemReader()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: ""}, // Empty ref - should be skipped
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Empty ref should be skipped
	if len(issues) != 0 {
		t.Errorf("expected no issues for empty ref, got: %d", len(issues))
	}
}

func TestInvalidReferenceRule_Check_AlternativePaths(t *testing.T) {
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	// File exists in ui/ subdirectory
	reader.AddFile("/test/scenario/ui/test.spec.ts", []byte("test content"))

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Should find file in alternative path
	if len(issues) != 0 {
		t.Errorf("expected no issues when file found in alternative path, got: %d", len(issues))
	}
}

func TestInvalidReferenceRule_Check_RefWithSymbol(t *testing.T) {
	// Test that refs with :: symbol separator work correctly
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	reader.AddFile("/test/scenario/api/internal/handlers/handlers_test.go", []byte("package handlers\n\nfunc TestHealthHandler(t *testing.T) {}"))

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					// This ref includes a test function name after ::
					{Type: types.ValTypeTest, Ref: "api/internal/handlers/handlers_test.go::TestHealthHandler"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Should find file (ignoring the ::TestHealthHandler suffix)
	if len(issues) != 0 {
		t.Errorf("expected no issues for ref with symbol suffix, got: %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.Message)
		}
	}
}

func TestInvalidReferenceRule_Check_BatsRefWithTestName(t *testing.T) {
	// Test BATS test refs with test names containing spaces
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	reader.AddFile("/test/scenario/bas/cases/health-check/api/health_endpoint.bats", []byte("@test \"health endpoint returns 200 OK\" {}"))

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "bas/cases/health-check/api/health_endpoint.bats::health endpoint returns 200 OK"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Should find file (ignoring the test name suffix with spaces)
	if len(issues) != 0 {
		t.Errorf("expected no issues for BATS ref with test name, got: %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.Message)
		}
	}
}

func TestInvalidReferenceRule_Check_RefWithSymbol_FileMissing(t *testing.T) {
	// Test that missing files are still detected even with :: symbol
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	// Don't add the file

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "nonexistent.go::TestFunction"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Should report missing file
	if len(issues) != 1 {
		t.Errorf("expected 1 issue for missing file with symbol ref, got: %d", len(issues))
	}
	if len(issues) > 0 {
		// Error message should show the file path, not the full ref with ::
		if !strings.Contains(issues[0].Message, "nonexistent.go") {
			t.Errorf("expected message to contain file path 'nonexistent.go', got: %s", issues[0].Message)
		}
		// Should NOT contain the symbol name in the error
		if strings.Contains(issues[0].Message, "TestFunction") {
			t.Errorf("expected message to NOT contain symbol 'TestFunction', got: %s", issues[0].Message)
		}
	}
}

func TestInvalidReferenceRule_Check_MalformedRef_NoFilePath(t *testing.T) {
	// Test malformed ref with no file path (only ::symbol)
	rule := &InvalidReferenceRule{}
	reader := newMemReader()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "::TestFunction"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Should report malformed ref
	if len(issues) != 1 {
		t.Errorf("expected 1 issue for malformed ref, got: %d", len(issues))
	}
	if len(issues) > 0 {
		if !strings.Contains(issues[0].Message, "malformed") {
			t.Errorf("expected message to indicate malformed ref, got: %s", issues[0].Message)
		}
	}
}

func TestInvalidReferenceRule_Check_MultipleRefsWithSymbols(t *testing.T) {
	// Test multiple refs in same requirement, some with symbols, some without
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	reader.AddFile("/test/scenario/api/handlers_test.go", []byte("test"))
	reader.AddFile("/test/scenario/api/service_test.go", []byte("test"))
	// Don't add missing.go

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "api/handlers_test.go::TestCreate"},      // exists
					{Type: types.ValTypeTest, Ref: "api/service_test.go"},                   // exists, no symbol
					{Type: types.ValTypeTest, Ref: "api/missing.go::TestMissing"},           // missing
					{Type: types.ValTypeManual, Ref: "api/also_missing.go::ManualTest"},     // manual - skipped
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Should only report 1 issue (missing.go), manual is skipped
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got: %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.Message)
		}
	}
}

func TestInvalidReferenceRule_Check_AlternativePathsWithSymbol(t *testing.T) {
	// Test that alternative paths work with symbol refs
	rule := &InvalidReferenceRule{}
	reader := newMemReader()
	// File exists in api/ subdirectory
	reader.AddFile("/test/scenario/api/handlers_test.go", []byte("test"))

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					// ref doesn't include api/ prefix, but file is in api/
					{Type: types.ValTypeTest, Ref: "handlers_test.go::TestFunction"},
				},
			},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index, ScenarioRoot: "/test/scenario", Reader: reader}
	issues := rule.Check(context.Background(), rctx)

	// Should find file in alternative api/ path
	if len(issues) != 0 {
		t.Errorf("expected no issues when file found in alternative path with symbol ref, got: %d", len(issues))
	}
}

// =============================================================================
// CycleDetectionRule Tests
// =============================================================================

func TestCycleDetectionRule_Name(t *testing.T) {
	rule := &CycleDetectionRule{}
	if rule.Name() != "cycle_detection" {
		t.Errorf("expected name 'cycle_detection', got: %s", rule.Name())
	}
}

func TestCycleDetectionRule_Check_NoCycles(t *testing.T) {
	rule := &CycleDetectionRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Children: []string{"REQ-002"}},
			{ID: "REQ-002", Children: []string{"REQ-003"}},
			{ID: "REQ-003"},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %d", len(issues))
	}
}

func TestCycleDetectionRule_Check_WithCycle(t *testing.T) {
	rule := &CycleDetectionRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-A", Children: []string{"REQ-B"}},
			{ID: "REQ-B", Children: []string{"REQ-C"}},
			{ID: "REQ-C", Children: []string{"REQ-A"}}, // Creates cycle
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) == 0 {
		t.Error("expected at least 1 issue for cycle")
	}
	if len(issues) > 0 && issues[0].Severity != types.SeverityError {
		t.Errorf("expected error severity, got: %s", issues[0].Severity)
	}
}

// =============================================================================
// OrphanedChildRule Tests
// =============================================================================

func TestOrphanedChildRule_Name(t *testing.T) {
	rule := &OrphanedChildRule{}
	if rule.Name() != "orphaned_child" {
		t.Errorf("expected name 'orphaned_child', got: %s", rule.Name())
	}
}

func TestOrphanedChildRule_Check_NoOrphans(t *testing.T) {
	rule := &OrphanedChildRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Children: []string{"REQ-002"}},
			{ID: "REQ-002"},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %d", len(issues))
	}
}

func TestOrphanedChildRule_Check_OrphanedChild(t *testing.T) {
	rule := &OrphanedChildRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Children: []string{"REQ-NONEXISTENT"}},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue for orphaned child, got: %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Severity != types.SeverityError {
		t.Errorf("expected error severity, got: %s", issues[0].Severity)
	}
}

func TestOrphanedChildRule_Check_OrphanedDependency(t *testing.T) {
	rule := &OrphanedChildRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", DependsOn: []string{"REQ-NONEXISTENT"}},
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue for orphaned dependency, got: %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Severity != types.SeverityWarning {
		t.Errorf("expected warning severity, got: %s", issues[0].Severity)
	}
}

// =============================================================================
// InvalidStatusRule Tests
// =============================================================================

func TestInvalidStatusRule_Name(t *testing.T) {
	rule := &InvalidStatusRule{}
	if rule.Name() != "invalid_status" {
		t.Errorf("expected name 'invalid_status', got: %s", rule.Name())
	}
}

func TestInvalidStatusRule_Check_ValidStatuses(t *testing.T) {
	rule := &InvalidStatusRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusPending},
			{ID: "REQ-002", Status: types.StatusPlanned},
			{ID: "REQ-003", Status: types.StatusInProgress},
			{ID: "REQ-004", Status: types.StatusComplete},
			{ID: "REQ-005", Status: types.StatusNotImplemented},
			{ID: "REQ-006", Status: ""}, // Empty is valid (defaults)
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 0 {
		t.Errorf("expected no issues for valid statuses, got: %d", len(issues))
	}
}

func TestInvalidStatusRule_Check_InvalidStatus(t *testing.T) {
	rule := &InvalidStatusRule{}

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: "invalid_status"},
			{ID: "REQ-002", Status: "done"}, // Not valid
			{ID: "REQ-003", Status: "todo"}, // Not valid
		},
	}
	index.AddModule(module)

	rctx := RuleContext{Index: index}
	issues := rule.Check(context.Background(), rctx)

	if len(issues) != 3 {
		t.Errorf("expected 3 issues for invalid statuses, got: %d", len(issues))
	}
	for _, issue := range issues {
		if issue.Severity != types.SeverityWarning {
			t.Errorf("expected warning severity, got: %s", issue.Severity)
		}
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestValidator_IntegrationTest(t *testing.T) {
	reader := newMemReader()
	reader.AddFile("/test/scenario/test.spec.ts", []byte("test content"))

	v := New(reader)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID:       "REQ-001",
				Title:    "First Requirement",
				Status:   types.StatusComplete,
				Children: []string{"REQ-002"},
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts"},
				},
			},
			{
				ID:     "REQ-002",
				Title:  "Second Requirement",
				Status: types.StatusInProgress,
			},
			// Add issues to verify detection
			{
				ID:     "REQ-003",
				Title:  "",        // Missing title
				Status: "invalid", // Invalid status
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "nonexistent.ts"}, // Invalid reference
				},
			},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	result := v.Validate(context.Background(), index, "/test/scenario")

	if !result.HasWarnings() {
		t.Error("expected warnings")
	}

	// Should have warnings for: missing title, invalid status, invalid reference
	if result.WarningCount() < 3 {
		t.Errorf("expected at least 3 warnings, got: %d", result.WarningCount())
		for _, issue := range result.Issues {
			t.Logf("  Issue: [%s] %s - %s", issue.Severity, issue.RequirementID, issue.Message)
		}
	}
}

func TestValidator_AllRulesRun(t *testing.T) {
	reader := newMemReader()
	v := New(reader)

	// Create an index with issues for each rule type
	index := parsing.NewModuleIndex()
	module1 := &types.RequirementModule{
		FilePath: "/test/module1.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Test"},
		},
	}
	module2 := &types.RequirementModule{
		FilePath: "/test/module2.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Duplicate"},                    // Duplicate ID
			{ID: "", Title: "Missing ID"},                          // Missing ID
			{ID: "REQ-002", Title: ""},                             // Missing title
			{ID: "REQ-003", Status: "bad"},                         // Invalid status
			{ID: "REQ-004", Children: []string{"REQ-NONEXISTENT"}}, // Orphaned child
		},
	}
	index.AddModule(module1)
	index.AddModule(module2)
	index.BuildHierarchy()

	result := v.Validate(context.Background(), index, "")

	// Should have multiple issues from different rules
	if len(result.Issues) < 4 {
		t.Errorf("expected at least 4 issues from different rules, got: %d", len(result.Issues))
	}

	// Verify we have both errors and warnings
	if !result.HasErrors() {
		t.Error("expected at least one error")
	}
	if !result.HasWarnings() {
		t.Error("expected at least one warning")
	}
}
