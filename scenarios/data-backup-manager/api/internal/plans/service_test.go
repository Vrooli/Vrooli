package plans_test

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/plans"
	"data-backup-manager/internal/plans/mocks"
)

type criticalPolicy map[string]bool

func (p criticalPolicy) IsCritical(_ context.Context, targetID string) (bool, error) {
	return p[targetID], nil
}

// TestPlan_ManyToManyBindings proves that:
//  1. A plan can be created with multiple target_ids and destination_ids.
//  2. GetPlan (via repo.GetByID) returns all members.
//  3. The same target_id can appear in a second plan — no uniqueness violation.
func TestPlan_ManyToManyBindings(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := plans.NewService(repo, nil)

	in := plans.CreateInput{
		Name:           "nightly",
		TargetIDs:      []string{"tgt-1", "tgt-2", "tgt-3"},
		DestinationIDs: []string{"dst-a", "dst-b"},
		Schedule:       "24h",
	}

	p1, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	if p1.ID == "" {
		t.Fatal("plan has empty id")
	}
	if !p1.Enabled {
		t.Fatal("plan should default to enabled=true")
	}

	// GetByID should return membership lists.
	got, err := repo.GetByID(ctx, p1.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.TargetIDs) != 3 {
		t.Fatalf("target_ids len = %d, want 3; got %v", len(got.TargetIDs), got.TargetIDs)
	}
	if len(got.DestinationIDs) != 2 {
		t.Fatalf("destination_ids len = %d, want 2; got %v", len(got.DestinationIDs), got.DestinationIDs)
	}

	// Same target_id (tgt-1, tgt-2) can be put into a second plan.
	in2 := plans.CreateInput{
		Name:           "weekly",
		TargetIDs:      []string{"tgt-1", "tgt-2"},
		DestinationIDs: []string{"dst-offsite"},
		Schedule:       "168h",
	}
	p2, err := svc.Create(ctx, in2)
	if err != nil {
		t.Fatalf("create second plan: %v", err)
	}
	got2, err := repo.GetByID(ctx, p2.ID)
	if err != nil {
		t.Fatalf("GetByID second: %v", err)
	}
	if len(got2.TargetIDs) != 2 {
		t.Fatalf("second plan target_ids = %v, want 2", got2.TargetIDs)
	}

	// Both plans coexist.
	all, err := svc.List(ctx, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("list count = %d, want 2", len(all))
	}
}

// TestPlan_Validation pins the typed validation errors.
func TestPlan_Validation(t *testing.T) {
	ctx := context.Background()
	svc := plans.NewService(mocks.NewFakeRepository(), nil)

	cases := []struct {
		name  string
		in    plans.CreateInput
		field string
	}{
		{"missing name", plans.CreateInput{TargetIDs: []string{"t"}, DestinationIDs: []string{"d"}}, "name"},
		{"missing targets", plans.CreateInput{Name: "x", DestinationIDs: []string{"d"}}, "target_ids"},
		{"missing destinations", plans.CreateInput{Name: "x", TargetIDs: []string{"t"}}, "destination_ids"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tc.in)
			var invalid plans.ErrInvalidPlan
			if !errors.As(err, &invalid) {
				t.Fatalf("want ErrInvalidPlan, got %v", err)
			}
			if invalid.Field != tc.field {
				t.Fatalf("field = %q, want %q", invalid.Field, tc.field)
			}
		})
	}
}

// TestPlan_DeleteAndGet proves delete returns removed=true and subsequent Get
// returns ErrPlanNotFound.
func TestPlan_DeleteAndGet(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := plans.NewService(repo, nil)

	p, err := svc.Create(ctx, plans.CreateInput{
		Name:           "tmp",
		TargetIDs:      []string{"t"},
		DestinationIDs: []string{"d"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	removed, err := svc.Delete(ctx, p.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !removed {
		t.Fatal("delete reported not removed")
	}

	var notFound plans.ErrPlanNotFound
	if _, err := svc.Get(ctx, p.ID); !errors.As(err, &notFound) {
		t.Fatalf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestPlan_CriticalTierRejectsUnclassifiedTarget(t *testing.T) {
	svc := plans.NewServiceWithTargetPolicy(
		mocks.NewFakeRepository(), nil,
		criticalPolicy{"critical-target": true},
	)
	_, err := svc.Create(context.Background(), plans.CreateInput{
		Name:           "critical",
		TargetIDs:      []string{"ordinary-target"},
		DestinationIDs: []string{"dst"},
		ProtectionTier: plans.TierCriticalPrimary,
	})
	var invalid plans.ErrInvalidPlan
	if !errors.As(err, &invalid) || invalid.Field != "target_ids" {
		t.Fatalf("want target classification error, got %v", err)
	}
}

func TestPlan_CriticalTierAcceptsOnlyApprovedTargets(t *testing.T) {
	svc := plans.NewServiceWithTargetPolicy(
		mocks.NewFakeRepository(), nil,
		criticalPolicy{"critical-target": true},
	)
	for _, tier := range []plans.ProtectionTier{plans.TierCriticalPrimary, plans.TierCriticalSecondary} {
		p, err := svc.Create(context.Background(), plans.CreateInput{
			Name:           string(tier),
			TargetIDs:      []string{"critical-target"},
			DestinationIDs: []string{"dst"},
			ProtectionTier: tier,
		})
		if err != nil {
			t.Fatalf("create %s: %v", tier, err)
		}
		if p.ProtectionTier != tier {
			t.Fatalf("tier = %q, want %q", p.ProtectionTier, tier)
		}
	}
}
