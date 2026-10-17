package autofiler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

type fakeGoals struct {
	createCalls int
	targets     map[string][]string
}

func (g *fakeGoals) Create(_ goals.CreateRequest) (*goals.GoalWithScope, error) {
	g.createCalls++
	if g.createCalls > 1 {
		return nil, errors.New("goal validation error: goal already exists")
	}
	return &goals.GoalWithScope{}, nil
}

func (g *fakeGoals) AddTargets(name string, targets []string) (*goals.GoalWithScope, error) {
	if g.targets == nil {
		g.targets = map[string][]string{}
	}
	existing := map[string]bool{}
	for _, target := range g.targets[name] {
		existing[target] = true
	}
	for _, target := range targets {
		if existing[target] {
			continue
		}
		g.targets[name] = append(g.targets[name], target)
		existing[target] = true
	}
	return &goals.GoalWithScope{}, nil
}

func TestFilerFilesSuggestedItemAndAttachesGoal(t *testing.T) {
	root := t.TempDir()
	store := backlog.NewFileStore(root)
	svc, err := backlog.NewService(backlog.ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	goalSvc := &fakeGoals{}
	filer := NewFiler(svc, goalSvc)

	result, err := filer.File(context.Background(), Finding{
		ID:                  "gct:alpha:tests",
		Scenario:            "alpha",
		Dimension:           "tests",
		Severity:            SeverityRed,
		Description:         "Test coverage is below threshold.",
		RecommendedSkillIDs: []string{"scientific-debugging", "unit-testing-architecture-steer"},
	}, FileOptions{
		Mode:     ModeSuggest,
		Strategy: StrategyFeaturePending,
		GoalName: "automated-maintenance",
	})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if !result.Created {
		t.Fatalf("Created = false, want true")
	}

	item, err := store.LoadItem(backlog.KindFix, result.Item.Name)
	if err != nil {
		t.Fatalf("LoadItem: %v", err)
	}
	if item.Status != backlog.StatusSuggested {
		t.Fatalf("status = %q, want suggested", item.Status)
	}
	if item.FindingRef != "gct:alpha:tests" {
		t.Fatalf("finding_ref = %q", item.FindingRef)
	}
	if want := []string{"scientific-debugging", "unit-testing-architecture-steer"}; !reflect.DeepEqual(item.SuggestedSkills, want) {
		t.Fatalf("suggested skills = %#v, want %#v", item.SuggestedSkills, want)
	}
	if item.CreatedBy == nil || item.CreatedBy.Source != "auto-filer/feature_pending/gct:alpha:tests" {
		t.Fatalf("created_by = %+v", item.CreatedBy)
	}
	if got := goalSvc.targets["automated-maintenance"]; len(got) != 1 || got[0] != "fix/"+item.Name {
		t.Fatalf("goal targets = %#v, want filed item ref", got)
	}
}

func TestFilerOmitsEmptyRecommendedSkills(t *testing.T) {
	item := itemForFinding(Finding{ID: "gct:alpha:tests", Scenario: "alpha"}, FileOptions{}, time.Time{})
	if len(item.SuggestedSkills) != 0 {
		t.Fatalf("suggested skills = %#v, want empty", item.SuggestedSkills)
	}
}

func TestFilerIsIdempotentByStableFindingName(t *testing.T) {
	root := t.TempDir()
	store := backlog.NewFileStore(root)
	svc, err := backlog.NewService(backlog.ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	goalSvc := &fakeGoals{}
	filer := NewFiler(svc, goalSvc)
	finding := Finding{ID: "gct:alpha:tests", Scenario: "alpha", Dimension: "tests"}
	opts := FileOptions{Mode: ModeAutoAdd, Strategy: StrategyImportance, GoalName: "automated-maintenance"}

	first, err := filer.File(context.Background(), finding, opts)
	if err != nil {
		t.Fatalf("File first: %v", err)
	}
	second, err := filer.File(context.Background(), finding, opts)
	if err != nil {
		t.Fatalf("File second: %v", err)
	}
	if !first.Created || second.Created {
		t.Fatalf("created flags = first %v second %v, want true false", first.Created, second.Created)
	}
	item, err := store.LoadItem(backlog.KindFix, first.Item.Name)
	if err != nil {
		t.Fatalf("LoadItem: %v", err)
	}
	if item.Status != backlog.StatusBacklog {
		t.Fatalf("status = %q, want backlog in auto_add mode", item.Status)
	}
	if got := goalSvc.targets["automated-maintenance"]; len(got) != 1 || got[0] != "fix/"+item.Name {
		t.Fatalf("goal targets = %#v, want one idempotent filed item ref", got)
	}
}
