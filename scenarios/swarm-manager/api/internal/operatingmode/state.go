package operatingmode

import (
	"fmt"
	"strings"
)

type PhaseAction struct {
	Phase     Phase  `json:"phase"`
	Startable bool   `json:"startable"`
	Reason    string `json:"reason,omitempty"`
	Next      bool   `json:"next,omitempty"`
}

type PhaseStateInput struct {
	Definition         Definition
	Rounds             []RoundEnvelope
	AcceptanceCriteria []string
	RequireRunStrategy bool
}

func ComputePhaseActions(input PhaseStateInput) []PhaseAction {
	def := input.Definition
	phases := orderedPhases(def)
	active := activeRound(input.Rounds)
	allowed := allowedNextPhases(def, input.Rounds)

	actions := make([]PhaseAction, 0, len(phases))
	for _, phase := range phases {
		action := PhaseAction{Phase: phase}
		phaseDef, err := def.PhaseDefinition(phase)
		switch {
		case err != nil:
			action.Reason = err.Error()
		case active != nil:
			action.Reason = fmt.Sprintf("round %03d is still %s", active.Round, active.Status)
		case !allowed[phase]:
			action.Reason = phaseNotAllowedReason(def, input.Rounds, phase)
		case phaseDef.RequiresCriteria && len(input.AcceptanceCriteria) == 0:
			action.Reason = fmt.Sprintf("phase %q requires initiative acceptance criteria", phase)
		default:
			action.Startable = true
			action.Next = true
			if input.RequireRunStrategy {
				if reason := validateRunStrategy(def, input.Rounds, phase); reason != "" {
					action.Startable = false
					action.Next = false
					action.Reason = reason
				}
			}
		}
		actions = append(actions, action)
	}
	return actions
}

func ValidatePhaseStart(def Definition, rounds []RoundEnvelope, phase Phase, acceptanceCriteria []string) error {
	phase = Phase(strings.TrimSpace(string(phase)))
	if _, err := def.PhaseDefinition(phase); err != nil {
		return err
	}
	for _, action := range ComputePhaseActions(PhaseStateInput{
		Definition:         def,
		Rounds:             rounds,
		AcceptanceCriteria: acceptanceCriteria,
		RequireRunStrategy: true,
	}) {
		if action.Phase != phase {
			continue
		}
		if action.Startable {
			return nil
		}
		if strings.TrimSpace(action.Reason) == "" {
			return fmt.Errorf("phase %q is not startable", phase)
		}
		return fmt.Errorf("phase %q is not startable: %s", phase, action.Reason)
	}
	return fmt.Errorf("phase %q is not registered for mode %q", phase, def.Mode)
}

func orderedPhases(def Definition) []Phase {
	phases := make([]Phase, 0, len(def.PhaseGraph.Phases))
	for phase := range def.PhaseGraph.Phases {
		phases = append(phases, phase)
	}
	order := map[Phase]int{def.PhaseGraph.StartPhase: 0}
	seen := map[Phase]bool{}
	var visit func(Phase)
	visit = func(phase Phase) {
		if seen[phase] {
			return
		}
		seen[phase] = true
		if _, ok := order[phase]; !ok {
			order[phase] = len(order)
		}
		for _, next := range def.PhaseGraph.Transitions[phase] {
			visit(next)
		}
	}
	if def.PhaseGraph.StartPhase != "" {
		visit(def.PhaseGraph.StartPhase)
	}
	for _, phase := range phases {
		if _, ok := order[phase]; !ok {
			order[phase] = len(order)
		}
	}
	for i := 0; i < len(phases)-1; i++ {
		for j := i + 1; j < len(phases); j++ {
			if order[phases[j]] < order[phases[i]] || (order[phases[j]] == order[phases[i]] && phases[j] < phases[i]) {
				phases[i], phases[j] = phases[j], phases[i]
			}
		}
	}
	return phases
}

func activeRound(rounds []RoundEnvelope) *RoundEnvelope {
	for i := range rounds {
		if isRoundActive(rounds[i]) {
			return &rounds[i]
		}
	}
	return nil
}

func lastCompletedRound(rounds []RoundEnvelope) *RoundEnvelope {
	for i := len(rounds) - 1; i >= 0; i-- {
		if rounds[i].Status == RoundStatusCompleted {
			return &rounds[i]
		}
	}
	return nil
}

func allowedNextPhases(def Definition, rounds []RoundEnvelope) map[Phase]bool {
	allowed := map[Phase]bool{}
	if activeRound(rounds) != nil {
		return allowed
	}
	last := lastCompletedRound(rounds)
	if last == nil {
		if def.PhaseGraph.StartPhase != "" {
			allowed[def.PhaseGraph.StartPhase] = true
		}
		return allowed
	}

	for _, phase := range nextPhasesForCompletedRound(def, *last) {
		allowed[phase] = true
	}
	return allowed
}

func nextPhasesForCompletedRound(def Definition, last RoundEnvelope) []Phase {
	from := Phase(last.Phase)
	if rules := def.PhaseGraph.TransitionRules[from]; len(rules) > 0 {
		payload := RoundPayload(last.Payload)
		for _, rule := range rules {
			if rule.When.Matches(payload) {
				return append([]Phase(nil), rule.Next...)
			}
		}
		return nil
	}
	return append([]Phase(nil), def.PhaseGraph.Transitions[from]...)
}

func (condition TransitionCondition) Matches(payload RoundPayloadView) bool {
	switch condition.Kind {
	case TransitionConditionAlways:
		return true
	case TransitionConditionPayloadBool:
		raw, ok := payload.get(condition.PayloadKey)
		value, isBool := raw.(bool)
		return ok && isBool && value == condition.BoolValue
	case TransitionConditionProgressDecision:
		progress, ok := payload.Progress()
		decision := ProgressDecision(strings.TrimSpace(string(progress.Decision)))
		return ok && decision == condition.ProgressDecision
	default:
		return false
	}
}

func roundProgress(round RoundEnvelope) (ProgressState, bool) {
	return RoundPayload(round.Payload).Progress()
}

func validateRunStrategy(def Definition, rounds []RoundEnvelope, phase Phase) string {
	switch def.RunStrategy.Kind {
	case RunStrategyExistingItemFlow:
		return "item-level phases are owned by the existing backlog execution flow"
	case RunStrategySequentialHandoff:
		if phase == def.PhaseGraph.StartPhase {
			return ""
		}
		last := lastCompletedRound(rounds)
		if last == nil {
			return "no completed handoff round exists"
		}
		if hasDurableHandoffContext(*last) {
			return ""
		}
		return fmt.Sprintf("phase %q requires durable handoff or progress context from round %03d", phase, last.Round)
	default:
		return ""
	}
}

func hasDurableHandoffContext(round RoundEnvelope) bool {
	if len(round.Handoffs) > 0 || len(round.ArtifactUpdates) > 0 {
		return true
	}
	_, ok := roundProgress(round)
	return ok
}

func phaseNotAllowedReason(def Definition, rounds []RoundEnvelope, phase Phase) string {
	last := lastCompletedRound(rounds)
	if last == nil {
		if def.PhaseGraph.StartPhase == "" {
			return "mode does not define a start phase"
		}
		return fmt.Sprintf("first phase must be %q", def.PhaseGraph.StartPhase)
	}
	if _, ok := def.PhaseGraph.Transitions[Phase(last.Phase)]; !ok {
		return fmt.Sprintf("phase %q has no registered transition to %q", last.Phase, phase)
	}
	if len(def.PhaseGraph.TransitionRules[Phase(last.Phase)]) > 0 {
		return fmt.Sprintf("last completed phase %q result does not transition to %q", last.Phase, phase)
	}
	return fmt.Sprintf("last completed phase %q does not transition to %q", last.Phase, phase)
}
