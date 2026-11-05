package backlog

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

	"swarm-manager/internal/agentsessions"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/identity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

func TestInteractiveGrantCannotReachUnreviewedMutations(t *testing.T) {
	for _, tc := range []struct {
		method, path, template string
		want                   int
	}{
		{http.MethodPatch, "/api/v1/backlog/execute/item/archive-item", "/api/v1/backlog/{kind}/{name}/archive-item", 204},
		{http.MethodDelete, "/api/v1/backlog/execute/item", "/api/v1/backlog/{kind}/{name}", 403},
		{http.MethodPost, apiconnect.GoalServiceCloseOutGoalProcedure, apiconnect.GoalServiceCloseOutGoalProcedure, 403},
		{http.MethodPost, apiconnect.BacklogServiceDecideAttemptProcedure, apiconnect.BacklogServiceDecideAttemptProcedure, 403},
		{http.MethodPost, apiconnect.GoalServiceUpdateGoalProcedure, apiconnect.GoalServiceUpdateGoalProcedure, 204},
		{http.MethodPost, apiconnect.BacklogServiceArchiveTerminalItemsProcedure, apiconnect.BacklogServiceArchiveTerminalItemsProcedure, 204},
		{http.MethodPost, "/api/v1/agent-sessions/sess/proposals/prop/reissue-decision-request", "/api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/reissue-decision-request", 204},
		{http.MethodPost, "/api/v1/events/notification-hub", "/api/v1/events/notification-hub", 204},
		{http.MethodPost, "/api/v1/search/ai", "/api/v1/search/ai", 204},
		{http.MethodPost, "/api/v1/records/search", "/api/v1/records/search", 204},
		{http.MethodPost, apiconnect.TransitionServiceStartTransitionProcedure, apiconnect.TransitionServiceStartTransitionProcedure, 204},
		{http.MethodPost, apiconnect.TransitionServiceApplyTransitionProcedure, apiconnect.TransitionServiceApplyTransitionProcedure, 204},
		{http.MethodPost, "/api/v1/search/ai/reconcile", "/api/v1/search/ai/reconcile", 403},
		{http.MethodPost, "/api/v1/search/ai/reconcile/cancel", "/api/v1/search/ai/reconcile/cancel", 403},
		{http.MethodPost, "/api/v1/search/ai/other", "/api/v1/search/ai/other", 403},
		{http.MethodPost, "/api/v1/records/search/extra", "/api/v1/records/search/extra", 403},
		{http.MethodPatch, "/api/v1/search/ai", "/api/v1/search/ai", 403},
		{http.MethodPost, "/api/v1/events/notification-hub/extra", "/api/v1/events/notification-hub/extra", 403},
		{http.MethodPost, "/future.Service/Mutate", "/future.Service/Mutate", 403},
	} {
		t.Run(tc.path, func(t *testing.T) {
			router := mux.NewRouter()
			router.Use(InteractiveGrantMiddleware)
			router.HandleFunc(tc.template, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }).Methods(tc.method)
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Grant: &identity.OperationGrant{ID: "grant"}})
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil).WithContext(ctx))
			if w.Code != tc.want {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

type reissueIntegrationPublisher struct{ events []eventbus.DomainEvent }

func (p *reissueIntegrationPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	return nil
}

func TestInteractiveGrantMiddlewareAndExactBookkeepingProofReachReissueHandler(t *testing.T) {
	store := agentsessions.NewFileStore(t.TempDir())
	service, err := agentsessions.NewService(agentsessions.ServiceConfig{Store: store, ProjectRoot: t.TempDir(), ProfileKey: "swarm-manager/default"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.Create(context.Background(), agentsessions.CreateRequest{Kind: agentsessions.KindSwarmOperations, Title: "Review prepared change"})
	if err != nil {
		t.Fatal(err)
	}
	proposal := agentsessions.Proposal{
		ID: "prop_middleware", Kind: agentsessions.ProposalMutationList, Status: agentsessions.ProposalStatusReady,
		Summary: "Mutation proposal", PayloadJSON: `{"form":"mutation_list","mutations":[{"id":"m1"}]}`,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Target:              &agentsessions.ProposalTarget{Type: agentsessions.ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
		DecisionPublication: &agentsessions.ProposalDecisionPublication{Version: 1, State: agentsessions.ProposalDecisionPublicationLegacyUnresolved},
	}
	if err := store.SaveProposal(session.ID, proposal); err != nil {
		t.Fatal(err)
	}
	digest := integrationTermsDigest(t, proposal)
	body, err := json.Marshal(map[string]string{"expected_terms_sha256": digest})
	if err != nil {
		t.Fatal(err)
	}
	publisher := &reissueIntegrationPublisher{}
	service.SetDecisionEventPublisher(publisher)
	router := mux.NewRouter()
	router.Use(InteractiveGrantMiddleware)
	agentsessions.NewHandler(service).RegisterRoutes(router)
	grant := &identity.OperationGrant{
		ID: "approved-bookkeeping", Approver: "operator:test", Operations: []string{"swarm.bookkeeping"},
		Targets: []string{"swarm-manager:proposal/" + session.ID + "/" + proposal.ID}, ExpiresAt: time.Now().Add(time.Hour),
	}
	principal := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "operator:test", Verified: true, Scopes: []string{"swarm-manager:write"}, Grant: grant}
	path := "/api/v1/agent-sessions/" + session.ID + "/proposals/" + proposal.ID + "/reissue-decision-request"
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)).WithContext(identity.WithPrincipal(context.Background(), principal))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("approved exact operation status=%d body=%q", response.Code, response.Body.String())
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published event count=%d, want one", len(publisher.events))
	}
}

type integrationDecisionTerms struct {
	ID          string                        `json:"id"`
	Kind        agentsessions.ProposalKind    `json:"kind"`
	Summary     string                        `json:"summary"`
	PayloadJSON string                        `json:"payload_json"`
	CreatedAt   string                        `json:"created_at"`
	Attribution *agentsessions.Attribution    `json:"attribution,omitempty"`
	Target      *agentsessions.ProposalTarget `json:"target,omitempty"`
}

func integrationTermsDigest(t *testing.T, proposal agentsessions.Proposal) string {
	t.Helper()
	encoded, err := json.Marshal(integrationDecisionTerms{
		ID: proposal.ID, Kind: proposal.Kind, Summary: proposal.Summary,
		PayloadJSON: proposal.PayloadJSON, CreatedAt: proposal.CreatedAt, Attribution: proposal.Attribution,
		Target: proposal.Target,
	})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func TestBookkeepingBoundaryCannotBeEvadedByDroppingAgentIdentity(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", nil)
	response := httptest.NewRecorder()
	InteractiveGrantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })).ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous caller reached retained operator mutation: %d", response.Code)
	}
}
