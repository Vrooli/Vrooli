package main

import (
	"context"
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

func (agentManagerConnectFake) EnsureProfile(_ context.Context, req *connect.Request[apipb.EnsureProfileRequest]) (*connect.Response[apipb.EnsureProfileResponse], error) {
	return connect.NewResponse(&apipb.EnsureProfileResponse{Profile: &domainpb.AgentProfile{Id: req.Msg.GetProfileKey()}, Created: true}), nil
}

func (agentManagerConnectFake) ReconcileScenarioProfiles(_ context.Context, req *connect.Request[apipb.ReconcileScenarioProfilesRequest]) (*connect.Response[apipb.ReconcileScenarioProfilesResponse], error) {
	return connect.NewResponse(&apipb.ReconcileScenarioProfilesResponse{Scenario: req.Msg.GetScenario()}), nil
}

func (agentManagerConnectFake) GetProfile(_ context.Context, req *connect.Request[apipb.GetProfileRequest]) (*connect.Response[apipb.GetProfileResponse], error) {
	return connect.NewResponse(&apipb.GetProfileResponse{Profile: &domainpb.AgentProfile{Id: req.Msg.GetProfileId()}}), nil
}

func (agentManagerConnectFake) CreateTask(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
	return connect.NewResponse(&apipb.CreateTaskResponse{Task: req.Msg.GetTask()}), nil
}

func (agentManagerConnectFake) GetTask(_ context.Context, req *connect.Request[apipb.GetTaskRequest]) (*connect.Response[apipb.GetTaskResponse], error) {
	if req.Msg.GetTaskId() == "missing" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&apipb.GetTaskResponse{Task: &domainpb.Task{Id: req.Msg.GetTaskId()}}), nil
}

func (agentManagerConnectFake) CreateRun(_ context.Context, _ *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
}

func (agentManagerConnectFake) GetRun(_ context.Context, req *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	return connect.NewResponse(&apipb.GetRunResponse{Run: &domainpb.Run{Id: req.Msg.GetRunId()}}), nil
}

func (agentManagerConnectFake) GetRunByTag(_ context.Context, req *connect.Request[apipb.GetRunByTagRequest]) (*connect.Response[apipb.GetRunByTagResponse], error) {
	if req.Msg.GetTag() == "missing" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&apipb.GetRunByTagResponse{Run: &domainpb.Run{Tag: req.Msg.GetTag()}}), nil
}

func (agentManagerConnectFake) StopRunByTag(_ context.Context, req *connect.Request[apipb.StopRunByTagRequest]) (*connect.Response[apipb.StopRunByTagResponse], error) {
	if req.Msg.GetTag() == "missing" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&apipb.StopRunByTagResponse{}), nil
}

func TestAgentManagerGeneratedConnectCalls(t *testing.T) {
	path, handler := apiconnect.NewAgentManagerServiceHandler(agentManagerConnectFake{})
	router := http.NewServeMux()
	router.Handle(path, handler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := newAgentManagerClient(time.Second)
	client.baseURLResolver = func(context.Context) (string, error) { return server.URL, nil }
	ctx := context.Background()

	profileResult, err := client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{ProfileKey: "scenario-auditor"})
	if err != nil || profileResult.GetProfile().GetId() != "scenario-auditor" || !profileResult.GetCreated() {
		t.Fatalf("EnsureProfile() = %+v, %v", profileResult, err)
	}

	profiles, err := client.ReconcileScenarioProfiles(ctx, "scenario-auditor")
	if err != nil || profiles.GetScenario() != "scenario-auditor" {
		t.Fatalf("ReconcileScenarioProfiles() = %+v, %v", profiles, err)
	}

	profile, err := client.GetProfile(ctx, "profile-1")
	if err != nil || profile.GetId() != "profile-1" {
		t.Fatalf("GetProfile() = %+v, %v", profile, err)
	}

	task, err := client.CreateTask(ctx, &domainpb.Task{Title: "connect task"})
	if err != nil || task.GetTitle() != "connect task" {
		t.Fatalf("CreateTask() = %+v, %v", task, err)
	}

	task, err = client.GetTask(ctx, "task-1")
	if err != nil || task.GetId() != "task-1" {
		t.Fatalf("GetTask() = %+v, %v", task, err)
	}
	if task, err = client.GetTask(ctx, "missing"); err != nil || task != nil {
		t.Fatalf("GetTask(missing) = %+v, %v; want nil, nil", task, err)
	}

	run, err := client.CreateRun(ctx, &apipb.CreateRunRequest{})
	if err != nil || run.GetId() != "run-1" {
		t.Fatalf("CreateRun() = %+v, %v", run, err)
	}

	run, err = client.GetRun(ctx, "run-2")
	if err != nil || run.GetId() != "run-2" {
		t.Fatalf("GetRun() = %+v, %v", run, err)
	}

	run, err = client.GetRunByTag(ctx, "auditor-agent")
	if err != nil || run.GetTag() != "auditor-agent" {
		t.Fatalf("GetRunByTag() = %+v, %v", run, err)
	}
	if run, err = client.GetRunByTag(ctx, "missing"); err != nil || run != nil {
		t.Fatalf("GetRunByTag(missing) = %+v, %v; want nil, nil", run, err)
	}
	if err := client.StopRunByTag(ctx, "auditor-agent"); err != nil {
		t.Fatalf("StopRunByTag() = %v", err)
	}
	if err := client.StopRunByTag(ctx, "missing"); err != nil {
		t.Fatalf("StopRunByTag(missing) = %v; want nil", err)
	}
}
