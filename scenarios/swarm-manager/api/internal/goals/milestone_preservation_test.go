package goals

import (
	"testing"

	"swarm-manager/internal/backlog"
)

// updateMilestone replaces the milestone definition wholesale. Every field the
// caller cannot express must survive, or writing acceptance criteria silently
// destroys membership — which is exactly what the CLI path did before
// carryServerOwnedMilestoneFields existed.
func TestUpdateMilestonePreservesMembershipAndArchival(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "a", "ready", nil),
		item("execute", "b", "ready", nil),
	})
	if _, err := svc.Create(CreateRequest{Name: "goal", Targets: []string{"execute/a", "execute/b"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.CreateMilestone("goal", Milestone{Name: "ms", Title: "Ms", AcceptanceCriteria: []string{"Given x, when y, then z."}}); err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}
	if _, err := svc.AssignMilestoneItems("goal", "ms", []string{"execute/a", "execute/b"}); err != nil {
		t.Fatalf("AssignMilestoneItems: %v", err)
	}

	// A criteria-only update, carrying no items — the shape the CLI sends.
	if _, err := svc.UpdateMilestone("goal", Milestone{
		Name:               "ms",
		Title:              "Ms",
		AcceptanceCriteria: []string{"Given x, when y, then z."},
	}); err != nil {
		t.Fatalf("UpdateMilestone: %v", err)
	}

	got, err := svc.Get("goal")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if n := len(got.Goal.Milestones[0].Items); n != 2 {
		t.Fatalf("update erased membership: got %d members, want 2", n)
	}
}

// A verdict judges one definition of done. Preserve it when the criteria are
// unchanged; clear it when they are not, so a milestone cannot stay "verified
// delivered" against criteria nobody reviewed.
func TestUpdateMilestoneClearsVerifiedStampOnlyWhenCriteriaChange(t *testing.T) {
	for _, tc := range []struct {
		name        string
		criteria    []string
		wantVerentf bool
	}{
		{name: "same criteria keeps the stamp", criteria: []string{"Given x, when y, then z."}, wantVerentf: true},
		{name: "changed criteria clears the stamp", criteria: []string{"Given a, when b, then c."}, wantVerentf: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "completed", nil)})
			if _, err := svc.Create(CreateRequest{Name: "goal", Targets: []string{"execute/a"}}); err != nil {
				t.Fatalf("Create: %v", err)
			}
			if _, err := svc.CreateMilestone("goal", Milestone{Name: "ms", Title: "Ms", AcceptanceCriteria: []string{"Given x, when y, then z."}}); err != nil {
				t.Fatalf("CreateMilestone: %v", err)
			}
			if _, err := svc.AssignMilestoneItems("goal", "ms", []string{"execute/a"}); err != nil {
				t.Fatalf("AssignMilestoneItems: %v", err)
			}
			if _, err := svc.MarkMilestoneDelivered("goal", "ms"); err != nil {
				t.Fatalf("MarkMilestoneDelivered: %v", err)
			}

			if _, err := svc.UpdateMilestone("goal", Milestone{Name: "ms", Title: "Ms", AcceptanceCriteria: tc.criteria}); err != nil {
				t.Fatalf("UpdateMilestone: %v", err)
			}

			got, err := svc.Get("goal")
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			stamped := got.Goal.Milestones[0].VerifiedDeliveredAt != nil
			if stamped != tc.wantVerentf {
				t.Fatalf("verified-delivered stamp present = %v, want %v", stamped, tc.wantVerentf)
			}
		})
	}
}

// A milestone that claims an item outside the goal's closure counts it in no
// rollup. Reporting it is the difference between a visible discrepancy and work
// that quietly stops being tracked.
func TestMilestoneRollupReportsOrphanedMembers(t *testing.T) {
	scope := ComputeScope(ScopeInput{
		Targets:    []string{"execute/in-scope"},
		ItemDeps:   map[string][]string{},
		ItemStatus: map[string]string{"execute/in-scope": "ready", "execute/elsewhere": "ready"},
		Milestones: []Milestone{{
			Name:  "ms",
			Items: []string{"execute/in-scope", "execute/elsewhere"},
		}},
	})

	if len(scope.Milestones) != 1 {
		t.Fatalf("expected one milestone rollup, got %d", len(scope.Milestones))
	}
	rollup := scope.Milestones[0]
	if len(rollup.Items) != 1 || rollup.Items[0] != "execute/in-scope" {
		t.Fatalf("rollup items = %v, want only the in-closure member", rollup.Items)
	}
	if len(rollup.Orphaned) != 1 || rollup.Orphaned[0] != "execute/elsewhere" {
		t.Fatalf("rollup orphaned = %v, want [execute/elsewhere]", rollup.Orphaned)
	}
}
