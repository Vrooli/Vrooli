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

	"swarm-manager/internal/agentactivity"
)

type (
	Mode                  string
	ScopeKind             string
	Phase                 string
	RunStrategyKind       string
	BacklogSyncCapability string
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

const (
	ProfileDefault  = "swarm-manager/default"
	ProfileDeepWork = "swarm-manager/deep-work"
	ProfileAnalysis = "swarm-manager/analysis"
)

type Definition struct {
	Mode        Mode
	Label       string
	Scope       ScopePolicy
	PhaseGraph  PhaseGraph
	RunStrategy RunStrategyPolicy
	Artifact    ArtifactPolicy
	Prompt      PromptPolicy
	Profile     ProfilePolicy
	BacklogSync BacklogSyncPolicy
	Metrics     MetricsPolicy
	Lock        LockPolicy
	UI          UIPolicy
}

type ScopePolicy struct {
	Kind ScopeKind
}

type PhaseGraph struct {
	StartPhase  Phase
	Terminal    []Phase
	Transitions map[Phase][]Phase
	Phases      map[Phase]PhaseDefinition
}

type PhaseDefinition struct {
	Phase            Phase
	ActivityPurpose  string
	LockPurpose      string
	CatalogID        string
	SkillID          string
	ProfileKey       string
	WritesRepo       bool
	OutputArtifacts  []ArtifactDefinition
	OutputContract   PhaseOutputContract
	RequiresCriteria bool
}

type PhaseOutputContract struct {
	RequiresStructuredResult bool
	RequiredArtifacts        []ArtifactDefinition
	RequiresProgress         bool
	RequiresVerdict          bool
	RequiresHandoff          bool
}

type RunStrategyPolicy struct {
	Kind RunStrategyKind
}

type ArtifactPolicy struct {
	Root      string
	RoundRoot string
}

type ArtifactDefinition struct {
	Path        string
	ContentType string
	Required    bool
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
}

type MetricsPolicy struct {
	EventSource string
}

type LockPolicy struct {
	InitiativeExclusive bool
}

type UIPolicy struct {
	WorkspaceTabID string
}

var registry = map[Mode]Definition{
	ModeItemLevel: {
		Mode:  ModeItemLevel,
		Label: "Item Level",
		Scope: ScopePolicy{Kind: ScopeBacklogItem},
		RunStrategy: RunStrategyPolicy{
			Kind: RunStrategyExistingItemFlow,
		},
		Profile: ProfilePolicy{DefaultProfileKey: ProfileDefault},
		BacklogSync: BacklogSyncPolicy{
			Capabilities: []BacklogSyncCapability{BacklogSyncReadOnly},
			EventSource:  "item-level",
		},
		Metrics: MetricsPolicy{EventSource: "item-level"},
		UI:      UIPolicy{WorkspaceTabID: "info"},
	},
	ModeHolisticLoop: {
		Mode:  ModeHolisticLoop,
		Label: "Holistic Loop",
		Scope: ScopePolicy{Kind: ScopeInitiative},
		PhaseGraph: PhaseGraph{
			StartPhase: "investigate",
			Terminal:   []Phase{"review"},
			Transitions: map[Phase][]Phase{
				"investigate": {"plan"},
				"plan":        {"execute"},
				"execute":     {"review", "investigate"},
				"review":      {"investigate"},
			},
			Phases: map[Phase]PhaseDefinition{
				"investigate": holisticPhase("investigate", ProfileDeepWork, false, []ArtifactDefinition{{Path: "modes/holistic-loop/findings.md", ContentType: "text/markdown", Required: true}}),
				"plan":        holisticPhase("plan", ProfileDeepWork, false, []ArtifactDefinition{{Path: "modes/holistic-loop/initiative-plan.md", ContentType: "text/markdown", Required: true}}),
				"execute":     holisticPhase("execute", ProfileDeepWork, true, nil),
				"review":      holisticPhase("review", ProfileAnalysis, false, nil),
			},
		},
		RunStrategy: RunStrategyPolicy{Kind: RunStrategyOperatorGatedLoop},
		Artifact:    ArtifactPolicy{Root: "modes/holistic-loop", RoundRoot: "modes/holistic-loop/rounds"},
		Prompt:      PromptPolicy{CatalogPrefix: "swarm-manager-holistic-loop"},
		Profile: ProfilePolicy{
			DefaultProfileKey: ProfileDeepWork,
			PhaseProfiles: map[Phase]string{
				"investigate": ProfileDeepWork,
				"plan":        ProfileDeepWork,
				"execute":     ProfileDeepWork,
				"review":      ProfileAnalysis,
			},
		},
		BacklogSync: BacklogSyncPolicy{
			Capabilities:       []BacklogSyncCapability{BacklogSyncReadOnly, BacklogSyncProposeMutations, BacklogSyncMarkComplete, BacklogSyncCreateFollowups, BacklogSyncUpdateScope},
			RequiresRunID:      true,
			RequiresMembership: true,
			EventSource:        string(ModeHolisticLoop),
		},
		Metrics: MetricsPolicy{EventSource: string(ModeHolisticLoop)},
		Lock:    LockPolicy{InitiativeExclusive: true},
		UI:      UIPolicy{WorkspaceTabID: "operating-mode"},
	},
	ModePhasedPlanDrain: {
		Mode:  ModePhasedPlanDrain,
		Label: "Phased Plan Drain",
		Scope: ScopePolicy{Kind: ScopeInitiative},
		PhaseGraph: PhaseGraph{
			StartPhase: "prepare_plan",
			Terminal:   []Phase{"review"},
			Transitions: map[Phase][]Phase{
				"prepare_plan":      {"execute_next"},
				"execute_next":      {"classify_progress"},
				"classify_progress": {"execute_next", "prepare_plan", "review"},
				"review":            {"prepare_plan"},
			},
			Phases: map[Phase]PhaseDefinition{
				"prepare_plan":      phasedPhase("prepare_plan", ProfileDeepWork, false, []ArtifactDefinition{{Path: "modes/phased-plan-drain/phased-plan.md", ContentType: "text/markdown", Required: true}}),
				"execute_next":      phasedPhase("execute_next", ProfileDeepWork, true, nil),
				"classify_progress": phasedPhase("classify_progress", ProfileAnalysis, false, []ArtifactDefinition{{Path: "modes/phased-plan-drain/progress.json", ContentType: "application/json", Required: true}}),
				"review":            phasedPhase("review", ProfileAnalysis, false, nil),
			},
		},
		RunStrategy: RunStrategyPolicy{Kind: RunStrategySequentialHandoff},
		Artifact:    ArtifactPolicy{Root: "modes/phased-plan-drain", RoundRoot: "modes/phased-plan-drain/rounds"},
		Prompt:      PromptPolicy{CatalogPrefix: "swarm-manager-phased-plan"},
		Profile: ProfilePolicy{
			DefaultProfileKey: ProfileDeepWork,
			PhaseProfiles: map[Phase]string{
				"prepare_plan":      ProfileDeepWork,
				"execute_next":      ProfileDeepWork,
				"classify_progress": ProfileAnalysis,
				"review":            ProfileAnalysis,
			},
		},
		BacklogSync: BacklogSyncPolicy{
			Capabilities:       []BacklogSyncCapability{BacklogSyncReadOnly, BacklogSyncProposeMutations, BacklogSyncMarkComplete, BacklogSyncCreateFollowups, BacklogSyncUpdateScope},
			RequiresRunID:      true,
			RequiresMembership: true,
			EventSource:        string(ModePhasedPlanDrain),
		},
		Metrics: MetricsPolicy{EventSource: string(ModePhasedPlanDrain)},
		Lock:    LockPolicy{InitiativeExclusive: true},
		UI:      UIPolicy{WorkspaceTabID: "operating-mode"},
	},
}

func holisticPhase(phase Phase, profile string, writesRepo bool, artifacts []ArtifactDefinition) PhaseDefinition {
	purposes := map[Phase]agentactivity.Purpose{
		"investigate": agentactivity.PurposeHolisticLoopInvestigate,
		"plan":        agentactivity.PurposeHolisticLoopPlan,
		"execute":     agentactivity.PurposeHolisticLoopExecute,
		"review":      agentactivity.PurposeHolisticLoopReview,
	}
	return PhaseDefinition{
		Phase:           phase,
		ActivityPurpose: string(purposes[phase]),
		LockPurpose:     string(purposes[phase]),
		CatalogID:       "swarm-manager-holistic-loop-" + string(phase),
		SkillID:         "swarm-manager-holistic-loop-" + string(phase),
		ProfileKey:      profile,
		WritesRepo:      writesRepo,
		OutputArtifacts: artifacts,
		OutputContract: PhaseOutputContract{
			RequiresStructuredResult: true,
			RequiredArtifacts:        requiredArtifacts(artifacts),
			RequiresVerdict:          phase == "review",
		},
		RequiresCriteria: phase == "review",
	}
}

func phasedPhase(phase Phase, profile string, writesRepo bool, artifacts []ArtifactDefinition) PhaseDefinition {
	skillPhase := strings.ReplaceAll(string(phase), "_", "-")
	if phase == "prepare_plan" {
		skillPhase = "prepare"
	}
	purposes := map[Phase]agentactivity.Purpose{
		"prepare_plan":      agentactivity.PurposePhasedPlanPrepare,
		"execute_next":      agentactivity.PurposePhasedPlanExecuteNext,
		"classify_progress": agentactivity.PurposePhasedPlanClassifyProgress,
		"review":            agentactivity.PurposePhasedPlanReview,
	}
	return PhaseDefinition{
		Phase:           phase,
		ActivityPurpose: string(purposes[phase]),
		LockPurpose:     string(purposes[phase]),
		CatalogID:       "swarm-manager-phased-plan-" + skillPhase,
		SkillID:         "swarm-manager-phased-plan-" + skillPhase,
		ProfileKey:      profile,
		WritesRepo:      writesRepo,
		OutputArtifacts: artifacts,
		OutputContract: PhaseOutputContract{
			RequiresStructuredResult: true,
			RequiredArtifacts:        requiredArtifacts(artifacts),
			RequiresProgress:         phase == "classify_progress",
			RequiresVerdict:          phase == "review",
			RequiresHandoff:          phase == "execute_next",
		},
		RequiresCriteria: phase == "review",
	}
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
	for mode, def := range registry {
		if mode == ModeItemLevel {
			continue
		}
		for phase, phaseDef := range def.PhaseGraph.Phases {
			entry, ok := resolve(string(mode), string(phase))
			if !ok {
				return fmt.Errorf("prompt catalog missing entry for mode %q phase %q", mode, phase)
			}
			if strings.TrimSpace(entry.CatalogID) != phaseDef.CatalogID {
				return fmt.Errorf("prompt catalog ID mismatch for mode %q phase %q: registry=%q catalog=%q", mode, phase, phaseDef.CatalogID, entry.CatalogID)
			}
			if strings.TrimSpace(entry.SkillID) != phaseDef.SkillID {
				return fmt.Errorf("prompt catalog skill mismatch for mode %q phase %q: registry=%q catalog=%q", mode, phase, phaseDef.SkillID, entry.SkillID)
			}
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
		if def.Scope.Kind == "" {
			return fmt.Errorf("mode %q scope kind is required", mode)
		}
		if def.RunStrategy.Kind == "" {
			return fmt.Errorf("mode %q run strategy is required", mode)
		}
		if strings.TrimSpace(def.Metrics.EventSource) == "" {
			return fmt.Errorf("mode %q metrics event source is required", mode)
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
	for phase, phaseDef := range def.PhaseGraph.Phases {
		if phaseDef.Phase != phase {
			return fmt.Errorf("mode %q phase map key %q contains phase %q", def.Mode, phase, phaseDef.Phase)
		}
		if strings.TrimSpace(phaseDef.CatalogID) == "" || strings.TrimSpace(phaseDef.SkillID) == "" {
			return fmt.Errorf("mode %q phase %q prompt catalog ID and skill ID are required", def.Mode, phase)
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
		if err := validatePhaseOutputContract(phaseDef); err != nil {
			return fmt.Errorf("mode %q phase %q: %w", def.Mode, phase, err)
		}
	}
	return nil
}

func validatePhaseArtifactPolicy(def Definition, phaseDef PhaseDefinition) error {
	for _, artifact := range phaseDef.OutputArtifacts {
		if _, err := cleanModeRelativePath(def, artifact.Path); err != nil {
			return fmt.Errorf("mode %q phase %q artifact %q: %w", def.Mode, phaseDef.Phase, artifact.Path, err)
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
