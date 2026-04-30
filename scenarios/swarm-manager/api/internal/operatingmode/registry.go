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
	RequiresCriteria bool
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
		Phase:            phase,
		ActivityPurpose:  string(purposes[phase]),
		LockPurpose:      string(purposes[phase]),
		CatalogID:        "swarm-manager-holistic-loop-" + string(phase),
		SkillID:          "swarm-manager-holistic-loop-" + string(phase),
		ProfileKey:       profile,
		WritesRepo:       writesRepo,
		OutputArtifacts:  artifacts,
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
		Phase:            phase,
		ActivityPurpose:  string(purposes[phase]),
		LockPurpose:      string(purposes[phase]),
		CatalogID:        "swarm-manager-phased-plan-" + skillPhase,
		SkillID:          "swarm-manager-phased-plan-" + skillPhase,
		ProfileKey:       profile,
		WritesRepo:       writesRepo,
		OutputArtifacts:  artifacts,
		RequiresCriteria: phase == "review",
	}
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

// RequiredProfileKeys returns every AgentManager profile key referenced by the
// operating-mode registry. Profile JSON remains the source of profile defaults;
// the registry only declares which scenario-owned keys must exist before the
// API serves traffic.
func RequiredProfileKeys() ([]string, error) {
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
