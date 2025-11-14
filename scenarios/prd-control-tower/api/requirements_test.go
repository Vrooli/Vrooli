package main

import "testing"

func TestParseOperationalTargets(t *testing.T) {
	markdown := `## 📊 Success Metrics
### Functional Requirements
- **Must Have (P0)**
  - [x] Ship capability _(done)_
- **Should Have (P1)**
  - [ ] Add bonus feature _(later)_
### Performance Criteria
- Placeholder
`

	targets := parseOperationalTargets(markdown, "scenario", "demo")
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	if targets[0].Status != "complete" || targets[0].Category != "Must Have" || targets[0].Criticality != "P0" {
		t.Fatalf("unexpected first target %+v", targets[0])
	}

	if targets[1].Status != "pending" || targets[1].Category != "Should Have" || targets[1].Criticality != "P1" {
		t.Fatalf("unexpected second target %+v", targets[1])
	}
}

func TestLinkTargetsAndRequirements(t *testing.T) {
	targets := []OperationalTarget{
		{ID: "t1", Title: "Visual workflow builder", Category: "Must Have"},
		{ID: "t2", Title: "Calendar scheduling", Category: "Should Have"},
	}

	groups := []RequirementGroup{
		{
			Requirements: []RequirementRecord{
				{ID: "req-1", PRDRef: "Functional Requirements > Must Have > Visual workflow builder", Title: "Visual workflow builder", Category: "projects"},
				{ID: "req-2", PRDRef: "Functional Requirements > Nice to Have > Visual diff", Title: "Visual diff", Category: "analysis"},
			},
		},
	}

	linked, unmatched := linkTargetsAndRequirements(targets, groups)
	if len(unmatched) != 1 || unmatched[0].ID != "req-2" {
		t.Fatalf("unexpected unmatched requirements: %+v", unmatched)
	}

	if len(linked[0].LinkedRequirements) != 1 || linked[0].LinkedRequirements[0] != "req-1" {
		t.Fatalf("expected req-1 linked to first target, got %+v", linked[0].LinkedRequirements)
	}
}
