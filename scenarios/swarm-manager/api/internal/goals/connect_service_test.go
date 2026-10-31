package goals

import (
	"context"
	"testing"

	"swarm-manager/internal/backlog"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
)

func TestConnectService_GoalAndMilestoneLifecycle(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "ready", nil)})
	connectSvc := NewConnectService(svc, NewHandler(svc))
	created, err := connectSvc.CreateGoal(context.Background(), connect.NewRequest(&apipb.CreateGoalRequest{Name: "g", Title: "Goal", Targets: []string{"execute/a"}}))
	if err != nil || created.Msg.Goal.Name != "g" {
		t.Fatalf("CreateGoal=%+v err=%v", created, err)
	}
	_, err = connectSvc.CreateMilestone(identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"swarm-manager:write"}}), connect.NewRequest(&apipb.CreateMilestoneRequest{GoalName: "g", Milestone: &sharedpb.Milestone{Name: "m", Title: "Milestone", AcceptanceCriteria: []string{"Given the milestone, when its items complete, then the behavior works."}}}))
	if err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}
	assigned, err := connectSvc.AssignMilestoneItems(identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"swarm-manager:write"}}), connect.NewRequest(&apipb.UpdateMilestoneItemsRequest{GoalName: "g", MilestoneName: "m", Items: []string{"execute/a"}}))
	if err != nil || assigned.Msg.Scope.Milestones[0].Ready != 1 {
		t.Fatalf("Assign=%+v err=%v", assigned, err)
	}
}

func TestBookkeepingCannotForgeIndependentMilestoneDelivery(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "g", Title: "Goal"}); err != nil {
		t.Fatal(err)
	}
	stamp := "2026-10-08T00:00:00Z"
	fake := []CriterionVerdict{{Criterion: "works", Verdict: "delivered", Evidence: []string{"forged"}}}
	if _, err := svc.CreateMilestone("g", Milestone{Name: "m", Title: "Milestone", AcceptanceCriteria: []string{"works"}, VerifiedDeliveredAt: &stamp, CriterionVerdicts: fake}); err != nil {
		t.Fatal(err)
	}
	goal, err := svc.Get("g")
	if err != nil {
		t.Fatal(err)
	}
	if goal.Goal.Milestones[0].VerifiedDeliveredAt != nil || len(goal.Goal.Milestones[0].CriterionVerdicts) != 0 {
		t.Fatal("bookkeeping forged an independent delivery verdict")
	}
	if _, err := svc.CloseOut("g"); err == nil {
		t.Fatal("forged milestone evidence allowed strategic goal acceptance")
	}
	if _, err := svc.MarkMilestoneDeliveredWithVerdicts("g", "m", []CriterionVerdict{{Criterion: "works", Verdict: "delivered", Evidence: []string{"independent-review"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateMilestone("g", Milestone{Name: "m", Title: "Updated", AcceptanceCriteria: []string{"works"}, CriterionVerdicts: fake}); err != nil {
		t.Fatal(err)
	}
	goal, err = svc.Get("g")
	if err != nil {
		t.Fatal(err)
	}
	if goal.Goal.Milestones[0].CriterionVerdicts[0].Evidence[0] != "independent-review" {
		t.Fatal("bookkeeping rewrote accepted review provenance")
	}
}
