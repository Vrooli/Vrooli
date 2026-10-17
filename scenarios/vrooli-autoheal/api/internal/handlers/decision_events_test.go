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
	recordCalls int
	claimCalls  int
	claimed     bool
}

func (s *oneShotClaimStore) RecordRemediationAuthorisation(_ context.Context, _, _, _, _, _ string, _ time.Time) error {
	s.recordCalls++
	return nil
}

func (s *oneShotClaimStore) ClaimRemediationAuthorisation(_ context.Context, _, _, _, _ string, _ time.Time) (bool, error) {
	s.claimCalls++
	if s.claimed {
		return false, nil
	}
	s.claimed = true
	return true, nil
}

// remediationFixture builds a handlers instance whose single incident carries one
// candidate at the given risk level, with a generated artifact and an approving
// verifier, autoheal enabled, and a stub runner that records what it executed.
func remediationFixture(t *testing.T, riskLevel string, requiresPrivilege bool) (*Handlers, *oneShotClaimStore, *approvalRunnerStub) {
	t.Helper()
	store := &oneShotClaimStore{
		mockStore: &mockStore{incident: &incidents.Incident{
			ID:             "inc-1",
			Fingerprint:    "fp-1",
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
		}},
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
		AskID: "ask-1", Answer: "approve", Actor: "operator-1",
	}})
	runner := &approvalRunnerStub{}
	h.remediationService.SetScriptRunner(runner)
	return h, store, runner
}

func approveResolution(answer string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		AskID:           "ask-1",
		State:           operatordecision.StateAnswered,
		Answer:          answer,
		AnsweredBy:      "operator-1",
		SourceEventType: incidents.RemediationDecisionEventType,
		Correlation: map[string]string{
			"incident_id":          "inc-1",
			"incident_fingerprint": "fp-1",
			"candidate_id":         "candidate-1",
			"candidate_risk_level": "low",
		},
	}
}

// Behaviour 1: a low-risk approve auto-applies through the event consumer.
func TestDecisionApproveLowRiskAutoApplies(t *testing.T) {
	h, store, runner := remediationFixture(t, "low", false)
	effect := remediationDecisionEffect{handlers: h}

	handled, err := effect.Apply(context.Background(), approveResolution("approve"))
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

	handled, err := effect.Apply(context.Background(), approveResolution("approve"))
	if err != nil || !handled {
		t.Fatalf("Apply handled=%v err=%v, want handled=true err=nil", handled, err)
	}
	// The approval was recorded and claimed (the authorisation stands for the
	// later manual apply), but nothing executed and no execution outcome exists.
	if store.recordCalls != 1 || store.claimCalls != 1 {
		t.Fatalf("expected the approval to be recorded and claimed once, record=%d claim=%d", store.recordCalls, store.claimCalls)
	}
	if runner.runPath != "" {
		t.Fatalf("expected NO execution for a high-risk candidate, runPath=%q", runner.runPath)
	}
	if store.recordedOutcome != nil {
		t.Fatalf("expected no execution outcome recorded, got %#v", store.recordedOutcome)
	}
}

// Behaviour 3: a replayed resolution (the authorisation already claimed) is
// refused — it neither executes again nor errors the webhook into a retry loop.
func TestDecisionApproveReplayIsRefused(t *testing.T) {
	h, store, runner := remediationFixture(t, "low", false)
	effect := remediationDecisionEffect{handlers: h}

	if handled, err := effect.Apply(context.Background(), approveResolution("approve")); err != nil || !handled {
		t.Fatalf("first Apply handled=%v err=%v", handled, err)
	}
	if store.recordedOutcome == nil || store.recordedOutcome.Status != "executed" {
		t.Fatalf("first apply should have executed, got %#v", store.recordedOutcome)
	}
	runner.runPath = ""
	store.recordedOutcome = nil

	// Replay the same resolution: the single-use claim now fails.
	handled, err := effect.Apply(context.Background(), approveResolution("approve"))
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

// Behaviour 4: a reject records a declined outcome and executes nothing.
func TestDecisionRejectRecordsDeclined(t *testing.T) {
	h, store, runner := remediationFixture(t, "low", false)
	effect := remediationDecisionEffect{handlers: h}

	handled, err := effect.Apply(context.Background(), approveResolution("reject"))
	if err != nil || !handled {
		t.Fatalf("Apply handled=%v err=%v, want handled=true err=nil", handled, err)
	}
	if runner.runPath != "" {
		t.Fatalf("a reject must execute nothing, runPath=%q", runner.runPath)
	}
	if store.recordCalls != 0 || store.claimCalls != 0 {
		t.Fatalf("a reject must not touch the authorisation store, record=%d claim=%d", store.recordCalls, store.claimCalls)
	}
	if store.recordedOutcome == nil || store.recordedOutcome.Status != "declined" {
		t.Fatalf("expected a declined outcome, got %#v", store.recordedOutcome)
	}
}
