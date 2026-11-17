package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/policygate"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/repo-contract-go/repocontracttest"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type gctAgentManagerStub struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	ownerHeader string
	task        *ampb.Task
	runRequest  *apipb.CreateRunRequest
}

func (s *gctAgentManagerStub) ListProfiles(_ context.Context, request *connect.Request[apipb.ListProfilesRequest]) (*connect.Response[apipb.ListProfilesResponse], error) {
	s.ownerHeader = request.Header().Get("Authorization")
	return connect.NewResponse(&apipb.ListProfilesResponse{Profiles: []*ampb.AgentProfile{{Id: "profile-1", ProfileKey: "reviewer", Name: "Reviewer"}}, Total: 1}), nil
}

func (s *gctAgentManagerStub) CreateTask(_ context.Context, request *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
	s.task = request.Msg.GetTask()
	return connect.NewResponse(&apipb.CreateTaskResponse{Task: &ampb.Task{Id: "task-1"}}), nil
}

func (s *gctAgentManagerStub) CreateRun(_ context.Context, request *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	s.runRequest = request.Msg
	return connect.NewResponse(&apipb.CreateRunResponse{Run: &ampb.Run{Id: "run-1"}}), nil
}

func (s *gctAgentManagerStub) GetRun(_ context.Context, request *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	s.ownerHeader = request.Header().Get("Authorization")
	return connect.NewResponse(&apipb.GetRunResponse{Run: &ampb.Run{Id: request.Msg.GetRunId(), Status: ampb.RunStatus_RUN_STATUS_RUNNING, CreatedAt: timestamppb.New(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))}}), nil
}

func newConnectAgentManagerClient(t *testing.T, handler apiconnect.AgentManagerServiceHandler) (*AgentManagerClient, *httptest.Server) {
	t.Helper()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(handler)
	mux := http.NewServeMux()
	mux.Handle(path, connectHandler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &AgentManagerClient{BaseClient: BaseClient{httpClient: server.Client(), resolver: discovery.NewStaticResolver(server.URL), serviceName: "agent-manager"}}, server
}

func verifiedAgentManagerReadContext() context.Context {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator", Verified: true,
		Source: identity.SourceScenarioAuthenticator, ExpiresAt: time.Now().Add(time.Hour),
		Scopes: []string{"agent-manager:read"},
	})
	return policygate.WithVerifiedOwnerProof(ctx, http.Header{"Authorization": []string{"Bearer verified-owner"}})
}

func TestAgentManagerClientUsesConnectAndForwardsVerifiedOwnerProof(t *testing.T) {
	stub := &gctAgentManagerStub{}
	client, _ := newConnectAgentManagerClient(t, stub)
	result, err := client.ListProfiles(verifiedAgentManagerReadContext())
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if got := result.GetProfiles()[0].GetProfileKey(); got != "reviewer" {
		t.Fatalf("profile key=%q", got)
	}
	if stub.ownerHeader != "Bearer verified-owner" {
		t.Fatalf("owner proof=%q", stub.ownerHeader)
	}

	run, err := client.GetRun(verifiedAgentManagerReadContext(), "run-2")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.GetRun().GetId() != "run-2" || stub.ownerHeader != "Bearer verified-owner" {
		t.Fatalf("run=%+v owner=%q", run.GetRun(), stub.ownerHeader)
	}
}

func TestAgentManagerClientBuildsProtoTaskAndRunRequests(t *testing.T) {
	stub := &gctAgentManagerStub{}
	client, _ := newConnectAgentManagerClient(t, stub)
	repoDir := t.TempDir()
	repocontracttest.WriteRepoContract(t, repoDir, "scenarios")
	task := buildAgentTaskData(repoDir, AgentRunRequest{ScenarioSlug: "demo", Prompt: "Review", AttachmentIDs: []string{"attachment-1"}})
	response, err := client.CreateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if response.GetTask().GetId() != "task-1" || stub.task.GetScopePath() == "" || stub.task.GetContextAttachments()[0].GetAttachmentId() != "attachment-1" {
		t.Fatalf("task request/response mismatch: request=%+v response=%+v", stub.task, response)
	}

	runRequest := buildAgentRunRequest(response.GetTask().GetId(), AgentRunRequest{ScenarioSlug: "demo", ProfileKey: "gct/reviewer"})
	runResponse, err := client.CreateRun(context.Background(), runRequest)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if runResponse.GetRun().GetId() != "run-1" || stub.runRequest.GetTaskId() != "task-1" || stub.runRequest.GetTag() != "gct-demo" || stub.runRequest.GetRunMode() != ampb.RunMode_RUN_MODE_SANDBOXED || stub.runRequest.GetProfileRef().GetProfileKey() != "gct/reviewer" {
		t.Fatalf("CreateRun request/response mismatch: request=%+v response=%+v", stub.runRequest, runResponse)
	}
}

func TestBuildAgentTaskDataUsesContractBackedScenarioScope(t *testing.T) {
	repoDir := t.TempDir()
	repocontracttest.WriteRepoContract(t, repoDir, "scenarios")
	task := buildAgentTaskData(repoDir, AgentRunRequest{ScenarioSlug: "my-app", Prompt: "Review this scenario"})
	if task.GetScopePath() != "scenarios/my-app/" {
		t.Fatalf("scope=%q", task.GetScopePath())
	}
	if task.GetTitle() != "GCT review: my-app" {
		t.Fatalf("title=%q", task.GetTitle())
	}
}

func TestAgentManagerUploadAttachmentRetainsMultipartAndForwardsOwnerProof(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/attachments/upload" {
			t.Errorf("unexpected upload route: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "multipart/form-data; boundary=fixture" {
			t.Errorf("content type=%q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer verified-owner" {
			t.Errorf("owner proof=%q", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "multipart-body" {
			t.Errorf("body=%q err=%v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireUploadAttachmentResponse{ID: "attachment-1", FileName: "screen.png", ContentType: "image/png", FileSize: 14})
	}))
	defer server.Close()
	client := &AgentManagerClient{BaseClient: BaseClient{httpClient: server.Client(), resolver: discovery.NewStaticResolver(server.URL), serviceName: "agent-manager"}}
	result, err := client.UploadAttachment(verifiedAgentManagerReadContext(), bytes.NewBufferString("multipart-body"), "multipart/form-data; boundary=fixture")
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}
	if result.ID != "attachment-1" || result.FileName != "screen.png" {
		t.Fatalf("response=%+v", result)
	}
}

type unavailableOnceAgentManagerStub struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	calls int
}

func (s *unavailableOnceAgentManagerStub) GetRun(_ context.Context, request *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	s.calls++
	if s.calls == 1 {
		return nil, connect.NewError(connect.CodeUnavailable, io.ErrUnexpectedEOF)
	}
	return connect.NewResponse(&apipb.GetRunResponse{Run: &ampb.Run{Id: request.Msg.GetRunId()}}), nil
}

func TestAgentManagerConnectRetryReResolvesAfterUnavailable(t *testing.T) {
	firstStub := &unavailableOnceAgentManagerStub{}
	firstPath, firstHandler := apiconnect.NewAgentManagerServiceHandler(firstStub)
	firstMux := http.NewServeMux()
	firstMux.Handle(firstPath, firstHandler)
	first := httptest.NewServer(firstMux)
	defer first.Close()

	secondStub := &gctAgentManagerStub{}
	secondPath, secondHandler := apiconnect.NewAgentManagerServiceHandler(secondStub)
	secondMux := http.NewServeMux()
	secondMux.Handle(secondPath, secondHandler)
	second := httptest.NewServer(secondMux)
	defer second.Close()

	resolutions := 0
	client := &AgentManagerClient{
		BaseClient: BaseClient{httpClient: first.Client(), serviceName: "agent-manager"},
		maxRetries: 1, retryBaseDelay: time.Millisecond,
		resolveURL: func(context.Context) (string, error) {
			resolutions++
			if resolutions == 1 {
				return first.URL, nil
			}
			return second.URL, nil
		},
	}
	response, err := client.GetRun(context.Background(), "run-after-relocation")
	if err != nil {
		t.Fatalf("GetRun after retry: %v", err)
	}
	if response.GetRun().GetId() != "run-after-relocation" || firstStub.calls != 1 || resolutions != 2 {
		t.Fatalf("response=%+v first_calls=%d resolutions=%d", response.GetRun(), firstStub.calls, resolutions)
	}
}

type invalidRunAgentManagerStub struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	calls int
}

func (s *invalidRunAgentManagerStub) GetRun(context.Context, *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	s.calls++
	return nil, connect.NewError(connect.CodeInvalidArgument, io.ErrClosedPipe)
}

func TestAgentManagerConnectDoesNotRetryApplicationErrors(t *testing.T) {
	stub := &invalidRunAgentManagerStub{}
	path, handler := apiconnect.NewAgentManagerServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	resolutions := 0
	client := &AgentManagerClient{
		BaseClient: BaseClient{httpClient: server.Client(), serviceName: "agent-manager"},
		maxRetries: 2, retryBaseDelay: time.Millisecond,
		resolveURL: func(context.Context) (string, error) { resolutions++; return server.URL, nil },
	}
	_, err := client.GetRun(context.Background(), "run")
	if connect.CodeOf(err) != connect.CodeInvalidArgument || stub.calls != 1 || resolutions != 1 {
		t.Fatalf("error=%v code=%s calls=%d resolutions=%d", err, connect.CodeOf(err), stub.calls, resolutions)
	}
}
