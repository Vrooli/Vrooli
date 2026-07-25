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
