package review_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	databasetest "github.com/vrooli/api-core/databasetest"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review/review_v1connect"
	"google.golang.org/protobuf/encoding/protojson"

	reviewhandler "react-component-library/handlers/review"
	domain "react-component-library/internal/designcritique"
	reviewdomain "react-component-library/internal/review"
	sketchdomain "react-component-library/internal/sketch"
)

type portTestReviewService struct{}

func (portTestReviewService) List(context.Context, reviewdomain.Filter, int, int) (reviewdomain.Page, error) {
	return reviewdomain.Page{}, nil
}

func (portTestReviewService) Get(context.Context, string) (reviewdomain.Finding, error) {
	return reviewdomain.Finding{}, nil
}

func (portTestReviewService) RecordDisposition(context.Context, string, string, string, string) (reviewdomain.Finding, error) {
	return reviewdomain.Finding{}, nil
}

func (portTestReviewService) Assign(context.Context, string, string) (reviewdomain.Finding, error) {
	return reviewdomain.Finding{}, nil
}

func (portTestReviewService) Summary(context.Context) (reviewdomain.Summary, error) {
	return reviewdomain.Summary{}, nil
}

type portTestAcceptanceEvaluator struct {
	called *reviewv1.CheckCandidateAcceptanceRequest
}

func (e *portTestAcceptanceEvaluator) ValidateAcceptanceCandidate(context.Context, *reviewv1.CheckCandidateAcceptanceRequest) error {
	return nil
}

func (e *portTestAcceptanceEvaluator) EvaluateAcceptance(_ context.Context, req *reviewv1.CheckCandidateAcceptanceRequest) (*reviewv1.CandidateAcceptanceCheck, error) {
	e.called = req
	return &reviewv1.CandidateAcceptanceCheck{PolicyVersion: "port-test/1", RenderHash: "render", Ready: true}, nil
}

type portTestAcceptanceStore struct {
	decision sketchdomain.AcceptanceDecision
	created  bool
}

func (s *portTestAcceptanceStore) RecordAcceptance(ctx context.Context, _, _ string, _ sketchdomain.AcceptanceIntent, evaluate func(context.Context) (sketchdomain.AcceptanceFacts, error)) (sketchdomain.AcceptanceDecision, error) {
	if !s.created {
		if _, err := evaluate(ctx); err != nil {
			return sketchdomain.AcceptanceDecision{}, err
		}
		s.created = true
	}
	return s.decision, nil
}

func (*portTestAcceptanceStore) ReadAcceptance(string, string, string) (sketchdomain.AcceptanceDecision, error) {
	return sketchdomain.AcceptanceDecision{}, nil
}

type portTestCritiqueRepository struct {
	store  *databasetest.SliceRepo[domain.Record]
	target domain.Target
	before string
}

func (portTestCritiqueRepository) Create(context.Context, string, domain.Review, domain.Assessment) (domain.Record, error) {
	return domain.Record{}, nil
}

func (r *portTestCritiqueRepository) List(_ context.Context, target domain.Target, before string) ([]domain.Record, string, error) {
	r.target, r.before = target, before
	return []domain.Record{{
		ID: "critique-1", Hash: "review-hash", RecordedAt: "2026-11-21T12:00:00Z",
		Review:     domain.Review{Critic: domain.Critic{Kind: "human", ID: "reviewer"}},
		Assessment: domain.Assessment{VisualFloorMet: true},
	}}, "older-page", nil
}

func (r *portTestCritiqueRepository) Get(ctx context.Context, id string) (domain.Record, error) {
	return r.store.Get(ctx, id)
}

func TestModuleFromServiceWithPortsRoutesAcceptanceAndCritiqueRPCs(t *testing.T) {
	evaluator := &portTestAcceptanceEvaluator{}
	acceptanceStore := &portTestAcceptanceStore{}
	critiqueRepo := &portTestCritiqueRepository{store: databasetest.NewSliceRepo(
		func(record domain.Record) string { return record.ID },
		func(record *domain.Record, id string) { record.ID = id },
		func(id string) error { return errors.New("critique not found: " + id) },
	)}
	acceptanceWrites := 0
	mod := reviewhandler.ModuleFromServiceWithPorts(portTestReviewService{}, reviewhandler.Ports{
		AcceptanceEvaluator: evaluator,
		AcceptanceStoreFor:  func(context.Context) (reviewhandler.AcceptanceStore, error) { return acceptanceStore, nil },
		AcceptanceWrite:     func(context.Context) { acceptanceWrites++ },
		CritiquesFor:        func(context.Context) (domain.Repository, error) { return critiqueRepo, nil },
	})
	router := mux.NewRouter()
	mod.Mount(router)

	call := func(path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	acceptance := call(reviewconnect.ReviewServiceCheckCandidateAcceptanceProcedure, `{"render":{"candidate":{"scenario":"demo","designId":"home","hash":"candidate"}},"expectedRenderHash":"render","critiqueIds":["critique-1"]}`)
	require.Equal(t, http.StatusOK, acceptance.Code, acceptance.Body.String())
	var check reviewv1.CandidateAcceptanceCheck
	require.NoError(t, protojson.Unmarshal(acceptance.Body.Bytes(), &check))
	require.True(t, check.GetReady())
	require.Equal(t, "port-test/1", check.GetPolicyVersion())
	require.Equal(t, "demo", evaluator.called.GetRender().GetCandidate().GetScenario())
	require.Equal(t, []string{"critique-1"}, evaluator.called.GetCritiqueIds())

	decisionRequest := `{"check":{"render":{"candidate":{"scenario":"demo","designId":"home","hash":"candidate"}},"expectedRenderHash":"render"},"actor":"tester","idempotencyKey":"decision-1"}`
	for i := 0; i < 2; i++ {
		decision := call(reviewconnect.ReviewServiceAcceptCandidateProcedure, decisionRequest)
		require.Equal(t, http.StatusOK, decision.Code, decision.Body.String())
	}
	require.Equal(t, 1, acceptanceWrites, "an idempotent replay must not repeat the acceptance write notification")

	critiques := call(reviewconnect.ReviewServiceListCritiquesProcedure, `{"target":{"scenario":"demo","designId":"home","revision":"rev-1","renderHash":"render-1"},"beforeId":"cursor"}`)
	require.Equal(t, http.StatusOK, critiques.Code, critiques.Body.String())
	var listed reviewv1.ListCritiquesResponse
	require.NoError(t, protojson.Unmarshal(critiques.Body.Bytes(), &listed))
	require.Equal(t, domain.Target{Scenario: "demo", DesignID: "home", Revision: "rev-1", RenderHash: "render-1"}, critiqueRepo.target)
	require.Equal(t, "cursor", critiqueRepo.before)
	require.Equal(t, "older-page", listed.GetNextBeforeId())
	require.Len(t, listed.GetReviews(), 1)
	require.Equal(t, "critique-1", listed.GetReviews()[0].GetId())
	require.Equal(t, "reviewer", listed.GetReviews()[0].GetCritic().GetId())
}
