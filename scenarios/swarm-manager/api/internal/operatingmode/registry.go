// Package operatingmode defines Swarm Manager's operating mode registry.
//
// Operating modes describe the unit of work, phase graph, run strategy,
// artifact policy, prompt routing, profile policy, and audit posture for a
// methodology. The registry is intentionally static: mode behavior is explicit
// code, while AgentManager cost/capability details live in scenario-owned
// profile JSON files.
package operatingmode

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type (
	Mode                    string
	ScopeKind               string
	Phase                   string
	PhaseKind               string
	RunStrategyKind         string
	BacklogSyncCapability   string
	BacklogSyncApplyMode    string
	TransitionConditionKind string
	ResultBindingKind       string
)

const (
	ModeItemLevel       Mode = "item-level"
	ModeHolisticLoop    Mode = "holistic-loop"
	ModePhasedPlanDrain Mode = "phased-plan-drain"
)

const (
	ScopeBacklogItem ScopeKind = "backlog_item"
	ScopeInitiative  ScopeKind = "initiative"
)

// PhaseKind classifies a phase by the kind of work it performs. Lane
// assignment, Operations Center column placement, and per-lane metric
// grouping all key off this axis. PhaseKind is descriptive, not restrictive:
// modes are free to define any phases they want — each phase declares its
// kind. Adding a fifth kind is a substrate change (lane plumbing, UI columns,
// authoring contract all move together) and is intentionally out of scope.
const (
	PhaseKindInvestigate PhaseKind = "investigate"
	PhaseKindExecute     PhaseKind = "execute"
	PhaseKindReview      PhaseKind = "review"
	PhaseKindReconcile   PhaseKind = "reconcile"
)

const (
	RunStrategyExistingItemFlow  RunStrategyKind = "existing_item_flow"
	RunStrategySinglePhaseRun    RunStrategyKind = "single_phase_run"
	RunStrategySequentialHandoff RunStrategyKind = "sequential_handoff"
	RunStrategyOperatorGatedLoop RunStrategyKind = "operator_gated_loop"
)

const (
	BacklogSyncReadOnly         BacklogSyncCapability = "read_only"
	BacklogSyncProposeMutations BacklogSyncCapability = "propose_mutations"
	BacklogSyncMarkComplete     BacklogSyncCapability = "mark_complete"
	BacklogSyncCreateFollowups  BacklogSyncCapability = "create_followups"
	BacklogSyncUpdateScope      BacklogSyncCapability = "update_scope"
)

// BacklogSyncApplyMode controls whether reconcile-phase proposals require an
// operator decision before mutating the backlog, or whether the system applies
// some/all proposed mutations automatically. Only operator-gated is
// implemented in v1; the auto-apply variants land as a typed
// ErrApplyModeNotImplemented at the apply seam so loud failures, not silent
// backlog edits, surface any future mode misconfiguration.
const (
	BacklogSyncApplyOperatorGated BacklogSyncApplyMode = "operator-gated"
	BacklogSyncApplyAutoSafe      BacklogSyncApplyMode = "auto-apply-safe"
	BacklogSyncApplyAutoAll       BacklogSyncApplyMode = "auto-apply-all"
)

// IsValidBacklogSyncApplyMode reports whether the given mode is one of the
// three registered apply modes. Empty is invalid; unknown values are invalid.
func IsValidBacklogSyncApplyMode(mode BacklogSyncApplyMode) bool {
	switch mode {
	case BacklogSyncApplyOperatorGated, BacklogSyncApplyAutoSafe, BacklogSyncApplyAutoAll:
		return true
	default:
		return false
	}
}

const (
	TransitionConditionAlways           TransitionConditionKind = "always"
	TransitionConditionPayloadBool      TransitionConditionKind = "payload_bool"
	TransitionConditionProgressDecision TransitionConditionKind = "progress_decision"
)

const (
	ResultBindingProgressArtifact ResultBindingKind = "progress_artifact"
)

const (
	ProfileDefault  = "swarm-manager/default"
	ProfileDeepWork = "swarm-manager/deep-work"
	ProfileAnalysis = "swarm-manager/analysis"
)

type Definition struct {
	Mode        Mode
	Label       string
	Description string
	// BestFor enumerates the work shapes this mode is the right pick for.
	// Surfaced in the picker and details page as decision support; populated
	// alongside the mode definition. Validator requires ≥1 entry per mode.
	BestFor []string
	// NotFor enumerates the work shapes this mode handles poorly. Validator
	// requires ≥1 entry per mode.
	NotFor []string
	// Tradeoffs enumerates the structural tradeoffs an operator accepts when
	// choosing this mode. Validator requires ≥1 entry per mode.
	Tradeoffs []string
	// WhenInDoubtPickInstead names a registered mode the operator should pick
	// if they're unsure between this one and another. Empty means this mode is
	// itself the safe default. Validator requires this to reference a
	// registered mode and not be self.
	WhenInDoubtPickInstead Mode
	Scope                  ScopePolicy
	PhaseGraph             PhaseGraph
	RunStrategy            RunStrategyPolicy
	Artifact               ArtifactPolicy
	Prompt                 PromptPolicy
	Profile                ProfilePolicy
	BacklogSync            BacklogSyncPolicy
	Metrics                MetricsPolicy
	Lock                   LockPolicy
	UI                     UIPolicy
}

type ScopePolicy struct {
	Kind ScopeKind
}

type PhaseGraph struct {
	StartPhase      Phase
	Terminal        []Phase
	Transitions     map[Phase][]Phase
	TransitionRules map[Phase][]TransitionRule
	Phases          map[Phase]PhaseDefinition
}

type PhaseDefinition struct {
	Phase Phase
	// Kind is the phase classification (investigate / execute / review /
	// reconcile). Lane assignment, Operations Center column placement, and
	// per-lane metric grouping all key off this axis. Must be set on every
	// initiative-scoped phase; the validator rejects empty values.
	Kind PhaseKind
	// AutoStartAfter lists phases whose successful completion should
	// automatically start this phase. Constrained to length ≤ 1 in v1; the
	// validator rejects multi-predecessor declarations to avoid
	// race-condition design pressure on the round refresher hook. The
	// auto-start path fires only on RoundStatusCompleted (not Failed /
	// Cancelled) and only after the initiative lock is released.
	AutoStartAfter   []Phase
	ActivityPurpose  string
	LockPurpose      string
	CatalogID        string
	SkillID          string
	PromptCatalog    PromptCatalogMetadata
	ProfileKey       string
	WritesRepo       bool
	OutputArtifacts  []ArtifactDefinition
	ResultBindings   []ResultBinding
	OutputContract   PhaseOutputContract
	RequiresCriteria bool
}

// IsValidPhaseKind reports whether the given kind is one of the four
// registered classifications. Empty is invalid; unknown values are invalid.
func IsValidPhaseKind(kind PhaseKind) bool {
	switch kind {
	case PhaseKindInvestigate, PhaseKindExecute, PhaseKindReview, PhaseKindReconcile:
		return true
	default:
		return false
	}
}

type PromptCatalogMetadata struct {
	Title   string
	Trigger string
	Purpose string
}

type ResultBinding struct {
	Kind     ResultBindingKind  `json:"kind"`
	Artifact ArtifactDefinition `json:"artifact"`
}

type TransitionRule struct {
	When TransitionCondition
	Next []Phase
}

type TransitionCondition struct {
	Kind             TransitionConditionKind
	PayloadKey       string
	BoolValue        bool
	ProgressDecision ProgressDecision
}

type PhaseOutputContract struct {
	RequiresStructuredResult bool
	RequiredArtifacts        []ArtifactDefinition
	RequiresProgress         bool
	RequiresVerdict          bool
	RequiresHandoff          bool
	// RequiresBacklogSync demands that the phase emit a non-nil BacklogSync
	// plan in its structured result envelope. Reconcile phases set this so a
	// misbehaving agent cannot complete the phase without producing a
	// proposal — the operator never lands in a state where an
	// auto-dispatched reconcile round leaves no backlog plan to apply.
	RequiresBacklogSync bool
}

type RunStrategyPolicy struct {
	Kind RunStrategyKind
}

type ArtifactPolicy struct {
	Root      string
	RoundRoot string
}

type ArtifactDefinition struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type PromptPolicy struct {
	CatalogPrefix string
}

type ProfilePolicy struct {
	DefaultProfileKey string
	PhaseProfiles     map[Phase]string
}

type BacklogSyncPolicy struct {
	Capabilities       []BacklogSyncCapability
	RequiresRunID      bool
	RequiresMembership bool
	EventSource        string
	// ApplyMode declares how reconcile-phase proposals are committed to the
	// backlog. Required on every initiative-scoped mode that emits proposals;
	// only BacklogSyncApplyOperatorGated is implemented in v1. The validator
	// rejects empty or unknown values for initiative-scoped modes.
	ApplyMode BacklogSyncApplyMode
}

type MetricsPolicy struct {
	EventSource            string
	ReplanSamplePhases     []Phase
	AcceptanceSamplePhases []Phase
	AcceptedVerdicts       []string
}

type LockPolicy struct {
	InitiativeExclusive bool
}

type UIPolicy struct {
	WorkspaceTabID string
}

var registry = map[Mode]Definition{
	ModeItemLevel:       itemLevelDefinition(),
	ModeHolisticLoop:    holisticLoopDefinition(),
	ModePhasedPlanDrain: phasedPlanDrainDefinition(),
}

func requiredArtifacts(artifacts []ArtifactDefinition) []ArtifactDefinition {
	required := make([]ArtifactDefinition, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Required {
			required = append(required, artifact)
		}
	}
	return required
}

func DefaultMode() Mode {
	return ModeItemLevel
}

func NormalizeMode(raw string) Mode {
	mode := Mode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return DefaultMode()
	}
	return mode
}

func ValidateMode(raw string) bool {
	_, ok := registry[NormalizeMode(raw)]
	return ok
}

func MustDefinition(mode Mode) Definition {
	def, err := DefinitionFor(mode)
	if err != nil {
		panic(err)
	}
	return def
}

func DefinitionFor(mode Mode) (Definition, error) {
	normalized := NormalizeMode(string(mode))
	def, ok := registry[normalized]
	if !ok {
		return Definition{}, fmt.Errorf("unknown operating mode %q", mode)
	}
	return def, nil
}

func Modes() []Mode {
	modes := make([]Mode, 0, len(registry))
	for mode := range registry {
		modes = append(modes, mode)
	}
	sort.Slice(modes, func(i, j int) bool { return modes[i] < modes[j] })
	return modes
}

func ModeList() string {
	modes := Modes()
	parts := make([]string, 0, len(modes))
	for _, mode := range modes {
		parts = append(parts, string(mode))
	}
	return strings.Join(parts, ", ")
}

func ValidateRegistry() error {
	return validateDefinitions(registry)
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
		if def.Scope.Kind == "" {
			return fmt.Errorf("mode %q scope kind is required", mode)
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
		if mode == ModeItemLevel {
			continue
		}
		if err := validateInitiativeModeDefinition(def); err != nil {
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

func validateInitiativeModeDefinition(def Definition) error {
	if def.PhaseGraph.StartPhase == "" {
		return fmt.Errorf("mode %q start phase is required", def.Mode)
	}
	if len(def.PhaseGraph.Phases) == 0 {
		return fmt.Errorf("mode %q phases are required", def.Mode)
	}
	if _, ok := def.PhaseGraph.Phases[def.PhaseGraph.StartPhase]; !ok {
		return fmt.Errorf("mode %q start phase %q is not registered", def.Mode, def.PhaseGraph.StartPhase)
	}
	if len(def.PhaseGraph.Terminal) == 0 {
		return fmt.Errorf("mode %q terminal phases are required", def.Mode)
	}
	if strings.TrimSpace(def.Artifact.Root) == "" || strings.TrimSpace(def.Artifact.RoundRoot) == "" {
		return fmt.Errorf("mode %q artifact root and round root are required", def.Mode)
	}
	if strings.TrimSpace(def.Prompt.CatalogPrefix) == "" {
		return fmt.Errorf("mode %q prompt catalog prefix is required", def.Mode)
	}
	if !def.Lock.InitiativeExclusive {
		return fmt.Errorf("mode %q must use initiative-exclusive locking", def.Mode)
	}
	if !IsValidBacklogSyncApplyMode(def.BacklogSync.ApplyMode) {
		return fmt.Errorf("mode %q backlog_sync apply_mode must be one of operator-gated|auto-apply-safe|auto-apply-all (got %q)", def.Mode, def.BacklogSync.ApplyMode)
	}
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
	if err := validateTransitionRules(def); err != nil {
		return err
	}
	for phase, phaseDef := range def.PhaseGraph.Phases {
		if phaseDef.Phase != phase {
			return fmt.Errorf("mode %q phase map key %q contains phase %q", def.Mode, phase, phaseDef.Phase)
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

func validateTransitionRules(def Definition) error {
	for from, rules := range def.PhaseGraph.TransitionRules {
		if _, ok := def.PhaseGraph.Phases[from]; !ok {
			return fmt.Errorf("mode %q transition rule source %q is not registered", def.Mode, from)
		}
		if len(rules) == 0 {
			return fmt.Errorf("mode %q transition rule source %q has no rules", def.Mode, from)
		}
		declaredNext := map[Phase]struct{}{}
		for _, next := range def.PhaseGraph.Transitions[from] {
			declaredNext[next] = struct{}{}
		}
		for i, rule := range rules {
			if err := validateTransitionCondition(rule.When); err != nil {
				return fmt.Errorf("mode %q transition rule %q[%d]: %w", def.Mode, from, i, err)
			}
			for _, to := range rule.Next {
				if _, ok := def.PhaseGraph.Phases[to]; !ok {
					return fmt.Errorf("mode %q transition rule %q -> %q references unregistered phase", def.Mode, from, to)
				}
				if _, ok := declaredNext[to]; !ok {
					return fmt.Errorf("mode %q transition rule %q -> %q is not declared in phase graph transitions", def.Mode, from, to)
				}
			}
		}
	}
	return nil
}

func validateTransitionCondition(condition TransitionCondition) error {
	switch condition.Kind {
	case TransitionConditionAlways:
		return nil
	case TransitionConditionPayloadBool:
		if strings.TrimSpace(condition.PayloadKey) == "" {
			return fmt.Errorf("payload bool condition requires a payload key")
		}
		return nil
	case TransitionConditionProgressDecision:
		return condition.ProgressDecision.Validate()
	default:
		return fmt.Errorf("unknown transition condition kind %q", condition.Kind)
	}
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

// RequiredProfileKeys returns every AgentManager profile key referenced by the
// operating-mode registry. Profile JSON remains the source of profile defaults;
// the registry only declares which scenario-owned keys must exist before the
// API serves traffic.
func RequiredProfileKeys() ([]string, error) {
	if err := ValidateRegistry(); err != nil {
		return nil, err
	}
	keys := map[string]struct{}{}
	for mode, def := range registry {
		if err := collectProfileKey(keys, mode, def.Profile.DefaultProfileKey); err != nil {
			return nil, err
		}
		for phase, key := range def.Profile.PhaseProfiles {
			if err := collectProfileKey(keys, mode, key); err != nil {
				return nil, fmt.Errorf("phase %q: %w", phase, err)
			}
		}
		for phase, phaseDef := range def.PhaseGraph.Phases {
			if err := collectProfileKey(keys, mode, phaseDef.ProfileKey); err != nil {
				return nil, fmt.Errorf("phase %q: %w", phase, err)
			}
		}
	}

	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out, nil
}

func collectProfileKey(keys map[string]struct{}, mode Mode, key string) error {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return nil
	}
	if !strings.HasPrefix(trimmed, "swarm-manager/") {
		return fmt.Errorf("mode %q references non-scenario-owned AgentManager profile key %q", mode, trimmed)
	}
	keys[trimmed] = struct{}{}
	return nil
}

func (d Definition) PhaseDefinition(phase Phase) (PhaseDefinition, error) {
	p, ok := d.PhaseGraph.Phases[phase]
	if !ok {
		return PhaseDefinition{}, fmt.Errorf("mode %q does not define phase %q", d.Mode, phase)
	}
	return p, nil
}

func (p MetricsPolicy) CountsReplanSample(phase Phase) bool {
	return phaseInSet(phase, p.ReplanSamplePhases)
}

func (p MetricsPolicy) CountsAcceptanceSample(phase Phase) bool {
	return phaseInSet(phase, p.AcceptanceSamplePhases)
}

func (p MetricsPolicy) IsAcceptedVerdict(verdict string) bool {
	normalized := strings.ToLower(strings.TrimSpace(verdict))
	for _, accepted := range p.AcceptedVerdicts {
		if normalized == strings.ToLower(strings.TrimSpace(accepted)) {
			return true
		}
	}
	return false
}

func phaseInSet(phase Phase, phases []Phase) bool {
	for _, candidate := range phases {
		if phase == candidate {
			return true
		}
	}
	return false
}
