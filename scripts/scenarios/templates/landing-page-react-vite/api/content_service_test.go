package main

import (
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
