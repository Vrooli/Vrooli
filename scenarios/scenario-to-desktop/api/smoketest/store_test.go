package smoketest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryStore(t *testing.T) {
	store := NewInMemoryStore()
	if store == nil {
		t.Fatalf("expected store to be created")
	}
	if store.statusMap == nil {
		t.Errorf("expected statusMap to be initialized")
	}
}

func TestFileStore_InMemoryMode(t *testing.T) {
	store := NewInMemoryStore()

	status := &Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	}

	store.Save(status)

	got, ok := store.Get("test-123")
	if !ok {
		t.Fatalf("expected to find saved status")
	}
	if got.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got %q", got.ScenarioName)
	}
}

func TestFileStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "smoke-tests.json")

	// Create store and save data
	store1, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	status := &Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "passed",
	}
	store1.Save(status)

	// Create new store and verify data persists
	store2, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}

	got, ok := store2.Get("test-123")
	if !ok {
		t.Fatalf("expected to find persisted status")
	}
	if got.Status != "passed" {
		t.Errorf("expected status 'passed', got %q", got.Status)
	}
}

func TestFileStore_GetNonExistent(t *testing.T) {
	store := NewInMemoryStore()

	_, ok := store.Get("nonexistent")
	if ok {
		t.Errorf("expected ok=false for nonexistent smoke test")
	}
}

func TestFileStore_Update(t *testing.T) {
	store := NewInMemoryStore()

	status := &Status{
		SmokeTestID: "test-123",
		Status:      "running",
	}
	store.Save(status)

	t.Run("update existing", func(t *testing.T) {
		updated := store.Update("test-123", func(s *Status) {
			s.Status = "passed"
			now := time.Now()
			s.CompletedAt = &now
		})
		if !updated {
			t.Errorf("expected Update to return true")
		}

		got, _ := store.Get("test-123")
		if got.Status != "passed" {
			t.Errorf("expected status 'passed', got %q", got.Status)
		}
	})

	t.Run("update nonexistent", func(t *testing.T) {
		updated := store.Update("nonexistent", func(s *Status) {
			s.Status = "failed"
		})
		if updated {
			t.Errorf("expected Update to return false for nonexistent")
		}
	})
}

func TestFileStore_Load_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "nonexistent", "store.json")

	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("expected no error for nonexistent file, got %v", err)
	}

	// Store should be empty but functional
	_, ok := store.Get("test")
	if ok {
		t.Errorf("expected empty store")
	}
}

func TestFileStore_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	err := os.WriteFile(storePath, []byte("not valid json"), 0o644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = NewStore(storePath)
	if err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

func TestFileStore_Load_SkipsNilAndEmptyID(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.json")

	// Write JSON with nil and empty ID entries
	data := `[
		{"smoke_test_id": "valid", "status": "passed"},
		{"smoke_test_id": "", "status": "invalid"},
		null
	]`
	err := os.WriteFile(storePath, []byte(data), 0o644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Only valid entry should be loaded
	_, ok := store.Get("valid")
	if !ok {
		t.Errorf("expected valid entry to be loaded")
	}

	// Empty ID should not be loaded
	_, ok = store.Get("")
	if ok {
		t.Errorf("expected empty ID entry to be skipped")
	}
}

func TestFileStore_Concurrency(t *testing.T) {
	store := NewInMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			testID := "test-" + string(rune('a'+i%26))
			store.Save(&Status{SmokeTestID: testID, Status: "running"})
			store.Get(testID)
			store.Update(testID, func(s *Status) {
				s.Status = "passed"
			})
		}(i)
	}
	wg.Wait()

	// Should not panic or corrupt data
}

func TestDefaultCancelManager(t *testing.T) {
	mgr := NewCancelManager()

	t.Run("set and take cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mgr.SetCancel("test-1", cancel)

		taken := mgr.TakeCancel("test-1")
		if taken == nil {
			t.Fatalf("expected to take cancel function")
		}

		// Should be removed after take
		taken2 := mgr.TakeCancel("test-1")
		if taken2 != nil {
			t.Errorf("expected nil after already taken")
		}

		// Verify cancel works
		taken()
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Errorf("expected context to be cancelled")
		}
	})

	t.Run("take nonexistent", func(t *testing.T) {
		taken := mgr.TakeCancel("nonexistent")
		if taken != nil {
			t.Errorf("expected nil for nonexistent")
		}
	})

	t.Run("clear", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		mgr.SetCancel("test-2", cancel)

		mgr.Clear("test-2")

		taken := mgr.TakeCancel("test-2")
		if taken != nil {
			t.Errorf("expected nil after clear")
		}
	})
}

func TestDefaultCancelManager_Concurrency(t *testing.T) {
	mgr := NewCancelManager()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "test-" + string(rune('a'+i%26))
			_, cancel := context.WithCancel(context.Background())
			mgr.SetCancel(id, cancel)
			mgr.TakeCancel(id)
			mgr.Clear(id)
		}(i)
	}
	wg.Wait()

	// Should not panic or corrupt data
}

func TestStatus_Fields(t *testing.T) {
	now := time.Now()
	completed := now.Add(30 * time.Second)
	status := &Status{
		SmokeTestID:          "test-123",
		ScenarioName:         "my-scenario",
		Platform:             "linux",
		Status:               "passed",
		ArtifactPath:         "/path/to/artifact",
		StartedAt:            now,
		CompletedAt:          &completed,
		Logs:                 []string{"starting...", "done"},
		Error:                "",
		TelemetryUploaded:    true,
		TelemetryUploadError: "",
	}

	if status.SmokeTestID != "test-123" {
		t.Errorf("expected SmokeTestID 'test-123'")
	}
	if status.Platform != "linux" {
		t.Errorf("expected Platform 'linux'")
	}
	if len(status.Logs) != 2 {
		t.Errorf("expected 2 log entries")
	}
	if !status.TelemetryUploaded {
		t.Errorf("expected TelemetryUploaded to be true")
	}
}

func TestStartRequest_Fields(t *testing.T) {
	req := StartRequest{
		ScenarioName: "my-scenario",
		Platform:     "darwin",
	}

	if req.ScenarioName != "my-scenario" {
		t.Errorf("expected ScenarioName 'my-scenario'")
	}
	if req.Platform != "darwin" {
		t.Errorf("expected Platform 'darwin'")
	}
}
