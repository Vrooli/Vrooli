package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
	domain "prompt-manager/internal/heartbeat"
	"prompt-manager/internal/requestpolicy"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
	"prompt-manager/internal/testutil/fixtures"
)

type refusedPreparationAuthority struct{}

func (refusedPreparationAuthority) AuthorizeOperation(context.Context, identity.Principal, string, string, string) error {
	return authn.ErrOperation
}

func TestFinitePreparationConnectUsesVerifiedActorAndPreservesRefusals(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []string{"approved", "spoofed-headers", "wrong-target", "request-authority", "mixed-activation", "ordinary-activation", "stale"} {
		t.Run(tc, func(t *testing.T) {
			root := t.TempDir()
			relations := store.NewFileRelationStore(root)
			teams := store.NewFileTeamStore(root, root, relations)
			team := fixtures.LeaderLedSingleProcessTeam("draft", "Draft", "leader", fixtures.WithEnabled(false))
			team.Runtime.Mode = teamconfig.RuntimeModeMultiProcess
			team.Coordination.MessagingMode = teamconfig.MessagingModeAsyncInbox
			team.Purpose = "delivery"
			team.Lifetime = "finite"
			team.EffortRefs = []string{"effort:draft"}
			if err := teams.Create(context.Background(), team); err != nil {
				t.Fatal(err)
			}
			if err := relations.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "draft", AgentID: "leader", Status: store.MemberStatusActive}); err != nil {
				t.Fatal(err)
			}
			binding := &teamconfig.FiniteLeader{EffortRef: "effort:draft", AcceptedRevision: "r1", CoordinatorPromptRef: "owner:goal", SourceRefs: []string{"owner:source"}}
			if err := teams.SetHeartbeatConfig(context.Background(), "draft", "leader", &store.HeartbeatConfig{FiniteLeader: binding, ProfileKey: "qualified", Schedule: "@hourly"}); err != nil {
				t.Fatal(err)
			}
			h := domain.NewHandlers(domain.HandlersDeps{TeamStore: teams, RelationStore: relations})
			path, mount := NewConnectMount(h)
			mux := http.NewServeMux()
			mux.Handle(path, mount)
			// Verified fixture provenance enters at the authority seam; arbitrary
			// request headers below remain unverified and cannot create this proof.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc != "spoofed-headers" {
					grant := &identity.OperationGrant{ID: "approved", Approver: "owner", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour)}
					if tc == "wrong-target" {
						grant.Targets = []string{"prompt-manager:team/other"}
					}
					if tc == "request-authority" {
						grant = nil
						r = r.WithContext(authn.WithRequestOperationAuthority(r.Context(), refusedPreparationAuthority{}))
					}
					r = r.WithContext(provenance.NewContext(r.Context(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "interactive-run", Subject: "owner", Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, Grant: grant}))
				}
				authn.Middleware(authn.Config{})(requestpolicy.PreparationBoundary(mux)).ServeHTTP(w, r)
			}))
			defer server.Close()
			client := heartbeatconnect.NewHeartbeatServiceClient(server.Client(), server.URL)
			if tc == "approved" {
				if _, err := client.GetHeartbeat(context.Background(), connect.NewRequest(&heartbeatv1.MemberRequest{TeamId: "draft", AgentId: "leader"})); err != nil {
					t.Fatal("bounded draft inspection failed", err)
				}
			}
			if tc == "wrong-target" || tc == "request-authority" {
				_, err := client.GetHeartbeat(context.Background(), connect.NewRequest(&heartbeatv1.MemberRequest{TeamId: "draft", AgentId: "leader"}))
				if connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatal("read escaped the selected target", err)
				}
			}
			next := *binding
			next.AcceptedRevision = "r2"
			raw, _ := json.Marshal(map[string]any{"finiteEffortTransition": map[string]any{"operation": "amend", "expectedRevision": "r1", "binding": next, "evidenceRef": "owner:feedback"}})
			var payload map[string]any
			json.Unmarshal(raw, &payload)
			if tc == "ordinary-activation" {
				delete(payload, "finiteEffortTransition")
				payload["enabled"] = true
			}
			if tc == "mixed-activation" {
				payload["enabled"] = true
			}
			if tc == "stale" {
				payload["finiteEffortTransition"].(map[string]any)["expectedRevision"] = "wrong"
			}
			body, _ := structpb.NewStruct(payload)
			req := connect.NewRequest(&heartbeatv1.MemberMutationRequest{TeamId: "draft", AgentId: "leader", Body: structpb.NewStructValue(body)})
			req.Header().Set("X-Agent-Identity-Token", "caller-text")
			req.Header().Set("X-Agent-ID", "interactive-run")
			_, err := client.UpdateHeartbeat(context.Background(), req)
			cfg, _ := teams.GetHeartbeatConfig(context.Background(), "draft", "leader")
			if tc == "approved" {
				if err != nil || cfg.FiniteLeader.AcceptedRevision != "r2" || len(cfg.PreparationAmendments) != 1 {
					t.Fatal("verified wire amendment failed", err)
				}
			} else {
				if err == nil {
					t.Fatal("unsafe wire mutation admitted")
				}
				if cfg.FiniteLeader.AcceptedRevision != "r1" || cfg.Enabled || len(cfg.PreparationAmendments) != 0 {
					t.Fatal("refusal changed state")
				}
				want := connect.CodePermissionDenied
				if tc == "ordinary-activation" {
					delete(payload, "finiteEffortTransition")
					payload["enabled"] = true
				}
				if tc == "mixed-activation" {
					want = connect.CodeInvalidArgument
				}
				if tc == "stale" {
					want = connect.CodeAborted
				}
				if connect.CodeOf(err) != want && !(tc == "stale" && strings.Contains(err.Error(), "revision")) {
					t.Fatalf("refusal code lost: %v", err)
				}
			}
		})
	}
}
