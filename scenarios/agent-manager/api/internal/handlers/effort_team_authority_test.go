package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/promptmanager"
	"agent-manager/internal/supervision"
	"connectrpc.com/connect"
	"github.com/jmoiron/sqlx"
	"github.com/vrooli/api-core/authn"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type teamOwnerRPC struct {
	teamsconnect.UnimplementedTeamsServiceHandler
}

func (teamOwnerRPC) GetTeam(_ context.Context, req *connect.Request[teamsv1.GetTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	coordination, _ := structpb.NewStruct(map[string]any{"leadAgentId": "leader"})
	return connect.NewResponse(&teamsv1.TeamDetails{Id: req.Msg.Id, Purpose: "delivery", Lifetime: "finite", HomePath: "delivery/owned", EffortRefs: []string{"effort:owned", "effort:owned-delivery"}, Coordination: coordination, Members: []*teamsv1.Member{{AgentId: "leader", Status: "active"}}, Revision: 1, ManagementAuthorized: req.Header().Get(cliutil.HeaderAgentIdentityToken) == "creator"}), nil
}

func TestTeamOwnerEnrollmentThroughGeneratedClients(t *testing.T) {
	testTeamOwnerEnrollmentThroughGeneratedClients(t, false)
}

func TestCombinedTeamAndDeliveryGrantPreparesTeamEnrollment(t *testing.T) {
	testTeamOwnerEnrollmentThroughGeneratedClients(t, true)
}

func testTeamOwnerEnrollmentThroughGeneratedClients(t *testing.T, combinedGrant bool) {
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(supervision.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := supervision.NewRepository(db)
	efforts := supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{Root: t.TempDir()})
	ownerPath, ownerAPI := teamsconnect.NewTeamsServiceHandler(teamOwnerRPC{})
	ownerMux := http.NewServeMux()
	ownerMux.Handle(ownerPath, ownerAPI)
	owner := httptest.NewServer(ownerMux)
	defer owner.Close()
	h := NewAgentManagerConnectHandler(nil, &supervision.Service{Efforts: efforts})
	h.SetManagedTeamReader(promptmanager.NewHTTPClientWithResolver(func(context.Context) (string, error) { return owner.URL, nil }, nil))
	// An available human authorizer must never be an agent fallback.
	h.SetWatchActionAuthorizer(allowWatchAction{})
	apiPath, api := apiconnect.NewAgentManagerServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(apiPath, api)
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		claims := &cliutil.VerifiedClaims{RunID: token, Subject: "owner", ExpiresAt: time.Now().Add(time.Hour).Unix()}
		if combinedGrant && token == "creator" {
			grant := &identity.OperationGrant{ID: "combined", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour), Rules: []identity.OperationRule{
				{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/owned"}},
				{Policy: "delivery-management", Operations: []string{"effort.delivery"}, Targets: []string{"agent-manager:effort/owned", "agent-manager:effort/owned-delivery"}},
			}}
			raw, err := json.Marshal(grant)
			if err != nil {
				t.Fatal(err)
			}
			claims.Scopes = []string{"prompt-manager:read", "prompt-manager:write", "agent-manager:read", "agent-manager:write"}
			claims.Meta = map[string]string{"operation_grant": string(raw), "authorization_id": "combined"}
		}
		return &cliutil.VerifyResult{Valid: true, Claims: claims}, nil
	})
	server := httptest.NewServer(provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(AgentOperationMiddleware(mux))))
	defer server.Close()
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)
	initial := &pb.EnrollEffortRequest{IdempotencyKey: "prepare", Enrollment: &pb.EffortEnrollment{EffortRef: "effort:owned", DisplayName: "Owned", DestinationRef: "path:delivery/owned/CHARTER.md", TargetRevision: "target", SourceRevision: "source", AuthorityRef: "owner-instruction", WorkShape: "orchestrated_epochs", Subjects: []*pb.EffortSubject{{Owner: "prompt-manager", Kind: "team-member", Reference: "owned/leader", Role: "orchestrator"}}}}
	call := func(token string, request *pb.EnrollEffortRequest) (*pb.EffortEnrollment, error) {
		r := connect.NewRequest(request)
		r.Header().Set(cliutil.HeaderAgentIdentityToken, token)
		out, err := client.EnrollEffort(ctx, r)
		if err != nil {
			return nil, err
		}
		return out.Msg, nil
	}
	for _, tc := range []struct {
		name, token string
		edit        func(*pb.EffortEnrollment)
	}{
		{name: "different caller", token: "other", edit: func(*pb.EffortEnrollment) {}},
		{name: "different effort", token: "creator", edit: func(e *pb.EffortEnrollment) { e.EffortRef = "effort:other" }},
		{name: "different leader", token: "creator", edit: func(e *pb.EffortEnrollment) { e.Subjects[0].Reference = "owned/intruder" }},
		{name: "destination escape", token: "creator", edit: func(e *pb.EffortEnrollment) { e.DestinationRef = "path:delivery/owned/../../other/CHARTER.md" }},
		{name: "extra subject", token: "creator", edit: func(e *pb.EffortEnrollment) {
			e.Subjects = append(e.Subjects, &pb.EffortSubject{RunId: "unrelated", Role: "worker"})
		}},
		{name: "start during preparation", token: "creator", edit: func(e *pb.EffortEnrollment) {
			e.Delivery = &pb.EffortDelivery{CommissionId: "unrequested", State: "running", BudgetSeconds: 60, MaxWorkers: 1}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := proto.Clone(initial).(*pb.EnrollEffortRequest)
			request.IdempotencyKey = tc.name
			tc.edit(request.Enrollment)
			if _, err := call(tc.token, request); err == nil {
				t.Fatal("invalid team-bound enrollment was admitted")
			}
			if rows, err := repo.ListEfforts(ctx, "", 10); err != nil || len(rows) != 0 {
				t.Fatalf("refused enrollment wrote state: %v %v", rows, err)
			}
		})
	}
	created, err := call("creator", initial)
	if err != nil {
		t.Fatalf("verified team manager could not prepare its effort: %v", err)
	}
	if created.AuthorizedBy != "creator" || created.Delivery != nil || created.Revision != 1 {
		t.Fatalf("preparation lost actor or started clock: %+v", created)
	}
	replay, err := call("creator", initial)
	if err != nil || !proto.Equal(replay, created) {
		t.Fatalf("retry changed enrollment: %+v %v", replay, err)
	}

	if combinedGrant {
		// A second approved policy must not remove existing delivery authority on
		// an operator-prepared enrollment that the team manager does not own.
		operatorRequest := proto.Clone(initial).(*pb.EnrollEffortRequest)
		operatorRequest.Enrollment.EffortRef = "effort:owned-delivery"
		operatorRequest.IdempotencyKey = "operator-prepare"
		prepared, err := efforts.Enroll(ctx, operatorRequest, supervision.EffortActor{ID: "original-operator", Operator: true})
		if err != nil {
			t.Fatal(err)
		}
		commission := &pb.EnrollEffortRequest{Enrollment: proto.Clone(prepared).(*pb.EffortEnrollment), ExpectedRevision: prepared.Revision, IdempotencyKey: "commission"}
		commission.Enrollment.Delivery = &pb.EffortDelivery{CommissionId: "approved", State: "running", BudgetSeconds: 3600, MaxWorkers: 2}
		commissioned, err := call("creator", commission)
		if err != nil {
			t.Fatalf("team-management rule masked valid delivery authority: %v", err)
		}
		if commissioned.AuthorizedBy != "original-operator" || commissioned.Delivery.GetState() != "running" {
			t.Fatal("commission lost enrollment ownership or clock state")
		}
	}
}
