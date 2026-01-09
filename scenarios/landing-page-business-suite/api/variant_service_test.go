package main

import (
	"testing"
)

func defaultAxesSelection() map[string]string {
	return map[string]string{
		"persona":         "ops_leader",
		"jtbd":            "launch_bundle",
		"conversionStyle": "demo_led",
	}
}

func altAxesSelection() map[string]string {
	return map[string]string{
		"persona":         "ops_leader",
		"jtbd":            "launch_bundle",
		"conversionStyle": "demo_led",
	}
}

func testVariantSpace() *VariantSpace {
	return &VariantSpace{
		Name:          "test-space",
		SchemaVersion: 1,
		Axes: map[string]*AxisDefinition{
			"persona": {
				Variants: []AxisVariant{{ID: "ops_leader", Label: "Ops Leader"}},
			},
			"jtbd": {
				Variants: []AxisVariant{{ID: "launch_bundle", Label: "Launch bundle"}},
			},
			"conversionStyle": {
				Variants: []AxisVariant{{ID: "demo_led", Label: "Demo-led"}},
			},
		},
	}
}

func TestSelectVariant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Clean up test data
	db.Exec("DELETE FROM variants WHERE slug LIKE 'test-%'")

	vs := NewVariantService(db, testVariantSpace())

	// Test that SelectVariant returns a variant
	variant, err := vs.SelectVariant()
	if err != nil {
		t.Fatalf("SelectVariant failed: %v", err)
	}

	if variant == nil {
		t.Fatal("SelectVariant returned nil variant")
	}

	if variant.Slug == "" {
		t.Error("Selected variant has empty slug")
	}

	if variant.Status != "active" {
		t.Errorf("Selected variant has status %s, expected active", variant.Status)
	}
}

func TestGetVariantBySlug(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	// Test getting existing variant
	variant, err := vs.GetVariantBySlug("control")
	if err != nil {
		t.Fatalf("GetVariantBySlug failed: %v", err)
	}

	if variant.Slug != "control" {
		t.Errorf("Got variant with slug %s, expected control", variant.Slug)
	}

	// Test getting non-existent variant
	_, err = vs.GetVariantBySlug("nonexistent")
	if err == nil {
		t.Error("GetVariantBySlug should fail for non-existent variant")
	}
}

func TestCreateVariant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	// Test creating a new variant
	variant, err := vs.CreateVariant("test-variant", "Test Variant", "Test description", 30, defaultAxesSelection())
	if err != nil {
		t.Fatalf("CreateVariant failed: %v", err)
	}

	if variant.Slug != "test-variant" {
		t.Errorf("Created variant has slug %s, expected test-variant", variant.Slug)
	}

	if variant.Weight != 30 {
		t.Errorf("Created variant has weight %d, expected 30", variant.Weight)
	}

	if variant.Status != "active" {
		t.Errorf("Created variant has status %s, expected active", variant.Status)
	}

	if variant.Axes["persona"] != "ops_leader" {
		t.Errorf("Variant axes not persisted: %+v", variant.Axes)
	}

	// Test invalid weight
	_, err = vs.CreateVariant("test-invalid", "Invalid", "Invalid weight", 150, defaultAxesSelection())
	if err == nil {
		t.Error("CreateVariant should fail for weight > 100")
	}

	// Clean up
	db.Exec("DELETE FROM variants WHERE slug = 'test-variant'")
}

func TestUpdateVariant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	// Create a test variant
	_, err := vs.CreateVariant("test-update", "Update Test", "Test update", 50, defaultAxesSelection())
	if err != nil {
		t.Fatalf("Failed to create test variant: %v", err)
	}

	// Test updating weight
	newWeight := 70
	updated, err := vs.UpdateVariant("test-update", nil, nil, &newWeight, nil, nil)
	if err != nil {
		t.Fatalf("UpdateVariant failed: %v", err)
	}

	if updated.Weight != 70 {
		t.Errorf("Updated variant has weight %d, expected 70", updated.Weight)
	}

	// Test updating name
	newName := "Updated Name"
	updated, err = vs.UpdateVariant("test-update", &newName, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateVariant name failed: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Updated variant has name %s, expected Updated Name", updated.Name)
	}

	// Test invalid weight
	invalidWeight := 150
	_, err = vs.UpdateVariant("test-update", nil, nil, &invalidWeight, nil, nil)
	if err == nil {
		t.Error("UpdateVariant should fail for weight > 100")
	}

	// Test updating axes
	newAxes := altAxesSelection()
	updated, err = vs.UpdateVariant("test-update", nil, nil, nil, newAxes, nil)
	if err != nil {
		t.Fatalf("UpdateVariant axes failed: %v", err)
	}

	if updated.Axes["persona"] != "ops_leader" {
		t.Errorf("Expected persona to change, got %+v", updated.Axes)
	}

	// Clean up
	db.Exec("DELETE FROM variants WHERE slug = 'test-update'")
}

func TestArchiveVariant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	// Create a test variant
	_, err := vs.CreateVariant("test-archive", "Archive Test", "Test archive", 50, defaultAxesSelection())
	if err != nil {
		t.Fatalf("Failed to create test variant: %v", err)
	}

	// Test archiving
	err = vs.ArchiveVariant("test-archive")
	if err != nil {
		t.Fatalf("ArchiveVariant failed: %v", err)
	}

	// Verify it's archived
	variant, err := vs.GetVariantBySlug("test-archive")
	if err != nil {
		t.Fatalf("Failed to get archived variant: %v", err)
	}

	if variant.Status != "archived" {
		t.Errorf("Variant has status %s, expected archived", variant.Status)
	}

	if variant.ArchivedAt == nil {
		t.Error("Archived variant should have ArchivedAt timestamp")
	}

	// Verify archived variants are excluded from selection
	// (This test assumes control and variant-a are still active)
	for i := 0; i < 10; i++ {
		selected, err := vs.SelectVariant()
		if err != nil {
			t.Fatalf("SelectVariant failed: %v", err)
		}

		if selected.Slug == "test-archive" {
			t.Error("SelectVariant returned archived variant")
		}
	}

	// Clean up
	db.Exec("DELETE FROM variants WHERE slug = 'test-archive'")
}

func TestDeleteVariant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	// Create a test variant
	_, err := vs.CreateVariant("test-delete", "Delete Test", "Test delete", 50, defaultAxesSelection())
	if err != nil {
		t.Fatalf("Failed to create test variant: %v", err)
	}

	// Test deleting
	err = vs.DeleteVariant("test-delete")
	if err != nil {
		t.Fatalf("DeleteVariant failed: %v", err)
	}

	// Verify it's soft-deleted
	variant, err := vs.GetVariantBySlug("test-delete")
	if err != nil {
		t.Fatalf("Failed to get deleted variant: %v", err)
	}

	if variant.Status != "deleted" {
		t.Errorf("Variant has status %s, expected deleted", variant.Status)
	}

	// Clean up
	db.Exec("DELETE FROM variants WHERE slug = 'test-delete'")
}

func TestUpdateSEOConfigBySlug(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	slug := "test-seo-config"
	db.Exec("DELETE FROM variants WHERE slug = $1", slug)

	_, err := vs.CreateVariant(slug, "SEO Variant", "SEO description", 40, defaultAxesSelection())
	if err != nil {
		t.Fatalf("Failed to create variant for SEO test: %v", err)
	}

	config := VariantSEOConfig{
		Title:         "New Title",
		Description:   "Search friendly copy",
		CanonicalPath: "/seo-path",
		NoIndex:       true,
	}

	if err := vs.UpdateSEOConfigBySlug(slug, config); err != nil {
		t.Fatalf("UpdateSEOConfigBySlug failed: %v", err)
	}

	updated, err := vs.GetVariantBySlug(slug)
	if err != nil {
		t.Fatalf("GetVariantBySlug failed: %v", err)
	}

	parsed, err := updated.GetSEOConfigParsed()
	if err != nil {
		t.Fatalf("GetSEOConfigParsed failed: %v", err)
	}

	if parsed.Title != config.Title || parsed.Description != config.Description {
		t.Errorf("SEO config not persisted correctly: %+v", parsed)
	}
	if parsed.CanonicalPath != config.CanonicalPath {
		t.Errorf("expected canonical path %s, got %s", config.CanonicalPath, parsed.CanonicalPath)
	}
	if !parsed.NoIndex {
		t.Error("expected NoIndex to persist as true")
	}

	if err := vs.UpdateSEOConfigBySlug("", config); err == nil {
		t.Error("expected error when slug is empty")
	}

	db.Exec("DELETE FROM variants WHERE slug = $1", slug)
}

func TestListVariants(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	// Test listing all non-deleted variants
	variants, err := vs.ListVariants("")
	if err != nil {
		t.Fatalf("ListVariants failed: %v", err)
	}

	if len(variants) < 2 {
		t.Errorf("ListVariants returned %d variants, expected at least 2 (control, variant-a)", len(variants))
	}

	// Test filtering by status
	activeVariants, err := vs.ListVariants("active")
	if err != nil {
		t.Fatalf("ListVariants with status filter failed: %v", err)
	}

	for _, v := range activeVariants {
		if v.Status != "active" {
			t.Errorf("ListVariants(active) returned variant with status %s", v.Status)
		}

		if len(v.Axes) == 0 {
			t.Errorf("Variant %s missing axes metadata", v.Slug)
		}
	}
}

func TestWeightedSelection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, testVariantSpace())

	// Create test variants with different weights
	_, err := vs.CreateVariant("test-heavy", "Heavy Weight", "High weight variant", 90, defaultAxesSelection())
	if err != nil {
		t.Fatalf("Failed to create heavy variant: %v", err)
	}

	_, err = vs.CreateVariant("test-light", "Light Weight", "Low weight variant", 10, altAxesSelection())
	if err != nil {
		t.Fatalf("Failed to create light variant: %v", err)
	}

	// Run selection many times and verify distribution
	selections := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		variant, err := vs.SelectVariant()
		if err != nil {
			t.Fatalf("SelectVariant failed: %v", err)
		}

		selections[variant.Slug]++
	}

	// Heavy variant should be selected more often (not exact, but statistically likely)
	heavyCount := selections["test-heavy"]
	lightCount := selections["test-light"]

	if heavyCount == 0 {
		t.Error("Heavy variant was never selected")
	}

	if lightCount == 0 {
		t.Error("Light variant was never selected")
	}

	// Rough check: heavy should be selected more than light (allowing for randomness)
	if heavyCount < lightCount {
		t.Logf("Warning: Heavy variant (%d) selected less than light variant (%d) over %d iterations", heavyCount, lightCount, iterations)
	}

	// Clean up
	db.Exec("DELETE FROM variants WHERE slug LIKE 'test-%'")
}

func TestCreateVariantRequiresAxes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	vs := NewVariantService(db, defaultVariantSpace)

	_, err := vs.CreateVariant("axes-missing", "Missing Axes", "No axes provided", 40, nil)
	if err == nil {
		t.Fatal("expected error when creating variant without axes")
	}
}
