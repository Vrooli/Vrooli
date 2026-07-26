package scenarios

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/goals"

	"github.com/gorilla/mux"
)

type staticScenarioHealth struct{ snapshot ScenarioHealthSnapshot }

func (s staticScenarioHealth) Snapshot(context.Context, string) ScenarioHealthSnapshot {
	return s.snapshot
}

type campaignCreatorStub struct {
	calls int
	err   error
}

type campaignTrackerStub struct {
	ref   string
	err   error
	calls int
}

func (s *campaignTrackerStub) ReconcileCampaign(context.Context, MaturityCampaignProposal) (string, error) {
	s.calls++
	return s.ref, s.err
}

func (s *campaignCreatorStub) Create(goals.CreateRequest) (*goals.GoalWithScope, error) {
	s.calls++
	return &goals.GoalWithScope{}, s.err
}

func campaignTestHandler(t *testing.T, health ScenarioHealthSnapshot, creator *campaignCreatorStub) (*Handler, *mux.Router) {
	t.Helper()
	root, sources := setupTestScenarios(t)
	handler := NewHandlerWithDeps(filepath.Join(root, "scenarios"), stubSource{scenarios: sources}, &stubLifecycle{}, stubCompleteness{scores: map[string]int{}})
	handler.SetHealthSource(staticScenarioHealth{snapshot: health})
	handler.SetCampaignCreator(creator)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return handler, router
}

func TestMaturityCampaignPreviewDoesNotCreateGoal(t *testing.T) {
	creator := &campaignCreatorStub{}
	_, router := campaignTestHandler(t, ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, Phases: []ScenarioHealthPhase{{Phase: "unit"}}}, creator)
	body := []byte(`{"target":{"scenarioName":"test-scenario-1","maturityTarget":"release ready","providerPhases":["unit"]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/maturity-campaign/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", rec.Code, rec.Body.String())
	}
	if creator.calls != 0 {
		t.Fatalf("preview created %d goals", creator.calls)
	}
}

func TestMaturityCampaignApplyIsIdempotentAndRejectsStaleEvidence(t *testing.T) {
	creator := &campaignCreatorStub{}
	health := ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, Phases: []ScenarioHealthPhase{{Phase: "unit"}}}
	_, router := campaignTestHandler(t, health, creator)
	proposal, err := BuildMaturityCampaignProposalForTarget(health, MaturityCampaignTarget{Scenario: "test-scenario-1", Target: "release ready", ProviderPhases: []string{"unit"}})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"target":{"scenarioName":"test-scenario-1","maturityTarget":"release ready","providerPhases":["unit"]},"fingerprint":"` + proposal.Fingerprint + `"}`)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/maturity-campaign/apply", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("apply status=%d body=%s", rec.Code, rec.Body.String())
		}
		if i == 0 {
			creator.err = errors.Join(goals.ErrValidation, errors.New("goal already exists"))
		}
	}
	if creator.calls != 2 {
		t.Fatalf("apply calls=%d", creator.calls)
	}
	staleCreator := &campaignCreatorStub{}
	_, staleRouter := campaignTestHandler(t, ScenarioHealthSnapshot{EvidenceState: HealthEvidenceStale, Phases: []ScenarioHealthPhase{{Phase: "unit"}}}, staleCreator)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/maturity-campaign/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	staleRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict || staleCreator.calls != 0 {
		t.Fatalf("stale evidence must not create campaign: status=%d calls=%d", rec.Code, staleCreator.calls)
	}
}

func TestMaturityCampaignAttachsOptionalTrackerOnlyAfterGoalCreation(t *testing.T) {
	creator := &campaignCreatorStub{}
	health := ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, Phases: []ScenarioHealthPhase{{Phase: "unit"}}}
	handler, router := campaignTestHandler(t, health, creator)
	tracker := &campaignTrackerStub{ref: "campaign/cartographer-42"}
	handler.SetCampaignTracker(tracker)
	proposal, err := BuildMaturityCampaignProposalForTarget(health, MaturityCampaignTarget{Scenario: "test-scenario-1", Target: "release ready", ProviderPhases: []string{"unit"}})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"target":{"scenarioName":"test-scenario-1","maturityTarget":"release ready","providerPhases":["unit"]},"fingerprint":"` + proposal.Fingerprint + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/maturity-campaign/apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || tracker.calls != 1 || !bytes.Contains(rec.Body.Bytes(), []byte("cartographer-42")) {
		t.Fatalf("tracker response status=%d calls=%d body=%s", rec.Code, tracker.calls, rec.Body.String())
	}
}
