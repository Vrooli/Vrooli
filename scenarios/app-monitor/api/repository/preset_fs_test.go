package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func newTestPresetRepo(t *testing.T) *FilePresetRepository {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}
	opts := storage.Options{
		ScenarioID:   "app-monitor",
		RootOverride: t.TempDir(),
	}
	return NewFilePresetRepositoryWithOpts(resolver, opts)
}

func TestCreateAndGetPreset(t *testing.T) {
	repo := newTestPresetRepo(t)
	ctx := context.Background()

	preset := &WorkspacePreset{
		Name:            "My Layout",
		Color:           "#ff0000",
		InteractionMode: "browse",
		WorkspaceZoom:   1.0,
		PaneApps:        []string{"app1", "app2"},
		ColumnFractions: []float64{0.5, 0.5},
		RowFractions:    []float64{1.0},
	}

	err := repo.CreatePreset(ctx, preset)
	if err != nil {
		t.Fatalf("CreatePreset failed: %v", err)
	}

	if preset.ID == "" {
		t.Fatal("Expected preset ID to be set after creation")
	}
	if preset.CreatedAt.IsZero() {
		t.Fatal("Expected CreatedAt to be set")
	}
	if preset.UpdatedAt.IsZero() {
		t.Fatal("Expected UpdatedAt to be set")
	}

	got, err := repo.GetPreset(ctx, preset.ID)
	if err != nil {
		t.Fatalf("GetPreset failed: %v", err)
	}

	if got.Name != "My Layout" {
		t.Errorf("Expected name 'My Layout', got %q", got.Name)
	}
	if got.Color != "#ff0000" {
		t.Errorf("Expected color '#ff0000', got %q", got.Color)
	}
	if got.InteractionMode != "browse" {
		t.Errorf("Expected interaction_mode 'browse', got %q", got.InteractionMode)
	}
	if got.WorkspaceZoom != 1.0 {
		t.Errorf("Expected workspace_zoom 1.0, got %f", got.WorkspaceZoom)
	}
	if len(got.PaneApps) != 2 {
		t.Errorf("Expected 2 pane_apps, got %d", len(got.PaneApps))
	}
	if len(got.ColumnFractions) != 2 {
		t.Errorf("Expected 2 column_fractions, got %d", len(got.ColumnFractions))
	}
}

func TestListPresets(t *testing.T) {
	repo := newTestPresetRepo(t)
	ctx := context.Background()

	names := []string{"First", "Second", "Third"}
	for _, name := range names {
		preset := &WorkspacePreset{
			Name:            name,
			Color:           "#000000",
			InteractionMode: "browse",
			WorkspaceZoom:   1.0,
			PaneApps:        []string{"app1"},
			ColumnFractions: []float64{1.0},
			RowFractions:    []float64{1.0},
		}
		if err := repo.CreatePreset(ctx, preset); err != nil {
			t.Fatalf("CreatePreset(%s) failed: %v", name, err)
		}
		// Small sleep to ensure distinct UpdatedAt timestamps
		time.Sleep(10 * time.Millisecond)
	}

	list, err := repo.ListPresets(ctx)
	if err != nil {
		t.Fatalf("ListPresets failed: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("Expected 3 presets, got %d", len(list))
	}

	// Should be sorted newest first
	if list[0].Name != "Third" {
		t.Errorf("Expected newest first to be 'Third', got %q", list[0].Name)
	}
	if list[2].Name != "First" {
		t.Errorf("Expected oldest last to be 'First', got %q", list[2].Name)
	}
}

func TestUpdatePreset(t *testing.T) {
	repo := newTestPresetRepo(t)
	ctx := context.Background()

	preset := &WorkspacePreset{
		Name:            "Original",
		Color:           "#000000",
		InteractionMode: "browse",
		WorkspaceZoom:   1.0,
		PaneApps:        []string{"app1"},
		ColumnFractions: []float64{1.0},
		RowFractions:    []float64{1.0},
	}

	if err := repo.CreatePreset(ctx, preset); err != nil {
		t.Fatalf("CreatePreset failed: %v", err)
	}

	originalCreatedAt := preset.CreatedAt
	originalUpdatedAt := preset.UpdatedAt

	// Small sleep to ensure distinct timestamps
	time.Sleep(10 * time.Millisecond)

	preset.Name = "Updated"
	preset.Color = "#ffffff"

	if err := repo.UpdatePreset(ctx, preset); err != nil {
		t.Fatalf("UpdatePreset failed: %v", err)
	}

	got, err := repo.GetPreset(ctx, preset.ID)
	if err != nil {
		t.Fatalf("GetPreset failed: %v", err)
	}

	if got.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got %q", got.Name)
	}
	if got.Color != "#ffffff" {
		t.Errorf("Expected color '#ffffff', got %q", got.Color)
	}
	if !got.CreatedAt.Equal(originalCreatedAt) {
		t.Error("Expected CreatedAt to be preserved")
	}
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be newer than original")
	}
}

func TestDeletePreset(t *testing.T) {
	repo := newTestPresetRepo(t)
	ctx := context.Background()

	preset := &WorkspacePreset{
		Name:            "ToDelete",
		Color:           "#000000",
		InteractionMode: "browse",
		WorkspaceZoom:   1.0,
		PaneApps:        []string{"app1"},
		ColumnFractions: []float64{1.0},
		RowFractions:    []float64{1.0},
	}

	if err := repo.CreatePreset(ctx, preset); err != nil {
		t.Fatalf("CreatePreset failed: %v", err)
	}

	if err := repo.DeletePreset(ctx, preset.ID); err != nil {
		t.Fatalf("DeletePreset failed: %v", err)
	}

	_, err := repo.GetPreset(ctx, preset.ID)
	if err == nil {
		t.Fatal("Expected error after deleting preset")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist, got: %v", err)
	}
}

func TestGetPresetNotFound(t *testing.T) {
	repo := newTestPresetRepo(t)
	ctx := context.Background()

	_, err := repo.GetPreset(ctx, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("Expected error for non-existent preset")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist, got: %v", err)
	}
}
