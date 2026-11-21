package teams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"prompt-manager/internal/requestpolicy"
	"prompt-manager/internal/store"
	domainteams "prompt-manager/internal/teams"
	"prompt-manager/internal/testutil/fixtures"
)

// `prompt-manager team update <id> -archived true` printed only
// "internal: operation or target is outside the approved agent grant". A refused
// agent must get a permission-denied error carrying the exact request for
// prompt-manager:team/<id>, which is what the CLI prints.
func TestRefusedTeamUpdateSurfacesTheRequestCommand(t *testing.T) {
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "")
	const teamID = "desktop-packaging-delivery"
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teamStore := store.NewFileTeamStore(root, root, relations)
	if err := teamStore.Create(context.Background(), fixtures.IndependentTeam(teamID, "Desktop packaging delivery")); err != nil {
		t.Fatalf("seed team: %v", err)
	}
	handler := domainteams.NewHandlers(teamStore, store.NewFileAgentStore(root), relations, nil, nil)
	path, mount := NewConnectMount(handler)
	mux := http.NewServeMux()
	mux.Handle(path, mount)
	boundary := requestpolicy.PreparationBoundary(mux)
	// The agent holds a team-management grant for a different team only.
	principal := identity.Principal{
		Kind: identity.ActorAgent, Subject: "run", Owner: "human-owner", Source: identity.SourceAgentProvenance, Verified: true,
		Scopes: []string{"prompt-manager:read", "prompt-manager:write"},
		Grant: &identity.OperationGrant{
			ID: "other-team", Policy: "team-management", Approver: "human-owner",
			Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/another"}, ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		boundary.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	}))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)

	want := `vrooli agent authorize "$VROOLI_AGENT_SESSION" --policy team-management --target prompt-manager:team/` + teamID + ` --reason "<why>"`
	_, err := client.UpdateTeam(context.Background(), connect.NewRequest(&teamsv1.UpdateTeamRequest{
		Id: teamID, Team: &teamsv1.TeamInput{Archived: true}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"archived"}},
	}))
	if err == nil {
		t.Fatal("a team outside the grant was changed")
	}
	if code := connect.CodeOf(err); code != connect.CodePermissionDenied {
		t.Fatalf("code = %s, want permission_denied (a refused grant is not an internal error): %v", code, err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("the CLI-visible error does not carry the request command\n got: %v\nwant substring: %s", err, want)
	}
	stored, err := teamStore.Get(context.Background(), teamID)
	if err != nil || stored.Archived {
		t.Fatalf("a refused update changed the team: archived=%v err=%v", stored != nil && stored.Archived, err)
	}
}
