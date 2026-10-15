package teams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"prompt-manager/internal/requestpolicy"
	"prompt-manager/internal/store"
	"prompt-manager/internal/testutil/fixtures"
)

func TestPreparationReadIsLimitedToApprovedTeam(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teams := store.NewFileTeamStore(root, root, relations)
	for _, id := range []string{"draft", "other"} {
		if err := teams.Create(context.Background(), fixtures.IndependentTeam(id, id, fixtures.WithEnabled(false))); err != nil {
			t.Fatal(err)
		}
	}
	path, mount := NewConnectMount(NewHandlers(teams, store.NewFileAgentStore(root), relations, nil, nil))
	mux := http.NewServeMux()
	mux.Handle(path, mount)
	wire := authn.Middleware(authn.Config{})(requestpolicy.PreparationBoundary(mux))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grant := &identity.OperationGrant{ID: "reviewed", Approver: "owner", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour)}
		r = r.WithContext(provenance.NewContext(r.Context(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, Grant: grant}))
		wire.ServeHTTP(w, r)
	}))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)
	if _, err := client.GetTeam(context.Background(), connect.NewRequest(&teamsv1.GetTeamRequest{Id: "draft"})); err != nil {
		t.Fatal("approved read refused", err)
	}
	if _, err := client.GetTeam(context.Background(), connect.NewRequest(&teamsv1.GetTeamRequest{Id: "other"})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatal("other team disclosed", err)
	}
}
