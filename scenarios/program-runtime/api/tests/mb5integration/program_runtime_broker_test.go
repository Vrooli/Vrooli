package mb5integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"
	access "github.com/vrooli/browser-automation-studio/automation/agentaccess"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	bashandlers "github.com/vrooli/browser-automation-studio/handlers/agentaccess"
	basfixture "github.com/vrooli/browser-automation-studio/testsupport/agentaccess"
	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
	agentaccessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess"
	agentaccessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess/agentaccessv1connect"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	programsessions "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/sessions"
	programsessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/sessions/sessions_v1connect"

	programhandlers "program-runtime/handlers/bindings"
	programsessionhandlers "program-runtime/handlers/sessions"
	"program-runtime/internal/bindings"
	"program-runtime/internal/server"
	programsessionstore "program-runtime/internal/sessions"
)

const (
	identityToken = "test-run-identity"
	grantReceipt  = "test-browser-grant"
	runID         = "00000000-0000-0000-0000-000000000071"
	browserID     = "mb5-program-runtime-session"
)

type staticVerifier struct{}

func (staticVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	if token != identityToken {
		return &cliutil.VerifyResult{Valid: false}, nil
	}
	return &cliutil.VerifyResult{Valid: true, RunStatus: "running", Claims: &cliutil.VerifiedClaims{RunID: runID}}, nil
}

type fakeWorkflowCatalog struct{}

func (fakeWorkflowCatalog) CreateWorkflow(context.Context, *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	return nil, fmt.Errorf("unused")
}
func (fakeWorkflowCatalog) GetWorkflowAPI(context.Context, *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	return nil, fmt.Errorf("unused")
}

// TestProgramRuntimeServerExecutesOverTheCLIOpenedBrokerSession composes the
// production Program Runtime server/registry and its private execution route
// with the real BAS REST + Connect handlers over one Broker. Only Agent
// Manager and driver HTTP owners are fake; no browser process is started.
func TestProgramRuntimeMCPAndCLIShareOneBrokerSessionAndTrace(t *testing.T) {
	mcpCLI := buildVrooliBrowserCLI(t)
	var control atomic.Bool
	control.Store(true)
	var fullRevoke atomic.Bool
	var amCalls atomic.Int32
	var effects atomic.Int32
	var starts atomic.Int32
	var closes atomic.Int32
	var currentURL atomic.Value
	currentURL.Store("https://example.test/start")

	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		amCalls.Add(1)
		if r.Header.Get("X-Agent-Identity-Token") != identityToken {
			http.Error(w, "wrong identity", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/api/v1/browser/resolve-context" {
			if fullRevoke.Load() {
				http.Error(w, "grant revoked", http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"caller_id": "run:" + runID, "run_id": runID, "grant_receipt": grantReceipt,
				"capability": map[string]any{"enabled": true, "grantReceipt": grantReceipt, "mode": "managed", "permissions": map[string]any{"profileId": "profile-1", "profileMode": "fresh"}}})
			return
		}
		if r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != grantReceipt || fullRevoke.Load() {
			http.Error(w, "grant revoked", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/browser/resolve-caller":
			_ = json.NewEncoder(w).Encode(map[string]any{"caller_id": "run:" + runID, "grant_receipt": grantReceipt})
		case "/api/v1/browser/resolve-profile":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "active", "run_id": runID, "grant_receipt": grantReceipt, "mode": "managed",
				"permission_profile":         map[string]any{"profileId": "profile-1", "profileMode": "fresh", "level": "act", "allowedOrigins": []string{"https://example.test"}, "allowHistory": true},
				"session_permission_mapping": map[string]any{"view": true, "control": control.Load(), "profile_mode": "fresh"},
				"budget":                     map[string]any{"sessionMinutes": 5, "actions": 20}, "unsupported_sessionpermission_fields": []string{},
			})
		case "/api/v1/browser/authorize-open":
			_ = json.NewEncoder(w).Encode(map[string]any{"authorized": true, "grant_receipt": grantReceipt})
		default:
			http.NotFound(w, r)
		}
	}))
	defer am.Close()

	driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/session/start":
			starts.Add(1)
			_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{SessionID: browserID, LeaseID: "lease-1", ActivePageID: "page-1"})
		case strings.HasSuffix(r.URL.Path, "/record/navigate"):
			var req struct {
				URL string `json:"url"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			currentURL.Store(req.URL)
			_ = json.NewEncoder(w).Encode(driver.NavigateResponse{DriverPageID: "page-1", URL: req.URL, Title: "Example"})
		case strings.HasSuffix(r.URL.Path, "/record/navigation-state"):
			_ = json.NewEncoder(w).Encode(driver.NavigationStateResponse{SessionID: browserID, URL: currentURL.Load().(string), Title: "Example"})
		case strings.HasSuffix(r.URL.Path, "/record/screenshot"):
			_ = json.NewEncoder(w).Encode(driver.CaptureScreenshotResponse{MediaType: "image/png", Width: 1, Height: 1, CapturedAt: "2026-11-19T00:00:00Z"})
		case strings.HasSuffix(r.URL.Path, "/run"):
			effects.Add(1)
			_ = json.NewEncoder(w).Encode(contracts.StepOutcome{Success: true, StepType: "click"})
		case strings.HasSuffix(r.URL.Path, "/close"):
			closes.Add(1)
			_ = json.NewEncoder(w).Encode(driver.CloseSessionResponse{Success: true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer driverServer.Close()

	driverClient, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
	requireNoError(t, err)
	basManager := session.NewManagerWithClient(driverClient)
	runContext := &access.AgentManagerRunContext{BaseURL: am.URL, Client: am.Client()}
	openAuth := &access.AgentManagerOpenAuthorizer{BaseURL: am.URL, Client: am.Client(), Context: runContext}
	broker, err := access.NewBroker(basManager, &access.AgentManagerProfileResolver{RunContext: runContext}, openAuth, runContext)
	requireNoError(t, err)
	broker.SetTrustedOpenContextResolver(runContext)
	broker.SetCapabilityReporter(basfixture.QualifiedManagedAct())
	broker.SetEvidenceRecorder(basfixture.NoopEvidenceRecorder())

	workflows := fakeWorkflowCatalog{}
	basConnect := bashandlers.Module(bashandlers.Deps{Broker: broker, Workflows: workflows})
	basMux := chi.NewRouter()
	basMux.Route("/api/v1", func(r chi.Router) { (&access.HTTPHandler{Broker: broker, Workflows: workflows}).RegisterRoutes(r) })
	basRoot := http.NewServeMux()
	basRoot.Handle(basConnect.Path, basConnect.Handler)
	basRoot.Handle("/api/v1/", basMux)
	basServer := httptest.NewServer(basRoot)
	defer basServer.Close()
	basAPI := basServer.URL + "/api/v1"

	// The actual `vrooli browser mcp` command fails closed when Agent Manager
	// does not inject both credentials, before it can contact BAS.
	if _, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, "", nil); err == nil || !strings.Contains(err.Error(), "VROOLI_BROWSER_GRANT_RECEIPT is required") {
		t.Fatalf("MCP without injected grant err=%v", err)
	}
	if starts.Load() != 0 || amCalls.Load() != 0 {
		t.Fatalf("missing MCP grant reached owners: starts=%d AM=%d", starts.Load(), amCalls.Load())
	}

	// A wrong grant is rejected by the real BAS route and AM resolver before
	// the existing Session.Manager/driver lifecycle starts.
	wrongGrantReplies, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, "forged-grant", []map[string]any{
		{"name": "browser_open", "arguments": map[string]any{"start_url": "https://example.test/start"}},
	})
	requireNoError(t, err)
	assertMCPToolError(t, wrongGrantReplies[1], "browser_open")
	if starts.Load() != 0 {
		t.Fatalf("wrong MCP grant reached browser owner: starts=%d", starts.Load())
	}

	// The MCP schema omits authority and executable controls, and the actual
	// stdio path rejects attempted injection before making an HTTP request.
	amCallsBeforeForgedArgs := amCalls.Load()
	forgedReplies, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_open", "arguments": map[string]any{"start_url": "https://example.test/start", "permission_ref": "profile-forged"}},
		{"name": "browser_open", "arguments": map[string]any{"start_url": "https://example.test/start", "executable_path": "/tmp/browser"}},
		{"name": "browser_act", "arguments": map[string]any{"session_id": "nonexistent", "instruction": map[string]any{"action": map[string]any{"type": "ACTION_TYPE_EVALUATE", "evaluate": map[string]any{"expression": "document.cookie"}}}}},
	})
	requireNoError(t, err)
	assertMCPToolError(t, forgedReplies[1], "browser_open")
	assertMCPToolError(t, forgedReplies[2], "browser_open")
	assertMCPToolError(t, forgedReplies[3], "browser_act")
	if amCalls.Load() != amCallsBeforeForgedArgs || starts.Load() != 0 {
		t.Fatalf("forged MCP args reached owners: AM before=%d after=%d starts=%d", amCallsBeforeForgedArgs, amCalls.Load(), starts.Load())
	}

	root, err := repocontract.ResolveRepoRoot()
	requireNoError(t, err)
	registry, err := bindings.LoadFiles(filepath.Join(root, "packages/proto/gen/descriptor/image.binpb"), []string{filepath.Join(root, "scenarios/browser-automation-studio/cli/manifest.json")})
	requireNoError(t, err)
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) { return basServer.URL, nil })
	registry.SetOutboundRequestPreparer(programhandlers.NewBrowserContextRequestPreparer(am.Client(), func(context.Context, string) (string, error) { return am.URL, nil }))

	prSessions := programsessionstore.NewManager(programsessionstore.Options{})
	prAPIServer := server.New(server.Deps{Clock: schedule.System(), Logger: log.New(io.Discard, "", 0), IdentityVerifier: staticVerifier{}},
		programhandlers.Module(registry), programsessionhandlers.Module(prSessions))
	privateMux := http.NewServeMux()
	privateMux.Handle("/internal/program-runtime/bindings/execute", provenance.Middleware(staticVerifier{})(programhandlers.BridgeWithPrivateIdentity(registry, prSessions, nil)))
	privateMux.Handle("/", prAPIServer.Handler())
	prServer := httptest.NewServer(privateMux)
	defer prServer.Close()

	// Create the Program Runtime session through its actual generated Connect
	// server; kernel calls below enter the actual mounted private HTTP route.
	prSessionClient := programsessionsconnect.NewSessionServiceClient(prServer.Client(), prServer.URL)
	created, err := prSessionClient.CreateSession(context.Background(), connect.NewRequest(&programsessions.CreateSessionRequest{Name: "mb5 browser integration", Grants: []string{"network:internal"}}))
	requireNoError(t, err)
	prSessionID := created.Msg.GetSession().GetId()

	// An invalid Agent Manager identity is rejected by the Program Runtime
	// provenance middleware before it calls AM or the Broker/session owner.
	amCallsBeforeUnverifiedIdentity := amCalls.Load()
	if _, err := programCallWithToken(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/open", map[string]any{"backend": "managed", "start_url": "https://example.test/start", "reuse_mode": "fresh"}, "unverified-token"); err == nil {
		t.Fatal("Program Runtime accepted an unverified Agent Manager identity")
	}
	if amCalls.Load() != amCallsBeforeUnverifiedIdentity || starts.Load() != 0 {
		t.Fatalf("unverified identity reached downstream owners: AM calls before=%d after=%d driver starts=%d", amCallsBeforeUnverifiedIdentity, amCalls.Load(), starts.Load())
	}

	// An absent persisted AM grant fails during the server-side request
	// preparer before the Broker's browser driver creates a session.
	fullRevoke.Store(true)
	if _, err := programCallError(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/open", map[string]any{"backend": "managed", "start_url": "https://example.test/start", "reuse_mode": "fresh"}); err == nil {
		t.Fatal("Program Runtime opened a browser without a current AM grant")
	}
	if starts.Load() != 0 {
		t.Fatalf("missing grant reached browser owner: starts=%d", starts.Load())
	}
	fullRevoke.Store(false)

	// Program Runtime's declared open contract does not expose profile
	// selection. A forged legacy field may reach BAS for compatibility, but
	// Broker.Open resolves trusted AM context and denies the mismatch before a
	// profile is used or the session owner starts.
	amCallsBeforeProfileInjection := amCalls.Load()
	if _, err := programCallError(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/open", map[string]any{"permission_ref": "profile-forged", "backend": "managed", "start_url": "https://example.test/start", "reuse_mode": "fresh"}); err == nil {
		t.Fatal("Program Runtime accepted caller-selected browser permission profile")
	}
	if amCalls.Load() <= amCallsBeforeProfileInjection || starts.Load() != 0 {
		t.Fatalf("profile mismatch must resolve trusted AM context but stop before browser creation: AM before=%d after=%d driver starts=%d", amCallsBeforeProfileInjection, amCalls.Load(), starts.Load())
	}

	openResult := programCall(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/open", map[string]any{"backend": "managed", "start_url": "https://example.test/start", "reuse_mode": "fresh"})
	var opened struct {
		SessionID string `json:"sessionId"`
	}
	if err := mapInto(openResult, &opened); err != nil || opened.SessionID != browserID {
		t.Fatalf("Program Runtime open=%v err=%v", openResult, err)
	}
	if _, leaked := openResult["permissionRef"]; leaked {
		t.Fatalf("Program Runtime open exposed trusted permission profile reference: %v", openResult)
	}
	if strings.Contains(fmt.Sprint(openResult), identityToken) || strings.Contains(fmt.Sprint(openResult), grantReceipt) {
		t.Fatalf("Program Runtime result exposed credentials: %v", openResult)
	}

	observeResult := programCall(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/observe", map[string]any{"session_id": browserID})
	if observeResult["sessionId"] != browserID {
		t.Fatalf("Program Runtime observe result=%v", observeResult)
	}
	programCall(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/act", map[string]any{"session_id": browserID, "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "#pr-button"}}})
	if effects.Load() != 1 {
		t.Fatalf("typed Program Runtime action effects=%d", effects.Load())
	}

	// Run MCP through the real CLI stdio dispatcher and the production BAS REST
	// handler. Observe succeeds under the active grant; action denial from
	// observe-only/control-narrowed and URL redirect creates no driver effect.
	mcpObserveAndTrace, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_observe", "arguments": map[string]any{"session_id": browserID}},
		{"name": "browser_trace", "arguments": map[string]any{"session_id": browserID}},
	})
	requireNoError(t, err)
	if mcpObserveAndTrace[1]["error"] != nil || mcpObserveAndTrace[2]["error"] != nil {
		t.Fatalf("MCP observe/trace failed: %v", mcpObserveAndTrace)
	}
	assertMCPNoAuthority(t, mcpObserveAndTrace)
	control.Store(false)
	observeOnlyReply, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_act", "arguments": map[string]any{"session_id": browserID, "instruction": map[string]any{"index": 0, "node_id": "mcp-observe-only", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "#mcp-button"}}}}},
	})
	requireNoError(t, err)
	assertMCPToolError(t, observeOnlyReply[1], "browser_act")
	if effects.Load() != 1 {
		t.Fatalf("observe-only MCP action reached driver: effects=%d", effects.Load())
	}
	control.Store(true)
	currentURL.Store("https://blocked.test/redirect")
	scopeReply, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_act", "arguments": map[string]any{"session_id": browserID, "instruction": map[string]any{"index": 0, "node_id": "mcp-redirect", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "#mcp-button"}}}}},
	})
	requireNoError(t, err)
	assertMCPToolError(t, scopeReply[1], "browser_act")
	if effects.Load() != 1 {
		t.Fatalf("redirected MCP action reached driver: effects=%d", effects.Load())
	}
	currentURL.Store("https://example.test/start")
	mcpActionAndTrace, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_act", "arguments": map[string]any{"session_id": browserID, "instruction": map[string]any{"index": 0, "node_id": "mcp-click", "action": map[string]any{"type": "ACTION_TYPE_CLICK", "click": map[string]any{"selector": "#mcp-button"}}}}},
		{"name": "browser_observe", "arguments": map[string]any{"session_id": browserID}},
		{"name": "browser_trace", "arguments": map[string]any{"session_id": browserID}},
	})
	requireNoError(t, err)
	if mcpActionAndTrace[1]["error"] != nil || mcpActionAndTrace[2]["error"] != nil || mcpActionAndTrace[3]["error"] != nil || effects.Load() != 2 {
		t.Fatalf("MCP action/trace failed or driver count drifted: replies=%v effects=%d", mcpActionAndTrace, effects.Load())
	}
	if err := assertMCPTrace(mcpActionAndTrace[3], browserID, "#mcp-button"); err != nil {
		t.Fatal(err)
	}
	assertMCPNoAuthority(t, mcpActionAndTrace)
	mcpDraft, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_draft_trace", "arguments": map[string]any{
			"session_id": browserID, "name": "MCP preview",
			"assertions": []any{
				map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "#mcp-button", "mode": "ASSERTION_MODE_VISIBLE"}},
				map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "#mcp-button", "mode": "ASSERTION_MODE_VISIBLE"}},
				map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "#mcp-button", "mode": "ASSERTION_MODE_VISIBLE"}},
			},
		}},
	})
	requireNoError(t, err)
	if mcpDraft[1]["error"] != nil {
		t.Fatalf("MCP draft tool error=%v", mcpDraft[1])
	}
	if _, err := assertMCPDraft(mcpDraft[1], browserID); err != nil {
		t.Fatal(err)
	}
	assertMCPNoAuthority(t, mcpDraft)

	// CLI REST can observe and act on the exact session opened by the Program
	// Runtime registry. The resulting Broker trace is shared across both paths.
	cliObserve(t, basServer.Client(), basServer.URL+"/api/v1/browser/sessions/"+browserID+"/observe")
	cliCall(t, basServer.Client(), basServer.URL+"/api/v1/browser/sessions/"+browserID+"/task", `{"operation":"click","selector":"#continue"}`, http.StatusOK)
	programCall(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/observe", map[string]any{"session_id": browserID})
	trace := programCall(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/read-trace", map[string]any{"session_id": browserID})
	if trace["sessionId"] != browserID {
		t.Fatalf("trace not bound to opened session: %v", trace)
	}
	if effects.Load() != 3 || starts.Load() != 1 {
		t.Fatalf("session effects=%d starts=%d, want 3/1", effects.Load(), starts.Load())
	}
	if !strings.Contains(fmt.Sprint(trace), "#mcp-button") {
		t.Fatalf("Program Runtime trace does not include MCP action: %v", trace)
	}
	sharedAssertions := []any{
		map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "#mcp-button", "mode": "ASSERTION_MODE_VISIBLE"}},
		map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "#mcp-button", "mode": "ASSERTION_MODE_VISIBLE"}},
		map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "#mcp-button", "mode": "ASSERTION_MODE_VISIBLE"}},
		map[string]any{"type": "ACTION_TYPE_ASSERT", "assert": map[string]any{"selector": "#mcp-button", "mode": "ASSERTION_MODE_VISIBLE"}},
	}
	sharedMCPReplies, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_draft_trace", "arguments": map[string]any{
			"session_id": browserID, "name": "Shared trace draft", "assertions": sharedAssertions,
		}},
	})
	requireNoError(t, err)
	sharedMCPDigest, err := assertMCPDraft(sharedMCPReplies[1], browserID)
	if err != nil {
		t.Fatal(err)
	}
	assertMCPNoAuthority(t, sharedMCPReplies)
	programDraft := programCall(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/draft-trace", map[string]any{
		"session_id": browserID, "name": "Shared trace draft", "assertions": sharedAssertions,
	})
	if programDraft["sessionId"] != browserID || programDraft["draftDigest"] == "" || programDraft["draftDigest"] != sharedMCPDigest || programDraft["definition"] == nil || programDraft["workflowId"] != nil {
		t.Fatalf("Program Runtime draft is not an unsaved session-bound preview: %v", programDraft)
	}
	if effects.Load() != 3 || starts.Load() != 1 {
		t.Fatalf("trace draft created a browser effect or second session: effects=%d starts=%d", effects.Load(), starts.Load())
	}

	// Scope loss and a full grant revocation are checked before any further
	// driver effect. The profile-narrowing case still permits only owner cleanup.
	currentURL.Store("https://blocked.test/redirect")
	if _, err := programCallError(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/act", map[string]any{"session_id": browserID, "action": map[string]any{"type": "ACTION_TYPE_CLICK"}}); err == nil {
		t.Fatal("Program Runtime action succeeded after out-of-scope redirect")
	}
	if effects.Load() != 3 {
		t.Fatalf("scope-denied action reached driver: effects=%d", effects.Load())
	}
	currentURL.Store("https://example.test/start")
	control.Store(false)
	if _, err := programCallError(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/act", map[string]any{"session_id": browserID, "action": map[string]any{"type": "ACTION_TYPE_CLICK"}}); err == nil {
		t.Fatal("Program Runtime action succeeded after profile control was narrowed")
	}
	if effects.Load() != 3 {
		t.Fatalf("profile-narrowed action reached driver: effects=%d", effects.Load())
	}

	fullRevoke.Store(true)
	mcpRevoked, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_observe", "arguments": map[string]any{"session_id": browserID}},
	})
	requireNoError(t, err)
	assertMCPToolError(t, mcpRevoked[1], "browser_observe")
	if strings.Contains(fmt.Sprint(mcpRevoked), identityToken) || strings.Contains(fmt.Sprint(mcpRevoked), grantReceipt) || strings.Contains(fmt.Sprint(mcpRevoked), "profile-1") {
		t.Fatalf("revoked MCP result leaked authority/profile: %v", mcpRevoked)
	}
	if _, err := programCallError(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/observe", map[string]any{"session_id": browserID}); err == nil {
		t.Fatal("Program Runtime call succeeded after full grant revocation")
	}
	staleGrantClient := agentaccessconnect.NewBrowserSessionServiceClient(basServer.Client(), basServer.URL, connect.WithProtoJSON())
	staleGrantAct := connect.NewRequest(&agentaccessv1.ActRequest{SessionId: browserID})
	staleGrantAct.Header().Set("X-Agent-Identity-Token", identityToken)
	staleGrantAct.Header().Set("X-Vrooli-Browser-Grant-Receipt", grantReceipt)
	if _, err := staleGrantClient.Act(context.Background(), staleGrantAct); err == nil {
		t.Fatal("BAS Broker accepted a stale grant after full revocation")
	}
	if effects.Load() != 3 {
		t.Fatalf("revoked call caused an effect: effects=%d", effects.Load())
	}
	fullRevoke.Store(false)
	control.Store(false)
	// An independent CLI/REST call owns cleanup after permission narrowing.
	cliCall(t, basServer.Client(), basServer.URL+"/api/v1/browser/sessions/"+browserID+"/close", `{}`, http.StatusNoContent)
	if closes.Load() != 1 {
		t.Fatalf("profile-narrowed owner cleanup count=%d, want 1", closes.Load())
	}
	mcpStale, err := invokeBrowserMCP(t, mcpCLI, basAPI, identityToken, grantReceipt, []map[string]any{
		{"name": "browser_observe", "arguments": map[string]any{"session_id": browserID}},
	})
	requireNoError(t, err)
	assertMCPToolError(t, mcpStale[1], "browser_observe")
	assertMCPNoAuthority(t, mcpStale)
	if _, err := programCallError(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/observe", map[string]any{"session_id": browserID}); err == nil {
		t.Fatal("closed/replayed browser session remained usable")
	}
	programCall(t, prServer.Client(), prServer.URL, prSessionID, "browser-automation-studio/session/close", map[string]any{"session_id": browserID})
	if closes.Load() != 1 {
		t.Fatalf("repeat CLI close redispatched driver close: %d", closes.Load())
	}
}

func programCall(t *testing.T, client *http.Client, base, sessionID, bindingID string, args map[string]any) map[string]any {
	t.Helper()
	result, err := programCallError(t, client, base, sessionID, bindingID, args)
	requireNoError(t, err)
	return result
}

func programCallError(t *testing.T, client *http.Client, base, sessionID, bindingID string, args map[string]any) (map[string]any, error) {
	return programCallWithToken(t, client, base, sessionID, bindingID, args, identityToken)
}

func programCallWithToken(t *testing.T, client *http.Client, base, sessionID, bindingID string, args map[string]any, token string) (map[string]any, error) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"session_id": sessionID, "program_id": "mb5-test", "provenance": "PROVENANCE_TEST", "binding_id": bindingID, "args": args})
	requireNoError(t, err)
	if bytes.Contains(body, []byte(identityToken)) || bytes.Contains(body, []byte(grantReceipt)) {
		return nil, fmt.Errorf("browser authority material entered Program Runtime request data")
	}
	req, err := http.NewRequest(http.MethodPost, base+"/internal/program-runtime/bindings/execute", bytes.NewReader(body))
	requireNoError(t, err)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	encodedResult, _ := json.Marshal(result)
	if strings.Contains(string(encodedResult), identityToken) || strings.Contains(string(encodedResult), grantReceipt) {
		return result, fmt.Errorf("Program Runtime response echoed browser authority material")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("Program Runtime HTTP status %d: %v", resp.StatusCode, result)
	}
	return result, nil
}

func cliObserve(t *testing.T, client *http.Client, endpoint string) {
	t.Helper()
	cliCall(t, client, endpoint, `{}`, http.StatusOK)
}

func cliCall(t *testing.T, client *http.Client, endpoint, body string, want int) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	requireNoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Identity-Token", identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", grantReceipt)
	resp, err := client.Do(req)
	requireNoError(t, err)
	defer resp.Body.Close()
	if resp.StatusCode != want {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("CLI route %s status=%d body=%s want=%d", endpoint, resp.StatusCode, data, want)
	}
}

// buildVrooliBrowserCLI builds the repository's actual CLI entrypoint into the
// test temp directory without contacting a package proxy or launching a browser.
func buildVrooliBrowserCLI(t *testing.T) string {
	t.Helper()
	root, err := repocontract.ResolveRepoRoot()
	requireNoError(t, err)
	binary := filepath.Join(t.TempDir(), "vrooli")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/vrooli")
	command.Dir = root
	command.Env = isolatedCLIEnv("", "", "")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build source CLI (GOPROXY=off) failed: %v\n%s", err, output)
	}
	return binary
}

// invokeBrowserMCP runs the actual `vrooli browser mcp` binary with newline-
// delimited MCP messages. It uses the same HTTP BAS endpoint as CLI REST and
// Program Runtime; the subprocess owns no browser or session lifecycle.
func invokeBrowserMCP(t *testing.T, binary, basAPI, token, grant string, calls []map[string]any) ([]map[string]any, error) {
	t.Helper()
	var input strings.Builder
	_, _ = input.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n")
	for index, call := range calls {
		message := map[string]any{"jsonrpc": "2.0", "id": index + 2, "method": "tools/call", "params": call}
		data, err := json.Marshal(message)
		requireNoError(t, err)
		input.Write(data)
		input.WriteByte('\n')
	}
	var output, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "browser", "mcp")
	command.Env = isolatedCLIEnv(basAPI, token, grant)
	command.Stdin = strings.NewReader(input.String())
	command.Stdout, command.Stderr = &output, &stderr
	err := command.Run()
	if err != nil {
		return nil, fmt.Errorf("vrooli browser mcp: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != len(calls)+1 {
		t.Fatalf("MCP stdio responses=%d want=%d body=%s", len(lines), len(calls)+1, output.String())
	}
	replies := make([]map[string]any, len(lines))
	for index, line := range lines {
		if err := json.Unmarshal([]byte(line), &replies[index]); err != nil {
			t.Fatalf("decode MCP response %d: %v; body=%s", index, err, output.String())
		}
	}
	return replies, nil
}

func isolatedCLIEnv(basAPI, token, grant string) []string {
	blocked := map[string]struct{}{
		"BAS_BROWSER_API_URL": {}, "VROOLI_AGENT_IDENTITY_TOKEN": {}, "VROOLI_BROWSER_GRANT_RECEIPT": {},
	}
	env := make([]string, 0, len(os.Environ())+3)
	for _, item := range os.Environ() {
		key, _, found := strings.Cut(item, "=")
		if _, remove := blocked[key]; found && !remove {
			env = append(env, item)
		}
	}
	env = append(env, "BAS_BROWSER_API_URL="+basAPI, "VROOLI_AGENT_IDENTITY_TOKEN="+token, "VROOLI_BROWSER_GRANT_RECEIPT="+grant)
	return env
}

func assertMCPToolError(t *testing.T, reply map[string]any, name string) {
	t.Helper()
	if reply["error"] == nil {
		t.Fatalf("MCP %s unexpectedly succeeded: %v", name, reply)
	}
	encoded, _ := json.Marshal(reply)
	if strings.Contains(string(encoded), identityToken) || strings.Contains(string(encoded), grantReceipt) {
		t.Fatalf("MCP %s error leaked authority: %s", name, encoded)
	}
}

func assertMCPNoAuthority(t *testing.T, replies []map[string]any) {
	t.Helper()
	encoded, _ := json.Marshal(replies)
	for _, secret := range []string{identityToken, grantReceipt, "profile-1", "profile-forged"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("MCP arguments/results leaked authority or profile data %q: %s", secret, encoded)
		}
	}
}

func assertMCPTrace(reply map[string]any, sessionID, selector string) error {
	result, ok := reply["result"].(map[string]any)
	if !ok {
		return fmt.Errorf("MCP trace result missing: %v", reply)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		return fmt.Errorf("MCP trace content missing: %v", result)
	}
	textResult, ok := content[0].(map[string]any)["text"].(string)
	if !ok {
		return fmt.Errorf("MCP trace text missing: %v", content[0])
	}
	var trace struct {
		SessionID     string `json:"session_id"`
		PermissionRef string `json:"permission_ref"`
		Events        []struct {
			Sequence    uint64 `json:"sequence"`
			SessionID   string `json:"session_id"`
			Kind        string `json:"kind"`
			Instruction *struct {
				Action struct {
					Params struct {
						Click struct {
							Selector string `json:"selector"`
						} `json:"click"`
					} `json:"params"`
				} `json:"action"`
			} `json:"instruction"`
		} `json:"events"`
	}
	if err := json.Unmarshal([]byte(textResult), &trace); err != nil {
		return fmt.Errorf("decode MCP trace text: %w", err)
	}
	if trace.SessionID != sessionID || trace.PermissionRef != "" {
		return fmt.Errorf("MCP trace session/profile binding=%q/%q want %q/omitted", trace.SessionID, trace.PermissionRef, sessionID)
	}
	var previous uint64
	found := false
	for _, event := range trace.Events {
		if event.SessionID != sessionID || event.Sequence <= previous {
			return fmt.Errorf("MCP trace event binding/order invalid: %#v", event)
		}
		previous = event.Sequence
		if event.Kind == "action" && event.Instruction != nil && event.Instruction.Action.Params.Click.Selector == selector {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("MCP trace omitted its typed action selector %q", selector)
	}
	return nil
}

func assertMCPDraft(reply map[string]any, sessionID string) (string, error) {
	result, ok := reply["result"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("MCP draft result missing: %v", reply)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("MCP draft content missing: %v", result)
	}
	textResult, ok := content[0].(map[string]any)["text"].(string)
	if !ok {
		return "", fmt.Errorf("MCP draft text missing: %v", content[0])
	}
	var draft struct {
		SessionID   string          `json:"session_id"`
		DraftDigest string          `json:"draft_digest"`
		Definition  json.RawMessage `json:"definition"`
		WorkflowID  string          `json:"workflow_id"`
	}
	if err := json.Unmarshal([]byte(textResult), &draft); err != nil {
		return "", fmt.Errorf("decode MCP draft text: %w", err)
	}
	if draft.SessionID != sessionID || draft.DraftDigest == "" || len(draft.Definition) == 0 || draft.WorkflowID != "" {
		return "", fmt.Errorf("MCP draft is not an unsaved session-bound preview: %#v", draft)
	}
	return draft.DraftDigest, nil
}

func mapInto(value map[string]any, target any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
