package discovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func (r *memReader) Stat(path string) (fs.FileInfo, error) {
	if _, ok := r.files[path]; ok {
		return &memFileInfo{name: filepath.Base(path), isDir: false}, nil
	}
	if _, ok := r.dirs[path]; ok {
		return &memFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (r *memReader) Exists(path string) bool {
	_, hasFile := r.files[path]
	_, hasDir := r.dirs[path]
	return hasFile || hasDir
}

type memFileInfo struct {
	name  string
	isDir bool
}

func (f *memFileInfo) Name() string       { return f.name }
func (f *memFileInfo) Size() int64        { return 0 }
func (f *memFileInfo) Mode() fs.FileMode  { return 0644 }
func (f *memFileInfo) ModTime() time.Time { return time.Time{} }
func (f *memFileInfo) IsDir() bool        { return f.isDir }
func (f *memFileInfo) Sys() any           { return nil }

type memDirEntry struct {
	name  string
	isDir bool
}

func (e *memDirEntry) Name() string               { return e.name }
func (e *memDirEntry) IsDir() bool                { return e.isDir }
func (e *memDirEntry) Type() fs.FileMode          { return 0 }
func (e *memDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestDiscoverer_Discover_NoRequirementsDir(t *testing.T) {
	reader := newMemReader()
	discoverer := New(reader)

	files, err := discoverer.Discover(context.Background(), "/test/scenario")

	if err == nil {
		t.Error("expected error for missing requirements dir")
	}
	if err != ErrNoRequirementsDir {
		t.Errorf("expected ErrNoRequirementsDir, got: %v", err)
	}
	if files != nil {
		t.Error("expected nil files")
	}
}

func TestDiscoverer_Discover_EmptyRequirementsDir(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{}
	discoverer := New(reader)

	files, err := discoverer.Discover(context.Background(), "/test/scenario")

	// No index.json, so it falls back to scanning - should find no files
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got: %d", len(files))
	}
}

func TestDiscoverer_Discover_WithIndex(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{
		&memDirEntry{name: "index.json", isDir: false},
	}
	reader.files["/test/scenario/requirements/index.json"] = []byte(`{
		"imports": ["01-core/module.json"]
	}`)
	reader.dirs["/test/scenario/requirements/01-core"] = []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	}
	reader.files["/test/scenario/requirements/01-core/module.json"] = []byte(`{
		"_metadata": {"module": "core"},
		"requirements": []
	}`)

	discoverer := New(reader)
	files, err := discoverer.Discover(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should find index.json + module.json
	if len(files) != 2 {
		t.Errorf("expected 2 files, got: %d", len(files))
	}

	// Verify index.json is marked as index
	foundIndex := false
	for _, f := range files {
		if f.IsIndex && filepath.Base(f.AbsolutePath) == "index.json" {
			foundIndex = true
		}
	}
	if !foundIndex {
		t.Error("expected to find index.json marked as IsIndex")
	}
}

func TestDiscoverer_Discover_NestedImports(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{}
	reader.files["/test/scenario/requirements/index.json"] = []byte(`{
		"imports": ["01-core/index.json"]
	}`)
	reader.files["/test/scenario/requirements/01-core/index.json"] = []byte(`{
		"imports": ["nested/module.json"]
	}`)
	reader.files["/test/scenario/requirements/01-core/nested/module.json"] = []byte(`{
		"requirements": []
	}`)

	discoverer := New(reader)
	files, err := discoverer.Discover(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should find: root index, 01-core/index, nested/module
	if len(files) != 3 {
		t.Errorf("expected 3 files, got: %d", len(files))
	}
}

func TestDiscoverer_Discover_MissingImport(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{}
	reader.files["/test/scenario/requirements/index.json"] = []byte(`{
		"imports": ["missing/module.json"]
	}`)

	discoverer := New(reader)
	files, err := discoverer.Discover(context.Background(), "/test/scenario")

	// Missing imports are logged but don't fail
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should only find index.json
	if len(files) != 1 {
		t.Errorf("expected 1 file (index only), got: %d", len(files))
	}
}

func TestDiscoverer_Discover_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{}
	reader.files["/test/scenario/requirements/index.json"] = []byte(`{"imports": []}`)

	discoverer := New(reader)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := discoverer.Discover(ctx, "/test/scenario")

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestDiscoveredFile_Fields(t *testing.T) {
	file := DiscoveredFile{
		AbsolutePath: "/test/requirements/module.json",
		RelativePath: "module.json",
		IsIndex:      false,
		ModuleDir:    "requirements",
	}

	if file.AbsolutePath != "/test/requirements/module.json" {
		t.Errorf("unexpected AbsolutePath: %s", file.AbsolutePath)
	}
	if file.IsIndex {
		t.Error("IsIndex should be false")
	}
}

// Integration test using real filesystem
func TestDiscoverer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenario")
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	moduleDir := filepath.Join(requirementsDir, "01-core")

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("create dirs: %v", err)
	}

	// Write index.json
	indexData := []byte(`{"imports": ["01-core/module.json"]}`)
	if err := os.WriteFile(filepath.Join(requirementsDir, "index.json"), indexData, 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	// Write module.json
	moduleData := []byte(`{"_metadata": {"module": "core"}, "requirements": [{"id": "REQ-001"}]}`)
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), moduleData, 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	discoverer := NewDefault()
	files, err := discoverer.Discover(context.Background(), scenarioDir)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got: %d", len(files))
	}

	t.Logf("Discovered %d files", len(files))
	for _, f := range files {
		t.Logf("  - %s (IsIndex: %v, ModuleDir: %s)", f.RelativePath, f.IsIndex, f.ModuleDir)
	}
}

// --- Error type tests ---

func TestParseError_Error(t *testing.T) {
	err := &ParseError{FilePath: "/test/file.json", Err: ErrInvalidJSON}

	msg := err.Error()
	if msg != "/test/file.json: invalid JSON in requirement file" {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestParseError_Unwrap(t *testing.T) {
	inner := ErrInvalidJSON
	err := &ParseError{FilePath: "/test/file.json", Err: inner}

	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestDiscoveryError_Error(t *testing.T) {
	err := &DiscoveryError{Path: "/test/dir", Err: ErrMissingReference}

	msg := err.Error()
	if msg != "discovery error at /test/dir: validation references non-existent file" {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestDiscoveryError_Unwrap(t *testing.T) {
	inner := ErrMissingReference
	err := &DiscoveryError{Path: "/test/dir", Err: inner}

	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

// --- Scanner tests ---

func TestScanner_ScanRecursive(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{
		&memDirEntry{name: "01-core", isDir: true},
		&memDirEntry{name: "index.json", isDir: false},
	}
	reader.dirs["/requirements/01-core"] = []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	}
	reader.files["/requirements/index.json"] = []byte(`{}`)
	reader.files["/requirements/01-core/module.json"] = []byte(`{}`)

	scanner := NewScanner(reader)
	files, err := scanner.ScanRecursive(context.Background(), "/requirements")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got: %d", len(files))
	}
}

func TestScanner_ScanRecursive_SkipsHiddenAndSpecialDirs(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{
		&memDirEntry{name: ".hidden", isDir: true},
		&memDirEntry{name: "node_modules", isDir: true},
		&memDirEntry{name: "valid", isDir: true},
	}
	reader.dirs["/requirements/.hidden"] = []fs.DirEntry{
		&memDirEntry{name: "secret.json", isDir: false},
	}
	reader.dirs["/requirements/node_modules"] = []fs.DirEntry{
		&memDirEntry{name: "dep.json", isDir: false},
	}
	reader.dirs["/requirements/valid"] = []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	}
	reader.files["/requirements/.hidden/secret.json"] = []byte(`{}`)
	reader.files["/requirements/node_modules/dep.json"] = []byte(`{}`)
	reader.files["/requirements/valid/module.json"] = []byte(`{}`)

	scanner := NewScanner(reader)
	files, err := scanner.ScanRecursive(context.Background(), "/requirements")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only find valid/module.json
	if len(files) != 1 {
		t.Errorf("expected 1 file (hidden and node_modules skipped), got: %d", len(files))
	}
}

func TestScanner_ScanRecursive_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{}

	scanner := NewScanner(reader)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := scanner.ScanRecursive(ctx, "/requirements")

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestScanner_ScanImmediate(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{
		&memDirEntry{name: "index.json", isDir: false},
		&memDirEntry{name: "subdir", isDir: true},
		&memDirEntry{name: "module.json", isDir: false},
	}
	reader.files["/requirements/index.json"] = []byte(`{}`)
	reader.files["/requirements/module.json"] = []byte(`{}`)

	scanner := NewScanner(reader)
	files, err := scanner.ScanImmediate(context.Background(), "/requirements")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only find immediate files, not recurse into subdir
	if len(files) != 2 {
		t.Errorf("expected 2 files, got: %d", len(files))
	}
}

func TestScanner_ScanImmediate_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{}

	scanner := NewScanner(reader)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := scanner.ScanImmediate(ctx, "/requirements")

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestScanner_FindModules(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{
		&memDirEntry{name: "01-core", isDir: true},
		&memDirEntry{name: "02-features", isDir: true},
		&memDirEntry{name: "no-module", isDir: true},
	}
	reader.files["/requirements/01-core/module.json"] = []byte(`{}`)
	reader.files["/requirements/02-features/module.json"] = []byte(`{}`)
	// no-module dir has no module.json

	scanner := NewScanner(reader)
	modules, err := scanner.FindModules(context.Background(), "/requirements")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got: %d", len(modules))
	}
}

func TestScanner_FindModules_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{}

	scanner := NewScanner(reader)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := scanner.FindModules(ctx, "/requirements")

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestScanner_FileExists(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/exists.json"] = []byte(`{}`)

	scanner := NewScanner(reader)

	if !scanner.FileExists("/test/exists.json") {
		t.Error("FileExists should return true for existing file")
	}
	if scanner.FileExists("/test/missing.json") {
		t.Error("FileExists should return false for missing file")
	}
}

func TestScanner_GetFileInfo(t *testing.T) {
	reader := newMemReader()
	reader.files["/test/file.json"] = []byte(`{}`)

	scanner := NewScanner(reader)

	info, err := scanner.GetFileInfo("/test/file.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name() != "file.json" {
		t.Errorf("unexpected name: %s", info.Name())
	}
}

// --- Helper function tests ---

func TestIsRequirementFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"module.json", true},
		{"index.json", true},
		{"requirements.json", true},
		{"package.json", false},
		{"tsconfig.json", false},
		{"jsconfig.json", false},
		{"package-lock.json", false},
		{"composer.json", false},
		{"readme.txt", false},
		{"test.ts", false},
	}

	for _, tt := range tests {
		result := isRequirementFile(tt.name)
		if result != tt.expected {
			t.Errorf("isRequirementFile(%q) = %v, want %v", tt.name, result, tt.expected)
		}
	}
}

func TestIsIndexFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"index.json", true},
		{"INDEX.JSON", true},
		{"Index.JSON", true},
		{"module.json", false},
		{"index.ts", false},
	}

	for _, tt := range tests {
		result := isIndexFile(tt.name)
		if result != tt.expected {
			t.Errorf("isIndexFile(%q) = %v, want %v", tt.name, result, tt.expected)
		}
	}
}

func TestIsSkippableDir(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"node_modules", true},
		{"NODE_MODULES", true},
		{"vendor", true},
		{"dist", true},
		{"build", true},
		{"coverage", true},
		{"__pycache__", true},
		{".git", true},
		{"target", true},
		{"bin", true},
		{"obj", true},
		{"src", false},
		{"lib", false},
		{"requirements", false},
	}

	for _, tt := range tests {
		result := isSkippableDir(tt.name)
		if result != tt.expected {
			t.Errorf("isSkippableDir(%q) = %v, want %v", tt.name, result, tt.expected)
		}
	}
}

func TestExtractModuleDir(t *testing.T) {
	tests := []struct {
		relDir   string
		expected string
	}{
		{"", ""},
		{"01-core", "01-core"},
		{"01-core/nested", "01-core"},
		{"deeply/nested/path", "deeply"},
	}

	for _, tt := range tests {
		result := extractModuleDir(tt.relDir)
		if result != tt.expected {
			t.Errorf("extractModuleDir(%q) = %q, want %q", tt.relDir, result, tt.expected)
		}
	}
}

func TestExtractPriorityPrefix(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		{"", 0},
		{"01-core", 1},
		{"02-features", 2},
		{"10-advanced", 10},
		{"99-experimental", 99},
		{"no-prefix", 1000},
		{"core", 1000},
		{"abc-test", 1000},
	}

	for _, tt := range tests {
		result := extractPriorityPrefix(tt.name)
		if result != tt.expected {
			t.Errorf("extractPriorityPrefix(%q) = %d, want %d", tt.name, result, tt.expected)
		}
	}
}

func TestSortByPriority(t *testing.T) {
	files := []DiscoveredFile{
		{RelativePath: "03-last/module.json", ModuleDir: "03-last", IsIndex: false},
		{RelativePath: "index.json", ModuleDir: "", IsIndex: true},
		{RelativePath: "01-first/module.json", ModuleDir: "01-first", IsIndex: false},
		{RelativePath: "02-middle/module.json", ModuleDir: "02-middle", IsIndex: false},
	}

	SortByPriority(files)

	// Index should come first
	if !files[0].IsIndex {
		t.Error("index file should be first")
	}
	// Then sorted by numeric prefix
	if files[1].ModuleDir != "01-first" {
		t.Errorf("expected 01-first second, got: %s", files[1].ModuleDir)
	}
	if files[2].ModuleDir != "02-middle" {
		t.Errorf("expected 02-middle third, got: %s", files[2].ModuleDir)
	}
	if files[3].ModuleDir != "03-last" {
		t.Errorf("expected 03-last last, got: %s", files[3].ModuleDir)
	}
}

func TestComparePriority(t *testing.T) {
	index := DiscoveredFile{IsIndex: true, ModuleDir: "", RelativePath: "index.json"}
	module1 := DiscoveredFile{IsIndex: false, ModuleDir: "01-core", RelativePath: "01-core/module.json"}
	module2 := DiscoveredFile{IsIndex: false, ModuleDir: "02-feat", RelativePath: "02-feat/module.json"}

	// Index comes before module
	if comparePriority(index, module1) >= 0 {
		t.Error("index should come before module")
	}
	if comparePriority(module1, index) <= 0 {
		t.Error("module should come after index")
	}

	// Lower numbered module comes first
	if comparePriority(module1, module2) >= 0 {
		t.Error("01-core should come before 02-feat")
	}

	// Same priority - compare by path
	sameModule := DiscoveredFile{IsIndex: false, ModuleDir: "01-core", RelativePath: "01-core/other.json"}
	result := comparePriority(module1, sameModule)
	if result == 0 {
		t.Error("different paths should not be equal")
	}
}

func TestDiscoverWithScanner(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/requirements"] = []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	}
	reader.files["/requirements/module.json"] = []byte(`{}`)

	files, err := DiscoverWithScanner(context.Background(), reader, "/requirements")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got: %d", len(files))
	}
}

func TestDiscoverer_Discover_InvalidIndexJSON(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{}
	reader.files["/test/scenario/requirements/index.json"] = []byte(`{invalid json`)

	discoverer := New(reader)
	_, err := discoverer.Discover(context.Background(), "/test/scenario")

	if err == nil {
		t.Error("expected error for invalid index.json")
	}
	// Should be a ParseError
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected ParseError, got: %T", err)
	}
}

func TestDiscoverer_Discover_CircularImports(t *testing.T) {
	reader := newMemReader()
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{}
	// Create circular import: index -> a -> b -> a
	reader.files["/test/scenario/requirements/index.json"] = []byte(`{"imports": ["a.json"]}`)
	reader.files["/test/scenario/requirements/a.json"] = []byte(`{"imports": ["b.json"]}`)
	reader.files["/test/scenario/requirements/b.json"] = []byte(`{"imports": ["a.json"]}`)

	discoverer := New(reader)
	files, err := discoverer.Discover(context.Background(), "/test/scenario")

	// Should not error - circular imports are handled via visited map
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should find index + a + b (no duplicates)
	if len(files) != 3 {
		t.Errorf("expected 3 files, got: %d", len(files))
	}
}

func TestDiscoverer_Discover_FallbackScanWithSubdirs(t *testing.T) {
	reader := newMemReader()
	// No index.json - should fall back to scanning
	reader.dirs["/test/scenario/requirements"] = []fs.DirEntry{
		&memDirEntry{name: "01-core", isDir: true},
		&memDirEntry{name: "orphan.json", isDir: false},
	}
	reader.files["/test/scenario/requirements/orphan.json"] = []byte(`{}`)
	reader.files["/test/scenario/requirements/01-core/module.json"] = []byte(`{}`)

	discoverer := New(reader)
	files, err := discoverer.Discover(context.Background(), "/test/scenario")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should find orphan.json + 01-core/module.json
	if len(files) != 2 {
		t.Errorf("expected 2 files, got: %d", len(files))
	}
}
