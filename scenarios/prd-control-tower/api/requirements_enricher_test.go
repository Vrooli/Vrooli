package main

import (
	"testing"
)

func TestEnrichGroupsRecursive(t *testing.T) {
	issueMap := map[string]*PRDValidationIssue{
		"REQ-001": {
			RequirementID: "REQ-001",
			PRDRef:        "Functional Requirements > Authentication > Login",
			IssueType:     "missing_section",
			Message:       "Section not found in PRD",
			Suggestions:   []string{"Try: Functional Requirements > Login"},
		},
	}

	testRefs := map[string][]TestFileReference{
		"REQ-002": {
			{
				FilePath:      "ui/src/components/Auth.test.tsx",
				RequirementID: "REQ-002",
				Lines:         []int{42, 45},
				TestNames:     []string{"should authenticate user", "should handle login failure"},
			},
		},
	}

	groups := []RequirementGroup{
		{
			ID:   "group1",
			Name: "Group 1",
			Requirements: []RequirementRecord{
				{
					ID:          "REQ-001",
					Title:       "Login functionality",
					Description: "User should be able to login",
					PRDRef:      "Functional Requirements > Authentication > Login",
				},
				{
					ID:          "REQ-002",
					Title:       "Authentication",
					Description: "Secure authentication",
					PRDRef:      "Functional Requirements > Authentication",
				},
			},
			Children: []RequirementGroup{
				{
					ID:   "child1",
					Name: "Child Group",
					Requirements: []RequirementRecord{
						{
							ID:     "REQ-003",
							Title:  "Child requirement",
							PRDRef: "Functional Requirements > Other",
						},
					},
				},
			},
		},
	}

	enriched := enrichGroupsRecursive(groups, issueMap, testRefs)

	if len(enriched) != 1 {
		t.Fatalf("expected 1 enriched group, got %d", len(enriched))
	}

	// Check that REQ-001 has the PRD issue
	if enriched[0].Requirements[0].PRDRefIssue == nil {
		t.Error("expected REQ-001 to have PRD issue")
	} else if enriched[0].Requirements[0].PRDRefIssue.Message != "Section not found in PRD" {
		t.Errorf("expected message 'Section not found in PRD', got '%s'", enriched[0].Requirements[0].PRDRefIssue.Message)
	}

	// Check that REQ-002 has test references
	if len(enriched[0].Requirements[1].TestFiles) != 1 {
		t.Errorf("expected REQ-002 to have 1 test file reference, got %d", len(enriched[0].Requirements[1].TestFiles))
	}

	// Check that child group was enriched
	if len(enriched[0].Children) != 1 {
		t.Errorf("expected 1 child group, got %d", len(enriched[0].Children))
	}

	// Check that REQ-003 has no enrichment
	if enriched[0].Children[0].Requirements[0].PRDRefIssue != nil {
		t.Error("expected REQ-003 to have no PRD issue")
	}
}

func TestEnrichGroupsRecursiveEmpty(t *testing.T) {
	enriched := enrichGroupsRecursive([]RequirementGroup{}, map[string]*PRDValidationIssue{}, map[string][]TestFileReference{})

	if len(enriched) != 0 {
		t.Errorf("expected empty result, got %d groups", len(enriched))
	}
}

func TestEnrichGroupsRecursiveNoEnrichment(t *testing.T) {
	groups := []RequirementGroup{
		{
			ID:   "group1",
			Name: "Group 1",
			Requirements: []RequirementRecord{
				{
					ID:     "REQ-001",
					Title:  "Some requirement",
					PRDRef: "Functional Requirements > Something",
				},
			},
		},
	}

	enriched := enrichGroupsRecursive(groups, map[string]*PRDValidationIssue{}, map[string][]TestFileReference{})

	if len(enriched) != 1 {
		t.Fatalf("expected 1 group, got %d", len(enriched))
	}

	// Should have no enrichment
	if enriched[0].Requirements[0].PRDRefIssue != nil {
		t.Error("expected no PRD issue")
	}

	if len(enriched[0].Requirements[0].TestFiles) != 0 {
		t.Error("expected no test files")
	}
}
