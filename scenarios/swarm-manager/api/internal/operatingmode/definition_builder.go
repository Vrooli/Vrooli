package operatingmode

import "strings"

type initiativeModeSpec struct {
	Mode                Mode
	Label               string
	Description         string
	RunStrategy         RunStrategyKind
	ArtifactRoot        string
	PromptCatalogPrefix string
	DefaultProfileKey   string
	StartPhase          Phase
	Terminal            []Phase
	Transitions         map[Phase][]Phase
	TransitionRules     map[Phase][]TransitionRule
	Phases              []initiativePhaseSpec
}

type initiativePhaseSpec struct {
	Phase            Phase
	Purpose          string
	LockPurpose      string
	PromptSuffix     string
	PromptTitle      string
	PromptPurpose    string
	PromptTrigger    string
	ProfileKey       string
	WritesRepo       bool
	OutputArtifacts  []ArtifactDefinition
	ResultBindings   []ResultBinding
	Metrics          PhaseMetricsSpec
	RequiresProgress bool
	RequiresVerdict  bool
	RequiresHandoff  bool
	RequiresCriteria bool
}

type PhaseMetricsSpec struct {
	CountsReplanSample     bool
	CountsAcceptanceSample bool
}

func buildInitiativeMode(spec initiativeModeSpec) Definition {
	phases := make(map[Phase]PhaseDefinition, len(spec.Phases))
	phaseProfiles := make(map[Phase]string, len(spec.Phases))
	metrics := MetricsPolicy{EventSource: string(spec.Mode)}
	for _, phaseSpec := range spec.Phases {
		phase := buildInitiativePhase(spec.PromptCatalogPrefix, phaseSpec)
		phases[phase.Phase] = phase
		phaseProfiles[phase.Phase] = phase.ProfileKey
		if phaseSpec.Metrics.CountsReplanSample {
			metrics.ReplanSamplePhases = append(metrics.ReplanSamplePhases, phase.Phase)
		}
		if phaseSpec.Metrics.CountsAcceptanceSample {
			metrics.AcceptanceSamplePhases = append(metrics.AcceptanceSamplePhases, phase.Phase)
		}
	}
	if len(metrics.AcceptanceSamplePhases) > 0 {
		metrics.AcceptedVerdicts = defaultAcceptedVerdicts()
	}

	return Definition{
		Mode:        spec.Mode,
		Label:       spec.Label,
		Description: spec.Description,
		Scope:       ScopePolicy{Kind: ScopeInitiative},
		PhaseGraph: PhaseGraph{
			StartPhase:      spec.StartPhase,
			Terminal:        append([]Phase(nil), spec.Terminal...),
			Transitions:     clonePhaseTransitionMap(spec.Transitions),
			TransitionRules: clonePhaseTransitionRuleMap(spec.TransitionRules),
			Phases:          phases,
		},
		RunStrategy: RunStrategyPolicy{Kind: spec.RunStrategy},
		Artifact: ArtifactPolicy{
			Root:      spec.ArtifactRoot,
			RoundRoot: strings.TrimRight(spec.ArtifactRoot, "/") + "/rounds",
		},
		Prompt: PromptPolicy{CatalogPrefix: spec.PromptCatalogPrefix},
		Profile: ProfilePolicy{
			DefaultProfileKey: spec.DefaultProfileKey,
			PhaseProfiles:     phaseProfiles,
		},
		BacklogSync: BacklogSyncPolicy{
			Capabilities:       defaultInitiativeBacklogSyncCapabilities(),
			RequiresRunID:      true,
			RequiresMembership: true,
			EventSource:        string(spec.Mode),
		},
		Metrics: metrics,
		Lock:    LockPolicy{InitiativeExclusive: true},
		UI:      UIPolicy{WorkspaceTabID: "operating-mode"},
	}
}

func buildInitiativePhase(promptCatalogPrefix string, spec initiativePhaseSpec) PhaseDefinition {
	promptSuffix := strings.TrimSpace(spec.PromptSuffix)
	if promptSuffix == "" {
		promptSuffix = string(spec.Phase)
	}

	lockPurpose := spec.LockPurpose
	if lockPurpose == "" {
		lockPurpose = spec.Purpose
	}

	outputArtifacts := mergeOutputArtifacts(spec.OutputArtifacts, spec.ResultBindings)

	return PhaseDefinition{
		Phase:           spec.Phase,
		ActivityPurpose: spec.Purpose,
		LockPurpose:     lockPurpose,
		CatalogID:       promptCatalogPrefix + "-" + promptSuffix,
		SkillID:         promptCatalogPrefix + "-" + promptSuffix,
		PromptCatalog: PromptCatalogMetadata{
			Title:   defaultString(spec.PromptTitle, humanizeToken(string(spec.Phase))),
			Trigger: defaultString(spec.PromptTrigger, "Operator starts "+promptCatalogPrefix+" "+promptSuffix+" phase"),
			Purpose: defaultString(spec.PromptPurpose, "Run the "+string(spec.Phase)+" phase."),
		},
		ProfileKey:      spec.ProfileKey,
		WritesRepo:      spec.WritesRepo,
		OutputArtifacts: outputArtifacts,
		ResultBindings:  cloneResultBindings(spec.ResultBindings),
		OutputContract: PhaseOutputContract{
			RequiresStructuredResult: true,
			RequiredArtifacts:        requiredArtifacts(outputArtifacts),
			RequiresProgress:         spec.RequiresProgress,
			RequiresVerdict:          spec.RequiresVerdict,
			RequiresHandoff:          spec.RequiresHandoff,
		},
		RequiresCriteria: spec.RequiresCriteria,
	}
}

func progressResultArtifact(path string) ResultBinding {
	return ResultBinding{
		Kind:     ResultBindingProgressArtifact,
		Artifact: requiredOutputArtifact(path, "application/json"),
	}
}

func requiredOutputArtifact(path string, contentType string) ArtifactDefinition {
	return ArtifactDefinition{
		Path:        path,
		ContentType: contentType,
		Required:    true,
	}
}

func defaultInitiativeBacklogSyncCapabilities() []BacklogSyncCapability {
	return []BacklogSyncCapability{
		BacklogSyncReadOnly,
		BacklogSyncProposeMutations,
		BacklogSyncMarkComplete,
		BacklogSyncCreateFollowups,
		BacklogSyncUpdateScope,
	}
}

func defaultAcceptedVerdicts() []string {
	return []string{"accept", "accepted"}
}

func clonePhaseTransitionMap(in map[Phase][]Phase) map[Phase][]Phase {
	out := make(map[Phase][]Phase, len(in))
	for phase, next := range in {
		out[phase] = append([]Phase(nil), next...)
	}
	return out
}

func mergeOutputArtifacts(artifacts []ArtifactDefinition, bindings []ResultBinding) []ArtifactDefinition {
	out := append([]ArtifactDefinition(nil), artifacts...)
	seen := map[string]int{}
	for i, artifact := range out {
		seen[strings.TrimSpace(artifact.Path)] = i
	}
	for _, binding := range bindings {
		path := strings.TrimSpace(binding.Artifact.Path)
		if path == "" {
			out = append(out, binding.Artifact)
			continue
		}
		if existing, ok := seen[path]; ok {
			out[existing] = mergeArtifactDefinition(out[existing], binding.Artifact)
			continue
		}
		seen[path] = len(out)
		out = append(out, binding.Artifact)
	}
	return out
}

func mergeArtifactDefinition(existing ArtifactDefinition, incoming ArtifactDefinition) ArtifactDefinition {
	if existing.ContentType == "" {
		existing.ContentType = incoming.ContentType
	}
	existing.Required = existing.Required || incoming.Required
	return existing
}

func cloneResultBindings(in []ResultBinding) []ResultBinding {
	return append([]ResultBinding(nil), in...)
}

func clonePhaseTransitionRuleMap(in map[Phase][]TransitionRule) map[Phase][]TransitionRule {
	out := make(map[Phase][]TransitionRule, len(in))
	for phase, rules := range in {
		out[phase] = cloneTransitionRules(rules)
	}
	return out
}

func cloneTransitionRules(in []TransitionRule) []TransitionRule {
	out := make([]TransitionRule, len(in))
	for i, rule := range in {
		rule.Next = append([]Phase(nil), rule.Next...)
		out[i] = rule
	}
	return out
}

func humanizeToken(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
