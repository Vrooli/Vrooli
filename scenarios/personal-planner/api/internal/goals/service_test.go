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
func (f *fakeRepo) SetTargetDate(context.Context, string, string) error { return nil }

func TestServiceValidatesTargetDate(t *testing.T) {
	if err := NewService(&fakeRepo{}).SetTargetDate(context.Background(), "goal-1", "2026-10-01"); err != nil {
		t.Fatal(err)
	}
	if err := NewService(&fakeRepo{}).SetTargetDate(context.Background(), "goal-1", "tomorrow"); err == nil {
		t.Fatal("expected target date validation")
	}
}

func TestServiceCreatePreservesOutcomeSemantics(t *testing.T) {
	repo := &fakeRepo{}
	goal, err := NewService(repo).Create(context.Background(), CreateInput{Title: "  Ship a calmer morning  ", Purpose: "Protect the first hour", TargetBasisPoints: 10000})
	if err != nil || goal.Title != "Ship a calmer morning" || goal.ProgressBasisPoints != 0 || goal.Status != StatusActive {
		t.Fatalf("goal=%#v err=%v", goal, err)
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
