package agentsessions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatordecision"
)

// This test intentionally uses only symbols present before the reissue route
// existed, so it also serves as a compile-safe old-handler causal control.
func TestLegacyProposalReissueRoutePublishesOnlyInspectedCurrentTerms(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	proposal := Proposal{
		ID:          "prop_legacy_route",
		Kind:        ProposalMutationList,
		Status:      ProposalStatusReady,
		Summary:     "Mutation proposal for Quality Gates",
		PayloadJSON: readyMutationPayload,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
		Target:      &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
		DecisionPublication: &ProposalDecisionPublication{
			Version: 1,
			State:   ProposalDecisionPublicationLegacyUnresolved,
		},
	}
	if err := store.SaveProposal(session.ID, proposal); err != nil {
		t.Fatalf("SaveProposal(legacy): %v", err)
	}
	digest := routeTermsDigest(t, proposal)
	body, err := json.Marshal(map[string]string{"expected_terms_sha256": digest})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	NewHandler(service).RegisterRoutes(router)
	path := "/api/v1/agent-sessions/" + session.ID + "/proposals/" + proposal.ID + "/reissue-decision-request"
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req = req.WithContext(identity.WithPrincipal(req.Context(), *grantPrincipal("run", []string{"swarm.bookkeeping"}, []string{"swarm-manager:proposal/" + session.ID + "/" + proposal.ID})))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("reissue status = %d, body %q; exact digest was supplied", response.Code, response.Body.String())
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want one", len(publisher.events))
	}
	eventID := operatordecision.EventID(publisher.events[0])
	stored, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := findProposal(stored, proposal.ID)
	if !ok || current.DecisionPublication == nil || current.DecisionPublication.EventID != eventID {
		t.Fatalf("response did not persist the exact current request: %+v", current)
	}
}

func TestLegacyProposalReissueRouteRequiresExactVerifiedBookkeepingGrant(t *testing.T) {
	for _, tc := range []struct {
		name      string
		principal *identity.Principal
	}{
		{name: "missing identity"},
		{name: "unverified identity", principal: &identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "operator:test", Verified: false}},
		{name: "missing grant", principal: &identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "operator:test", Verified: true, Scopes: []string{"swarm-manager:write"}}},
		{name: "wrong operation", principal: grantPrincipal("run", []string{"swarm.proposal.apply"}, []string{"swarm-manager:proposal/session/proposal"})},
		{name: "wrong target", principal: grantPrincipal("run", []string{"swarm.bookkeeping"}, []string{"swarm-manager:proposal/other/proposal"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, publisher, _ := newProposalDecisionService(t)
			session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
			proposal := recordLegacyReadyMutationProposal(t, service, session.ID)
			digest, err := proposalDecisionTermsSHA256(proposal)
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(map[string]string{"expected_terms_sha256": digest})
			if err != nil {
				t.Fatal(err)
			}
			router := mux.NewRouter()
			NewHandler(service).RegisterRoutes(router)
			path := "/api/v1/agent-sessions/" + session.ID + "/proposals/" + proposal.ID + "/reissue-decision-request"
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
			if tc.principal != nil {
				req = req.WithContext(identity.WithPrincipal(req.Context(), *tc.principal))
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != http.StatusForbidden {
				t.Fatalf("reissue status=%d body=%q, want forbidden", response.Code, response.Body.String())
			}
			if len(publisher.events) != 0 {
				t.Fatalf("denied request emitted %d events", len(publisher.events))
			}
			stored, err := service.store.(*FileStore).LoadSession(session.ID)
			if err != nil {
				t.Fatal(err)
			}
			current, ok := findProposal(stored, proposal.ID)
			if !ok || current.PreviousDecisionPublication != nil || current.DecisionPublication == nil || current.DecisionPublication.State != ProposalDecisionPublicationLegacyUnresolved {
				t.Fatalf("denied request changed proposal state: %+v", current)
			}
		})
	}
}

func grantPrincipal(subject string, operations, targets []string) *identity.Principal {
	return &identity.Principal{
		Kind: identity.ActorAgent, Subject: subject, Owner: "operator:test", Verified: true,
		Scopes: []string{"swarm-manager:write"},
		Grant:  &identity.OperationGrant{ID: "approved-bookkeeping", Approver: "operator:test", Operations: operations, Targets: targets, ExpiresAt: time.Now().Add(time.Hour)},
	}
}

type legacyRouteTerms struct {
	ID          string          `json:"id"`
	Kind        ProposalKind    `json:"kind"`
	Summary     string          `json:"summary"`
	PayloadJSON string          `json:"payload_json"`
	CreatedAt   string          `json:"created_at"`
	Attribution *Attribution    `json:"attribution,omitempty"`
	Target      *ProposalTarget `json:"target,omitempty"`
}

func routeTermsDigest(t *testing.T, proposal Proposal) string {
	t.Helper()
	encoded, err := json.Marshal(legacyRouteTerms{
		ID: proposal.ID, Kind: proposal.Kind,
		Summary: proposal.Summary, PayloadJSON: proposal.PayloadJSON,
		CreatedAt: proposal.CreatedAt, Attribution: proposal.Attribution,
		Target: proposal.Target,
	})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
