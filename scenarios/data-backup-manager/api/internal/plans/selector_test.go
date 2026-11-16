package plans_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"data-backup-manager/internal/plans"
	"data-backup-manager/internal/plans/mocks"

	"github.com/vrooli/api-core/scheduletest"
)

type catalogOf struct{ ids []string }

func (c *catalogOf) RegisteredTargetIDs(context.Context) ([]string, error) {
	return append([]string(nil), c.ids...), nil
}

func selectorService(catalog *catalogOf) (plans.Service, *mocks.FakeRepository) {
	repo := mocks.NewFakeRepository()
	return plans.WithTargetCatalog(plans.NewService(repo, nil), catalog), repo
}

// "Registered" has to mean "backed up": a plan that selects every registered
// target picks up one registered tomorrow with nobody editing the plan.
func TestSelectorPlanFollowsTheCatalogWithoutEdits(t *testing.T) {
	ctx := context.Background()
	catalog := &catalogOf{ids: []string{"t2", "t1"}}
	svc, repo := selectorService(catalog)

	created, err := svc.Create(ctx, plans.CreateInput{
		Name: "declared targets", TargetSelector: plans.SelectRegistered,
		DestinationIDs: []string{"dst"}, Schedule: "24h",
		TargetIDs: []string{"ignored"}, // a selector replaces any explicit list
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if want := []string{"t1", "t2"}; !reflect.DeepEqual(created.TargetIDs, want) {
		t.Fatalf("targets at create = %v, want %v", created.TargetIDs, want)
	}
	stored, _ := repo.GetByID(ctx, created.ID)
	if len(stored.TargetIDs) != 0 || stored.TargetSelector != plans.SelectRegistered {
		t.Fatalf("a selector plan stores the rule, not a list: %+v", stored)
	}

	catalog.ids = append(catalog.ids, "t3") // registered later
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"t1", "t2", "t3"}; !reflect.DeepEqual(got.TargetIDs, want) {
		t.Fatalf("targets after a new registration = %v, want %v", got.TargetIDs, want)
	}
	list, _ := svc.List(ctx, 0)
	sched, _ := svc.SchedulablePlans(ctx)
	if len(list) != 1 || len(list[0].TargetIDs) != 3 || len(sched) != 1 || len(sched[0].TargetIDs) != 3 {
		t.Fatalf("every read resolves the selector: list=%v schedulable=%v", list, sched)
	}
}

func TestSelectorPlanExclusionsKeepNamedTargetsOut(t *testing.T) {
	ctx := context.Background()
	svc, _ := selectorService(&catalogOf{ids: []string{"a", "b", "c"}})
	created, err := svc.Create(ctx, plans.CreateInput{
		Name: "local copy", TargetSelector: plans.SelectRegistered, ExcludedTargetIDs: []string{" b ", "b", ""},
		DestinationIDs: []string{"dst"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "c"}; !reflect.DeepEqual(created.TargetIDs, want) {
		t.Fatalf("targets = %v, want %v", created.TargetIDs, want)
	}
	if want := []string{"b"}; !reflect.DeepEqual(created.ExcludedTargetIDs, want) {
		t.Fatalf("exclusions = %v, want %v", created.ExcludedTargetIDs, want)
	}
}

func TestExplicitPlansStillWorkAndRefuseSelectorOnlyFields(t *testing.T) {
	ctx := context.Background()
	svc, _ := selectorService(&catalogOf{ids: []string{"a", "b"}})
	explicit, err := svc.Create(ctx, plans.CreateInput{Name: "explicit", TargetIDs: []string{"a"}, DestinationIDs: []string{"dst"}})
	if err != nil || len(explicit.TargetIDs) != 1 || explicit.TargetSelector != plans.SelectNone {
		t.Fatalf("an explicit list keeps working: %+v %v", explicit, err)
	}
	for name, in := range map[string]plans.CreateInput{
		"unknown selector":            {Name: "x", TargetSelector: "everything", DestinationIDs: []string{"d"}},
		"exclusions without selector": {Name: "x", TargetIDs: []string{"a"}, ExcludedTargetIDs: []string{"b"}, DestinationIDs: []string{"d"}},
		"critical tier with selector": {Name: "x", TargetSelector: plans.SelectRegistered, ProtectionTier: plans.TierCriticalPrimary, DestinationIDs: []string{"d"}},
		"nothing to back up":          {Name: "x", DestinationIDs: []string{"d"}},
	} {
		var invalid plans.ErrInvalidPlan
		if _, err := svc.Create(ctx, in); !errors.As(err, &invalid) {
			t.Errorf("%s: err = %v, want ErrInvalidPlan", name, err)
		}
	}
}

func TestUpdateCanSwitchAPlanToASelectorAndBack(t *testing.T) {
	ctx := context.Background()
	svc, _ := selectorService(&catalogOf{ids: []string{"a", "b"}})
	p, err := svc.Create(ctx, plans.CreateInput{Name: "p", TargetIDs: []string{"a"}, DestinationIDs: []string{"d"}})
	if err != nil {
		t.Fatal(err)
	}
	p, err = svc.Update(ctx, plans.UpdateInput{ID: p.ID, Name: "p", TargetSelector: plans.SelectRegistered, DestinationIDs: []string{"d"}, Enabled: true})
	if err != nil || len(p.TargetIDs) != 2 {
		t.Fatalf("switch to selector: %+v %v", p, err)
	}
	p, err = svc.Update(ctx, plans.UpdateInput{ID: p.ID, Name: "p", TargetIDs: []string{"b"}, DestinationIDs: []string{"d"}, Enabled: true})
	if err != nil || !reflect.DeepEqual(p.TargetIDs, []string{"b"}) || p.TargetSelector != plans.SelectNone {
		t.Fatalf("switch back to a list: %+v %v", p, err)
	}
}

func TestSQLiteKeepsASelectorAndItsExclusions(t *testing.T) {
	ctx := context.Background()
	db := newPlansDB(t)
	repo := plans.NewSQLiteRepository(db, scheduletest.New(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)))
	saved, err := repo.Create(ctx, plans.Plan{
		Name: "declared targets", DestinationIDs: []string{"d"}, Enabled: true, ProtectionTier: plans.TierFullPrimary,
		TargetSelector: plans.SelectRegistered, ExcludedTargetIDs: []string{"x", "y"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetSelector != plans.SelectRegistered || !reflect.DeepEqual(got.ExcludedTargetIDs, []string{"x", "y"}) || len(got.TargetIDs) != 0 {
		t.Fatalf("round trip lost the rule: %+v", got)
	}
	got.ExcludedTargetIDs = []string{"z"}
	if _, err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	again, _ := repo.GetByID(ctx, saved.ID)
	if !reflect.DeepEqual(again.ExcludedTargetIDs, []string{"z"}) {
		t.Fatalf("update must replace the exclusions, got %v", again.ExcludedTargetIDs)
	}
	if err := plans.EnsureColumns(ctx, db); err != nil {
		t.Fatalf("EnsureColumns on a catalog that already has the column: %v", err)
	}
}
