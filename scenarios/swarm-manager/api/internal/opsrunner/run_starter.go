package opsrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/operatingmode"
)

// PhaseEngine is the live operating-mode dispatch seam. It is exactly
// operatingmode.Service.StartTargetPhase: the engine's single non-initiative
// phase-start entry point, which compiles and pins the execution, renders the
// mode prompt, and spawns the agent through the agentactivity chokepoint. It is
// kept as an interface so the runner stays testable with a fake engine and free
// of a hard dependency on the concrete Service.
type PhaseEngine interface {
	StartTargetPhase(ctx context.Context, req operatingmode.StartTargetPhaseRequest) (operatingmode.RoundEnvelope, error)
}

// EngineRunStarter is the production RunStarter: it starts one live operating-mode
// round on a target through StartTargetPhase and returns immediately with the run
// association, leaving the round running until Runner.CommitResult delivers its
// resolved outcome. It performs no classification and no transition — the round's
// eventual completion is routed back to CommitResult by the engine's round-refresh
// seam.
//
// The operation's typed caller inputs reach the engine faithfully, without any
// lossy collapse: OPERATOR_NOTE goes to StartTargetPhaseRequest.Note (the
// operator-note channel -> generic.operator_note), and every other string caller
// input is forwarded by name in StartTargetPhaseRequest.OperatorInputs, which the
// mode's structured caller-context generic providers read (generic.user_question,
// generic.gap_report, …). Neither channel is StartTargetPhaseRequest.Inputs: that
// feeds the mode's typed caller-input contract, which the backlog modes leave
// empty, so forwarding operation inputs there would trip the unknown-key guard and
// break the empty-set engine invariant.
type EngineRunStarter struct {
	engine      PhaseEngine
	requestedBy string
	// noteInput is the single operation-input name whose string value becomes the
	// operator note. Every other string input is forwarded structurally by name.
	noteInput string
}

// NewEngineRunStarter builds a live run-starter over a phase engine.
func NewEngineRunStarter(engine PhaseEngine, requestedBy string) *EngineRunStarter {
	return &EngineRunStarter{
		engine:      engine,
		requestedBy: strings.TrimSpace(requestedBy),
		noteInput:   "OPERATOR_NOTE",
	}
}

// Start dispatches one live round and returns its run association.
func (s *EngineRunStarter) Start(ctx context.Context, prep Prepared, run RunHandle) (StartHandle, error) {
	if s.engine == nil {
		return StartHandle{}, fmt.Errorf("opsrunner: no live phase engine configured")
	}
	note, operatorInputs, err := s.routeInputs(prep.EffectiveInputs)
	if err != nil {
		return StartHandle{}, err
	}
	env, err := s.engine.StartTargetPhase(ctx, operatingmode.StartTargetPhaseRequest{
		Mode:           prep.Mode,
		TargetRef:      run.Target.ID,
		Note:           note,
		OperatorInputs: operatorInputs,
		RequestedBy:    s.requestedBy,
	})
	if err != nil {
		return StartHandle{}, err
	}
	return StartHandle{
		RunID:     strings.TrimSpace(env.RunID),
		CreatedAt: strings.TrimSpace(env.GeneratedAt),
	}, nil
}

// routeInputs splits the operation's validated caller inputs into the two engine
// channels without loss: OPERATOR_NOTE becomes the operator note, and every other
// present, non-empty string input is forwarded by name as a structured operator
// input. Non-string inputs are skipped (the current caller-input vocabulary is
// entirely string-typed); operatorInputs is nil when no structured input applies.
func (s *EngineRunStarter) routeInputs(effective json.RawMessage) (string, map[string]string, error) {
	if len(effective) == 0 {
		return "", nil, nil
	}
	var inputs map[string]any
	if err := json.Unmarshal(effective, &inputs); err != nil {
		return "", nil, fmt.Errorf("opsrunner: decode effective inputs for engine routing: %w", err)
	}
	var note string
	var operatorInputs map[string]string
	for name, v := range inputs {
		str, ok := v.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(str)
		if trimmed == "" {
			continue
		}
		if name == s.noteInput {
			note = trimmed
			continue
		}
		if operatorInputs == nil {
			operatorInputs = map[string]string{}
		}
		operatorInputs[name] = trimmed
	}
	return note, operatorInputs, nil
}
