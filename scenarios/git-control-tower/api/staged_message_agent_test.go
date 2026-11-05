package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/stagedmessages"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	advisoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stagedWorkflowFake struct {
	execution                             *ampb.WorkflowExecution
	start                                 *apipb.StartWorkflowExecutionRequest
	startHeaders                          http.Header
	starts, gets, waits, cancels, results int
	cancel                                *apipb.WorkflowExecutionOperationRequest
	wait                                  *apipb.WaitWorkflowExecutionRequest
	cancelStatus                          ampb.WorkflowExecutionStatus
	startErr                              error
}

func (f *stagedWorkflowFake) StartWorkflowExecution(_ context.Context, r *connect.Request[apipb.StartWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	f.starts++
	f.start = proto.Clone(r.Msg).(*apipb.StartWorkflowExecutionRequest)
	f.startHeaders = r.Header().Clone()
	if f.startErr != nil {
		return nil, f.startErr
	}
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: f.execution}), nil
}

func (f *stagedWorkflowFake) GetWorkflowExecution(context.Context, *connect.Request[apipb.GetWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	f.gets++
	if f.execution == nil {
		return connect.NewResponse(&apipb.WorkflowExecutionResponse{}), nil
	}
	metadata := proto.Clone(f.execution).(*ampb.WorkflowExecution)
	metadata.Input, metadata.Output = nil, nil
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: metadata}), nil
}

func (f *stagedWorkflowFake) GetWorkflowExecutionResult(_ context.Context, r *connect.Request[apipb.GetWorkflowExecutionResultRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	f.results++
	if !r.Msg.GetExplicitlyAuthorized() {
		return nil, errors.New("missing explicit authorization")
	}
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: f.execution}), nil
}

func (f *stagedWorkflowFake) WaitWorkflowExecution(_ context.Context, r *connect.Request[apipb.WaitWorkflowExecutionRequest]) (*connect.Response[apipb.WaitWorkflowExecutionResponse], error) {
	f.waits++
	f.wait = r.Msg
	return connect.NewResponse(&apipb.WaitWorkflowExecutionResponse{Execution: f.execution, TimedOut: true}), nil
}

func (f *stagedWorkflowFake) CancelWorkflowExecution(_ context.Context, r *connect.Request[apipb.WorkflowExecutionOperationRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error) {
	f.cancels++
	f.cancel = r.Msg
	f.execution.Status = f.cancelStatus
	return connect.NewResponse(&apipb.WorkflowExecutionOperationResponse{Execution: f.execution}), nil
}

type stagedSearchFake struct {
	requests []*ampb.SearchConversationsRequest
	headers  []http.Header
	response *ampb.SearchConversationsResponse
	err      error
}

func (f *stagedSearchFake) SearchConversations(_ context.Context, r *connect.Request[ampb.SearchConversationsRequest]) (*connect.Response[ampb.SearchConversationsResponse], error) {
	f.requests = append(f.requests, proto.Clone(r.Msg).(*ampb.SearchConversationsRequest))
	f.headers = append(f.headers, r.Header().Clone())
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.response), nil
}

func stagedAgentFixture(w *stagedWorkflowFake, s *stagedSearchFake) *stagedMessageAgent {
	return &stagedMessageAgent{resolve: func(context.Context) (string, error) { return "http://fixture", nil }, workflow: func(string) stagedMessageWorkflowAPI { return w }, search: func(string) stagedMessageSearchAPI { return s }, repository: func(context.Context, string) (string, error) { return "/workspace/registered-fixture", nil }}
}

func stagedAgentSnapshot() *advisoryv1.StagedMessageSnapshot {
	return &advisoryv1.StagedMessageSnapshot{Digest: "snapshot-1", RepositoryId: "fixture", Files: []*advisoryv1.StagedMessageFile{{Path: "api/session.go", Patch: "-return validateSession(token)\n+return validateBoundSession(token, owner)"}}}
}

func stagedAgentOwnerContext() context.Context {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Source: identity.SourceScenarioAuthenticator, ExpiresAt: time.Now().Add(time.Hour), Scopes: []string{"agent-manager:read"}})
	return policygate.WithVerifiedOwnerProof(ctx, http.Header{"Authorization": []string{"Bearer owner-fixture"}})
}

func stagedAgentExecution(status ampb.WorkflowExecutionStatus) *ampb.WorkflowExecution {
	return &ampb.WorkflowExecution{Id: "execution-1", Owner: "git-control-tower", WorkflowKey: stagedMessageWorkflowKey, Status: status, Version: 7, CurrentNodeId: "draft"}
}

func TestStagedMessageStartPinsInputAndIdempotencyWithoutMutations(t *testing.T) {
	w := &stagedWorkflowFake{execution: stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING)}
	p := stagedAgentFixture(w, nil)
	input := stagedmessages.DraftInput{Snapshot: stagedAgentSnapshot(), OperatorNote: "Explain behavior"}
	got, err := p.Start(stagedAgentOwnerContext(), input, "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "queued" || w.starts != 1 || w.start.GetIdempotencyKey() != "request-1" || w.start.GetWorkflowKey() != stagedMessageWorkflowKey {
		t.Fatalf("unbound start: %+v %+v", got, w.start)
	}
	if w.startHeaders.Get("Authorization") != "Bearer owner-fixture" || w.startHeaders.Get("X-Vrooli-Workflow-Initiator") != "human" {
		t.Fatalf("verified caller proof lost: %v", w.startHeaders)
	}
	fields := w.start.GetInput().GetStructValue().GetFields()
	if fields["snapshot"].GetStructValue().GetFields()["digest"].GetStringValue() != "snapshot-1" || fields["contexts"].GetListValue() == nil {
		t.Fatal("immutable input or empty context array lost")
	}
	if fields["repository_directory"].GetStringValue() != "/workspace/registered-fixture" {
		t.Fatal("workflow omitted trusted checkout selection")
	}
	input.OperatorNote = strings.Repeat("x", 4001)
	if _, err := p.Start(context.Background(), input, "request-2"); err == nil || w.starts != 1 {
		t.Fatal("oversized input dispatched")
	}
}

func TestStagedMessageStartRequiresForwardableCallerProof(t *testing.T) {
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "ambient-server-identity-is-not-browser-proof")
	for _, test := range []struct {
		name  string
		valid bool
	}{{"missing", false}, {"personal local", false}, {"invalid agent", false}, {"verified agent", true}} {
		t.Run(test.name, func(t *testing.T) {
			w := &stagedWorkflowFake{execution: stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING)}
			p := stagedAgentFixture(w, nil)
			req := httptest.NewRequest(http.MethodPost, "/start", nil)
			if test.name == "personal local" {
				req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: identity.ActorHuman, Subject: "osuser:1001", Source: identity.SourcePersonalLocal, Verified: true}))
			}
			if test.name == "invalid agent" || test.name == "verified agent" {
				req.Header.Set(cliutil.HeaderAgentIdentityToken, "inbound-agent")
			}
			var startErr error
			h := provenance.Middleware(provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
				return &cliutil.VerifyResult{Valid: test.valid, Claims: &cliutil.VerifiedClaims{RunID: "verified-parent"}}, nil
			}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				_, startErr = p.Start(r.Context(), stagedmessages.DraftInput{Snapshot: stagedAgentSnapshot()}, "request-proof")
			}))
			h.ServeHTTP(httptest.NewRecorder(), req)
			if test.valid {
				if startErr != nil || w.starts != 1 || w.startHeaders.Get("X-Vrooli-Workflow-Initiator") == "human" {
					t.Fatalf("verified agent incorrectly rejected or promoted: %v", startErr)
				}
			} else if connect.CodeOf(startErr) != connect.CodeFailedPrecondition || !strings.Contains(startErr.Error(), "Sign in through a trusted browser session") || w.starts != 0 {
				t.Fatalf("missing proof reached execution owner or lacks recovery: starts=%d err=%v", w.starts, startErr)
			}
		})
	}
}

type stagedMessageReadOwner struct {
	domainconnect.UnimplementedConversationSearchServiceHandler
	headers []http.Header
}

func (o *stagedMessageReadOwner) SearchConversations(_ context.Context, r *connect.Request[ampb.SearchConversationsRequest]) (*connect.Response[ampb.SearchConversationsResponse], error) {
	o.headers = append(o.headers, r.Header().Clone())
	return connect.NewResponse(&ampb.SearchConversationsResponse{}), nil
}

func TestStagedMessageConnectUsesRequestProofNeverServerCredential(t *testing.T) {
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "server-token-must-not-be-used")
	owner := &stagedMessageReadOwner{}
	path, handler := domainconnect.NewConversationSearchServiceHandler(owner)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	c := NewAgentManagerClient(time.Second)
	c.cachedBaseURL = server.URL
	c.httpClient = server.Client()
	p := newStagedMessageProvider(&Server{agentManagerClient: c})
	if _, _, err := p.Search(stagedAgentOwnerContext(), stagedAgentSnapshot()); err != nil {
		t.Fatal(err)
	}
	if len(owner.headers) == 0 {
		t.Fatal("owner was not queried")
	}
	for _, h := range owner.headers {
		if h.Get("Authorization") != "Bearer owner-fixture" || h.Get("X-Agent-Identity-Token") != "" || h.Get("Cookie") != "" {
			t.Fatalf("caller proof substituted or broadened: %v", h)
		}
	}
	owner.headers = nil
	if _, _, err := p.Search(context.Background(), stagedAgentSnapshot()); err != nil {
		t.Fatal(err)
	}
	for _, h := range owner.headers {
		if h.Get("Authorization") != "" || h.Get("X-Agent-Identity-Token") != "" {
			t.Fatal("anonymous context inherited server credential")
		}
	}
}

func TestStagedMessageGetWaitAndCancelUseOwnerLifecycle(t *testing.T) {
	w := &stagedWorkflowFake{execution: stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING), cancelStatus: ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLING}
	p := stagedAgentFixture(w, nil)
	got, err := p.Get(context.Background(), "execution-1", stagedmessages.DraftInput{})
	if err != nil || got.Status != "running" {
		t.Fatalf("get: %+v %v", got, err)
	}
	got, err = p.Wait(context.Background(), "execution-1", stagedmessages.DraftInput{}, 5*time.Minute)
	if err != nil || got.Status != "running" || w.wait.GetTimeoutSeconds() != 60 {
		t.Fatalf("wait: %+v %v", got, err)
	}
	got, err = p.Cancel(context.Background(), "execution-1", stagedmessages.DraftInput{})
	if err != nil || got.Status != "cancelling" || w.cancel.GetExpectedVersion() != 7 || w.cancel.GetIdempotencyKey() == "" {
		t.Fatalf("cancel prematurely terminal: %+v %v", got, err)
	}
	if w.starts != 0 {
		t.Fatal("reconnect/wait/cancel dispatched another agent")
	}
	w.execution.Status = ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED
	got, err = p.Cancel(context.Background(), "execution-1", stagedmessages.DraftInput{})
	if err != nil || got.Status != "cancelled" || w.cancels != 1 {
		t.Fatal("terminal cancellation repeated owner mutation")
	}
}

func stagedAgentComplete(t *testing.T) *ampb.WorkflowExecution {
	t.Helper()
	e := stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED)
	input, _ := structpb.NewValue(map[string]any{"snapshot": map[string]any{"digest": "snapshot-1", "files": []any{map[string]any{"path": "api/session.go"}}}, "contexts": []any{map[string]any{"id": "context-1"}}})
	output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"snapshot_digest": "snapshot-1", "title": "fix: bind session validation", "body": "Validate the owner when accepting a session.", "trailers": []any{}, "covered_paths": []any{"api/session.go"}, "uncovered_paths": []any{}, "partial": false, "source_ids": []any{"context-1"}, "unknowns": []any{}}})
	e.Input, e.Output = input, output
	return e
}

func stagedLegacyInput(t *testing.T, e *ampb.WorkflowExecution) stagedmessages.DraftInput {
	t.Helper()
	raw, _ := json.Marshal(e.Input.AsInterface())
	var input stagedmessages.DraftInput
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	return input
}

func TestStagedMessageResultsRejectUnboundAndInventedEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{{"valid", func(map[string]any) {}}, {"wrong snapshot", func(r map[string]any) { r["snapshot_digest"] = "other" }}, {"unknown path", func(r map[string]any) { r["covered_paths"] = []any{"secrets.go"} }}, {"unknown source", func(r map[string]any) { r["source_ids"] = []any{"other"} }}, {"inferred trailer", func(r map[string]any) { r["trailers"] = []any{map[string]any{"key": "Vrooli-Run", "value": "x"}} }}, {"multiline title", func(r map[string]any) { r["title"] = "first\nsecond" }}, {"coverage omitted", func(r map[string]any) { r["covered_paths"] = []any{} }}} {
		t.Run(test.name, func(t *testing.T) {
			e := stagedAgentComplete(t)
			out := e.Output.AsInterface().(map[string]any)
			test.mutate(out["result"].(map[string]any))
			e.Output, _ = structpb.NewValue(out)
			w := &stagedWorkflowFake{execution: e}
			got, err := stagedAgentFixture(w, nil).Get(context.Background(), e.Id, stagedLegacyInput(t, e))
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "valid" {
				if got.Status != "succeeded" || got.Result == nil {
					t.Fatalf("valid result refused: %+v", got)
				}
			} else if got.Status != "failed" || got.Result != nil {
				t.Fatalf("unsafe output adopted: %+v", got)
			}
		})
	}
}

func TestStagedMessageManyPathCoverageFitsValidatedOutputBound(t *testing.T) {
	e := stagedAgentComplete(t)
	input := e.Input.AsInterface().(map[string]any)
	snapshot := input["snapshot"].(map[string]any)
	var files, paths []any
	for i := 0; i < 256; i++ {
		path := fmt.Sprintf("directory/%03d/%s.go", i, strings.Repeat("long_path_segment", 6))
		files = append(files, map[string]any{"path": path})
		paths = append(paths, path)
	}
	snapshot["files"] = files
	e.Input, _ = structpb.NewValue(input)
	output := e.Output.AsInterface().(map[string]any)
	output["result"].(map[string]any)["covered_paths"] = paths
	e.Output, _ = structpb.NewValue(output)
	encoded, _ := json.Marshal(output)
	if len(encoded) <= 20000 {
		t.Fatal("fixture did not reproduce former bound mismatch")
	}
	got, err := decodeStagedMessageResult(e)
	if err != nil || len(got.GetCoverage().GetCoveredPaths()) != 256 || got.GetCoverage().GetTotalEdits() != 256 {
		t.Fatalf("complete bounded coverage rejected: %+v %v", got, err)
	}
}

type stagedDeadlineSearch struct{ calls int }

func (f *stagedDeadlineSearch) SearchConversations(ctx context.Context, _ *connect.Request[ampb.SearchConversationsRequest]) (*connect.Response[ampb.SearchConversationsResponse], error) {
	f.calls++
	if f.calls == 1 {
		return connect.NewResponse(&ampb.SearchConversationsResponse{Hits: []*ampb.ConversationSearchHit{{RunId: "first", EventId: "event", Snippet: "api/session.go"}}}), nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestStagedMessageContextDeadlineRetainsPartialEvidenceAndCallerCancellation(t *testing.T) {
	s := &stagedDeadlineSearch{}
	p := stagedAgentFixture(nil, nil)
	p.search = func(string) stagedMessageSearchAPI { return s }
	p.searchBudget = 20 * time.Millisecond
	contexts, omissions, err := p.Search(context.Background(), stagedAgentSnapshot())
	if err != nil || len(contexts) != 1 || len(omissions) == 0 || !strings.Contains(omissions[len(omissions)-1].GetReason(), "wall-time") {
		t.Fatalf("partial evidence lost at search budget: %v %v %v", contexts, omissions, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := p.Search(ctx, stagedAgentSnapshot()); !errors.Is(err, context.Canceled) {
		t.Fatalf("caller cancellation concealed: %v", err)
	}
}

func TestStagedMessageContextSearchIsBoundedPolicyVisibleAndNoAgent(t *testing.T) {
	stamp := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	snapshot := stagedAgentSnapshot()
	snapshot.Files[0].TimingHints = []*advisoryv1.StagedMessageTimingHint{{Source: "filesystem_mtime", Timestamp: stamp.Format(time.RFC3339), ContentMatchesIndex: true}}
	s := &stagedSearchFake{response: &ampb.SearchConversationsResponse{Hits: []*ampb.ConversationSearchHit{{RunId: "run-1", EventId: "event-1", ChunkId: "chunk-1", OccurredAt: timestamppb.New(stamp), Snippet: "Write api/session.go validateBoundSession", Provenance: &ampb.ConversationSourceProvenance{Importer: "agent-manager.transcript-import", SourceSessionId: "native-session"}}}}}
	w := &stagedWorkflowFake{}
	p := stagedAgentFixture(w, s)
	contexts, _, err := p.Search(stagedAgentOwnerContext(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(contexts) != 1 || contexts[0].GetFullMatch() || contexts[0].GetMatchedEdits() != 0 || contexts[0].GetRevision() == "" || w.starts != 0 {
		t.Fatalf("context became attribution or started agent: %+v", contexts)
	}
	unknowns := strings.Join(contexts[0].Unknowns, " ")
	if !strings.Contains(unknowns, "timestamp origin") || !strings.Contains(unknowns, "tail/resumption freshness") {
		t.Fatalf("missing provenance limits: %s", unknowns)
	}
	var timed bool
	for i, r := range s.requests {
		if r.GetMode() != ampb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_TEXT || !r.GetFilters().GetIncludeToolEvents() || len(r.GetFilters().GetProjectScopes()) > 0 {
			t.Fatal("search bypassed text/tool or unknown-scope policy")
		}
		if s.headers[i].Get("Authorization") != "Bearer owner-fixture" {
			t.Fatal("read proof lost")
		}
		if r.Filters.OccurredAfter != nil {
			timed = true
			if r.Query != "" || r.Sort != ampb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST || r.Filters.OccurredAfter.AsTime() != stamp.Add(-2*time.Minute) {
				t.Fatal("mtime query did not use event window")
			}
		}
	}
	if !timed || len(s.requests) > 6 {
		t.Fatal("missing bounded time retrieval")
	}
	s.err = connect.NewError(connect.CodeUnauthenticated, errors.New("denied"))
	s.requests = nil
	contexts, omissions, err := p.Search(context.Background(), snapshot)
	if err != nil || len(contexts) != 0 || len(omissions) == 0 || len(s.requests) != 1 {
		t.Fatal("owner denial was retried or concealed")
	}
}

func TestStagedMessageReadOnlyWorkflowDeclaration(t *testing.T) {
	for _, path := range []string{"../.vrooli/agent-manager/staged-message-profile.json", "../.vrooli/agent-manager/staged-message.json"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var asset map[string]any
		if err = json.Unmarshal(data, &asset); err != nil {
			t.Fatal(err)
		}
		var config map[string]any
		wantMode := "protected"
		if strings.Contains(path, "profile") {
			wantMode = "SANDBOX_MODE_PROTECTED"
			config = asset["sandboxConfig"].(map[string]any)
			if scopes := asset["declaredScopes"].([]any); len(scopes) != 0 {
				t.Fatal("drafting agent receives API mutation authority")
			}
			if asset["toolRestrictionPolicy"] != "advisory" {
				t.Fatal("Codex unsupported tool allowlist must not be claimed enforced")
			}
			if asset["networkAccess"] != "NETWORK_ACCESS_NONE" || config["networkMode"] != "NETWORK_ACCESS_FULL" {
				t.Fatal("native tools and model transport network boundaries were conflated")
			}
			if asset["spawnPolicy"].(map[string]any)["require"].([]any)[0].(map[string]any)["executionMode"] != "codec_pipe" {
				t.Fatal("protected worker could choose an unsupported interactive substrate")
			}
		} else {
			run := asset["nodes"].([]any)[0].(map[string]any)["run"].(map[string]any)
			config = run["sandboxConfig"].(map[string]any)
			if run["profileKey"] != "git-control-tower/staged-message" {
				t.Fatal("wrong identity ceiling")
			}
			spec := run["resultSpec"].(map[string]any)
			if spec["extractionMode"] != "deterministic_only" {
				t.Fatal("result completion could trigger inference")
			}
		}
		if config["mode"] != wantMode || config["autoApply"] != false || config["applyOnFailure"] != false || len(config["writePolicy"].(map[string]any)["paths"].([]any)) != 0 {
			t.Fatalf("read-only execution boundary lost in %s", path)
		}
	}
}

func TestStagedMessageBudgetFailureExplainsActualOwnerReason(t *testing.T) {
	for _, tc := range []struct{ name, budget, message, want string }{
		{"actual retained token exhaustion", "tokens", "", "Drafting stopped because the workflow token budget was exhausted."},
		{"owner message retained", "tokens", "Provider token allowance was reached.", "Provider token allowance was reached."},
		{"charge exhaustion", "charge", "", "Drafting stopped because the workflow spending budget was exhausted."},
		{"time exhaustion", "wall_time", "", "Drafting stopped because the workflow time budget was exhausted."},
		{"unknown budget", "future-budget", "", "Drafting stopped because a workflow budget was exhausted."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED)
			e.TerminalReason = &ampb.WorkflowTerminalReason{Code: "budget_exhausted", BudgetName: tc.budget, Message: tc.message}
			p := stagedAgentFixture(&stagedWorkflowFake{execution: e}, nil)
			got, err := p.Get(context.Background(), e.Id, stagedmessages.DraftInput{})
			if err != nil || got.Status != "failed" || got.Error != tc.want {
				t.Fatalf("owner failure lost: status=%s reason=%q err=%v", got.Status, got.Error, err)
			}
		})
	}
	t.Run("missing typed reason", func(t *testing.T) {
		e := stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED)
		if got := stagedMessageTerminalError(e); got != "Drafting stopped because a workflow budget was exhausted." {
			t.Fatalf("missing reason left budget failure unexplained: %q", got)
		}
	})
}
