package handlers

import (
	"context"
	"testing"

	"agent-manager/internal/orchestration"

	"connectrpc.com/connect"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

type maintenanceServiceFake struct {
	request orchestration.PurgeRequest
	result  *orchestration.PurgeResult
	calls   int
}

func (f *maintenanceServiceFake) PurgeData(_ context.Context, request orchestration.PurgeRequest) (*orchestration.PurgeResult, error) {
	f.calls++
	f.request = request
	return f.result, nil
}

func TestPurgeDataHandlerValidatesRequestBeforeCallingService(t *testing.T) {
	fake := &maintenanceServiceFake{}
	h := New(orchestration.HandlerServices{MaintenanceService: fake})
	for _, request := range []*apipb.PurgeDataRequest{
		{Targets: []apipb.PurgeTarget{apipb.PurgeTarget_PURGE_TARGET_RUNS}},
		{Pattern: "remove"},
		{Pattern: "remove", Targets: []apipb.PurgeTarget{apipb.PurgeTarget_PURGE_TARGET_UNSPECIFIED}},
	} {
		_, err := NewAgentManagerConnectHandler(h).PurgeData(context.Background(), connect.NewRequest(request))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("request=%+v err=%v, want invalid argument", request, err)
		}
	}
	if fake.calls != 0 {
		t.Fatalf("service calls=%d, want validation to reject before service", fake.calls)
	}
}

func TestPurgeDataHandlerMapsTargetsAndCounts(t *testing.T) {
	fake := &maintenanceServiceFake{result: &orchestration.PurgeResult{
		Matched: orchestration.PurgeCounts{Profiles: 2, Tasks: 3, Runs: 4},
		Deleted: orchestration.PurgeCounts{Profiles: 1, Tasks: 2, Runs: 3},
		DryRun:  true,
	}}
	h := New(orchestration.HandlerServices{MaintenanceService: fake})
	request := &apipb.PurgeDataRequest{Pattern: "remove", Targets: []apipb.PurgeTarget{apipb.PurgeTarget_PURGE_TARGET_PROFILES, apipb.PurgeTarget_PURGE_TARGET_TASKS, apipb.PurgeTarget_PURGE_TARGET_RUNS}, DryRun: true}
	rpc, err := NewAgentManagerConnectHandler(h).PurgeData(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 || fake.request.Pattern != "remove" || !fake.request.DryRun || len(fake.request.Targets) != 3 {
		t.Fatalf("service request=%+v calls=%d", fake.request, fake.calls)
	}
	if fake.request.Targets[0] != orchestration.PurgeTargetProfiles || fake.request.Targets[1] != orchestration.PurgeTargetTasks || fake.request.Targets[2] != orchestration.PurgeTargetRuns {
		t.Fatalf("mapped targets=%v", fake.request.Targets)
	}
	response := rpc.Msg
	if !response.GetDryRun() || response.GetMatched().GetProfiles() != 2 || response.GetMatched().GetRuns() != 4 || response.GetDeleted().GetTasks() != 2 {
		t.Fatalf("response=%+v", &response)
	}
}
