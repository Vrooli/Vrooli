package agentmanager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
	"knowledge-observatory/internal/services/deepsearch"
	"knowledge-observatory/internal/services/dochealing"
)

func connectRoute[Req any, Resp any](mux *http.ServeMux, procedure string, fn func(context.Context, *connect.Request[Req]) (*connect.Response[Resp], error)) {
	mux.Handle(procedure, connect.NewUnaryHandler(procedure, fn))
}

func TestConnectClientDeepSearchCreateRunAndIdentity(t *testing.T) {
	cfg := DefaultDeepSearchProfileConfig()
	mux := http.NewServeMux()
	connectRoute(mux, apiconnect.AgentManagerServiceReconcileScenarioProfilesProcedure, func(_ context.Context, req *connect.Request[apipb.ReconcileScenarioProfilesRequest]) (*connect.Response[apipb.ReconcileScenarioProfilesResponse], error) {
		if req.Msg.Scenario != "knowledge-observatory" {
			t.Fatalf("scenario = %q", req.Msg.Scenario)
		}
		return connect.NewResponse(&apipb.ReconcileScenarioProfilesResponse{Results: []*apipb.ProfileReconcileResult{{ProfileKey: cfg.ProfileKey, ProfileId: "profile"}}}), nil
	})
	connectRoute(mux, apiconnect.AgentManagerServiceCreateTaskProcedure, func(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		if req.Msg.Task.GetTitle() != "Search" {
			t.Fatalf("task = %+v", req.Msg.Task)
		}
		return connect.NewResponse(&apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task"}}), nil
	})
	connectRoute(mux, apiconnect.AgentManagerServiceCreateRunProcedure, func(_ context.Context, req *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
		if req.Header().Get("Authorization") != "Bearer fixture-owner" {
			t.Error("caller authorization was not forwarded")
		}
		if req.Msg.GetTaskId() != "task" || req.Msg.GetProfileRef().GetProfileKey() != cfg.ProfileKey {
			t.Fatalf("run request = %+v", req.Msg)
		}
		return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run"}}), nil
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewDeepSearchClientWithBaseURL(time.Second, cfg, server.URL)
	if _, err := client.CreateRun(context.Background(), deepsearch.AgentRunRequest{Title: "Search", Prompt: "look"}); err == nil {
		t.Fatal("anonymous create run succeeded")
	}
	id, err := client.CreateRun(auth01FixtureCaller(t, context.Background()), deepsearch.AgentRunRequest{Title: "Search", Prompt: "look"})
	if err != nil || id != "run" {
		t.Fatalf("CreateRun = %q, %v", id, err)
	}
}

func TestConnectClientDocHealingDiffApprovalAndRejection(t *testing.T) {
	mux := http.NewServeMux()
	connectRoute(mux, apiconnect.AgentManagerServiceGetRunDiffProcedure, func(_ context.Context, req *connect.Request[apipb.GetRunDiffRequest]) (*connect.Response[apipb.GetRunDiffResponse], error) {
		return connect.NewResponse(&apipb.GetRunDiffResponse{Diff: &domainpb.RunDiff{RunId: req.Msg.RunId, Files: []*domainpb.FileDiff{{Path: "README.md", ChangeType: "modified", Additions: 1}}}}), nil
	})
	connectRoute(mux, apiconnect.AgentManagerServiceApproveRunProcedure, func(_ context.Context, req *connect.Request[apipb.ApproveRunRequest]) (*connect.Response[apipb.ApproveRunResponse], error) {
		if req.Msg.GetRunId() != "run" || req.Msg.GetActor() != "tester" {
			t.Fatalf("approval request = %+v", req.Msg)
		}
		return connect.NewResponse(&apipb.ApproveRunResponse{Result: &domainpb.ApproveResult{Success: true, FilesApplied: 1}}), nil
	})
	connectRoute(mux, apiconnect.AgentManagerServiceRejectRunProcedure, func(_ context.Context, req *connect.Request[apipb.RejectRunRequest]) (*connect.Response[apipb.RejectRunResponse], error) {
		if req.Msg.GetRunId() != "run" || req.Msg.GetReason() != "nope" {
			t.Fatalf("rejection request = %+v", req.Msg)
		}
		return connect.NewResponse(&apipb.RejectRunResponse{}), nil
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewDocHealingClientWithBaseURL(time.Second, DefaultDocHealingProfileConfig(), server.URL)
	diff, err := client.GetRunDiff(context.Background(), "run")
	if err != nil || diff == nil || len(diff.Files) != 1 {
		t.Fatalf("GetRunDiff = %+v, %v", diff, err)
	}
	if _, err = client.ApproveRun(context.Background(), dochealing.ApprovalRequest{RunID: "run", Actor: "tester"}); err != nil {
		t.Fatal(err)
	}
	if err = client.RejectRun(context.Background(), dochealing.RejectRequest{RunID: "run", Actor: "tester", Reason: "nope"}); err != nil {
		t.Fatal(err)
	}
}

func TestConnectClientNotFoundAndValidation(t *testing.T) {
	mux := http.NewServeMux()
	connectRoute(mux, apiconnect.AgentManagerServiceGetRunProcedure, func(context.Context, *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	c := NewClientWithBaseURL(time.Second, server.URL)
	run, err := c.GetRun(context.Background(), "missing")
	if err != nil || run != nil {
		t.Fatalf("not found = %+v, %v", run, err)
	}
	if _, err = c.GetRunDiff(context.Background(), " "); err == nil {
		t.Fatal("empty run id accepted")
	}
	if _, err = c.ApproveRun(context.Background(), nil); err == nil {
		t.Fatal("nil approval accepted")
	}
}

func TestConnectClientHealthAndRunEvents(t *testing.T) {
	mux := http.NewServeMux()
	connectRoute(mux, apiconnect.AgentManagerServiceHealthProcedure, func(context.Context, *connect.Request[apipb.HealthRequest]) (*connect.Response[apipb.HealthResponse], error) {
		return connect.NewResponse(&apipb.HealthResponse{Status: commonpb.HealthStatus_HEALTH_STATUS_HEALTHY}), nil
	})
	connectRoute(mux, apiconnect.AgentManagerServiceGetRunEventsProcedure, func(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
		if req.Msg.GetRunId() != "run" || req.Msg.GetAfterSequence() != 12 {
			t.Fatalf("event request = %+v", req.Msg)
		}
		return connect.NewResponse(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: 13}}}), nil
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	c := NewClientWithBaseURL(time.Second, server.URL)
	healthy, err := c.Health(context.Background())
	if err != nil || !healthy {
		t.Fatalf("Health = %v, %v", healthy, err)
	}
	events, err := c.GetRunEvents(context.Background(), "run", 12)
	if err != nil || len(events) != 1 || events[0].Sequence != 13 {
		t.Fatalf("GetRunEvents = %+v, %v", events, err)
	}
}

func TestConnectClientMapsTypedApprovalPendingBeforeCreateRunRedaction(t *testing.T) {
	mux := http.NewServeMux()
	taskProcedure := apiconnect.AgentManagerServiceCreateTaskProcedure
	mux.Handle(taskProcedure, connect.NewUnaryHandler(taskProcedure, func(context.Context, *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		detail, err := connect.NewErrorDetail(&apipb.AuthorizationApprovalPending{Subject: "knowledge-observatory", Detail: "operator approval requested; the task was not started"})
		if err != nil {
			t.Fatal(err)
		}
		connectErr := connect.NewError(connect.CodeFailedPrecondition, errors.New("approval pending"))
		connectErr.AddDetail(detail)
		return nil, connectErr
	}))
	procedure := apiconnect.AgentManagerServiceCreateRunProcedure
	mux.HandleFunc(procedure, func(w http.ResponseWriter, _ *http.Request) {
		writeApprovalPendingTestResponse(t, w, "operator approval requested; the run was not started")
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	c := NewClientWithBaseURL(time.Second, server.URL)
	if _, taskErr := c.CreateTask(context.Background(), &domainpb.Task{Title: "task"}); !errors.Is(taskErr, ErrAuthorizationApprovalPending) {
		t.Fatalf("CreateTask error = %v, want typed approval-pending outcome", taskErr)
	}
	_, err := c.CreateRun(auth01FixtureCaller(t, context.Background()), &apipb.CreateRunRequest{TaskId: "task"})
	if !errors.Is(err, ErrAuthorizationApprovalPending) {
		t.Fatalf("CreateRun error = %v, want typed approval-pending outcome", err)
	}
	if strings.Contains(err.Error(), "create run failed") || !strings.Contains(err.Error(), "operation was not started") || !strings.Contains(err.Error(), "Retry after approval") {
		t.Fatalf("approval outcome was redacted or lost: %v", err)
	}
}

func writeApprovalPendingTestResponse(t *testing.T, w http.ResponseWriter, message string) {
	t.Helper()
	payload, err := proto.Marshal(&apipb.AuthorizationApprovalPending{Subject: "knowledge-observatory", Detail: message})
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"code": "failed_precondition", "message": "approval pending",
		"details": []map[string]string{{"type": "type.googleapis.com/agent_manager.v1.AuthorizationApprovalPending", "value": base64.StdEncoding.EncodeToString(payload)}},
	}); err != nil {
		t.Fatal(err)
	}
}
