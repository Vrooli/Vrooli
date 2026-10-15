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
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type denyAuthority struct{}

func (denyAuthority) AuthorizeOperation(context.Context, identity.Principal, string, string, string) error {
	return authn.ErrOperation
}

func TestPreparationGrantCannotReachOtherMutationHandlers(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	paths := []string{"/unknown.Service/NewMutation", "/teams/draft", "/vrooli.prompt_manager.v1.skills.SkillsService/CreateSkill"}
	// Enumerate actual generated owner surfaces: new RPCs default to denied.
	for _, file := range []protoreflect.FileDescriptor{heartbeatv1.File_prompt_manager_v1_heartbeat_heartbeat_proto, teamsv1.File_prompt_manager_v1_teams_teams_proto} {
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				paths = append(paths, "/"+string(service.FullName())+"/"+string(service.Methods().Get(j).Name()))
			}
		}
	}
	for _, ingress := range []string{"interactive-grant", "request-authority"} {
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
				grant := &identity.OperationGrant{ID: "reviewed", Approver: "owner", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour)}
				if ingress == "request-authority" {
					grant = nil
					r = r.WithContext(authn.WithRequestOperationAuthority(r.Context(), denyAuthority{}))
				}
				r = r.WithContext(provenance.NewContext(r.Context(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, Grant: grant}))
				w := httptest.NewRecorder()
				wire.ServeHTTP(w, r)
				allowed := path == teamsconnect.TeamsServiceGetTeamProcedure || path == heartbeatconnect.HeartbeatServiceGetHeartbeatProcedure || path == heartbeatconnect.HeartbeatServiceUpdateHeartbeatProcedure
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

func TestPreparationBoundaryLeavesOrdinaryRequestsWithTheirOwner(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	entered := false
	w := httptest.NewRecorder()
	PreparationBoundary(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { entered = true })).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/legacy", nil))
	if !entered {
		t.Fatal("boundary changed ordinary legacy caller behavior")
	}
}

func TestPreparationGrantAllowsOnlyReadOnlyBootstrapHealth(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []struct {
		method, path  string
		read, allowed bool
	}{
		{"GET", "/health", true, true}, {"GET", "/api/v1/health", true, true},
		{"GET", "/health", false, false}, {"POST", "/health", true, false},
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
			grant := &identity.OperationGrant{ID: "approved", Approver: "owner", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour)}
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
