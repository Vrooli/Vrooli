package parsing

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/requirements/discovery"
	reqparsing "test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// mockParser implements reqparsing.Parser for testing.
type mockParser struct {
	index *reqparsing.ModuleIndex
	err   error
}

func (m *mockParser) Parse(ctx context.Context, filePath string) (*types.RequirementModule, error) {
	return nil, m.err
}

func (m *mockParser) ParseAll(ctx context.Context, files []discovery.DiscoveredFile) (*reqparsing.ModuleIndex, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.index != nil {
		return m.index, nil
	}
	return reqparsing.NewModuleIndex(), nil
}

func TestParse_Success(t *testing.T) {
	index := reqparsing.NewModuleIndex()
	index.Modules = []*types.RequirementModule{
		{
			ModuleName: "01-core",
			FilePath:   "/test/requirements/01-core/module.json",
			Requirements: []types.Requirement{
				{ID: "REQ-001", Title: "First Requirement"},
				{ID: "REQ-002", Title: "Second Requirement"},
			},
		},
	}
	for i := range index.Modules[0].Requirements {
		req := &index.Modules[0].Requirements[i]
		index.ByID[req.ID] = req
	}

	mock := &mockParser{index: index}
	v := NewWithParser(mock, io.Discard)

	files := []DiscoveredFile{
		{AbsolutePath: "/test/requirements/01-core/module.json", RelativePath: "01-core/module.json", ModuleDir: "01-core"},
	}

	result := v.Parse(context.Background(), files)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ModuleCount != 1 {
		t.Errorf("expected 1 module, got %d", result.ModuleCount)
	}
	if result.RequirementCount != 2 {
		t.Errorf("expected 2 requirements, got %d", result.RequirementCount)
	}
	if result.Index == nil {
		t.Error("expected index to be set")
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestParse_SystemError(t *testing.T) {
	mock := &mockParser{err: errors.New("read error")}
	v := NewWithParser(mock, io.Discard)

	files := []DiscoveredFile{
		{AbsolutePath: "/test/module.json"},
	}

	result := v.Parse(context.Background(), files)

	if result.Success {
		t.Fatal("expected failure for system error")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

func TestParse_WithParsingWarnings(t *testing.T) {
	index := reqparsing.NewModuleIndex()
	index.Modules = []*types.RequirementModule{
		{
			ModuleName: "01-core",
			Requirements: []types.Requirement{
				{ID: "REQ-001", Title: "Test"},
			},
		},
	}
	index.Errors = []error{
		errors.New("warning: deprecated field used"),
	}

	mock := &mockParser{index: index}
	v := NewWithParser(mock, io.Discard)

	files := []DiscoveredFile{
		{AbsolutePath: "/test/module.json"},
	}

	result := v.Parse(context.Background(), files)

	// Should succeed even with warnings
	if !result.Success {
		t.Fatalf("expected success with warnings, got error: %v", result.Error)
	}

	// Should include warning observation
	hasWarning := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning observation for parsing issue")
	}
}

func TestParse_EmptyFiles(t *testing.T) {
	mock := &mockParser{index: reqparsing.NewModuleIndex()}
	v := NewWithParser(mock, io.Discard)

	var files []DiscoveredFile

	result := v.Parse(context.Background(), files)

	if !result.Success {
		t.Fatalf("expected success for empty files, got error: %v", result.Error)
	}
	if result.ModuleCount != 0 {
		t.Errorf("expected 0 modules, got %d", result.ModuleCount)
	}
	if result.RequirementCount != 0 {
		t.Errorf("expected 0 requirements, got %d", result.RequirementCount)
	}
}

// Integration test using real filesystem
func TestParse_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements", "01-core")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	moduleContent := `{
		"requirements": [
			{"id": "REQ-001", "title": "First", "status": "draft", "criticality": "p1", "validation": [{"type": "manual", "ref": "docs"}]},
			{"id": "REQ-002", "title": "Second", "status": "complete", "criticality": "p2", "validation": [{"type": "test", "ref": "test.go"}]}
		]
	}`
	modulePath := filepath.Join(reqDir, "module.json")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0o644); err != nil {
		t.Fatalf("failed to write module: %v", err)
	}

	v := New(io.Discard)
	files := []DiscoveredFile{
		{AbsolutePath: modulePath, RelativePath: "01-core/module.json", ModuleDir: "01-core"},
	}

	result := v.Parse(context.Background(), files)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ModuleCount != 1 {
		t.Errorf("expected 1 module, got %d", result.ModuleCount)
	}
	if result.RequirementCount != 2 {
		t.Errorf("expected 2 requirements, got %d", result.RequirementCount)
	}
}

// Ensure validator satisfies the interface at compile time
var _ Validator = (*validator)(nil)

// Ensure mock satisfies the requirements/parsing interface
var _ reqparsing.Parser = (*mockParser)(nil)

// Benchmarks

func BenchmarkParse(b *testing.B) {
	index := reqparsing.NewModuleIndex()
	index.Modules = []*types.RequirementModule{
		{
			ModuleName: "01-core",
			Requirements: []types.Requirement{
				{ID: "REQ-001", Title: "First"},
				{ID: "REQ-002", Title: "Second"},
			},
		},
	}

	mock := &mockParser{index: index}
	v := NewWithParser(mock, io.Discard)

	files := []DiscoveredFile{
		{AbsolutePath: "/test/module.json", ModuleDir: "01-core"},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Parse(ctx, files)
	}
}
