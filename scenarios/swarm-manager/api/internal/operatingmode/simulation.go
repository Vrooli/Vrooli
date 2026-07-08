package operatingmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const simulationInitiativeName = "simulation-sandbox"

// maxSimulationSteps bounds preset traces that intentionally revisit phases
// (replan / continue loops). Presets are scripted to terminate well within
// this cap; it exists only as a runaway guard so a mis-scripted preset fails
// loudly instead of looping forever.
const maxSimulationSteps = 24

type SimulationRequest struct {
	Mode   string `json:"mode,omitempty"`
	Preset string `json:"preset,omitempty"`
}

type SimulationResponse struct {
	Mode         string             `json:"mode"`
	Label        string             `json:"label"`
	Presets      []SimulationPreset `json:"presets"`
	ActivePreset string             `json:"active_preset"`
	Initiative   InitiativeSnapshot `json:"initiative"`
	Trace        []SimulationStep   `json:"trace"`
}

// SimulationPreset is operator-facing metadata describing one deterministic
// scenario. Presets seed different phase outputs to exercise real transition
// branches; they never bypass the registered phase graph or its guards. The
// UI renders id/label/description/branch to let operators pick a work shape.
type SimulationPreset struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Branch      string `json:"branch"`
	Scenario    string `json:"scenario"`
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
	// SkillID and ProfileKey name the agent skill and profile this phase would
	// spawn, so the UI can label the Instructions tab without a render call.
	SkillID    string `json:"skill_id,omitempty"`
	ProfileKey string `json:"profile_key,omitempty"`
	// PromptVariables is the resolved template substitution map for this step,
	// computed purely (no ReadSkill). The UI shows what was substituted — and
	// falls back to it for the Instructions tab — even when the render endpoint
	// is unavailable.
	PromptVariables map[string]string `json:"prompt_variables,omitempty"`
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

func (s *Service) SimulateMode(_ context.Context, mode Mode, presetID string) (SimulationResponse, error) {
	def, err := DefinitionFor(NormalizeMode(string(mode)))
	if err != nil {
		return SimulationResponse{}, err
	}
	if def.Mode == ModeItemLevel || def.PhaseGraph.StartPhase == "" {
		return SimulationResponse{}, fmt.Errorf("mode %q has no operating-mode phase graph to simulate", def.Mode)
	}
	presets := simulationPresetsForMode(def.Mode)
	preset := resolveSimulationPreset(presets, presetID)
	init := simulationInitiative(def, preset)
	items := append([]RoundItem(nil), preset.items...)
	rounds := make([]RoundEnvelope, 0, len(def.PhaseGraph.Phases))
	trace := make([]SimulationStep, 0, len(def.PhaseGraph.Phases))
	visits := map[Phase]int{}

	phase := def.PhaseGraph.StartPhase
	for i := 0; i < maxSimulationSteps; i++ {
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
		visit := visits[phase]
		visits[phase]++
		result := preset.result(def, phaseDef, cannedSimulationResult(def, phaseDef, preset), visit)
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
		stepCtx := simulationStepContext(def, phaseDef, inputs)
		step := SimulationStep{
			Index:           len(trace),
			Phase:           string(phaseDef.Phase),
			PhaseKind:       string(phaseDef.Kind),
			Inputs:          inputs,
			Output:          result,
			Round:           round,
			Transition:      transition,
			Terminal:        transition == nil || strings.TrimSpace(transition.To) == "",
			SkillID:         phaseDef.SkillID,
			ProfileKey:      phaseDef.ProfileKey,
			PromptVariables: promptVariables(stepCtx, round, ""),
		}
		trace = append(trace, step)
		rounds = append(rounds, round)
		if step.Terminal {
			return SimulationResponse{
				Mode:         string(def.Mode),
				Label:        def.Label,
				Presets:      presetMetadata(presets),
				ActivePreset: preset.meta.ID,
				Initiative:   init,
				Trace:        trace,
			}, nil
		}
		phase = Phase(transition.To)
	}
	return SimulationResponse{}, fmt.Errorf("simulate %s preset %q: phase graph did not terminate within %d steps", def.Mode, preset.meta.ID, maxSimulationSteps)
}

// simulationStepContext rebuilds the phaseContext that fed a simulation step
// from its recorded inputs. It is the bridge that lets the render-preview
// endpoint substitute the exact same fixture data the trace already shows, and
// it is pure (no store reads), matching the isolated nature of SimulateMode.
func simulationStepContext(def Definition, phaseDef PhaseDefinition, inputs SimulationInputs) phaseContext {
	return phaseContext{
		init:      inputs.Initiative,
		def:       def,
		phaseDef:  phaseDef,
		items:     inputs.Items,
		artifacts: inputs.Artifacts,
		rounds:    inputs.PriorRounds,
	}
}

// RenderPromptResponse is the render-preview payload: the literal prompt an
// agent would receive for one simulation step, plus the inputs that produced
// it. When the prompt-manager seam is unavailable the response is marked
// Degraded and Prompt is empty, so the UI can fall back to Variables.
type RenderPromptResponse struct {
	Mode           string            `json:"mode"`
	Preset         string            `json:"preset"`
	StepIndex      int               `json:"step_index"`
	Phase          string            `json:"phase"`
	SkillID        string            `json:"skill_id"`
	ProfileKey     string            `json:"profile_key"`
	Variables      map[string]string `json:"variables"`
	Prompt         string            `json:"prompt,omitempty"`
	Degraded       bool              `json:"degraded,omitempty"`
	DegradedReason string            `json:"degraded_reason,omitempty"`
}

// RenderSimulationPrompt re-derives the deterministic fixture context for one
// step of a preset trace and renders the literal agent prompt for it via the
// shared renderPhasePrompt seam — no spawning, locking, or persistence. When
// the prompt client/catalog is unavailable it returns a typed degraded
// response (Degraded=true, Variables populated) rather than an error, so the UI
// can still show what would be substituted.
func (s *Service) RenderSimulationPrompt(ctx context.Context, mode Mode, presetID string, stepIndex int) (RenderPromptResponse, error) {
	sim, err := s.SimulateMode(ctx, mode, presetID)
	if err != nil {
		return RenderPromptResponse{}, err
	}
	if stepIndex < 0 || stepIndex >= len(sim.Trace) {
		return RenderPromptResponse{}, fmt.Errorf("simulate render: step_index %d out of range [0,%d) for mode %q preset %q", stepIndex, len(sim.Trace), sim.Mode, sim.ActivePreset)
	}
	step := sim.Trace[stepIndex]
	phaseDef, err := DefinitionFor(NormalizeMode(sim.Mode))
	if err != nil {
		return RenderPromptResponse{}, err
	}
	pd, err := phaseDef.PhaseDefinition(Phase(step.Phase))
	if err != nil {
		return RenderPromptResponse{}, err
	}
	resp := RenderPromptResponse{
		Mode:       sim.Mode,
		Preset:     sim.ActivePreset,
		StepIndex:  stepIndex,
		Phase:      step.Phase,
		SkillID:    step.SkillID,
		ProfileKey: step.ProfileKey,
		Variables:  step.PromptVariables,
	}
	stepCtx := simulationStepContext(phaseDef, pd, step.Inputs)
	rendered, err := s.renderPhasePrompt(ctx, stepCtx, step.Round, "")
	if err != nil {
		if errors.Is(err, ErrPromptRenderUnavailable) {
			resp.Degraded = true
			resp.DegradedReason = err.Error()
			return resp, nil
		}
		return RenderPromptResponse{}, err
	}
	resp.Prompt = rendered.Prompt
	resp.Variables = rendered.Variables
	return resp, nil
}

// simulationPreset couples operator-facing metadata with a deterministic
// output script. The script chooses phase payloads only; the real transition
// guards in simulationTransitionForCompletedRound still decide routing, so a
// preset can shape a branch but never fabricate one the registry disallows.
type simulationPreset struct {
	meta                  SimulationPreset
	initiativeTitle       string
	initiativeDescription string
	items                 []RoundItem
	criteria              []string
	// result maps the canned base output for a phase into the scripted output.
	// visit is the 0-based count of prior visits to this phase in the trace,
	// which lets loop-back presets terminate on the second pass.
	result func(def Definition, phaseDef PhaseDefinition, base PhaseResult, visit int) PhaseResult
}

func presetMetadata(presets []simulationPreset) []SimulationPreset {
	out := make([]SimulationPreset, 0, len(presets))
	for _, preset := range presets {
		out = append(out, preset.meta)
	}
	return out
}

// resolveSimulationPreset returns the requested preset by id, or the first
// (happy-path) preset when the id is empty or unknown. Presets always has at
// least one entry for phase modes.
func resolveSimulationPreset(presets []simulationPreset, id string) simulationPreset {
	id = strings.TrimSpace(id)
	if id != "" {
		for _, preset := range presets {
			if preset.meta.ID == id {
				return preset
			}
		}
	}
	return presets[0]
}

// passthroughResult is the default scripting hook: use the canned base output
// unchanged. Happy-path presets rely on it.
func passthroughResult(_ Definition, _ PhaseDefinition, base PhaseResult, _ int) PhaseResult {
	return base
}

func simulationPresetsForMode(mode Mode) []simulationPreset {
	switch mode {
	case ModeHolisticLoop:
		return holisticLoopPresets()
	case ModePhasedPlanDrain:
		return phasedPlanDrainPresets()
	default:
		return []simulationPreset{genericHappyPathPreset()}
	}
}

// genericHappyPathPreset is a safe default for any future phase mode that has
// not authored bespoke presets yet: it walks the graph on canned outputs.
func genericHappyPathPreset() simulationPreset {
	return simulationPreset{
		meta: SimulationPreset{
			ID:          "happy-path",
			Label:       "Happy path",
			Description: "Every phase completes cleanly and the mode reaches its terminal phase.",
			Branch:      "Straight-through completion with no loop-backs.",
			Scenario:    "A well-scoped initiative where the first pass through every phase succeeds.",
		},
		initiativeTitle:       "",
		initiativeDescription: "",
		items:                 defaultSimulationItems(),
		criteria:              defaultSimulationCriteria(),
		result:                passthroughResult,
	}
}

func holisticLoopPresets() []simulationPreset {
	items := []RoundItem{
		{Ref: "execute/audio-session-teardown", Title: "Tear down idle audio session", Status: "in_progress", Priority: 1, Effort: "M"},
		{Ref: "fix/mic-wedge-regression", Title: "Fix mic wedge after backgrounding", Status: "todo", Priority: 2, Effort: "S"},
	}
	criteria := []string{
		"An idle tab never holds the system audio session.",
		"The microphone recovers after backgrounding without an app restart.",
	}
	return []simulationPreset{
		{
			meta: SimulationPreset{
				ID:          "happy-path",
				Label:       "Clean pass",
				Description: "Investigate → plan → execute → review → reconcile with no replanning.",
				Branch:      "execute → review (replan not needed)",
				Scenario:    "Investigation surfaces the full picture up front, the plan holds, execution needs no rework, and review accepts the result before the backlog is reconciled.",
			},
			initiativeTitle:       "Unify the audio-session lifecycle",
			initiativeDescription: "Coupled audio fixes that can only be validated together across the tab lifecycle.",
			items:                 items,
			criteria:              criteria,
			result:                passthroughResult,
		},
		{
			meta: SimulationPreset{
				ID:          "replan-after-execute",
				Label:       "Execute triggers replan",
				Description: "Execution discovers the plan was wrong and loops back to investigate before finishing.",
				Branch:      "execute → investigate (replan_needed), then a clean second pass",
				Scenario:    "The first execution attempt reveals the audio-session bug spans a subsystem the plan missed, so the run reports replan_needed and the loop returns to investigate. The second pass completes and review accepts.",
			},
			initiativeTitle:       "Unify the audio-session lifecycle",
			initiativeDescription: "Exploratory audio work where the first plan is expected to miss something material.",
			items:                 items,
			criteria:              criteria,
			result: func(_ Definition, phaseDef PhaseDefinition, base PhaseResult, visit int) PhaseResult {
				if phaseDef.Phase == "execute" && visit == 0 {
					base.ReplanNeeded = true
					if base.Handoff != nil {
						base.Handoff.Summary = "Execution surfaced a cross-subsystem gap; requesting a replan before continuing."
						base.Handoff.NextStep = "investigate"
					}
				}
				return base
			},
		},
		{
			meta: SimulationPreset{
				ID:          "review-not-accepted",
				Label:       "Review requests changes",
				Description: "Review returns a non-accepting verdict; reconcile still proposes follow-up backlog work.",
				Branch:      "review → reconcile (verdict recorded, routing unchanged)",
				Scenario:    "The acceptance review finds the result short of one criterion and records a changes_requested verdict. The verdict is informational for routing — reconcile still runs and proposes follow-up items so the gap is tracked.",
			},
			initiativeTitle:       "Unify the audio-session lifecycle",
			initiativeDescription: "A pass where acceptance review is not satisfied on the first cycle.",
			items:                 items,
			criteria:              criteria,
			result: func(_ Definition, phaseDef PhaseDefinition, base PhaseResult, _ int) PhaseResult {
				if phaseDef.Phase == "review" {
					base.Verdict = "changes_requested"
				}
				return base
			},
		},
	}
}

func phasedPlanDrainPresets() []simulationPreset {
	items := []RoundItem{
		{Ref: "execute/durable-run-primitive", Title: "Add durable_run primitive", Status: "in_progress", Priority: 1, Effort: "M"},
		{Ref: "execute/migrate-test-genie-execute", Title: "Migrate test-genie execute to durable handles", Status: "todo", Priority: 2, Effort: "M"},
	}
	criteria := []string{
		"Every execute command returns a durable run handle.",
		"The legacy in-process primitive is removed.",
	}
	title := "Migrate CLI commands to durable run handles"
	return []simulationPreset{
		{
			meta: SimulationPreset{
				ID:          "happy-path",
				Label:       "Drains in one slice",
				Description: "Prepare → execute → classify (complete) → review → reconcile in a single pass.",
				Branch:      "classify_progress → review (complete)",
				Scenario:    "The prepared plan is small enough that one execution slice finishes it. Classify reports complete, review accepts, and reconcile aligns the backlog.",
			},
			initiativeTitle:       title,
			initiativeDescription: "A stable, well-decomposed plan that a single slice can drain.",
			items:                 items,
			criteria:              criteria,
			result:                passthroughResult,
		},
		{
			meta: SimulationPreset{
				ID:          "continue-next-slice",
				Label:       "Continue to next slice",
				Description: "Classify reports the current slice is done but more remain, so execution continues.",
				Branch:      "classify_progress → execute_next (continue), then complete",
				Scenario:    "The first slice lands the durable_run primitive and classify reports continue — there is more plan to drain. Execution runs the next slice, classify then reports complete, and the cycle finishes at review.",
			},
			initiativeTitle:       title,
			initiativeDescription: "A multi-slice plan drained across sequential handoffs.",
			items:                 items,
			criteria:              criteria,
			result: func(_ Definition, phaseDef PhaseDefinition, base PhaseResult, visit int) PhaseResult {
				if phaseDef.Phase == "classify_progress" && visit == 0 && base.Progress != nil {
					base.Progress.Decision = ProgressContinue
					base.Progress.Rationale = "The first slice is complete and the next contiguous slice is ready to drain."
				}
				return base
			},
		},
		{
			meta: SimulationPreset{
				ID:          "replan-plan",
				Label:       "Progress forces a replan",
				Description: "Classify decides the plan is wrong and routes back to prepare a new plan.",
				Branch:      "classify_progress → prepare_plan (replan), then complete",
				Scenario:    "Executing the first slice reveals the phased plan mis-ordered a dependency. Classify reports replan and routing returns to prepare_plan. The revised plan drains cleanly and the cycle completes.",
			},
			initiativeTitle:       title,
			initiativeDescription: "A plan that needs revision partway through the drain.",
			items:                 items,
			criteria:              criteria,
			result: func(_ Definition, phaseDef PhaseDefinition, base PhaseResult, visit int) PhaseResult {
				if phaseDef.Phase == "classify_progress" && visit == 0 && base.Progress != nil {
					base.Progress.Decision = ProgressReplan
					base.Progress.Rationale = "Execution exposed a dependency the plan mis-ordered; the plan must be revised."
				}
				return base
			},
		},
		{
			meta: SimulationPreset{
				ID:          "blocked",
				Label:       "Work is blocked",
				Description: "Classify reports the drain cannot proceed; the cycle ends without review.",
				Branch:      "classify_progress → (blocked, terminal)",
				Scenario:    "A slice hits an external blocker — an unavailable dependency — that the agent cannot resolve. Classify reports blocked, which is a terminal decision: the cycle stops before review so an operator can intervene.",
			},
			initiativeTitle:       title,
			initiativeDescription: "A drain that stalls on an external blocker.",
			items:                 items,
			criteria:              criteria,
			result: func(_ Definition, phaseDef PhaseDefinition, base PhaseResult, visit int) PhaseResult {
				if phaseDef.Phase == "classify_progress" && visit == 0 && base.Progress != nil {
					base.Progress.Decision = ProgressBlocked
					base.Progress.Rationale = "An external dependency is unavailable; the drain cannot proceed without operator intervention."
				}
				return base
			},
		},
	}
}

func defaultSimulationItems() []RoundItem {
	return []RoundItem{
		{Ref: "execute/simulation-alpha", Title: "Primary scoped work item", Status: "in_progress", Priority: 1, Effort: "M"},
		{Ref: "fix/simulation-beta", Title: "Secondary follow-up item", Status: "todo", Priority: 2, Effort: "S"},
	}
}

func defaultSimulationCriteria() []string {
	return []string{"The simulated mode reaches its terminal phase with a deterministic result."}
}

func simulationInitiative(def Definition, preset simulationPreset) InitiativeSnapshot {
	title := preset.initiativeTitle
	if strings.TrimSpace(title) == "" {
		title = def.Label + " Simulation"
	}
	description := preset.initiativeDescription
	if strings.TrimSpace(description) == "" {
		description = "Ephemeral sandbox initiative used to preview operating-mode flow."
	}
	itemRefs := make([]string, 0, len(preset.items))
	for _, item := range preset.items {
		itemRefs = append(itemRefs, item.Ref)
	}
	return InitiativeSnapshot{
		Name:               simulationInitiativeName,
		Title:              title,
		Description:        description,
		Mode:               string(def.Mode),
		Items:              itemRefs,
		AcceptanceCriteria: append([]string(nil), preset.criteria...),
	}
}

func cannedSimulationResult(def Definition, phaseDef PhaseDefinition, preset simulationPreset) PhaseResult {
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
		result.BacklogSync = simulationBacklogSync(preset)
	}
	return result
}

// simulationBacklogSync builds a realistic reconcile proposal from the preset's
// seeded items: the primary item is marked completed and a scoped follow-up is
// proposed, so the reconcile step reads as a real backlog alignment rather than
// a no-op.
func simulationBacklogSync(preset simulationPreset) *BacklogSyncPlan {
	plan := &BacklogSyncPlan{
		Rationale: "Align the backlog with the work the drained cycle just completed.",
	}
	if len(preset.items) > 0 {
		plan.CompletedItems = []string{preset.items[0].Ref}
	}
	if len(preset.items) > 1 {
		plan.UpdatedItems = []string{preset.items[1].Ref}
	}
	plan.CreatedItems = []string{"fix/simulation-followup"}
	return plan
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
