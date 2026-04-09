package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreIgnoresNonJSONFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create non-JSON files
	_ = os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("ignore me"), 0o644)
	_ = os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte("hidden file"), 0o644)
	_ = os.Mkdir(filepath.Join(tempDir, "subdir"), 0o755)

	// Create one valid JSON file
	validJSON := `{"pipeline_id": "valid", "scenario_name": "test", "status": "completed", "stages": {}}`
	_ = os.WriteFile(filepath.Join(tempDir, "valid.json"), []byte(validJSON), 0o644)

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
	_ = os.WriteFile(filepath.Join(tempDir, "invalid.json"), []byte("not valid json"), 0o644)

	// Create valid JSON file
	validJSON := `{"pipeline_id": "valid", "scenario_name": "test", "status": "completed", "stages": {}}`
	_ = os.WriteFile(filepath.Join(tempDir, "valid.json"), []byte(validJSON), 0o644)

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

// createResumedPipelineStatus builds a Status with ResumedInput for testing persistence.
func createResumedPipelineStatus() *Status {
	now := time.Now().Unix()
	return &Status{
		PipelineID:        "resumed-pipeline-456",
		ScenarioName:      "test-scenario",
		Status:            StatusCompleted,
		StoppedAfterStage: "generate",
		StartedAt:         now,
		CompletedAt:       now,
		Stages: map[string]*StageResult{
			"bundle":    {Stage: "bundle", Status: StatusSkipped, CompletedAt: now},
			"preflight": {Stage: "preflight", Status: StatusSkipped, CompletedAt: now},
			"generate":  {Stage: "generate", Status: StatusCompleted, CompletedAt: now},
		},
		StageOrder: []string{"bundle", "preflight", "generate", "build", "smoketest", "deploy"},
		Config: &Config{
			ScenarioName:     "test-scenario",
			ParentPipelineID: "parent-pipeline-123",
			ResumeFromStage:  "generate",
		},
		ParentPipelineID: "parent-pipeline-123",
		ResumedInput: &StageInput{
			PipelineID:   "parent-pipeline-123",
			ScenarioPath: "/scenarios/test-scenario",
			DesktopPath:  "/scenarios/test-scenario/platforms/electron",
			Config: &Config{
				ScenarioName:   "test-scenario",
				Platforms:      []string{"linux", "win"},
				DeploymentMode: "bundled",
				TemplateType:   "basic",
			},
		},
	}
}

// TestFileStoreResumedInputPersistence verifies that ResumedInput is properly
// persisted and restored across store recreation. This is critical for pipeline
// resumption after server restarts.
func TestFileStoreResumedInputPersistence(t *testing.T) {
	tempDir := t.TempDir()

	store1, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}

	status := createResumedPipelineStatus()
	store1.Save(status)

	t.Run("file contains resumed_input JSON", func(t *testing.T) {
		assertFileContains(t, filepath.Join(tempDir, "resumed-pipeline-456.json"),
			"resumed_input", "parent-pipeline-123")
	})

	t.Run("basic fields restored after store recreation", func(t *testing.T) {
		retrieved := getResumedPipeline(t, tempDir)
		if retrieved.PipelineID != "resumed-pipeline-456" {
			t.Errorf("expected PipelineID 'resumed-pipeline-456', got %q", retrieved.PipelineID)
		}
		if retrieved.ParentPipelineID != "parent-pipeline-123" {
			t.Errorf("expected ParentPipelineID 'parent-pipeline-123', got %q", retrieved.ParentPipelineID)
		}
		if retrieved.StoppedAfterStage != "generate" {
			t.Errorf("expected StoppedAfterStage 'generate', got %q", retrieved.StoppedAfterStage)
		}
	})

	t.Run("ResumedInput restored after store recreation", func(t *testing.T) {
		retrieved := getResumedPipeline(t, tempDir)
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
	})

	t.Run("ResumedInput.Config restored after store recreation", func(t *testing.T) {
		retrieved := getResumedPipeline(t, tempDir)
		if retrieved.ResumedInput == nil || retrieved.ResumedInput.Config == nil {
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
	})
}

// assertFileContains reads a file and asserts it contains all given substrings.
func assertFileContains(t *testing.T, filePath string, substrings ...string) {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", filePath, err)
	}
	content := string(data)
	for _, s := range substrings {
		if !contains(content, s) {
			t.Errorf("expected file to contain %q", s)
		}
	}
}

// getResumedPipeline recreates the store and retrieves the resumed pipeline.
func getResumedPipeline(t *testing.T, tempDir string) *Status {
	t.Helper()
	store, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("NewFileStore error: %v", err)
	}
	retrieved, ok := store.Get("resumed-pipeline-456")
	if !ok {
		t.Fatalf("expected to retrieve pipeline after store recreation")
	}
	return retrieved
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
