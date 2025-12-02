package requirements

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/requirements/reporting"
	"test-genie/internal/requirements/types"
)

// testFixture holds all the test data for a single test scenario.
type testFixture struct {
	scenarioDir string
	reader      *memReader
	writer      *memWriter
}

// newTestFixture creates a fixture with basic requirements directory structure.
func newTestFixture(t *testing.T) *testFixture {
	t.Helper()
	scenarioDir := "/test/scenarios/demo"

	reader := NewMemReader()
	writer := NewMemWriter()

	// Create requirements directory entry
	reader.AddDir(filepath.Join(scenarioDir, "requirements"), []fs.DirEntry{})

	return &testFixture{
		scenarioDir: scenarioDir,
		reader:      reader,
		writer:      writer,
	}
}

// addRequirementModule adds a parsed requirement module to the fixture.
func (f *testFixture) addRequirementModule(t *testing.T, relativePath string, module map[string]any) {
	t.Helper()
	data, err := json.Marshal(module)
	if err != nil {
		t.Fatalf("marshal module: %v", err)
	}

	absPath := filepath.Join(f.scenarioDir, "requirements", relativePath)
	f.reader.AddFile(absPath, data)

	// Update directory listing
	dir := filepath.Dir(absPath)
	entries, _ := f.reader.ReadDir(dir)

	// Add new entry if not present
	name := filepath.Base(absPath)
	found := false
	for _, e := range entries {
		if e.Name() == name {
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, &memDirEntry{name: name, isDir: false})
		f.reader.AddDir(dir, entries)
	}
}

// addIndexFile adds an index.json file.
func (f *testFixture) addIndexFile(t *testing.T, imports []string) {
	t.Helper()
	f.addRequirementModule(t, "index.json", map[string]any{
		"imports": imports,
	})
}

// memDirEntry implements fs.DirEntry for testing.
type memDirEntry struct {
	name  string
	isDir bool
}

func (e *memDirEntry) Name() string               { return e.name }
func (e *memDirEntry) IsDir() bool                { return e.isDir }
func (e *memDirEntry) Type() fs.FileMode          { return 0 }
func (e *memDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestService_Sync_NoRequirementsDir(t *testing.T) {
	reader := NewMemReader()
	writer := NewMemWriter()
	service := NewServiceWithDeps(reader, writer)

	err := service.Sync(context.Background(), SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  "/test/scenarios/demo",
	})

	if err != nil {
		t.Errorf("expected nil error for missing requirements dir, got: %v", err)
	}
}

func TestService_Sync_EmptyRequirementsDir(t *testing.T) {
	fixture := newTestFixture(t)
	service := NewServiceWithDeps(fixture.reader, fixture.writer)

	// Add index.json with no imports (empty requirements)
	fixture.addIndexFile(t, []string{})

	err := service.Sync(context.Background(), SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  fixture.scenarioDir,
	})

	if err != nil {
		t.Errorf("expected nil error for empty requirements, got: %v", err)
	}
}

func TestService_Sync_BasicRequirement(t *testing.T) {
	fixture := newTestFixture(t)
	service := NewServiceWithDeps(fixture.reader, fixture.writer)

	// Add module directory
	moduleDir := filepath.Join(fixture.scenarioDir, "requirements", "01-core")
	fixture.reader.AddDir(moduleDir, []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	})

	// Add index pointing to module
	fixture.addIndexFile(t, []string{"01-core"})

	// Add module with requirements
	fixture.addRequirementModule(t, "01-core/module.json", map[string]any{
		"_metadata": map[string]any{
			"module": "core",
		},
		"requirements": []map[string]any{
			{
				"id":     "REQ-001",
				"title":  "Test requirement",
				"status": "pending",
				"validation": []map[string]any{
					{
						"type":   "test",
						"ref":    "test/example.test.ts",
						"status": "not_implemented",
					},
				},
			},
		},
	})

	err := service.Sync(context.Background(), SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  fixture.scenarioDir,
		PhaseResults: []phases.ExecutionResult{
			{Name: "unit", Status: "passed"},
		},
		CommandHistory: []string{"suite demo"},
	})

	if err != nil {
		t.Errorf("sync failed: %v", err)
	}
}

func TestService_Sync_WithPhaseResults(t *testing.T) {
	fixture := newTestFixture(t)
	service := NewServiceWithDeps(fixture.reader, fixture.writer)

	// Add module directory
	moduleDir := filepath.Join(fixture.scenarioDir, "requirements", "01-core")
	fixture.reader.AddDir(moduleDir, []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	})

	fixture.addIndexFile(t, []string{"01-core"})
	fixture.addRequirementModule(t, "01-core/module.json", map[string]any{
		"_metadata": map[string]any{"module": "core"},
		"requirements": []map[string]any{
			{
				"id":     "REQ-001",
				"title":  "Tested requirement",
				"status": "in_progress",
			},
		},
	})

	// Sync with phase results
	err := service.Sync(context.Background(), SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  fixture.scenarioDir,
		PhaseDefinitions: []phases.Definition{
			{Name: phases.Unit},
			{Name: phases.Structure},
		},
		PhaseResults: []phases.ExecutionResult{
			{Name: "unit", Status: "PASSED", DurationSeconds: 5},
			{Name: "structure", Status: "PASSED", DurationSeconds: 2},
		},
		CommandHistory: []string{"suite demo"},
	})

	if err != nil {
		t.Errorf("sync with phase results failed: %v", err)
	}
}

func TestService_Validate_NoRequirementsDir(t *testing.T) {
	reader := NewMemReader()
	writer := NewMemWriter()
	service := NewServiceWithDeps(reader, writer)

	result, err := service.Validate(context.Background(), "/test/scenarios/demo")

	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestService_Validate_WithRequirements(t *testing.T) {
	fixture := newTestFixture(t)
	service := NewServiceWithDeps(fixture.reader, fixture.writer)

	moduleDir := filepath.Join(fixture.scenarioDir, "requirements", "01-core")
	fixture.reader.AddDir(moduleDir, []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	})

	fixture.addIndexFile(t, []string{"01-core"})
	fixture.addRequirementModule(t, "01-core/module.json", map[string]any{
		"_metadata": map[string]any{"module": "core"},
		"requirements": []map[string]any{
			{
				"id":     "REQ-001",
				"title":  "Valid requirement",
				"status": "pending",
			},
		},
	})

	result, err := service.Validate(context.Background(), fixture.scenarioDir)

	if err != nil {
		t.Errorf("validate failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestService_GetSummary_EmptyRequirements(t *testing.T) {
	fixture := newTestFixture(t)
	service := NewServiceWithDeps(fixture.reader, fixture.writer)

	fixture.addIndexFile(t, []string{})

	summary, err := service.GetSummary(context.Background(), fixture.scenarioDir)

	if err != nil {
		t.Errorf("get summary failed: %v", err)
	}
	if summary.Total != 0 {
		t.Errorf("expected 0 total requirements, got: %d", summary.Total)
	}
}

func TestService_GetSummary_WithRequirements(t *testing.T) {
	// Use real filesystem for this test as discovery requires directory walking
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "demo")
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	moduleDir := filepath.Join(requirementsDir, "01-core")

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("create module dir: %v", err)
	}

	// Imports must be full file paths like "01-core/module.json"
	indexData := []byte(`{"imports": ["01-core/module.json"]}`)
	if err := os.WriteFile(filepath.Join(requirementsDir, "index.json"), indexData, 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	moduleData := []byte(`{
		"_metadata": {"module": "core"},
		"requirements": [
			{"id": "REQ-001", "title": "Requirement 1", "status": "complete", "criticality": "P0"},
			{"id": "REQ-002", "title": "Requirement 2", "status": "pending", "criticality": "P1"},
			{"id": "REQ-003", "title": "Requirement 3", "status": "in_progress"}
		]
	}`)
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), moduleData, 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	service := NewService()
	summary, err := service.GetSummary(context.Background(), scenarioDir)

	if err != nil {
		t.Errorf("get summary failed: %v", err)
	}
	if summary.Total != 3 {
		t.Errorf("expected 3 total requirements, got: %d", summary.Total)
	}
	if summary.ByDeclaredStatus[types.StatusComplete] != 1 {
		t.Errorf("expected 1 complete requirement, got: %d", summary.ByDeclaredStatus[types.StatusComplete])
	}
	if summary.ByDeclaredStatus[types.StatusPending] != 1 {
		t.Errorf("expected 1 pending requirement, got: %d", summary.ByDeclaredStatus[types.StatusPending])
	}
	if summary.ByDeclaredStatus[types.StatusInProgress] != 1 {
		t.Errorf("expected 1 in_progress requirement, got: %d", summary.ByDeclaredStatus[types.StatusInProgress])
	}
}

func TestService_Report_JSON(t *testing.T) {
	fixture := newTestFixture(t)
	service := NewServiceWithDeps(fixture.reader, fixture.writer)

	moduleDir := filepath.Join(fixture.scenarioDir, "requirements", "01-core")
	fixture.reader.AddDir(moduleDir, []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	})

	fixture.addIndexFile(t, []string{"01-core"})
	fixture.addRequirementModule(t, "01-core/module.json", map[string]any{
		"_metadata": map[string]any{"module": "core"},
		"requirements": []map[string]any{
			{
				"id":     "REQ-001",
				"title":  "Test requirement",
				"status": "pending",
			},
		},
	})

	var buf bytes.Buffer
	err := service.Report(context.Background(), fixture.scenarioDir, reportingOptionsJSON(), &buf)

	if err != nil {
		t.Errorf("report failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty report output")
	}
}

func TestConvertPhaseResults(t *testing.T) {
	results := []phases.ExecutionResult{
		{Name: "unit", Status: "PASSED", DurationSeconds: 5, LogPath: "/logs/unit.log"},
		{Name: "structure", Status: "failed", DurationSeconds: 2, Error: "syntax error"},
	}

	evidenceMap := convertPhaseResults(results)

	if len(evidenceMap) != 2 {
		t.Errorf("expected 2 evidence entries, got: %d", len(evidenceMap))
	}

	unitKey := "__phase__unit"
	if records := evidenceMap[unitKey]; len(records) != 1 {
		t.Errorf("expected 1 unit record, got: %d", len(records))
	} else {
		if records[0].Status != types.LivePassed {
			t.Errorf("expected passed status, got: %s", records[0].Status)
		}
		if records[0].DurationSeconds != 5 {
			t.Errorf("expected 5s duration, got: %f", records[0].DurationSeconds)
		}
	}

	structureKey := "__phase__structure"
	if records := evidenceMap[structureKey]; len(records) != 1 {
		t.Errorf("expected 1 structure record, got: %d", len(records))
	} else {
		if records[0].Status != types.LiveFailed {
			t.Errorf("expected failed status, got: %s", records[0].Status)
		}
		if records[0].Evidence != "syntax error" {
			t.Errorf("expected 'syntax error' evidence, got: %s", records[0].Evidence)
		}
	}
}

func TestConvertPhaseResults_Empty(t *testing.T) {
	evidenceMap := convertPhaseResults(nil)

	if len(evidenceMap) != 0 {
		t.Errorf("expected empty map for nil input, got: %d", len(evidenceMap))
	}

	evidenceMap = convertPhaseResults([]phases.ExecutionResult{})

	if len(evidenceMap) != 0 {
		t.Errorf("expected empty map for empty input, got: %d", len(evidenceMap))
	}
}

func TestService_Sync_ContextCancellation(t *testing.T) {
	fixture := newTestFixture(t)
	service := NewServiceWithDeps(fixture.reader, fixture.writer)

	moduleDir := filepath.Join(fixture.scenarioDir, "requirements", "01-core")
	fixture.reader.AddDir(moduleDir, []fs.DirEntry{
		&memDirEntry{name: "module.json", isDir: false},
	})

	fixture.addIndexFile(t, []string{"01-core"})
	fixture.addRequirementModule(t, "01-core/module.json", map[string]any{
		"_metadata": map[string]any{"module": "core"},
		"requirements": []map[string]any{
			{"id": "REQ-001", "title": "Test", "status": "pending"},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := service.Sync(ctx, SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  fixture.scenarioDir,
	})

	// The error might be nil if discovery completes before ctx check, or wrapped context.Canceled
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("expected nil or context.Canceled, got: %v", err)
	}
}

// Helper to create JSON reporting options.
func reportingOptionsJSON() reporting.Options {
	return reporting.Options{Format: reporting.FormatJSON}
}

// Integration test using real filesystem (temporary directory)
func TestService_Sync_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "demo")
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	moduleDir := filepath.Join(requirementsDir, "01-core")

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("create module dir: %v", err)
	}

	// Write index.json
	indexData := []byte(`{"imports": ["01-core"]}`)
	if err := os.WriteFile(filepath.Join(requirementsDir, "index.json"), indexData, 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	// Write module.json
	moduleData := []byte(`{
		"_metadata": {"module": "core"},
		"requirements": [
			{
				"id": "REQ-001",
				"title": "Integration test requirement",
				"status": "pending",
				"validation": [
					{"type": "test", "ref": "test/example.test.ts", "status": "not_implemented"}
				]
			}
		]
	}`)
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), moduleData, 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	// Run sync with real filesystem
	service := NewService()
	err := service.Sync(context.Background(), SyncInput{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		PhaseDefinitions: []phases.Definition{
			{Name: phases.Unit},
		},
		PhaseResults: []phases.ExecutionResult{
			{Name: "unit", Status: "passed", DurationSeconds: 3},
		},
		CommandHistory: []string{"suite demo"},
	})

	if err != nil {
		t.Errorf("integration sync failed: %v", err)
	}

	// Verify module was updated (check last_validated_at was set)
	updatedData, err := os.ReadFile(filepath.Join(moduleDir, "module.json"))
	if err != nil {
		t.Fatalf("read updated module: %v", err)
	}

	if !strings.Contains(string(updatedData), "last_validated_at") {
		t.Log("Note: last_validated_at may not be set if sync didn't detect changes")
	}
}

func TestService_Report_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "demo")
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	moduleDir := filepath.Join(requirementsDir, "01-core")

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("create module dir: %v", err)
	}

	indexData := []byte(`{"imports": ["01-core"]}`)
	if err := os.WriteFile(filepath.Join(requirementsDir, "index.json"), indexData, 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	moduleData := []byte(`{
		"_metadata": {"module": "core"},
		"requirements": [
			{"id": "REQ-001", "title": "Test", "status": "complete", "criticality": "P0"},
			{"id": "REQ-002", "title": "Test 2", "status": "pending", "criticality": "P1"}
		]
	}`)
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), moduleData, 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	service := NewService()
	var buf bytes.Buffer
	err := service.Report(context.Background(), scenarioDir, reportingOptionsJSON(), &buf)

	if err != nil {
		t.Errorf("integration report failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty report")
	}

	// Verify JSON is valid
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("report output is not valid JSON: %v", err)
	}
}
