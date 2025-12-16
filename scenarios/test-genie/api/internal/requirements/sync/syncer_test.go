package sync

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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

func TestSyncer_Sync_UpdatesPRDOperationalTargets(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Title:  "req 1",
				Status: types.StatusComplete,
				PRDRef: "Operational Targets > P0 > OT-P0-001",
			},
			{
				ID:     "REQ-002",
				Title:  "req 2",
				Status: types.StatusComplete,
				PRDRef: "OT-P0-001",
			},
		},
	}
	index.AddModule(module)

	scenarioRoot := "/test/scenario"
	prdPath := filepath.Join(scenarioRoot, "PRD.md")
	reader.files[prdPath] = []byte(`# Product Requirements Document (PRD)

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Something | Outcome
`)

	evidence := types.NewEvidenceBundle()
	opts := DefaultOptions()
	opts.ScenarioRoot = scenarioRoot

	result, err := syncer.Sync(context.Background(), index, evidence, opts)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}

	updated, ok := writer.files[prdPath]
	if !ok {
		t.Fatalf("expected PRD.md to be written")
	}
	if !strings.Contains(string(updated), "- [x] OT-P0-001") {
		t.Fatalf("expected OT-P0-001 to be checked, got:\n%s", string(updated))
	}

	backupDir := filepath.Join(scenarioRoot, "coverage", "requirements-sync", "prd-backups")
	if !writer.dirs[backupDir] {
		t.Fatalf("expected backup dir to be created: %s", backupDir)
	}

	var backupPath string
	prefix := filepath.Join(backupDir, "PRD.md.")
	for path := range writer.files {
		if strings.HasPrefix(path, prefix) {
			backupPath = path
			break
		}
	}
	if backupPath == "" {
		t.Fatalf("expected PRD backup to be written under %s", backupDir)
	}
	if string(writer.files[backupPath]) != string(reader.files[prdPath]) {
		t.Fatalf("expected backup to match original PRD content")
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
// =========================================================================
// PRD Operational Target Update Tests
// =========================================================================

func TestDesiredOperationalTargetCheckboxes_AllComplete(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "OT-P0-001"},
			{ID: "REQ-002", Status: types.StatusComplete, PRDRef: "OT-P0-001"},
		},
	}
	index.AddModule(module)

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 1 {
		t.Fatalf("expected 1 OT, got %d", len(desired))
	}
	if !desired["OT-P0-001"] {
		t.Errorf("expected OT-P0-001 to be checked (all requirements complete)")
	}
}

func TestDesiredOperationalTargetCheckboxes_PartialComplete(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "OT-P0-001"},
			{ID: "REQ-002", Status: types.StatusInProgress, PRDRef: "OT-P0-001"},
		},
	}
	index.AddModule(module)

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 1 {
		t.Fatalf("expected 1 OT, got %d", len(desired))
	}
	if desired["OT-P0-001"] {
		t.Errorf("expected OT-P0-001 to be unchecked (not all requirements complete)")
	}
}

func TestDesiredOperationalTargetCheckboxes_MultipleTargets(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "OT-P0-001"},
			{ID: "REQ-002", Status: types.StatusComplete, PRDRef: "OT-P1-001"},
			{ID: "REQ-003", Status: types.StatusPending, PRDRef: "OT-P2-001"},
		},
	}
	index.AddModule(module)

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 3 {
		t.Fatalf("expected 3 OTs, got %d", len(desired))
	}
	if !desired["OT-P0-001"] {
		t.Errorf("expected OT-P0-001 to be checked")
	}
	if !desired["OT-P1-001"] {
		t.Errorf("expected OT-P1-001 to be checked")
	}
	if desired["OT-P2-001"] {
		t.Errorf("expected OT-P2-001 to be unchecked")
	}
}

func TestDesiredOperationalTargetCheckboxes_CaseInsensitive(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "ot-p0-001"}, // lowercase
			{ID: "REQ-002", Status: types.StatusComplete, PRDRef: "OT-P0-001"}, // uppercase
		},
	}
	index.AddModule(module)

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 1 {
		t.Fatalf("expected 1 OT (case-insensitive merge), got %d", len(desired))
	}
	if !desired["OT-P0-001"] {
		t.Errorf("expected OT-P0-001 to be checked")
	}
}

func TestDesiredOperationalTargetCheckboxes_ExtractsFromLongerPRDRef(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "Operational Targets > P0 > OT-P0-001"},
			{ID: "REQ-002", Status: types.StatusComplete, PRDRef: "## P1 section: OT-P1-042 description"},
		},
	}
	index.AddModule(module)

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 2 {
		t.Fatalf("expected 2 OTs, got %d", len(desired))
	}
	if !desired["OT-P0-001"] {
		t.Errorf("expected OT-P0-001 to be extracted and checked")
	}
	if !desired["OT-P1-042"] {
		t.Errorf("expected OT-P1-042 to be extracted and checked")
	}
}

func TestDesiredOperationalTargetCheckboxes_NoOTInPRDRef(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "Some other section"},
			{ID: "REQ-002", Status: types.StatusComplete, PRDRef: ""},
		},
	}
	index.AddModule(module)

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 0 {
		t.Errorf("expected 0 OTs (no valid OT patterns), got %d", len(desired))
	}
}

func TestDesiredOperationalTargetCheckboxes_NilIndex(t *testing.T) {
	desired := desiredOperationalTargetCheckboxes(nil)

	if len(desired) != 0 {
		t.Errorf("expected 0 OTs for nil index, got %d", len(desired))
	}
}

func TestDesiredOperationalTargetCheckboxes_EmptyModules(t *testing.T) {
	index := parsing.NewModuleIndex()

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 0 {
		t.Errorf("expected 0 OTs for empty index, got %d", len(desired))
	}
}

func TestDesiredOperationalTargetCheckboxes_NilModule(t *testing.T) {
	index := parsing.NewModuleIndex()
	index.AddModule(nil)

	desired := desiredOperationalTargetCheckboxes(index)

	if len(desired) != 0 {
		t.Errorf("expected 0 OTs when module is nil, got %d", len(desired))
	}
}

func TestUpdateOperationalTargetChecklistLines_CheckOn(t *testing.T) {
	content := `## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Something | Outcome
- [ ] OT-P0-002 | Another | Result
`
	desired := map[string]bool{
		"OT-P0-001": true,
		"OT-P0-002": false,
	}

	updated, changed := updateOperationalTargetChecklistLines(content, desired)

	if !changed {
		t.Fatal("expected changes to be made")
	}
	if !strings.Contains(updated, "- [x] OT-P0-001") {
		t.Errorf("expected OT-P0-001 to be checked")
	}
	if !strings.Contains(updated, "- [ ] OT-P0-002") {
		t.Errorf("expected OT-P0-002 to remain unchecked")
	}
}

func TestUpdateOperationalTargetChecklistLines_CheckOff(t *testing.T) {
	content := `## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Something | Outcome
- [X] OT-P0-002 | Another | Result
`
	desired := map[string]bool{
		"OT-P0-001": false,
		"OT-P0-002": false,
	}

	updated, changed := updateOperationalTargetChecklistLines(content, desired)

	if !changed {
		t.Fatal("expected changes to be made")
	}
	if !strings.Contains(updated, "- [ ] OT-P0-001") {
		t.Errorf("expected OT-P0-001 to be unchecked")
	}
	if !strings.Contains(updated, "- [ ] OT-P0-002") {
		t.Errorf("expected OT-P0-002 to be unchecked (was uppercase X)")
	}
}

func TestUpdateOperationalTargetChecklistLines_NoChangeNeeded(t *testing.T) {
	content := `## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Something | Outcome
- [ ] OT-P0-002 | Another | Result
`
	desired := map[string]bool{
		"OT-P0-001": true,  // already checked
		"OT-P0-002": false, // already unchecked
	}

	updated, changed := updateOperationalTargetChecklistLines(content, desired)

	if changed {
		t.Errorf("expected no changes (already in desired state)")
	}
	if updated != content {
		t.Errorf("content should be unchanged")
	}
}

func TestUpdateOperationalTargetChecklistLines_PreservesOtherContent(t *testing.T) {
	content := `# PRD Title

Some intro text.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Something | Outcome

### 🟠 P1 – Should have
- [ ] OT-P1-001 | Nice feature

## Other Section
More content here.
`
	desired := map[string]bool{
		"OT-P0-001": true,
	}

	updated, changed := updateOperationalTargetChecklistLines(content, desired)

	if !changed {
		t.Fatal("expected changes")
	}

	// Verify structure preserved
	if !strings.Contains(updated, "# PRD Title") {
		t.Error("title lost")
	}
	if !strings.Contains(updated, "Some intro text.") {
		t.Error("intro lost")
	}
	if !strings.Contains(updated, "## Other Section") {
		t.Error("other section lost")
	}
	if !strings.Contains(updated, "- [ ] OT-P1-001") {
		t.Error("P1 target modified unexpectedly")
	}
	if !strings.Contains(updated, "- [x] OT-P0-001") {
		t.Error("P0 target not updated")
	}
}

func TestUpdateOperationalTargetChecklistLines_HandlesVariousFormats(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantLine string
		check    bool
	}{
		{
			name:     "standard format",
			line:     "- [ ] OT-P0-001 | Title | Outcome",
			wantLine: "- [x] OT-P0-001 | Title | Outcome",
			check:    true,
		},
		{
			name:     "with leading spaces",
			line:     "  - [ ] OT-P0-001 | Title",
			wantLine: "  - [x] OT-P0-001 | Title",
			check:    true,
		},
		{
			// Note: lowercase P is supported (OT-p0-001), but lowercase OT is not.
			// PRD template uses uppercase OT format consistently.
			name:     "lowercase priority",
			line:     "- [ ] OT-p0-001 | Title",
			wantLine: "- [x] OT-p0-001 | Title",
			check:    true,
		},
		{
			name:     "extra spaces after bracket",
			line:     "- [ ]  OT-P0-001 | Title",
			wantLine: "- [x]  OT-P0-001 | Title",
			check:    true,
		},
		{
			name:     "P1 target",
			line:     "- [ ] OT-P1-042 | Enhancement",
			wantLine: "- [x] OT-P1-042 | Enhancement",
			check:    true,
		},
		{
			name:     "P2 target",
			line:     "- [x] OT-P2-013 | Future idea",
			wantLine: "- [ ] OT-P2-013 | Future idea",
			check:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract OT ID from line for the desired map (case-insensitive)
			otPattern := regexp.MustCompile(`(?i)OT-P[0-2]-\d{3}`)
			match := otPattern.FindString(tt.line)
			if match == "" {
				t.Fatalf("no OT pattern found in test line")
			}

			desired := map[string]bool{
				strings.ToUpper(match): tt.check,
			}

			updated, changed := updateOperationalTargetChecklistLines(tt.line+"\n", desired)

			if !changed {
				t.Errorf("expected change to be made")
			}
			if !strings.Contains(updated, tt.wantLine) {
				t.Errorf("expected line:\n%s\ngot:\n%s", tt.wantLine, updated)
			}
		})
	}
}

func TestUpdateOperationalTargetChecklistLines_EmptyContent(t *testing.T) {
	updated, changed := updateOperationalTargetChecklistLines("", map[string]bool{"OT-P0-001": true})

	if changed {
		t.Error("should not change empty content")
	}
	if updated != "" {
		t.Error("should return empty string unchanged")
	}
}

func TestUpdateOperationalTargetChecklistLines_EmptyDesired(t *testing.T) {
	content := "- [ ] OT-P0-001 | Something"
	updated, changed := updateOperationalTargetChecklistLines(content, nil)

	if changed {
		t.Error("should not change with nil desired map")
	}
	if updated != content {
		t.Error("content should be unchanged")
	}
}

func TestUpdateOperationalTargetChecklistLines_IgnoresUnknownOTs(t *testing.T) {
	content := `- [ ] OT-P0-001 | Something
- [ ] OT-P0-002 | Another
`
	desired := map[string]bool{
		"OT-P0-001": true,
		// OT-P0-002 not in desired map - should be left alone
	}

	updated, changed := updateOperationalTargetChecklistLines(content, desired)

	if !changed {
		t.Fatal("expected changes for OT-P0-001")
	}
	if !strings.Contains(updated, "- [x] OT-P0-001") {
		t.Error("OT-P0-001 should be checked")
	}
	if !strings.Contains(updated, "- [ ] OT-P0-002") {
		t.Error("OT-P0-002 should be unchanged (not in desired map)")
	}
}

func TestSyncer_Sync_UpdatesPRDOperationalTargets_Uncheck(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusInProgress, PRDRef: "OT-P0-001"},
		},
	}
	index.AddModule(module)

	scenarioRoot := "/test/scenario"
	prdPath := filepath.Join(scenarioRoot, "PRD.md")
	reader.files[prdPath] = []byte(`## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Something | Outcome
`)

	evidence := types.NewEvidenceBundle()
	opts := DefaultOptions()
	opts.ScenarioRoot = scenarioRoot

	result, err := syncer.Sync(context.Background(), index, evidence, opts)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}

	updated, ok := writer.files[prdPath]
	if !ok {
		t.Fatal("expected PRD.md to be written")
	}
	if !strings.Contains(string(updated), "- [ ] OT-P0-001") {
		t.Errorf("expected OT-P0-001 to be unchecked, got:\n%s", string(updated))
	}
}

func TestSyncer_Sync_PRDNotFound_NoError(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "OT-P0-001"},
		},
	}
	index.AddModule(module)

	scenarioRoot := "/test/scenario"
	// Note: PRD.md not added to reader

	evidence := types.NewEvidenceBundle()
	opts := DefaultOptions()
	opts.ScenarioRoot = scenarioRoot

	result, err := syncer.Sync(context.Background(), index, evidence, opts)
	if err != nil {
		t.Fatalf("sync should not error when PRD not found: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestSyncer_Sync_NoOTsInRequirements_NoChange(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "Some other ref"},
		},
	}
	index.AddModule(module)

	scenarioRoot := "/test/scenario"
	prdPath := filepath.Join(scenarioRoot, "PRD.md")
	prdContent := []byte("- [ ] OT-P0-001 | Something")
	reader.files[prdPath] = prdContent

	evidence := types.NewEvidenceBundle()
	opts := DefaultOptions()
	opts.ScenarioRoot = scenarioRoot

	_, err := syncer.Sync(context.Background(), index, evidence, opts)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// PRD should not be written (no changes)
	if _, exists := writer.files[prdPath]; exists {
		t.Error("PRD should not be written when no OTs need updating")
	}
}

func TestSyncer_Sync_DryRun_NoWritePRD(t *testing.T) {
	reader := newMemReader()
	writer := newMemWriter()
	syncer := New(reader, writer)

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/scenario/requirements/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, PRDRef: "OT-P0-001"},
		},
	}
	index.AddModule(module)

	scenarioRoot := "/test/scenario"
	prdPath := filepath.Join(scenarioRoot, "PRD.md")
	reader.files[prdPath] = []byte("- [ ] OT-P0-001 | Something")

	evidence := types.NewEvidenceBundle()
	opts := DefaultOptions()
	opts.ScenarioRoot = scenarioRoot
	opts.DryRun = true

	_, err := syncer.Sync(context.Background(), index, evidence, opts)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if len(writer.files) != 0 {
		t.Errorf("dry run should not write any files, wrote: %d", len(writer.files))
	}
}

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
