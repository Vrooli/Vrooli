package agentmanager

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type agentManagerConnectFake struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
}

func (agentManagerConnectFake) ReconcileScenarioProfiles(_ context.Context, req *connect.Request[apipb.ReconcileScenarioProfilesRequest]) (*connect.Response[apipb.ReconcileScenarioProfilesResponse], error) {
	return connect.NewResponse(&apipb.ReconcileScenarioProfilesResponse{Scenario: req.Msg.GetScenario()}), nil
}

func (agentManagerConnectFake) GetProfile(_ context.Context, req *connect.Request[apipb.GetProfileRequest]) (*connect.Response[apipb.GetProfileResponse], error) {
	return connect.NewResponse(&apipb.GetProfileResponse{Profile: &domainpb.AgentProfile{Id: req.Msg.GetProfileId()}}), nil
}

func (agentManagerConnectFake) GetRolePolicyCatalog(_ context.Context, _ *connect.Request[apipb.GetRolePolicyCatalogRequest]) (*connect.Response[apipb.GetRolePolicyCatalogResponse], error) {
	return connect.NewResponse(&apipb.GetRolePolicyCatalogResponse{}), nil
}

func (agentManagerConnectFake) CreateTask(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
	return connect.NewResponse(&apipb.CreateTaskResponse{Task: req.Msg.GetTask()}), nil
}

func (agentManagerConnectFake) CreateRun(_ context.Context, _ *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-created"}}), nil
}

func (agentManagerConnectFake) GetRun(_ context.Context, req *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	if req.Msg.GetRunId() == "missing" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("run missing"))
	}
	return connect.NewResponse(&apipb.GetRunResponse{Run: &domainpb.Run{Id: req.Msg.GetRunId()}}), nil
}

func (agentManagerConnectFake) ListRuns(_ context.Context, _ *connect.Request[apipb.ListRunsRequest]) (*connect.Response[apipb.ListRunsResponse], error) {
	return connect.NewResponse(&apipb.ListRunsResponse{Runs: []*domainpb.Run{{Id: "run-listed"}}}), nil
}

func (agentManagerConnectFake) GetRunByTag(_ context.Context, req *connect.Request[apipb.GetRunByTagRequest]) (*connect.Response[apipb.GetRunByTagResponse], error) {
	if req.Msg.GetTag() == "missing" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("run missing"))
	}
	return connect.NewResponse(&apipb.GetRunByTagResponse{Run: &domainpb.Run{Id: "tagged-run"}}), nil
}

func (agentManagerConnectFake) StopRun(_ context.Context, _ *connect.Request[apipb.StopRunRequest]) (*connect.Response[apipb.StopRunResponse], error) {
	return connect.NewResponse(&apipb.StopRunResponse{}), nil
}

func (agentManagerConnectFake) StopRunByTag(_ context.Context, _ *connect.Request[apipb.StopRunByTagRequest]) (*connect.Response[apipb.StopRunByTagResponse], error) {
	return connect.NewResponse(&apipb.StopRunByTagResponse{}), nil
}

func (agentManagerConnectFake) StopAllRuns(_ context.Context, _ *connect.Request[apipb.StopAllRunsRequest]) (*connect.Response[apipb.StopAllRunsResponse], error) {
	return connect.NewResponse(&apipb.StopAllRunsResponse{Result: &domainpb.StopAllResult{StoppedCount: 2, Failures: []*domainpb.StopFailure{{Error: "one failed"}}}}), nil
}

func (agentManagerConnectFake) GetRunEvents(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
	return connect.NewResponse(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: req.Msg.GetAfterSequence() + 1}}}), nil
}

func (agentManagerConnectFake) GetRunnerStatus(_ context.Context, _ *connect.Request[apipb.GetRunnerStatusRequest]) (*connect.Response[apipb.GetRunnerStatusResponse], error) {
	return connect.NewResponse(&apipb.GetRunnerStatusResponse{Runners: []*domainpb.RunnerStatus{{RunnerType: domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE}}}), nil
}

func TestAgentManagerClientUsesGeneratedConnectProcedures(t *testing.T) {
	path, handler := apiconnect.NewAgentManagerServiceHandler(agentManagerConnectFake{})
	router := http.NewServeMux()
	router.Handle(path, handler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := NewClientWithResolver(time.Second, func(context.Context) (string, error) { return server.URL, nil })
	ctx := context.Background()

	profiles, err := client.ReconcileScenarioProfiles(ctx, "test-genie")
	if err != nil || profiles.GetScenario() != "test-genie" {
		t.Fatalf("ReconcileScenarioProfiles() = %+v, %v", profiles, err)
	}
	profile, err := client.GetProfile(ctx, "profile-1")
	if err != nil || profile.GetId() != "profile-1" {
		t.Fatalf("GetProfile() = %+v, %v", profile, err)
	}
	if _, err := client.GetRolePolicyCatalog(ctx); err != nil {
		t.Fatalf("GetRolePolicyCatalog() = %v", err)
	}

	task, err := client.CreateTask(ctx, &domainpb.Task{Title: "Connect task"})
	if err != nil || task.GetTitle() != "Connect task" {
		t.Fatalf("CreateTask() = %+v, %v", task, err)
	}
	run, err := client.CreateRun(ctx, &apipb.CreateRunRequest{})
	if err != nil || run.GetId() != "run-created" {
		t.Fatalf("CreateRun() = %+v, %v", run, err)
	}
	run, err = client.GetRun(ctx, "run-1")
	if err != nil || run.GetId() != "run-1" {
		t.Fatalf("GetRun() = %+v, %v", run, err)
	}
	run, err = client.GetRun(ctx, "missing")
	if err != nil || run != nil {
		t.Fatalf("GetRun(missing) = %+v, %v; want nil, nil", run, err)
	}

	runs, err := client.ListRuns(ctx, ListRunsOptions{Limit: 5})
	if err != nil || len(runs) != 1 || runs[0].GetId() != "run-listed" {
		t.Fatalf("ListRuns() = %+v, %v", runs, err)
	}
	taggedRun, err := client.GetRunByTag(ctx, "custom-tag")
	if err != nil || taggedRun.GetId() != "tagged-run" {
		t.Fatalf("GetRunByTag() = %+v, %v", taggedRun, err)
	}
	taggedRun, err = client.GetRunByTag(ctx, "missing")
	if err != nil || taggedRun != nil {
		t.Fatalf("GetRunByTag(missing) = %+v, %v; want nil, nil", taggedRun, err)
	}
	if err := client.StopRun(ctx, "run-1"); err != nil {
		t.Fatalf("StopRun() = %v", err)
	}
	if err := client.StopRunByTag(ctx, "tag"); err != nil {
		t.Fatalf("StopRunByTag() = %v", err)
	}
	stopped, err := client.StopAllRuns(ctx, "test-")
	if err != nil || stopped.StoppedCount != 2 || stopped.FailedCount != 1 || len(stopped.Errors) != 1 {
		t.Fatalf("StopAllRuns() = %+v, %v", stopped, err)
	}
	status, err := client.GetRunnerStatus(ctx)
	if err != nil || len(status) != 1 || status[0].GetRunnerType() != domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE {
		t.Fatalf("GetRunnerStatus() = %+v, %v", status, err)
	}
	events, err := client.GetRunEvents(ctx, "run-1", 4)
	if err != nil || len(events) != 1 || events[0].GetSequence() != 5 {
		t.Fatalf("GetRunEvents() = %+v, %v", events, err)
	}
}
