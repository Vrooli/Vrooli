package development

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

type operatorProvider struct{}

func (operatorProvider) Source() coreidentity.AuthSource {
	return coreidentity.SourceScenarioAuthenticator
}

func (operatorProvider) VerifyRequest(_ context.Context, req *http.Request) (coreidentity.Principal, error) {
	if req.Header.Get("X-Test-Operator") != "verified-fixture" {
		return coreidentity.Principal{}, coreidentity.NewFailure(coreidentity.FailureMissing, coreidentity.SourceScenarioAuthenticator)
	}
	return coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "test-operator", Verified: true, Source: coreidentity.SourceScenarioAuthenticator, Scopes: []string{"swarm-manager:write"}}, nil
}

// [REQ:SWM-P0-004] [REQ:SWM-P0-017]
func TestConnectApprovalRetainsBytesAndRequiresVerifiedDecisionCapability(t *testing.T) {
	r, p := fixture(t)
	repo := openRepo(t, filepath.Join(t.TempDir(), "state.db"))
	s := NewService(repo, r, func(context.Context, string) error { return nil }, nil)
	mux := mux.NewRouter()
	RegisterRoutes(mux, s, authn.Config{Providers: []authn.Provider{operatorProvider{}}})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := apiconnect.NewDevelopmentServiceClient(server.Client(), server.URL)
	review, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	request := connect.NewRequest(&api.ApproveDevelopmentRequest{Proposal: ProposalProto(p), ReviewedDigest: review.ProposalDigest, ExpectedVersion: 0, Reason: "reviewed this target"})
	if _, err := client.ApproveDevelopment(context.Background(), request); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("anonymous approval: %v", err)
	}
	if _, err := repo.Get(context.Background(), p.WorkItem); err != ErrNotFound {
		t.Fatal("anonymous approval persisted state")
	}
	request.Header().Set("X-Test-Operator", "verified-fixture")
	approved, err := client.ApproveDevelopment(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Msg.Version != 1 || approved.Msg.Status != "approved" || len(approved.Msg.LaunchBlockers) == 0 || len(approved.Msg.Attempts) != 0 {
		t.Fatalf("approval response lost readiness distinction: %v", approved.Msg)
	}
	writeFixture(t, r.RepoRoot, "scenarios/example/PRD.md", "live source changed")
	retained, err := client.GetDevelopmentArtifact(context.Background(), connect.NewRequest(&api.GetDevelopmentArtifactRequest{WorkItem: p.WorkItem, Digest: review.ProposalDigest, Path: "scenarios/example/PRD.md"}))
	if err != nil || string(retained.Msg.Content) != "Protected product outcome" {
		t.Fatalf("retained content: %v", err)
	}
	_, err = client.GetDevelopmentArtifact(context.Background(), connect.NewRequest(&api.GetDevelopmentArtifactRequest{WorkItem: p.WorkItem, Digest: "other-revision", Path: "scenarios/example/PRD.md"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unowned snapshot read: %v", err)
	}
	read, err := client.GetDevelopment(context.Background(), connect.NewRequest(&api.GetDevelopmentRequest{WorkItem: p.WorkItem}))
	if err != nil || read.Msg.Approvals[0].Actor != "test-operator" {
		t.Fatalf("verified actor not retained: %v", err)
	}
}

func TestProposalTransportPreservesPlanAndStrategy(t *testing.T) {
	_, p := fixture(t)
	p.ExecutionStrategy = AdaptiveStrategy
	roundTrip, err := ProposalFromProto(ProposalProto(p))
	if err != nil {
		t.Fatal(err)
	}
	if !planReferencesEqual(roundTrip.PlanRef, p.PlanRef) || roundTrip.ExecutionStrategy != p.ExecutionStrategy {
		t.Fatalf("proposal transport lost work package: %#v", roundTrip)
	}
}

func TestConnectReadWorksWhenAuthenticationIsNotConfigured(t *testing.T) {
	s, p, _ := setupService(t)
	mux := mux.NewRouter()
	RegisterRoutes(mux, s, authn.Config{})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := apiconnect.NewDevelopmentServiceClient(server.Client(), server.URL)
	read, err := client.GetDevelopment(context.Background(), connect.NewRequest(&api.GetDevelopmentRequest{WorkItem: p.WorkItem}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.RevokeDevelopment(context.Background(), connect.NewRequest(&api.RevokeDevelopmentRequest{WorkItem: p.WorkItem, ExpectedVersion: read.Msg.Version, Reason: "anonymous revoke"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfigured authentication allowed a decision: %v", err)
	}
	after, _ := s.Get(context.Background(), p.WorkItem)
	if after.Version != read.Msg.Version {
		t.Fatal("refused decision changed authority")
	}
}
