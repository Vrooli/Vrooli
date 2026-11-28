package goals

import (
	"context"
	"testing"
)

type fakeRepo struct {
	created   Goal
	progress  Goal
	milestone Milestone
}

func (f *fakeRepo) ListMilestones(context.Context, string) ([]Milestone, error) {
	return []Milestone{f.milestone}, nil
}

func (f *fakeRepo) CreateMilestone(_ context.Context, milestone Milestone) (Milestone, error) {
	f.milestone = milestone
	return milestone, nil
}

func (f *fakeRepo) UpdateMilestoneStatus(_ context.Context, id, status string, revision int64) (Milestone, error) {
	f.milestone.ID = id
	f.milestone.Status = status
	f.milestone.Revision = revision + 1
	return f.milestone, nil
}
func (f *fakeRepo) WorkItemExists(context.Context, string) (bool, error) { return true, nil }
func (f *fakeRepo) MilestonesExistForGoal(context.Context, string, []string) (bool, error) {
	return true, nil
}

func (f *fakeRepo) List(context.Context) ([]Goal, error) { return []Goal{f.created}, nil }
func (f *fakeRepo) Create(_ context.Context, goal Goal) (Goal, error) {
	f.created = goal
	return goal, nil
}

func (f *fakeRepo) UpdateProgress(_ context.Context, id string, progress, revision int64) (Goal, error) {
	f.progress = f.created
	f.progress.ID = id
	f.progress.ProgressBasisPoints = progress
	f.progress.Revision = revision + 1
	return f.progress, nil
}
func (f *fakeRepo) SetTargetDate(context.Context, string, string, string, int64) error { return nil }
func (f *fakeRepo) ListTargetDateHistory(context.Context, string) ([]TargetDateHistory, error) {
	return nil, nil
}

func TestServiceValidatesTargetDate(t *testing.T) {
	if err := NewService(&fakeRepo{}).SetTargetDate(context.Background(), "goal-1", "2026-10-01", "Changed priority", 7); err != nil {
		t.Fatal(err)
	}
	if err := NewService(&fakeRepo{}).SetTargetDate(context.Background(), "goal-1", "tomorrow", "Changed priority", 7); err == nil {
		t.Fatal("expected target date validation")
	}
	if err := NewService(&fakeRepo{}).SetTargetDate(context.Background(), "goal-1", "2026-10-01", "Changed priority", 0); err == nil {
		t.Fatal("expected revision validation")
	}
	if err := NewService(&fakeRepo{}).SetTargetDate(context.Background(), "goal-1", "2026-10-01", "  ", 7); err == nil {
		t.Fatal("expected reason validation")
	}
}

func TestServiceCreatePreservesOutcomeSemantics(t *testing.T) {
	repo := &fakeRepo{}
	goal, err := NewService(repo).Create(context.Background(), CreateInput{Title: "  Ship a calmer morning  ", Purpose: "Protect the first hour", TargetBasisPoints: 10000})
	if err != nil || goal.Title != "Ship a calmer morning" || goal.ProgressBasisPoints != 0 || goal.Status != StatusActive {
		t.Fatalf("goal=%#v err=%v", goal, err)
	}
}

func TestServiceValidatesAndPreservesCurrencyMeasureMeaning(t *testing.T) {
	repo := &fakeRepo{}
	input := CreateInput{Title: "First $1,000", TargetBasisPoints: 10000, TargetDate: "2026-11-30", Measure: &Measure{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "usd", Basis: "after_costs_before_tax", Recurrence: "one_time", Period: "once"}}
	goal, err := NewService(repo).Create(context.Background(), input)
	if err != nil || goal.Measure == nil || goal.Measure.CurrencyCode != "USD" || goal.TargetDate != "2026-11-30" || goal.ProgressBasisPoints != 0 {
		t.Fatalf("goal=%#v err=%v", goal, err)
	}
	input.Measure = &Measure{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "recurring", Period: "month"}
	input.TargetDate = "2026-12-31"
	goal, err = NewService(repo).Create(context.Background(), input)
	if err != nil || goal.Measure == nil || goal.Measure.Recurrence != "recurring" || goal.Measure.Period != "month" {
		t.Fatalf("recurring goal=%#v err=%v", goal, err)
	}
}

func TestServiceRejectsIncompleteOrContradictoryCurrencyMeasures(t *testing.T) {
	valid := Measure{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "recurring", Period: "month"}
	cases := []struct {
		name    string
		measure Measure
		date    string
	}{
		{"missing amount", Measure{Unit: "currency", CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "recurring", Period: "month"}, "2026-12-31"},
		{"unsupported currency", Measure{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "JPY", Basis: "after_costs_before_tax", Recurrence: "recurring", Period: "month"}, "2026-12-31"},
		{"missing basis", Measure{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Recurrence: "recurring", Period: "month"}, "2026-12-31"},
		{"contradictory recurrence", Measure{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "one_time", Period: "month"}, "2026-12-31"},
		{"missing target date", valid, ""},
		{"invalid target date", valid, "tomorrow"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewService(&fakeRepo{}).Create(context.Background(), CreateInput{Title: "Measured", TargetBasisPoints: 10000, TargetDate: tc.date, Measure: &tc.measure})
			if err == nil {
				t.Fatal("expected measure validation error")
			}
		})
	}
}

func TestServiceRejectsImplicitAchievement(t *testing.T) {
	if _, err := NewService(&fakeRepo{}).Create(context.Background(), CreateInput{Title: "Ship", TargetBasisPoints: 0}); err == nil {
		t.Fatal("expected target validation")
	}
	if _, err := NewService(&fakeRepo{}).UpdateProgress(context.Background(), "goal-1", 10001, 1); err == nil {
		t.Fatal("expected progress validation")
	}
}

func TestServiceCreatesMilestoneWithHonestCompletionFields(t *testing.T) {
	m, err := NewService(&fakeRepo{}).CreateMilestone(context.Background(), CreateMilestoneInput{GoalID: "goal-1", Title: "Protect the first hour", Criteria: "No meetings before 10", DueDate: "2026-10-01", LinkedWorkItemID: "work-1", PrerequisiteMilestoneIDs: []string{"m-1", "m-1"}})
	if err != nil || m.Status != MilestoneOpen || m.GoalID != "goal-1" || m.DueDate != "2026-10-01" || m.LinkedWorkItemID != "work-1" {
		t.Fatalf("milestone=%#v err=%v", m, err)
	}
	if len(m.PrerequisiteMilestoneIDs) != 1 || m.PrerequisiteMilestoneIDs[0] != "m-1" {
		t.Fatalf("prerequisites=%v", m.PrerequisiteMilestoneIDs)
	}
	if _, err := NewService(&fakeRepo{}).CreateMilestone(context.Background(), CreateMilestoneInput{GoalID: "goal-1", Title: "Ship", DueDate: "tomorrow"}); err == nil {
		t.Fatal("expected date validation")
	}
}
