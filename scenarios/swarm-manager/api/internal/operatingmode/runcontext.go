package operatingmode

import (
	"fmt"
	"sort"
	"strings"

	"swarm-manager/internal/operatingmode/promptcatalog"
)

// This file is the generic run-context of the v2 unit-of-work decoupling
// (EXECUTION-MODES.md, D1/D1b): the per-run identity and state the engine
// threads through a mode run, plus the composed Reads contract. The variables
// available to a phase are the union of the generic-base provider (below) and
// the mode's target adapter (target.go) — there is no scope switch and no
// conditional emptiness anywhere in assembly.

// Generic-base read names: scope-agnostic reads every phase of every mode may
// declare, regardless of target kind.
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

// BaseReadNames returns the read names the generic-base provider supplies:
// mode/phase identity, run strategy, round number, profile key, operator note,
// accumulated prior rounds/handoffs, mode artifacts, and the shared framework
// snippets.
func BaseReadNames() []string {
	return []string{
		ReadOperatingMode,
		ReadModeLabel,
		ReadPhase,
		ReadRunStrategy,
		ReadRoundNumber,
		ReadAgentProfileKey,
		ReadOperatorNote,
		ReadPriorRoundsJSON,
		ReadModeArtifactsJSON,
		promptcatalog.BacklogSyncProposalVariableKey,
		promptcatalog.ElasticSliceVariableKey,
	}
}

// AvailableReadNames returns the full composed read vocabulary for a target
// kind: base ∪ adapter, sorted. This is the set the loader validates declared
// phase reads against and the set surfaced as per-phase reads metadata.
func AvailableReadNames(kind TargetKind) ([]string, error) {
	adapterNames, err := TargetReadNames(kind)
	if err != nil {
		return nil, err
	}
	names := append(BaseReadNames(), adapterNames...)
	sort.Strings(names)
	return names, nil
}

// RunContext is the generic per-run identity and state the engine threads
// through a mode run: which mode and phase, the resolved target instance, and
// the durable rounds and artifacts accumulated so far. It is target-agnostic;
// the target adapter fills in the target-specific parts.
type RunContext struct {
	Def       Definition
	PhaseDef  PhaseDefinition
	Target    TargetInstance
	Artifacts []ArtifactSnapshot
	Rounds    []RoundEnvelope
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
	adapter, err := rc.Adapter()
	if err != nil {
		return nil, err
	}
	reads := rc.baseReads(round, note)
	for name, value := range adapter.Reads(rc.Target) {
		reads[name] = value
	}
	return reads, nil
}

// DeclaredReads resolves exactly the phase's declared read contract from the
// composed available set. A declared read the composed providers do not supply
// is a typed error — the runtime twin of the loader's read-side validation.
func (rc RunContext) DeclaredReads(round RoundEnvelope, note string) (map[string]string, error) {
	available, err := rc.AvailableReads(round, note)
	if err != nil {
		return nil, err
	}
	reads := make(map[string]string, len(rc.PhaseDef.Reads))
	for _, name := range rc.PhaseDef.Reads {
		value, ok := available[name]
		if !ok {
			return nil, fmt.Errorf("mode %q phase %q declares read %q, which target %q does not provide", rc.Def.Mode, rc.PhaseDef.Phase, name, rc.Def.Target.Kind)
		}
		reads[name] = value
	}
	return reads, nil
}

// baseReads is the generic-base provider: scope-agnostic reads assembled from
// the run context alone.
func (rc RunContext) baseReads(round RoundEnvelope, note string) map[string]string {
	return map[string]string{
		ReadOperatingMode:     string(rc.Def.Mode),
		ReadModeLabel:         rc.Def.Label,
		ReadPhase:             string(rc.PhaseDef.Phase),
		ReadRunStrategy:       string(rc.Def.RunStrategy.Kind),
		ReadRoundNumber:       fmt.Sprintf("%d", round.Round),
		ReadAgentProfileKey:   rc.PhaseDef.ProfileKey,
		ReadOperatorNote:      strings.TrimSpace(note),
		ReadPriorRoundsJSON:   mustJSON(rc.Rounds),
		ReadModeArtifactsJSON: mustJSON(rc.Artifacts),
		promptcatalog.BacklogSyncProposalVariableKey: promptcatalog.BacklogSyncProposalSnippet(),
		promptcatalog.ElasticSliceVariableKey:        promptcatalog.ElasticSliceSnippet(),
	}
}

// PhaseReadsSummary groups a phase's declared reads by supplying provider —
// generic base vs target adapter — for catalog/UI rendering. Derived from the
// declared contract, never from a hardcoded category list.
type PhaseReadsSummary struct {
	Base   []string `json:"base,omitempty"`
	Target []string `json:"target,omitempty"`
}

// summarizePhaseReads splits a phase's declared reads into base-provided and
// adapter-provided groups, preserving declaration order.
func summarizePhaseReads(declared []string) PhaseReadsSummary {
	base := map[string]struct{}{}
	for _, name := range BaseReadNames() {
		base[name] = struct{}{}
	}
	out := PhaseReadsSummary{}
	for _, name := range declared {
		if _, ok := base[name]; ok {
			out.Base = append(out.Base, name)
			continue
		}
		out.Target = append(out.Target, name)
	}
	return out
}
