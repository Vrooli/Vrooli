package operatingmode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const simulationInitiativeName = "simulation-sandbox"

type SimulationRequest struct {
	Mode string `json:"mode,omitempty"`
}

type SimulationResponse struct {
	Mode       string             `json:"mode"`
	Label      string             `json:"label"`
	Initiative InitiativeSnapshot `json:"initiative"`
	Trace      []SimulationStep   `json:"trace"`
}

type SimulationStep struct {
	Index      int                   `json:"index"`
	Phase      string                `json:"phase"`
	PhaseKind  string                `json:"phase_kind"`
	Inputs     SimulationInputs      `json:"inputs"`
	Output     PhaseResult           `json:"output"`
	Round      RoundEnvelope         `json:"round"`
	Transition *SimulationTransition `json:"transition,omitempty"`
	Terminal   bool                  `json:"terminal,omitempty"`
}

type SimulationInputs struct {
	Initiative         InitiativeSnapshot `json:"initiative"`
	Items              []RoundItem        `json:"items"`
	Artifacts          []ArtifactSnapshot `json:"artifacts"`
	PriorRounds        []RoundEnvelope    `json:"prior_rounds"`
	AcceptanceCriteria []string           `json:"acceptance_criteria"`
}

type SimulationTransition struct {
	From             string `json:"from"`
	To               string `json:"to,omitempty"`
	ConditionKind    string `json:"condition_kind"`
	Label            string `json:"label"`
	PayloadKey       string `json:"payload_key,omitempty"`
	ProgressDecision string `json:"progress_decision,omitempty"`
}

func (s *Service) SimulateMode(_ context.Context, mode Mode) (SimulationResponse, error) {
	def, err := DefinitionFor(NormalizeMode(string(mode)))
	if err != nil {
		return SimulationResponse{}, err
	}
	if def.Mode == ModeItemLevel || def.PhaseGraph.StartPhase == "" {
		return SimulationResponse{}, fmt.Errorf("mode %q has no operating-mode phase graph to simulate", def.Mode)
	}
	init := simulationInitiative(def)
	items := s.collectItems(init.Items)
	rounds := make([]RoundEnvelope, 0, len(def.PhaseGraph.Phases))
	trace := make([]SimulationStep, 0, len(def.PhaseGraph.Phases))

	phase := def.PhaseGraph.StartPhase
	for i := 0; i < len(def.PhaseGraph.Phases)+1; i++ {
		phaseDef, err := def.PhaseDefinition(phase)
		if err != nil {
			return SimulationResponse{}, err
		}
		inputs := SimulationInputs{
			Initiative:         init,
			Items:              append([]RoundItem(nil), items...),
			Artifacts:          simulationArtifacts(def, rounds),
			PriorRounds:        cloneRounds(rounds),
			AcceptanceCriteria: append([]string(nil), init.AcceptanceCriteria...),
		}
		result := cannedSimulationResult(def, phaseDef)
		output, err := encodePhaseResultEnvelope(result)
		if err != nil {
			return SimulationResponse{}, err
		}
		round := RoundEnvelope{
			Round:           len(rounds) + 1,
			Mode:            string(def.Mode),
			ScopeKind:       string(def.Scope.Kind),
			ScopeID:         init.Name,
			InitiativeName:  init.Name,
			Phase:           string(phaseDef.Phase),
			RunStrategy:     string(def.RunStrategy.Kind),
			AgentProfileKey: phaseDef.ProfileKey,
			GeneratedAt:     s.clock().UTC().Format(timeFormatRFC3339),
			RunID:           fmt.Sprintf("simulation-%03d", len(rounds)+1),
			Status:          RoundStatusCompleted,
			Items:           append([]RoundItem(nil), items...),
			Payload: map[string]any{
				"simulation": true,
				"skill_id":   phaseDef.SkillID,
				"catalog_id": phaseDef.CatalogID,
			},
		}
		if err := s.applyPhaseResultInMemory(&round, output); err != nil {
			return SimulationResponse{}, fmt.Errorf("simulate %s.%s: %w", def.Mode, phaseDef.Phase, err)
		}
		transition := simulationTransitionForCompletedRound(def, round)
		step := SimulationStep{
			Index:      len(trace),
			Phase:      string(phaseDef.Phase),
			PhaseKind:  string(phaseDef.Kind),
			Inputs:     inputs,
			Output:     result,
			Round:      round,
			Transition: transition,
			Terminal:   transition == nil || strings.TrimSpace(transition.To) == "",
		}
		trace = append(trace, step)
		rounds = append(rounds, round)
		if step.Terminal {
			return SimulationResponse{
				Mode:       string(def.Mode),
				Label:      def.Label,
				Initiative: init,
				Trace:      trace,
			}, nil
		}
		phase = Phase(transition.To)
	}
	return SimulationResponse{}, fmt.Errorf("simulate %s: phase graph did not terminate", def.Mode)
}

func simulationInitiative(def Definition) InitiativeSnapshot {
	return InitiativeSnapshot{
		Name:               simulationInitiativeName,
		Title:              def.Label + " Simulation",
		Description:        "Ephemeral sandbox initiative used to preview operating-mode flow.",
		Mode:               string(def.Mode),
		Items:              []string{"execute/simulation-alpha", "fix/simulation-beta"},
		AcceptanceCriteria: []string{"The simulated mode reaches review with a deterministic result."},
	}
}

func cannedSimulationResult(def Definition, phaseDef PhaseDefinition) PhaseResult {
	result := PhaseResult{
		Handoff: &Handoff{
			Summary:         fmt.Sprintf("Simulated %s phase completed.", phaseDef.Phase),
			CompletedPhases: []string{string(phaseDef.Phase)},
			NextStep:        "continue",
		},
	}
	for _, artifact := range phaseDef.OutputContract.RequiredArtifacts {
		result.Artifacts = append(result.Artifacts, ArtifactResult{
			Path:        artifact.Path,
			Content:     fmt.Sprintf("# Simulated %s\n\nGenerated by the %s simulation.", phaseDef.Phase, def.Label),
			ContentType: artifact.ContentType,
		})
	}
	if phaseDef.OutputContract.RequiresProgress {
		result.Progress = &ProgressState{
			Decision:        ProgressComplete,
			CompletedPhases: []string{string(phaseDef.Phase)},
			CurrentPhase:    string(phaseDef.Phase),
			Rationale:       "Simulation fixture advances to review.",
		}
	}
	if phaseDef.OutputContract.RequiresVerdict {
		result.Verdict = "accepted"
	}
	if phaseDef.OutputContract.RequiresBacklogSync {
		result.BacklogSync = &BacklogSyncPlan{
			UpdatedItems: []string{"execute/simulation-alpha"},
			Rationale:    "Simulation fixture proposes a no-op alignment update.",
		}
	}
	return result
}

func encodePhaseResultEnvelope(result PhaseResult) (string, error) {
	body, err := json.Marshal(map[string]PhaseResult{resultEnvelopeKey: result})
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func simulationTransitionForCompletedRound(def Definition, round RoundEnvelope) *SimulationTransition {
	from := Phase(round.Phase)
	payload := RoundPayload(round.Payload)
	for _, rule := range def.PhaseGraph.TransitionRules[from] {
		if !rule.When.Matches(payload) {
			continue
		}
		transition := transitionFromCondition(from, rule.When)
		if len(rule.Next) > 0 {
			transition.To = string(rule.Next[0])
		}
		return &transition
	}
	next := def.PhaseGraph.Transitions[from]
	if len(next) == 0 {
		return nil
	}
	return &SimulationTransition{
		From:          string(from),
		To:            string(next[0]),
		ConditionKind: string(TransitionConditionAlways),
		Label:         transitionLabel(TransitionCondition{Kind: TransitionConditionAlways}),
	}
}

func transitionFromCondition(from Phase, condition TransitionCondition) SimulationTransition {
	return SimulationTransition{
		From:             string(from),
		ConditionKind:    string(condition.Kind),
		Label:            transitionLabel(condition),
		PayloadKey:       condition.PayloadKey,
		ProgressDecision: string(condition.ProgressDecision),
	}
}

func simulationArtifacts(def Definition, rounds []RoundEnvelope) []ArtifactSnapshot {
	seen := map[string]ArtifactSnapshot{}
	for _, phase := range def.PhaseGraph.Phases {
		for _, artifact := range phase.OutputArtifacts {
			if strings.TrimSpace(artifact.Path) == "" {
				continue
			}
			seen[artifact.Path] = ArtifactSnapshot{
				Path:        artifact.Path,
				ContentType: artifact.ContentType,
				Required:    artifact.Required,
			}
		}
	}
	for _, round := range rounds {
		for _, update := range round.ArtifactUpdates {
			snapshot := seen[update.Path]
			snapshot.Path = update.Path
			snapshot.ContentType = defaultString(update.ContentType, snapshot.ContentType)
			snapshot.Required = update.Required || snapshot.Required
			snapshot.UpdatedAt = update.UpdatedAt
			snapshot.Content = "Simulated artifact generated by " + round.Phase + "."
			seen[update.Path] = snapshot
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sortStrings(paths)
	out := make([]ArtifactSnapshot, 0, len(paths))
	for _, path := range paths {
		out = append(out, seen[path])
	}
	return out
}

func cloneRounds(rounds []RoundEnvelope) []RoundEnvelope {
	out := make([]RoundEnvelope, 0, len(rounds))
	for _, round := range rounds {
		out = append(out, cloneRoundForPhaseResult(round))
	}
	return out
}
