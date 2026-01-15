package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreCreation(t *testing.T) {
	tempDir := t.TempDir()

	store, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store to be created")
	}
}

func TestFileStoreCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "nested", "pipeline", "data")

	store, err := NewFileStore(subDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store to be created")
	}

	// Directory should exist
	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected path to be a directory")
	}
}

func TestFileStoreSaveAndGet(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	status := &Status{
		PipelineID:   "pipeline-123",
		ScenarioName: "test-scenario",
		Status:       StatusRunning,
		StartedAt:    time.Now().Unix(),
		Stages:       make(map[string]*StageResult),
	}

	store.Save(status)

	retrieved, ok := store.Get("pipeline-123")
	if !ok {
		t.Fatalf("expected to retrieve saved status")
	}
	if retrieved.PipelineID != "pipeline-123" {
		t.Errorf("expected PipelineID 'pipeline-123', got %q", retrieved.PipelineID)
	}
	if retrieved.ScenarioName != "test-scenario" {
		t.Errorf("expected ScenarioName 'test-scenario', got %q", retrieved.ScenarioName)
	}
}

func TestFileStoreGetNonexistent(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	_, ok := store.Get("nonexistent")
	if ok {
		t.Errorf("expected false for nonexistent pipeline")
	}
}

func TestFileStorePersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create first store instance and save data
	store1, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}

	status := &Status{
		PipelineID:   "persistent-pipeline",
		ScenarioName: "persistence-test",
		Status:       StatusCompleted,
		StartedAt:    time.Now().Unix(),
		CompletedAt:  time.Now().Unix(),
		Stages: map[string]*StageResult{
			"bundle": {
				Stage:       "bundle",
				Status:      StatusCompleted,
				CompletedAt: time.Now().Unix(),
			},
		},
		StageOrder: []string{"bundle", "preflight", "generate"},
		Config: &Config{
			ScenarioName: "persistence-test",
		},
	}
	store1.Save(status)

	// Verify file was created
	filePath := filepath.Join(tempDir, "persistent-pipeline.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("expected file to be created at %s", filePath)
	}

	// Create a new store instance (simulating restart)
	store2, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error on second create: %v", err)
	}

	// Data should be loaded from disk
	retrieved, ok := store2.Get("persistent-pipeline")
	if !ok {
		t.Fatalf("expected to retrieve pipeline after store recreation")
	}
	if retrieved.PipelineID != "persistent-pipeline" {
		t.Errorf("expected PipelineID 'persistent-pipeline', got %q", retrieved.PipelineID)
	}
	if retrieved.ScenarioName != "persistence-test" {
		t.Errorf("expected ScenarioName 'persistence-test', got %q", retrieved.ScenarioName)
	}
	if retrieved.Status != StatusCompleted {
		t.Errorf("expected Status 'completed', got %q", retrieved.Status)
	}
	if len(retrieved.Stages) != 1 {
		t.Errorf("expected 1 stage result, got %d", len(retrieved.Stages))
	}
	if retrieved.Config == nil {
		t.Errorf("expected Config to be preserved")
	}
}

func TestFileStoreLoadOnStartup(t *testing.T) {
	tempDir := t.TempDir()

	// Manually create JSON files before creating store
	pipelines := []struct {
		id       string
		scenario string
	}{
		{"pipeline-1", "scenario-a"},
		{"pipeline-2", "scenario-b"},
		{"pipeline-3", "scenario-c"},
	}

	for _, p := range pipelines {
		filePath := filepath.Join(tempDir, p.id+".json")
		content := `{
			"pipeline_id": "` + p.id + `",
			"scenario_name": "` + p.scenario + `",
			"status": "completed",
			"stages": {},
			"stage_order": ["bundle"]
		}`
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	// Create store - should load existing files
	store, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}

	// All pipelines should be loaded
	for _, p := range pipelines {
		retrieved, ok := store.Get(p.id)
		if !ok {
			t.Errorf("expected pipeline %q to be loaded", p.id)
			continue
		}
		if retrieved.ScenarioName != p.scenario {
			t.Errorf("expected ScenarioName %q for %q, got %q", p.scenario, p.id, retrieved.ScenarioName)
		}
	}

	// Verify list returns all
	all := store.List()
	if len(all) != 3 {
		t.Errorf("expected 3 pipelines in list, got %d", len(all))
	}
}

func TestFileStoreUpdate(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	status := &Status{
		PipelineID:   "update-test",
		ScenarioName: "test",
		Status:       StatusPending,
		Stages:       make(map[string]*StageResult),
	}
	store.Save(status)

	// Update the status
	updated := store.Update("update-test", func(s *Status) {
		s.Status = StatusCompleted
		s.CompletedAt = time.Now().Unix()
	})
	if !updated {
		t.Errorf("expected Update to return true")
	}

	// Verify in-memory update
	retrieved, _ := store.Get("update-test")
	if retrieved.Status != StatusCompleted {
		t.Errorf("expected status 'completed', got %q", retrieved.Status)
	}

	// Verify persisted to disk
	store2, _ := NewFileStore(tempDir)
	retrieved2, ok := store2.Get("update-test")
	if !ok {
		t.Fatalf("expected to retrieve updated pipeline from disk")
	}
	if retrieved2.Status != StatusCompleted {
		t.Errorf("expected persisted status 'completed', got %q", retrieved2.Status)
	}
}

func TestFileStoreUpdateNonexistent(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	updated := store.Update("nonexistent", func(s *Status) {
		s.Status = StatusCompleted
	})
	if updated {
		t.Errorf("expected Update to return false for nonexistent")
	}
}

func TestFileStoreUpdateStage(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	status := &Status{
		PipelineID:   "stage-update-test",
		ScenarioName: "test",
		Status:       StatusRunning,
		Stages:       make(map[string]*StageResult),
	}
	store.Save(status)

	// Update a stage
	result := &StageResult{
		Stage:       "bundle",
		Status:      StatusCompleted,
		CompletedAt: time.Now().Unix(),
	}
	updated := store.UpdateStage("stage-update-test", "bundle", result)
	if !updated {
		t.Errorf("expected UpdateStage to return true")
	}

	// Verify in-memory update
	retrieved, _ := store.Get("stage-update-test")
	if retrieved.Stages["bundle"].Status != StatusCompleted {
		t.Errorf("expected bundle stage status 'completed'")
	}

	// Verify persisted to disk
	store2, _ := NewFileStore(tempDir)
	retrieved2, ok := store2.Get("stage-update-test")
	if !ok {
		t.Fatalf("expected to retrieve pipeline from disk")
	}
	if retrieved2.Stages["bundle"].Status != StatusCompleted {
		t.Errorf("expected persisted bundle stage status 'completed'")
	}
}

func TestFileStoreDelete(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	status := &Status{
		PipelineID:   "delete-test",
		ScenarioName: "test",
		Status:       StatusCompleted,
		Stages:       make(map[string]*StageResult),
	}
	store.Save(status)

	// Verify file exists
	filePath := filepath.Join(tempDir, "delete-test.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("expected file to exist before delete")
	}

	// Delete
	deleted := store.Delete("delete-test")
	if !deleted {
		t.Errorf("expected Delete to return true")
	}

	// Verify removed from memory
	_, ok := store.Get("delete-test")
	if ok {
		t.Errorf("expected pipeline to be deleted from memory")
	}

	// Verify file deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted from disk")
	}
}

func TestFileStoreDeleteNonexistent(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	deleted := store.Delete("nonexistent")
	if deleted {
		t.Errorf("expected Delete to return false for nonexistent")
	}
}

func TestFileStoreList(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	// Save multiple pipelines
	for i := 0; i < 5; i++ {
		store.Save(&Status{
			PipelineID:   "list-test-" + string(rune('a'+i)),
			ScenarioName: "test",
			Status:       StatusCompleted,
			Stages:       make(map[string]*StageResult),
		})
	}

	all := store.List()
	if len(all) != 5 {
		t.Errorf("expected 5 pipelines, got %d", len(all))
	}
}

func TestFileStoreCleanup(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)
	now := time.Now()

	// Create pipelines with different completion times
	oldCompleted := &Status{
		PipelineID:  "old-completed",
		Status:      StatusCompleted,
		CompletedAt: now.Add(-48 * time.Hour).Unix(),
		Stages:      make(map[string]*StageResult),
	}
	recentCompleted := &Status{
		PipelineID:  "recent-completed",
		Status:      StatusCompleted,
		CompletedAt: now.Add(-1 * time.Hour).Unix(),
		Stages:      make(map[string]*StageResult),
	}
	running := &Status{
		PipelineID: "running",
		Status:     StatusRunning,
		Stages:     make(map[string]*StageResult),
	}
	oldFailed := &Status{
		PipelineID:  "old-failed",
		Status:      StatusFailed,
		CompletedAt: now.Add(-72 * time.Hour).Unix(),
		Stages:      make(map[string]*StageResult),
	}

	store.Save(oldCompleted)
	store.Save(recentCompleted)
	store.Save(running)
	store.Save(oldFailed)

	// Cleanup pipelines older than 24 hours
	cutoffTime := now.Add(-24 * time.Hour).Unix()
	store.Cleanup(cutoffTime)

	// Old completed and old failed should be deleted
	if _, ok := store.Get("old-completed"); ok {
		t.Error("expected old-completed to be cleaned up")
	}
	if _, ok := store.Get("old-failed"); ok {
		t.Error("expected old-failed to be cleaned up")
	}

	// Recent completed and running should remain
	if _, ok := store.Get("recent-completed"); !ok {
		t.Error("expected recent-completed to remain")
	}
	if _, ok := store.Get("running"); !ok {
		t.Error("expected running to remain")
	}

	// Verify files are also deleted
	if _, err := os.Stat(filepath.Join(tempDir, "old-completed.json")); !os.IsNotExist(err) {
		t.Error("expected old-completed.json to be deleted")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "old-failed.json")); !os.IsNotExist(err) {
		t.Error("expected old-failed.json to be deleted")
	}
}

func TestFileStoreIgnoresNonJSONFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create non-JSON files
	os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("ignore me"), 0o644)
	os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte("hidden file"), 0o644)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0o755)

	// Create one valid JSON file
	validJSON := `{"pipeline_id": "valid", "scenario_name": "test", "status": "completed", "stages": {}}`
	os.WriteFile(filepath.Join(tempDir, "valid.json"), []byte(validJSON), 0o644)

	store, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}

	// Only the valid pipeline should be loaded
	all := store.List()
	if len(all) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(all))
	}

	if _, ok := store.Get("valid"); !ok {
		t.Error("expected 'valid' pipeline to be loaded")
	}
}

func TestFileStoreSkipsInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid JSON file
	os.WriteFile(filepath.Join(tempDir, "invalid.json"), []byte("not valid json"), 0o644)

	// Create valid JSON file
	validJSON := `{"pipeline_id": "valid", "scenario_name": "test", "status": "completed", "stages": {}}`
	os.WriteFile(filepath.Join(tempDir, "valid.json"), []byte(validJSON), 0o644)

	// Should not error, just skip invalid file
	store, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}

	// Only valid pipeline should be loaded
	all := store.List()
	if len(all) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(all))
	}
}

func TestFileStoreWithLogger(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}

	store, err := NewFileStore(tempDir, WithFileStoreLogger(logger))
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store to be created")
	}
	if store.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestFileStoreAtomicWrites(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	status := &Status{
		PipelineID:   "atomic-test",
		ScenarioName: "test",
		Status:       StatusRunning,
		Stages:       make(map[string]*StageResult),
	}
	store.Save(status)

	// Temp file should not exist after save
	tempPath := filepath.Join(tempDir, "atomic-test.json.tmp")
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("expected temp file to be cleaned up after save")
	}

	// Final file should exist
	finalPath := filepath.Join(tempDir, "atomic-test.json")
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		t.Error("expected final file to exist")
	}
}

func TestFileStoreConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			status := &Status{
				PipelineID:   "concurrent-" + string(rune('0'+id)),
				ScenarioName: "test",
				Status:       StatusRunning,
				Stages:       make(map[string]*StageResult),
			}
			store.Save(status)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All should be saved
	all := store.List()
	if len(all) != 10 {
		t.Errorf("expected 10 pipelines, got %d", len(all))
	}
}

// TestFileStoreResumedInputPersistence verifies that ResumedInput is properly
// persisted and restored across store recreation. This is critical for pipeline
// resumption after server restarts.
func TestFileStoreResumedInputPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create first store instance and save data with ResumedInput
	store1, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}

	// Create a status with ResumedInput populated
	resumedInput := &StageInput{
		PipelineID:   "parent-pipeline-123",
		ScenarioPath: "/scenarios/test-scenario",
		DesktopPath:  "/scenarios/test-scenario/platforms/electron",
		Config: &Config{
			ScenarioName:   "test-scenario",
			Platforms:      []string{"linux", "win"},
			DeploymentMode: "bundled",
			TemplateType:   "basic",
		},
	}

	status := &Status{
		PipelineID:        "resumed-pipeline-456",
		ScenarioName:      "test-scenario",
		Status:            StatusCompleted,
		StoppedAfterStage: "generate",
		StartedAt:         time.Now().Unix(),
		CompletedAt:       time.Now().Unix(),
		Stages: map[string]*StageResult{
			"bundle": {
				Stage:       "bundle",
				Status:      StatusSkipped,
				CompletedAt: time.Now().Unix(),
			},
			"preflight": {
				Stage:       "preflight",
				Status:      StatusSkipped,
				CompletedAt: time.Now().Unix(),
			},
			"generate": {
				Stage:       "generate",
				Status:      StatusCompleted,
				CompletedAt: time.Now().Unix(),
			},
		},
		StageOrder: []string{"bundle", "preflight", "generate", "build", "smoketest", "distribution"},
		Config: &Config{
			ScenarioName:     "test-scenario",
			ParentPipelineID: "parent-pipeline-123",
			ResumeFromStage:  "generate",
		},
		ParentPipelineID: "parent-pipeline-123",
		ResumedInput:     resumedInput,
	}
	store1.Save(status)

	// Verify file was created
	filePath := filepath.Join(tempDir, "resumed-pipeline-456.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("expected file to be created at %s", filePath)
	}

	// Verify the JSON file contains resumed_input
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}
	if !contains(string(data), "resumed_input") {
		t.Errorf("expected JSON to contain 'resumed_input' field")
	}
	if !contains(string(data), "parent-pipeline-123") {
		t.Errorf("expected JSON to contain parent pipeline ID")
	}

	// Create a new store instance (simulating server restart)
	store2, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error on second create: %v", err)
	}

	// Data should be loaded from disk
	retrieved, ok := store2.Get("resumed-pipeline-456")
	if !ok {
		t.Fatalf("expected to retrieve pipeline after store recreation")
	}

	// Verify basic fields
	if retrieved.PipelineID != "resumed-pipeline-456" {
		t.Errorf("expected PipelineID 'resumed-pipeline-456', got %q", retrieved.PipelineID)
	}
	if retrieved.ParentPipelineID != "parent-pipeline-123" {
		t.Errorf("expected ParentPipelineID 'parent-pipeline-123', got %q", retrieved.ParentPipelineID)
	}
	if retrieved.StoppedAfterStage != "generate" {
		t.Errorf("expected StoppedAfterStage 'generate', got %q", retrieved.StoppedAfterStage)
	}

	// Critical: Verify ResumedInput was restored
	if retrieved.ResumedInput == nil {
		t.Fatalf("expected ResumedInput to be restored, got nil")
	}
	if retrieved.ResumedInput.PipelineID != "parent-pipeline-123" {
		t.Errorf("expected ResumedInput.PipelineID 'parent-pipeline-123', got %q", retrieved.ResumedInput.PipelineID)
	}
	if retrieved.ResumedInput.ScenarioPath != "/scenarios/test-scenario" {
		t.Errorf("expected ResumedInput.ScenarioPath, got %q", retrieved.ResumedInput.ScenarioPath)
	}
	if retrieved.ResumedInput.DesktopPath != "/scenarios/test-scenario/platforms/electron" {
		t.Errorf("expected ResumedInput.DesktopPath, got %q", retrieved.ResumedInput.DesktopPath)
	}

	// Verify nested Config in ResumedInput
	if retrieved.ResumedInput.Config == nil {
		t.Fatalf("expected ResumedInput.Config to be restored, got nil")
	}
	if retrieved.ResumedInput.Config.ScenarioName != "test-scenario" {
		t.Errorf("expected ResumedInput.Config.ScenarioName 'test-scenario', got %q",
			retrieved.ResumedInput.Config.ScenarioName)
	}
	if len(retrieved.ResumedInput.Config.Platforms) != 2 {
		t.Errorf("expected ResumedInput.Config.Platforms to have 2 items, got %d",
			len(retrieved.ResumedInput.Config.Platforms))
	}
}

// TestFileStoreResumedInputUpdate verifies that ResumedInput updates are persisted.
func TestFileStoreResumedInputUpdate(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewFileStore(tempDir)

	// Create initial status without ResumedInput
	status := &Status{
		PipelineID:   "update-resumed-test",
		ScenarioName: "test",
		Status:       StatusRunning,
		Stages:       make(map[string]*StageResult),
	}
	store.Save(status)

	// Update with ResumedInput
	updated := store.Update("update-resumed-test", func(s *Status) {
		s.Status = StatusCompleted
		s.StoppedAfterStage = "preflight"
		s.ResumedInput = &StageInput{
			PipelineID:   "parent-123",
			ScenarioPath: "/test/path",
		}
	})
	if !updated {
		t.Errorf("expected Update to return true")
	}

	// Verify persisted to disk
	store2, _ := NewFileStore(tempDir)
	retrieved, ok := store2.Get("update-resumed-test")
	if !ok {
		t.Fatalf("expected to retrieve updated pipeline from disk")
	}
	if retrieved.ResumedInput == nil {
		t.Fatalf("expected ResumedInput to be persisted after update")
	}
	if retrieved.ResumedInput.PipelineID != "parent-123" {
		t.Errorf("expected ResumedInput.PipelineID 'parent-123', got %q", retrieved.ResumedInput.PipelineID)
	}
}

// contains is a helper function for string containment check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
