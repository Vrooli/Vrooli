package corpusgen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/servicetoken"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

type fakeRunEventsReader struct {
	pages  map[string][]AgentManagerEventPage
	calls  map[string]int
	status domain.RunStatus
}

func (f *fakeRunEventsReader) GetRun(_ context.Context, runID string) (*domain.Run, error) {
	return &domain.Run{Id: runID, Status: f.status}, nil
}

func (f *fakeRunEventsReader) GetRunEvents(_ context.Context, runID string, _ int64, _ int) (AgentManagerEventPage, error) {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	index := f.calls[runID]
	f.calls[runID]++
	if index >= len(f.pages[runID]) {
		return AgentManagerEventPage{}, nil
	}
	return f.pages[runID][index], nil
}

func TestHarvestAgentManagerCallersRetainsOnlyRealOperatorUserRecords(t *testing.T) {
	run := "run-real"
	reader := &fakeRunEventsReader{status: domain.RunStatus_RUN_STATUS_COMPLETE, pages: map[string][]AgentManagerEventPage{run: {
		{Events: []AgentManagerEvent{
			{ID: "e1", RunID: run, Sequence: 1, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "user", Source: "operator", ConversationID: "conv-1", Content: "Where is retry handled?"}},
			{ID: "e2", RunID: run, Sequence: 2, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "user", Source: "injected_context", ConversationID: "conv-1", Content: "synthetic injected prompt"}},
			{ID: "e3", RunID: run, Sequence: 3, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "assistant", Source: "operator", ConversationID: "conv-1", Content: "assistant response"}},
			{ID: "e4", RunID: run, Sequence: 4, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "user", Source: "operator", ConversationID: "conv-1", Content: "deleted question"}},
			{ID: "e5", RunID: run, Sequence: 5, EventType: "RUN_EVENT_TYPE_MESSAGE_DELETED", MessageDeleted: &AgentManagerMessageDeletion{TargetEventID: "e4"}},
		}},
	}}}
	got, err := HarvestAgentManagerCallers(context.Background(), reader, []string{run})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d callers, want 1: %+v", len(got), got)
	}
	if got[0].RunID != run || got[0].EventID != "e1" || got[0].Sequence != 1 || got[0].ConversationID != "conv-1" || got[0].MessageRole != "user" || got[0].MessageSource != "operator" || got[0].Query != "Where is retry handled?" {
		t.Fatalf("harvest lost source identity or caller role: %+v", got[0])
	}
}

func TestAgentManagerConnectReaderUsesBodyFreeIdentityReceiptOwnerRoute(t *testing.T) {
	h := strings.Repeat("a", 64)
	binding := AgentManagerCodeTaskSourceIdentityBinding{Purpose: "local-coding-benchmark-fixture/v1", RunID: "00000000-0000-4000-8000-000000000001", EventID: "00000000-0000-4000-8000-000000000002", Sequence: 8, ConversationID: "00000000-0000-4000-8000-000000000003", MessageSHA256: h, TaskSpecSHA256: h, RepositoryID: "vrooli", RepositoryRevision: strings.Repeat("b", 40), RepositoryTreeSHA256: strings.Repeat("c", 64)}
	want := AgentManagerCodeTaskSourceIdentityReceipt{Binding: binding, ActorSubjectSHA256: strings.Repeat("d", 64), ActorSubjectDigestVersion: "agent-manager-source-author-hmac/v1", ActorIdentityReceiptID: "am-source-identity-v1:" + binding.EventID, ActorIdentityReceiptSHA256: strings.Repeat("e", 64)}
	var received AgentManagerCodeTaskSourceIdentityBinding
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/code-task/source-identity-receipts" {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fixture-service-token" {
			t.Errorf("authorization=%q", got)
		}
		if r.Header.Get("X-Agent-Identity-Token") != "" {
			t.Error("forwarded caller identity")
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer server.Close()
	reader := NewAgentManagerConnectReader(server.URL, server.Client())
	reader.Identity = fixtureSearchHubAgentManagerTokens()
	got, err := reader.GetCodeTaskSourceIdentityReceipt(context.Background(), binding)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("receipt=%+v want=%+v", got, want)
	}
	if received != binding {
		t.Fatalf("request binding=%+v want=%+v", received, binding)
	}
}

func fixtureSearchHubAgentManagerTokens() *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{Credential: func(context.Context) (string, error) { return "fixture-credential", nil }, Expect: servicetoken.ExpectIdentity(agentManagerSourcePrincipal, agentManagerSourceAudience, agentManagerSourceScope)}, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "fixture-service-token", ExpiresAt: time.Now().Add(time.Hour), Subject: agentManagerSourcePrincipal, Audience: agentManagerSourceAudience, Scopes: []string{agentManagerSourceScope}}, nil
	}))
}

func TestAgentManagerIdentityReceiptClientFailsClosedWithoutIdentityAndOnOwnerErrors(t *testing.T) {
	binding := AgentManagerCodeTaskSourceIdentityBinding{Purpose: "local-coding-benchmark-fixture/v1", RunID: "run-1", EventID: "event-1", Sequence: 7, ConversationID: "conv-1", MessageSHA256: strings.Repeat("a", 64), TaskSpecSHA256: strings.Repeat("a", 64), RepositoryID: "vrooli", RepositoryRevision: strings.Repeat("b", 40), RepositoryTreeSHA256: strings.Repeat("c", 64)}
	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		http.Error(w, "private owner detail", http.StatusUnprocessableEntity)
	}))
	defer server.Close()
	reader := NewAgentManagerConnectReader(server.URL, server.Client())
	if _, err := reader.GetCodeTaskSourceIdentityReceipt(context.Background(), binding); err == nil || called != 0 {
		t.Fatalf("unconfigured identity err=%v calls=%d", err, called)
	}
	reader.Identity = fixtureSearchHubAgentManagerTokens()
	if _, err := reader.GetCodeTaskSourceIdentityReceipt(context.Background(), binding); err == nil || strings.Contains(err.Error(), "private owner detail") || called != 1 {
		t.Fatalf("owner refusal err=%v calls=%d", err, called)
	}
}

func TestDefaultAgentManagerSourceOwnersUseAuthenticatedReceiptRoute(t *testing.T) {
	source, identity := NewDefaultAgentManagerCodeTaskSourceOwners()
	if nilCodeTaskOwner(source) || nilCodeTaskOwner(identity) {
		t.Fatal("default Agent Manager source owners must be constructed")
	}
	clients := CodeTaskEvidenceOwnerClients{Source: source, SourceIdentity: identity}
	if _, err := NewCodeTaskEvidenceOwnerAdapter(clients); err == nil {
		t.Fatal("partial production owner set must remain unavailable until all owners are configured")
	}
}

func TestAuthenticatedHTTPReceiptRoundTripRejectsMismatchedTaskAndRepo(t *testing.T) {
	candidate := codeTaskCandidateFixture(982, "single-file-change")
	binding := CodeTaskSourceIdentityBinding{Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository}
	want := AgentManagerCodeTaskSourceIdentityBinding{Purpose: "local-coding-benchmark-fixture/v1", RunID: candidate.Source.RunID, EventID: candidate.Source.EventID, Sequence: candidate.Source.Sequence, ConversationID: candidate.Source.ConversationID, MessageSHA256: candidate.Source.MessageSHA256, TaskSpecSHA256: candidate.TaskSpecSHA256, RepositoryID: candidate.Repository.RepositoryID, RepositoryRevision: candidate.Repository.Revision, RepositoryTreeSHA256: candidate.Repository.TreeSHA256}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != agentManagerSourceReceiptPath {
			t.Errorf("route=%s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer fixture-service-token" {
			t.Errorf("service authorization missing")
		}
		if r.Header.Get("X-Agent-Identity-Token") != "" || r.Header.Get("Cookie") != "" {
			t.Error("caller identity forwarded")
		}
		var got AgentManagerCodeTaskSourceIdentityBinding
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if got != want {
			t.Errorf("request binding=%+v want %+v", got, want)
		}
		_ = json.NewEncoder(w).Encode(AgentManagerCodeTaskSourceIdentityReceipt{Binding: want, ActorSubjectSHA256: candidate.Source.ActorSubjectSHA256, ActorSubjectDigestVersion: candidate.Source.ActorSubjectDigestVersion, ActorIdentityReceiptID: candidate.Source.ActorIdentityReceiptID, ActorIdentityReceiptSHA256: candidate.Source.ActorIdentityReceiptSHA256})
	}))
	defer server.Close()
	reader := NewAgentManagerConnectReader(server.URL, server.Client())
	reader.Identity = fixtureSearchHubAgentManagerTokens()
	verifier := AgentManagerCodeTaskSourceIdentityVerifier{Receipts: reader}
	if err := verifier.VerifyCodeTaskActorIdentity(context.Background(), binding); err != nil {
		t.Fatalf("authenticated receipt rejected: %v", err)
	}
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-service-token" {
			t.Errorf("service authorization missing")
		}
		wrong := want
		wrong.RepositoryRevision = strings.Repeat("f", 40)
		_ = json.NewEncoder(w).Encode(AgentManagerCodeTaskSourceIdentityReceipt{Binding: wrong, ActorSubjectSHA256: candidate.Source.ActorSubjectSHA256, ActorSubjectDigestVersion: candidate.Source.ActorSubjectDigestVersion, ActorIdentityReceiptID: candidate.Source.ActorIdentityReceiptID, ActorIdentityReceiptSHA256: candidate.Source.ActorIdentityReceiptSHA256})
	}))
	defer badServer.Close()
	reader.BaseURL = badServer.URL
	verifier.Receipts = reader
	if err := verifier.VerifyCodeTaskActorIdentity(context.Background(), binding); err == nil {
		t.Fatal("accepted owner receipt for a different repository revision")
	}
}

func TestHarvestAgentManagerCallersValidatesPaginationIdentity(t *testing.T) {
	reader := &fakeRunEventsReader{status: domain.RunStatus_RUN_STATUS_COMPLETE, pages: map[string][]AgentManagerEventPage{"run-1": {{Events: []AgentManagerEvent{{ID: "wrong-run-event", RunID: "run-other", Sequence: 1}}}}}}
	if _, err := HarvestAgentManagerCallers(context.Background(), reader, []string{"run-1"}); err == nil {
		t.Fatal("expected mismatched event run identity to be rejected")
	}
}

func TestHarvestAgentManagerCallersRefusesActiveRunBeforeReadingEvents(t *testing.T) {
	reader := &fakeRunEventsReader{status: domain.RunStatus_RUN_STATUS_RUNNING, pages: map[string][]AgentManagerEventPage{
		"run-1": {{Events: []AgentManagerEvent{{ID: "private-active-message", RunID: "run-1", Sequence: 1, EventType: "RUN_EVENT_TYPE_MESSAGE", Message: &AgentManagerMessage{Role: "user", Source: "operator", Content: "active transcript"}}}}},
	}}
	if _, err := HarvestAgentManagerCallers(context.Background(), reader, []string{"run-1"}); err == nil || !strings.Contains(err.Error(), "unless the exact run is complete") {
		t.Fatalf("active run transcript was not refused: %v", err)
	}
	if reader.calls["run-1"] != 0 {
		t.Fatalf("active run events were read before completion proof: calls=%d", reader.calls["run-1"])
	}
}

type agentManagerEventsFixture struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	pages          map[int64]*api.GetRunEventsResponse
	sourceMessages map[int64]*api.GetRunEventsResponse
	deletionPages  map[int64]*api.GetRunEventsResponse
	calls          int
	runID          string
	status         domain.RunStatus
}

func (f *agentManagerEventsFixture) GetRun(_ context.Context, req *connect.Request[api.GetRunRequest]) (*connect.Response[api.GetRunResponse], error) {
	if req.Msg.GetRunId() != f.runID {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&api.GetRunResponse{Run: &domain.Run{Id: f.runID, Status: f.status}}), nil
}

func (f *agentManagerEventsFixture) GetRunEvents(_ context.Context, req *connect.Request[api.GetRunEventsRequest]) (*connect.Response[api.GetRunEventsResponse], error) {
	f.calls++
	if req.Msg.GetRunId() == "" || req.Msg.GetSummaryOnly() {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	var source map[int64]*api.GetRunEventsResponse
	switch {
	case len(req.Msg.GetEventTypes()) == 2 && req.Msg.GetLimit() == 1000:
		source = f.pages
	case len(req.Msg.GetEventTypes()) == 1 && req.Msg.GetEventTypes()[0] == domain.RunEventType_RUN_EVENT_TYPE_MESSAGE && req.Msg.GetLimit() == 1:
		source = f.sourceMessages
	case len(req.Msg.GetEventTypes()) == 1 && req.Msg.GetEventTypes()[0] == domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED && req.Msg.GetLimit() == 1000:
		source = f.deletionPages
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	response, ok := source[req.Msg.GetAfterSequence()]
	if !ok {
		return connect.NewResponse(&api.GetRunEventsResponse{}), nil
	}
	return connect.NewResponse(response), nil
}

func runEvent(id, runID string, sequence int64, data any) *domain.RunEvent {
	event := &domain.RunEvent{Id: id, RunId: runID, Sequence: sequence}
	switch data := data.(type) {
	case *domain.RunEvent_Message:
		event.EventType = domain.RunEventType_RUN_EVENT_TYPE_MESSAGE
		event.Data = data
	case *domain.RunEvent_MessageDeleted:
		event.EventType = domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED
		event.Data = data
	}
	return event
}

func serveAgentManagerEventFixture(t *testing.T, fixture *agentManagerEventsFixture) AgentManagerConnectReader {
	t.Helper()
	path, handler := apiconnect.NewAgentManagerServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewAgentManagerConnectReader(server.URL, server.Client())
}

func TestAgentManagerConnectReaderHarvestsDeletionFilteredCaller(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	fixture := &agentManagerEventsFixture{pages: map[int64]*api.GetRunEventsResponse{
		-1: {Events: []*domain.RunEvent{
			runEvent("event-live", runID, 1, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "Where is retry handled?", ConversationId: "conversation-1"}}),
			runEvent("event-deleted", runID, 2, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "Remove this query", ConversationId: "conversation-1"}}),
			runEvent("delete-marker", runID, 3, &domain.RunEvent_MessageDeleted{MessageDeleted: &domain.MessageDeletedEventData{TargetEventId: "event-deleted"}}),
		}},
	}, runID: runID, status: domain.RunStatus_RUN_STATUS_COMPLETE}
	reader := serveAgentManagerEventFixture(t, fixture)
	got, err := HarvestAgentManagerCallers(context.Background(), reader, []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].EventID != "event-live" || got[0].ConversationID != "conversation-1" || got[0].MessageRole != "user" || got[0].MessageSource != "operator" || fixture.calls != 1 {
		t.Fatalf("Connect harvest = %+v calls=%d; want only the live identified operator message", got, fixture.calls)
	}
}

func TestAgentManagerCodeTaskSourceVerifierRequiresCompletedExactOperatorEvent(t *testing.T) {
	candidate := codeTaskCandidateFixture(930, "single-file-change")
	binding := CodeTaskSourceIdentityBinding{Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository}
	want := AgentManagerCodeTaskSourceIdentityBinding{Purpose: "local-coding-benchmark-fixture/v1", RunID: candidate.Source.RunID, EventID: candidate.Source.EventID, Sequence: candidate.Source.Sequence, ConversationID: candidate.Source.ConversationID, MessageSHA256: candidate.Source.MessageSHA256, TaskSpecSHA256: candidate.TaskSpecSHA256, RepositoryID: candidate.Repository.RepositoryID, RepositoryRevision: candidate.Repository.Revision, RepositoryTreeSHA256: candidate.Repository.TreeSHA256}
	reader := &sourceIdentityReceiptFake{receipt: AgentManagerCodeTaskSourceIdentityReceipt{Binding: want, ActorSubjectSHA256: candidate.Source.ActorSubjectSHA256, ActorSubjectDigestVersion: candidate.Source.ActorSubjectDigestVersion, ActorIdentityReceiptID: candidate.Source.ActorIdentityReceiptID, ActorIdentityReceiptSHA256: candidate.Source.ActorIdentityReceiptSHA256}}
	verifier := AgentManagerCodeTaskSourceVerifier{Receipts: reader}
	if err := verifier.VerifyCodeTaskSource(context.Background(), binding); err != nil {
		t.Fatalf("completed owner-bound source rejected: %v", err)
	}
	wrong := reader.receipt
	wrong.Binding.RepositoryRevision = strings.Repeat("f", 40)
	reader.receipt = wrong
	if err := verifier.VerifyCodeTaskSource(context.Background(), binding); err == nil {
		t.Fatal("source verifier accepted a receipt for a different repository revision")
	}
}

func TestAgentManagerConnectReaderFollowsHasMoreCursorIncludingSequenceZero(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174001"
	fixture := &agentManagerEventsFixture{pages: map[int64]*api.GetRunEventsResponse{
		-1: {Events: []*domain.RunEvent{runEvent("event-1", runID, 0, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "first query", ConversationId: "conversation-1"}})}, HasMore: true},
		0:  {Events: []*domain.RunEvent{runEvent("event-2", runID, 1, &domain.RunEvent_Message{Message: &domain.MessageEventData{Role: "user", Source: "operator", Content: "second query", ConversationId: "conversation-1"}})}, HasMore: false},
	}, runID: runID, status: domain.RunStatus_RUN_STATUS_COMPLETE}
	reader := serveAgentManagerEventFixture(t, fixture)
	got, err := HarvestAgentManagerCallers(context.Background(), reader, []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if fixture.calls != 2 || len(got) != 2 || got[0].Sequence != 0 || got[1].Sequence != 1 {
		t.Fatalf("pagination calls=%d, callers=%+v; want sequence zero followed by one across two pages", fixture.calls, got)
	}
}

func TestAuditedAgentManagerCallerCaseRetainsConversationRoleAndReviewGate(t *testing.T) {
	record := AgentManagerCallerRecord{RunID: "run-1", EventID: "event-1", Sequence: 0, ConversationID: "conversation-1", MessageRole: "user", MessageSource: "operator", Query: "where is retry handled"}
	route := gatewayRouteFixture()
	route.SourceSequence = 0
	route.GatewayRequestID, _ = searcheval.AgentManagerReplayRequestID(record.RunID, record.EventID, int64(record.Sequence), route.TaskRole, route.Consumer)
	candidate, err := AuditedAgentManagerCallerCase(record, route, CallerLabelReview{})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.GetStatus() != "candidate" || len(candidate.GetExpectIds()) != 0 || !strings.Contains(candidate.GetNote(), "source=agent-manager://runs/run-1/events/event-1 seq=0") || !strings.Contains(candidate.GetNote(), "conversation=agent-manager://conversations/conversation-1") || !strings.Contains(candidate.GetNote(), "message_role=user") {
		t.Fatalf("candidate lost source identity or entered acceptance: %+v", candidate)
	}
	reviewed, err := AuditedAgentManagerCallerCase(record, route, CallerLabelReview{Reviewer: "independent", EvidenceURI: "audit://labels/1", ExpectedIDs: []string{"src/retry.go"}, IndependentlyValidated: true})
	if err != nil {
		t.Fatal(err)
	}
	counts, err := ValidateModelRoleCoverage([]*evalv1.EvalCase{candidate, reviewed}, []string{"chat.default", "code.local"}, 1)
	if err == nil || counts["chat.default"] != 1 {
		t.Fatalf("reviewed role coverage = %v, %v; want chat.default counted and other roles reported deficient", counts, err)
	}
}

func TestAuditedAgentManagerCallerCaseRejectsUnboundGatewayRequest(t *testing.T) {
	record := AgentManagerCallerRecord{RunID: "run-1", EventID: "event-1", Sequence: 5, ConversationID: "conversation-1", MessageRole: "user", MessageSource: "operator", Query: "private text omitted"}
	route := gatewayRouteFixture()
	route.SourceSequence = record.Sequence
	route.GatewayRequestID = "gateway-request-without-source-binding"
	if _, err := AuditedAgentManagerCallerCase(record, route, CallerLabelReview{}); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("unbound route accepted, error = %v", err)
	}
}
