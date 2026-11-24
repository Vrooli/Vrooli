package agentaccess

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	automationaccess "github.com/vrooli/browser-automation-studio/automation/agentaccess"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	agentaccessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess"
	agentaccessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess/agentaccessv1connect"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

type brokerFixture struct {
	openRequest    automationaccess.OpenRequest
	openInfo       automationaccess.SessionInfo
	attachSession  string
	attachRequest  automationaccess.AttachRequest
	observeSession string
	observeRequest automationaccess.ObserveRequest
	actSession     string
	actRequest     automationaccess.ActRequest
	actRequests    []automationaccess.ActRequest
	traceSession   string
	trace          automationaccess.Trace
	promoteSession string
	promoteRequest automationaccess.PromoteRequest
	replaySession  string
	closeSession   string
	calls          []string
}

func (b *brokerFixture) Open(_ context.Context, req automationaccess.OpenRequest) (automationaccess.SessionInfo, error) {
	b.calls = append(b.calls, "open")
	b.openRequest = req
	if b.openInfo.SessionID != "" {
		return b.openInfo, nil
	}
	return automationaccess.SessionInfo{SessionID: "session-1", PermissionRef: req.PermissionRef, Mode: "managed", Capabilities: []string{"observe"}}, nil
}
func (b *brokerFixture) Attach(_ context.Context, req automationaccess.AttachRequest) (automationaccess.SessionInfo, error) {
	b.calls = append(b.calls, "attach")
	b.attachSession, b.attachRequest = req.SessionID, req
	return automationaccess.SessionInfo{SessionID: req.SessionID, PermissionRef: req.PermissionRef, Mode: "managed", Capabilities: []string{"observe"}}, nil
}
func (b *brokerFixture) Observe(_ context.Context, sessionID string, req automationaccess.ObserveRequest) (automationaccess.Observation, error) {
	b.calls = append(b.calls, "observe")
	b.observeSession, b.observeRequest = sessionID, req
	return automationaccess.Observation{SessionID: sessionID, ObservedAt: time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC), URL: "https://example.test", Title: "Example", Text: "hello", Screenshot: nil}, nil
}
func (b *brokerFixture) Act(_ context.Context, sessionID string, req automationaccess.ActRequest) (automationaccess.ActionResult, error) {
	b.calls = append(b.calls, "act")
	b.actSession, b.actRequest = sessionID, req
	b.actRequests = append(b.actRequests, req)
	return automationaccess.ActionResult{SessionID: sessionID, Outcome: contracts.StepOutcome{Success: true, StepType: "click"}}, nil
}
func (b *brokerFixture) Trace(_ context.Context, sessionID string) (automationaccess.Trace, error) {
	b.calls = append(b.calls, "trace")
	b.traceSession = sessionID
	return b.trace, nil
}
func (b *brokerFixture) PromoteTrace(_ context.Context, sessionID string, req automationaccess.PromoteRequest) (automationaccess.PromotedFlow, error) {
	b.calls = append(b.calls, "promote")
	b.promoteSession, b.promoteRequest = sessionID, req
	return automationaccess.PromotedFlow{TraceID: "trace-1", SessionID: sessionID, ActionCount: 1, Definition: &workflowsv1.WorkflowDefinitionV2{Metadata: &workflowsv1.WorkflowMetadataV2{Name: proto.String(req.Name), Labels: map[string]string{"source": "agent-browser-trace"}}}}, nil
}
func (b *brokerFixture) DraftTrace(_ context.Context, sessionID string, req automationaccess.PromoteRequest) (automationaccess.PromotedFlow, error) {
	b.calls = append(b.calls, "draft")
	b.promoteSession, b.promoteRequest = sessionID, req
	return automationaccess.PromotedFlow{DraftDigest: "sha256:fixture", TraceID: "trace-1", SessionID: sessionID, ActionCount: 1, Definition: &workflowsv1.WorkflowDefinitionV2{Metadata: &workflowsv1.WorkflowMetadataV2{Name: proto.String(req.Name)}}}, nil
}
func (b *brokerFixture) ReplayPromotedFlow(_ context.Context, sessionID string, _ *workflowsv1.WorkflowDefinitionV2, _ map[string]string) (automationaccess.ReplayResult, error) {
	b.calls = append(b.calls, "replay")
	b.replaySession = sessionID
	return automationaccess.ReplayResult{SessionID: sessionID, Completed: true}, nil
}
func (b *brokerFixture) Close(_ context.Context, sessionID string) error {
	b.calls = append(b.calls, "close")
	b.closeSession = sessionID
	return nil
}

func testClient(t *testing.T, broker SessionBroker) (agentaccessconnect.BrowserSessionServiceClient, func()) {
	t.Helper()
	mount := Module(Deps{Broker: broker, Workflows: &workflowCatalogFixture{}, ProjectScopes: allowProjectScope{}})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	server := httptest.NewServer(mux)
	client := agentaccessconnect.NewBrowserSessionServiceClient(server.Client(), server.URL, connect.WithProtoJSON())
	return client, server.Close
}

type allowProjectScope struct{}

func (allowProjectScope) AuthorizeProjectRead(context.Context, string) error  { return nil }
func (allowProjectScope) AuthorizeProjectWrite(context.Context, string) error { return nil }

type projectScopeFixture struct {
	projectID     string
	readProjectID string
	err           error
}

func (p *projectScopeFixture) AuthorizeProjectWrite(_ context.Context, projectID string) error {
	p.projectID = projectID
	return p.err
}

func (p *projectScopeFixture) AuthorizeProjectRead(_ context.Context, projectID string) error {
	p.readProjectID = projectID
	return p.err
}

type workflowCatalogFixture struct {
	createRequest  *basapi.CreateWorkflowRequest
	createResponse *basapi.CreateWorkflowResponse
	getRequest     *basapi.GetWorkflowRequest
	getResponse    *basapi.GetWorkflowResponse
	createErr      error
	getErr         error
}

func (w *workflowCatalogFixture) CreateWorkflow(_ context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	w.createRequest = req
	if w.createErr != nil {
		return nil, w.createErr
	}
	if w.createResponse != nil {
		return w.createResponse, nil
	}
	return &basapi.CreateWorkflowResponse{Workflow: &basapi.WorkflowSummary{Id: "workflow-1", Version: 1, FlowDefinition: req.GetFlowDefinition()}}, nil
}
func (w *workflowCatalogFixture) GetWorkflowAPI(_ context.Context, req *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	w.getRequest = req
	if w.getErr != nil {
		return nil, w.getErr
	}
	if w.getResponse != nil {
		return w.getResponse, nil
	}
	return &basapi.GetWorkflowResponse{Workflow: &basapi.WorkflowSummary{Id: req.GetWorkflowId(), ProjectId: "project-1", Version: 1, FlowDefinition: &workflowsv1.WorkflowDefinitionV2{Metadata: &workflowsv1.WorkflowMetadataV2{Labels: map[string]string{"source": "agent-browser-trace"}}}}}, nil
}

func authorizedRequest[T any](msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set(identityHeader, "signed-agent-token")
	req.Header().Set(grantHeader, "opaque-browser-grant")
	return req
}

func TestBrowserConnectRequiresExactlyOneProofPair(t *testing.T) {
	for name, headers := range map[string]http.Header{
		"missing":         {},
		"missing-grant":   {identityHeader: []string{"signed-agent-token"}},
		"duplicate-token": {identityHeader: []string{"token-a", "token-b"}, grantHeader: []string{"opaque-browser-grant"}},
		"duplicate-grant": {identityHeader: []string{"signed-agent-token"}, grantHeader: []string{"grant-a", "grant-b"}},
	} {
		t.Run(name, func(t *testing.T) {
			broker := &brokerFixture{}
			client, closeServer := testClient(t, broker)
			defer closeServer()
			req := connect.NewRequest(&agentaccessv1.OpenSessionRequest{PermissionRef: "profile-1"})
			for key, values := range headers {
				for _, value := range values {
					req.Header().Add(key, value)
				}
			}
			_, err := client.OpenSession(context.Background(), req)
			if connect.CodeOf(err) != connect.CodeUnauthenticated {
				t.Fatalf("error=%v code=%s, want unauthenticated", err, connect.CodeOf(err))
			}
			if len(broker.calls) != 0 {
				t.Fatalf("broker calls=%v, want none", broker.calls)
			}
		})
	}
}

func TestOpenSessionMapsOnlyPermissionedOpenFields(t *testing.T) {
	broker := &brokerFixture{openInfo: automationaccess.SessionInfo{
		SessionID: "session-1", PermissionRef: "profile-1", Mode: "managed", Capabilities: []string{"observe"},
		IdentityProfile: &sessionpermission.BrowserIdentityProfile{Preset: "balanced", ViewportWidth: 1440, ViewportHeight: 900, Locale: "en-GB", TimezoneID: "Europe/London", ColorScheme: "dark"},
		ActualViewport:  &driver.ActualViewport{Width: 1440, Height: 900, Source: driver.ViewportSourceFingerprint, Reason: "identity profile"},
		IdentityEvidence: &driver.IdentityEvidence{
			Engine: "chromium", EngineVersion: "140.0.7339.80", EngineSource: "playwright.browser",
			UserAgent: driver.IdentityDimensionEvidence{Configured: "Chrome/120.0.0.0", Observed: "Chrome/120.0.0.0", Status: "mismatch", Source: "playwright.page.about_blank", Reason: "UA major does not match serving engine"},
			Locale:    driver.IdentityDimensionEvidence{Configured: "en-GB", Observed: "en-GB", Effective: "en-GB", Status: "effective", Source: "playwright.page.about_blank"},
		},
	}}
	client, closeServer := testClient(t, broker)
	defer closeServer()
	response, err := client.OpenSession(context.Background(), authorizedRequest(&agentaccessv1.OpenSessionRequest{
		PermissionRef: "profile-1", StartUrl: "https://example.test", Backend: "managed", ReuseMode: "fresh",
		ViewportWidth: 1200, ViewportHeight: 800, ConsentReceipt: "consent-1", TabId: "tab-1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := automationaccess.OpenRequest{PermissionRef: "profile-1", StartURL: "https://example.test", Backend: "managed", ReuseMode: "fresh", ViewportWidth: 1200, ViewportHeight: 800, ConsentReceipt: "consent-1", TabID: "tab-1"}
	if broker.openRequest != want {
		t.Fatalf("open request=%+v, want %+v", broker.openRequest, want)
	}
	if broker.openRequest.GrantReceipt != "" {
		t.Fatal("caller request populated a trusted grant receipt")
	}
	if response.Msg.GetPermissionRef() != "" {
		t.Fatalf("Connect open exposed trusted permission profile reference %q", response.Msg.GetPermissionRef())
	}
	if response.Msg.IdentityProfile == nil || response.Msg.IdentityProfile.Preset != "balanced" || response.Msg.IdentityProfile.Locale != "en-GB" || response.Msg.ActualViewport == nil || response.Msg.ActualViewport.Width != 1440 || response.Msg.ActualViewport.Source != string(driver.ViewportSourceFingerprint) {
		t.Fatalf("Connect response omitted configured identity or owner-observed viewport: %#v", response.Msg)
	}
	if evidence := response.Msg.IdentityEvidence; evidence == nil || evidence.Engine != "chromium" || evidence.EngineVersion != "140.0.7339.80" || evidence.UserAgent == nil || evidence.UserAgent.Status != "mismatch" || evidence.UserAgent.Effective != nil || evidence.Locale == nil || evidence.Locale.Status != "effective" || evidence.Locale.GetEffective().GetStringValue() != "en-GB" {
		t.Fatalf("Connect response misrepresented owner identity evidence: %#v", response.Msg.IdentityEvidence)
	}
}

func TestProofHeadersEnterOnlyTheBrokerTrustedContext(t *testing.T) {
	var receivedIdentity, receivedGrant string
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedIdentity = r.Header.Get(identityHeader)
		receivedGrant = r.Header.Get(grantHeader)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/browser/resolve-context" {
			t.Errorf("Agent Manager request=%s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"caller_id": "run:fixture", "run_id": "fixture-run", "grant_receipt": receivedGrant, "capability": map[string]any{"permissions": map[string]string{"profileId": "profile-1"}}})
	}))
	defer owner.Close()
	broker := &contextProbeBroker{brokerFixture: &brokerFixture{}, runContext: &automationaccess.AgentManagerRunContext{BaseURL: owner.URL, Client: owner.Client()}}
	client, closeServer := testClient(t, broker)
	defer closeServer()
	_, err := client.OpenSession(context.Background(), authorizedRequest(&agentaccessv1.OpenSessionRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if broker.openRequest.PermissionRef != "" {
		t.Fatalf("empty request selected permission profile %q before trusted Broker resolution", broker.openRequest.PermissionRef)
	}
	if receivedIdentity != "signed-agent-token" || receivedGrant != "opaque-browser-grant" {
		t.Fatalf("broker trusted context forwarded identity=%q grant=%q", receivedIdentity, receivedGrant)
	}
}

func TestOpenSessionWithoutAMProofCannotCreateSessionWhenProfileIsOmitted(t *testing.T) {
	broker := &brokerFixture{}
	client, closeServer := testClient(t, broker)
	defer closeServer()
	_, err := client.OpenSession(context.Background(), connect.NewRequest(&agentaccessv1.OpenSessionRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("open without AM proof error=%v, want unauthenticated", err)
	}
	if len(broker.calls) != 0 {
		t.Fatalf("broker calls=%v, want no session creation", broker.calls)
	}
}

func TestObserveActTracePromotionAndCloseMapToBroker(t *testing.T) {
	action := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK}
	assertion := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT}
	broker := &brokerFixture{trace: automationaccess.Trace{
		ID: "trace-1", SessionID: "session-1", PermissionRef: "profile-1",
		Events: []automationaccess.TraceEvent{{Sequence: 2, Kind: "action", OccurredAt: time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC), Instruction: &contracts.CompiledInstruction{Action: action}, Outcome: &contracts.StepOutcome{Success: true}, Parameters: []automationaccess.Parameter{{Name: "query", Kind: "text"}}}},
	}}
	client, closeServer := testClient(t, broker)
	defer closeServer()

	observed, err := client.Observe(context.Background(), authorizedRequest(&agentaccessv1.ObserveRequest{SessionId: "session-1", PageId: "page-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if broker.observeSession != "session-1" || broker.observeRequest.PageID != "page-1" || observed.Msg.GetTitle() != "Example" {
		t.Fatalf("observe mapping: broker=%+v response=%+v", broker.observeRequest, observed.Msg)
	}

	acted, err := client.Act(context.Background(), authorizedRequest(&agentaccessv1.ActRequest{SessionId: "session-1", InstructionIndex: 3, NodeId: "node-3", Action: action, TargetUrl: "https://example.test"}))
	if err != nil {
		t.Fatal(err)
	}
	if broker.actSession != "session-1" || broker.actRequest.Instruction.Index != 3 || broker.actRequest.Instruction.NodeID != "node-3" || !proto.Equal(broker.actRequest.Instruction.Action, action) || !acted.Msg.GetSuccess() {
		t.Fatalf("act mapping: broker=%+v response=%+v", broker.actRequest, acted.Msg)
	}

	trace, err := client.ReadTrace(context.Background(), authorizedRequest(&agentaccessv1.ReadTraceRequest{SessionId: "session-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if broker.traceSession != "session-1" || trace.Msg.GetTraceId() != "trace-1" || trace.Msg.GetPermissionRef() != "" || len(trace.Msg.GetEvents()) != 1 || trace.Msg.Events[0].GetInstruction().Type != action.Type || !trace.Msg.Events[0].GetActionSucceeded() {
		t.Fatalf("trace mapping: response=%+v", trace.Msg)
	}
	draft, err := client.DraftTrace(context.Background(), authorizedRequest(&agentaccessv1.DraftTraceRequest{SessionId: "session-1", Name: "preview", Assertions: []*actionsv1.ActionDefinition{assertion}}))
	if err != nil {
		t.Fatal(err)
	}
	if broker.promoteSession != "session-1" || broker.promoteRequest.Name != "preview" || draft.Msg.GetDraftDigest() != "sha256:fixture" {
		t.Fatalf("draft mapping persisted or lost broker output: broker=%+v response=%+v", broker.promoteRequest, draft.Msg)
	}

	promoted, err := client.PromoteTrace(context.Background(), authorizedRequest(&agentaccessv1.PromoteTraceRequest{SessionId: "session-1", ProjectId: "project-1", Name: "saved task", Assertions: []*actionsv1.ActionDefinition{assertion}}))
	if err != nil {
		t.Fatal(err)
	}
	if broker.promoteSession != "session-1" || broker.promoteRequest.Name != "saved task" || len(broker.promoteRequest.Assertions) != 1 || !proto.Equal(broker.promoteRequest.Assertions[0], assertion) || promoted.Msg.GetTraceId() != "trace-1" || promoted.Msg.GetWorkflowId() != "workflow-1" {
		t.Fatalf("promotion mapping: broker=%+v response=%+v", broker.promoteRequest, promoted.Msg)
	}

	closed, err := client.CloseSession(context.Background(), authorizedRequest(&agentaccessv1.CloseSessionRequest{SessionId: "session-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if broker.closeSession != "session-1" || !closed.Msg.GetClosed() {
		t.Fatalf("close mapping: broker session=%q response=%+v", broker.closeSession, closed.Msg)
	}
}

func TestAttachUsesTheExistingBrokerSession(t *testing.T) {
	broker := &brokerFixture{}
	client, closeServer := testClient(t, broker)
	defer closeServer()
	response, err := client.AttachSession(context.Background(), authorizedRequest(&agentaccessv1.AttachSessionRequest{
		SessionId: "session-existing", PermissionRef: "profile-1", TabId: "tab-1", ConsentReceipt: "consent-1", ApprovalReceipt: "approval-1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := automationaccess.AttachRequest{SessionID: "session-existing", PermissionRef: "profile-1", TabID: "tab-1", ConsentReceipt: "consent-1", ApprovalReceipt: "approval-1"}
	if broker.attachRequest != want || response.Msg.GetSessionId() != "session-existing" || response.Msg.GetPermissionRef() != "" {
		t.Fatalf("attach request=%+v response=%+v, want existing session and %+v", broker.attachRequest, response.Msg, want)
	}
	if len(broker.calls) != 1 || broker.calls[0] != "attach" {
		t.Fatalf("broker calls=%v, want attach only", broker.calls)
	}
}

func TestRunTaskRechecksActionsOnOneBrokerSession(t *testing.T) {
	first := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE}
	second := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK}
	broker := &brokerFixture{}
	client, closeServer := testClient(t, broker)
	defer closeServer()
	response, err := client.RunTask(context.Background(), authorizedRequest(&agentaccessv1.RunTaskRequest{
		SessionId: "session-existing",
		Instructions: []*agentaccessv1.RunTaskInstruction{
			{InstructionIndex: 4, NodeId: "navigate", Action: first, TargetUrl: "https://example.test"},
			{InstructionIndex: 5, NodeId: "click", Action: second},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !response.Msg.GetCompleted() || response.Msg.GetSessionId() != "session-existing" || len(response.Msg.GetResults()) != 2 {
		t.Fatalf("run-task response=%+v", response.Msg)
	}
	if len(broker.actRequests) != 2 || broker.actSession != "session-existing" {
		t.Fatalf("act calls=%d session=%q, want 2 actions on the existing session", len(broker.actRequests), broker.actSession)
	}
	if broker.actRequests[0].Instruction.Index != 4 || broker.actRequests[0].Instruction.NodeID != "navigate" || !proto.Equal(broker.actRequests[0].Instruction.Action, first) || broker.actRequests[0].TargetURL != "https://example.test" {
		t.Fatalf("first action mapping=%+v", broker.actRequests[0])
	}
	if broker.actRequests[1].Instruction.Index != 5 || broker.actRequests[1].Instruction.NodeID != "click" || !proto.Equal(broker.actRequests[1].Instruction.Action, second) {
		t.Fatalf("second action mapping=%+v", broker.actRequests[1])
	}
	if len(broker.calls) != 2 || broker.calls[0] != "act" || broker.calls[1] != "act" {
		t.Fatalf("broker calls=%v, run-task must only delegate to Broker.Act", broker.calls)
	}
}

func TestPromotePersistsIntoBASWorkflowCatalogAndReplayUsesBrokerSession(t *testing.T) {
	broker := &brokerFixture{}
	workflows := &workflowCatalogFixture{}
	scope := &projectScopeFixture{}
	svc := &service{broker: broker, workflows: workflows, projectScopes: scope}
	assertion := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT}
	promoted, err := svc.PromoteTrace(context.Background(), connect.NewRequest(&agentaccessv1.PromoteTraceRequest{
		SessionId: "session-1", ProjectId: "project-7", Name: "saved journey", Assertions: []*actionsv1.ActionDefinition{assertion},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if workflows.createRequest == nil || workflows.createRequest.GetProjectId() != "project-7" || workflows.createRequest.GetFolderPath() != "agent-promoted" || workflows.createRequest.GetName() != "saved journey" || promoted.Msg.GetWorkflowId() != "workflow-1" {
		t.Fatalf("durable workflow create request=%+v response=%+v", workflows.createRequest, promoted.Msg)
	}
	if workflows.createRequest.GetFlowDefinition() != promoted.Msg.GetDefinition() {
		t.Fatal("persisted BAS workflow definition diverged from promoted flow")
	}

	replayed, err := svc.ReplayWorkflow(context.Background(), connect.NewRequest(&agentaccessv1.ReplayWorkflowRequest{SessionId: "session-1", WorkflowId: "workflow-1", WorkflowVersion: 1, Parameters: map[string]string{"q": "search"}}))
	if err != nil {
		t.Fatal(err)
	}
	if workflows.getRequest == nil || workflows.getRequest.GetWorkflowId() != "workflow-1" || workflows.getRequest.GetVersion() != 1 || scope.readProjectID != "project-1" || broker.replaySession != "session-1" || !replayed.Msg.GetCompleted() || replayed.Msg.GetWorkflowId() != "workflow-1" || replayed.Msg.GetWorkflowVersion() != 1 {
		t.Fatalf("saved replay get=%+v scope-read=%q broker-session=%q response=%+v", workflows.getRequest, scope.readProjectID, broker.replaySession, replayed.Msg)
	}
}

func TestReplayFailsClosedWithoutProjectScopeBeforeWorkflowRead(t *testing.T) {
	broker := &brokerFixture{}
	workflows := &workflowCatalogFixture{}
	svc := &service{broker: broker, workflows: workflows}
	_, err := svc.ReplayWorkflow(context.Background(), connect.NewRequest(&agentaccessv1.ReplayWorkflowRequest{
		SessionId: "session-1", WorkflowId: "other-project-workflow", WorkflowVersion: 1,
	}))
	if err == nil {
		t.Fatal("replay without an owner-issued project-scope verifier should fail closed")
	}
	if workflows.getRequest != nil || broker.replaySession != "" {
		t.Fatalf("scope-less replay read workflow or acted: get=%+v replay-session=%q", workflows.getRequest, broker.replaySession)
	}
}

func TestReplayChecksExactResolvedProjectBeforeBrokerActions(t *testing.T) {
	broker := &brokerFixture{}
	workflows := &workflowCatalogFixture{}
	scope := &projectScopeFixture{err: errors.New("project receipt not authorized for read")}
	svc := &service{broker: broker, workflows: workflows, projectScopes: scope}
	_, err := svc.ReplayWorkflow(context.Background(), connect.NewRequest(&agentaccessv1.ReplayWorkflowRequest{
		SessionId: "session-1", WorkflowId: "workflow-1", WorkflowVersion: 1,
	}))
	if err == nil || scope.readProjectID != "project-1" {
		t.Fatalf("replay scope project=%q err=%v, expected exact resolved project denial", scope.readProjectID, err)
	}
	if broker.replaySession != "" {
		t.Fatalf("Broker acted before project scope approval: session=%q", broker.replaySession)
	}
}

func TestReplayRejectsSavedFlowWithoutOwningProject(t *testing.T) {
	broker := &brokerFixture{}
	workflows := &workflowCatalogFixture{getResponse: &basapi.GetWorkflowResponse{Workflow: &basapi.WorkflowSummary{
		Id: "workflow-1", FlowDefinition: &workflowsv1.WorkflowDefinitionV2{Metadata: &workflowsv1.WorkflowMetadataV2{Labels: map[string]string{"source": "agent-browser-trace"}}},
	}}}
	scope := &projectScopeFixture{}
	svc := &service{broker: broker, workflows: workflows, projectScopes: scope}
	_, err := svc.ReplayWorkflow(context.Background(), connect.NewRequest(&agentaccessv1.ReplayWorkflowRequest{
		SessionId: "session-1", WorkflowId: "workflow-1", WorkflowVersion: 1,
	}))
	if err == nil {
		t.Fatal("replay without persisted owning project should fail closed")
	}
	if scope.readProjectID != "" || broker.replaySession != "" {
		t.Fatalf("missing project reached scope/action: project=%q replay-session=%q", scope.readProjectID, broker.replaySession)
	}
}

func TestPromoteFailsClosedWithoutProjectScopeReceiptVerifier(t *testing.T) {
	broker := &brokerFixture{}
	workflows := &workflowCatalogFixture{}
	svc := &service{broker: broker, workflows: workflows}
	assertion := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT}
	_, err := svc.PromoteTrace(context.Background(), connect.NewRequest(&agentaccessv1.PromoteTraceRequest{
		SessionId: "session-1", ProjectId: "other-users-project", Name: "must not persist", Assertions: []*actionsv1.ActionDefinition{assertion},
	}))
	if err == nil {
		t.Fatal("promotion without an owner-issued project-scope verifier should fail closed")
	}
	if workflows.createRequest != nil {
		t.Fatalf("cross-user project write reached WorkflowService without a scope receipt: %+v", workflows.createRequest)
	}
}

func TestPromoteRejectsOwnerDeniedProjectScopeBeforeWorkflowWrite(t *testing.T) {
	broker := &brokerFixture{}
	workflows := &workflowCatalogFixture{}
	scope := &projectScopeFixture{err: errors.New("receipt scope mismatch")}
	svc := &service{broker: broker, workflows: workflows, projectScopes: scope}
	assertion := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT}
	_, err := svc.PromoteTrace(context.Background(), connect.NewRequest(&agentaccessv1.PromoteTraceRequest{
		SessionId: "session-1", ProjectId: "project-owned-by-other-run", Name: "must not persist", Assertions: []*actionsv1.ActionDefinition{assertion},
	}))
	if err == nil || scope.projectID != "project-owned-by-other-run" {
		t.Fatalf("scope verifier result=%q error=%v, expected exact project authorization denial", scope.projectID, err)
	}
	if workflows.createRequest != nil {
		t.Fatalf("WorkflowService write occurred after scope denial: %+v", workflows.createRequest)
	}
}

func TestObserveBoundsScreenshotAndPreservesTimestamp(t *testing.T) {
	broker := &brokerFixture{}
	brokerWithScreenshot := screenshotBroker{brokerFixture: broker, data: base64.StdEncoding.EncodeToString([]byte("png"))}
	client, closeServer := testClient(t, brokerWithScreenshot)
	defer closeServer()
	response, err := client.Observe(context.Background(), authorizedRequest(&agentaccessv1.ObserveRequest{SessionId: "session-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetScreenshotData() != base64.StdEncoding.EncodeToString([]byte("png")) || response.Msg.GetScreenshotMediaType() != "image/png" || response.Msg.GetScreenshotCapturedAt().AsTime().UTC().Format(time.RFC3339) != "2026-11-19T10:00:00Z" {
		t.Fatalf("screenshot mapping=%+v", response.Msg)
	}

	tooLarge := screenshotBroker{brokerFixture: &brokerFixture{}, data: base64.StdEncoding.EncodeToString(make([]byte, maxScreenshot+1))}
	client2, closeServer2 := testClient(t, tooLarge)
	defer closeServer2()
	_, err = client2.Observe(context.Background(), authorizedRequest(&agentaccessv1.ObserveRequest{SessionId: "session-1"}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("error=%v code=%s, want resource exhausted", err, connect.CodeOf(err))
	}
}

type screenshotBroker struct {
	*brokerFixture
	data string
}

func (b screenshotBroker) Observe(_ context.Context, sessionID string, req automationaccess.ObserveRequest) (automationaccess.Observation, error) {
	b.calls = append(b.calls, "observe")
	b.observeSession, b.observeRequest = sessionID, req
	return automationaccess.Observation{SessionID: sessionID, Screenshot: &driver.CaptureScreenshotResponse{Data: b.data, MediaType: "image/png", Width: 1, Height: 1, CapturedAt: "2026-11-19T10:00:00Z"}}, nil
}

func TestBrokerErrorIsFailedPrecondition(t *testing.T) {
	broker := &errorBroker{brokerFixture: &brokerFixture{}, err: errors.New("permission profile denied")}
	client, closeServer := testClient(t, broker)
	defer closeServer()
	_, err := client.OpenSession(context.Background(), authorizedRequest(&agentaccessv1.OpenSessionRequest{PermissionRef: "profile-1"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("error=%v code=%s, want failed precondition", err, connect.CodeOf(err))
	}
}

type errorBroker struct {
	*brokerFixture
	err error
}

func (b *errorBroker) Open(context.Context, automationaccess.OpenRequest) (automationaccess.SessionInfo, error) {
	return automationaccess.SessionInfo{}, b.err
}

type contextProbeBroker struct {
	*brokerFixture
	runContext *automationaccess.AgentManagerRunContext
}

func (b *contextProbeBroker) Open(ctx context.Context, req automationaccess.OpenRequest) (automationaccess.SessionInfo, error) {
	if _, err := b.runContext.ResolveBrowserOpen(ctx); err != nil {
		return automationaccess.SessionInfo{}, err
	}
	return b.brokerFixture.Open(ctx, req)
}
