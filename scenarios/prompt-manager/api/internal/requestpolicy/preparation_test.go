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
	actionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/actions/actions_v1connect"
	agentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents"
	agentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents/agents_v1connect"
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
	for _, file := range []protoreflect.FileDescriptor{heartbeatv1.File_prompt_manager_v1_heartbeat_heartbeat_proto, teamsv1.File_prompt_manager_v1_teams_teams_proto, agentsv1.File_prompt_manager_v1_agents_agents_proto, discoveryv1.File_prompt_manager_v1_discovery_discovery_proto, aisearchv1.File_prompt_manager_v1_aisearch_aisearch_proto} {
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				paths = append(paths, "/"+string(service.FullName())+"/"+string(service.Methods().Get(j).Name()))
			}
		}
	}
	allowedPaths := map[string]bool{
		discoveryconnect.DiscoveryServiceDiscoverProcedure:                 true,
		aisearchconnect.AISearchServiceSearchSkillsProcedure:               true,
		teamsconnect.TeamsServiceListKnowledgeProcedure:                    true,
		teamsconnect.TeamsServiceGetTeamProcedure:                          true,
		teamsconnect.TeamsServiceGetRolesProcedure:                         true,
		teamsconnect.TeamsServiceAddKnowledgeProcedure:                     true,
		teamsconnect.TeamsServiceCloseKnowledgeProcedure:                   true,
		heartbeatconnect.HeartbeatServiceGetHeartbeatProcedure:             true,
		heartbeatconnect.HeartbeatServiceGetHeartbeatInstructionsProcedure: true,
		heartbeatconnect.HeartbeatServiceListHeartbeatsProcedure:           true,
		heartbeatconnect.HeartbeatServiceUpdateHeartbeatProcedure:          true,
		heartbeatconnect.HeartbeatServiceCreateHeartbeatProcedure:          true,
		heartbeatconnect.HeartbeatServiceDeleteHeartbeatProcedure:          true,
		heartbeatconnect.HeartbeatServiceSetHeartbeatInstructionsProcedure: true,
		heartbeatconnect.HeartbeatServiceCaptureBugProcedure:               true,
		heartbeatconnect.HeartbeatServiceClearTeamQueueRunningProcedure:    true,
		heartbeatconnect.HeartbeatServicePreviewPromptProcedure:            true,
		heartbeatconnect.HeartbeatServicePreviewPromptStructuredProcedure:  true,
		heartbeatconnect.HeartbeatServiceListHeartbeatAttemptsProcedure:    true,
		heartbeatconnect.HeartbeatServiceGetTeamExecutionStatusProcedure:   true,
		heartbeatconnect.HeartbeatServiceGetTaskBoardProcedure:             true,
		heartbeatconnect.HeartbeatServiceGetTeamHeartbeatControlProcedure:  true,
		heartbeatconnect.HeartbeatServiceGetHeartbeatControlProcedure:      true,
		heartbeatconnect.HeartbeatServiceGetHandoffHistoryProcedure:        true,
		heartbeatconnect.HeartbeatServiceListLogsProcedure:                 true,
		heartbeatconnect.HeartbeatServiceGetLogProcedure:                   true,
		teamsconnect.TeamsServiceUpdateTeamProcedure:                       true,
		teamsconnect.TeamsServiceCreateTeamProcedure:                       true,
		teamsconnect.TeamsServiceSendMessageProcedure:                      true,
		teamsconnect.TeamsServiceUpdateKnowledgeProcedure:                  false,
		agentsconnect.AgentsServiceListAgentsProcedure:                     true,
		agentsconnect.AgentsServiceGetAgentProcedure:                       true,
		agentsconnect.AgentsServiceCreateAgentProcedure:                    true,
		teamsconnect.TeamsServiceAddMemberProcedure:                        true,
		teamsconnect.TeamsServiceUpdateMemberProcedure:                     true,
		teamsconnect.TeamsServiceRemoveMemberProcedure:                     true,
		// The boundary admits this declared owner-authorized route; SetTeamRoles
		// still requires its exact team.manage target before replacing roles.
		teamsconnect.TeamsServiceSetRolesProcedure: true,
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

func TestGlobalHeartbeatControlReadRequiresVerifiedIdentityAndLeavesMutationsClosed(t *testing.T) {
	grant := &identity.OperationGrant{
		ID: "unrelated", Policy: "agent-management", Approver: "owner",
		Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/other"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	verifiedGrant := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "verified-run", Verified: true,
		Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: grant,
	})
	for _, tc := range []struct {
		name    string
		path    string
		ctx     context.Context
		allowed bool
	}{
		{name: "verified identity with unrelated grant can read status", path: heartbeatconnect.HeartbeatServiceGetHeartbeatControlProcedure, ctx: verifiedGrant, allowed: true},
		{name: "unverified bounded caller cannot read status", path: heartbeatconnect.HeartbeatServiceGetHeartbeatControlProcedure, ctx: authn.WithRequestOperationAuthority(context.Background(), denyAuthority{}), allowed: false},
		{name: "grant cannot pause heartbeats", path: heartbeatconnect.HeartbeatServicePauseHeartbeatControlProcedure, ctx: verifiedGrant, allowed: false},
		{name: "grant cannot resume heartbeats", path: heartbeatconnect.HeartbeatServiceResumeHeartbeatControlProcedure, ctx: verifiedGrant, allowed: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			handler := PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{}`)).WithContext(tc.ctx)
			req.Header.Set("Connect-Protocol-Version", "1")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if entered != tc.allowed {
				t.Fatalf("handler entered=%v want=%v status=%d body=%q", entered, tc.allowed, rec.Code, rec.Body.String())
			}
			if tc.allowed && rec.Code != http.StatusNoContent {
				t.Fatalf("read status=%d body=%q", rec.Code, rec.Body.String())
			}
			if !tc.allowed && rec.Code != http.StatusForbidden {
				t.Fatalf("refused status=%d body=%q, want 403", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSetHeartbeatInstructionsUsesOnlyItsConnectPostAndLegacyPutRoutes(t *testing.T) {
	procedure := heartbeatconnect.HeartbeatServiceSetHeartbeatInstructionsProcedure
	allowed := map[routeKey]bool{
		{method: http.MethodPost, path: procedure}:                                                             true,
		{method: http.MethodPut, path: "/teams/team-1/members/bug-investigator/heartbeat-instructions"}:        true,
		{method: http.MethodPut, path: "/api/v1/teams/team-1/members/bug-investigator/heartbeat-instructions"}: true,
	}
	ctx := authn.WithRequestOperationAuthority(context.Background(), denyAuthority{})
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, procedure},
		{http.MethodGet, procedure},
		{http.MethodPut, procedure},
		{http.MethodDelete, procedure},
		{http.MethodPut, "/teams/team-1/members/bug-investigator/heartbeat-instructions"},
		{http.MethodPut, "/api/v1/teams/team-1/members/bug-investigator/heartbeat-instructions"},
		{http.MethodPost, "/teams/team-1/members/bug-investigator/heartbeat-instructions"},
		{http.MethodPut, "/teams/team-1/members/bug-investigator/heartbeat-instructions/"},
		{http.MethodPut, "/api/v1teams/team-1/members/bug-investigator/heartbeat-instructions"},
		{http.MethodPut, "/teams/team-1/members/bug-investigator/responsibilities"},
	} {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			entered := false
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`)).WithContext(ctx)
			rec := httptest.NewRecorder()
			PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(rec, req)
			want := allowed[routeKey{method: tc.method, path: tc.path}]
			if entered != want {
				t.Fatalf("handler entered=%v want=%v status=%d body=%q", entered, want, rec.Code, rec.Body.String())
			}
			if !want && rec.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%q, want 403", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestGetHeartbeatInstructionsUsesOrdinaryPMReadRoute(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "verified-run", Verified: true,
		Scopes: []string{"prompt-manager:read"},
	})
	ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
	allowed := map[routeKey]bool{
		{method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetHeartbeatInstructionsProcedure}:    true,
		{method: http.MethodGet, path: "/teams/team-1/members/bug-investigator/heartbeat-instructions"}:        true,
		{method: http.MethodGet, path: "/api/v1/teams/team-1/members/bug-investigator/heartbeat-instructions"}: true,
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, heartbeatconnect.HeartbeatServiceGetHeartbeatInstructionsProcedure},
		{http.MethodGet, heartbeatconnect.HeartbeatServiceGetHeartbeatInstructionsProcedure},
		{http.MethodGet, "/teams/team-1/members/bug-investigator/heartbeat-instructions"},
		{http.MethodGet, "/api/v1/teams/team-1/members/bug-investigator/heartbeat-instructions"},
		{http.MethodPost, "/teams/team-1/members/bug-investigator/heartbeat-instructions"},
		{http.MethodGet, "/teams/team-1/members/bug-investigator/responsibilities"},
	} {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			entered := false
			req := httptest.NewRequest(tc.method, tc.path, nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(rec, req)
			want := allowed[routeKey{method: tc.method, path: tc.path}]
			if entered != want {
				t.Fatalf("handler entered=%v want=%v status=%d body=%q", entered, want, rec.Code, rec.Body.String())
			}
			if !want && rec.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%q, want 403", rec.Code, rec.Body.String())
			}
		})
	}

	withoutRead := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "verified-run", Verified: true,
		Scopes: []string{"prompt-manager:write"},
	})
	withoutRead = authn.WithRequestOperationAuthority(withoutRead, denyAuthority{})
	rec := httptest.NewRecorder()
	PreparationBoundary(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler entered without prompt-manager:read") })).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, heartbeatconnect.HeartbeatServiceGetHeartbeatInstructionsProcedure, nil).WithContext(withoutRead))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing read capability status=%d body=%q, want 403", rec.Code, rec.Body.String())
	}
}

func TestUnknownBoundedRouteRefusalIsNotAnApprovalRequest(t *testing.T) {
	grant := &identity.OperationGrant{
		ID: "unrelated", Approver: "owner", Operations: []string{"team.manage"},
		Targets: []string{"prompt-manager:team/scenario-qa"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "run-123", Owner: "owner", Verified: true,
		Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, Grant: grant,
	})
	called := false
	next := PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	r := httptest.NewRequest(http.MethodPost, "/unknown.Service/NewMutation", strings.NewReader("{}"))
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Connect-Protocol-Version", "1")
	w := httptest.NewRecorder()
	next.ServeHTTP(w, r)
	if called || w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "no bounded request contract is declared") || strings.Contains(w.Body.String(), authn.ErrOperation.Error()) {
		t.Fatalf("unknown bounded route was misreported as a grant problem: called=%v status=%d body=%q", called, w.Code, w.Body.String())
	}
}

func TestBoundedOrdinaryTeamReadsRequireVerifiedIdentityAndKeepMutationsClosed(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	grant := &identity.OperationGrant{
		ID: "unrelated-agent-management", Policy: "agent-management", Approver: "owner",
		ExpiresAt: time.Now().Add(time.Hour),
		Rules:     []identity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run-123"}}},
	}
	verified := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "run-123", Subject: "owner", Scopes: []string{"prompt-manager:read"}, Grant: grant,
	})
	verifiedReadWithoutGrant := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "run-123", Subject: "owner", Scopes: []string{"prompt-manager:read"},
	})
	verifiedWithoutRead := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "run-123", Subject: "owner", Scopes: []string{"prompt-manager:write"}, Grant: grant,
	})
	verifiedOperator := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator", Verified: true,
		Scopes: []string{"prompt-manager:write"}, Grant: grant,
	})
	unverified := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "run-123", Grant: grant})
	unverified = authn.WithRequestOperationAuthority(unverified, denyAuthority{})
	cases := []struct {
		name, method, path string
		ctx                context.Context
		allow              bool
	}{
		{name: "task board is a verified read", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetTaskBoardProcedure, ctx: verified, allow: true},
		{name: "team heartbeat control status is a verified read", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetTeamHeartbeatControlProcedure, ctx: verified, allow: true},
		{name: "handoff history is a verified read", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetHandoffHistoryProcedure, ctx: verified, allow: true},
		{name: "handoff history uses verified identity without an operation grant", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetHandoffHistoryProcedure, ctx: provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-123", Subject: "owner"}), allow: true},
		{name: "unverified handoff-history caller is refused", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetHandoffHistoryProcedure, ctx: unverified},
		{name: "wrong method for handoff-history is refused", method: http.MethodGet, path: heartbeatconnect.HeartbeatServiceGetHandoffHistoryProcedure, ctx: verified},
		{name: "unverified task-board caller is refused", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetTaskBoardProcedure, ctx: unverified},
		{name: "unverified control-status caller is refused", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetTeamHeartbeatControlProcedure, ctx: unverified},
		{name: "verified member log list is an ordinary read", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceListLogsProcedure, ctx: verified, allow: true},
		{name: "verified member log list needs no operation grant", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceListLogsProcedure, ctx: verifiedReadWithoutGrant, allow: true},
		{name: "verified member log list still requires baseline read", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceListLogsProcedure, ctx: verifiedWithoutRead},
		{name: "verified member log detail is an ordinary read", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetLogProcedure, ctx: verified, allow: true},
		{name: "verified member log detail needs no operation grant", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetLogProcedure, ctx: verifiedReadWithoutGrant, allow: true},
		{name: "verified member log detail still requires baseline read", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetLogProcedure, ctx: verifiedWithoutRead},
		{name: "exact team.manage grant reaches team read owner", method: http.MethodPost, path: teamsconnect.TeamsServiceGetTeamProcedure, ctx: provenance.NewContext(context.Background(), provenance.Provenance{
			Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
			RunID: "run-123", Subject: "owner", Grant: &identity.OperationGrant{
				ID: "team-manager", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
				Rules: []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/draft"}}},
			},
		}), allow: true},
		{name: "exact team.manage grant reaches role replacement owner", method: http.MethodPost, path: teamsconnect.TeamsServiceSetRolesProcedure, ctx: provenance.NewContext(context.Background(), provenance.Provenance{
			Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
			RunID: "run-123", Subject: "owner", Grant: &identity.OperationGrant{
				ID: "team-manager", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
				Rules: []identity.OperationRule{{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/team-1"}}},
			},
		}), allow: true},
		{name: "unverified member log list is refused", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceListLogsProcedure, ctx: unverified},
		{name: "unverified member log detail is refused", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceGetLogProcedure, ctx: unverified},
		{name: "wrong method for member log list is refused", method: http.MethodGet, path: heartbeatconnect.HeartbeatServiceListLogsProcedure, ctx: verified},
		{name: "wrong method for member log detail is refused", method: http.MethodGet, path: heartbeatconnect.HeartbeatServiceGetLogProcedure, ctx: verified},
		{name: "unknown RPC remains closed", method: http.MethodPost, path: "/vrooli.prompt_manager.v1.heartbeat.HeartbeatService/FutureRead", ctx: verified},
		{name: "task creation remains closed", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceAddTaskProcedure, ctx: verified},
		{name: "team pause remains closed", method: http.MethodPost, path: heartbeatconnect.HeartbeatServicePauseTeamHeartbeatControlProcedure, ctx: verified},
		{name: "team control-policy mutation remains closed", method: http.MethodPost, path: heartbeatconnect.HeartbeatServiceUpdateTeamHeartbeatControlPolicyProcedure, ctx: verified},
		{name: "verified identity can reach message owner check", method: http.MethodPost, path: teamsconnect.TeamsServiceSendMessageProcedure, ctx: verified, allow: true},
		{name: "verified operator with baseline write remains eligible", method: http.MethodPost, path: teamsconnect.TeamsServiceSendMessageProcedure, ctx: verifiedOperator, allow: true},
		{name: "unverified bounded message sender is refused", method: http.MethodPost, path: teamsconnect.TeamsServiceSendMessageProcedure, ctx: unverified},
		{name: "wrong method for message send is refused", method: http.MethodPut, path: teamsconnect.TeamsServiceSendMessageProcedure, ctx: verified},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			})))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`)).WithContext(tc.ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if entered != tc.allow {
				t.Fatalf("handler entered=%v, want %v; status=%d body=%s", entered, tc.allow, rec.Code, rec.Body.String())
			}
			if tc.allow && rec.Code != http.StatusNoContent {
				t.Fatalf("verified read returned status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !tc.allow && (rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "permission_denied")) {
				t.Fatalf("bounded refusal lost: status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListHeartbeatsUsesBaselineReadCapability(t *testing.T) {
	route := heartbeatconnect.HeartbeatServiceListHeartbeatsProcedure
	unrelatedGrant := &identity.OperationGrant{
		ID: "unrelated-agent-management", Policy: "agent-management", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
		Rules: []identity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run-123"}}},
	}
	verified := func(scopes []string, grant *identity.OperationGrant) context.Context {
		return identity.WithPrincipal(context.Background(), identity.Principal{
			Kind: identity.ActorAgent, Subject: "verified-agent", Verified: true, Scopes: scopes, Grant: grant,
		})
	}
	withoutIdentity := context.Background()

	for _, tc := range []struct {
		name   string
		method string
		ctx    context.Context
		allow  bool
	}{
		{name: "unrelated grant preserves baseline read", method: http.MethodPost, ctx: verified([]string{"prompt-manager:read"}, unrelatedGrant), allow: true},
		{name: "unrelated request authority preserves baseline read", method: http.MethodPost, ctx: authn.WithRequestOperationAuthority(verified([]string{"prompt-manager:read"}, nil), denyAuthority{}), allow: true},
		{name: "missing baseline capability denied", method: http.MethodPost, ctx: verified(nil, unrelatedGrant)},
		{name: "missing identity denied", method: http.MethodPost, ctx: authn.WithRequestOperationAuthority(withoutIdentity, denyAuthority{})},
		{name: "wrong method denied", method: http.MethodGet, ctx: verified([]string{"prompt-manager:read"}, unrelatedGrant)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			handler := PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(tc.method, route, strings.NewReader("{}")).WithContext(tc.ctx)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if entered != tc.allow {
				t.Fatalf("handler entered=%v, want %v; status=%d body=%s", entered, tc.allow, w.Code, w.Body.String())
			}
			if tc.allow && w.Code != http.StatusNoContent {
				t.Fatalf("allowed read returned status=%d body=%s", w.Code, w.Body.String())
			}
			if !tc.allow && w.Code != http.StatusForbidden {
				t.Fatalf("denied request returned status=%d, want 403; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestBoundedSkillProjectionRefreshPreservesOrdinaryVerifiedRoute(t *testing.T) {
	grant := &identity.OperationGrant{
		ID: "unrelated-agent-management", Policy: "agent-management", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
		Rules: []identity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run-123"}}},
	}
	verified := identity.Principal{
		Kind: identity.ActorAgent, Subject: "verified-agent", Verified: true,
		Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, Grant: grant,
	}
	requestAuthority := authn.WithRequestOperationAuthority(identity.WithPrincipal(context.Background(), verified), denyAuthority{})
	unverifiedBounded := authn.WithRequestOperationAuthority(identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "unverified", Grant: grant}), denyAuthority{})
	cases := []struct {
		name, method, path string
		ctx                context.Context
		allowed            bool
	}{
		{
			name:   "unrelated interactive grant does not narrow refresh",
			method: http.MethodPost, path: skillsconnect.SkillsServiceRefreshProjectionProcedure,
			ctx: identity.WithPrincipal(context.Background(), verified), allowed: true,
		},
		{
			name:   "unrelated request authority does not narrow refresh",
			method: http.MethodPost, path: skillsconnect.SkillsServiceRefreshProjectionProcedure,
			ctx: requestAuthority, allowed: true,
		},
		{
			name:   "unverified bounded caller is refused",
			method: http.MethodPost, path: skillsconnect.SkillsServiceRefreshProjectionProcedure,
			ctx: unverifiedBounded,
		},
		{
			name:   "unrelated skill mutation remains closed",
			method: http.MethodPost, path: skillsconnect.SkillsServiceDeleteSkillProcedure,
			ctx: identity.WithPrincipal(context.Background(), verified),
		},
		{
			name:   "unknown route remains closed",
			method: http.MethodPost, path: "/unknown.Service/NewMutation",
			ctx: requestAuthority,
		},
		{
			name:   "wrong method remains closed",
			method: http.MethodGet, path: skillsconnect.SkillsServiceRefreshProjectionProcedure,
			ctx: identity.WithPrincipal(context.Background(), verified),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			handler := PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			req = req.WithContext(tc.ctx)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if entered != tc.allowed {
				t.Fatalf("handler entered=%v, allowed=%v, status=%d, body=%s", entered, tc.allowed, rec.Code, rec.Body.String())
			}
			if tc.allowed && rec.Code != http.StatusNoContent {
				t.Fatalf("allowed ordinary route returned status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !tc.allowed && (rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "permission_denied")) {
				t.Fatalf("refused route returned status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
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
	t.Run("exact team management target", func(t *testing.T) {
		ctx := identity.WithPrincipal(context.Background(), identity.Principal{
			Kind: identity.ActorAgent, Subject: "verified-agent", Verified: true,
			Scopes: []string{"prompt-manager:read", "prompt-manager:write"},
			Grant: &identity.OperationGrant{
				ID: "combined-management", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
				Rules: []identity.OperationRule{
					{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/draft"}},
					{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"prompt-manager:team/other"}},
				},
			},
		})
		if err := RequirePreparationRead(ctx, "draft"); err != nil {
			t.Fatalf("exact team.manage grant could not read its managed team: %v", err)
		}
		if err := RequirePreparationRead(ctx, "other"); err == nil {
			t.Fatal("team.manage operation was paired with a target from a different grant rule")
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

func TestCommissioningGrantsCannotReplaceKnowledge(t *testing.T) {
	for _, policyName := range []string{"prompt-agent-creation", "team-management"} {
		t.Run(policyName, func(t *testing.T) {
			policy, ok := authn.InteractivePolicy(policyName)
			if !ok {
				t.Fatal("missing canonical policy")
			}
			grant := &identity.OperationGrant{ID: "reviewed", Policy: policyName, Approver: "owner", Operations: policy.Operations, Targets: []string{"prompt-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}
			p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: policy.Scopes, Grant: grant}
			req := httptest.NewRequest(http.MethodPost, teamsconnect.TeamsServiceUpdateKnowledgeProcedure, strings.NewReader(`{}`)).WithContext(identity.WithPrincipal(context.Background(), p))
			rec := httptest.NewRecorder()
			effects := 0
			PreparationBoundary(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { effects++ })).ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden || effects != 0 {
				t.Fatalf("grant reached unrelated knowledge mutation: status=%d effects=%d", rec.Code, effects)
			}
		})
	}
	// Unrelated supplemental grants do not replace ordinary identity. The
	// ordinary corpus owner still receives requests without a bounded principal.
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "member", Verified: true, Scopes: []string{"prompt-manager:write"}}
	req := httptest.NewRequest(http.MethodPost, teamsconnect.TeamsServiceUpdateKnowledgeProcedure, strings.NewReader(`{}`)).WithContext(identity.WithPrincipal(context.Background(), p))
	rec := httptest.NewRecorder()
	effects := 0
	PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { effects++; w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || effects != 1 {
		t.Fatalf("ordinary caller changed: status=%d effects=%d", rec.Code, effects)
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
		{"GET", "/health", true, true},
		{"GET", "/api/v1/health", true, true},
		{"GET", "/health", false, true},
		{"POST", "/health", true, false},
		{"PUT", "/api/v1/health", true, false},
		{"GET", "/health/other", true, false},
		{"GET", "/teams/draft", true, false},
		{"GET", "/unknown", true, false},
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
		{name: "list skills with verified identity", method: http.MethodPost, path: skillsconnect.SkillsServiceListSkillsProcedure, verified: true, allowed: true},
		{name: "get skill with verified identity", method: http.MethodPost, path: skillsconnect.SkillsServiceGetSkillProcedure, verified: true, allowed: true},
		{name: "sync skill content with verified identity", method: http.MethodPost, path: skillsconnect.SkillsServiceSyncSkillsProcedure, verified: true, allowed: true},
		{name: "list skills with unrelated zero-scope grant", method: http.MethodPost, path: skillsconnect.SkillsServiceListSkillsProcedure, verified: true, bounded: true, allowed: true},
		{name: "get skill with unrelated zero-scope grant", method: http.MethodPost, path: skillsconnect.SkillsServiceGetSkillProcedure, verified: true, bounded: true, allowed: true},
		{name: "sync skill content with unrelated zero-scope grant", method: http.MethodPost, path: skillsconnect.SkillsServiceSyncSkillsProcedure, verified: true, bounded: true, allowed: true},
		{name: "list skills with verified baseline and request authority", method: http.MethodPost, path: skillsconnect.SkillsServiceListSkillsProcedure, verified: true, requestAuthority: true, allowed: true},
		{name: "get skill with verified baseline and request authority", method: http.MethodPost, path: skillsconnect.SkillsServiceGetSkillProcedure, verified: true, requestAuthority: true, allowed: true},
		{name: "sync skills with verified baseline and request authority", method: http.MethodPost, path: skillsconnect.SkillsServiceSyncSkillsProcedure, verified: true, requestAuthority: true, allowed: true},
		{name: "verified baseline with request authority", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, verified: true, requestAuthority: true, allowed: true},
		{name: "semantic skill search with unrelated zero-scope grant", method: http.MethodPost, path: aisearchconnect.AISearchServiceSearchSkillsProcedure, verified: true, bounded: true, allowed: true},
		{name: "catalog discover with unrelated zero-scope grant", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, verified: true, bounded: true, allowed: true},
		{name: "unverified bounded semantic search", method: http.MethodPost, path: aisearchconnect.AISearchServiceSearchSkillsProcedure, bounded: true},
		{name: "unverified bounded discover", method: http.MethodPost, path: discoveryconnect.DiscoveryServiceDiscoverProcedure, bounded: true},
		{name: "unverified bounded list skills", method: http.MethodPost, path: skillsconnect.SkillsServiceListSkillsProcedure, bounded: true},
		{name: "unverified bounded get skill", method: http.MethodPost, path: skillsconnect.SkillsServiceGetSkillProcedure, bounded: true},
		{name: "unverified bounded sync skills", method: http.MethodPost, path: skillsconnect.SkillsServiceSyncSkillsProcedure, bounded: true},
		{name: "skill creation remains closed", method: http.MethodPost, path: skillsconnect.SkillsServiceCreateSkillProcedure, verified: true, bounded: true},
		{name: "skill deletion remains closed", method: http.MethodPost, path: skillsconnect.SkillsServiceDeleteSkillProcedure, verified: true, bounded: true},
		{name: "wrong method", method: http.MethodPut, path: skillsconnect.SkillsServiceGetSkillProcedure, verified: true, bounded: true},
		{name: "wrong service method remains closed", method: http.MethodPost, path: "/vrooli.prompt_manager.v1.skills.SkillsService/FutureRead", verified: true, bounded: true},
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

func TestManualMemberTriggerRequiresExistingRunCreateAuthorityWhenBounded(t *testing.T) {
	unrelatedTeamPrepare := &identity.OperationGrant{
		ID: "unrelated-team-preparation", Policy: "team-preparation", Approver: "owner",
		Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/scenario-qa"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	runCreate := &identity.OperationGrant{
		ID: "approved-run-create", Policy: "run-creation", Approver: "owner",
		Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	wrongTargetRunCreate := &identity.OperationGrant{
		ID: "wrong-target-run-create", Policy: "run-creation", Approver: "owner",
		Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/other"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	cases := []struct {
		name             string
		principal        identity.Principal
		denyRequestProof bool
		wantPass         bool
	}{
		{
			name: "team.prepare-only agent cannot use ambient service dispatch",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: unrelatedTeamPrepare,
			},
		},
		{
			name: "existing exact run.create grant admits without human browser proof",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: runCreate,
			},
			wantPass: true,
		},
		{
			name: "human baseline capability remains valid with unrelated bounded grant",
			principal: identity.Principal{
				Kind: identity.ActorHuman, Subject: "owner", Verified: true,
				Scopes: []string{"agent-manager:write"}, Grant: unrelatedTeamPrepare,
			},
			wantPass: true,
		},
		{
			name: "wrong-target run.create grant remains denied",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: wrongTargetRunCreate,
			},
		},
		{
			name: "expired run.create grant remains denied",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: &identity.OperationGrant{
					ID: "expired-run-create", Policy: "run-creation", Approver: "owner",
					Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(-time.Hour),
				},
			},
		},
		{
			name: "denied request-operation proof cannot be overridden by agent grant",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: runCreate,
			},
			denyRequestProof: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			effects := 0
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				effects++
				w.WriteHeader(http.StatusAccepted)
			})
			req := httptest.NewRequest(http.MethodPost, heartbeatconnect.HeartbeatServiceTriggerHeartbeatProcedure, strings.NewReader(`{}`))
			ctx := identity.WithPrincipal(req.Context(), tc.principal)
			if tc.denyRequestProof {
				ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
			}
			req = req.WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			rec := httptest.NewRecorder()
			PreparationBoundary(next).ServeHTTP(rec, req)
			if (effects == 1) != tc.wantPass {
				t.Fatalf("effects=%d, want pass=%v; status=%d body=%q", effects, tc.wantPass, rec.Code, rec.Body.String())
			}
			wantStatus := http.StatusForbidden
			if tc.wantPass {
				wantStatus = http.StatusAccepted
			}
			if rec.Code != wantStatus {
				t.Fatalf("status=%d, want %d: %s", rec.Code, wantStatus, rec.Body.String())
			}
		})
	}
}

func TestManualMemberTriggerContractDoesNotAdmitTeamFanoutOrWrongMethod(t *testing.T) {
	if got := contractForBoundedRoute(http.MethodPost, heartbeatconnect.HeartbeatServiceTriggerHeartbeatProcedure); got != runCreateAuthorizedRoute {
		t.Fatalf("member trigger route contract=%v, want exact run.create authority", got)
	}
	if got := contractForBoundedRoute(http.MethodPost, heartbeatconnect.HeartbeatServiceTriggerTeamProcedure); got != denyBoundedRoute {
		t.Fatalf("team fanout route contract=%v, want closed default", got)
	}
	if got := contractForBoundedRoute(http.MethodGet, heartbeatconnect.HeartbeatServiceTriggerHeartbeatProcedure); got != denyBoundedRoute {
		t.Fatalf("wrong method route contract=%v, want closed default", got)
	}
}

// A grant for a different operation must not remove ordinary action discovery.
func TestBoundedActionReadsPreserveOrdinaryVerifiedIdentity(t *testing.T) {
	grant := &identity.OperationGrant{ID: "unrelated", Approver: "owner", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/scenario-qa"}, ExpiresAt: time.Now().Add(time.Hour)}
	for _, ingress := range []string{"grant", "request-authority"} {
		for _, verified := range []bool{false, true} {
			principal := identity.Principal{Kind: identity.ActorAgent, Subject: "run-123", Owner: "owner", Verified: verified}
			if ingress == "grant" {
				principal.Grant = grant
			}
			ctx := identity.WithPrincipal(context.Background(), principal)
			// An unverified principal cannot establish grant boundedness by itself.
			// Retain an explicit request authority to test refusal at this boundary.
			if ingress == "request-authority" || !verified {
				ctx = authn.WithRequestOperationAuthority(ctx, denyAuthority{})
			}
			for _, route := range []struct {
				path string
				read bool
			}{
				{actionsconnect.ActionsServiceListActionsProcedure, true},
				{actionsconnect.ActionsServiceGetActionProcedure, true},
				{actionsconnect.ActionsServiceAuthorActionProcedure, false},
				{actionsconnect.ActionsServiceUpdateActionProcedure, false},
				{actionsconnect.ActionsServiceDeleteActionProcedure, false},
				{actionsconnect.ActionsServiceRunActionProcedure, false},
				{"/unknown.Service/NewMutation", false},
			} {
				t.Run(ingress+"/verified="+map[bool]string{true: "true", false: "false"}[verified]+route.path, func(t *testing.T) {
					entered := false
					req := httptest.NewRequest(http.MethodPost, route.path, strings.NewReader("{}")).WithContext(ctx)
					rec := httptest.NewRecorder()
					PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { entered = true; w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
					want := verified && route.read
					if entered != want {
						t.Fatalf("entered=%v want=%v status=%d body=%q", entered, want, rec.Code, rec.Body.String())
					}
					if !want && rec.Code != http.StatusForbidden {
						t.Fatalf("status=%d want403", rec.Code)
					}
				})
			}
		}
	}
}
