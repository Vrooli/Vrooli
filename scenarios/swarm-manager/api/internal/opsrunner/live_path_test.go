package opsrunner

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
)

func TestValidateOperationInputsHappyPath(t *testing.T) {
	contract := agentops.OperationContract{
		Inputs: []agentops.CallerInput{
			{Name: "OPERATOR_NOTE", Type: "string", Retention: "value", Sensitivity: "internal"},
		},
	}
	snap, err := validateOperationInputs(contract, map[string]any{"OPERATOR_NOTE": "focus on the flaky test"})
	if err != nil {
		t.Fatalf("validateOperationInputs: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(snap, &decoded); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if decoded["OPERATOR_NOTE"] != "focus on the flaky test" {
		t.Fatalf("snapshot lost the operator note: %s", snap)
	}
}

func TestValidateOperationInputsRejectsUnknownKey(t *testing.T) {
	contract := agentops.OperationContract{
		Inputs: []agentops.CallerInput{{Name: "OPERATOR_NOTE", Type: "string", Retention: "value"}},
	}
	_, err := validateOperationInputs(contract, map[string]any{"NOT_DECLARED": "x"})
	if !errors.Is(err, ErrInvalidCallerInput) {
		t.Fatalf("want ErrInvalidCallerInput for unknown key, got %v", err)
	}
	if !strings.Contains(err.Error(), "NOT_DECLARED") {
		t.Fatalf("error should name the unknown key: %v", err)
	}
}

func TestValidateOperationInputsRejectsMissingRequired(t *testing.T) {
	contract := agentops.OperationContract{
		Inputs: []agentops.CallerInput{{Name: "USER_MESSAGE", Type: "string", Required: true, Retention: "value"}},
	}
	_, err := validateOperationInputs(contract, map[string]any{})
	if !errors.Is(err, ErrInvalidCallerInput) {
		t.Fatalf("want ErrInvalidCallerInput for missing required, got %v", err)
	}
}

func TestValidateOperationInputsRejectsSensitiveRetention(t *testing.T) {
	contract := agentops.OperationContract{
		Inputs: []agentops.CallerInput{{Name: "SECRET", Type: "string", Sensitivity: "sensitive", Retention: "value"}},
	}
	_, err := validateOperationInputs(contract, map[string]any{"SECRET": "token"})
	if !errors.Is(err, ErrInvalidCallerInput) {
		t.Fatalf("want ErrInvalidCallerInput for sensitive input, got %v", err)
	}
}

func TestValidateOperationInputsOptionalAbsentIsOmitted(t *testing.T) {
	contract := agentops.OperationContract{
		Inputs: []agentops.CallerInput{{Name: "OPERATOR_NOTE", Type: "string", Retention: "value"}},
	}
	snap, err := validateOperationInputs(contract, map[string]any{})
	if err != nil {
		t.Fatalf("validateOperationInputs: %v", err)
	}
	if string(snap) != "{}" {
		t.Fatalf("absent optional input should yield an empty snapshot, got %s", snap)
	}
}

// fakePhaseEngine records the StartTargetPhase request and returns a fixed round.
type fakePhaseEngine struct {
	req operatingmode.StartTargetPhaseRequest
	env operatingmode.RoundEnvelope
	err error
}

func (f *fakePhaseEngine) StartTargetPhase(_ context.Context, req operatingmode.StartTargetPhaseRequest) (operatingmode.RoundEnvelope, error) {
	f.req = req
	if f.err != nil {
		return operatingmode.RoundEnvelope{}, f.err
	}
	return f.env, nil
}

func TestEngineRunStarterRoutesStructuredInputsAndReturnsRunAssociation(t *testing.T) {
	engine := &fakePhaseEngine{env: operatingmode.RoundEnvelope{RunID: "run-123", GeneratedAt: "2026-07-14T00:00:00Z"}}
	starter := NewEngineRunStarter(engine, "swarm-manager")

	effective, _ := json.Marshal(map[string]any{
		"OPERATOR_NOTE":  "keep it tight",
		"DECISION_TOPIC": "storage layout",
		"USER_QUESTION":  "should we shard by tenant?",
	})
	prep := Prepared{Mode: "backlog-clarify", EffectiveInputs: effective}
	run := RunHandle{ExecutionID: "exec-1", Target: TargetRef{Kind: agentops.TargetBacklogItem, ID: "fix/flaky"}}

	handle, err := starter.Start(context.Background(), prep, run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if handle.RunID != "run-123" {
		t.Fatalf("want run id run-123, got %q", handle.RunID)
	}
	if engine.req.Mode != "backlog-clarify" || engine.req.TargetRef != "fix/flaky" {
		t.Fatalf("engine received wrong mode/target: %+v", engine.req)
	}
	// OPERATOR_NOTE takes the operator-note channel; nothing else leaks into it.
	if engine.req.Note != "keep it tight" {
		t.Fatalf("operator note channel = %q, want the OPERATOR_NOTE value only", engine.req.Note)
	}
	// Every other typed caller input is forwarded structurally by name, without
	// collapse, for the mode's structured caller-context providers to read.
	if engine.req.OperatorInputs["DECISION_TOPIC"] != "storage layout" {
		t.Fatalf("DECISION_TOPIC lost: %+v", engine.req.OperatorInputs)
	}
	if engine.req.OperatorInputs["USER_QUESTION"] != "should we shard by tenant?" {
		t.Fatalf("USER_QUESTION lost: %+v", engine.req.OperatorInputs)
	}
	if _, ok := engine.req.OperatorInputs["OPERATOR_NOTE"]; ok {
		t.Fatalf("OPERATOR_NOTE must not double into structured inputs: %+v", engine.req.OperatorInputs)
	}
}

func TestEngineRunStarterEmptyInputsYieldEmptyChannels(t *testing.T) {
	engine := &fakePhaseEngine{env: operatingmode.RoundEnvelope{RunID: "run-1"}}
	starter := NewEngineRunStarter(engine, "swarm-manager")
	if _, err := starter.Start(context.Background(), Prepared{Mode: "backlog-research"}, RunHandle{Target: TargetRef{ID: "fix/x"}}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if engine.req.Note != "" {
		t.Fatalf("empty effective inputs should yield an empty note, got %q", engine.req.Note)
	}
	if engine.req.OperatorInputs != nil {
		t.Fatalf("empty effective inputs should yield nil operator inputs, got %+v", engine.req.OperatorInputs)
	}
}

func TestEngineRunStarterSurfacesEngineError(t *testing.T) {
	engine := &fakePhaseEngine{err: errors.New("model registry unavailable")}
	starter := NewEngineRunStarter(engine, "swarm-manager")
	_, err := starter.Start(context.Background(), Prepared{Mode: "backlog-research"}, RunHandle{Target: TargetRef{ID: "fix/x"}})
	if err == nil || !strings.Contains(err.Error(), "model registry") {
		t.Fatalf("want engine dispatch error surfaced, got %v", err)
	}
}
