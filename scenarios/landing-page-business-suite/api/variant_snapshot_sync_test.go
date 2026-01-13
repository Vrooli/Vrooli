package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncVariantSnapshots_ContentOnlyPreservesWeightAndStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())
	cs := NewContentService(db)

	variant, err := vs.CreateVariant("content-only", "Content Only", "Initial", 12, defaultAxesSelection())
	if err != nil {
		t.Fatalf("create variant: %v", err)
	}

	weight := 77
	if _, err := vs.UpdateVariant(variant.Slug, nil, nil, &weight, nil, nil); err != nil {
		t.Fatalf("update weight: %v", err)
	}
	if err := vs.ArchiveVariant(variant.Slug); err != nil {
		t.Fatalf("archive variant: %v", err)
	}

	dir := t.TempDir()
	writeSnapshot(t, dir, VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug:        variant.Slug,
			Name:        "Content Only Updated",
			Description: "Updated description",
			Axes:        defaultAxesSelection(),
		},
		Sections: []VariantSectionInput{
			{
				SectionType: "hero",
				Content: map[string]interface{}{
					"title": "Updated hero",
				},
				Order:   1,
				Enabled: boolPtr(true),
			},
		},
	})

	t.Setenv("VARIANT_SNAPSHOT_DIR", dir)
	t.Setenv("VARIANT_SNAPSHOT_MODE", "content-only")

	if err := syncVariantSnapshots(vs, cs); err != nil {
		t.Fatalf("sync snapshots: %v", err)
	}

	updated, err := vs.GetVariantBySlug(variant.Slug)
	if err != nil {
		t.Fatalf("load variant: %v", err)
	}
	if updated.Weight != weight {
		t.Fatalf("expected weight %d, got %d", weight, updated.Weight)
	}
	if updated.Status != "archived" {
		t.Fatalf("expected status archived, got %s", updated.Status)
	}

	sections, err := cs.GetSections(int64(updated.ID))
	if err != nil {
		t.Fatalf("load sections: %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].SectionType != "hero" {
		t.Fatalf("expected hero section, got %s", sections[0].SectionType)
	}
}

func TestSyncVariantSnapshots_PrunesMissingSnapshotsByArchiving(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())
	cs := NewContentService(db)

	_, err := vs.CreateVariant("to-archive", "To Archive", "Initial", 15, defaultAxesSelection())
	if err != nil {
		t.Fatalf("create variant: %v", err)
	}

	dir := t.TempDir()
	writeSnapshot(t, dir, VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug:        "kept",
			Name:        "Kept Variant",
			Description: "Kept",
			Axes:        defaultAxesSelection(),
		},
		Sections: []VariantSectionInput{
			{
				SectionType: "hero",
				Content: map[string]interface{}{
					"title": "Kept hero",
				},
				Order:   1,
				Enabled: boolPtr(true),
			},
		},
	})

	t.Setenv("VARIANT_SNAPSHOT_DIR", dir)
	t.Setenv("VARIANT_SNAPSHOT_MODE", "content-only")
	t.Setenv("VARIANT_SNAPSHOT_PRUNE", "archive")

	if err := syncVariantSnapshots(vs, cs); err != nil {
		t.Fatalf("sync snapshots: %v", err)
	}

	archived, err := vs.GetVariantBySlug("to-archive")
	if err != nil {
		t.Fatalf("load variant: %v", err)
	}
	if archived.Status != "archived" {
		t.Fatalf("expected status archived, got %s", archived.Status)
	}
}

func TestSyncVariantSnapshots_ResurrectsDeletedVariantsWhenEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())
	cs := NewContentService(db)

	_, err := vs.CreateVariant("zombie", "Zombie", "Initial", 10, defaultAxesSelection())
	if err != nil {
		t.Fatalf("create variant: %v", err)
	}
	if err := vs.DeleteVariant("zombie"); err != nil {
		t.Fatalf("delete variant: %v", err)
	}

	dir := t.TempDir()
	writeSnapshot(t, dir, VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug:        "zombie",
			Name:        "Zombie Restored",
			Description: "Restored",
			Axes:        defaultAxesSelection(),
		},
		Sections: []VariantSectionInput{
			{
				SectionType: "hero",
				Content: map[string]interface{}{
					"title": "Resurrected hero",
				},
				Order:   1,
				Enabled: boolPtr(true),
			},
		},
	})

	t.Setenv("VARIANT_SNAPSHOT_DIR", dir)
	t.Setenv("VARIANT_SNAPSHOT_MODE", "content-only")
	t.Setenv("VARIANT_SNAPSHOT_ALLOW_RESURRECT", "true")

	if err := syncVariantSnapshots(vs, cs); err != nil {
		t.Fatalf("sync snapshots: %v", err)
	}

	resurrected, err := vs.GetVariantBySlug("zombie")
	if err != nil {
		t.Fatalf("load variant: %v", err)
	}
	if resurrected.Status != "active" {
		t.Fatalf("expected status active, got %s", resurrected.Status)
	}
}

func writeSnapshot(t *testing.T, dir string, snapshot VariantSnapshotInput) {
	t.Helper()

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	path := filepath.Join(dir, snapshot.Variant.Slug+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
}

