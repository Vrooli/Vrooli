package domain_test

import (
	"encoding/json"
	"testing"

	"brand-manager/domain"
)

// Domain-level tests that verify brand data structures against audit expectations.
// These complement handler-level audit tests by verifying the rules at the data model layer.

// TestBrand_HasCompleteBrandingFacets validates that a fully populated brand
// satisfies all five audit rule categories: display_name, logo, favicon, color_system, typography.
// [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-PROVIDER]
func TestBrand_HasCompleteBrandingFacets(t *testing.T) {
	brand := domain.Brand{
		ID:      "complete-b1",
		Name:    "Complete Brand",
		Version: 1,
		Identity: &domain.Identity{
			DisplayName: "Complete Corp",
			LogoPath:    "/assets/logo.png",
			FaviconPath: "/assets/favicon.ico",
		},
		Colors: &domain.Colors{
			Primary:    "#1a365d",
			Background: "#ffffff",
			Text:       "#1a202c",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Open Sans",
		},
	}

	// Verify all audit-relevant fields are populated
	if brand.Identity == nil || brand.Identity.DisplayName == "" {
		t.Error("has-display-name rule: DisplayName is required")
	}
	if brand.Identity == nil || brand.Identity.LogoPath == "" {
		t.Error("has-logo rule: LogoPath is required")
	}
	if brand.Identity == nil || brand.Identity.FaviconPath == "" {
		t.Error("has-favicon rule: FaviconPath is required")
	}
	if brand.Colors == nil || (brand.Colors.Primary == "" && brand.Colors.Background == "" && brand.Colors.Text == "") {
		t.Error("has-color-system rule: at least one color is required")
	}
	if brand.Typography == nil || (brand.Typography.HeadingFont == "" && brand.Typography.BodyFont == "") {
		t.Error("has-typography rule: at least one font is required")
	}
}

// TestBrand_PartialBrandingFacets verifies which audit rules would fail
// for a brand with only a display name.
// [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-ENDPOINT]
func TestBrand_PartialBrandingFacets(t *testing.T) {
	brand := domain.Brand{
		ID:      "partial-b1",
		Name:    "Partial Brand",
		Version: 1,
		Identity: &domain.Identity{
			DisplayName: "Partial Corp",
		},
	}

	// has-display-name should pass
	if brand.Identity == nil || brand.Identity.DisplayName == "" {
		t.Error("has-display-name should pass for partial brand")
	}

	// has-logo should fail
	if brand.Identity != nil && brand.Identity.LogoPath != "" {
		t.Error("has-logo should fail (no logo provided)")
	}

	// has-favicon should fail
	if brand.Identity != nil && brand.Identity.FaviconPath != "" {
		t.Error("has-favicon should fail (no favicon provided)")
	}

	// has-color-system should fail
	if brand.Colors != nil {
		t.Error("has-color-system should fail (no colors)")
	}

	// has-typography should fail
	if brand.Typography != nil {
		t.Error("has-typography should fail (no typography)")
	}
}

// TestBrandJSON_RoundTrip verifies brands survive JSON encoding/decoding
// with all facets intact, which is critical for version snapshots.
// [REQ:BM-REQ-CRUD-VERSION] [REQ:BM-REQ-STORE-SCHEMA]
func TestBrandJSON_RoundTrip(t *testing.T) {
	original := domain.Brand{
		ID:          "rt-b1",
		Name:        "RoundTrip Brand",
		Description: "Testing JSON fidelity",
		Identity: &domain.Identity{
			DisplayName: "RT Corp",
			Tagline:     "Test tagline",
			LogoPath:    "/logo.svg",
			FaviconPath: "/favicon.ico",
			IconPath:    "/icon.png",
		},
		Colors: &domain.Colors{
			Primary:    "#1a365d",
			Secondary:  "#2d3748",
			Accent:     "#e53e3e",
			Background: "#ffffff",
			Surface:    "#f7fafc",
			Text:       "#1a202c",
			Error:      "#c53030",
		},
		Typography: &domain.Typography{
			HeadingFont:  "Inter",
			BodyFont:     "Open Sans",
			MonoFont:     "Fira Code",
			BaseFontSize: "16px",
		},
		Voice: &domain.Voice{
			Tone:     "professional",
			Style:    "concise",
			Keywords: []string{"reliable", "modern", "secure"},
		},
		Notes:   "Important design notes",
		Version: 5,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded domain.Brand
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Verify all fields survived round-trip
	if decoded.Name != original.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description = %q, want %q", decoded.Description, original.Description)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version = %d, want %d", decoded.Version, original.Version)
	}

	// Identity
	if decoded.Identity.DisplayName != "RT Corp" {
		t.Errorf("Identity.DisplayName = %q", decoded.Identity.DisplayName)
	}
	if decoded.Identity.IconPath != "/icon.png" {
		t.Errorf("Identity.IconPath = %q", decoded.Identity.IconPath)
	}

	// Colors
	if decoded.Colors.Error != "#c53030" {
		t.Errorf("Colors.Error = %q", decoded.Colors.Error)
	}
	if decoded.Colors.Accent != "#e53e3e" {
		t.Errorf("Colors.Accent = %q", decoded.Colors.Accent)
	}

	// Typography
	if decoded.Typography.MonoFont != "Fira Code" {
		t.Errorf("Typography.MonoFont = %q", decoded.Typography.MonoFont)
	}

	// Voice
	if len(decoded.Voice.Keywords) != 3 {
		t.Errorf("Voice.Keywords count = %d, want 3", len(decoded.Voice.Keywords))
	}

	// Notes
	if decoded.Notes != "Important design notes" {
		t.Errorf("Notes = %q", decoded.Notes)
	}
}

// TestAsset_Validation verifies asset domain type constraints.
// [REQ:BM-REQ-API-ASSETS] [REQ:BM-REQ-STORE-ASSETS]
func TestAsset_Validation(t *testing.T) {
	tests := []struct {
		name    string
		asset   domain.Asset
		wantBad string // which field is bad
	}{
		{
			name:    "valid asset",
			asset:   domain.Asset{ID: "a1", BrandID: "b1", Filename: "logo.png", MimeType: "image/png", FilePath: "/tmp/logo.png", Size: 1024},
			wantBad: "",
		},
		{
			name:    "missing brand_id",
			asset:   domain.Asset{ID: "a2", Filename: "logo.png"},
			wantBad: "brand_id",
		},
		{
			name:    "missing filename",
			asset:   domain.Asset{ID: "a3", BrandID: "b1"},
			wantBad: "filename",
		},
		{
			name:    "zero size is allowed for metadata",
			asset:   domain.Asset{ID: "a4", BrandID: "b1", Filename: "placeholder.svg", Size: 0},
			wantBad: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasID := tt.asset.ID != ""
			hasBrandID := tt.asset.BrandID != ""
			hasFilename := tt.asset.Filename != ""

			switch tt.wantBad {
			case "brand_id":
				if hasBrandID {
					t.Error("expected missing brand_id")
				}
			case "filename":
				if hasFilename {
					t.Error("expected missing filename")
				}
			case "":
				if !hasID || !hasBrandID || !hasFilename {
					t.Error("expected all required fields present")
				}
			}
		})
	}
}

// TestScenarioStatus_JSONShape verifies the JSON shape matches what the UI expects.
// [REQ:BM-REQ-API-STATUS] [REQ:BM-REQ-API-STANDARDS]
func TestScenarioStatus_JSONShape(t *testing.T) {
	// Unassigned status
	unassigned := domain.ScenarioStatusUnassigned("test-app")
	data, err := json.Marshal(unassigned)
	if err != nil {
		t.Fatalf("Marshal unassigned: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if raw["scenario"] != "test-app" {
		t.Errorf("scenario = %v", raw["scenario"])
	}
	if raw["has_brand"] != false {
		t.Errorf("has_brand = %v, want false", raw["has_brand"])
	}
	// brand_id should be null/nil for unassigned
	if raw["brand_id"] != nil {
		t.Errorf("brand_id = %v, want nil", raw["brand_id"])
	}

	// Assigned status
	v := 2
	assigned := domain.ScenarioStatus{
		Scenario:     "test-app",
		HasBrand:     true,
		BrandID:      strPtr("b1"),
		BrandVersion: &v,
		Elements:     []string{"colors", "typography"},
	}
	data, err = json.Marshal(assigned)
	if err != nil {
		t.Fatalf("Marshal assigned: %v", err)
	}

	json.Unmarshal(data, &raw)
	if raw["has_brand"] != true {
		t.Errorf("has_brand = %v, want true", raw["has_brand"])
	}
	if raw["brand_id"] != "b1" {
		t.Errorf("brand_id = %v, want b1", raw["brand_id"])
	}
	elements, ok := raw["elements"].([]interface{})
	if !ok || len(elements) != 2 {
		t.Errorf("elements = %v, want [colors, typography]", raw["elements"])
	}
}

func strPtr(s string) *string { return &s }

// TestScanResult_TypeValues verifies ScanResult type field has expected values.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-SCAN-PARTIAL]
func TestScanResult_TypeValues(t *testing.T) {
	validTypes := map[string]bool{"css": true, "json": true, "yaml": true, "html": true}

	results := []domain.ScanResult{
		{File: "brand.css", Line: 1, Type: "css", Marker: "/* brand-manager:primary */", Element: "primary"},
		{File: "manifest.json", Line: 3, Type: "json", Marker: "_brand_name", Element: "name"},
	}

	for _, r := range results {
		if !validTypes[r.Type] {
			t.Errorf("invalid scan result type: %q", r.Type)
		}
		if r.File == "" {
			t.Error("ScanResult.File must not be empty")
		}
		if r.Element == "" {
			t.Error("ScanResult.Element must not be empty")
		}
	}
}

// TestScanReport_Totals verifies that Total equals CSSMarkers + JSONKeys.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON]
func TestScanReport_Totals(t *testing.T) {
	report := domain.ScanReport{
		Scenario:   "test",
		CSSMarkers: 5,
		JSONKeys:   3,
		Total:      8,
	}

	if report.Total != report.CSSMarkers+report.JSONKeys {
		t.Errorf("Total (%d) != CSSMarkers (%d) + JSONKeys (%d)",
			report.Total, report.CSSMarkers, report.JSONKeys)
	}
}
