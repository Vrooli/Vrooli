package main

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:PCT-REQ-TARGETS] Operational target parser extracts checklist items from PRD
func TestExtractOperationalTargets(t *testing.T) {
	// Create a temporary PRD file for testing
	tmpDir := t.TempDir()
	vrooliRoot := filepath.Join(tmpDir, "vrooli")
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", "test-scenario")

	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	prdContent := `# PRD

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Implement user authentication | Handles login flows
- [x] OT-P0-002 | Add data persistence | Stores workspace state

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Email notifications | Notify users on status changes
`

	prdPath := filepath.Join(scenarioDir, "PRD.md")
	if err := os.WriteFile(prdPath, []byte(prdContent), 0644); err != nil {
		t.Fatalf("failed to write PRD: %v", err)
	}

	// Set VROOLI_ROOT for test
	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)
	defer os.Setenv("VROOLI_ROOT", oldRoot)

	targets, err := extractOperationalTargets("scenario", "test-scenario")
	if err != nil {
		t.Fatalf("extractOperationalTargets() error = %v", err)
	}

	if len(targets) < 2 {
		t.Errorf("expected at least 2 targets, got %d", len(targets))
	}

	// Check that we have P0 and P1 targets
	hasP0 := false
	hasP1 := false
	for _, target := range targets {
		if target.Criticality == "P0" {
			hasP0 = true
		}
		if target.Criticality == "P1" {
			hasP1 = true
		}
	}

	if !hasP0 {
		t.Error("expected at least one P0 target")
	}
	if !hasP1 {
		t.Error("expected at least one P1 target")
	}
}

func TestParseCategoryLabel(t *testing.T) {
	tests := []struct {
		input           string
		wantLabel       string
		wantCriticality string
	}{
		{"Must Have (P0)", "Must Have", "P0"},
		{"Should Have (P1)", "Should Have", "P1"},
		{"Nice to Have (P2)", "Nice to Have", "P2"},
		{"  Must Have (P0)  ", "Must Have", "P0"},
		{"invalid", "invalid", ""},
		{"Category ()", "Category", ""},
		{"No parenthesis", "No parenthesis", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotLabel, gotCriticality := parseCategoryLabel(tt.input)
			if gotLabel != tt.wantLabel {
				t.Errorf("parseCategoryLabel(%q) label = %q, want %q", tt.input, gotLabel, tt.wantLabel)
			}
			if gotCriticality != tt.wantCriticality {
				t.Errorf("parseCategoryLabel(%q) criticality = %q, want %q", tt.input, gotCriticality, tt.wantCriticality)
			}
		})
	}
}

func TestSplitLegacyTargetLine(t *testing.T) {
	tests := []struct {
		input     string
		wantTitle string
		wantNotes string
	}{
		{"- [ ] Implement feature", "Implement feature", ""},
		{"- [ ] Task with notes _(P0)_", "Task with notes", "P0"},
		{"- [x] Completed task _(high priority)_", "Completed task", "high priority"},
		{"- [ ] Simple task", "Simple task", ""},
		{"- [x] Done without notes", "Done without notes", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotTitle, gotNotes := splitLegacyTargetLine(tt.input)
			if gotTitle != tt.wantTitle {
				t.Errorf("splitLegacyTargetLine(%q) title = %q, want %q", tt.input, gotTitle, tt.wantTitle)
			}
			if gotNotes != tt.wantNotes {
				t.Errorf("splitLegacyTargetLine(%q) notes = %q, want %q", tt.input, gotNotes, tt.wantNotes)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple Text", "simple-text"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!@#$%Characters", "specialcharacters"},
		{"123 Numbers", "123-numbers"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple Text", "simpletext"},
		{"  Extra   Spaces  ", "extraspaces"},
		{"Mixed-Case_Text", "mixedcasetext"},
		{"With/Slash", "withslash"},
		{"Dots.Here", "dotshere"},
		{"Parens(test)", "parenstest"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeText(tt.input)
			if got != tt.want {
				t.Errorf("normalizeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLastSegment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Functional Requirements > Authentication > Login", "Login"},
		{"Simple", "Simple"},
		{"A > B > C > D", "D"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := lastSegment(tt.input)
			if got != tt.want {
				t.Errorf("lastSegment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSecondSegment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Functional Requirements > Core Features > Authentication", "Core Features"},
		{"A > B", "B"},
		{"single", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := secondSegment(tt.input)
			if got != tt.want {
				t.Errorf("secondSegment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFlattenRequirements(t *testing.T) {
	groups := []RequirementGroup{
		{
			ID:   "parent",
			Name: "Parent",
			Requirements: []RequirementRecord{
				{ID: "REQ-001", Title: "Parent requirement"},
			},
			Children: []RequirementGroup{
				{
					ID:   "child",
					Name: "Child",
					Requirements: []RequirementRecord{
						{ID: "REQ-002", Title: "Child requirement"},
					},
				},
			},
		},
	}

	flattened := flattenRequirements(groups)

	if len(flattened) != 2 {
		t.Errorf("expected 2 flattened requirements, got %d", len(flattened))
	}

	ids := make(map[string]bool)
	for _, req := range flattened {
		ids[req.ID] = true
	}

	if !ids["REQ-001"] {
		t.Error("expected REQ-001 in flattened requirements")
	}
	if !ids["REQ-002"] {
		t.Error("expected REQ-002 in flattened requirements")
	}
}

// [REQ:PCT-REQ-LINKAGE] Draft workspace surfaces target coverage and unmatched requirements
func TestLinkTargetsAndRequirements(t *testing.T) {
	targets := []OperationalTarget{
		{
			ID:       "target-auth-login",
			Category: "Authentication",
			Title:    "Login",
		},
		{
			ID:       "target-auth-signup",
			Category: "Authentication",
			Title:    "Sign up",
		},
		{
			ID:       "target-data-save",
			Category: "Data Persistence",
			Title:    "Save data",
		},
	}

	requirements := []RequirementGroup{
		{
			ID:   "auth-group",
			Name: "Authentication",
			Requirements: []RequirementRecord{
				{
					ID:     "REQ-001",
					Title:  "User can login",
					PRDRef: "Functional Requirements > Authentication > Login",
				},
				{
					ID:     "REQ-002",
					Title:  "User can sign up",
					PRDRef: "Functional Requirements > Authentication > Sign up",
				},
				{
					ID:     "REQ-003",
					Title:  "Modern target link",
					PRDRef: "Operational Targets > P0 > target-auth-login",
				},
			},
		},
		{
			ID:   "orphan-group",
			Name: "Orphans",
			Requirements: []RequirementRecord{
				{
					ID:     "REQ-ORPHAN",
					Title:  "Orphan requirement",
					PRDRef: "Functional Requirements > NonExistent > Feature",
				},
			},
		},
	}

	linkedTargets, unmatched := linkTargetsAndRequirements(targets, requirements)

	// Check that targets have linked requirements
	var authLoginTarget *OperationalTarget
	for i := range linkedTargets {
		if linkedTargets[i].ID == "target-auth-login" {
			authLoginTarget = &linkedTargets[i]
			break
		}
	}

	if authLoginTarget == nil {
		t.Fatal("expected to find target-auth-login")
	}

	if len(authLoginTarget.LinkedRequirements) == 0 {
		t.Error("expected target-auth-login to have linked requirements")
	}

	// Check unmatched requirements
	foundOrphan := false
	for _, req := range unmatched {
		if req.ID == "REQ-ORPHAN" {
			foundOrphan = true
			break
		}
	}

	if !foundOrphan {
		t.Error("expected REQ-ORPHAN to be in unmatched requirements")
	}
}

// [REQ:PCT-REQ-LINKAGE] Draft workspace surfaces target coverage and unmatched requirements
func TestFindMatchingTargets(t *testing.T) {
	targets := []OperationalTarget{
		{
			ID:       "target-1",
			Category: "Authentication",
			Title:    "User login",
		},
		{
			ID:       "target-2",
			Category: "Authentication",
			Title:    "User registration",
		},
		{
			ID:       "target-3",
			Category: "Data",
			Title:    "Save user data",
		},
	}

	tests := []struct {
		name        string
		requirement RequirementRecord
		wantCount   int
		wantIDs     []string
	}{
		{
			name: "exact match on login",
			requirement: RequirementRecord{
				Title:  "Login functionality",
				PRDRef: "Functional Requirements > Authentication > User login",
			},
			wantCount: 1,
			wantIDs:   []string{"target-1"},
		},
		{
			name: "no match - missing prd_ref",
			requirement: RequirementRecord{
				Title:  "Something",
				PRDRef: "",
			},
			wantCount: 0,
		},
		{
			name: "no match - different category",
			requirement: RequirementRecord{
				Title:  "Login",
				PRDRef: "Functional Requirements > Security > Login",
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := findMatchingTargets(&tt.requirement, targets)

			if len(matches) != tt.wantCount {
				t.Errorf("expected %d matches, got %d", tt.wantCount, len(matches))
			}

			if tt.wantIDs != nil {
				for _, wantID := range tt.wantIDs {
					found := false
					for _, gotID := range matches {
						if gotID == wantID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected to find %s in matches", wantID)
					}
				}
			}
		})
	}
}

func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name      string
		list      []string
		candidate string
		want      []string
	}{
		{
			name:      "add to empty list",
			list:      []string{},
			candidate: "new",
			want:      []string{"new"},
		},
		{
			name:      "add unique item",
			list:      []string{"a", "b"},
			candidate: "c",
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "skip duplicate",
			list:      []string{"a", "b", "c"},
			candidate: "b",
			want:      []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendUnique(tt.list, tt.candidate)

			if len(got) != len(tt.want) {
				t.Errorf("expected length %d, got %d", len(tt.want), len(got))
			}

			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("at index %d: expected %s, got %s", i, v, got[i])
				}
			}
		})
	}
}

func TestParseOperationalTargetsEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
	}{
		{
			name: "modern layout",
			content: `## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Feature A | Notes
- [x] OT-P0-002 | Feature B | Notes

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Feature C | Notes

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Feature D | Notes
`,
			wantCount: 4,
		},
		{
			name: "multiple categories",
			content: `### Functional Requirements
- **Must Have (P0)**
  - [ ] Feature A
  - [x] Feature B

- **Should Have (P1)**
  - [ ] Feature C

- **Nice to Have (P2)**
  - [ ] Feature D
  - [ ] Feature E
`,
			wantCount: 5,
		},
		{
			name: "empty functional requirements",
			content: `### Functional Requirements

### Other Section
- [ ] Not a target
`,
			wantCount: 0,
		},
		{
			name: "no functional requirements section",
			content: `# PRD

## Overview
Some content here

## Technical Details
More content
`,
			wantCount: 0,
		},
		{
			name: "targets with notes",
			content: `### Functional Requirements
- **Must Have (P0)**
  - [ ] Feature with notes _(REQ-001)_
  - [x] Another feature _(links to ABC-123)_
`,
			wantCount: 2,
		},
		{
			name: "mixed valid and invalid lines",
			content: `### Functional Requirements
- **Must Have (P0)**
  - [ ] Valid target
  - Some random text
  - [x] Another valid target
  Regular list item without checkbox
`,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := parseOperationalTargets(tt.content, "scenario", "test")

			if len(targets) != tt.wantCount {
				t.Errorf("expected %d targets, got %d", tt.wantCount, len(targets))
				for i, target := range targets {
					t.Logf("  [%d] %s - %s", i, target.ID, target.Title)
				}
			}
		})
	}
}

func TestParseOperationalTargetsMetadata(t *testing.T) {
	content := `### Functional Requirements
- **Must Have (P0)**
  - [x] Completed feature
  - [ ] Pending feature _(with notes)_
`

	targets := parseOperationalTargets(content, "scenario", "my-scenario")

	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	// Check first target
	if targets[0].Status != "complete" {
		t.Errorf("expected first target status \"complete\", got \"%s\"", targets[0].Status)
	}

	if targets[0].Criticality != "P0" {
		t.Errorf("expected first target criticality \"P0\", got \"%s\"", targets[0].Criticality)
	}

	if targets[0].Category != "Must Have" {
		t.Errorf("expected first target category \"Must Have\", got \"%s\"", targets[0].Category)
	}

	if targets[0].EntityType != "scenario" {
		t.Errorf("expected first target entity_type \"scenario\", got \"%s\"", targets[0].EntityType)
	}

	if targets[0].EntityName != "my-scenario" {
		t.Errorf("expected first target entity_name \"my-scenario\", got \"%s\"", targets[0].EntityName)
	}

	// Check second target
	if targets[1].Status != "pending" {
		t.Errorf("expected second target status \"pending\", got \"%s\"", targets[1].Status)
	}

	if targets[1].Notes != "with notes" {
		t.Errorf("expected second target notes \"with notes\", got \"%s\"", targets[1].Notes)
	}
}

func TestParseModernOperationalTargetsMetadata(t *testing.T) {
	content := "## 🎯 Operational Targets\n" +
		"### 🔴 P0 – Must ship for viability\n" +
		"- [x] OT-P0-001 | Completed feature | shipped\n" +
		"- [ ] OT-P0-002 | Pending feature | with notes `[req:REQ-1,REQ-2]`\n"

	targets := parseOperationalTargets(content, "scenario", "another")

	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	if targets[0].ID != "OT-P0-001" {
		t.Errorf("expected first target ID OT-P0-001, got %s", targets[0].ID)
	}
	if targets[0].Path != "Operational Targets > P0 > OT-P0-001" {
		t.Errorf("unexpected path: %s", targets[0].Path)
	}
	if targets[0].Status != "complete" {
		t.Errorf("expected status complete, got %s", targets[0].Status)
	}
	if targets[1].Notes != "with notes" {
		t.Errorf("expected notes \"with notes\", got %s", targets[1].Notes)
	}
	if len(targets[1].LinkedRequirements) != 2 {
		t.Fatalf("expected explicit requirement links, got %#v", targets[1].LinkedRequirements)
	}
}
