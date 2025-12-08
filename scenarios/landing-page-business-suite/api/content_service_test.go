package main

import (
	"strings"
	"testing"
)

// [REQ:CUSTOM-SPLIT,CUSTOM-LIVE] - Content service powers the section editor
func TestContentService_GetSections(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)

	// Create test variant first
	variantID := createTestVariant(t, db)

	// Create test sections
	testContent := map[string]interface{}{
		"title":    "Test Hero",
		"subtitle": "Test Subtitle",
		"cta_text": "Get Started",
	}

	section1 := ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     testContent,
		Order:       0,
		Enabled:     true,
	}

	created1, err := service.CreateSection(section1)
	if err != nil {
		t.Fatalf("Failed to create section 1: %v", err)
	}

	section2 := ContentSection{
		VariantID:   variantID,
		SectionType: "features",
		Content:     testContent,
		Order:       1,
		Enabled:     true,
	}

	created2, err := service.CreateSection(section2)
	if err != nil {
		t.Fatalf("Failed to create section 2: %v", err)
	}

	// Test GetSections
	sections, err := service.GetSections(variantID)
	if err != nil {
		t.Fatalf("GetSections failed: %v", err)
	}

	if len(sections) != 2 {
		t.Errorf("Expected 2 sections, got %d", len(sections))
	}

	// Verify ordering
	if sections[0].Order != 0 || sections[1].Order != 1 {
		t.Error("Sections not ordered correctly")
	}

	// Cleanup
	_ = service.DeleteSection(created1.ID)
	_ = service.DeleteSection(created2.ID)
}

func TestContentService_GetPublicSections_FiltersDisabledAndOrders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	firstEnabled, _ := service.CreateSection(ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     map[string]interface{}{"title": "Visible hero"},
		Order:       1,
		Enabled:     true,
	})
	hidden, _ := service.CreateSection(ContentSection{
		VariantID:   variantID,
		SectionType: "features",
		Content:     map[string]interface{}{"title": "Hidden features"},
		Order:       0,
		Enabled:     false,
	})
	secondEnabled, _ := service.CreateSection(ContentSection{
		VariantID:   variantID,
		SectionType: "cta",
		Content:     map[string]interface{}{"title": "Visible CTA"},
		Order:       5,
		Enabled:     true,
	})

	sections, err := service.GetPublicSections(variantID)
	if err != nil {
		t.Fatalf("GetPublicSections failed: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 enabled sections, got %d", len(sections))
	}
	if sections[0].ID != firstEnabled.ID || sections[1].ID != secondEnabled.ID {
		t.Fatalf("expected sections ordered by display order, got IDs %d then %d", sections[0].ID, sections[1].ID)
	}
	for _, section := range sections {
		if !section.Enabled {
			t.Fatalf("expected only enabled sections in public payload, saw %+v", section)
		}
	}

	_ = service.DeleteSection(firstEnabled.ID)
	_ = service.DeleteSection(secondEnabled.ID)
	_ = service.DeleteSection(hidden.ID)
}

func TestContentService_GetSection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	testContent := map[string]interface{}{
		"title":    "Test Section",
		"subtitle": "Test Content",
	}

	section := ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     testContent,
		Order:       0,
		Enabled:     true,
	}

	created, err := service.CreateSection(section)
	if err != nil {
		t.Fatalf("Failed to create section: %v", err)
	}

	// Test GetSection
	retrieved, err := service.GetSection(created.ID)
	if err != nil {
		t.Fatalf("GetSection failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, retrieved.ID)
	}

	if retrieved.SectionType != "hero" {
		t.Errorf("Expected section type 'hero', got '%s'", retrieved.SectionType)
	}

	if retrieved.Content["title"] != "Test Section" {
		t.Errorf("Content not preserved correctly")
	}

	// Cleanup
	_ = service.DeleteSection(created.ID)
}

func TestContentService_GetSection_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)

	_, err := service.GetSection(99999)
	if err == nil {
		t.Error("Expected error for non-existent section, got nil")
	}
}

func TestContentService_UpdateSection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	testContent := map[string]interface{}{
		"title": "Original Title",
	}

	section := ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     testContent,
		Order:       0,
		Enabled:     true,
	}

	created, err := service.CreateSection(section)
	if err != nil {
		t.Fatalf("Failed to create section: %v", err)
	}

	// Update content
	updatedContent := map[string]interface{}{
		"title":    "Updated Title",
		"subtitle": "New Subtitle",
	}

	err = service.UpdateSection(created.ID, updatedContent)
	if err != nil {
		t.Fatalf("UpdateSection failed: %v", err)
	}

	// Verify update
	retrieved, err := service.GetSection(created.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated section: %v", err)
	}

	if retrieved.Content["title"] != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", retrieved.Content["title"])
	}

	if retrieved.Content["subtitle"] != "New Subtitle" {
		t.Errorf("Expected subtitle 'New Subtitle', got '%s'", retrieved.Content["subtitle"])
	}

	// Cleanup
	_ = service.DeleteSection(created.ID)
}

func TestContentService_CreateSection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	testContent := map[string]interface{}{
		"title":     "Hero Title",
		"subtitle":  "Hero Subtitle",
		"cta_text":  "Get Started",
		"image_url": "https://example.com/hero.jpg",
	}

	section := ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     testContent,
		Order:       0,
		Enabled:     true,
	}

	created, err := service.CreateSection(section)
	if err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("Expected non-zero ID after creation")
	}

	if created.VariantID != variantID {
		t.Errorf("Expected variant ID %d, got %d", variantID, created.VariantID)
	}

	if created.SectionType != "hero" {
		t.Errorf("Expected section type 'hero', got '%s'", created.SectionType)
	}

	if created.CreatedAt.IsZero() {
		t.Error("Expected non-zero created_at timestamp")
	}

	// Cleanup
	_ = service.DeleteSection(created.ID)
}

func TestContentService_CopySectionsFromVariant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	createVariant := func(slug string) int64 {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO variants (slug, name, description, weight, status, created_at, updated_at)
			VALUES ($1, $2, $3, 50, 'active', NOW(), NOW())
			RETURNING id
		`, slug, strings.Title(slug), "copy test variant").Scan(&id); err != nil {
			t.Fatalf("failed to insert variant %s: %v", slug, err)
		}
		t.Cleanup(func() { db.Exec(`DELETE FROM variants WHERE slug = $1`, slug) })
		return id
	}

	sourceVariantID := createVariant("copy-source")
	targetVariantID := createVariant("copy-target")

	hero, err := service.CreateSection(ContentSection{
		VariantID:   sourceVariantID,
		SectionType: "hero",
		Content:     map[string]interface{}{"title": "Source Hero"},
		Order:       2,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("failed to seed source hero: %v", err)
	}
	features, err := service.CreateSection(ContentSection{
		VariantID:   sourceVariantID,
		SectionType: "features",
		Content:     map[string]interface{}{"heading": "Capabilities"},
		Order:       1,
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("failed to seed source features: %v", err)
	}

	if err := service.CopySectionsFromVariant(sourceVariantID, targetVariantID); err != nil {
		t.Fatalf("CopySectionsFromVariant failed: %v", err)
	}

	targetSections, err := service.GetSections(targetVariantID)
	if err != nil {
		t.Fatalf("failed to read target sections: %v", err)
	}
	if len(targetSections) != 2 {
		t.Fatalf("expected 2 copied sections, got %d", len(targetSections))
	}
	if targetSections[0].VariantID != targetVariantID || targetSections[1].VariantID != targetVariantID {
		t.Fatalf("copied sections should belong to target variant, got %+v", targetSections)
	}
	if targetSections[0].ID == hero.ID || targetSections[1].ID == features.ID {
		t.Fatalf("expected new section records, saw source IDs in %v", targetSections)
	}
	if targetSections[0].Order != 1 || targetSections[1].Order != 2 {
		t.Fatalf("expected orders to be preserved, got %d then %d", targetSections[0].Order, targetSections[1].Order)
	}
	if targetSections[0].SectionType != "features" || targetSections[0].Enabled {
		t.Fatalf("expected features to remain disabled and ordered first, got %+v", targetSections[0])
	}
	if targetSections[1].Content["title"] != "Source Hero" {
		t.Fatalf("expected hero content to be copied, got %+v", targetSections[1].Content)
	}

	sourceSections, err := service.GetSections(sourceVariantID)
	if err != nil {
		t.Fatalf("failed to read source sections: %v", err)
	}
	if len(sourceSections) != 2 {
		t.Fatalf("source sections should remain intact, got %d", len(sourceSections))
	}
}

func TestContentService_DeleteSection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	testContent := map[string]interface{}{"title": "Test"}

	section := ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     testContent,
		Order:       0,
		Enabled:     true,
	}

	created, err := service.CreateSection(section)
	if err != nil {
		t.Fatalf("Failed to create section: %v", err)
	}

	// Delete section
	err = service.DeleteSection(created.ID)
	if err != nil {
		t.Fatalf("DeleteSection failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetSection(created.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted section, got nil")
	}
}

func TestContentService_DeleteSection_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)

	err := service.DeleteSection(99999)
	if err == nil {
		t.Error("Expected error when deleting non-existent section, got nil")
	}
}

func TestContentService_ContentJSON_Marshaling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	// Test with complex nested content
	testContent := map[string]interface{}{
		"title": "Complex Content",
		"features": []interface{}{
			map[string]interface{}{
				"name": "Feature 1",
				"icon": "check",
			},
			map[string]interface{}{
				"name": "Feature 2",
				"icon": "star",
			},
		},
		"metadata": map[string]interface{}{
			"author": "Test Author",
			"tags":   []string{"landing", "saas"},
		},
	}

	section := ContentSection{
		VariantID:   variantID,
		SectionType: "features",
		Content:     testContent,
		Order:       0,
		Enabled:     true,
	}

	created, err := service.CreateSection(section)
	if err != nil {
		t.Fatalf("Failed to create section with complex content: %v", err)
	}

	// Retrieve and verify complex content
	retrieved, err := service.GetSection(created.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve section: %v", err)
	}

	// Verify nested arrays
	features, ok := retrieved.Content["features"].([]interface{})
	if !ok {
		t.Error("Features array not preserved")
	}
	if len(features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(features))
	}

	// Cleanup
	_ = service.DeleteSection(created.ID)
}

func TestContentService_ReplaceSectionsTx_ReplacesAndRenumbers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	// Seed an existing section to ensure replacement clears previous rows.
	original, err := service.CreateSection(ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     map[string]interface{}{"title": "original"},
		Order:       99,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("failed to seed section: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	disabled := false
	err = service.ReplaceSectionsTx(tx, variantID, []VariantSectionInput{
		{SectionType: "hero", Content: map[string]interface{}{"title": "updated hero"}, Order: 0},
		{SectionType: "features", Content: map[string]interface{}{"items": []string{"one", "two"}}, Order: -2},
		{SectionType: "cta", Content: map[string]interface{}{"cta": "Join"}, Order: 5, Enabled: &disabled},
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("ReplaceSectionsTx failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	sections, err := service.GetSections(variantID)
	if err != nil {
		t.Fatalf("GetSections failed: %v", err)
	}

	if len(sections) != 3 {
		t.Fatalf("expected 3 sections after replacement, got %d", len(sections))
	}

	typeSummary := map[string]ContentSection{}
	for _, section := range sections {
		typeSummary[section.SectionType] = section
	}

	hero := typeSummary["hero"]
	features := typeSummary["features"]
	cta := typeSummary["cta"]
	if hero.Order != 2 || features.Order != 1 || cta.Order != 5 {
		t.Fatalf("expected orders hero=2, features=1, cta=5; got hero=%d features=%d cta=%d", hero.Order, features.Order, cta.Order)
	}
	if !hero.Enabled || !features.Enabled || cta.Enabled {
		t.Fatalf("expected enabled defaults to true and explicit false preserved, got hero=%t features=%t cta=%t", hero.Enabled, features.Enabled, cta.Enabled)
	}
	if hero.Content["title"] != "updated hero" || cta.Content["cta"] != "Join" || len(features.Content) == 0 {
		t.Fatalf("content did not persist: %+v", sections)
	}
	for _, section := range sections {
		if section.ID == original.ID {
			t.Fatalf("expected original section to be replaced, still found id %d", original.ID)
		}
	}
}

func TestContentService_ReplaceSectionsTx_ValidatesBeforeClearing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	valid, err := service.CreateSection(ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     map[string]interface{}{"title": "keep"},
		Order:       1,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("failed to seed section: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	validationErr := service.ReplaceSectionsTx(tx, variantID, []VariantSectionInput{
		{SectionType: "unknown", Content: map[string]interface{}{"title": "bad"}},
		{SectionType: "hero", Content: nil},
	})
	if validationErr == nil {
		tx.Rollback()
		t.Fatal("expected validation error for unsupported section type and missing content")
	}
	tx.Rollback()

	sections, err := service.GetSections(variantID)
	if err != nil {
		t.Fatalf("GetSections failed: %v", err)
	}
	if len(sections) != 1 || sections[0].ID != valid.ID {
		t.Fatalf("expected existing sections to remain after validation failure, got %+v", sections)
	}
	if !strings.Contains(validationErr.Error(), "section_type") {
		t.Fatalf("expected section type error, got %v", validationErr)
	}
}

func TestContentService_ReplaceSectionsTx_RequiresTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewContentService(db)
	variantID := createTestVariant(t, db)

	existing, err := service.CreateSection(ContentSection{
		VariantID:   variantID,
		SectionType: "hero",
		Content:     map[string]interface{}{"title": "keep me"},
		Order:       1,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("failed to seed existing section: %v", err)
	}

	err = service.ReplaceSectionsTx(nil, variantID, []VariantSectionInput{
		{SectionType: "hero", Content: map[string]interface{}{"title": "new"}, Order: 1},
	})
	if err == nil || !strings.Contains(err.Error(), "transaction required") {
		t.Fatalf("expected transaction required error, got %v", err)
	}

	sections, err := service.GetSections(variantID)
	if err != nil {
		t.Fatalf("GetSections failed: %v", err)
	}
	if len(sections) != 1 || sections[0].ID != existing.ID {
		t.Fatalf("expected existing sections to remain untouched, got %+v", sections)
	}
}
