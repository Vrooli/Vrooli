package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/remediation"
)

// oneShotClaimStore claims the authorisation exactly once: the first claim
// succeeds, every later claim fails, modelling the single-use SQL guarantee so a
// replayed resolution is refused.
type oneShotClaimStore struct {
	*mockStore
	recordCalls  int
	claimCalls   int
	claimed      bool
	declineCalls int
}

func (s *oneShotClaimStore) RecordRemediationAuthorisation(_ context.Context, _, _, _, _ string, _ int, _, _, _, _, _ string, _ time.Time) error {
	s.recordCalls++
	return nil
}

func (s *oneShotClaimStore) ClaimRemediationAuthorisation(_ context.Context, _, _, _, _ string, _ int, _, _, _, _ string, _ time.Time) (bool, error) {
	s.claimCalls++
	if s.claimed {
		return false, nil
	}
	s.claimed = true
	return true, nil
}

func (s *oneShotClaimStore) RecordRemediationDeclineIfCurrent(_ context.Context, _, _, _, _, _ string, _ int, _, _ string, outcome incidents.Outcome) (bool, error) {
	s.declineCalls++
	s.recordedOutcome = &outcome
	return true, nil
}

// remediationFixture builds a handlers instance whose single incident carries one
// candidate at the given risk level, with a generated artifact and an approving
// verifier, autoheal enabled, and a stub runner that records what it executed.
func remediationFixture(t *testing.T, riskLevel string, requiresPrivilege bool) (*Handlers, *oneShotClaimStore, *approvalRunnerStub) {
	t.Helper()
	incident := &incidents.Incident{
		ID:             "inc-1",
		Fingerprint:    "fp-1",
		Severity:       incidents.SeverityCritical,
		Status:         incidents.StatusOpen,
		SourceCheckIDs: []string{"host-kernel-module-drift"},
		RemediationCandidates: []incidents.RemediationCandidate{{
			ID:                "candidate-1",
			Applicability:     "applicable",
			RiskLevel:         riskLevel,
			RequiresPrivilege: requiresPrivilege,
		}},
		RemediationArtifacts: []incidents.RemediationArtifact{{
			RemediationID: "candidate-1",
			Path:          "/tmp/fixture-remediation",
		}},
	}
	incident.EventCount = 1
	incident.RemediationDecisionGeneration = 1
	incident.RemediationDecisionEventID = "decision-event-1"
	incident.RemediationDecisionCandidateID = "candidate-1"
	expires := time.Now().UTC().Add(time.Hour)
	incident.RemediationDecisionExpiresAt = &expires
	digest, err := incidents.RemediationDecisionSnapshotDigest(*incident, incident.RemediationCandidates[0])
	if err != nil {
		t.Fatal(err)
	}
	incident.RemediationCandidateDigest = digest
	store := &oneShotClaimStore{mockStore: &mockStore{incident: incident}}
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
		AskID: "ask-1", Answer: "approve", Actor: "operator-1",
		RequesterRef: incident.RemediationDecisionEventID, SourceEventType: incidents.RemediationDecisionEventType,
	}})
	runner := &approvalRunnerStub{}
	h.remediationService.SetScriptRunner(runner)
	return h, store, runner
}

func approveResolution(answer, digest string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		AskID:           "ask-1",
		State:           operatordecision.StateAnswered,
		Answer:          answer,
		AnsweredBy:      "operator-1",
		SourceEventType: incidents.RemediationDecisionEventType,
		RequesterRef:    "decision-event-1",
		Correlation: map[string]string{
			"incident_id":          "inc-1",
			"incident_fingerprint": "fp-1",
			"candidate_id":         "candidate-1",
			"candidate_risk_level": "low",
			"generation":           "1",
			"candidate_digest":     digest,
		},
	}
}

// Behaviour 1: a low-risk approve auto-applies through the event consumer.
func TestDecisionApproveLowRiskAutoApplies(t *testing.T) {
	h, store, runner := remediationFixture(t, "low", false)
	effect := remediationDecisionEffect{handlers: h}

	handled, err := effect.Apply(context.Background(), approveResolution("approve", store.incident.RemediationCandidateDigest))
	if err != nil || !handled {
		t.Fatalf("Apply handled=%v err=%v, want handled=true err=nil", handled, err)
	}
	if runner.runPath != "/tmp/fixture-remediation/remediation.sh" {
		t.Fatalf("expected the low-risk candidate to execute, runPath=%q", runner.runPath)
	}
	if store.recordedOutcome == nil || store.recordedOutcome.Status != "executed" {
		t.Fatalf("expected an executed outcome, got %#v", store.recordedOutcome)
	}
}

// Behaviour 2: a high-risk / privileged approve records the approval but does NOT
// auto-apply; the operator must still trigger the manual apply step.
func TestDecisionApproveHighRiskRecordsButDoesNotApply(t *testing.T) {
	h, store, runner := remediationFixture(t, "moderate", true)
	effect := remediationDecisionEffect{handlers: h}

	handled, err := effect.Apply(context.Background(), approveResolution("approve", store.incident.RemediationCandidateDigest))
	if err != nil || !handled {
		t.Fatalf("Apply handled=%v err=%v, want handled=true err=nil", handled, err)
	}
	// The webhook records the exact Ask but must leave its single-use execution
	// claim available for the explicit manual apply route.
	if store.recordCalls != 1 || store.claimCalls != 0 {
		t.Fatalf("high-risk approval record/claim calls = %d/%d, want 1/0", store.recordCalls, store.claimCalls)
	}
	if runner.runPath != "" || store.recordedOutcome != nil {
		t.Fatalf("high-risk webhook executed unexpectedly: run=%q outcome=%#v", runner.runPath, store.recordedOutcome)
	}

	outcome, err := h.ApproveRemediationFromDecision(context.Background(), remediationApprovalInput{
		IncidentID: "inc-1", CandidateID: "candidate-1", AskID: "ask-1",
		IncidentFingerprint: "fp-1", RequesterRef: "decision-event-1", Generation: 1,
		CandidateDigest: store.incident.RemediationCandidateDigest,
	})
	if err != nil || !outcome.Applied {
		t.Fatalf("manual apply after high-risk webhook = %#v, err=%v", outcome, err)
	}
	if store.claimCalls != 1 || runner.runPath != "/tmp/fixture-remediation/remediation.sh" {
		t.Fatalf("manual apply did not consume the original approval once: claims=%d run=%q", store.claimCalls, runner.runPath)
	}

	runner.runPath = ""
	if handled, err := effect.Apply(context.Background(), approveResolution("approve", store.incident.RemediationCandidateDigest)); err != nil || !handled {
		t.Fatalf("duplicate webhook handled=%v err=%v", handled, err)
	}
	if store.claimCalls != 1 || runner.runPath != "" {
		t.Fatalf("duplicate webhook must not claim or execute again: claims=%d run=%q", store.claimCalls, runner.runPath)
	}
}

// Behaviour 3: a replayed resolution (the authorisation already claimed) is
// refused — it neither executes again nor errors the webhook into a retry loop.
func TestDecisionApproveReplayIsRefused(t *testing.T) {
	h, store, runner := remediationFixture(t, "low", false)
	effect := remediationDecisionEffect{handlers: h}

	if handled, err := effect.Apply(context.Background(), approveResolution("approve", store.incident.RemediationCandidateDigest)); err != nil || !handled {
		t.Fatalf("first Apply handled=%v err=%v", handled, err)
	}
	if store.recordedOutcome == nil || store.recordedOutcome.Status != "executed" {
		t.Fatalf("first apply should have executed, got %#v", store.recordedOutcome)
	}
	runner.runPath = ""
	store.recordedOutcome = nil

	// Replay the same resolution: the single-use claim now fails.
	handled, err := effect.Apply(context.Background(), approveResolution("approve", store.incident.RemediationCandidateDigest))
	if err != nil {
		t.Fatalf("replay should be a benign ack, got err=%v", err)
	}
	if !handled {
		t.Fatalf("replay should be acknowledged (handled=true) to avoid a retry loop")
	}
	if runner.runPath != "" {
		t.Fatalf("replay must NOT execute again, runPath=%q", runner.runPath)
	}
	if store.recordedOutcome != nil {
		t.Fatalf("replay must NOT record a second execution, got %#v", store.recordedOutcome)
	}
}

func TestDecisionApprovalRejectsWrongRequesterOrDigestWithoutEffects(t *testing.T) {
	for _, tc := range []struct {
		name      string
		requester string
		digest    string
	}{
		{name: "foreign requester reference", requester: "foreign-event", digest: "current"},
		{name: "stale candidate digest", requester: "decision-event-1", digest: "stale"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, store, runner := remediationFixture(t, "low", false)
			digest := store.incident.RemediationCandidateDigest
			resolution := approveResolution("approve", digest)
			resolution.RequesterRef = tc.requester
			if tc.digest != "current" {
				resolution.Correlation["candidate_digest"] = tc.digest
			}
			handled, err := (remediationDecisionEffect{handlers: h}).Apply(context.Background(), resolution)
			if err == nil || !handled {
				t.Fatalf("Apply handled=%v err=%v, want handled error for mismatched binding", handled, err)
			}
			if store.recordCalls != 0 || store.claimCalls != 0 || store.recordedOutcome != nil || runner.runPath != "" {
				t.Fatalf("mismatched decision had effects: records=%d claims=%d outcome=%#v run=%q", store.recordCalls, store.claimCalls, store.recordedOutcome, runner.runPath)
			}
		})
	}
}

func TestDecisionApprovalRejectsAskWithoutCurrentSourceBinding(t *testing.T) {
	for _, tc := range []struct{ name, sourceType, requester string }{
		{name: "wrong source event", sourceType: "unrelated.request.v1", requester: "decision-event-1"},
		{name: "missing requester reference", sourceType: incidents.RemediationDecisionEventType, requester: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, store, runner := remediationFixture(t, "low", false)
			h.SetRemediationAskVerifier(approvalVerifierStub{approval: remediation.AskApproval{
				AskID: "ask-1", Answer: "approve", Actor: "operator-1", SourceEventType: tc.sourceType, RequesterRef: tc.requester,
			}})
			_, err := h.ApproveRemediationFromDecision(context.Background(), remediationApprovalInput{
				IncidentID: "inc-1", CandidateID: "candidate-1", AskID: "ask-1", IncidentFingerprint: "fp-1",
				RequesterRef: "decision-event-1", Generation: 1, CandidateDigest: store.incident.RemediationCandidateDigest,
			})
			if err == nil {
				t.Fatal("approval with unbound Ask unexpectedly succeeded")
			}
			if store.recordCalls != 0 || store.claimCalls != 0 || runner.runPath != "" || store.recordedOutcome != nil {
				t.Fatalf("unbound Ask had effects: records=%d claims=%d run=%q outcome=%#v", store.recordCalls, store.claimCalls, runner.runPath, store.recordedOutcome)
			}
		})
	}
}

// Behaviour 4: a reject records a declined outcome and executes nothing.
func TestDecisionRejectRecordsDeclined(t *testing.T) {
	h, store, runner := remediationFixture(t, "low", false)
	effect := remediationDecisionEffect{handlers: h}

	handled, err := effect.Apply(context.Background(), approveResolution("reject", store.incident.RemediationCandidateDigest))
	if err != nil || !handled {
		t.Fatalf("Apply handled=%v err=%v, want handled=true err=nil", handled, err)
	}
	if runner.runPath != "" {
		t.Fatalf("a reject must execute nothing, runPath=%q", runner.runPath)
	}
	if store.recordCalls != 0 || store.claimCalls != 0 || store.declineCalls != 1 {
		t.Fatalf("a reject must only write one decline, records=%d claims=%d declines=%d", store.recordCalls, store.claimCalls, store.declineCalls)
	}
	if store.recordedOutcome == nil || store.recordedOutcome.Status != "declined" {
		t.Fatalf("expected a declined outcome, got %#v", store.recordedOutcome)
	}
}
