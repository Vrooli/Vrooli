package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRequirementsForEntity(t *testing.T) {
	tests := []struct {
		name         string
		entityType   string
		entityName   string
		wantErr      bool
		wantMinItems int
	}{
		{
			name:         "load prd-control-tower requirements",
			entityType:   "scenario",
			entityName:   "prd-control-tower",
			wantErr:      false,
			wantMinItems: 0, // May or may not have requirements
		},
		{
			name:       "non-existent entity",
			entityType: "scenario",
			entityName: "does-not-exist-xyz",
			wantErr:    false, // Returns empty, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, err := loadRequirementsForEntity(tt.entityType, tt.entityName)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadRequirementsForEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(groups) < tt.wantMinItems {
				t.Errorf("expected at least %d groups, got %d", tt.wantMinItems, len(groups))
			}

			// Verify caching
			if !tt.wantErr {
				groupsCached, err := loadRequirementsForEntity(tt.entityType, tt.entityName)
				if err != nil {
					t.Errorf("cached load failed: %v", err)
				}
				if len(groupsCached) != len(groups) {
					t.Errorf("cached result differs: got %d groups, want %d", len(groupsCached), len(groups))
				}
			}
		})
	}
}

func TestParseRequirementGroups(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test index.json
	indexData := requirementsFile{
		Metadata: map[string]any{
			"description": "Test requirements",
		},
		Requirements: []RequirementRecordInput{
			{
				ID:          "TEST-001",
				Category:    "foundation",
				PRDRef:      "Test > Requirement",
				Title:       "Test requirement",
				Description: "A test requirement",
				Status:      "complete",
				Criticality: "P0",
			},
		},
		Imports: []string{"child/child.json"},
	}

	indexPath := filepath.Join(tmpDir, "index.json")
	indexBytes, _ := json.MarshalIndent(indexData, "", "  ")
	if err := os.WriteFile(indexPath, indexBytes, 0644); err != nil {
		t.Fatalf("failed to create test index.json: %v", err)
	}

	// Create child directory and file
	childDir := filepath.Join(tmpDir, "child")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}

	childData := requirementsFile{
		Metadata: map[string]any{
			"description": "Child requirements",
		},
		Requirements: []RequirementRecordInput{
			{
				ID:          "TEST-CHILD-001",
				Category:    "child",
				Title:       "Child requirement",
				Description: "A child requirement",
				Status:      "in_progress",
				Criticality: "P1",
			},
		},
	}

	childPath := filepath.Join(childDir, "child.json")
	childBytes, _ := json.MarshalIndent(childData, "", "  ")
	if err := os.WriteFile(childPath, childBytes, 0644); err != nil {
		t.Fatalf("failed to create test child.json: %v", err)
	}

	// Test parsing
	groups, err := parseRequirementGroups(tmpDir, "index.json", map[string]bool{})
	if err != nil {
		t.Fatalf("parseRequirementGroups() error = %v", err)
	}

	if len(groups) == 0 {
		t.Fatal("expected at least one group")
	}

	// Check parent group
	parentGroup := groups[0]
	if len(parentGroup.Requirements) != 1 {
		t.Errorf("expected 1 requirement in parent, got %d", len(parentGroup.Requirements))
	}

	if parentGroup.Requirements[0].ID != "TEST-001" {
		t.Errorf("expected requirement ID TEST-001, got %s", parentGroup.Requirements[0].ID)
	}

	// Check child groups
	if len(parentGroup.Children) != 1 {
		t.Errorf("expected 1 child group, got %d", len(parentGroup.Children))
	}

	if len(parentGroup.Children) > 0 {
		childGroup := parentGroup.Children[0]
		if len(childGroup.Requirements) != 1 {
			t.Errorf("expected 1 requirement in child, got %d", len(childGroup.Requirements))
		}

		if childGroup.Requirements[0].ID != "TEST-CHILD-001" {
			t.Errorf("expected requirement ID TEST-CHILD-001, got %s", childGroup.Requirements[0].ID)
		}
	}
}

func TestParseRequirementGroupsCircularImport(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create index.json that imports child.json
	indexData := requirementsFile{
		Requirements: []RequirementRecordInput{},
		Imports:      []string{"child.json"},
	}

	indexPath := filepath.Join(tmpDir, "index.json")
	indexBytes, _ := json.MarshalIndent(indexData, "", "  ")
	if err := os.WriteFile(indexPath, indexBytes, 0644); err != nil {
		t.Fatalf("failed to create test index.json: %v", err)
	}

	// Create child.json that imports index.json (circular)
	childData := requirementsFile{
		Requirements: []RequirementRecordInput{},
		Imports:      []string{"index.json"},
	}

	childPath := filepath.Join(tmpDir, "child.json")
	childBytes, _ := json.MarshalIndent(childData, "", "  ")
	if err := os.WriteFile(childPath, childBytes, 0644); err != nil {
		t.Fatalf("failed to create test child.json: %v", err)
	}

	// Test parsing - should detect circular import
	_, err := parseRequirementGroups(tmpDir, "index.json", map[string]bool{})
	if err == nil {
		t.Error("expected error for circular import, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' in error message, got: %v", err)
	}
}

func TestGroupNameFromPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"index.json", "index"},
		{"catalog/core.json", "core"},
		{"drafts/lifecycle.json", "lifecycle"},
		{"ai/generation.json", "generation"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := groupNameFromPath(tt.input)
			if got != tt.want {
				t.Errorf("groupNameFromPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
