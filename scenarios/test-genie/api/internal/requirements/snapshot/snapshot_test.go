package snapshot

import (
	"context"
	"encoding/json"
	"io/fs"
	"testing"

	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

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

func (w *memWriter) GetFile(path string) ([]byte, bool) {
	data, ok := w.files[path]
	return data, ok
}

// =============================================================================
// Builder Tests
// =============================================================================

func TestBuilder_New(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("expected non-nil builder")
	}
}

func TestBuilder_Build_ContextCancellation(t *testing.T) {
	b := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := b.Build(ctx, parsing.NewModuleIndex(), enrichment.Summary{})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestBuilder_Build_EmptyIndex(t *testing.T) {
	b := New()

	snapshot, err := b.Build(context.Background(), parsing.NewModuleIndex(), enrichment.Summary{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if snapshot.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got: %s", snapshot.Version)
	}
	if snapshot.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
}

func TestBuilder_Build_NilIndex(t *testing.T) {
	b := New()

	snapshot, err := b.Build(context.Background(), nil, enrichment.Summary{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(snapshot.Modules) != 0 {
		t.Errorf("expected 0 modules, got: %d", len(snapshot.Modules))
	}
}

func TestBuilder_Build_Summary(t *testing.T) {
	b := New()

	summary := enrichment.Summary{
		Total:          10,
		CriticalityGap: 2,
		ByDeclaredStatus: map[types.DeclaredStatus]int{
			types.StatusComplete:   5,
			types.StatusInProgress: 3,
			types.StatusPending:    2,
		},
		ByLiveStatus: map[types.LiveStatus]int{
			types.LivePassed: 6,
			types.LiveFailed: 2,
		},
		ValidationStats: enrichment.ValidationStats{
			Total:       8,
			Implemented: 5,
		},
	}

	snapshot, err := b.Build(context.Background(), parsing.NewModuleIndex(), summary)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if snapshot.Summary.TotalRequirements != 10 {
		t.Errorf("expected total 10, got: %d", snapshot.Summary.TotalRequirements)
	}
	if snapshot.Summary.TotalValidations != 8 {
		t.Errorf("expected validations 8, got: %d", snapshot.Summary.TotalValidations)
	}
	if snapshot.Summary.CriticalGap != 2 {
		t.Errorf("expected critical gap 2, got: %d", snapshot.Summary.CriticalGap)
	}
	// Completion rate = 5/10 = 50%
	if snapshot.Summary.CompletionRate != 50 {
		t.Errorf("expected completion rate 50, got: %.1f", snapshot.Summary.CompletionRate)
	}
	// Pass rate = 6/(6+2) = 75%
	if snapshot.Summary.PassRate != 75 {
		t.Errorf("expected pass rate 75, got: %.1f", snapshot.Summary.PassRate)
	}
}

func TestBuilder_Build_ModuleSnapshots(t *testing.T) {
	b := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete},
			{ID: "REQ-002", Status: types.StatusComplete},
			{ID: "REQ-003", Status: types.StatusInProgress},
			{ID: "REQ-004", Status: types.StatusPending},
			{ID: "REQ-005", Status: types.StatusPlanned},
		},
	}
	index.AddModule(module)

	snapshot, err := b.Build(context.Background(), index, enrichment.Summary{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(snapshot.Modules) != 1 {
		t.Fatalf("expected 1 module, got: %d", len(snapshot.Modules))
	}

	mod := snapshot.Modules[0]
	if mod.Name != "test-module" {
		t.Errorf("expected module name 'test-module', got: %s", mod.Name)
	}
	if mod.FilePath != "/test/module.json" {
		t.Errorf("expected file path, got: %s", mod.FilePath)
	}
	if mod.Total != 5 {
		t.Errorf("expected total 5, got: %d", mod.Total)
	}
	if mod.Complete != 2 {
		t.Errorf("expected complete 2, got: %d", mod.Complete)
	}
	if mod.InProgress != 1 {
		t.Errorf("expected in_progress 1, got: %d", mod.InProgress)
	}
	if mod.Pending != 2 {
		t.Errorf("expected pending 2 (pending + planned), got: %d", mod.Pending)
	}
	// Completion rate = 2/5 = 40%
	if mod.CompletionRate != 40 {
		t.Errorf("expected completion rate 40, got: %.1f", mod.CompletionRate)
	}
}

func TestBuilder_Build_OperationalTargets(t *testing.T) {
	b := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Feature A Part 1", PRDRef: "PRD-001", Status: types.StatusComplete, Criticality: types.CriticalityP0},
			{ID: "REQ-002", Title: "Feature A Part 2", PRDRef: "PRD-001", Status: types.StatusComplete},
			{ID: "REQ-003", Title: "Feature B Part 1", PRDRef: "PRD-002", Status: types.StatusInProgress, Criticality: types.CriticalityP1},
			{ID: "REQ-004", Title: "Feature B Part 2", PRDRef: "PRD-002", Status: types.StatusPending},
			{ID: "REQ-005", Title: "No PRD Ref", Status: types.StatusComplete}, // No PRD ref
		},
	}
	index.AddModule(module)

	snapshot, err := b.Build(context.Background(), index, enrichment.Summary{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(snapshot.OperationalTargets) != 2 {
		t.Fatalf("expected 2 operational targets, got: %d", len(snapshot.OperationalTargets))
	}

	// Find each target
	var prd001, prd002 *OperationalTarget
	for i := range snapshot.OperationalTargets {
		ot := &snapshot.OperationalTargets[i]
		if ot.ID == "PRD-001" {
			prd001 = ot
		} else if ot.ID == "PRD-002" {
			prd002 = ot
		}
	}

	if prd001 == nil || prd002 == nil {
		t.Fatal("expected both PRD-001 and PRD-002 targets")
	}

	// PRD-001: Both complete = 100%
	if prd001.Status != "complete" {
		t.Errorf("expected PRD-001 status 'complete', got: %s", prd001.Status)
	}
	if prd001.CompletionRate != 100 {
		t.Errorf("expected PRD-001 completion 100, got: %.1f", prd001.CompletionRate)
	}
	if len(prd001.RequirementIDs) != 2 {
		t.Errorf("expected 2 requirement IDs, got: %d", len(prd001.RequirementIDs))
	}
	if prd001.Priority != "P0" {
		t.Errorf("expected priority P0, got: %s", prd001.Priority)
	}

	// PRD-002: 0/2 complete = 0%, pending status (code only sets in_progress when complete > 0)
	if prd002.Status != "pending" {
		t.Errorf("expected PRD-002 status 'pending', got: %s", prd002.Status)
	}
	if prd002.CompletionRate != 0 {
		t.Errorf("expected PRD-002 completion 0, got: %.1f", prd002.CompletionRate)
	}
	if prd002.Priority != "P1" {
		t.Errorf("expected priority P1, got: %s", prd002.Priority)
	}
}

func TestBuilder_Build_OperationalTargets_AllPending(t *testing.T) {
	b := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", PRDRef: "PRD-001", Status: types.StatusPending},
			{ID: "REQ-002", PRDRef: "PRD-001", Status: types.StatusPending},
		},
	}
	index.AddModule(module)

	snapshot, err := b.Build(context.Background(), index, enrichment.Summary{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(snapshot.OperationalTargets) != 1 {
		t.Fatalf("expected 1 operational target, got: %d", len(snapshot.OperationalTargets))
	}

	// All pending = "pending" status
	if snapshot.OperationalTargets[0].Status != "pending" {
		t.Errorf("expected status 'pending', got: %s", snapshot.OperationalTargets[0].Status)
	}
}

func TestBuilder_Build_MultipleModules(t *testing.T) {
	b := New()

	index := parsing.NewModuleIndex()
	module1 := &types.RequirementModule{
		FilePath:   "/test/module1.json",
		ModuleName: "module1",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete},
		},
	}
	module2 := &types.RequirementModule{
		FilePath:   "/test/module2.json",
		ModuleName: "module2",
		Requirements: []types.Requirement{
			{ID: "REQ-002", Status: types.StatusInProgress},
			{ID: "REQ-003", Status: types.StatusInProgress},
		},
	}
	index.AddModule(module1)
	index.AddModule(module2)

	snapshot, err := b.Build(context.Background(), index, enrichment.Summary{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(snapshot.Modules) != 2 {
		t.Errorf("expected 2 modules, got: %d", len(snapshot.Modules))
	}
}

func TestBuilder_Build_EmptyModule(t *testing.T) {
	b := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:     "/test/module.json",
		ModuleName:   "empty-module",
		Requirements: []types.Requirement{},
	}
	index.AddModule(module)

	snapshot, err := b.Build(context.Background(), index, enrichment.Summary{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(snapshot.Modules) != 1 {
		t.Fatalf("expected 1 module, got: %d", len(snapshot.Modules))
	}

	mod := snapshot.Modules[0]
	if mod.Total != 0 {
		t.Errorf("expected total 0, got: %d", mod.Total)
	}
	if mod.CompletionRate != 0 {
		t.Errorf("expected completion rate 0, got: %.1f", mod.CompletionRate)
	}
}

// =============================================================================
// WriteSnapshot Tests
// =============================================================================

func TestWriteSnapshot(t *testing.T) {
	writer := newMemWriter()

	snapshot := &Snapshot{
		Version: "1.0.0",
		Summary: SnapshotSummary{
			TotalRequirements: 5,
			TotalValidations:  3,
			CompletionRate:    60,
			PassRate:          80,
		},
		Modules: []ModuleSnapshot{
			{Name: "test", Total: 5, Complete: 3},
		},
	}

	err := WriteSnapshot(writer, "/output/snapshot.json", snapshot)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	data, ok := writer.GetFile("/output/snapshot.json")
	if !ok {
		t.Fatal("expected file to be written")
	}

	// Verify it's valid JSON
	var parsed Snapshot
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	if parsed.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got: %s", parsed.Version)
	}
	if parsed.Summary.TotalRequirements != 5 {
		t.Errorf("expected 5 requirements, got: %d", parsed.Summary.TotalRequirements)
	}
}

func TestWriteSnapshot_Formatting(t *testing.T) {
	writer := newMemWriter()

	snapshot := &Snapshot{
		Version: "1.0.0",
	}

	err := WriteSnapshot(writer, "/output/snapshot.json", snapshot)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	data, _ := writer.GetFile("/output/snapshot.json")

	// Should be indented (pretty-printed)
	if !containsIndentation(data) {
		t.Error("expected indented JSON output")
	}

	// Should end with newline
	if len(data) > 0 && data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func containsIndentation(data []byte) bool {
	// Check if there are spaces after a newline (indicating indentation)
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\n' && data[i+1] == ' ' {
			return true
		}
	}
	return false
}

// =============================================================================
// Snapshot Struct Tests
// =============================================================================

func TestSnapshot_JSON(t *testing.T) {
	snapshot := &Snapshot{
		ScenarioName: "test-scenario",
		Version:      "1.0.0",
		Summary: SnapshotSummary{
			TotalRequirements: 10,
			TotalValidations:  5,
			CompletionRate:    50,
			PassRate:          80,
			CriticalGap:       2,
		},
		OperationalTargets: []OperationalTarget{
			{
				ID:             "PRD-001",
				Title:          "Feature",
				Priority:       "P0",
				Status:         "complete",
				RequirementIDs: []string{"REQ-001", "REQ-002"},
				CompletionRate: 100,
			},
		},
		Modules: []ModuleSnapshot{
			{
				Name:           "core",
				FilePath:       "/test/core.json",
				Total:          5,
				Complete:       3,
				InProgress:     1,
				Pending:        1,
				CompletionRate: 60,
			},
		},
	}

	// Serialize
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Errorf("failed to marshal: %v", err)
	}

	// Deserialize
	var parsed Snapshot
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("failed to unmarshal: %v", err)
	}

	// Verify fields
	if parsed.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got: %s", parsed.ScenarioName)
	}
	if len(parsed.OperationalTargets) != 1 {
		t.Errorf("expected 1 OT, got: %d", len(parsed.OperationalTargets))
	}
	if len(parsed.Modules) != 1 {
		t.Errorf("expected 1 module, got: %d", len(parsed.Modules))
	}
}

func TestSnapshotSummary_Defaults(t *testing.T) {
	summary := SnapshotSummary{}

	if summary.TotalRequirements != 0 {
		t.Errorf("expected default 0, got: %d", summary.TotalRequirements)
	}
	if summary.CompletionRate != 0 {
		t.Errorf("expected default 0, got: %.1f", summary.CompletionRate)
	}
}

func TestOperationalTarget_JSON(t *testing.T) {
	ot := OperationalTarget{
		ID:             "PRD-001",
		Title:          "Test Feature",
		Priority:       "P0",
		Status:         "in_progress",
		RequirementIDs: []string{"REQ-001", "REQ-002"},
		CompletionRate: 50,
	}

	data, err := json.Marshal(ot)
	if err != nil {
		t.Errorf("failed to marshal: %v", err)
	}

	var parsed OperationalTarget
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("failed to unmarshal: %v", err)
	}

	if len(parsed.RequirementIDs) != 2 {
		t.Errorf("expected 2 requirement IDs, got: %d", len(parsed.RequirementIDs))
	}
}

func TestModuleSnapshot_JSON(t *testing.T) {
	mod := ModuleSnapshot{
		Name:           "test-module",
		FilePath:       "/path/to/module.json",
		Total:          10,
		Complete:       5,
		InProgress:     3,
		Pending:        2,
		CompletionRate: 50,
	}

	data, err := json.Marshal(mod)
	if err != nil {
		t.Errorf("failed to marshal: %v", err)
	}

	var parsed ModuleSnapshot
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("failed to unmarshal: %v", err)
	}

	if parsed.Name != "test-module" {
		t.Errorf("expected name 'test-module', got: %s", parsed.Name)
	}
	if parsed.Total != 10 {
		t.Errorf("expected total 10, got: %d", parsed.Total)
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestBuilder_FullIntegration(t *testing.T) {
	b := New()

	index := parsing.NewModuleIndex()
	module1 := &types.RequirementModule{
		FilePath:   "/test/core/module.json",
		ModuleName: "core",
		Requirements: []types.Requirement{
			{ID: "CORE-001", Title: "Core Feature 1", PRDRef: "PRD-CORE", Status: types.StatusComplete, Criticality: types.CriticalityP0},
			{ID: "CORE-002", Title: "Core Feature 2", PRDRef: "PRD-CORE", Status: types.StatusComplete},
			{ID: "CORE-003", Title: "Core Feature 3", Status: types.StatusInProgress},
		},
	}
	module2 := &types.RequirementModule{
		FilePath:   "/test/ext/module.json",
		ModuleName: "extensions",
		Requirements: []types.Requirement{
			{ID: "EXT-001", Title: "Extension 1", PRDRef: "PRD-EXT", Status: types.StatusPending, Criticality: types.CriticalityP1},
		},
	}
	index.AddModule(module1)
	index.AddModule(module2)

	summary := enrichment.Summary{
		Total:          4,
		CriticalityGap: 1,
		ByDeclaredStatus: map[types.DeclaredStatus]int{
			types.StatusComplete:   2,
			types.StatusInProgress: 1,
			types.StatusPending:    1,
		},
		ByLiveStatus: map[types.LiveStatus]int{
			types.LivePassed: 2,
			types.LiveFailed: 0,
		},
		ValidationStats: enrichment.ValidationStats{
			Total:       3,
			Implemented: 2,
		},
	}

	snapshot, err := b.Build(context.Background(), index, summary)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify summary
	if snapshot.Summary.TotalRequirements != 4 {
		t.Errorf("expected 4 requirements, got: %d", snapshot.Summary.TotalRequirements)
	}
	if snapshot.Summary.CriticalGap != 1 {
		t.Errorf("expected critical gap 1, got: %d", snapshot.Summary.CriticalGap)
	}

	// Verify modules
	if len(snapshot.Modules) != 2 {
		t.Errorf("expected 2 modules, got: %d", len(snapshot.Modules))
	}

	// Verify operational targets
	if len(snapshot.OperationalTargets) != 2 {
		t.Errorf("expected 2 OTs (PRD-CORE and PRD-EXT), got: %d", len(snapshot.OperationalTargets))
	}

	// Write and verify
	writer := newMemWriter()
	err = WriteSnapshot(writer, "/output/snapshot.json", snapshot)
	if err != nil {
		t.Errorf("failed to write: %v", err)
	}

	data, ok := writer.GetFile("/output/snapshot.json")
	if !ok {
		t.Error("snapshot file not written")
	}

	var verified Snapshot
	if err := json.Unmarshal(data, &verified); err != nil {
		t.Errorf("invalid JSON in output: %v", err)
	}

	if verified.Summary.TotalRequirements != 4 {
		t.Errorf("verification failed: expected 4 requirements, got: %d", verified.Summary.TotalRequirements)
	}
}
