package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/proposals"
)

// [REQ:SWM-P0-009] A ready backlog proposal must use the same session
// decision endpoint as a goal proposal and apply through proposals.Applier.
func TestCompositeMutationProcessorAppliesBacklogProposal(t *testing.T) {
	root := t.TempDir()
	store := backlog.NewFileStore(root)
	goalService := goals.NewService(goals.NewStore(root), store)
	if _, err := goalService.Create(goals.CreateRequest{Name: "release", Title: "Release", Priority: 5}); err != nil {
		t.Fatal(err)
	}
	if _, err := goalService.CreateMilestone("release", goals.Milestone{Name: "build", Title: "Build", AcceptanceCriteria: []string{"Given the sources, when the build runs, then it produces an artifact."}}); err != nil {
		t.Fatal(err)
	}
	item := backlog.BacklogItem{Name: "compile", Title: "Compile", Kind: backlog.KindExecute, Status: backlog.StatusBacklog, Priority: 3, Milestone: "release/build", Created: "2026-07-24T00:00:00Z", Updated: "2026-07-24T00:00:00Z"}
	if err := os.MkdirAll(store.ItemDir(item.Kind, item.Name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveItem(item); err != nil {
		t.Fatal(err)
	}
	if _, err := goalService.AddTargets("release", []string{"execute/compile"}); err != nil {
		t.Fatal(err)
	}
	if _, err := goalService.AssignMilestoneItems("release", "build", []string{"execute/compile"}); err != nil {
		t.Fatal(err)
	}
	assigner := goals.NewBacklogMilestoneAssigner(goalService)
	lifecycle, err := backlog.NewService(backlog.ServiceConfig{Store: store, Assigner: assigner})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := newCompositeMutationProcessor(goalService, store, assigner, lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	priority := 8
	payload, err := json.Marshal(proposals.Proposal{Form: proposals.FormMutationList, Mutations: []proposals.Mutation{{ID: "raise-priority", Op: proposals.OpChangePriority, Target: "execute/compile", Priority: &priority}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := processor.Apply(context.Background(), agentsessions.ProposalTarget{Type: agentsessions.ContextBacklogItem, Ref: "execute/compile", Name: "Compile"}, string(payload), nil, agentsessions.MutationProposalSource{SessionID: "session-1", RunID: "run-1", DecidedAt: "2026-07-24T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Outcomes) != 1 || !result.Outcomes[0].Applied {
		t.Fatalf("outcomes = %#v", result.Outcomes)
	}
	updated, err := store.LoadItem(backlog.KindExecute, "compile")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Priority != priority {
		t.Fatalf("priority = %d, want %d", updated.Priority, priority)
	}
}

func TestCompositeMutationProcessorAppliesCaptureProposalAsSuggestedWork(t *testing.T) {
	root := t.TempDir()
	store := backlog.NewFileStore(root)
	goalService := goals.NewService(goals.NewStore(root), store)
	processor := newTestCaptureMutationProcessor(t, store, goalService)

	// Deliberately put create_goal first. Intake must reorder the batch so the
	// goal's computed scope can see the proposed item, while the item remains
	// unattached and visibly suggested until the operator decides what to do.
	payload := captureProposalPayload(t)
	result, err := processor.Apply(context.Background(), agentsessions.ProposalTarget{Type: agentsessions.ContextCapture, Ref: "cap-1", Name: "Capture"}, string(payload), nil, agentsessions.MutationProposalSource{SessionID: "session-1", RunID: "run-1", DecidedAt: "2026-08-09T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	assertCaptureProposalApplied(t, result)
	assertCaptureItemSuggested(t, store)
	assertCaptureGoalLineage(t, goalService)
}

func newTestCaptureMutationProcessor(t *testing.T, store *backlog.FileStore, goalService *goals.Service) *compositeMutationProcessor {
	t.Helper()
	assigner := goals.NewBacklogMilestoneAssigner(goalService)
	lifecycle, err := backlog.NewService(backlog.ServiceConfig{Store: store, Assigner: assigner})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := newCompositeMutationProcessor(goalService, store, assigner, lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	return processor
}

func captureProposalPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := json.Marshal(proposals.Proposal{
		Form: proposals.FormMutationList,
		Mutations: []proposals.Mutation{
			{ID: "capture-goal", Op: proposals.OpCreateGoal, Goal: &proposals.GoalSpec{
				Name: "capture-goal", Title: "Capture goal", Priority: 5,
				Targets: []string{"idea/capture-item"}, SpawnedFrom: "cap-1",
				Milestones: []proposals.GoalMilestone{{Name: "first-step", Title: "First step", AcceptanceCriteria: []string{"The proposed idea is delivered."}, Items: []string{"idea/capture-item"}, SpawnedFrom: "cap-1"}},
			}},
			{ID: "capture-item", Op: proposals.OpAddItem, Item: &proposals.ItemSpec{
				Kind: "idea", Name: "capture-item", Title: "Capture item", Priority: 4, SpawnedFrom: "cap-1",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func assertCaptureProposalApplied(t *testing.T, result agentsessions.MutationProposalApplication) {
	t.Helper()
	if len(result.Outcomes) != 2 || !result.Outcomes[0].Applied || !result.Outcomes[1].Applied {
		t.Fatalf("outcomes = %#v", result.Outcomes)
	}
}

func assertCaptureItemSuggested(t *testing.T, store *backlog.FileStore) {
	t.Helper()
	item, err := store.LoadItem(backlog.KindIdea, "capture-item")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != backlog.StatusSuggested || item.Milestone != "" || item.SpawnedFrom != "cap-1" {
		t.Fatalf("capture item = %+v; want suggested, unattached, and capture lineage", item)
	}
}

func assertCaptureGoalLineage(t *testing.T, goalService *goals.Service) {
	t.Helper()
	goal, err := goalService.Get("capture-goal")
	if err != nil {
		t.Fatal(err)
	}
	if goal.Goal.SpawnedFrom != "cap-1" || len(goal.Goal.Milestones) != 1 || goal.Goal.Milestones[0].SpawnedFrom != "cap-1" {
		t.Fatalf("capture goal lineage = %+v", goal.Goal)
	}
}
