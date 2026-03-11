package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func testStorageResolver(t *testing.T, rootDir string) *storage.Resolver {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		EnvGet: func(key string) string {
			if key == "VROOLI_DATA" {
				return rootDir
			}
			return ""
		},
		UserHomeDir: func() (string, error) { return rootDir, nil },
	})
	if err != nil {
		t.Fatalf("create resolver: %v", err)
	}
	return resolver
}

func TestStorage_SaveAndList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	for i := 0; i < 3; i++ {
		meta := SnapshotSetMeta{
			ID:              "snap-" + string(rune('a'+i)),
			ScenarioSlug:    "my-scenario",
			TriggerType:     "manual",
			Pages:           []string{"/"},
			ScreenshotCount: 1,
			CreatedAt:       time.Now().UTC().Add(time.Duration(i) * time.Minute),
			Status:          "complete",
		}
		err := store.SaveSnapshotSet(1, meta, map[string][]byte{"_root_.png": {0x89, 0x50}}, nil)
		if err != nil {
			t.Fatalf("SaveSnapshotSet %d: %v", i, err)
		}
	}

	list, err := store.ListSnapshotSets(1, "my-scenario")
	if err != nil {
		t.Fatalf("ListSnapshotSets: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(list))
	}
	// Should be newest-first
	if list[0].ID != "snap-c" {
		t.Errorf("expected newest first (snap-c), got %s", list[0].ID)
	}
}

func TestStorage_GetDetail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	meta := SnapshotSetMeta{
		ID:              "detail-1",
		ScenarioSlug:    "test-scenario",
		TriggerType:     "manual",
		Pages:           []string{"/", "/about"},
		ScreenshotCount: 2,
		CreatedAt:       time.Now().UTC(),
		Status:          "complete",
	}
	err := store.SaveSnapshotSet(1, meta, map[string][]byte{
		"_root_.png":  {0x89, 0x50, 0x4E, 0x47},
		"_about_.png": {0x89, 0x50},
	}, nil)
	if err != nil {
		t.Fatalf("SaveSnapshotSet: %v", err)
	}

	detail, err := store.GetSnapshotSet(1, "test-scenario", "detail-1")
	if err != nil {
		t.Fatalf("GetSnapshotSet: %v", err)
	}
	if len(detail.Screenshots) != 2 {
		t.Errorf("expected 2 screenshots, got %d", len(detail.Screenshots))
	}
}

func TestStorage_ServeFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	origData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}
	meta := SnapshotSetMeta{
		ID:              "serve-1",
		ScenarioSlug:    "test-scenario",
		TriggerType:     "manual",
		ScreenshotCount: 1,
		CreatedAt:       time.Now().UTC(),
		Status:          "complete",
	}
	err := store.SaveSnapshotSet(1, meta, map[string][]byte{"_root_.png": origData}, nil)
	if err != nil {
		t.Fatalf("SaveSnapshotSet: %v", err)
	}

	data, err := store.GetScreenshotFile(1, "test-scenario", "serve-1", "_root_.png")
	if err != nil {
		t.Fatalf("GetScreenshotFile: %v", err)
	}
	if len(data) != len(origData) {
		t.Errorf("expected %d bytes, got %d", len(origData), len(data))
	}
}

func TestStorage_Delete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	meta := SnapshotSetMeta{
		ID:           "del-1",
		ScenarioSlug: "test-scenario",
		TriggerType:  "manual",
		CreatedAt:    time.Now().UTC(),
		Status:       "complete",
	}
	err := store.SaveSnapshotSet(1, meta, map[string][]byte{"_root_.png": {0x89}}, nil)
	if err != nil {
		t.Fatalf("SaveSnapshotSet: %v", err)
	}

	err = store.DeleteSnapshotSet(1, "test-scenario", "del-1")
	if err != nil {
		t.Fatalf("DeleteSnapshotSet: %v", err)
	}

	list, err := store.ListSnapshotSets(1, "test-scenario")
	if err != nil {
		t.Fatalf("ListSnapshotSets: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 snapshots after delete, got %d", len(list))
	}
}

func TestStorage_ClearAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	for _, slug := range []string{"scenario-a", "scenario-b"} {
		meta := SnapshotSetMeta{
			ID:           "s-1",
			ScenarioSlug: slug,
			TriggerType:  "manual",
			CreatedAt:    time.Now().UTC(),
			Status:       "complete",
		}
		err := store.SaveSnapshotSet(1, meta, map[string][]byte{"_root_.png": {0x89}}, nil)
		if err != nil {
			t.Fatalf("SaveSnapshotSet for %s: %v", slug, err)
		}
	}

	err := store.ClearAllSnapshots(1)
	if err != nil {
		t.Fatalf("ClearAllSnapshots: %v", err)
	}

	stats, err := store.GetStorageStats(1)
	if err != nil {
		t.Fatalf("GetStorageStats: %v", err)
	}
	if stats.SnapshotCount != 0 {
		t.Errorf("expected 0 snapshots after clear, got %d", stats.SnapshotCount)
	}
}

func TestStorage_Retention(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	// Save 12 snapshots - retention should keep only 10
	for i := 0; i < 12; i++ {
		meta := SnapshotSetMeta{
			ID:              "ret-" + string(rune('a'+i)),
			ScenarioSlug:    "my-scenario",
			TriggerType:     "manual",
			ScreenshotCount: 1,
			CreatedAt:       time.Now().UTC().Add(time.Duration(i) * time.Minute),
			Status:          "complete",
		}
		err := store.SaveSnapshotSet(1, meta, map[string][]byte{"_root_.png": {0x89}}, nil)
		if err != nil {
			t.Fatalf("SaveSnapshotSet %d: %v", i, err)
		}
	}

	list, err := store.ListSnapshotSets(1, "my-scenario")
	if err != nil {
		t.Fatalf("ListSnapshotSets: %v", err)
	}
	if len(list) > 10 {
		t.Errorf("expected at most 10 snapshots, got %d", len(list))
	}
}

func TestStorage_Stats(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	for _, slug := range []string{"scenario-a", "scenario-b"} {
		for i := 0; i < 2; i++ {
			meta := SnapshotSetMeta{
				ID:           slug + "-" + string(rune('0'+i)),
				ScenarioSlug: slug,
				TriggerType:  "manual",
				CreatedAt:    time.Now().UTC().Add(time.Duration(i) * time.Minute),
				Status:       "complete",
			}
			err := store.SaveSnapshotSet(1, meta, map[string][]byte{"_root_.png": {0x89, 0x50}}, nil)
			if err != nil {
				t.Fatalf("SaveSnapshotSet: %v", err)
			}
		}
	}

	stats, err := store.GetStorageStats(1)
	if err != nil {
		t.Fatalf("GetStorageStats: %v", err)
	}
	if stats.SnapshotCount != 4 {
		t.Errorf("expected 4 total snapshots, got %d", stats.SnapshotCount)
	}
	if len(stats.PerScenario) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(stats.PerScenario))
	}
}

func TestStorage_PathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	// Create a valid snapshot first
	meta := SnapshotSetMeta{
		ID:           "pt-1",
		ScenarioSlug: "test-scenario",
		TriggerType:  "manual",
		CreatedAt:    time.Now().UTC(),
		Status:       "complete",
	}
	// Write a file at the parent level to verify traversal is blocked
	parentDir, _ := store.snapshotDir(1, "test-scenario", "pt-1")
	if err := os.MkdirAll(filepath.Dir(parentDir), 0o755); err != nil {
		t.Fatalf("create parent dir parent: %v", err)
	}
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatalf("create parent dir: %v", err)
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(parentDir, "metadata.json"), metaBytes, 0o644); err != nil {
		t.Fatalf("write metadata.json: %v", err)
	}

	_, err = store.GetScreenshotFile(1, "test-scenario", "pt-1", "../metadata.json")
	if err == nil {
		t.Fatal("expected error for path traversal filename")
	}
}
