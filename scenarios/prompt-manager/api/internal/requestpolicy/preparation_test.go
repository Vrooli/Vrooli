package requestpolicy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	aisearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch"
	aisearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/aisearch/aisearch_v1connect"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery/discovery_v1connect"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type denyAuthority struct{}

func (denyAuthority) AuthorizeOperation(context.Context, identity.Principal, string, string, string) error {
	return authn.ErrOperation
}

func TestPreparationGrantAllowsOnlyKnownOrdinaryAndOwnerCheckedOperations(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	paths := []string{"/unknown.Service/NewMutation", "/teams/draft", "/vrooli.prompt_manager.v1.skills.SkillsService/CreateSkill"}
	// Enumerate actual generated owner surfaces: new RPCs default to denied.
	for _, file := range []protoreflect.FileDescriptor{heartbeatv1.File_prompt_manager_v1_heartbeat_heartbeat_proto, teamsv1.File_prompt_manager_v1_teams_teams_proto, discoveryv1.File_prompt_manager_v1_discovery_discovery_proto, aisearchv1.File_prompt_manager_v1_aisearch_aisearch_proto} {
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				paths = append(paths, "/"+string(service.FullName())+"/"+string(service.Methods().Get(j).Name()))
			}
		}
	}
	allowedPaths := map[string]bool{
		discoveryconnect.DiscoveryServiceDiscoverProcedure:                true,
		aisearchconnect.AISearchServiceSearchSkillsProcedure:              true,
		teamsconnect.TeamsServiceListKnowledgeProcedure:                   true,
		teamsconnect.TeamsServiceGetTeamProcedure:                         true,
		teamsconnect.TeamsServiceAddKnowledgeProcedure:                    true,
		teamsconnect.TeamsServiceCloseKnowledgeProcedure:                  true,
		heartbeatconnect.HeartbeatServiceGetHeartbeatProcedure:            true,
		heartbeatconnect.HeartbeatServiceUpdateHeartbeatProcedure:         true,
		heartbeatconnect.HeartbeatServiceCreateHeartbeatProcedure:         true,
		heartbeatconnect.HeartbeatServiceDeleteHeartbeatProcedure:         true,
		heartbeatconnect.HeartbeatServiceCaptureBugProcedure:              true,
		heartbeatconnect.HeartbeatServicePreviewPromptProcedure:           true,
		heartbeatconnect.HeartbeatServicePreviewPromptStructuredProcedure: true,
		heartbeatconnect.HeartbeatServiceListHeartbeatAttemptsProcedure:   true,
		heartbeatconnect.HeartbeatServiceGetTeamExecutionStatusProcedure:  true,
	}
	for _, ingress := range []string{"interactive-grant", "mixed-interactive-grant", "request-authority"} {
		for _, path := range paths {
			t.Run(ingress+path, func(t *testing.T) {
				entered := false
				next := PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					entered = true
					w.WriteHeader(http.StatusNoContent)
				}))
				wire := authn.Middleware(authn.Config{})(next)
				r := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Connect-Protocol-Version", "1")
				grant := &identity.OperationGrant{
					ID: "reviewed", Policy: "team-preparation", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
					Rules: []identity.OperationRule{{Policy: "team-preparation", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}}},
				}
				if ingress == "mixed-interactive-grant" {
					grant.Policy = "team-preparation+agent-management"
					grant.Rules = append(grant.Rules, identity.OperationRule{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run"}})
				}
				if ingress == "request-authority" {
					grant = nil
					r = r.WithContext(authn.WithRequestOperationAuthority(r.Context(), denyAuthority{}))
				}
				r = r.WithContext(provenance.NewContext(r.Context(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, Grant: grant}))
				w := httptest.NewRecorder()
				wire.ServeHTTP(w, r)
				allowed := allowedPaths[path]
				if entered != allowed {
					t.Fatalf("handler entered=%v, allowed=%v: %s", entered, allowed, w.Body.String())
				}
				if !allowed && (w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "permission_denied")) {
					t.Fatalf("wire refusal lost: %d %s", w.Code, w.Body.String())
				}
			})
		}
	}
}

func TestRequirePreparationReadPreservesBaselineReadAlongsideUnrelatedProof(t *testing.T) {
	readPrincipal := identity.Principal{
		Kind: identity.ActorAgent, Subject: "verified-agent", Verified: true,
		Scopes: []string{"prompt-manager:read"},
	}
	ctx := identity.WithPrincipal(context.Background(), readPrincipal)
	ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
	if err := RequirePreparationRead(ctx, "draft"); err != nil {
		t.Fatalf("unrelated request proof narrowed the ordinary PM read: %v", err)
	}
}

func TestRequirePreparationReadDoesNotElevateMissingBaselineOrBroadenGrantTarget(t *testing.T) {
	t.Run("unrelated proof without baseline", func(t *testing.T) {
		ctx := identity.WithPrincipal(context.Background(), identity.Principal{
			Kind: identity.ActorAgent, Subject: "verified-agent", Verified: true,
		})
		ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
		if err := RequirePreparationRead(ctx, "draft"); err == nil {
			t.Fatal("unrelated request proof supplied a missing baseline read or preparation operation")
		}
	})

	grantFor := func(target string) *identity.OperationGrant {
		return &identity.OperationGrant{
			ID: "reviewed-preparation", Policy: "team-preparation", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
			Operations: []string{"team.prepare"}, Targets: []string{target},
		}
	}
	t.Run("grant exact target", func(t *testing.T) {
		ctx := identity.WithPrincipal(context.Background(), identity.Principal{
			Kind: identity.ActorAgent, Subject: "verified-agent", Verified: true,
			Scopes: []string{"prompt-manager:read"}, Grant: grantFor("prompt-manager:team/draft"),
		})
		if err := RequirePreparationRead(ctx, "draft"); err != nil {
			t.Fatalf("exact reviewed preparation target refused: %v", err)
		}
	})
	t.Run("grant cannot cross team target", func(t *testing.T) {
		ctx := identity.WithPrincipal(context.Background(), identity.Principal{
			Kind: identity.ActorAgent, Subject: "verified-agent", Verified: true,
			Scopes: []string{"prompt-manager:read"}, Grant: grantFor("prompt-manager:team/other"),
		})
		if err := RequirePreparationRead(ctx, "draft"); err == nil {
			t.Fatal("preparation grant crossed its reviewed team target")
		}
	})
}

func TestPreparationBoundaryLeavesOrdinaryRequestsWithTheirOwner(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, path := range []string{"/legacy", aisearchconnect.AISearchServiceSearchSkillsProcedure, discoveryconnect.DiscoveryServiceDiscoverProcedure} {
		t.Run(path, func(t *testing.T) {
			entered := false
			w := httptest.NewRecorder()
			PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { entered = true; w.WriteHeader(http.StatusNoContent) })).ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
			if !entered || w.Code != http.StatusNoContent {
				t.Fatalf("boundary changed unbounded request path %q: entered=%v status=%d", path, entered, w.Code)
			}
		})
	}
}

func TestStaticSkillReadUsesOrdinaryCapabilityWithUnrelatedOrMixedGrants(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []struct {
		name   string
		policy string
		rules  []identity.OperationRule
	}{
		{
			name:   "unrelated-agent-management",
			policy: "agent-management",
			rules: []identity.OperationRule{{
				Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run-123"},
			}},
		},
		{
			name:   "mixed-preparation-and-agent-management",
			policy: "team-preparation+agent-management",
			rules: []identity.OperationRule{
				{Policy: "team-preparation", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}},
				{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run-123"}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := authn.RequireCapability(r.Context(), "prompt-manager:read"); err != nil {
					http.Error(w, "prompt-manager read capability required", http.StatusForbidden)
					return
				}
				entered = true
				w.WriteHeader(http.StatusNoContent)
			})
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(inner))
			req := httptest.NewRequest(http.MethodPost, skillsconnect.SkillsServiceReadSkillsProcedure, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			grant := &identity.OperationGrant{
				ID: "mixed-reviewed-grant", Policy: tc.policy, Approver: "owner", ExpiresAt: time.Now().Add(time.Hour), Rules: tc.rules,
			}
			req = req.WithContext(provenance.NewContext(req.Context(), provenance.Provenance{
				Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
				RunID: "run-123", Subject: "owner", Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, Grant: grant,
			}))
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if !entered || rec.Code != http.StatusNoContent {
				t.Fatalf("%s grant changed static skill-read path: entered=%v status=%d body=%s", tc.name, entered, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestVerifiedAgentManagementGrantCanUseOrdinaryBootstrapReads(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	policy, ok := authn.InteractivePolicy("agent-management")
	if !ok {
		t.Fatal("agent-management policy is not registered")
	}
	grant := &identity.OperationGrant{
		ID: "agent-management", Policy: "agent-management", Approver: "owner",
		Operations: append([]string(nil), policy.Operations...), ExpiresAt: time.Now().Add(time.Hour),
	}
	for _, tc := range []struct {
		name, method, path string
		scopes             []string
	}{
		{name: "static skills", method: http.MethodPost, path: skillsconnect.SkillsServiceReadSkillsProcedure, scopes: append([]string(nil), policy.Scopes...)},
		{name: "catalog discovery", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, scopes: append([]string(nil), policy.Scopes...)},
		{name: "semantic skill catalog search", method: http.MethodPost, path: aisearchconnect.AISearchServiceSearchSkillsProcedure, scopes: append([]string(nil), policy.Scopes...)},
		{name: "liveness", method: http.MethodGet, path: "/health", scopes: append([]string(nil), policy.Scopes...)},
		{name: "readiness", method: http.MethodGet, path: "/api/v1/health", scopes: append([]string(nil), policy.Scopes...)},
		{name: "zero-scope static skills", method: http.MethodPost, path: skillsconnect.SkillsServiceReadSkillsProcedure},
		{name: "zero-scope catalog discovery", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure},
		{name: "zero-scope semantic skill catalog search", method: http.MethodPost, path: aisearchconnect.AISearchServiceSearchSkillsProcedure},
		{name: "zero-scope health", method: http.MethodGet, path: "/health"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			})))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			req = req.WithContext(provenance.NewContext(req.Context(), provenance.Provenance{
				Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
				RunID: "run-agent-management", Subject: "owner", Scopes: append([]string(nil), tc.scopes...), Grant: grant,
			}))
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if !entered || rec.Code != http.StatusNoContent {
				t.Fatalf("verified identity should retain ordinary-read access: scopes=%v entered=%v status=%d body=%s", tc.scopes, entered, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBootstrapReadExceptionRequiresVerifiedAgentIdentity(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	policy, ok := authn.InteractivePolicy("agent-management")
	if !ok {
		t.Fatal("agent-management policy is not registered")
	}
	grant := &identity.OperationGrant{ID: "agent-management", Policy: "agent-management", Approver: "owner", Operations: append([]string(nil), policy.Operations...), ExpiresAt: time.Now().Add(time.Hour)}
	for _, tc := range []struct {
		name      string
		principal identity.Principal
	}{
		{name: "invalid agent", principal: identity.Principal{Kind: identity.ActorAgent, Subject: "run", Scopes: append(append([]string(nil), policy.Scopes...), "prompt-manager:read"), Grant: grant}},
		{name: "unknown actor", principal: identity.Principal{Kind: identity.ActorUnknown, Scopes: []string{"prompt-manager:read"}, Grant: grant}},
		{name: "anonymous", principal: identity.Principal{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := identity.WithPrincipal(context.Background(), tc.principal)
			if IsVerifiedIdentity(ctx) {
				t.Fatal("unverified or anonymous principal entered ordinary-read exception")
			}
		})
	}
}

func TestBoundedCaptureBugRequiresVerifiedIdentity(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	grant := &identity.OperationGrant{
		ID: "unrelated", Policy: "agent-management", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
		Rules: []identity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run-123"}}},
	}
	for _, tc := range []struct {
		name string
		ctx  func(context.Context) context.Context
	}{
		{
			name: "unverified principal with request authority",
			ctx: func(ctx context.Context) context.Context {
				ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorAgent, Subject: "run-123", Grant: grant})
				return authn.WithRequestOperationAuthority(ctx, denyAuthority{})
			},
		},
		{
			name: "authority without principal",
			ctx: func(ctx context.Context) context.Context {
				return authn.WithRequestOperationAuthority(ctx, denyAuthority{})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			wire := PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodPost, heartbeatconnect.HeartbeatServiceCaptureBugProcedure, strings.NewReader(`{"team_id":"scenario-qa"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			req = req.WithContext(tc.ctx(req.Context()))
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if entered || rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "permission_denied") {
				t.Fatalf("unverified capture caller was admitted: entered=%v status=%d body=%s", entered, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPreparationGrantAllowsOnlyReadOnlyBootstrapHealth(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []struct {
		method, path  string
		read, allowed bool
	}{
		{"GET", "/health", true, true}, {"GET", "/api/v1/health", true, true},
		{"GET", "/health", false, true}, {"POST", "/health", true, false},
		{"PUT", "/api/v1/health", true, false}, {"GET", "/health/other", true, false},
		{"GET", "/teams/draft", true, false}, {"GET", "/unknown", true, false},
	} {
		t.Run(tc.method+tc.path+map[bool]string{true: "read", false: "no-read"}[tc.read], func(t *testing.T) {
			entered := false
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered = true; w.WriteHeader(http.StatusNoContent) })))
			scopes := []string{"prompt-manager:write"}
			if tc.read {
				scopes = append(scopes, "prompt-manager:read")
			}
			grant := &identity.OperationGrant{ID: "approved", Policy: "team-preparation", Approver: "owner", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour)}
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req = req.WithContext(provenance.NewContext(req.Context(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Scopes: scopes, Grant: grant}))
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if entered != tc.allowed || (!tc.allowed && rec.Code != http.StatusForbidden) {
				t.Fatalf("bootstrap boundary mismatch: entered=%v code=%d", entered, rec.Code)
			}
		})
	}
}

func TestBoundedKnowledgeOperationsUseVerifiedIdentityWithoutNewGrant(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []struct {
		name     string
		method   string
		path     string
		verified bool
		allowed  bool
	}{
		{name: "read with unrelated grant", method: http.MethodPost, path: teamsconnect.TeamsServiceListKnowledgeProcedure, verified: true, allowed: true},
		{name: "attributed knowledge append with unrelated grant", method: http.MethodPost, path: teamsconnect.TeamsServiceAddKnowledgeProcedure, verified: true, allowed: true},
		{name: "attributed closure bookkeeping with unrelated grant", method: http.MethodPost, path: teamsconnect.TeamsServiceCloseKnowledgeProcedure, verified: true, allowed: true},
		{name: "unverified caller cannot read", method: http.MethodPost, path: teamsconnect.TeamsServiceListKnowledgeProcedure},
		{name: "unverified caller cannot append", method: http.MethodPost, path: teamsconnect.TeamsServiceAddKnowledgeProcedure},
		{name: "unverified caller cannot close", method: http.MethodPost, path: teamsconnect.TeamsServiceCloseKnowledgeProcedure},
		{name: "wrong method", method: http.MethodPut, path: teamsconnect.TeamsServiceListKnowledgeProcedure, verified: true},
		{name: "append is POST-only", method: http.MethodPut, path: teamsconnect.TeamsServiceAddKnowledgeProcedure, verified: true},
		{name: "close is POST-only", method: http.MethodPut, path: teamsconnect.TeamsServiceCloseKnowledgeProcedure, verified: true},
		{name: "ordinary exception does not admit knowledge update", method: http.MethodPost, path: teamsconnect.TeamsServiceUpdateKnowledgeProcedure, verified: true},
		{name: "ordinary exception does not admit knowledge delete", method: http.MethodPost, path: teamsconnect.TeamsServiceDeleteKnowledgeProcedure, verified: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grant := &identity.OperationGrant{ID: "unrelated", Policy: "agent-management", Approver: "owner", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}
			ctx := context.Background()
			if tc.verified {
				ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Grant: grant})
			} else {
				ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorAgent, Subject: "run", Grant: grant})
				ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
			}
			entered := false
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { entered = true; w.WriteHeader(http.StatusNoContent) })))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if entered != tc.allowed || (!tc.allowed && rec.Code != http.StatusForbidden) || (tc.allowed && rec.Code != http.StatusNoContent) {
				t.Fatalf("knowledge read boundary: entered=%v status=%d body=%s", entered, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBoundedSkillCatalogDiscoveryUsesVerifiedIdentityOnly(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []struct {
		name             string
		method           string
		path             string
		verified         bool
		bounded          bool
		requestAuthority bool
		allowed          bool
	}{
		{name: "semantic skill search with verified identity", method: http.MethodPost, path: aisearchconnect.AISearchServiceSearchSkillsProcedure, verified: true, allowed: true},
		{name: "catalog discover with verified identity", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, verified: true, allowed: true},
		{name: "verified baseline with request authority", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, verified: true, requestAuthority: true, allowed: true},
		{name: "semantic skill search with unrelated zero-scope grant", method: http.MethodPost, path: aisearchconnect.AISearchServiceSearchSkillsProcedure, verified: true, bounded: true, allowed: true},
		{name: "catalog discover with unrelated zero-scope grant", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, verified: true, bounded: true, allowed: true},
		{name: "unverified bounded semantic search", method: http.MethodPost, path: aisearchconnect.AISearchServiceSearchSkillsProcedure, bounded: true},
		{name: "unverified bounded discover", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, bounded: true},
		{name: "wrong method", method: http.MethodPut, path: aisearchconnect.AISearchServiceSearchSkillsProcedure, verified: true, bounded: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ctx context.Context = context.Background()
			grant := &identity.OperationGrant{ID: "unrelated", Policy: "agent-management", Approver: "owner", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}
			if tc.verified {
				var activeGrant *identity.OperationGrant
				if tc.bounded {
					activeGrant = grant
				}
				ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Grant: activeGrant})
				if tc.requestAuthority {
					ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
				}
			} else {
				if tc.bounded {
					ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorAgent, Subject: "run", Grant: grant})
					ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
				} else {
					ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorAgent, Subject: "run"})
				}
			}
			entered := false
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			})))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if entered != tc.allowed || (tc.allowed && rec.Code != http.StatusNoContent) || (!tc.allowed && rec.Code != http.StatusForbidden) {
				t.Fatalf("catalog read boundary: entered=%v status=%d body=%s, allowed=%v", entered, rec.Code, rec.Body.String(), tc.allowed)
			}
		})
	}
}

func TestBoundedPromptPreviewReadsUseVerifiedIdentityWithoutMutationGrant(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []struct {
		name     string
		method   string
		path     string
		verified bool
		allowed  bool
	}{
		{name: "full preview with unrelated grant", method: http.MethodPost, path: heartbeatconnect.HeartbeatServicePreviewPromptProcedure, verified: true, allowed: true},
		{name: "structured preview with unrelated grant", method: http.MethodPost, path: heartbeatconnect.HeartbeatServicePreviewPromptStructuredProcedure, verified: true, allowed: true},
		{name: "unverified caller", method: http.MethodPost, path: heartbeatconnect.HeartbeatServicePreviewPromptProcedure},
		{name: "wrong method", method: http.MethodPut, path: heartbeatconnect.HeartbeatServicePreviewPromptProcedure, verified: true},
		{name: "unknown heartbeat RPC remains closed", method: http.MethodPost, path: "/vrooli.prompt_manager.v1.heartbeat.HeartbeatService/FutureRead", verified: true},
		{name: "run creation remains closed", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceCreateRunProcedure, verified: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grant := &identity.OperationGrant{ID: "unrelated", Policy: "agent-management", Approver: "owner", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}
			ctx := context.Background()
			if tc.verified {
				ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Grant: grant})
			} else {
				ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorAgent, Subject: "run", Grant: grant})
				ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
			}
			entered := false
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { entered = true; w.WriteHeader(http.StatusNoContent) })))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if entered != tc.allowed || (!tc.allowed && rec.Code != http.StatusForbidden) || (tc.allowed && rec.Code != http.StatusNoContent) {
				t.Fatalf("prompt preview boundary: entered=%v status=%d body=%s", entered, rec.Code, rec.Body.String())
			}
		})
	}
}
