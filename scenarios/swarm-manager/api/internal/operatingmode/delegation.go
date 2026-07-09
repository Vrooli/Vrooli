package operatingmode

import (
	"fmt"
	"strings"
)

// This file implements phase delegation — the `executed_by` composition model
// (EXECUTION-MODES.md, D3). A phase may declare `executed_by: <sub-mode-id>`;
// the engine then runs the sub-mode's loop *as* that phase:
//
//   - Each round of the delegated phase is one round of the sub-mode's loop,
//     executed with the sub-mode's prompt, reads, declared output, and
//     classified edges — but persisted under the PARENT run (parent mode,
//     parent scope id, parent phase id) and holding the parent's ownership
//     lock. The round payload records `delegated_mode` and `delegated_phase`
//     so every downstream surface can resolve the effective execution
//     contract.
//   - After a delegated round completes, the SUB-mode's guards are evaluated
//     first: an onward sub-route (e.g. the drain's `continue` self-loop) keeps
//     the delegated phase in progress — the next startable phase is the parent
//     phase again, continuing the sub-loop from where it left off. A sub-mode
//     stop (guarded stop, terminal sub-phase) ends the delegation: the round's
//     resolved output — including any edge-classified routing field, e.g.
//     `progress` ∈ complete/blocked — becomes the delegating phase's result,
//     and the PARENT's transitions route on it. The parent stays the routing
//     SSOT; the sub-mode never routes among parent phases.
//   - Target context flows from the parent: a same-target-kind sub-mode
//     receives the parent's resolved target instance unchanged; an
//     initiative-target mode may delegate to a `plan-manager-plan`-target
//     sub-mode when it declares `target.plan_ref.required` — the initiative's
//     bound plan is the sub-mode's unit of work. No other combination is
//     compatible (loader-rejected).
//
// Deliberate limits (loader-enforced): exactly one composition level (a
// sub-mode containing its own delegated phase is rejected), no
// self-delegation, no unknown sub-modes, no runtime inheritance.

// Round payload keys recording which sub-mode/sub-phase actually executed a
// delegated round. Written at phase start; read by every surface that needs
// the effective execution contract (resolution, classification, routing,
// prompt render).
const (
	payloadDelegatedMode  = "delegated_mode"
	payloadDelegatedPhase = "delegated_phase"
)

// delegationSubDefinition resolves a delegated phase's sub-mode from the
// process registry. Load-time validation guarantees registered modes only
// delegate to registered, non-nesting sub-modes; a miss here is a typed error,
// not a panic, because draft definitions can also flow through the runtime.
func delegationSubDefinition(phaseDef PhaseDefinition) (Definition, error) {
	sub, err := DefinitionFor(phaseDef.ExecutedBy)
	if err != nil {
		return Definition{}, fmt.Errorf("phase %q delegates to unregistered sub-mode %q: %w", phaseDef.Phase, phaseDef.ExecutedBy, err)
	}
	return sub, nil
}

// validateDelegations is the full-set semantic validation of every delegated
// phase: the sub-mode must exist in the set, run mode rounds, not be the mode
// itself, and contain no delegated phases of its own (one level deep); the
// delegating mode's target must be able to supply the sub-mode's target
// context; and the delegating phase's own transitions must route on fields the
// sub-mode's terminal outcome can actually carry.
func validateDelegations(defs map[Mode]Definition) error {
	for _, mode := range SortedModes(defs) {
		def := defs[mode]
		if !def.RunsModeRounds() {
			continue
		}
		for phase, phaseDef := range def.PhaseGraph.Phases {
			if !phaseDef.Delegated() {
				continue
			}
			if phaseDef.ExecutedBy == mode {
				return fmt.Errorf("mode %q phase %q cannot delegate to its own mode (self-delegation)", mode, phase)
			}
			sub, ok := defs[phaseDef.ExecutedBy]
			if !ok {
				return fmt.Errorf("mode %q phase %q delegates to unknown sub-mode %q", mode, phase, phaseDef.ExecutedBy)
			}
			if !sub.RunsModeRounds() {
				return fmt.Errorf("mode %q phase %q delegates to %q, which runs no mode rounds (run strategy %s) and cannot execute a phase", mode, phase, sub.Mode, sub.RunStrategy.Kind)
			}
			for subPhase, subPhaseDef := range sub.PhaseGraph.Phases {
				if subPhaseDef.Delegated() {
					return fmt.Errorf("mode %q phase %q delegates to %q, whose phase %q itself declares executed_by=%q: composition is exactly one level deep (no nesting)", mode, phase, sub.Mode, subPhase, subPhaseDef.ExecutedBy)
				}
			}
			if err := validateDelegationTargetCompatibility(def, sub); err != nil {
				return fmt.Errorf("mode %q phase %q: %w", mode, phase, err)
			}
		}
	}
	return nil
}

// validateDelegationTargetCompatibility enforces the target-context flow
// contract: the delegating mode must be able to supply the sub-mode's target
// from its own resolved target instance.
func validateDelegationTargetCompatibility(parent, sub Definition) error {
	if parent.Target.Kind == sub.Target.Kind {
		return nil
	}
	if parent.Target.Kind == TargetInitiative && sub.Target.Kind == TargetPlanManagerPlan {
		if !parent.Target.PlanRef.Required {
			return fmt.Errorf("delegating to plan-target sub-mode %q requires target.plan_ref.required on the initiative-target mode (the initiative's bound plan supplies the sub-mode's plan)", sub.Mode)
		}
		return nil
	}
	return fmt.Errorf("target %q cannot supply the target context of sub-mode %q (target %q); compatible combinations: same target kind, or initiative → plan-manager-plan with a required bound plan_ref", parent.Target.Kind, sub.Mode, sub.Target.Kind)
}

// delegatedOutcomeFields returns the top-level output field segments a
// sub-mode's terminal outcome can carry: every sub-phase's declared output
// fields plus every edge-classified routing field. The delegating phase's own
// guards are validated against this set — the composition twin of the regular
// declared-output guard check.
func delegatedOutcomeFields(sub Definition) map[string]struct{} {
	out := map[string]struct{}{}
	for _, subPhaseDef := range sub.PhaseGraph.Phases {
		for field := range declaredTopLevelFields(subPhaseDef) {
			out[field] = struct{}{}
		}
		if c := subPhaseDef.TransitionClassification; c != nil {
			out[topFieldSegment(c.Field)] = struct{}{}
		}
	}
	return out
}

// delegatedRoundSubPhase reads the sub-phase a delegated round executed from
// its payload marker.
func delegatedRoundSubPhase(round RoundEnvelope) (Phase, bool) {
	raw, ok := RoundPayload(round.Payload).get(payloadDelegatedPhase)
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return Phase(strings.TrimSpace(value)), true
}

// effectiveRoundExecution resolves the definition + phase whose contract
// actually executed a round: for a round of a delegated phase, the sub-mode's
// definition and the sub-phase recorded on the round payload; otherwise the
// round's own mode and phase. Every surface that consumes a round's execution
// contract (resolution ladder, edge classification, artifact contract) goes
// through this seam so delegated rounds resolve against the sub-mode's
// declared output, not the delegating phase's (empty) one.
func effectiveRoundExecution(def Definition, round RoundEnvelope) (Definition, PhaseDefinition, error) {
	phaseDef, err := def.PhaseDefinition(Phase(round.Phase))
	if err != nil {
		return Definition{}, PhaseDefinition{}, err
	}
	if !phaseDef.Delegated() {
		return def, phaseDef, nil
	}
	sub, err := delegationSubDefinition(phaseDef)
	if err != nil {
		return Definition{}, PhaseDefinition{}, err
	}
	subPhase, ok := delegatedRoundSubPhase(round)
	if !ok {
		return Definition{}, PhaseDefinition{}, fmt.Errorf("round %03d of delegated phase %q carries no %s payload marker", round.Round, round.Phase, payloadDelegatedPhase)
	}
	subPhaseDef, err := sub.PhaseDefinition(subPhase)
	if err != nil {
		return Definition{}, PhaseDefinition{}, fmt.Errorf("round %03d of delegated phase %q: %w", round.Round, round.Phase, err)
	}
	return sub, subPhaseDef, nil
}

// delegationRouteForLookup evaluates the sub-mode's guards for a completed
// delegated round's output. continuing=true with a sub-phase means the
// delegation is still in progress (the parent phase runs again, continuing
// the sub-loop there); continuing=false means the sub-mode reached a stop
// (guarded stop, terminal sub-phase, or no matching guard) and the parent's
// guards take over on the same output.
func delegationRouteForLookup(sub Definition, subPhase Phase, lookup FieldLookup) (Phase, bool, error) {
	for _, terminal := range sub.PhaseGraph.Terminal {
		if terminal == subPhase {
			return "", false, nil
		}
	}
	next, matched := selectNextPhases(sub, subPhase, lookup)
	if !matched || len(next) == 0 {
		return "", false, nil
	}
	if len(next) > 1 {
		return "", false, fmt.Errorf("sub-mode %q phase %q routed to multiple targets %v; delegated sub-routes must be deterministic", sub.Mode, subPhase, next)
	}
	return next[0], true, nil
}

// nextDelegatedSubPhase decides which sub-phase the next round of a delegated
// phase runs: the sub-route continuation of the last completed delegated
// round, or the sub-mode's start phase when the delegation is entered fresh
// (first visit, or re-entry after a prior delegation run stopped and the
// parent routed back).
func nextDelegatedSubPhase(sub Definition, parentPhase Phase, rounds []RoundEnvelope) (Phase, error) {
	for i := len(rounds) - 1; i >= 0; i-- {
		round := rounds[i]
		if round.Status != RoundStatusCompleted || Phase(round.Phase) != parentPhase {
			continue
		}
		subPhase, ok := delegatedRoundSubPhase(round)
		if !ok {
			break
		}
		next, continuing, err := delegationRouteForLookup(sub, subPhase, NewMapFieldLookup(round.Payload))
		if err != nil {
			return "", err
		}
		if continuing {
			return next, nil
		}
		break
	}
	return sub.PhaseGraph.StartPhase, nil
}

// deriveSubTarget produces the sub-mode's target instance from the parent's
// resolved one — the target-context flow of composition. Same target kind
// passes the parent instance through unchanged; an initiative parent
// delegating to a plan-manager-plan sub-mode hands over its bound plan
// context as the sub-mode's unit of work.
func deriveSubTarget(parentDef, sub Definition, parent TargetInstance) (TargetInstance, error) {
	if parentDef.Target.Kind == sub.Target.Kind {
		return parent, nil
	}
	if parentDef.Target.Kind == TargetInitiative && sub.Target.Kind == TargetPlanManagerPlan {
		plan := parent.Plan
		if plan == nil || plan.Missing {
			return TargetInstance{}, fmt.Errorf("delegation to plan-target sub-mode %q requires the initiative %q to have a resolved bound plan (plan_ref)", sub.Mode, parent.ID)
		}
		id := firstNonEmpty(plan.ExecutionID, plan.PlanID)
		if id == "" && plan.PlanRef != nil {
			id = firstNonEmpty(plan.PlanRef.PlanID, plan.PlanRef.Slug)
		}
		if id == "" {
			return TargetInstance{}, fmt.Errorf("delegation to plan-target sub-mode %q: initiative %q bound plan carries no execution/plan id", sub.Mode, parent.ID)
		}
		return TargetInstance{
			Kind:  TargetPlanManagerPlan,
			ID:    id,
			Title: fmt.Sprintf("Plan %s", id),
			Plan:  plan,
		}, nil
	}
	return TargetInstance{}, fmt.Errorf("target %q cannot supply the target context of sub-mode %q (target %q)", parentDef.Target.Kind, sub.Mode, sub.Target.Kind)
}

// collectDelegatedRunContext builds the execution run-context for the next
// round of a delegated phase: the sub-mode's definition and next sub-phase,
// the sub target derived from the parent's, and the parent run's accumulated
// rounds/artifacts as continuity context. Persistence, locking, and routing
// stay keyed to the PARENT run context; only prompt/reads/output resolution
// use this one.
func collectDelegatedRunContext(parent RunContext) (RunContext, error) {
	sub, err := delegationSubDefinition(parent.PhaseDef)
	if err != nil {
		return RunContext{}, err
	}
	subPhase, err := nextDelegatedSubPhase(sub, parent.PhaseDef.Phase, parent.Rounds)
	if err != nil {
		return RunContext{}, err
	}
	subPhaseDef, err := sub.PhaseDefinition(subPhase)
	if err != nil {
		return RunContext{}, err
	}
	target, err := deriveSubTarget(parent.Def, sub, parent.Target)
	if err != nil {
		return RunContext{}, err
	}
	return RunContext{
		Def:       sub,
		PhaseDef:  subPhaseDef,
		Target:    target,
		Artifacts: parent.Artifacts,
		Rounds:    parent.Rounds,
	}, nil
}
