package scenarios

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/backlog"

	"github.com/gorilla/mux"
)

type remediationCreatorStub struct {
	calls int
	err   error
	item  backlog.BacklogItem
}

func (s *remediationCreatorStub) Create(item backlog.BacklogItem, _ backlog.CreationContext) error {
	s.calls++
	s.item = item
	return s.err
}

func remediationTestRouter(t *testing.T, health ScenarioHealthSnapshot, creator *remediationCreatorStub) *mux.Router {
	t.Helper()
	root, sources := setupTestScenarios(t)
	handler := NewHandlerWithDeps(filepath.Join(root, "scenarios"), stubSource{scenarios: sources}, &stubLifecycle{}, stubCompleteness{scores: map[string]int{}})
	handler.SetHealthSource(staticScenarioHealth{snapshot: health})
	handler.SetRemediationCreator(creator)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return router
}

func TestPhaseRemediationPreviewDoesNotCreateAndApplyIsIdempotent(t *testing.T) {
	health := ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, Phases: []ScenarioHealthPhase{{Phase: "unit", CurrentRung: "L1", NextRung: "L2", PriorityCapabilityID: "coverage", PriorityCapabilityLabel: "Coverage"}}}
	creator := &remediationCreatorStub{}
	router := remediationTestRouter(t, health, creator)
	previewBody := []byte(`{"target":{"scenarioName":"test-scenario-1","providerPhase":"unit","capabilityId":"coverage"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/remediation/preview", bytes.NewReader(previewBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || creator.calls != 0 {
		t.Fatalf("preview status=%d calls=%d body=%s", rec.Code, creator.calls, rec.Body.String())
	}
	proposal, err := BuildPhaseRemediationProposal(health, RemediationTarget{Scenario: "test-scenario-1", ProviderPhase: "unit", CapabilityID: "coverage"}, "manual")
	if err != nil {
		t.Fatal(err)
	}
	applyBody := []byte(`{"target":{"scenarioName":"test-scenario-1","providerPhase":"unit","capabilityId":"coverage"},"fingerprint":"` + proposal.Fingerprint + `"}`)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/remediation/apply", bytes.NewReader(applyBody))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("apply status=%d body=%s", rec.Code, rec.Body.String())
		}
		if i == 0 {
			creator.err = backlog.ErrItemExists
		}
	}
	if creator.item.FindingRef != proposal.Fingerprint || creator.calls != 2 {
		t.Fatalf("unexpected applied item=%#v calls=%d", creator.item, creator.calls)
	}
}

func TestPhaseRemediationRejectsStaleEvidence(t *testing.T) {
	creator := &remediationCreatorStub{err: errors.New("must not be called")}
	router := remediationTestRouter(t, ScenarioHealthSnapshot{EvidenceState: HealthEvidenceStale, Phases: []ScenarioHealthPhase{{Phase: "unit", PriorityCapabilityID: "coverage"}}}, creator)
	body := []byte(`{"target":{"scenarioName":"test-scenario-1","providerPhase":"unit","capabilityId":"coverage"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/remediation/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict || creator.calls != 0 {
		t.Fatalf("stale preview status=%d calls=%d", rec.Code, creator.calls)
	}
}

func TestHealthProjectionReconcilesExistingPhaseWork(t *testing.T) {
	health := ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, Phases: []ScenarioHealthPhase{{Phase: "unit", PriorityCapabilityID: "coverage"}}}
	fingerprint, err := (RemediationTarget{Scenario: "test-scenario-1", ProviderPhase: "unit", CapabilityID: "coverage"}).Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	root, sources := setupTestScenarios(t)
	handler := NewHandlerWithDeps(filepath.Join(root, "scenarios"), stubSource{scenarios: sources}, &stubLifecycle{}, stubCompleteness{scores: map[string]int{}})
	handler.SetHealthSource(staticScenarioHealth{snapshot: health})
	handler.SetBacklogLister(stubBacklogLister{items: []backlog.BacklogItem{{Name: "existing", Kind: backlog.KindFix, Status: backlog.StatusSuggested, FindingRef: fingerprint, Updated: "2026-07-26T00:00:00Z"}}})
	scenario, err := handler.Load("test-scenario-1")
	if err != nil || scenario.Health == nil || len(scenario.Health.Remediation) != 1 {
		t.Fatalf("health reconciliation = %#v, %v", scenario.Health, err)
	}
	if scenario.Health.Remediation[0].WorkRef != "fix/existing" {
		t.Fatalf("unexpected remediation summary: %#v", scenario.Health.Remediation[0])
	}
}
