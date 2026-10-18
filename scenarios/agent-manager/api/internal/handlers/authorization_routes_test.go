package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/owneridentity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

// Phase 3: the middleware and the CreateRun handler must agree on an anonymous
// create-run via ONE switch read. These assert the SAME anonymous request is
// refused by the middleware when VROOLI_AUTH01_ENFORCE is on and admitted to the
// handler when it is off — the gate no longer silently overrides the switch.
func TestAnonymousCreateRunHonorsTheEnforcementSwitch(t *testing.T) {
	for _, path := range []string{"/api/v1/runs", "/api/v1/tasks"} {
		// Switch OFF: anonymous create-run/task reaches the handler.
		t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "")
		reached := false
		AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true; w.WriteHeader(http.StatusAccepted) })).
			ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, path, nil))
		if !reached {
			t.Fatalf("switch off: anonymous %s did not reach the handler (middleware overrode the switch)", path)
		}
		// Switch ON: the same anonymous request is refused at the middleware.
		t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
		w := httptest.NewRecorder()
		AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatalf("switch on: anonymous %s reached the handler", path)
		})).
			ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("switch on: anonymous %s want 401, got %d", path, w.Code)
		}
	}
}

// A non-create anonymous operation is refused regardless of the switch — the
// reconciliation is scoped to create-run/task, not a blanket anonymous allow.
func TestAnonymousNonCreateOpAlwaysRefused(t *testing.T) {
	for _, enforce := range []string{"", "true"} {
		t.Setenv(owneridentity.CreateRunCallerEnforceEnv, enforce)
		w := httptest.NewRecorder()
		AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatalf("anonymous global mutation reached the handler (enforce=%q)", enforce)
		})).
			ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous non-create op want 401 (enforce=%q), got %d", enforce, w.Code)
		}
	}
}

func TestAgentRoutesRetainGlobalManagementBoundary(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		for _, path := range []string{apiconnect.AgentManagerServiceCreateCohortWatchProcedure, "/agent_manager.v1.AgentManagerService/FutureMutation", "/api/v1/runs/stop-all"} {
			p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Scopes: []string{"agent-manager:write"}}
			if interactive {
				p.Grant = &identity.OperationGrant{}
			}
			r := httptest.NewRequest(http.MethodPost, path, nil)
			r = r.WithContext(identity.WithPrincipal(r.Context(), p))
			w := httptest.NewRecorder()
			AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("agent acquired global mutation") })).ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("%s: %d", path, w.Code)
			}
		}
	}
}

func TestDeclarationsPlanIsVerifiedAgentReadWithoutOpeningReconcileWrites(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		principal  *identity.Principal
		wantStatus int
	}{
		{
			name: "verified agent may validate declarations",
			path: "/api/v1/declarations/plan",
			principal: &identity.Principal{
				Kind: identity.ActorAgent, Subject: "run-1", Verified: true,
				Scopes: []string{"agent-manager:write"},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "unverified agent remains refused",
			path: "/api/v1/declarations/plan",
			principal: &identity.Principal{
				Kind: identity.ActorAgent, Subject: "run-1", Scopes: []string{"agent-manager:write"},
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "anonymous plan remains refused",
			path:       "/api/v1/declarations/plan",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "verified agent cannot reconcile declarations",
			path: "/api/v1/declarations/reconcile-scenario",
			principal: &identity.Principal{
				Kind: identity.ActorAgent, Subject: "run-1", Verified: true,
				Scopes: []string{"agent-manager:write"},
			},
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			h := AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			if tc.principal != nil {
				req = req.WithContext(identity.WithPrincipal(req.Context(), *tc.principal))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.wantStatus || reached != (tc.wantStatus == http.StatusNoContent) {
				t.Fatalf("status=%d reached=%v want status=%d", w.Code, reached, tc.wantStatus)
			}
		})
	}
}

func TestAgentRoutesPreserveOwnedLocalLifecycleContract(t *testing.T) {
	h := AgentOperationMiddleware(lifecycle.LoopbackOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })))
	for _, tc := range []struct {
		peer string
		want int
	}{{"127.0.0.1:1234", http.StatusNoContent}, {"192.0.2.1:1234", http.StatusForbidden}} {
		r := httptest.NewRequest(http.MethodPost, commonconnect.LifecycleMaintenanceServicePrepareProcedure, nil)
		r.RemoteAddr = tc.peer
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("peer %s: %d", tc.peer, w.Code)
		}
	}
}

func TestLocalIdentityOnlyAttachmentDoesNotRequireAnExistingGrant(t *testing.T) {
	h := AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, tc := range []struct {
		peer string
		want int
	}{{"127.0.0.1:1234", http.StatusNoContent}, {"192.0.2.1:1234", http.StatusForbidden}} {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/runs/attach", nil)
		r.RemoteAddr = tc.peer
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("peer %s: %d", tc.peer, w.Code)
		}
	}
}

// G2: transcript continuity (import-transcript, external/tombstone) is mounted
// LoopbackOnly — a loopback caller (a co-located web-console) reaches the
// handler; a non-loopback caller is refused. This is the chosen contract:
// web-console holds no agent-manager service credential, and continuity is a
// same-host, zero-authority operation, so the loopback fence is its authority.
func TestTranscriptContinuityIsLoopbackOnly(t *testing.T) {
	h := AgentOperationMiddleware(lifecycle.LoopbackOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })))
	for _, path := range []string{"/api/v1/runs/import-transcript", "/api/v1/runs/external/tombstone"} {
		for _, tc := range []struct {
			peer string
			want int
		}{
			{"127.0.0.1:5555", http.StatusCreated},    // positive: loopback import succeeds
			{"192.0.2.10:5555", http.StatusForbidden}, // negative: non-loopback refused
		} {
			r := httptest.NewRequest(http.MethodPost, path, nil)
			r.RemoteAddr = tc.peer
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("%s from %s: got %d want %d", path, tc.peer, w.Code, tc.want)
			}
		}
	}
}

// G2 regression: the previous behavior was a hard 401 for an anonymous
// import-transcript (it fell through to "verified operator required"). Confirm
// an anonymous NON-loopback caller is now refused by the loopback fence (403),
// not silently admitted — the fix does not open the endpoint to anyone.
func TestTranscriptContinuityAnonymousNonLoopbackRefused(t *testing.T) {
	reached := false
	h := AgentOperationMiddleware(lifecycle.LoopbackOnly(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })))
	r := httptest.NewRequest(http.MethodPost, "/api/v1/runs/import-transcript", nil)
	r.RemoteAddr = "203.0.113.7:4444"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if reached {
		t.Fatal("non-loopback anonymous import-transcript reached the handler")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-loopback anonymous import-transcript want 403, got %d", w.Code)
	}
}

func TestSignedEventWebhooksReachTheirSignatureCheckWithoutACaller(t *testing.T) {
	for _, path := range []string{ProgramEventWebhookPath, DecisionEventWebhookPath} {
		reached := false
		h := AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true; w.WriteHeader(http.StatusUnauthorized) }))
		r := httptest.NewRequest(http.MethodPost, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if !reached {
			t.Fatalf("%s: the middleware refused a signed webhook before its handler verified the signature (%d)", path, w.Code)
		}
	}
}

func TestAgentRunsMayRecordEffortBookkeeping(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		for _, path := range []string{apiconnect.AgentManagerServiceRecordEffortSignalProcedure, apiconnect.AgentManagerServiceMarkEffortDecisionRecordedProcedure} {
			p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true}
			if interactive {
				p.Grant = &identity.OperationGrant{}
			}
			r := httptest.NewRequest(http.MethodPost, path, nil)
			r = r.WithContext(identity.WithPrincipal(r.Context(), p))
			w := httptest.NewRecorder()
			AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(w, r)
			if w.Code != http.StatusNoContent {
				t.Fatalf("%s (interactive=%v): %d", path, interactive, w.Code)
			}
		}
	}
}

func TestInteractiveGrantsReachEffortMetadataHandler(t *testing.T) {
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: &identity.OperationGrant{}}
	r := httptest.NewRequest(http.MethodPost, apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure, nil)
	r = r.WithContext(identity.WithPrincipal(r.Context(), p))
	reached := false
	AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })).ServeHTTP(httptest.NewRecorder(), r)
	if !reached {
		t.Fatal("interactive grant could not reach the handler that checks effort.reconcile")
	}
}

func TestOrchestratorRoutesReachTheirOwnHandlerChecks(t *testing.T) {
	for _, tc := range []struct {
		path        string
		interactive bool
		want        bool
	}{
		{"/api/v1/tasks", false, true},
		{"/api/v1/tasks", true, true},
		{"/api/v1/runs/wake-by-key", false, true},
		{"/api/v1/runs/wake-by-key", true, true},
	} {
		p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Scopes: []string{"agent-manager:write"}}
		if tc.interactive {
			p.Grant = &identity.OperationGrant{}
		}
		r := httptest.NewRequest(http.MethodPost, tc.path, nil)
		r = r.WithContext(identity.WithPrincipal(r.Context(), p))
		reached := false
		AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })).ServeHTTP(httptest.NewRecorder(), r)
		if reached != tc.want {
			t.Fatalf("%s interactive=%v: reached=%v, want %v", tc.path, tc.interactive, reached, tc.want)
		}
	}
}
