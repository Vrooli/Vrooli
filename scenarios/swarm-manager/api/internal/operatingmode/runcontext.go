package operatingmode

// This file is the generic run-context of the v2 unit-of-work decoupling
// (EXECUTION-MODES.md, D1/D1b): the per-run identity and state the engine
// threads through a mode run, plus the composed Reads contract. The variables
// available to a phase are the union of the generic-base provider (below) and
// the mode's target adapter (target.go) — there is no scope switch and no
// conditional emptiness anywhere in assembly.

// Prompt-facing aliases retained as stable public vocabulary. Their logical
// identities and sources live exclusively in each mode's input_contract.
const (
	ReadOperatingMode     = "OPERATING_MODE"
	ReadModeLabel         = "MODE_LABEL"
	ReadPhase             = "PHASE"
	ReadRunStrategy       = "RUN_STRATEGY"
	ReadRoundNumber       = "ROUND_NUMBER"
	ReadAgentProfileKey   = "AGENT_PROFILE_KEY"
	ReadOperatorNote      = "OPERATOR_NOTE"
	ReadPriorRoundsJSON   = "PRIOR_ROUNDS_JSON"
	ReadModeArtifactsJSON = "MODE_ARTIFACTS_JSON"
)

// RunContext is the generic per-run identity and state the engine threads
// through a mode run: which mode and phase, the resolved target instance, and
// the durable rounds and artifacts accumulated so far. It is target-agnostic;
// the target adapter fills in the target-specific parts.
type RunContext struct {
	Def       Definition
	PhaseDef  PhaseDefinition
	Execution *OperatingModeExecution
	Target    TargetInstance
	Artifacts []ArtifactSnapshot
	Rounds    []RoundEnvelope
	// OperatorInputs are optional structured caller-context strings the run was
	// started with, keyed by operation caller-input name (e.g. USER_QUESTION,
	// CONTEXT_PATHS). Unlike the per-action operator note, these are set once at
	// run start and belong to the run identity, so they ride on the context
	// rather than a render parameter. The structured caller-context generic
	// providers (generic.user_question, generic.gap_report, …) read from here;
	// they are always optional, so an absent key resolves to an empty string.
	OperatorInputs map[string]string
}

// operatorInput returns the structured caller-context string supplied for name,
// or "" when the run carries none (the optional-input contract every structured
// caller-context provider honors, matching generic.operator_note's empty note).
func (rc RunContext) operatorInput(name string) string {
	if rc.OperatorInputs == nil {
		return ""
	}
	return rc.OperatorInputs[name]
}

// Adapter returns the run's target adapter.
func (rc RunContext) Adapter() (TargetAdapter, error) {
	return AdapterFor(rc.Def.Target.Kind)
}

// OwnershipKey is the exclusive-run lock identity for this run's target
// instance. Initiative targets keep the initiative name; plan targets use a
// plan-scoped key, so a plan-target run never touches an initiative lock.
func (rc RunContext) OwnershipKey() (string, error) {
	return OwnershipKeyFor(rc.Def.Target.Kind, rc.Target.ID)
}

// AvailableReads assembles the full composed read map for this run:
// generic-base ∪ target adapter, by union. No branching on target kind.
func (rc RunContext) AvailableReads(round RoundEnvelope, note string) (map[string]string, error) {
	return rc.resolveCompiledInputs(round, note, false)
}

// DeclaredReads resolves exactly the phase's declared read contract from the
// composed available set. A declared read the composed providers do not supply
// is a typed error — the runtime twin of the loader's read-side validation.
func (rc RunContext) DeclaredReads(round RoundEnvelope, note string) (map[string]string, error) {
	return rc.resolveCompiledInputs(round, note, true)
}

// PhaseReadsSummary groups a phase's declared reads by supplying provider —
// generic base vs target adapter — for catalog/UI rendering. Derived from the
// declared contract, never from a hardcoded category list.
type PhaseReadsSummary struct {
	Base   []string `json:"base,omitempty"`
	Target []string `json:"target,omitempty"`
}

// summarizePhaseReads splits a phase's declared aliases by their authored
// source binding, preserving phase declaration order. Caller, default, and
// derived values are grouped with the non-target inputs until the public
// projection grows dedicated source-kind buckets.
func summarizePhaseReads(def Definition, phase PhaseDefinition) PhaseReadsSummary {
	inputByAlias := make(map[string]string, len(def.InputContract.Aliases))
	for _, alias := range def.InputContract.Aliases {
		inputByAlias[alias.Name] = alias.InputID
	}
	sourceByInput := make(map[string]InputSourceKind, len(def.InputContract.Sources))
	for _, source := range def.InputContract.Sources {
		sourceByInput[source.InputID] = source.Kind
	}
	out := PhaseReadsSummary{}
	for _, name := range phase.Reads {
		if sourceByInput[inputByAlias[name]] == InputSourceTargetAdapter {
			out.Target = append(out.Target, name)
		} else {
			out.Base = append(out.Base, name)
		}
	}
	return out
}
