package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lifecycle"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

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
		{"/api/v1/tasks", true, false},
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
