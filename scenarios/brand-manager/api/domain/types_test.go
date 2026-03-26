package domain_test

import (
	"testing"
	"time"

	"brand-manager/domain"
)

// TestApplyPartialUpdate verifies that only non-zero fields overwrite existing values.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestApplyPartialUpdate(t *testing.T) {
	tests := []struct {
		name     string
		existing domain.Brand
		update   domain.Brand
		wantName string
		wantDesc string
		wantNil  bool // whether Identity should remain nil
	}{
		{
			name:     "overwrite name only",
			existing: domain.Brand{Name: "Old", Description: "Keep this"},
			update:   domain.Brand{Name: "New"},
			wantName: "New",
			wantDesc: "Keep this",
			wantNil:  true,
		},
		{
			name:     "empty fields are not applied",
			existing: domain.Brand{Name: "Keep", Description: "Also keep"},
			update:   domain.Brand{},
			wantName: "Keep",
			wantDesc: "Also keep",
			wantNil:  true,
		},
		{
			name:     "pointer fields overwrite when non-nil",
			existing: domain.Brand{Name: "B", Identity: &domain.Identity{DisplayName: "Old"}},
			update:   domain.Brand{Identity: &domain.Identity{DisplayName: "New"}},
			wantName: "B",
			wantNil:  false,
		},
		{
			name:     "nil pointer fields leave existing intact",
			existing: domain.Brand{Name: "B", Identity: &domain.Identity{DisplayName: "Keep"}},
			update:   domain.Brand{Name: "B2"},
			wantName: "B2",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.existing
			b.ApplyPartialUpdate(tt.update)

			if b.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", b.Name, tt.wantName)
			}
			if tt.wantDesc != "" && b.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", b.Description, tt.wantDesc)
			}
			if tt.wantNil && b.Identity != nil {
				t.Error("expected Identity to remain nil")
			}
			if !tt.wantNil && b.Identity == nil {
				t.Error("expected Identity to be non-nil")
			}
		})
	}
}

// TestScenarioStatusUnassigned verifies the unassigned factory.
// [REQ:BM-REQ-API-STATUS]
func TestScenarioStatusUnassigned(t *testing.T) {
	s := domain.ScenarioStatusUnassigned("my-app")
	if s.Scenario != "my-app" {
		t.Errorf("Scenario = %q, want %q", s.Scenario, "my-app")
	}
	if s.HasBrand {
		t.Error("HasBrand should be false")
	}
	if s.BrandID != nil {
		t.Error("BrandID should be nil")
	}
}

// TestScenarioStatusFromAssignment verifies the assigned factory.
// [REQ:BM-REQ-API-STATUS] [REQ:BM-REQ-ASSIGN-LINK]
func TestScenarioStatusFromAssignment(t *testing.T) {
	a := &domain.Assignment{
		BrandID:      "b1",
		BrandVersion: 3,
		Elements:     []string{"logo", "colors"},
	}
	s := domain.ScenarioStatusFromAssignment("test-sc", a)
	if !s.HasBrand {
		t.Error("HasBrand should be true")
	}
	if s.BrandID == nil || *s.BrandID != "b1" {
		t.Errorf("BrandID = %v, want 'b1'", s.BrandID)
	}
	if s.BrandVersion == nil || *s.BrandVersion != 3 {
		t.Errorf("BrandVersion = %v, want 3", s.BrandVersion)
	}
	if len(s.Elements) != 2 {
		t.Errorf("Elements count = %d, want 2", len(s.Elements))
	}
}

// TestApplyPartialUpdateAllFacets verifies that all pointer facets (Colors,
// Typography, Voice) are correctly overwritten or preserved during partial update.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestApplyPartialUpdateAllFacets(t *testing.T) {
	original := domain.Brand{
		Name:       "Original",
		Colors:     &domain.Colors{Primary: "#000"},
		Typography: &domain.Typography{HeadingFont: "Inter"},
		Voice:      &domain.Voice{Tone: "Formal"},
		Notes:      "original notes",
	}

	// Update only Colors and Notes
	original.ApplyPartialUpdate(domain.Brand{
		Colors: &domain.Colors{Primary: "#fff"},
		Notes:  "updated notes",
	})

	if original.Colors.Primary != "#fff" {
		t.Errorf("Colors.Primary = %q, want #fff", original.Colors.Primary)
	}
	if original.Typography.HeadingFont != "Inter" {
		t.Error("Typography should be preserved")
	}
	if original.Voice.Tone != "Formal" {
		t.Error("Voice should be preserved")
	}
	if original.Notes != "updated notes" {
		t.Errorf("Notes = %q, want 'updated notes'", original.Notes)
	}
}

// TestScenarioStatusFromAssignmentWithAppliedAt verifies the timestamp is preserved.
// [REQ:BM-REQ-API-STATUS] [REQ:BM-REQ-ASSIGN-TRACK]
func TestScenarioStatusFromAssignmentWithAppliedAt(t *testing.T) {
	now := time.Now().UTC()
	a := &domain.Assignment{
		BrandID:      "b1",
		BrandVersion: 2,
		AppliedAt:    now,
	}
	s := domain.ScenarioStatusFromAssignment("app", a)
	if s.AppliedAt == nil {
		t.Fatal("AppliedAt should not be nil")
	}
	if !s.AppliedAt.Equal(now) {
		t.Errorf("AppliedAt = %v, want %v", *s.AppliedAt, now)
	}
}

// TestScenarioStatusUnassignedFields verifies all fields for unassigned status.
// [REQ:BM-REQ-API-STATUS]
func TestScenarioStatusUnassignedFields(t *testing.T) {
	s := domain.ScenarioStatusUnassigned("test-app")
	if s.BrandVersion != nil {
		t.Error("BrandVersion should be nil for unassigned")
	}
	if s.AppliedAt != nil {
		t.Error("AppliedAt should be nil for unassigned")
	}
	if len(s.Elements) != 0 {
		t.Errorf("Elements should be empty, got %v", s.Elements)
	}
}

// TestScanReportStructure verifies the ScanReport type stores results correctly.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON]
func TestScanReportStructure(t *testing.T) {
	report := domain.ScanReport{
		Scenario:   "test-scenario",
		CSSMarkers: 3,
		JSONKeys:   2,
		Total:      5,
		Results: []domain.ScanResult{
			{File: "index.css", Line: 10, Type: "css", Marker: "/* brand-manager:primary */", Element: "primary"},
			{File: "manifest.json", Line: 5, Type: "json", Marker: "_brand_name", Element: "name"},
		},
	}

	if report.Total != 5 {
		t.Errorf("Total = %d, want 5", report.Total)
	}
	if len(report.Results) != 2 {
		t.Errorf("Results count = %d, want 2", len(report.Results))
	}
	if report.Results[0].Type != "css" {
		t.Errorf("first result type = %q, want css", report.Results[0].Type)
	}
	if report.Results[1].Type != "json" {
		t.Errorf("second result type = %q, want json", report.Results[1].Type)
	}
}

// TestBrandFilterDefaults verifies zero-value filter matches all.
// [REQ:BM-REQ-CRUD-READ]
func TestBrandFilterDefaults(t *testing.T) {
	f := domain.BrandFilter{}
	if f.NameContains != "" {
		t.Error("default NameContains should be empty")
	}
	if f.Limit != 0 {
		t.Error("default Limit should be 0 (meaning no limit)")
	}
	if f.Offset != 0 {
		t.Error("default Offset should be 0")
	}
}
