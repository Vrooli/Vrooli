package records

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store to be created")
	}
}

func TestFileStore_EmptyPath(t *testing.T) {
	_, err := NewFileStore("")
	if err == nil {
		t.Errorf("expected error for empty path")
	}
}

func TestFileStore_Upsert(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	t.Run("insert new", func(t *testing.T) {
		record := &DesktopAppRecord{
			ID:           "record-123",
			ScenarioName: "test-scenario",
			OutputPath:   "/path/to/output",
		}

		err := store.Upsert(record)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, ok := store.Get("record-123")
		if !ok {
			t.Fatalf("expected to find record")
		}
		if got.ScenarioName != "test-scenario" {
			t.Errorf("expected scenario name 'test-scenario'")
		}
		if got.CreatedAt.IsZero() {
			t.Errorf("expected CreatedAt to be set")
		}
		if got.UpdatedAt.IsZero() {
			t.Errorf("expected UpdatedAt to be set")
		}
	})

	t.Run("update existing preserves CreatedAt", func(t *testing.T) {
		record, _ := store.Get("record-123")
		originalCreatedAt := record.CreatedAt

		time.Sleep(10 * time.Millisecond) // Ensure different timestamp

		updatedRecord := &DesktopAppRecord{
			ID:           "record-123",
			ScenarioName: "updated-scenario",
			OutputPath:   "/new/path",
		}

		err := store.Upsert(updatedRecord)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _ := store.Get("record-123")
		if got.ScenarioName != "updated-scenario" {
			t.Errorf("expected updated scenario name")
		}
		if got.CreatedAt != originalCreatedAt {
			t.Errorf("expected CreatedAt to be preserved")
		}
		if got.UpdatedAt.Before(originalCreatedAt) || got.UpdatedAt.Equal(originalCreatedAt) {
			t.Errorf("expected UpdatedAt to be updated")
		}
	})

	t.Run("upsert nil record", func(t *testing.T) {
		err := store.Upsert(nil)
		if err == nil {
			t.Errorf("expected error for nil record")
		}
	})

	t.Run("upsert empty ID", func(t *testing.T) {
		err := store.Upsert(&DesktopAppRecord{ID: ""})
		if err == nil {
			t.Errorf("expected error for empty ID")
		}
	})
}

func TestFileStore_Get(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	t.Run("get nonexistent", func(t *testing.T) {
		_, ok := store.Get("nonexistent")
		if ok {
			t.Errorf("expected ok=false for nonexistent record")
		}
	})

	t.Run("get existing", func(t *testing.T) {
		_ = store.Upsert(&DesktopAppRecord{
			ID:           "record-1",
			ScenarioName: "scenario-1",
		})

		got, ok := store.Get("record-1")
		if !ok {
			t.Fatalf("expected to find record")
		}
		if got.ScenarioName != "scenario-1" {
			t.Errorf("expected scenario name 'scenario-1'")
		}
	})
}

func TestFileStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	t.Run("empty store", func(t *testing.T) {
		records := store.List()
		if len(records) != 0 {
			t.Errorf("expected empty list")
		}
	})

	t.Run("with records", func(t *testing.T) {
		_ = store.Upsert(&DesktopAppRecord{ID: "1", ScenarioName: "s1"})
		_ = store.Upsert(&DesktopAppRecord{ID: "2", ScenarioName: "s2"})

		records := store.List()
		if len(records) != 2 {
			t.Errorf("expected 2 records, got %d", len(records))
		}
	})
}

func TestFileStore_DeleteByScenario(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	_ = store.Upsert(&DesktopAppRecord{ID: "1", ScenarioName: "scenario-a"})
	_ = store.Upsert(&DesktopAppRecord{ID: "2", ScenarioName: "scenario-a"})
	_ = store.Upsert(&DesktopAppRecord{ID: "3", ScenarioName: "scenario-b"})

	t.Run("delete existing scenario", func(t *testing.T) {
		removed := store.DeleteByScenario("scenario-a")
		if removed != 2 {
			t.Errorf("expected 2 removed, got %d", removed)
		}

		records := store.List()
		if len(records) != 1 {
			t.Errorf("expected 1 remaining record")
		}
		if records[0].ScenarioName != "scenario-b" {
			t.Errorf("expected scenario-b to remain")
		}
	})

	t.Run("delete nonexistent scenario", func(t *testing.T) {
		removed := store.DeleteByScenario("nonexistent")
		if removed != 0 {
			t.Errorf("expected 0 removed for nonexistent scenario")
		}
	})
}

func TestFileStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	// Create store and save data
	store1, _ := NewFileStore(storePath)
	_ = store1.Upsert(&DesktopAppRecord{
		ID:           "record-1",
		ScenarioName: "test-scenario",
		OutputPath:   "/path/to/output",
	})

	// Create new store and verify persistence
	store2, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}

	got, ok := store2.Get("record-1")
	if !ok {
		t.Fatalf("expected to find persisted record")
	}
	if got.ScenarioName != "test-scenario" {
		t.Errorf("expected persisted scenario name")
	}
}

func TestFileStore_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(storePath, []byte("not valid json"), 0o644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = NewFileStore(storePath)
	if err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

func TestFileStore_Load_SkipsTestRecords(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	// Write JSON with test record that should be filtered
	data := `[
		{"id": "valid", "scenario_name": "real", "output_path": "/real/path"},
		{"id": "test", "scenario_name": "test", "output_path": "/tmp/scenario-to-desktop-test/output"}
	]`
	err := os.WriteFile(storePath, []byte(data), 0o644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Only valid entry should be loaded
	_, ok := store.Get("valid")
	if !ok {
		t.Errorf("expected valid entry to be loaded")
	}

	// Test record should be filtered out
	_, ok = store.Get("test")
	if ok {
		t.Errorf("expected test record to be filtered out")
	}
}

func TestFileStore_Load_SkipsNilAndEmptyID(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	data := `[
		{"id": "valid", "scenario_name": "real"},
		{"id": "", "scenario_name": "empty"},
		null
	]`
	err := os.WriteFile(storePath, []byte(data), 0o644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Only valid entry should be loaded
	records := store.List()
	if len(records) != 1 {
		t.Errorf("expected 1 valid record, got %d", len(records))
	}
}

func TestFileStore_Concurrency(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "record-" + string(rune('a'+i%26))
			_ = store.Upsert(&DesktopAppRecord{
				ID:           id,
				ScenarioName: "scenario-" + id,
			})
			store.Get(id)
			store.List()
		}(i)
	}
	wg.Wait()

	// Should not panic or corrupt data
	records := store.List()
	for _, r := range records {
		if r.ID == "" {
			t.Errorf("found corrupted record entry")
		}
	}
}

func TestDesktopAppRecord_Fields(t *testing.T) {
	now := time.Now()
	record := &DesktopAppRecord{
		ID:              "record-123",
		BuildID:         "build-456",
		ScenarioName:    "my-scenario",
		AppDisplayName:  "My App",
		TemplateType:    "universal",
		Framework:       "electron",
		LocationMode:    "default",
		OutputPath:      "/output/path",
		DestinationPath: "/dest/path",
		StagingPath:     "/staging/path",
		CustomPath:      "/custom/path",
		DeploymentMode:  "proxy",
		Icon:            "/path/to/icon.png",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if record.ID != "record-123" {
		t.Errorf("expected ID 'record-123'")
	}
	if record.Framework != "electron" {
		t.Errorf("expected Framework 'electron'")
	}
	if record.DeploymentMode != "proxy" {
		t.Errorf("expected DeploymentMode 'proxy'")
	}
}

func TestRecordWithBuild_Fields(t *testing.T) {
	rwb := &RecordWithBuild{
		Record: &DesktopAppRecord{
			ID:           "record-1",
			ScenarioName: "scenario-1",
		},
		Build: &BuildStatusView{
			Status:     "ready",
			OutputPath: "/output",
			Metadata:   map[string]interface{}{"key": "value"},
		},
		HasBuild:   true,
		BuildState: "ready",
	}

	if rwb.Record.ID != "record-1" {
		t.Errorf("expected Record.ID 'record-1'")
	}
	if !rwb.HasBuild {
		t.Errorf("expected HasBuild to be true")
	}
	if rwb.Build.Status != "ready" {
		t.Errorf("expected Build.Status 'ready'")
	}
}

func TestFileStore_Upsert_DeduplicatesByScenarioAndOutput(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	// Simulate two pipeline runs for the same scenario writing to the same output path.
	// This is exactly what QueueBuild does — each call generates a new UUID.
	first := &DesktopAppRecord{
		ID:           "build-aaa",
		BuildID:      "build-aaa",
		ScenarioName: "git-control-tower",
		OutputPath:   "/home/user/Vrooli/scenarios/git-control-tower/platforms/electron",
		TemplateType: "basic",
	}
	second := &DesktopAppRecord{
		ID:           "build-bbb",
		BuildID:      "build-bbb",
		ScenarioName: "git-control-tower",
		OutputPath:   "/home/user/Vrooli/scenarios/git-control-tower/platforms/electron",
		TemplateType: "basic",
	}

	if err := store.Upsert(first); err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	if err := store.Upsert(second); err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	// BUG REPRODUCTION: There should be exactly 1 record — the latest build
	// for this scenario+output combination, not 2 separate entries.
	records := store.List()
	if len(records) != 1 {
		t.Errorf("expected 1 record (deduplicated), got %d", len(records))
	}

	// The surviving record should be the second (newer) one
	if len(records) == 1 {
		if records[0].ID != "build-bbb" {
			t.Errorf("expected surviving record to be 'build-bbb', got %q", records[0].ID)
		}
	}
}

func TestFileStore_Upsert_DifferentOutputPathsNotDeduplicated(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	// Same scenario but different output locations should remain separate
	r1 := &DesktopAppRecord{
		ID:           "build-1",
		ScenarioName: "my-app",
		OutputPath:   "/path/a/platforms/electron",
	}
	r2 := &DesktopAppRecord{
		ID:           "build-2",
		ScenarioName: "my-app",
		OutputPath:   "/path/b/platforms/electron",
	}

	_ = store.Upsert(r1)
	_ = store.Upsert(r2)

	records := store.List()
	if len(records) != 2 {
		t.Errorf("expected 2 records for different output paths, got %d", len(records))
	}
}

func TestFileStore_Upsert_DifferentScenariosNotDeduplicated(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	store, _ := NewFileStore(storePath)

	// Different scenarios with same output path structure should remain separate
	r1 := &DesktopAppRecord{
		ID:           "build-1",
		ScenarioName: "app-a",
		OutputPath:   "/scenarios/app-a/platforms/electron",
	}
	r2 := &DesktopAppRecord{
		ID:           "build-2",
		ScenarioName: "app-b",
		OutputPath:   "/scenarios/app-b/platforms/electron",
	}

	_ = store.Upsert(r1)
	_ = store.Upsert(r2)

	records := store.List()
	if len(records) != 2 {
		t.Errorf("expected 2 records for different scenarios, got %d", len(records))
	}
}

func TestFileStore_Load_DeduplicatesExistingRecords(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "records.json")

	// Write a JSON file with multiple records for the same scenario+output,
	// simulating the accumulated duplicates from repeated pipeline runs.
	data := `[
		{"id": "old-1", "scenario_name": "git-control-tower", "output_path": "/scenarios/gct/platforms/electron", "updated_at": "2026-01-01T00:00:00Z"},
		{"id": "old-2", "scenario_name": "git-control-tower", "output_path": "/scenarios/gct/platforms/electron", "updated_at": "2026-01-02T00:00:00Z"},
		{"id": "latest", "scenario_name": "git-control-tower", "output_path": "/scenarios/gct/platforms/electron", "updated_at": "2026-01-03T00:00:00Z"},
		{"id": "other", "scenario_name": "other-scenario", "output_path": "/scenarios/other/platforms/electron", "updated_at": "2026-01-01T00:00:00Z"}
	]`
	if err := os.WriteFile(storePath, []byte(data), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	records := store.List()
	if len(records) != 2 {
		t.Errorf("expected 2 records (1 per scenario), got %d", len(records))
	}

	// The git-control-tower record should be the latest one
	gct, ok := store.Get("latest")
	if !ok {
		t.Errorf("expected latest git-control-tower record to survive dedup")
	}
	if gct != nil && gct.ScenarioName != "git-control-tower" {
		t.Errorf("unexpected scenario name: %s", gct.ScenarioName)
	}

	// Old duplicates should be gone
	if _, ok := store.Get("old-1"); ok {
		t.Errorf("expected old-1 to be removed by dedup")
	}
	if _, ok := store.Get("old-2"); ok {
		t.Errorf("expected old-2 to be removed by dedup")
	}

	// Other scenario should be unaffected
	if _, ok := store.Get("other"); !ok {
		t.Errorf("expected other-scenario record to survive")
	}
}

func TestIsTestRecord(t *testing.T) {
	tests := []struct {
		name     string
		record   *DesktopAppRecord
		expected bool
	}{
		{"nil record", nil, false},
		{"normal record", &DesktopAppRecord{OutputPath: "/real/path"}, false},
		{"test record", &DesktopAppRecord{OutputPath: "/tmp/scenario-to-desktop-test/output"}, true},
		{"test record nested", &DesktopAppRecord{OutputPath: "/tmp/scenario-to-desktop-test/nested/output"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTestRecord(tt.record)
			if result != tt.expected {
				t.Errorf("expected isTestRecord to return %v", tt.expected)
			}
		})
	}
}
