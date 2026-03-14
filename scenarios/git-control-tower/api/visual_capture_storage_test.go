package main

import (
	"encoding/json"
	"fmt"
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

func TestStorage_DeleteSnapshotsByRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	// Save baseline + 2 captures
	for _, m := range []SnapshotSetMeta{
		{ID: "b1", ScenarioSlug: "s", Role: SnapshotRoleBaseline, TriggerType: "manual", CreatedAt: time.Now().UTC().Add(-2 * time.Minute), Status: "complete"},
		{ID: "c1", ScenarioSlug: "s", Role: SnapshotRoleCapture, TriggerType: "manual", CreatedAt: time.Now().UTC().Add(-1 * time.Minute), Status: "complete"},
		{ID: "c2", ScenarioSlug: "s", Role: SnapshotRoleCapture, TriggerType: "manual", CreatedAt: time.Now().UTC(), Status: "complete"},
	} {
		if err := store.SaveSnapshotSet(1, m, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
			t.Fatalf("save %s: %v", m.ID, err)
		}
	}

	// Delete all captures — baseline should survive
	if err := store.DeleteSnapshotsByRole(1, "s", SnapshotRoleCapture); err != nil {
		t.Fatalf("DeleteSnapshotsByRole: %v", err)
	}

	list, err := store.ListSnapshotSets(1, "s")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 snapshot (baseline), got %d", len(list))
	}
	if list[0].ID != "b1" {
		t.Errorf("expected baseline b1, got %s", list[0].ID)
	}
}

func TestStorage_ClearScenarioSnapshots(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	for _, id := range []string{"a", "b"} {
		m := SnapshotSetMeta{ID: id, ScenarioSlug: "target", TriggerType: "manual", CreatedAt: time.Now().UTC(), Status: "complete"}
		if err := store.SaveSnapshotSet(1, m, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	// Save to a different scenario
	other := SnapshotSetMeta{ID: "x", ScenarioSlug: "other", TriggerType: "manual", CreatedAt: time.Now().UTC(), Status: "complete"}
	if err := store.SaveSnapshotSet(1, other, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
		t.Fatalf("save other: %v", err)
	}

	if err := store.ClearScenarioSnapshots(1, "target"); err != nil {
		t.Fatalf("clear: %v", err)
	}

	list, _ := store.ListSnapshotSets(1, "target")
	if len(list) != 0 {
		t.Errorf("expected 0 target snapshots, got %d", len(list))
	}
	otherList, _ := store.ListSnapshotSets(1, "other")
	if len(otherList) != 1 {
		t.Errorf("expected other scenario untouched (1), got %d", len(otherList))
	}
}

func TestStorage_RetentionPreservesBaseline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	// Save a baseline as the oldest snapshot
	baseline := SnapshotSetMeta{
		ID: "baseline", ScenarioSlug: "s", Role: SnapshotRoleBaseline,
		TriggerType: "manual", ScreenshotCount: 1,
		CreatedAt: time.Now().UTC().Add(-20 * time.Minute), Status: "complete",
	}
	if err := store.SaveSnapshotSet(1, baseline, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
		t.Fatalf("save baseline: %v", err)
	}

	// Save 11 captures (exceeds retention of 10 total)
	for i := 0; i < 11; i++ {
		m := SnapshotSetMeta{
			ID: fmt.Sprintf("c%d", i), ScenarioSlug: "s", Role: SnapshotRoleCapture,
			TriggerType: "manual", ScreenshotCount: 1,
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute), Status: "complete",
		}
		if err := store.SaveSnapshotSet(1, m, map[string][]byte{"_root_.png": {0x89}}, nil); err != nil {
			t.Fatalf("save c%d: %v", i, err)
		}
	}

	list, err := store.ListSnapshotSets(1, "s")
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// Baseline must survive retention regardless of age
	foundBaseline := false
	for _, m := range list {
		if m.ID == "baseline" {
			foundBaseline = true
			break
		}
	}
	if !foundBaseline {
		t.Error("baseline was evicted by retention — it should be preserved")
	}
}

func TestStorage_EffectiveRole_LegacySnapshots(t *testing.T) {
	t.Parallel()

	// Legacy snapshot without role field defaults to "capture"
	legacy := SnapshotSetMeta{ID: "old", Role: ""}
	if got := legacy.EffectiveRole(); got != SnapshotRoleCapture {
		t.Errorf("expected legacy role %q, got %q", SnapshotRoleCapture, got)
	}

	// Explicit roles pass through
	b := SnapshotSetMeta{Role: SnapshotRoleBaseline}
	if got := b.EffectiveRole(); got != SnapshotRoleBaseline {
		t.Errorf("expected %q, got %q", SnapshotRoleBaseline, got)
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

func TestParsePresetFromFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantW     int
		wantH     int
		wantTheme string
		wantOK    bool
	}{
		{"light", "_root_@1440x900_light.png", 1440, 900, "light", true},
		{"dark", "_about_@390x844_dark.png", 390, 844, "dark", true},
		{"no preset", "_root_.png", 0, 0, "", false},
		{"old format no theme", "_root_@1440x900.png", 0, 0, "", false},
		{"not png", "_root_@1440x900_light.jpg", 0, 0, "", false},
		{"partial", "_root_@1440_light.png", 0, 0, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, h, theme, ok := parsePresetFromFilename(tc.input)
			if ok != tc.wantOK || w != tc.wantW || h != tc.wantH || theme != tc.wantTheme {
				t.Errorf("parsePresetFromFilename(%q) = (%d, %d, %q, %v), want (%d, %d, %q, %v)", tc.input, w, h, theme, ok, tc.wantW, tc.wantH, tc.wantTheme, tc.wantOK)
			}
		})
	}
}

func TestStorage_GetSnapshotSet_ParsesPresetDimensions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewVisualCaptureStorage(testStorageResolver(t, dir), OSFileIO{})

	meta := SnapshotSetMeta{
		ID:              "vp-1",
		ScenarioSlug:    "test-app",
		Role:            SnapshotRoleBaseline,
		TriggerType:     "manual",
		Pages:           []string{"/"},
		ScreenshotCount: 2,
		Presets:         []CapturePreset{{Name: "Desktop Light", Width: 1440, Height: 900, Theme: "light"}, {Name: "Mobile Dark", Width: 390, Height: 844, Theme: "dark"}},
		CreatedAt:       time.Now().UTC(),
		Status:          "complete",
	}
	screenshots := map[string][]byte{
		"_root_@1440x900_light.png": {0x89, 0x50, 0x4E, 0x47},
		"_root_@390x844_dark.png":   {0x89, 0x50, 0x4E, 0x47},
	}
	if err := store.SaveSnapshotSet(1, meta, screenshots, nil); err != nil {
		t.Fatalf("SaveSnapshotSet: %v", err)
	}

	detail, err := store.GetSnapshotSet(1, "test-app", "vp-1")
	if err != nil {
		t.Fatalf("GetSnapshotSet: %v", err)
	}
	if len(detail.Screenshots) != 2 {
		t.Fatalf("expected 2 screenshots, got %d", len(detail.Screenshots))
	}

	foundDesktop := false
	foundMobile := false
	for _, sf := range detail.Screenshots {
		if sf.ViewportWidth == 1440 && sf.ViewportHeight == 900 && sf.Theme == "light" {
			foundDesktop = true
		}
		if sf.ViewportWidth == 390 && sf.ViewportHeight == 844 && sf.Theme == "dark" {
			foundMobile = true
		}
	}
	if !foundDesktop {
		t.Error("expected screenshot with preset 1440x900 light")
	}
	if !foundMobile {
		t.Error("expected screenshot with preset 390x844 dark")
	}
}
