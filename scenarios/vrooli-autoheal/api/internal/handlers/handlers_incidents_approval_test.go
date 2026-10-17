package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/remediation"
)

type approvalVerifierStub struct {
	approval remediation.AskApproval
	err      error
}

func (s approvalVerifierStub) Verify(context.Context, string) (remediation.AskApproval, error) {
	return s.approval, s.err
}

type approvalRunnerStub struct {
	preflightPath string
	runPath       string
}

func (r *approvalRunnerStub) Preflight(_ context.Context, path string) error {
	r.preflightPath = path
	return nil
}

func (r *approvalRunnerStub) Run(_ context.Context, path string) (int, string, error) {
	r.runPath = path
	return 0, "fixture remediation executed", nil
}

type approvalStoreStub struct {
	*mockStore
	recordedAsk         string
	recordedIncident    string
	recordedFingerprint string
	recordedRemediation string
	recordedApprovedBy  string
	claimed             bool
	claimAsk            string
	claimIncident       string
	claimFingerprint    string
	claimRemediation    string
}

func (s *approvalStoreStub) RecordRemediationAuthorisation(_ context.Context, askID, incidentID, fingerprint, remediationID, approvedBy string, _ time.Time) error {
	s.recordedAsk = askID
	s.recordedIncident = incidentID
	s.recordedFingerprint = fingerprint
	s.recordedRemediation = remediationID
	s.recordedApprovedBy = approvedBy
	return nil
}

func (s *approvalStoreStub) ClaimRemediationAuthorisation(_ context.Context, askID, incidentID, fingerprint, remediationID string, _ time.Time) (bool, error) {
	s.claimAsk = askID
	s.claimIncident = incidentID
	s.claimFingerprint = fingerprint
	s.claimRemediation = remediationID
	return s.claimed, nil
}

func TestApproveIncidentRemediationExecutesVerifiedAskAtHTTPBoundary(t *testing.T) {
	store := &approvalStoreStub{
		mockStore: &mockStore{incident: &incidents.Incident{
			ID:             "inc-1",
			Fingerprint:    "fp-1",
			SourceCheckIDs: []string{"host-kernel-module-drift"},
			RemediationCandidates: []incidents.RemediationCandidate{{
				ID:            "candidate-1",
				Applicability: "applicable",
			}},
			RemediationArtifacts: []incidents.RemediationArtifact{{
				RemediationID: "candidate-1",
				Path:          "/tmp/fixture-remediation",
			}},
		}},
		claimed: true,
	}
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	registry.Register(&mockHealableCheckCritical{
		mockCheck:       mockCheck{id: "host-kernel-module-drift", status: checks.StatusCritical},
		recoveryActions: []checks.RecoveryAction{{ID: "candidate-1", Available: true}},
	})
	registry.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"host-kernel-module-drift": true},
	})
	h := NewWithInterface(registry, store, caps)
	h.SetRemediationAskVerifier(approvalVerifierStub{approval: remediation.AskApproval{
		AskID:  "ask-1",
		Answer: "approve",
		Actor:  "operator-1",
	}})
	runner := &approvalRunnerStub{}
	h.remediationService.SetScriptRunner(runner)

	body, err := json.Marshal(map[string]string{
		"askId":               "ask-1",
		"incidentFingerprint": "fp-1",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/inc-1/remediations/candidate-1/approve", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"incidentId": "inc-1", "remediationId": "candidate-1"})
	resp := httptest.NewRecorder()

	h.ApproveIncidentRemediation(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if runner.preflightPath != "/tmp/fixture-remediation/remediation.sh" || runner.runPath != runner.preflightPath {
		t.Fatalf("runner paths = preflight %q run %q", runner.preflightPath, runner.runPath)
	}
	if store.recordedAsk != "ask-1" || store.recordedIncident != "inc-1" || store.recordedFingerprint != "fp-1" || store.recordedRemediation != "candidate-1" || store.recordedApprovedBy != "operator-1" {
		t.Fatalf("recorded authorisation = ask=%q incident=%q fingerprint=%q remediation=%q actor=%q", store.recordedAsk, store.recordedIncident, store.recordedFingerprint, store.recordedRemediation, store.recordedApprovedBy)
	}
	if store.claimAsk != "ask-1" || store.claimIncident != "inc-1" || store.claimFingerprint != "fp-1" || store.claimRemediation != "candidate-1" {
		t.Fatalf("claimed authorisation = ask=%q incident=%q fingerprint=%q remediation=%q", store.claimAsk, store.claimIncident, store.claimFingerprint, store.claimRemediation)
	}
	if store.recordedOutcome == nil || store.recordedOutcome.Status != "executed" || store.recordedOutcome.AskID != "ask-1" || store.recordedOutcome.ScriptPath != runner.runPath {
		t.Fatalf("recorded outcome = %#v", store.recordedOutcome)
	}
}
