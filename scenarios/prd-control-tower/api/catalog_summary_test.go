package main

import (
	"testing"
)

// Note: Using RequirementRecord as the type name (the actual type in requirements_types.go)

// [REQ:PCT-REQ-PARSE] Summarize requirement groups for catalog display
func TestSummarizeRequirementGroupsForCatalog(t *testing.T) {
	tests := []struct {
		name     string
		groups   []RequirementGroup
		expected CatalogRequirementSummary
	}{
		{
			name:   "empty groups",
			groups: []RequirementGroup{},
			expected: CatalogRequirementSummary{
				Total:      0,
				Completed:  0,
				InProgress: 0,
				Pending:    0,
				P0:         0,
				P1:         0,
				P2:         0,
			},
		},
		{
			name: "single group with complete requirements",
			groups: []RequirementGroup{
				{
					ID:   "group1",
					Name: "Core Features",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "complete", Criticality: "P0"},
						{ID: "REQ-002", Status: "done", Criticality: "P1"},
						{ID: "REQ-003", Status: "COMPLETE", Criticality: "P0"},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      3,
				Completed:  3,
				InProgress: 0,
				Pending:    0,
				P0:         2,
				P1:         1,
				P2:         0,
			},
		},
		{
			name: "group with in_progress requirements",
			groups: []RequirementGroup{
				{
					ID:   "group1",
					Name: "Features",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "in_progress", Criticality: "P0"},
						{ID: "REQ-002", Status: "in-progress", Criticality: "P1"},
						{ID: "REQ-003", Status: "IN_PROGRESS", Criticality: "P2"},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      3,
				Completed:  0,
				InProgress: 3,
				Pending:    0,
				P0:         1,
				P1:         1,
				P2:         1,
			},
		},
		{
			name: "group with pending requirements",
			groups: []RequirementGroup{
				{
					ID:   "group1",
					Name: "Future",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "pending", Criticality: "P0"},
						{ID: "REQ-002", Status: "not_started", Criticality: "P1"},
						{ID: "REQ-003", Status: "", Criticality: "P2"},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      3,
				Completed:  0,
				InProgress: 0,
				Pending:    3,
				P0:         1,
				P1:         1,
				P2:         1,
			},
		},
		{
			name: "nested groups",
			groups: []RequirementGroup{
				{
					ID:   "parent",
					Name: "Parent Group",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "complete", Criticality: "P0"},
					},
					Children: []RequirementGroup{
						{
							ID:   "child1",
							Name: "Child Group 1",
							Requirements: []RequirementRecord{
								{ID: "REQ-002", Status: "in_progress", Criticality: "P1"},
								{ID: "REQ-003", Status: "pending", Criticality: "P2"},
							},
						},
						{
							ID:   "child2",
							Name: "Child Group 2",
							Requirements: []RequirementRecord{
								{ID: "REQ-004", Status: "complete", Criticality: "P0"},
							},
						},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      4,
				Completed:  2,
				InProgress: 1,
				Pending:    1,
				P0:         2,
				P1:         1,
				P2:         1,
			},
		},
		{
			name: "deeply nested groups",
			groups: []RequirementGroup{
				{
					ID:   "level1",
					Name: "Level 1",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "complete", Criticality: "P0"},
					},
					Children: []RequirementGroup{
						{
							ID:   "level2",
							Name: "Level 2",
							Requirements: []RequirementRecord{
								{ID: "REQ-002", Status: "in_progress", Criticality: "P1"},
							},
							Children: []RequirementGroup{
								{
									ID:   "level3",
									Name: "Level 3",
									Requirements: []RequirementRecord{
										{ID: "REQ-003", Status: "pending", Criticality: "P2"},
										{ID: "REQ-004", Status: "complete", Criticality: "P0"},
									},
								},
							},
						},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      4,
				Completed:  2,
				InProgress: 1,
				Pending:    1,
				P0:         2,
				P1:         1,
				P2:         1,
			},
		},
		{
			name: "circular reference prevention",
			groups: func() []RequirementGroup {
				// Create a circular reference
				child := RequirementGroup{
					ID:   "child",
					Name: "Child",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "complete", Criticality: "P0"},
					},
				}
				parent := RequirementGroup{
					ID:   "parent",
					Name: "Parent",
					Requirements: []RequirementRecord{
						{ID: "REQ-002", Status: "in_progress", Criticality: "P1"},
					},
					Children: []RequirementGroup{child},
				}
				// In a real circular reference, child would have parent as child
				// But we simulate the same ID being visited twice
				return []RequirementGroup{parent, child}
			}(),
			expected: CatalogRequirementSummary{
				Total:      2,
				Completed:  1,
				InProgress: 1,
				Pending:    0,
				P0:         1,
				P1:         1,
				P2:         0,
			},
		},
		{
			name: "mixed status capitalization",
			groups: []RequirementGroup{
				{
					ID:   "group1",
					Name: "Mixed Case",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "Complete", Criticality: "P0"},
						{ID: "REQ-002", Status: "DONE", Criticality: "p1"},
						{ID: "REQ-003", Status: "In_Progress", Criticality: "P2"},
						{ID: "REQ-004", Status: "IN-PROGRESS", Criticality: "p0"},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      4,
				Completed:  2,
				InProgress: 2,
				Pending:    0,
				P0:         2,
				P1:         1,
				P2:         1,
			},
		},
		{
			name: "no criticality specified",
			groups: []RequirementGroup{
				{
					ID:   "group1",
					Name: "No Priority",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "complete", Criticality: ""},
						{ID: "REQ-002", Status: "in_progress", Criticality: ""},
						{ID: "REQ-003", Status: "pending", Criticality: "P3"},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      3,
				Completed:  1,
				InProgress: 1,
				Pending:    1,
				P0:         0,
				P1:         0,
				P2:         0,
			},
		},
		{
			name: "whitespace in status and criticality",
			groups: []RequirementGroup{
				{
					ID:   "group1",
					Name: "Whitespace",
					Requirements: []RequirementRecord{
						{ID: "REQ-001", Status: "  complete  ", Criticality: "  P0  "},
						{ID: "REQ-002", Status: "\tin_progress\t", Criticality: "\tP1\t"},
					},
				},
			},
			expected: CatalogRequirementSummary{
				Total:      2,
				Completed:  1,
				InProgress: 1,
				Pending:    0,
				P0:         1,
				P1:         1,
				P2:         0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summarizeRequirementGroupsForCatalog(tt.groups)

			if result.Total != tt.expected.Total {
				t.Errorf("Total = %d, want %d", result.Total, tt.expected.Total)
			}
			if result.Completed != tt.expected.Completed {
				t.Errorf("Completed = %d, want %d", result.Completed, tt.expected.Completed)
			}
			if result.InProgress != tt.expected.InProgress {
				t.Errorf("InProgress = %d, want %d", result.InProgress, tt.expected.InProgress)
			}
			if result.Pending != tt.expected.Pending {
				t.Errorf("Pending = %d, want %d", result.Pending, tt.expected.Pending)
			}
			if result.P0 != tt.expected.P0 {
				t.Errorf("P0 = %d, want %d", result.P0, tt.expected.P0)
			}
			if result.P1 != tt.expected.P1 {
				t.Errorf("P1 = %d, want %d", result.P1, tt.expected.P1)
			}
			if result.P2 != tt.expected.P2 {
				t.Errorf("P2 = %d, want %d", result.P2, tt.expected.P2)
			}

			// Verify pending is never negative
			if result.Pending < 0 {
				t.Errorf("Pending count is negative: %d", result.Pending)
			}
		})
	}
}
