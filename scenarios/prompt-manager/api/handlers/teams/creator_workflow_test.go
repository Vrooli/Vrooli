package teams

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	heartbeatmount "prompt-manager/handlers/heartbeat"
	heartbeatdomain "prompt-manager/internal/heartbeat"
	"prompt-manager/internal/requestpolicy"
	"prompt-manager/internal/store"
	"prompt-manager/internal/testutil/fixtures"
)

// The actual generated client, identity middleware, domain owner and file store
// must agree on creator authority; a store-only bypass is insufficient.
func TestCreatorConfiguresAndEnablesOwnTeamWithoutConsent(t *testing.T) {
	root := t.TempDir()
	relations := store.NewFileRelationStore(root)
	teams := store.NewFileTeamStore(root, root, relations)
	agents := store.NewFileAgentStore(root)
	if err := agents.Create(context.Background(), &store.Agent{ID: "worker", DisplayName: "Worker"}); err != nil {
		t.Fatal(err)
	}
	domain := NewHandlers(teams, agents, relations, nil, nil)
	heartbeats := heartbeatdomain.NewHandlers(heartbeatdomain.HandlersDeps{TeamStore: teams, AgentStore: agents, RelationStore: relations})
	mux := http.NewServeMux()
	teamPath, teamAPI := NewConnectMount(domain)
	heartbeatPath, heartbeatAPI := heartbeatmount.NewConnectMount(heartbeats)
	mux.Handle(teamPath, teamAPI)
	mux.Handle(heartbeatPath, heartbeatAPI)
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		if token == "invalid" {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: token, Subject: "owner", Scopes: []string{}, ExpiresAt: time.Now().Add(time.Hour).Unix()}}, nil
	})
	server := httptest.NewServer(provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(requestpolicy.PreparationBoundary(mux))))
	defer server.Close()
	client := teamsconnect.NewTeamsServiceClient(server.Client(), server.URL)
	hb := heartbeatconnect.NewHeartbeatServiceClient(server.Client(), server.URL)
	raw, err := json.Marshal(fixtures.IndependentTeam("owned", "Owned", fixtures.WithEnabled(false), fixtures.WithContractAgents("worker")))
	if err != nil {
		t.Fatal(err)
	}
	input := &teamsv1.TeamInput{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, input); err != nil {
		t.Fatal(err)
	}
	create := connect.NewRequest(&teamsv1.CreateTeamRequest{Team: input})
	create.Header().Set(cliutil.HeaderAgentIdentityToken, "creator")
	created, err := client.CreateTeam(context.Background(), create)
	if err != nil {
		t.Fatal(err)
	}
	if !created.Msg.ManagementAuthorized || created.Msg.Revision == 0 {
		t.Fatal("creation did not expose its verified management authority and revision")
	}
	add := connect.NewRequest(&teamsv1.AddMemberRequest{TeamId: "owned", AgentId: "worker"})
	add.Header().Set(cliutil.HeaderAgentIdentityToken, "creator")
	if _, err := client.AddMember(context.Background(), add); err != nil {
		t.Fatalf("creator membership: %v", err)
	}
	body, _ := structpb.NewValue(map[string]any{"enabled": false})
	config := connect.NewRequest(&heartbeatv1.MemberMutationRequest{TeamId: "owned", AgentId: "worker", Body: body})
	config.Header().Set(cliutil.HeaderAgentIdentityToken, "creator")
	if _, err := hb.CreateHeartbeat(context.Background(), config); err != nil {
		t.Fatalf("creator heartbeat: %v", err)
	}
	enable := connect.NewRequest(&teamsv1.UpdateTeamRequest{Id: "owned", Team: &teamsv1.TeamInput{Enabled: true}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled"}}})
	enable.Header().Set(cliutil.HeaderAgentIdentityToken, "creator")
	if _, err := client.UpdateTeam(context.Background(), enable); err != nil {
		t.Fatalf("creator enable: %v", err)
	}
	stored, err := teams.Get(context.Background(), "owned")
	if err != nil || !stored.Enabled || stored.CreationProof == nil || stored.CreationProof.Subject != "creator" {
		t.Fatalf("creator state not retained: %+v %v", stored, err)
	}
	for _, token := range []string{"other", "invalid"} {
		req := connect.NewRequest(&teamsv1.UpdateTeamRequest{Id: "owned", Team: &teamsv1.TeamInput{Mission: "intrusion"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"mission"}}})
		req.Header().Set(cliutil.HeaderAgentIdentityToken, token)
		if _, err := client.UpdateTeam(context.Background(), req); err == nil {
			t.Fatalf("%s mutated creator's team", token)
		}
		read := connect.NewRequest(&teamsv1.GetTeamRequest{Id: "owned"})
		read.Header().Set(cliutil.HeaderAgentIdentityToken, token)
		response, err := client.GetTeam(context.Background(), read)
		if err == nil && response.Msg.ManagementAuthorized {
			t.Fatalf("%s acquired management authority in readback", token)
		}
	}
	// Bad offered identity must not fall through to an unattributed successful create.
	input.Id = "unattributed"
	invalid := connect.NewRequest(&teamsv1.CreateTeamRequest{Team: input})
	invalid.Header().Set(cliutil.HeaderAgentIdentityToken, "invalid")
	if _, err := client.CreateTeam(context.Background(), invalid); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("invalid create should refuse identity: %v", err)
	}
	if _, err := teams.Get(context.Background(), "unattributed"); err == nil {
		t.Fatal("invalid create persisted state")
	}
}
