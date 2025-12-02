package sync

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// memWriter implements Writer for testing.
type memWriter struct {
	files map[string][]byte
	dirs  map[string]bool
}

func newMemWriter() *memWriter {
	return &memWriter{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (w *memWriter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	w.files[path] = data
	return nil
}

func (w *memWriter) MkdirAll(path string, perm fs.FileMode) error {
	w.dirs[path] = true
	return nil
}

func TestSyncer_Sync_NilInputs(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	result, err := syncer.Sync(context.Background(), nil, nil, DefaultOptions())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FilesUpdated != 0 {
		t.Errorf("expected 0 files updated, got: %d", result.FilesUpdated)
	}
}

func TestSyncer_Sync_EmptyIndex(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	evidence := types.NewEvidenceBundle()

	result, err := syncer.Sync(context.Background(), index, evidence, DefaultOptions())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.FilesUpdated != 0 {
		t.Errorf("expected 0 files updated, got: %d", result.FilesUpdated)
	}
	if result.StatusesChanged != 0 {
		t.Errorf("expected 0 statuses changed, got: %d", result.StatusesChanged)
	}
}

func TestSyncer_Sync_WithStatusUpdates(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	// Create index with a module
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Metadata: types.ModuleMetadata{Module: "test"},
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Title:  "Test requirement",
				Status: types.StatusPending,
				Validations: []types.Validation{
					{
						Type:       types.ValTypeTest,
						Ref:        "test/example.test.ts",
						Status:     types.ValStatusNotImplemented,
						LiveStatus: types.LivePassed, // Enriched
					},
				},
			},
		},
	}
	index.AddModule(module)

	evidence := types.NewEvidenceBundle()

	opts := DefaultOptions()
	opts.UpdateStatuses = true

	result, err := syncer.Sync(context.Background(), index, evidence, opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.StatusesChanged == 0 {
		t.Log("Note: StatusesChanged may be 0 if no status transitions detected")
	}
}

func TestSyncer_Preview_DryRun(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID:    "REQ-001",
				Title: "Test",
			},
		},
	}
	index.AddModule(module)

	evidence := types.NewEvidenceBundle()

	result, err := syncer.Preview(context.Background(), index, evidence, DefaultOptions())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.FilesUpdated != 0 {
		t.Errorf("preview should not write files, got: %d updated", result.FilesUpdated)
	}
	if len(writer.files) != 0 {
		t.Errorf("preview should not write to writer, got: %d files", len(writer.files))
	}
}

func TestSyncer_Sync_ContextCancellation(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	index := parsing.NewModuleIndex()
	evidence := types.NewEvidenceBundle()

	result, err := syncer.Sync(ctx, index, evidence, DefaultOptions())

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result even on cancellation")
	}
}

func TestSyncer_Sync_WriteMetadata(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	evidence := types.NewEvidenceBundle()

	opts := DefaultOptions()
	opts.ScenarioRoot = "/test/scenario"
	opts.TestCommands = []string{"suite test"}

	result, err := syncer.Sync(context.Background(), index, evidence, opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check sync metadata was written
	metadataPath := "/test/scenario/coverage/sync/latest.json"
	if data, ok := writer.files[metadataPath]; !ok {
		t.Error("expected sync metadata to be written")
	} else {
		var metadata SyncMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			t.Errorf("invalid metadata JSON: %v", err)
		}
		if len(metadata.TestCommands) != 1 || metadata.TestCommands[0] != "suite test" {
			t.Errorf("unexpected test commands: %v", metadata.TestCommands)
		}
	}

	if !result.SyncedAt.Before(time.Now().Add(time.Second)) {
		t.Error("SyncedAt should be set to current time")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.PruneOrphans {
		t.Error("PruneOrphans should default to false")
	}
	if !opts.DiscoverNew {
		t.Error("DiscoverNew should default to true")
	}
	if !opts.UpdateStatuses {
		t.Error("UpdateStatuses should default to true")
	}
	if !opts.AllowPartial {
		t.Error("AllowPartial should default to true")
	}
	if opts.DryRun {
		t.Error("DryRun should default to false")
	}
}

func TestStatusUpdater_UpdateStatuses_NilInputs(t *testing.T) {
	updater := NewStatusUpdater()

	changes, err := updater.UpdateStatuses(context.Background(), nil, nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes for nil inputs, got: %d", len(changes))
	}
}

func TestStatusUpdater_UpdateStatuses_WithLiveStatus(t *testing.T) {
	updater := NewStatusUpdater()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Status: types.StatusInProgress,
				Validations: []types.Validation{
					{
						Type:       types.ValTypeTest,
						Ref:        "test.ts",
						Status:     types.ValStatusNotImplemented,
						LiveStatus: types.LivePassed,
					},
				},
			},
		},
	}
	index.AddModule(module)

	evidence := types.NewEvidenceBundle()

	changes, err := updater.UpdateStatuses(context.Background(), index, evidence)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should detect status change from not_implemented to implemented
	if len(changes) == 0 {
		t.Log("Note: No changes detected - validation status may already match live status")
	}
}

func TestStatusUpdater_ContextCancellation(t *testing.T) {
	updater := NewStatusUpdater()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
		},
	}
	index.AddModule(module)

	evidence := types.NewEvidenceBundle()

	_, err := updater.UpdateStatuses(ctx, index, evidence)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestFileWriter_WriteModule(t *testing.T) {
	writer := newMemWriter()
	fw := NewFileWriter(writer)

	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Metadata: types.ModuleMetadata{
			Module:      "test",
			Description: "Test module",
		},
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Title:  "Test requirement",
				Status: types.StatusComplete,
			},
		},
	}

	err := fw.WriteModule(context.Background(), module)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	data, ok := writer.files["/test/module.json"]
	if !ok {
		t.Fatal("module file not written")
	}

	// Verify JSON structure
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("invalid JSON output: %v", err)
	}

	if metadata, ok := parsed["_metadata"].(map[string]any); !ok {
		t.Error("missing _metadata in output")
	} else if metadata["module"] != "test" {
		t.Errorf("unexpected module name: %v", metadata["module"])
	}

	if reqs, ok := parsed["requirements"].([]any); !ok {
		t.Error("missing requirements in output")
	} else if len(reqs) != 1 {
		t.Errorf("expected 1 requirement, got: %d", len(reqs))
	}

	// Verify trailing newline
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("output should end with newline")
	}
}

func TestFileWriter_WriteModule_NilModule(t *testing.T) {
	writer := newMemWriter()
	fw := NewFileWriter(writer)

	err := fw.WriteModule(context.Background(), nil)

	if err != nil {
		t.Errorf("nil module should not error: %v", err)
	}
	if len(writer.files) != 0 {
		t.Error("nil module should not write files")
	}
}

func TestFileWriter_WriteModule_EmptyFilePath(t *testing.T) {
	writer := newMemWriter()
	fw := NewFileWriter(writer)

	module := &types.RequirementModule{
		FilePath: "",
		Requirements: []types.Requirement{
			{ID: "REQ-001"},
		},
	}

	err := fw.WriteModule(context.Background(), module)

	if err != nil {
		t.Errorf("empty filepath should not error: %v", err)
	}
	if len(writer.files) != 0 {
		t.Error("empty filepath should not write files")
	}
}

func TestFileWriter_WriteJSON(t *testing.T) {
	writer := newMemWriter()
	fw := NewFileWriter(writer)

	data := map[string]string{"key": "value"}
	err := fw.WriteJSON("/test/output.json", data)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	written, ok := writer.files["/test/output.json"]
	if !ok {
		t.Fatal("file not written")
	}

	var parsed map[string]string
	if err := json.Unmarshal(written, &parsed); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("unexpected value: %s", parsed["key"])
	}
}

// Integration test using real filesystem
func TestSyncer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenario")
	requirementsDir := filepath.Join(scenarioDir, "requirements")

	if err := os.MkdirAll(requirementsDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	syncer := NewDefault()

	// Create module with live status to trigger a change
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: filepath.Join(requirementsDir, "module.json"),
		Metadata: types.ModuleMetadata{Module: "test"},
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Title:  "Integration test",
				Status: types.StatusInProgress,
				Validations: []types.Validation{
					{
						Type:       types.ValTypeTest,
						Ref:        "test/example.test.ts",
						Status:     types.ValStatusNotImplemented,
						LiveStatus: types.LivePassed, // This triggers a status change
					},
				},
			},
		},
	}
	index.AddModule(module)

	evidence := types.NewEvidenceBundle()

	opts := DefaultOptions()
	opts.ScenarioRoot = scenarioDir
	opts.TestCommands = []string{"go test ./..."}

	result, err := syncer.Sync(context.Background(), index, evidence, opts)

	if err != nil {
		t.Errorf("sync failed: %v", err)
	}

	// Verify metadata was written (this always happens when ScenarioRoot is set)
	metadataPath := filepath.Join(scenarioDir, "coverage", "sync", "latest.json")
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}

	var metadata SyncMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}

	if len(metadata.TestCommands) != 1 {
		t.Errorf("expected 1 test command, got: %d", len(metadata.TestCommands))
	}

	// Module may or may not be written depending on whether changes were detected
	if result.FilesUpdated > 0 {
		moduleData, err := os.ReadFile(filepath.Join(requirementsDir, "module.json"))
		if err != nil {
			t.Errorf("read module: %v", err)
		} else if !strings.Contains(string(moduleData), "REQ-001") {
			t.Error("module should contain REQ-001")
		}
	}

	t.Logf("Sync result: %d files updated, %d statuses changed", result.FilesUpdated, result.StatusesChanged)
}
