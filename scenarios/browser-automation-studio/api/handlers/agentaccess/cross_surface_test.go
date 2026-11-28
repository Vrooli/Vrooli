package agentaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	access "github.com/vrooli/browser-automation-studio/automation/agentaccess"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	agentaccessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess"
	agentaccessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess/agentaccessv1connect"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

const (
	crossIdentity = "signed-run-token"
	crossGrant    = "browser-grant-run-1"
	crossRunID    = "00000000-0000-0000-0000-000000000001"
	crossSession  = "managed-session-cross-surface"
)

// This composes the BAS Connect mount used by Program Runtime and BAS REST
// routes used by the CLI over one real Broker instance. Only Agent Manager and
// the browser driver are fake HTTP owners; no browser process is started.
func TestCLIAndProgramRuntimeShareOneGrantedBrokerSession(t *testing.T) {
	var control atomic.Bool
	control.Store(true)
	var effects atomic.Int32
	var driverCloses atomic.Int32
	var driverURL atomic.Value

	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotGrant := r.Header.Get("X-Vrooli-Browser-Grant-Receipt")
		if r.Header.Get("X-Agent-Identity-Token") != crossIdentity || (gotGrant != crossGrant && !(r.URL.Path == "/api/v1/browser/resolve-context" && gotGrant == "")) {
			// Program Runtime resolves the opaque receipt using verified identity
			// before making the downstream Connect call; this initial owner lookup
			// intentionally has no receipt header.
			if r.URL.Path != "/api/v1/browser/resolve-context" || r.Header.Get("X-Agent-Identity-Token") != crossIdentity || gotGrant != "" {
				http.Error(w, "wrong run authority", http.StatusUnauthorized)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/browser/resolve-context":
			_, _ = fmt.Fprintf(w, `{"caller_id":%q,"run_id":%q,"grant_receipt":%q,"capability":{"enabled":true,"grantReceipt":%q,"mode":"managed","permissions":{"profileId":"profile-1","profileMode":"fresh"}}}`, "run:"+crossRunID, crossRunID, crossGrant, crossGrant)
		case "/api/v1/browser/resolve-caller":
			_, _ = fmt.Fprintf(w, `{"caller_id":%q,"grant_receipt":%q}`, "run:"+crossRunID, crossGrant)
		case "/api/v1/browser/resolve-profile":
			var request struct {
				ProfileRef string `json:"profile_ref"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.ProfileRef != "profile-1" {
				http.Error(w, "wrong profile", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "active", "run_id": crossRunID, "grant_receipt": crossGrant, "mode": "managed",
				"permission_profile":         map[string]any{"profileId": "profile-1", "profileMode": "fresh", "level": "act", "allowedOrigins": []string{"https://example.test"}, "allowHistory": true},
				"session_permission_mapping": map[string]any{"view": true, "control": control.Load(), "profile_mode": "fresh"},
				"budget":                     map[string]any{"sessionMinutes": 5, "actions": 20}, "unsupported_sessionpermission_fields": []string{},
			})
		case "/api/v1/browser/authorize-open":
			var request struct {
				ProfileID  string `json:"profile_id"`
				InitialURL string `json:"initial_url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.ProfileID != "profile-1" || request.InitialURL != "https://example.test/start" {
				http.Error(w, "open authorization mismatch", http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"authorized": true, "grant_receipt": crossGrant})
		default:
			http.NotFound(w, r)
		}
	}))
	defer am.Close()

	driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/session/start":
			_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{SessionID: crossSession, LeaseID: "lease-1", ActivePageID: "page-1"})
		case strings.HasSuffix(r.URL.Path, "/record/navigate"):
			var request struct {
				URL string `json:"url"`
			}
			_ = json.NewDecoder(r.Body).Decode(&request)
			driverURL.Store(request.URL)
			_ = json.NewEncoder(w).Encode(driver.NavigateResponse{DriverPageID: "page-1", URL: request.URL, Title: "Example"})
		case strings.HasSuffix(r.URL.Path, "/record/navigation-state"):
			_ = json.NewEncoder(w).Encode(driver.NavigationStateResponse{SessionID: crossSession, URL: driverURL.Load().(string), Title: "Example"})
		case strings.HasSuffix(r.URL.Path, "/record/screenshot"):
			_ = json.NewEncoder(w).Encode(driver.CaptureScreenshotResponse{MediaType: "image/png", Width: 1, Height: 1, CapturedAt: "2026-11-19T00:00:00Z"})
		case strings.HasSuffix(r.URL.Path, "/run"):
			effects.Add(1)
			_ = json.NewEncoder(w).Encode(contracts.StepOutcome{Success: true, StepType: "click"})
		case strings.HasSuffix(r.URL.Path, "/close"):
			driverCloses.Add(1)
			_ = json.NewEncoder(w).Encode(driver.CloseSessionResponse{Success: true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer driverServer.Close()

	driverClient, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManagerWithClient(driverClient)
	runContext := &access.AgentManagerRunContext{BaseURL: am.URL, Client: am.Client()}
	openAuthorizer := &access.AgentManagerOpenAuthorizer{BaseURL: am.URL, Client: am.Client(), Context: runContext}
	broker, err := access.NewBroker(manager, &access.AgentManagerProfileResolver{RunContext: runContext}, openAuthorizer, runContext)
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(runContext)
	broker.SetEvidenceRecorder(crossSurfaceEvidenceRecorder{})
	broker.SetCapabilityReporter(crossSurfaceCapabilities{{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "fake-owner", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl, sessionpermission.CapabilityHistoryRead}}})

	// The two transports are deliberately mounted over this same broker.
	workflows := &crossSurfaceWorkflowCatalog{}
	connectMount := Module(Deps{Broker: broker, Workflows: workflows})
	api := chi.NewRouter()
	api.Route("/api/v1", func(r chi.Router) { (&access.HTTPHandler{Broker: broker, Workflows: workflows}).RegisterRoutes(r) })
	mux := http.NewServeMux()
	mux.Handle(connectMount.Path, connectMount.Handler)
	mux.Handle("/", api)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := agentaccessconnect.NewBrowserSessionServiceClient(server.Client(), server.URL, connect.WithProtoJSON())

	// CLI opens through REST with run-scoped credentials supplied out of band.
	openBody := `{"permission_ref":"profile-1","backend":"managed","start_url":"https://example.test/start","reuse_mode":"fresh"}`
	openResponse := crossSurfaceREST(t, server.Client(), server.URL+"/api/v1/browser/sessions", http.MethodPost, openBody, crossIdentity, crossGrant, http.StatusOK)
	var opened struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(openResponse, &opened); err != nil || opened.SessionID != crossSession {
		t.Fatalf("CLI open response=%s err=%v", openResponse, err)
	}

	// The PR resolver produces the same grant from the verified run identity;
	// the Connect binding then observes the CLI-opened session.
	resolved := crossSurfaceREST(t, server.Client(), am.URL+"/api/v1/browser/resolve-context", http.MethodPost, "{}", crossIdentity, "", http.StatusOK)
	var grant struct {
		GrantReceipt string `json:"grant_receipt"`
	}
	if err := json.Unmarshal(resolved, &grant); err != nil || grant.GrantReceipt != crossGrant {
		t.Fatalf("PR context=%s err=%v", resolved, err)
	}
	observeReq := connect.NewRequest(&agentaccessv1.ObserveRequest{SessionId: opened.SessionID})
	crossSurfaceConnectHeaders(observeReq.Header(), crossIdentity, grant.GrantReceipt)
	observed, err := client.Observe(context.Background(), observeReq)
	if err != nil || observed.Msg.GetSessionId() != opened.SessionID || observed.Msg.GetUrl() != "https://example.test/start" {
		t.Fatalf("PR observe response=%+v err=%v", observed, err)
	}

	// The CLI's typed task stays on that same lease. It is followed by another
	// PR observation, so the proof spans both public transport paths.
	taskBody := `{"operation":"click","selector":"button#continue"}`
	crossSurfaceREST(t, server.Client(), server.URL+"/api/v1/browser/sessions/"+opened.SessionID+"/task", http.MethodPost, taskBody, crossIdentity, crossGrant, http.StatusOK)
	observeReq = connect.NewRequest(&agentaccessv1.ObserveRequest{SessionId: opened.SessionID})
	crossSurfaceConnectHeaders(observeReq.Header(), crossIdentity, grant.GrantReceipt)
	if _, err := client.Observe(context.Background(), observeReq); err != nil {
		t.Fatalf("PR observe after CLI action: %v", err)
	}
	if effects.Load() != 1 {
		t.Fatalf("driver effects=%d, want one CLI action", effects.Load())
	}

	traceReq := connect.NewRequest(&agentaccessv1.ReadTraceRequest{SessionId: opened.SessionID})
	crossSurfaceConnectHeaders(traceReq.Header(), crossIdentity, grant.GrantReceipt)
	trace, err := client.ReadTrace(context.Background(), traceReq)
	if err != nil {
		t.Fatalf("shared trace: %v", err)
	}
	if trace.Msg.GetSessionId() != opened.SessionID || len(trace.Msg.GetEvents()) < 3 {
		t.Fatalf("cross-surface trace=%+v", trace.Msg)
	}
	for index, event := range trace.Msg.GetEvents() {
		if event.GetSequence() != uint64(index+1) {
			t.Fatalf("trace[%d]=%+v is not ordered", index, event)
		}
	}

	// Simulate a page redirected out of the allowed origin. Both observation
	// and the next action must fail before another driver action is dispatched.
	driverURL.Store("https://blocked.test/redirect")
	redirectedObserve := connect.NewRequest(&agentaccessv1.ObserveRequest{SessionId: opened.SessionID})
	crossSurfaceConnectHeaders(redirectedObserve.Header(), crossIdentity, grant.GrantReceipt)
	if _, err := client.Observe(context.Background(), redirectedObserve); err == nil {
		t.Fatal("Program Runtime observed a page after it redirected out of scope")
	}
	redirectedAct := connect.NewRequest(&agentaccessv1.ActRequest{SessionId: opened.SessionID, Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK}})
	crossSurfaceConnectHeaders(redirectedAct.Header(), crossIdentity, grant.GrantReceipt)
	if _, err := client.Act(context.Background(), redirectedAct); err == nil {
		t.Fatal("out-of-scope page allowed a Program Runtime action")
	}
	if effects.Load() != 1 {
		t.Fatalf("redirected action reached driver; effects=%d", effects.Load())
	}

	// Revoke action permission at the owner; the next PR write is denied before
	// reaching the driver. The same authenticated owner can still close/release
	// its lease after the grant narrows.
	driverURL.Store("https://example.test/start")
	control.Store(false)
	actReq := connect.NewRequest(&agentaccessv1.ActRequest{SessionId: opened.SessionID, Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK}})
	crossSurfaceConnectHeaders(actReq.Header(), crossIdentity, grant.GrantReceipt)
	if _, err := client.Act(context.Background(), actReq); err == nil {
		t.Fatal("revoked control still allowed a Program Runtime action")
	}
	if effects.Load() != 1 {
		t.Fatalf("revoked action reached driver; effects=%d", effects.Load())
	}
	crossSurfaceREST(t, server.Client(), server.URL+"/api/v1/browser/sessions/"+opened.SessionID+"/close", http.MethodPost, "{}", crossIdentity, crossGrant, http.StatusNoContent)
	if driverCloses.Load() != 1 {
		t.Fatalf("CLI close reached the driver %d times, want one", driverCloses.Load())
	}
	staleReq := connect.NewRequest(&agentaccessv1.ObserveRequest{SessionId: opened.SessionID})
	crossSurfaceConnectHeaders(staleReq.Header(), crossIdentity, grant.GrantReceipt)
	if _, err := client.Observe(context.Background(), staleReq); err == nil {
		t.Fatal("closed CLI session remained usable through Program Runtime")
	}
	closeReq := connect.NewRequest(&agentaccessv1.CloseSessionRequest{SessionId: opened.SessionID})
	crossSurfaceConnectHeaders(closeReq.Header(), crossIdentity, grant.GrantReceipt)
	if _, err := client.CloseSession(context.Background(), closeReq); err != nil {
		t.Fatalf("repeat close should be idempotent for the same run: %v", err)
	}
	if driverCloses.Load() != 1 {
		t.Fatalf("repeat close reached the driver again; closes=%d", driverCloses.Load())
	}
}

func crossSurfaceConnectHeaders(header http.Header, identity, grant string) {
	header.Set("X-Agent-Identity-Token", identity)
	header.Set("X-Vrooli-Browser-Grant-Receipt", grant)
}

func crossSurfaceREST(t *testing.T, client *http.Client, endpoint, method, body, identity, grant string, wantStatus int) []byte {
	t.Helper()
	req, err := http.NewRequest(method, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Identity-Token", identity)
	if grant != "" {
		req.Header.Set("X-Vrooli-Browser-Grant-Receipt", grant)
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var data json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil && response.StatusCode != http.StatusNoContent {
		t.Fatalf("decode %s response: %v", endpoint, err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d body=%s, want %d", method, endpoint, response.StatusCode, data, wantStatus)
	}
	return data
}

type crossSurfaceCapabilities []sessionpermission.BackendReport

func (c crossSurfaceCapabilities) BackendReports(context.Context) ([]sessionpermission.BackendReport, error) {
	return c, nil
}

type crossSurfaceEvidenceRecorder struct{}

func (crossSurfaceEvidenceRecorder) RecordBrowserSession(context.Context, string, sessionpermission.Backend, string, string) error {
	return nil
}

func (crossSurfaceEvidenceRecorder) RecordBrowserAction(context.Context, string, sessionpermission.Backend, string, string, string) error {
	return nil
}

type crossSurfaceWorkflowCatalog struct{}

func (crossSurfaceWorkflowCatalog) CreateWorkflow(context.Context, *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	return nil, fmt.Errorf("not used in this test")
}
func (crossSurfaceWorkflowCatalog) GetWorkflowAPI(context.Context, *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	return nil, fmt.Errorf("not used in this test")
}
