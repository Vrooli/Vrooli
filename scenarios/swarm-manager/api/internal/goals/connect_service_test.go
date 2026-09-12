package goals

import (
	"context"
	"testing"

	"swarm-manager/internal/backlog"

	"connectrpc.com/connect"
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
	_, err = connectSvc.CreateMilestone(context.Background(), connect.NewRequest(&apipb.CreateMilestoneRequest{GoalName: "g", Milestone: &sharedpb.Milestone{Name: "m", Title: "Milestone", AcceptanceCriteria: []string{"Given the milestone, when its items complete, then the behavior works."}}}))
	if err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}
	assigned, err := connectSvc.AssignMilestoneItems(context.Background(), connect.NewRequest(&apipb.UpdateMilestoneItemsRequest{GoalName: "g", MilestoneName: "m", Items: []string{"execute/a"}}))
	if err != nil || assigned.Msg.Scope.Milestones[0].Ready != 1 {
		t.Fatalf("Assign=%+v err=%v", assigned, err)
	}
}
