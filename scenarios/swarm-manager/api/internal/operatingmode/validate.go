package operatingmode

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// ValidateRegistry ensures the data-backed registry is loaded and valid.
// Loading is the validation step: LoadModesFromDir runs ValidateLoadedModes
// (the same mode/phase/metric/profile invariants plus generic guard-graph
// validation) over every discovered mode, so a successful ensureRegistry means
// the whole registry passed. The server calls this at startup, turning any
// malformed mode data into a fatal, actionable startup error.
func ValidateRegistry() error {
	_, err := ensureRegistry()
	return err
}

func ValidatePromptCatalog(resolve PromptCatalogResolver) error {
	if resolve == nil {
		return fmt.Errorf("operating-mode prompt catalog resolver is required")
	}
	for _, expected := range PromptCatalogEntries() {
		entry, ok := resolve(expected.Mode, expected.Phase)
		if !ok {
			return fmt.Errorf("prompt catalog missing entry for mode %q phase %q", expected.Mode, expected.Phase)
		}
		if strings.TrimSpace(entry.CatalogID) != expected.CatalogID {
			return fmt.Errorf("prompt catalog ID mismatch for mode %q phase %q: registry=%q catalog=%q", expected.Mode, expected.Phase, expected.CatalogID, entry.CatalogID)
		}
		if strings.TrimSpace(entry.SkillID) != expected.SkillID {
			return fmt.Errorf("prompt catalog skill mismatch for mode %q phase %q: registry=%q catalog=%q", expected.Mode, expected.Phase, expected.SkillID, entry.SkillID)
		}
		if strings.TrimSpace(entry.Mode) != "" && strings.TrimSpace(entry.Mode) != expected.Mode {
			return fmt.Errorf("prompt catalog mode mismatch for mode %q phase %q: registry=%q catalog=%q", expected.Mode, expected.Phase, expected.Mode, entry.Mode)
		}
		if strings.TrimSpace(entry.Phase) != "" && strings.TrimSpace(entry.Phase) != expected.Phase {
			return fmt.Errorf("prompt catalog phase mismatch for mode %q phase %q: registry=%q catalog=%q", expected.Mode, expected.Phase, expected.Phase, entry.Phase)
		}
		if !sameStringSlice(entry.OutputPaths, expected.OutputPaths) {
			return fmt.Errorf("prompt catalog output paths mismatch for mode %q phase %q: registry=%v catalog=%v", expected.Mode, expected.Phase, expected.OutputPaths, entry.OutputPaths)
		}
	}
	return nil
}

func validateDefinitions(defs map[Mode]Definition) error {
	for mode, def := range defs {
		if def.Mode != mode {
			return fmt.Errorf("registry key %q contains definition for mode %q", mode, def.Mode)
		}
		if strings.TrimSpace(def.Label) == "" {
			return fmt.Errorf("mode %q label is required", mode)
		}
		if strings.TrimSpace(def.Description) == "" {
			return fmt.Errorf("mode %q description is required", mode)
		}
		if err := validateDecisionMetadata(defs, mode, def); err != nil {
			return err
		}
		if !IsValidTargetKind(def.Target.Kind) {
			return fmt.Errorf("mode %q target kind must be one of %s|%s|%s (got %q)", mode, TargetBacklogItem, TargetInitiative, TargetPlanExecution, def.Target.Kind)
		}
		if def.Target.Kind != TargetInitiative && (def.Target.PlanRef.Required || strings.TrimSpace(def.Target.PlanRef.Role) != "") {
			return fmt.Errorf("mode %q target.plan_ref is initiative-adapter configuration; target %q must not declare it", mode, def.Target.Kind)
		}
		if def.RunStrategy.Kind == "" {
			return fmt.Errorf("mode %q run strategy is required", mode)
		}
		if strings.TrimSpace(def.Metrics.EventSource) == "" {
			return fmt.Errorf("mode %q metrics event source is required", mode)
		}
		if err := validateMetricsPolicy(def); err != nil {
			return err
		}
		if strings.TrimSpace(def.UI.WorkspaceTabID) == "" {
			return fmt.Errorf("mode %q UI workspace tab is required", mode)
		}
		if err := validateDefinitionProfiles(mode, def); err != nil {
			return err
		}
		if !def.RunsModeRounds() {
			continue
		}
		if err := validatePhaseModeDefinition(def); err != nil {
			return err
		}
	}
	return nil
}

func validateDecisionMetadata(defs map[Mode]Definition, mode Mode, def Definition) error {
	if err := validateNonEmptyStringList(mode, "best_for", def.BestFor); err != nil {
		return err
	}
	if err := validateNonEmptyStringList(mode, "not_for", def.NotFor); err != nil {
		return err
	}
	if err := validateNonEmptyStringList(mode, "tradeoffs", def.Tradeoffs); err != nil {
		return err
	}
	if def.WhenInDoubtPickInstead != "" {
		if def.WhenInDoubtPickInstead == mode {
			return fmt.Errorf("mode %q when_in_doubt_pick_instead cannot reference itself", mode)
		}
		if _, ok := defs[def.WhenInDoubtPickInstead]; !ok {
			return fmt.Errorf("mode %q when_in_doubt_pick_instead references unregistered mode %q", mode, def.WhenInDoubtPickInstead)
		}
	}
	return nil
}

func validateNonEmptyStringList(mode Mode, field string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("mode %q %s requires at least one entry", mode, field)
	}
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("mode %q %s entry %d cannot be blank", mode, field, i)
		}
	}
	return nil
}

func validateMetricsPolicy(def Definition) error {
	if len(def.Metrics.AcceptanceSamplePhases) > 0 && len(def.Metrics.AcceptedVerdicts) == 0 {
		return fmt.Errorf("mode %q metrics acceptance policy requires accepted verdict values", def.Mode)
	}
	if len(def.Metrics.AcceptedVerdicts) > 0 && len(def.Metrics.AcceptanceSamplePhases) == 0 {
		return fmt.Errorf("mode %q metrics accepted verdict values require acceptance sample phases", def.Mode)
	}
	for _, phase := range def.Metrics.ReplanSamplePhases {
		if _, ok := def.PhaseGraph.Phases[phase]; !ok {
			return fmt.Errorf("mode %q metrics replan phase %q is not registered", def.Mode, phase)
		}
	}
	for _, phase := range def.Metrics.AcceptanceSamplePhases {
		phaseDef, ok := def.PhaseGraph.Phases[phase]
		if !ok {
			return fmt.Errorf("mode %q metrics acceptance phase %q is not registered", def.Mode, phase)
		}
		if !phaseDef.OutputContract.RequiresVerdict {
			return fmt.Errorf("mode %q metrics acceptance phase %q must require a verdict", def.Mode, phase)
		}
	}
	for _, verdict := range def.Metrics.AcceptedVerdicts {
		if strings.TrimSpace(verdict) == "" {
			return fmt.Errorf("mode %q metrics accepted verdict values cannot be blank", def.Mode)
		}
	}
	return nil
}

func validateDefinitionProfiles(mode Mode, def Definition) error {
	if err := collectProfileKey(map[string]struct{}{}, mode, def.Profile.DefaultProfileKey); err != nil {
		return err
	}
	for phase, key := range def.Profile.PhaseProfiles {
		if err := collectProfileKey(map[string]struct{}{}, mode, key); err != nil {
			return fmt.Errorf("mode %q phase %q profile policy: %w", mode, phase, err)
		}
	}
	return nil
}

func validatePhaseModeDefinition(def Definition) error {
	if err := validateModeTopLevel(def); err != nil {
		return err
	}
	if err := validateModeTransitions(def); err != nil {
		return err
	}
	for phase, phaseDef := range def.PhaseGraph.Phases {
		if err := validateModePhase(def, phase, phaseDef); err != nil {
			return err
		}
	}
	return nil
}

// validateModeTopLevel checks the mode-wide invariants that do not depend on
// iterating phases or transitions: start phase, phase presence, terminal
// presence, artifact roots, prompt prefix, locking, and backlog-sync policy.
func validateModeTopLevel(def Definition) error {
	if def.PhaseGraph.StartPhase == "" {
		return fmt.Errorf("mode %q start phase is required", def.Mode)
	}
	if len(def.PhaseGraph.Phases) == 0 {
		return fmt.Errorf("mode %q phases are required", def.Mode)
	}
	if _, ok := def.PhaseGraph.Phases[def.PhaseGraph.StartPhase]; !ok {
		return fmt.Errorf("mode %q start phase %q is not registered", def.Mode, def.PhaseGraph.StartPhase)
	}
	// A mode must be able to stop: either it declares terminal phases, or at
	// least one of its edges is a guarded stop (a matched guard with no target
	// — e.g. the generic drain's complete/blocked routes). A single-phase
	// self-loop mode has no terminal phase at all; its stops are guarded.
	if len(def.PhaseGraph.Terminal) == 0 && !hasGuardedStop(def) {
		return fmt.Errorf("mode %q must declare terminal phases or at least one guarded-stop transition (a guard routing to no phase)", def.Mode)
	}
	if strings.TrimSpace(def.Artifact.Root) == "" || strings.TrimSpace(def.Artifact.RoundRoot) == "" {
		return fmt.Errorf("mode %q artifact root and round root are required", def.Mode)
	}
	if strings.TrimSpace(def.Prompt.CatalogPrefix) == "" {
		return fmt.Errorf("mode %q prompt catalog prefix is required", def.Mode)
	}
	if def.Target.Kind == TargetInitiative {
		if !def.Lock.InitiativeExclusive {
			return fmt.Errorf("mode %q must use initiative-exclusive locking", def.Mode)
		}
		if !IsValidBacklogSyncApplyMode(def.BacklogSync.ApplyMode) {
			return fmt.Errorf("mode %q backlog_sync apply_mode must be one of operator-gated|auto-apply-safe|auto-apply-all (got %q)", def.Mode, def.BacklogSync.ApplyMode)
		}
	}
	if def.Target.PlanRef.Required && strings.TrimSpace(def.Target.PlanRef.Role) != PlanRefRoleOperatingModePlan {
		return fmt.Errorf("mode %q target.plan_ref role must be %q when required", def.Mode, PlanRefRoleOperatingModePlan)
	}
	return nil
}

// hasGuardedStop reports whether any phase declares a guarded stop: a
// transition whose guard can match but routes to no phase.
func hasGuardedStop(def Definition) bool {
	for _, guards := range def.PhaseGraph.Guards {
		for _, gt := range guards {
			if len(gt.To) == 0 {
				return true
			}
		}
	}
	return false
}

// validateModeTransitions checks that every terminal phase and every phase
// graph transition references a registered phase.
func validateModeTransitions(def Definition) error {
	for _, terminal := range def.PhaseGraph.Terminal {
		if _, ok := def.PhaseGraph.Phases[terminal]; !ok {
			return fmt.Errorf("mode %q terminal phase %q is not registered", def.Mode, terminal)
		}
	}
	for from, nextPhases := range def.PhaseGraph.Transitions {
		if _, ok := def.PhaseGraph.Phases[from]; !ok {
			return fmt.Errorf("mode %q transition source %q is not registered", def.Mode, from)
		}
		for _, to := range nextPhases {
			if _, ok := def.PhaseGraph.Phases[to]; !ok {
				return fmt.Errorf("mode %q transition %q -> %q references unregistered phase", def.Mode, from, to)
			}
		}
	}
	return nil
}

// validateModePhase validates a single phase entry: map-key consistency,
// prompt-catalog metadata, purpose tokens, profile policy, and the artifact,
// result-binding, output-contract, kind, and auto-start sub-policies.
func validateModePhase(def Definition, phase Phase, phaseDef PhaseDefinition) error {
	if phaseDef.Phase != phase {
		return fmt.Errorf("mode %q phase map key %q contains phase %q", def.Mode, phase, phaseDef.Phase)
	}
	if phaseDef.Delegated() {
		// A delegated phase carries no execution surface of its own (no
		// prompt catalog entry, purposes, profile, artifacts, or output
		// contract) — the sub-mode's phases own all of that and are validated
		// in their own mode. Delegation-specific semantics (sub-mode
		// existence, nesting, target compatibility, guard fields) are checked
		// by validateDelegations/validateGuardGraph over the full mode set.
		if !IsValidPhaseKind(phaseDef.Kind) {
			return fmt.Errorf("mode %q phase %q kind must be one of investigate|execute|review|reconcile (got %q)", def.Mode, phase, phaseDef.Kind)
		}
		if err := validateEvidenceRequirements(phaseDef); err != nil {
			return fmt.Errorf("mode %q phase %q: %w", def.Mode, phase, err)
		}
		return validatePhaseAutoStartAfter(def, phaseDef)
	}
	if strings.TrimSpace(phaseDef.CatalogID) == "" || strings.TrimSpace(phaseDef.SkillID) == "" {
		return fmt.Errorf("mode %q phase %q prompt catalog ID and skill ID are required", def.Mode, phase)
	}
	if strings.TrimSpace(phaseDef.PromptCatalog.Title) == "" || strings.TrimSpace(phaseDef.PromptCatalog.Trigger) == "" || strings.TrimSpace(phaseDef.PromptCatalog.Purpose) == "" {
		return fmt.Errorf("mode %q phase %q prompt catalog title, trigger, and purpose are required", def.Mode, phase)
	}
	if !isValidPurposeToken(phaseDef.ActivityPurpose) {
		return fmt.Errorf("mode %q phase %q activity purpose must be a lowercase snake-case token", def.Mode, phase)
	}
	if !isValidPurposeToken(phaseDef.LockPurpose) {
		return fmt.Errorf("mode %q phase %q lock purpose must be a lowercase snake-case token", def.Mode, phase)
	}
	if got := def.Profile.PhaseProfiles[phase]; got != phaseDef.ProfileKey {
		return fmt.Errorf("mode %q phase %q profile mismatch: policy=%q phase=%q", def.Mode, phase, got, phaseDef.ProfileKey)
	}
	if err := collectProfileKey(map[string]struct{}{}, def.Mode, phaseDef.ProfileKey); err != nil {
		return fmt.Errorf("mode %q phase %q: %w", def.Mode, phase, err)
	}
	if err := validatePhaseArtifactPolicy(def, phaseDef); err != nil {
		return err
	}
	if err := validatePhaseResultBindings(def, phaseDef); err != nil {
		return err
	}
	if err := validatePhaseOutputContract(phaseDef); err != nil {
		return fmt.Errorf("mode %q phase %q: %w", def.Mode, phase, err)
	}
	if !IsValidPhaseKind(phaseDef.Kind) {
		return fmt.Errorf("mode %q phase %q kind must be one of investigate|execute|review|reconcile (got %q)", def.Mode, phase, phaseDef.Kind)
	}
	if err := validatePhaseAutoStartAfter(def, phaseDef); err != nil {
		return err
	}
	if err := validateEvidenceRequirements(phaseDef); err != nil {
		return fmt.Errorf("mode %q phase %q: %w", def.Mode, phase, err)
	}
	return nil
}

func validateEvidenceRequirements(phaseDef PhaseDefinition) error {
	seen := map[string]struct{}{}
	for i, requirement := range phaseDef.EvidenceRequirements {
		ledger := requirement.LedgerRequirement()
		if err := ledger.Validate(); err != nil {
			return fmt.Errorf("evidence_requirements[%d]: %w", i, err)
		}
		fields, err := json.Marshal(ledger.MatchFields)
		if err != nil {
			return fmt.Errorf("evidence_requirements[%d] encode match_fields: %w", i, err)
		}
		key := strings.Join([]string{ledger.SubjectKind, ledger.Action, ledger.ProducerID, string(ledger.MinConfidence), string(fields)}, "\x00")
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("evidence_requirements[%d] duplicates an earlier requirement", i)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// validatePhaseAutoStartAfter enforces v1 constraints on auto-start
// declarations: at most one predecessor, the predecessor must exist in the
// mode's phase graph, and a phase cannot list itself. These rules keep the
// round-refresher auto-dispatch hook deterministic — multi-predecessor races
// are deferred to a future plan that revisits the locking model.
func validatePhaseAutoStartAfter(def Definition, phaseDef PhaseDefinition) error {
	if len(phaseDef.AutoStartAfter) == 0 {
		return nil
	}
	if len(phaseDef.AutoStartAfter) > 1 {
		return fmt.Errorf("mode %q phase %q auto_start_after supports at most one predecessor in v1 (got %d)", def.Mode, phaseDef.Phase, len(phaseDef.AutoStartAfter))
	}
	target := phaseDef.AutoStartAfter[0]
	if target == phaseDef.Phase {
		return fmt.Errorf("mode %q phase %q auto_start_after cannot reference itself", def.Mode, phaseDef.Phase)
	}
	if _, ok := def.PhaseGraph.Phases[target]; !ok {
		return fmt.Errorf("mode %q phase %q auto_start_after references unregistered phase %q", def.Mode, phaseDef.Phase, target)
	}
	return nil
}

func isValidPurposeToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if unicode.IsLower(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return false
	}
	return true
}

func validatePhaseArtifactPolicy(def Definition, phaseDef PhaseDefinition) error {
	for _, artifact := range phaseDef.OutputArtifacts {
		if _, err := cleanModeRelativePath(def, artifact.Path); err != nil {
			return fmt.Errorf("mode %q phase %q artifact %q: %w", def.Mode, phaseDef.Phase, artifact.Path, err)
		}
	}
	return nil
}

func validatePhaseResultBindings(def Definition, phaseDef PhaseDefinition) error {
	declared := map[string]ArtifactDefinition{}
	for _, artifact := range phaseDef.OutputArtifacts {
		declared[filepath.ToSlash(filepath.Clean(artifact.Path))] = artifact
	}
	for i, binding := range phaseDef.ResultBindings {
		if binding.Kind != ResultBindingProgressArtifact {
			return fmt.Errorf("mode %q phase %q result binding %d has unknown kind %q", def.Mode, phaseDef.Phase, i, binding.Kind)
		}
		clean, err := cleanModeRelativePath(def, binding.Artifact.Path)
		if err != nil {
			return fmt.Errorf("mode %q phase %q result binding %d artifact %q: %w", def.Mode, phaseDef.Phase, i, binding.Artifact.Path, err)
		}
		if _, ok := declared[clean]; !ok {
			return fmt.Errorf("mode %q phase %q result binding %d artifact %q is not declared as a phase output", def.Mode, phaseDef.Phase, i, binding.Artifact.Path)
		}
	}
	return nil
}

func validatePhaseOutputContract(phaseDef PhaseDefinition) error {
	contract := phaseDef.OutputContract
	if !contract.RequiresStructuredResult {
		return fmt.Errorf("output contract must require structured results")
	}
	declared := map[string]ArtifactDefinition{}
	for _, artifact := range phaseDef.OutputArtifacts {
		declared[filepath.ToSlash(filepath.Clean(artifact.Path))] = artifact
	}
	for _, artifact := range contract.RequiredArtifacts {
		path := filepath.ToSlash(filepath.Clean(strings.TrimSpace(artifact.Path)))
		declaredArtifact, ok := declared[path]
		if path == "." || !ok {
			return fmt.Errorf("required contract artifact %q is not declared as a phase output", artifact.Path)
		}
		if !declaredArtifact.Required {
			return fmt.Errorf("required contract artifact %q is not marked required in phase outputs", artifact.Path)
		}
	}
	return nil
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}
