package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/policygate"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	agentreadsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/agent_reads"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type agentReadOwnerStub struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	profiles   *apipb.ListProfilesResponse
	runs       *apipb.ListRunsResponse
	run        *apipb.GetRunResponse
	task       *apipb.GetTaskResponse
	events     *apipb.GetRunEventsResponse
	diff       *apipb.GetRunDiffResponse
	err        error
	tagPrefix  string
	limit      int32
	after      int64
	auth       string
	runCalls   int
	eventCalls int
}

func (s *agentReadOwnerStub) ListProfiles(_ context.Context, req *connect.Request[apipb.ListProfilesRequest]) (*connect.Response[apipb.ListProfilesResponse], error) {
	s.auth = req.Header().Get("Authorization")
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.profiles), nil
}
func (s *agentReadOwnerStub) ListRuns(_ context.Context, req *connect.Request[apipb.ListRunsRequest]) (*connect.Response[apipb.ListRunsResponse], error) {
	s.auth = req.Header().Get("Authorization")
	s.tagPrefix, s.limit = req.Msg.GetTagPrefix(), req.Msg.GetLimit()
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.runs), nil
}
func (s *agentReadOwnerStub) GetRun(_ context.Context, req *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	s.runCalls++
	s.auth = req.Header().Get("Authorization")
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.run), nil
}
func (s *agentReadOwnerStub) GetTask(_ context.Context, req *connect.Request[apipb.GetTaskRequest]) (*connect.Response[apipb.GetTaskResponse], error) {
	s.auth = req.Header().Get("Authorization")
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.task), nil
}
func (s *agentReadOwnerStub) GetRunEvents(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
	s.eventCalls++
	s.auth = req.Header().Get("Authorization")
	s.after, s.limit = req.Msg.GetAfterSequence(), req.Msg.GetLimit()
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.events), nil
}
func (s *agentReadOwnerStub) GetRunDiff(_ context.Context, req *connect.Request[apipb.GetRunDiffRequest]) (*connect.Response[apipb.GetRunDiffResponse], error) {
	s.auth = req.Header().Get("Authorization")
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.diff), nil
}

func agentReadTestServer(t *testing.T, status CapabilityStatus, stub *agentReadOwnerStub) *Server {
	t.Helper()
	path, handler := apiconnect.NewAgentManagerServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	upstream := httptest.NewServer(mux)
	t.Cleanup(upstream.Close)
	client := &AgentManagerClient{BaseClient: BaseClient{httpClient: upstream.Client(), resolver: discovery.NewStaticResolver(upstream.URL), serviceName: "agent-manager"}}
	registry := NewCapabilityRegistry([]CapabilityDef{{ID: "agent-manager"}}, map[string]StatusChecker{"agent-manager": &FakeStatusChecker{Status: status}}, time.Minute)
	git := NewFakeGitRunner()
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	git.RepoRoot = repoRoot
	return &Server{git: git, agentManagerClient: client, capabilities: registry}
}

func verifiedAgentReadContext() context.Context {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Source: identity.SourceScenarioAuthenticator, ExpiresAt: time.Now().Add(time.Hour), Scopes: []string{"git-control-tower:read", "agent-manager:read"}}
	ctx := identity.WithPrincipal(context.Background(), principal)
	return policygate.WithVerifiedOwnerProof(ctx, http.Header{"Authorization": []string{"Bearer verified-owner-token"}})
}

func TestAgentReadConnectProxiesTypedOwnerResultsWithVerifiedIdentity(t *testing.T) {
	created := timestamppb.New(time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC))
	stub := &agentReadOwnerStub{
		profiles: &apipb.ListProfilesResponse{Profiles: []*ampb.AgentProfile{{Id: "p1", ProfileKey: "reviewer", Name: "Reviewer", RoleRef: "code.review"}}, Total: 1},
		runs:     &apipb.ListRunsResponse{Runs: []*ampb.Run{{Id: "r1", Tag: "gct-git-control-tower", TaskId: "t1", Status: ampb.RunStatus_RUN_STATUS_COMPLETE, PromptPreview: "inspect", CreatedAt: created, Summary: &ampb.RunSummary{FilesModified: []string{"main.go"}, TokensUsed: 21, TurnsUsed: 2}}, {Id: "r-other", Tag: "gct-git-control-tower-extra", Status: ampb.RunStatus_RUN_STATUS_RUNNING}}, Total: 2},
		run:      &apipb.GetRunResponse{Run: &ampb.Run{Id: "r1", Tag: "gct-git-control-tower", TaskId: "t1", Status: ampb.RunStatus_RUN_STATUS_RUNNING, CreatedAt: created, Actions: &ampb.RunActions{CanStop: true, CanContinue: true}}},
		task:     &apipb.GetTaskResponse{Task: &ampb.Task{Id: "t1", ScopePath: "scenarios/git-control-tower/"}},
		events:   &apipb.GetRunEventsResponse{Events: []*ampb.RunEvent{{Id: "e1", RunId: "r1", Sequence: 7, EventType: ampb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Timestamp: created, Data: &ampb.RunEvent_Message{Message: &ampb.MessageEventData{Role: "assistant", Content: "hello"}}}}},
		diff:     &apipb.GetRunDiffResponse{Diff: &ampb.RunDiff{Content: "diff --git", Files: []*ampb.FileDiff{{Path: "main.go", ChangeType: "modified", Additions: 3, Deletions: 1, Patch: "+x"}}}},
	}
	s := agentReadConnectServer{server: agentReadTestServer(t, StatusAvailable, stub)}
	ctx := verifiedAgentReadContext()

	profiles, err := s.ListProfiles(ctx, connect.NewRequest(&agentreadsv1.ListProfilesRequest{}))
	if err != nil || profiles.Msg.GetProfiles()[0].GetKey() != "reviewer" || profiles.Msg.GetTotal() != 1 || stub.auth != "Bearer verified-owner-token" {
		t.Fatalf("profile projection/auth = %#v, %v, %q", profiles, err, stub.auth)
	}
	runs, err := s.ListRuns(ctx, connect.NewRequest(&agentreadsv1.ListRunsRequest{ScenarioSlug: "git-control-tower", Limit: 80}))
	if err != nil || stub.tagPrefix != "gct-git-control-tower" || stub.limit != 50 || runs.Msg.GetTotal() != 1 || runs.Msg.GetRuns()[0].GetSummary().GetTokensUsed() != 21 || runs.Msg.GetRuns()[0].GetStatus() != "complete" {
		t.Fatalf("run list = %#v, %v (tag=%q limit=%d)", runs, err, stub.tagPrefix, stub.limit)
	}
	run, err := s.GetRun(ctx, connect.NewRequest(&agentreadsv1.GetRunRequest{RunId: "r1", ScenarioSlug: "git-control-tower"}))
	if err != nil || run.Msg.GetActions().GetCanStop() != true || run.Msg.GetCreatedAt() != "2026-11-21T10:00:00Z" {
		t.Fatalf("run detail = %#v, %v", run, err)
	}
	events, err := s.GetRunEvents(ctx, connect.NewRequest(&agentreadsv1.GetRunEventsRequest{RunId: "r1", ScenarioSlug: "git-control-tower", AfterSequence: int64Ptr(5), Limit: 900}))
	if err != nil || stub.after != 5 || stub.limit != 500 || events.Msg.GetEvents()[0].GetData().GetFields()["content"].GetStringValue() != "hello" {
		t.Fatalf("events = %#v, %v (after=%d limit=%d)", events, err, stub.after, stub.limit)
	}
	diff, err := s.GetRunDiff(ctx, connect.NewRequest(&agentreadsv1.GetRunDiffRequest{RunId: "r1", ScenarioSlug: "git-control-tower"}))
	if err != nil || diff.Msg.GetContent() != "diff --git" || diff.Msg.GetFiles()[0].GetAdditions() != 3 {
		t.Fatalf("diff = %#v, %v", diff, err)
	}
}

func TestAgentReadConnectDefaultsAndRejectsInvalidWithoutOwnerCall(t *testing.T) {
	stub := &agentReadOwnerStub{events: &apipb.GetRunEventsResponse{Events: []*ampb.RunEvent{}}}
	s := agentReadConnectServer{server: agentReadTestServer(t, StatusAvailable, stub)}
	stub.run = &apipb.GetRunResponse{Run: &ampb.Run{Id: "r1", Tag: "gct-git-control-tower", TaskId: "t1"}}
	stub.task = &apipb.GetTaskResponse{Task: &ampb.Task{Id: "t1", ScopePath: "scenarios/git-control-tower/"}}
	_, err := s.GetRunEvents(verifiedAgentReadContext(), connect.NewRequest(&agentreadsv1.GetRunEventsRequest{RunId: "r1", ScenarioSlug: "git-control-tower"}))
	if err != nil || stub.after != -1 || stub.limit != 100 {
		t.Fatalf("event defaults after=%d limit=%d err=%v", stub.after, stub.limit, err)
	}
	_, err = s.ListRuns(verifiedAgentReadContext(), connect.NewRequest(&agentreadsv1.ListRunsRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty scenario code=%v err=%v", connect.CodeOf(err), err)
	}
	if stub.tagPrefix != "" {
		t.Fatalf("invalid request reached owner: %q", stub.tagPrefix)
	}
}

func TestAgentReadConnectRejectsRunsBoundToAnotherScenario(t *testing.T) {
	stub := &agentReadOwnerStub{run: &apipb.GetRunResponse{Run: &ampb.Run{Id: "r1", Tag: "gct-other", TaskId: "t1"}}}
	s := agentReadConnectServer{server: agentReadTestServer(t, StatusAvailable, stub)}
	_, err := s.GetRunEvents(verifiedAgentReadContext(), connect.NewRequest(&agentreadsv1.GetRunEventsRequest{RunId: "r1", ScenarioSlug: "git-control-tower"}))
	if connect.CodeOf(err) != connect.CodeNotFound || stub.runCalls != 1 || stub.eventCalls != 0 {
		t.Fatalf("cross-scenario run binding code=%v err=%v owner get=%d events=%d", connect.CodeOf(err), err, stub.runCalls, stub.eventCalls)
	}
}

func TestAgentReadConnectRejectsTaskOutsideSelectedScenarioScope(t *testing.T) {
	stub := &agentReadOwnerStub{
		run:  &apipb.GetRunResponse{Run: &ampb.Run{Id: "r1", Tag: "gct-git-control-tower", TaskId: "t1"}},
		task: &apipb.GetTaskResponse{Task: &ampb.Task{Id: "t1", ScopePath: "scenarios/other/"}},
	}
	s := agentReadConnectServer{server: agentReadTestServer(t, StatusAvailable, stub)}
	request := connect.NewRequest(&agentreadsv1.GetRunEventsRequest{RunId: "r1", ScenarioSlug: "git-control-tower"})
	_, err := s.GetRunEvents(verifiedAgentReadContext(), request)
	if connect.CodeOf(err) != connect.CodeNotFound || stub.eventCalls != 0 {
		t.Fatalf("scope mismatch code=%v err=%v events=%d", connect.CodeOf(err), err, stub.eventCalls)
	}
}

func TestAgentReadConnectPreservesOwnerAuthFailureAndUnavailable(t *testing.T) {
	denied := &agentReadOwnerStub{err: connect.NewError(connect.CodePermissionDenied, errors.New("agent-manager read scope required"))}
	s := agentReadConnectServer{server: agentReadTestServer(t, StatusAvailable, denied)}
	_, err := s.GetRun(verifiedAgentReadContext(), connect.NewRequest(&agentreadsv1.GetRunRequest{RunId: "r1", ScenarioSlug: "demo"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("owner identity failure code=%v err=%v", connect.CodeOf(err), err)
	}
	unavailable := &agentReadOwnerStub{}
	s = agentReadConnectServer{server: agentReadTestServer(t, StatusUnavailable, unavailable)}
	_, err = s.ListProfiles(context.Background(), connect.NewRequest(&agentreadsv1.ListProfilesRequest{}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("capability failure code=%v err=%v", connect.CodeOf(err), err)
	}
}

func int64Ptr(value int64) *int64 { return &value }
