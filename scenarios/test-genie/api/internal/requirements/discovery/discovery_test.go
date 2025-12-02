package discovery

import (
	"context"
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
