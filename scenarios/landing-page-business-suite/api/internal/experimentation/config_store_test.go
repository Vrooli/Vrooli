package experimentation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigStoreVariantFilePathRejectsTraversal(t *testing.T) {
	store := &ConfigStore{variantsDir: t.TempDir()}

	path, err := store.variantFilePath("control")
	if err != nil {
		t.Fatalf("valid slug returned error: %v", err)
	}
	if got, want := filepath.Base(path), "control.json"; got != want {
		t.Fatalf("variant filename = %q, want %q", got, want)
	}
	for _, slug := range []string{"../outside", "control/other", "control.json", "Control"} {
		if _, err := store.variantFilePath(slug); err == nil {
			t.Fatalf("slug %q unexpectedly produced a path", slug)
		}
	}
}

func TestConfigStore_LoadAll(t *testing.T) {
	cs := setupTestConfigStore(t)

	// Verify variants were loaded
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Error("Expected at least one variant to be loaded")
	}

	// Verify branding was loaded
	branding := cs.GetBranding()
	if branding == nil {
		t.Fatal("Expected branding to be loaded")
	}
	if branding.SiteName == "" {
		t.Error("Expected branding to have a site name")
	}
}

func TestConfigStore_GetVariant(t *testing.T) {
	cs := setupTestConfigStore(t)

	// Get a known variant (control is always present)
	variant, err := cs.GetVariant("control")
	if err != nil {
		t.Fatalf("Failed to get control variant: %v", err)
	}

	if variant.Variant.Slug != "control" {
		t.Errorf("Expected slug 'control', got %q", variant.Variant.Slug)
	}

	// Test not found case
	_, err = cs.GetVariant("nonexistent-variant-slug")
	if err == nil {
		t.Error("Expected error for nonexistent variant")
	}
}

func TestConfigStore_ListVariants(t *testing.T) {
	cs := setupTestConfigStore(t)

	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Error("Expected at least one variant")
	}

	// Verify variants are sorted by slug
	for i := 1; i < len(variants); i++ {
		if variants[i-1].Variant.Slug > variants[i].Variant.Slug {
			t.Errorf("Variants not sorted: %s > %s", variants[i-1].Variant.Slug, variants[i].Variant.Slug)
		}
	}
}

func TestConfigStore_GetBranding(t *testing.T) {
	cs := setupTestConfigStore(t)

	branding := cs.GetBranding()
	if branding == nil {
		t.Fatal("Expected branding to be returned")
	}

	// Verify required fields
	if branding.SiteName == "" {
		t.Error("Expected site name to be set")
	}
	if branding.ID != 1 {
		t.Error("Expected branding ID to be 1 (singleton)")
	}
}

func TestConfigStore_SaveAndLoadVariant(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	variantsDir := filepath.Join(tmpDir, "variants")
	if err := os.MkdirAll(variantsDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp variants dir: %v", err)
	}

	brandingPath := filepath.Join(tmpDir, "branding.json")
	brandingData := []byte(`{"site_name": "Test Site"}`)
	if err := os.WriteFile(brandingPath, brandingData, 0o644); err != nil {
		t.Fatalf("Failed to write branding file: %v", err)
	}

	// Use defaultVariantSpace for axes validation
	cs := NewConfigStore(variantsDir, brandingPath, DefaultVariantSpace())
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create a test variant with axes matching variant_space.json
	testVariant := &VariantSnapshot{
		Variant: VariantSnapshotMeta{
			Slug:        "test-variant",
			Name:        "Test Variant",
			Description: "A test variant",
			Axes: map[string]string{
				"persona":         "silentFounder",
				"jtbd":            "automation",
				"conversionStyle": "emotional",
			},
			HeaderConfig: LandingHeaderConfig{
				Branding: HeaderBrandingConfig{
					Mode:  "logo_and_name",
					Label: "Test Variant",
				},
			},
		},
		Sections: []VariantSection{
			{
				SectionType: "hero",
				Content:     json.RawMessage(`{"title": "Hero Title"}`),
				Order:       1,
				Enabled:     true,
			},
		},
	}

	// Save the variant
	if err := cs.SaveVariant("test-variant", testVariant); err != nil {
		t.Fatalf("Failed to save variant: %v", err)
	}

	// Verify file was created
	variantPath := filepath.Join(variantsDir, "test-variant.json")
	if _, err := os.Stat(variantPath); os.IsNotExist(err) {
		t.Error("Variant file was not created")
	}

	// Load the variant back
	loaded, err := cs.GetVariant("test-variant")
	if err != nil {
		t.Fatalf("Failed to get saved variant: %v", err)
	}

	if loaded.Variant.Name != "Test Variant" {
		t.Errorf("Expected name 'Test Variant', got %q", loaded.Variant.Name)
	}
	if len(loaded.Sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(loaded.Sections))
	}
}

func TestConfigStore_DeleteVariant(t *testing.T) {
	tmpDir := t.TempDir()
	variantsDir := filepath.Join(tmpDir, "variants")
	if err := os.MkdirAll(variantsDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp variants dir: %v", err)
	}

	brandingPath := filepath.Join(tmpDir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name": "Test"}`), 0o644); err != nil {
		t.Fatalf("Failed to write branding file: %v", err)
	}

	// Create a variant file with axes matching variant_space.json
	variantData := `{
		"variant": {
			"slug": "delete-me",
			"name": "Delete Me",
			"axes": {
				"persona": "silentFounder",
				"jtbd": "automation",
				"conversionStyle": "emotional"
			}
		},
		"sections": []
	}`
	variantPath := filepath.Join(variantsDir, "delete-me.json")
	if err := os.WriteFile(variantPath, []byte(variantData), 0o644); err != nil {
		t.Fatalf("Failed to write variant file: %v", err)
	}

	cs := NewConfigStore(variantsDir, brandingPath, DefaultVariantSpace())
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify variant exists
	if _, err := cs.GetVariant("delete-me"); err != nil {
		t.Fatalf("Variant should exist before deletion: %v", err)
	}

	// Delete the variant
	if err := cs.DeleteVariant("delete-me"); err != nil {
		t.Fatalf("Failed to delete variant: %v", err)
	}

	// Verify variant is gone from cache
	_, err := cs.GetVariant("delete-me")
	if err == nil {
		t.Error("Expected error when getting deleted variant")
	}

	// Verify file is deleted
	if _, err := os.Stat(variantPath); !os.IsNotExist(err) {
		t.Error("Variant file should have been deleted")
	}

	// Try to delete nonexistent variant
	err = cs.DeleteVariant("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent variant")
	}
}

func TestConfigStore_SaveBranding(t *testing.T) {
	tmpDir := t.TempDir()
	brandingPath := filepath.Join(tmpDir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name": "Original"}`), 0o644); err != nil {
		t.Fatalf("Failed to write branding file: %v", err)
	}

	cs := NewConfigStore("", brandingPath, nil)
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Update branding
	tagline := "New Tagline"
	newBranding := &SiteBranding{
		SiteName: "Updated Site",
		Tagline:  &tagline,
	}

	if err := cs.SaveBranding(newBranding); err != nil {
		t.Fatalf("Failed to save branding: %v", err)
	}

	// Verify cache was updated
	current := cs.GetBranding()
	if current.SiteName != "Updated Site" {
		t.Errorf("Expected site name 'Updated Site', got %q", current.SiteName)
	}

	// Verify file was updated
	data, err := os.ReadFile(brandingPath)
	if err != nil {
		t.Fatalf("Failed to read branding file: %v", err)
	}

	var saved map[string]interface{}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to parse branding JSON: %v", err)
	}

	if saved["site_name"] != "Updated Site" {
		t.Errorf("Expected saved site_name 'Updated Site', got %v", saved["site_name"])
	}
	if saved["tagline"] != "New Tagline" {
		t.Errorf("Expected saved tagline 'New Tagline', got %v", saved["tagline"])
	}
}

func TestConfigStore_UpdateBranding(t *testing.T) {
	tmpDir := t.TempDir()
	brandingPath := filepath.Join(tmpDir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name": "Original", "tagline": "Original Tagline"}`), 0o644); err != nil {
		t.Fatalf("Failed to write branding file: %v", err)
	}

	cs := NewConfigStore("", brandingPath, nil)
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Partial update - only update tagline
	newTagline := "Updated Tagline"
	updated, err := cs.UpdateBranding(&BrandingUpdateRequest{
		Tagline: &newTagline,
	})
	if err != nil {
		t.Fatalf("Failed to update branding: %v", err)
	}

	// Verify site name was preserved
	if updated.SiteName != "Original" {
		t.Errorf("Expected site name 'Original', got %q", updated.SiteName)
	}

	// Verify tagline was updated
	if updated.Tagline == nil || *updated.Tagline != "Updated Tagline" {
		t.Errorf("Expected tagline 'Updated Tagline', got %v", updated.Tagline)
	}
}

func TestConfigStore_ClearBrandingField(t *testing.T) {
	tmpDir := t.TempDir()
	brandingPath := filepath.Join(tmpDir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name": "Test", "tagline": "Clear Me"}`), 0o644); err != nil {
		t.Fatalf("Failed to write branding file: %v", err)
	}

	cs := NewConfigStore("", brandingPath, nil)
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify tagline exists
	branding := cs.GetBranding()
	if branding.Tagline == nil {
		t.Fatal("Expected tagline to be set initially")
	}

	// Clear the tagline
	if err := cs.ClearBrandingField("tagline"); err != nil {
		t.Fatalf("Failed to clear branding field: %v", err)
	}

	// Verify tagline is cleared
	branding = cs.GetBranding()
	if branding.Tagline != nil {
		t.Errorf("Expected tagline to be nil after clearing, got %v", branding.Tagline)
	}

	// Verify file was updated
	data, err := os.ReadFile(brandingPath)
	if err != nil {
		t.Fatalf("Failed to read branding file: %v", err)
	}

	var saved map[string]interface{}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to parse branding JSON: %v", err)
	}

	if _, exists := saved["tagline"]; exists {
		t.Error("Expected tagline to be removed from saved file")
	}
}

func TestConfigStore_VariantCount(t *testing.T) {
	cs := setupTestConfigStore(t)

	count := cs.VariantCount()
	variants := cs.ListVariants()

	if count != len(variants) {
		t.Errorf("VariantCount() = %d, but ListVariants() returned %d items", count, len(variants))
	}
}

func TestConfigStore_Paths(t *testing.T) {
	cs := NewConfigStore("/test/variants", "/test/branding.json", nil)

	if cs.GetVariantsDir() != "/test/variants" {
		t.Errorf("Expected variants dir '/test/variants', got %q", cs.GetVariantsDir())
	}

	if cs.GetBrandingPath() != "/test/branding.json" {
		t.Errorf("Expected branding path '/test/branding.json', got %q", cs.GetBrandingPath())
	}
}
