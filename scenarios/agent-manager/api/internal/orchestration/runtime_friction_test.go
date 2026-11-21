package orchestration

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/toolfriction"
	"github.com/google/uuid"
)

func TestRuntimeSignalKeepsVerifyCommandSeparateFromEvaluatorID(t *testing.T) {
	call := &domain.RunEvent{ID: uuid.New(), Timestamp: time.Now()}
	signal := runtimeSignalFromInvocation(call, runsignal.InvocationFact{
		SemanticsKind: "verify",
		CommandPath:   "test-genie runs wait",
		Executable:    "test-genie",
		Outcome:       runsignal.OutcomeFailure,
	})
	if signal.CheckID != "" || signal.CommandPath != "test-genie runs wait" || signal.SemanticsKind != "verify" {
		t.Fatalf("verification signal assigned non-authoritative identity: %+v", signal)
	}
}

func TestRuntimeSignalRetainsRefusalAsGuardEvidenceWithoutInventingIdentity(t *testing.T) {
	call := &domain.RunEvent{ID: uuid.New(), Timestamp: time.Now()}
	duration := int64(41)
	signal := runtimeSignalFromInvocation(call, runsignal.InvocationFact{
		Outcome:    runsignal.OutcomeRefused,
		DurationMS: &duration,
	})
	if signal.Kind != toolfriction.RuntimeGuardCheck || signal.CheckID != "" || signal.Outcome != toolfriction.RuntimeOutcomeBlocked {
		t.Fatalf("refusal was not retained as an unknown-identity guard event: %+v", signal)
	}
	if !signal.DurationKnown || signal.DurationMS != duration {
		t.Fatalf("measured refusal duration was lost: %+v", signal)
	}
}

func TestRuntimeSignalPreservesNativeEvaluatorIdentityAndMeasuredDuration(t *testing.T) {
	call := &domain.RunEvent{ID: uuid.New(), Timestamp: time.Now()}
	duration := int64(37)
	signal := runtimeSignalFromInvocation(call, runsignal.InvocationFact{
		Outcome:         runsignal.OutcomeRefused,
		CheckID:         "scenario-dependency-analyzer:dependency-addition",
		CheckDurationMS: &duration,
	})
	if signal.Kind != toolfriction.RuntimeGuardCheck || signal.CheckID != "scenario-dependency-analyzer:dependency-addition" || signal.Check != signal.CheckID || signal.Outcome != toolfriction.RuntimeOutcomeBlocked {
		t.Fatalf("native evaluator refusal lost its authoritative identity: %+v", signal)
	}
	if !signal.DurationKnown || signal.DurationMS != duration {
		t.Fatalf("measured native-hook duration was lost: %+v", signal)
	}
}

func TestRuntimeGuardSignalRequiresStructuredSourceAndPreservesUnknowns(t *testing.T) {
	at := time.Now()
	for _, details := range []map[string]interface{}{
		{"guardCheck": map[string]interface{}{"source": "command-output", "checkId": "fake", "outcome": "blocked"}},
		{"guardCheck": map[string]interface{}{"source": "workspace-sandbox-http-403", "outcome": "blocked", "durationKnown": false}},
	} {
		payload := &domain.ErrorEventData{Details: details}
		signal, ok := runtimeGuardSignalFromError(at, payload)
		if !ok {
			if details["guardCheck"].(map[string]interface{})["source"] == "workspace-sandbox-http-403" {
				t.Fatal("structured denial was not retained")
			}
			continue
		}
		if signal.CheckID != "" || signal.Outcome != toolfriction.RuntimeOutcomeBlocked || signal.DurationKnown {
			t.Fatalf("missing historical fields were guessed: %+v", signal)
		}
	}
}
