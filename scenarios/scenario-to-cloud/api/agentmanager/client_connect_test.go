package agentmanager

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type testResolver string

func (r testResolver) ResolveBaseURL(context.Context) (string, error) { return string(r), nil }

func TestClientUsesGeneratedConnectProcedures(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = true
		var response any
		switch r.URL.Path {
		case apiconnect.AgentManagerServiceHealthProcedure:
			response = &apipb.HealthResponse{Readiness: true}
		case apiconnect.AgentManagerServiceGetProfileProcedure:
			response = &apipb.GetProfileResponse{Profile: &domainpb.AgentProfile{Id: "profile-1"}}
		case apiconnect.AgentManagerServiceEnsureProfileProcedure:
			response = &apipb.EnsureProfileResponse{Profile: &domainpb.AgentProfile{Id: "profile-1"}}
		case apiconnect.AgentManagerServiceCreateTaskProcedure:
			response = &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}}
		case apiconnect.AgentManagerServiceCreateRunProcedure:
			response = &apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}
		case apiconnect.AgentManagerServiceGetRunProcedure:
			response = &apipb.GetRunResponse{Run: &domainpb.Run{Id: "run-1"}}
		case apiconnect.AgentManagerServiceStopRunProcedure:
			response = &apipb.StopRunResponse{Run: &domainpb.Run{Id: "run-1"}}
		case apiconnect.AgentManagerServiceGetRunEventsProcedure:
			response = &apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{RunId: "run-1", Sequence: 4}}}
		default:
			http.NotFound(w, r)
			return
		}
		body, err := protojson.Marshal(response.(proto.Message))
		if err != nil {
			t.Errorf("marshal %s: %v", r.URL.Path, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client := NewClientWithResolver(time.Second, testResolver(server.URL))
	ctx := context.Background()
	ready, err := client.Health(ctx)
	if err != nil || !ready {
		t.Fatalf("Health() = %v, %v", ready, err)
	}
	profile, err := client.GetProfile(ctx, "profile-1")
	if err != nil || profile.GetId() != "profile-1" {
		t.Fatalf("GetProfile() = %v, %v", profile, err)
	}
	ensured, err := client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{ProfileKey: "cloud"})
	if err != nil || ensured.GetProfile().GetId() != "profile-1" {
		t.Fatalf("EnsureProfile() = %v, %v", ensured, err)
	}
	task, err := client.CreateTask(ctx, &domainpb.Task{Title: "test"})
	if err != nil || task.GetId() != "task-1" {
		t.Fatalf("CreateTask() = %v, %v", task, err)
	}
	run, err := client.CreateRun(ctx, &apipb.CreateRunRequest{})
	if err != nil || run.GetId() != "run-1" {
		t.Fatalf("CreateRun() = %v, %v", run, err)
	}
	run, err = client.GetRun(ctx, "run-1")
	if err != nil || run.GetId() != "run-1" {
		t.Fatalf("GetRun() = %v, %v", run, err)
	}
	if err := client.StopRun(ctx, "run-1"); err != nil {
		t.Fatalf("StopRun() error = %v", err)
	}
	events, err := client.GetRunEvents(ctx, "run-1", 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("GetRunEvents() = %v, %v", events, err)
	}
	for _, path := range []string{
		apiconnect.AgentManagerServiceHealthProcedure,
		apiconnect.AgentManagerServiceGetProfileProcedure,
		apiconnect.AgentManagerServiceEnsureProfileProcedure,
		apiconnect.AgentManagerServiceCreateTaskProcedure,
		apiconnect.AgentManagerServiceCreateRunProcedure,
		apiconnect.AgentManagerServiceGetRunProcedure,
		apiconnect.AgentManagerServiceStopRunProcedure,
		apiconnect.AgentManagerServiceGetRunEventsProcedure,
	} {
		if !seen[path] {
			t.Errorf("generated procedure not called: %s", path)
		}
	}
}

func TestCreateTaskAndRunMapTypedApprovalPendingOverConnect(t *testing.T) {
	approval := func() error {
		err := connect.NewError(connect.CodeFailedPrecondition, errors.New("approval pending"))
		detail, detailErr := connect.NewErrorDetail(&apipb.AuthorizationApprovalPending{Subject: "service:scenario-to-cloud", Detail: "operator approval requested"})
		if detailErr != nil {
			t.Fatalf("create approval detail: %v", detailErr)
		}
		err.AddDetail(detail)
		return err
	}
	mux := http.NewServeMux()
	mux.Handle(apiconnect.AgentManagerServiceCreateTaskProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateTaskProcedure,
		func(context.Context, *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
			return nil, approval()
		}))
	mux.Handle(apiconnect.AgentManagerServiceCreateRunProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateRunProcedure,
		func(context.Context, *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
			return nil, approval()
		}))
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewClientWithResolver(time.Second, testResolver(server.URL))

	_, taskErr := client.CreateTask(context.Background(), &domainpb.Task{Title: "approval test"})
	assertApprovalPending(t, taskErr)
	_, runErr := client.CreateRun(context.Background(), &apipb.CreateRunRequest{TaskId: "task-1"})
	assertApprovalPending(t, runErr)
}

func assertApprovalPending(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("operation unexpectedly succeeded")
	}
	var pending *ApprovalPendingError
	if !errors.As(err, &pending) {
		t.Fatalf("error %T %v is not ApprovalPendingError", err, err)
	}
	if pending.Subject != "service:scenario-to-cloud" || pending.Detail != "operator approval requested" {
		t.Fatalf("approval detail = %+v", pending)
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("Connect code = %s", connect.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "operation was not started") {
		t.Fatalf("error does not clearly indicate no start: %v", err)
	}
}
