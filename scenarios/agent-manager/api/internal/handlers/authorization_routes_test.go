package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/owneridentity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

func TestNarrowWorkflowChildCanReadOwningExecutionWithoutManagementScope(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorAgent, Subject: "child-run", Verified: true, Source: identity.SourceAgentProvenance, Scopes: []string{"git-control-tower:read"}}
	request := httptest.NewRequest(http.MethodPost, apiconnect.AgentManagerServiceGetWorkflowExecutionProcedure, strings.NewReader(`{"executionId":"owning-workflow"}`))
	request = request.WithContext(identity.WithPrincipal(request.Context(), principal))
	passed := false
	h := AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { passed = true; w.WriteHeader(http.StatusNoContent) }))
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	if !passed || recorder.Code != http.StatusNoContent {
		t.Fatalf("read-only child cannot verify its evidence owner: %d", recorder.Code)
	}
}

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

func TestPolicyValidationAndPreviewReadsDoNotOpenMutationRoutes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "role policy explain REST", path: "/api/v1/role-policy/explain", wantStatus: http.StatusNoContent},
		{name: "role policy explain Connect", path: apiconnect.AgentManagerServiceExplainRolePolicyProcedure, wantStatus: http.StatusNoContent},
		{name: "role policy validate REST", path: "/api/v1/role-policy/validate", wantStatus: http.StatusNoContent},
		{name: "role policy validate Connect", path: apiconnect.AgentManagerServiceValidateRolePolicyCatalogProcedure, wantStatus: http.StatusNoContent},
		{name: "permission policy validate REST", path: "/api/v1/permission-policy/validate", wantStatus: http.StatusNoContent},
		{name: "permission policy validate Connect", path: apiconnect.AgentManagerServiceValidatePermissionPolicyCatalogProcedure, wantStatus: http.StatusNoContent},
		{name: "permission policy plan REST", path: "/api/v1/permission-policy/plan", wantStatus: http.StatusNoContent},
		{name: "permission policy plan Connect", path: apiconnect.AgentManagerServicePlanPermissionPolicyProcedure, wantStatus: http.StatusNoContent},
		{name: "permission policy doctor REST", path: "/api/v1/permission-policy/doctor", wantStatus: http.StatusNoContent},
		{name: "permission policy doctor Connect", path: apiconnect.AgentManagerServiceDoctorPermissionPolicyProcedure, wantStatus: http.StatusNoContent},
		{name: "role policy reload REST stays protected", path: "/api/v1/role-policy/reload", wantStatus: http.StatusForbidden},
		{name: "role policy reload Connect stays protected", path: apiconnect.AgentManagerServiceReloadRolePolicyCatalogProcedure, wantStatus: http.StatusForbidden},
		{name: "permission policy reload REST stays protected", path: "/api/v1/permission-policy/reload", wantStatus: http.StatusForbidden},
		{name: "permission policy reconcile REST stays protected", path: "/api/v1/permission-policy/reconcile", wantStatus: http.StatusForbidden},
		{name: "permission policy reconcile Connect stays protected", path: apiconnect.AgentManagerServiceReconcilePermissionPolicyProcedure, wantStatus: http.StatusForbidden},
		{name: "unknown route stays protected", path: "/api/v1/permission-policy/future", wantStatus: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			principal := identity.Principal{Kind: identity.ActorAgent, Subject: "run-read", Verified: true}
			h := AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"profile_id":"00000000-0000-4000-8000-000000000001"}`))
			req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			wantReached := tc.wantStatus == http.StatusNoContent
			if w.Code != tc.wantStatus || reached != wantReached {
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

func TestDeliveryCommissionReachesItsBoundedOwnerHandler(t *testing.T) {
	ctx := grantContext([]string{"effort.delivery"}, []string{"agent-manager:effort/hub"})
	r := httptest.NewRequest(http.MethodPost, apiconnect.AgentManagerServiceEnrollEffortProcedure, nil).WithContext(ctx)
	reached := false
	AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })).ServeHTTP(httptest.NewRecorder(), r)
	if !reached {
		t.Fatal("approved delivery call cannot reach its domain authorization check")
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

func TestVerifiedAgentCanWaitForWorkflowExecutionWithoutOpeningMutations(t *testing.T) {
	paths := []string{
		"/api/v1/workflow-executions/{id}/wait",
		"/api/v1/workflow-executions/{id}/advance",
		"/api/v1/workflow-executions/{id}/signals",
		"/api/v1/workflow-executions/{id}/cancel",
		"/api/v1/workflow-executions/{id}/retry",
		"/api/v1/workflow-executions/{id}/resume",
	}
	for _, tc := range []struct {
		name       string
		path       string
		grant      bool
		wantStatus int
	}{
		{name: "wait is an ordinary verified-agent read", path: paths[0], wantStatus: http.StatusNoContent},
		{name: "interactive grant does not open advance", path: paths[1], grant: true, wantStatus: http.StatusForbidden},
		{name: "interactive grant does not open signals", path: paths[2], grant: true, wantStatus: http.StatusForbidden},
		{name: "interactive grant does not open cancel", path: paths[3], grant: true, wantStatus: http.StatusForbidden},
		{name: "interactive grant does not open retry", path: paths[4], grant: true, wantStatus: http.StatusForbidden},
		{name: "interactive grant does not open resume", path: paths[5], grant: true, wantStatus: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := mux.NewRouter()
			router.Handle(tc.path, AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))).Methods(http.MethodPost)
			p := identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Verified: true, Scopes: []string{"agent-manager:write"}}
			if tc.grant {
				p.Grant = &identity.OperationGrant{}
			}
			r := httptest.NewRequest(http.MethodPost, strings.Replace(tc.path, "{id}", "execution-1", 1), nil)
			r = r.WithContext(identity.WithPrincipal(r.Context(), p))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tc.wantStatus {
				t.Fatalf("status=%d want %d body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestConversationSearchUsesOrdinaryReadAdmission(t *testing.T) {
	path := "/agent_manager.v1.ConversationSearchService/SearchConversations"
	for _, tc := range []struct {
		name          string
		principal     *identity.Principal
		invalidHeader bool
		want          int
	}{
		{name: "anonymous local read", want: http.StatusNoContent},
		{name: "ordinary read middleware with token", invalidHeader: true, want: http.StatusNoContent},
		{name: "ordinary read middleware with unverified context", principal: &identity.Principal{Kind: identity.ActorAgent, Subject: "run", Scopes: []string{"agent-manager:read"}}, want: http.StatusNoContent},
		{name: "verified identity without read scope", principal: &identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Scopes: []string{"other:read"}}, want: http.StatusNoContent},
		{name: "read", principal: &identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Scopes: []string{"agent-manager:read"}}, want: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			if tc.principal != nil {
				req = req.WithContext(identity.WithPrincipal(req.Context(), *tc.principal))
			}
			if tc.invalidHeader {
				req.Header.Set("X-Agent-Identity-Token", "invalid")
			}
			rec := httptest.NewRecorder()
			AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d want=%d", rec.Code, tc.want)
			}
		})
	}
}

// An agent's evidence prune reaches the handler, which turns a real prune into
// an operator decision instead of the middleware's bare global-management 403.
func TestAgentEvidencePruneReachesTheApprovalHandler(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true}
		if interactive {
			p.Grant = &identity.OperationGrant{}
		}
		r := httptest.NewRequest(http.MethodPost, "/api/v1/storage/evidence/prune", strings.NewReader(`{"efforts":["x"],"reason":"r"}`))
		r = r.WithContext(identity.WithPrincipal(r.Context(), p))
		reached := false
		w := httptest.NewRecorder()
		AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true; w.WriteHeader(http.StatusAccepted) })).ServeHTTP(w, r)
		if !reached || w.Code != http.StatusAccepted {
			t.Fatalf("interactive=%t: agent evidence prune refused at the middleware: %d %s", interactive, w.Code, w.Body.String())
		}
	}
}
