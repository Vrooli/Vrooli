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

// SimulationTransition describes the guarded edge a simulated round took. The
// condition is rendered generically from the guard (kind + human label + the
// leaf field/value when applicable) so no closed branch vocabulary leaks into
// the wire shape.
type SimulationTransition struct {
	From          string `json:"from"`
	To            string `json:"to,omitempty"`
	ConditionKind string `json:"condition_kind"`
	Label         string `json:"label"`
	Field         string `json:"field,omitempty"`
	Value         string `json:"value,omitempty"`
}

func (s *Service) SimulateMode(ctx context.Context, mode Mode, presetID string) (SimulationResponse, error) {
	def, err := DefinitionFor(NormalizeMode(string(mode)))
	if err != nil {
		return SimulationResponse{}, err
	}
	return s.simulateDefinition(ctx, def, presetID)
}

// simulateDefinition is the transport-agnostic simulation core: it walks the
// given Definition's real generic guards against the resolved preset, with no
// dependency on the process registry. SimulateMode feeds it a registered mode;
// SimulateModeDraft feeds it a mode loaded fresh from disk, so an author can
// simulate a scaffolded mode before it is registered (no restart).
func (s *Service) simulateDefinition(ctx context.Context, def Definition, presetID string) (SimulationResponse, error) {
	if def.Mode == ModeItemLevel || def.PhaseGraph.StartPhase == "" {
		return SimulationResponse{}, fmt.Errorf("mode %q has no operating-mode phase graph to simulate", def.Mode)
	}
	presets := simulationPresets(def)
	preset := resolveSimulationPreset(presets, presetID)
	init := simulationInitiative(def, preset)
	items := preset.roundItems()
	rounds := make([]RoundEnvelope, 0, len(def.PhaseGraph.Phases))
	trace := make([]SimulationStep, 0, len(def.PhaseGraph.Phases))
	stepIdx := 0

	phase := def.PhaseGraph.StartPhase
	// subCur tracks the sub-mode phase while the walk is inside a delegated
	// phase (executed_by); empty when the phase executes its own contract.
	var subCur Phase
	for i := 0; i < maxSimulationSteps; i++ {
		phaseDef, err := def.PhaseDefinition(phase)
		if err != nil {
			return SimulationResponse{}, err
		}
		// Resolve the execution contract for this visit: the phase's own, or
		// the sub-mode's current sub-phase for a delegated phase.
		execDef, execPhaseDef := def, phaseDef
		if phaseDef.Delegated() {
			sub, err := delegationSubDefinition(phaseDef)
			if err != nil {
				return SimulationResponse{}, err
			}
			if subCur == "" {
				subCur = sub.PhaseGraph.StartPhase
			}
			execDef = sub
			execPhaseDef, err = sub.PhaseDefinition(subCur)
			if err != nil {
				return SimulationResponse{}, err
			}
		}
		inputs := SimulationInputs{
			Initiative:         init,
			Items:              append([]RoundItem(nil), items...),
			Artifacts:          simulationArtifacts(def, rounds),
			PriorRounds:        cloneRounds(rounds),
			AcceptanceCriteria: append([]string(nil), init.AcceptanceCriteria...),
		}
		// Consume the preset's next seeded step when it names this phase; the
		// step's output overrides the contract-derived canned scaffolding. An
		// unconsumed step whose phase does not match the walk is an authoring
		// error in the example-run, surfaced loudly.
		var seeded map[string]any
		if stepIdx < len(preset.steps) && Phase(preset.steps[stepIdx].Phase) == phase {
			seeded = preset.steps[stepIdx].Output
			stepIdx++
		} else if stepIdx < len(preset.steps) {
			return SimulationResponse{}, fmt.Errorf("simulate %s preset %q: step %d is phase %q but the walk reached %q", def.Mode, preset.meta.ID, stepIdx, preset.steps[stepIdx].Phase, phase)
		}
		result, resultMap, err := mergeSeededResult(cannedSimulationResult(execDef, execPhaseDef, items), seeded)
		if err != nil {
			return SimulationResponse{}, fmt.Errorf("simulate %s.%s: %w", def.Mode, phaseDef.Phase, err)
		}
		output, err := encodeEnvelopeMap(resultMap)
		if err != nil {
			return SimulationResponse{}, err
		}
		payload := map[string]any{
			"simulation": true,
			"skill_id":   execPhaseDef.SkillID,
			"catalog_id": execPhaseDef.CatalogID,
		}
		if phaseDef.Delegated() {
			payload[payloadDelegatedMode] = string(execDef.Mode)
			payload[payloadDelegatedPhase] = string(execPhaseDef.Phase)
		}
		round := RoundEnvelope{
			Round:           len(rounds) + 1,
			Mode:            string(def.Mode),
			ScopeKind:       string(def.Target.Kind),
			ScopeID:         init.Name,
			InitiativeName:  init.Name,
			Phase:           string(phaseDef.Phase),
			RunStrategy:     string(def.RunStrategy.Kind),
			AgentProfileKey: execPhaseDef.ProfileKey,
			GeneratedAt:     s.clock().UTC().Format(timeFormatRFC3339),
			RunID:           fmt.Sprintf("simulation-%03d", len(rounds)+1),
			Status:          RoundStatusCompleted,
			Items:           append([]RoundItem(nil), items...),
			Payload:         payload,
		}
		resolved, err := s.applyPhaseResultInMemory(ctx, def, &round, output)
		if err != nil {
			return SimulationResponse{}, fmt.Errorf("simulate %s.%s: %w", def.Mode, phaseDef.Phase, err)
		}
		if !resolved.Resolved() {
			// Simulation example-runs feed controlled, contract-satisfying
			// outputs; an abstain means the fixture output is malformed for the
			// declared schema, which is an authoring error worth surfacing loudly.
			return SimulationResponse{}, fmt.Errorf("simulate %s.%s: %s", def.Mode, phaseDef.Phase, resolved.AbstainReason())
		}
		// Classification-on-transition, deterministic rungs only: simulation
		// never spawns agents or calls a live classifier, so a preset for a
		// classified transition must short-circuit (seed the routing field) or
		// L1-derive (carry it inline on the source object). An abstain here is a
		// fixture authoring error, surfaced loudly like a resolution abstain.
		if cls := s.classifyTransitionRoutingForDef(ctx, def, &round, resolved.Envelope, false); cls != nil && cls.Abstained() {
			return SimulationResponse{}, fmt.Errorf("simulate %s.%s: %s; seed %q in the preset output", def.Mode, phaseDef.Phase, cls.AbstainReason(), cls.Field)
		}
		var transition *SimulationTransition
		nextSubCur := Phase("")
		if phaseDef.Delegated() {
			transition, nextSubCur, err = delegatedSimulationTransition(def, execDef, phaseDef.Phase, subCur, round)
			if err != nil {
				return SimulationResponse{}, fmt.Errorf("simulate %s.%s: %w", def.Mode, phaseDef.Phase, err)
			}
		} else {
			transition = simulationTransitionForCompletedRound(def, round)
		}
		subCur = nextSubCur
		stepCtx, err := simulationExecutionContext(def, phaseDef, inputs, round)
		if err != nil {
			return SimulationResponse{}, fmt.Errorf("simulate %s.%s: %w", def.Mode, phaseDef.Phase, err)
		}
		step := SimulationStep{
			Index:           len(trace),
			Phase:           string(phaseDef.Phase),
			PhaseKind:       string(execPhaseDef.Kind),
			Inputs:          inputs,
			Output:          result,
			Round:           round,
			Transition:      transition,
			Terminal:        transition == nil || strings.TrimSpace(transition.To) == "",
			SkillID:         execPhaseDef.SkillID,
			ProfileKey:      execPhaseDef.ProfileKey,
			PromptVariables: promptVariables(stepCtx, round, ""),
		}
		trace = append(trace, step)
		rounds = append(rounds, round)
		if step.Terminal {
			if err := assertSimulatedPath(preset, trace, stepIdx); err != nil {
				return SimulationResponse{}, err
			}
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

// simulationStepContext rebuilds the RunContext that fed a simulation step
// from its recorded inputs. It is the bridge that lets the render-preview
// endpoint substitute the exact same fixture data the trace already shows, and
// it is pure (no store reads or adapter resolution), matching the isolated
// nature of SimulateMode. The sandbox initiative doubles as the target
// instance; the mode's declared target kind still drives read composition.
func simulationStepContext(def Definition, phaseDef PhaseDefinition, inputs SimulationInputs) RunContext {
	rc := RunContext{
		Def:      def,
		PhaseDef: phaseDef,
		Target: TargetInstance{
			Kind:        def.Target.Kind,
			ID:          inputs.Initiative.Name,
			Title:       inputs.Initiative.Title,
			Description: inputs.Initiative.Description,
			Initiative:  inputs.Initiative,
			Items:       inputs.Items,
		},
		Artifacts: inputs.Artifacts,
		Rounds:    inputs.PriorRounds,
	}
	if def.Target.PlanRef.Required {
		// A bound-plan mode always has plan context by the time phases run;
		// the simulation substitutes a deterministic fixture so PLAN_CONTEXT
		// reads — and delegated plan-target sub-contexts — resolve without a
		// live plan-manager call.
		rc.Target.Plan = simulatedPlanContext()
	}
	return rc
}

// simulatedPlanContext is the deterministic bound-plan fixture simulation
// substitutes for a live plan-manager resolution.
func simulatedPlanContext() *PlanExecutionContext {
	return &PlanExecutionContext{
		Required:    true,
		Source:      "simulation",
		ExecutionID: "simulated-plan-execution",
		PlanID:      "simulated-plan",
		PlanRef: &PlanRef{
			Provider: PlanRefProviderPlanManager,
			PlanID:   "simulated-plan",
			Role:     PlanRefRoleOperatingModePlan,
		},
	}
}

// simulationExecutionContext resolves the run context whose prompt/reads a
// simulation step renders: the phase's own context, or — for a delegated
// phase — the sub-mode context derived from it, with the sub-phase read back
// from the simulated round's payload marker.
func simulationExecutionContext(def Definition, phaseDef PhaseDefinition, inputs SimulationInputs, round RoundEnvelope) (RunContext, error) {
	rc := simulationStepContext(def, phaseDef, inputs)
	if !phaseDef.Delegated() {
		return rc, nil
	}
	sub, err := delegationSubDefinition(phaseDef)
	if err != nil {
		return RunContext{}, err
	}
	subPhase, ok := delegatedRoundSubPhase(round)
	if !ok {
		return RunContext{}, fmt.Errorf("delegated phase %q round carries no %s marker", phaseDef.Phase, payloadDelegatedPhase)
	}
	subPhaseDef, err := sub.PhaseDefinition(subPhase)
	if err != nil {
		return RunContext{}, err
	}
	target, err := deriveSubTarget(def, sub, rc.Target)
	if err != nil {
		return RunContext{}, err
	}
	return RunContext{
		Def:       sub,
		PhaseDef:  subPhaseDef,
		Target:    target,
		Artifacts: rc.Artifacts,
		Rounds:    rc.Rounds,
	}, nil
}

// delegatedSimulationTransition routes a completed delegated round: the
// sub-mode's guards evaluate first — an onward sub-route renders as an inline
// self-edge on the delegating phase (the sub-loop continues there) — and a
// sub-mode stop hands the same output to the parent's guards. Returns the
// rendered transition plus the sub-phase the next visit should run (empty when
// the delegation ended).
func delegatedSimulationTransition(def, sub Definition, parentPhase, subPhase Phase, round RoundEnvelope) (*SimulationTransition, Phase, error) {
	lookup := RoundPayload(round.Payload).ResultFieldLookup()
	for _, gt := range sub.PhaseGraph.Guards[subPhase] {
		if !gt.When.Eval(lookup) {
			continue
		}
		if len(gt.To) == 0 {
			// Guarded stop inside the sub-mode: delegation ends; the parent's
			// guards route on the same output.
			return simulationTransitionForCompletedRound(def, round), "", nil
		}
		if len(gt.To) > 1 {
			return nil, "", fmt.Errorf("sub-mode %q phase %q routed to multiple targets %v; delegated sub-routes must be deterministic", sub.Mode, subPhase, gt.To)
		}
		transition := simulationTransitionFromGuard(parentPhase, gt.When)
		transition.To = string(parentPhase)
		return &transition, gt.To[0], nil
	}
	// Terminal sub-phase or no matching sub-guard: delegation ends.
	return simulationTransitionForCompletedRound(def, round), "", nil
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
	stepCtx, err := simulationExecutionContext(phaseDef, pd, step.Inputs, step.Round)
	if err != nil {
		return RenderPromptResponse{}, err
	}
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

// simPreset is the simulator's internal view of a preset: operator-facing
// metadata, the seeded sandbox initiative, and the ordered per-visit output
// overrides that steer the branch. It is assembled from a mode-owned ExampleRun,
// or synthesized (steps/expectedPath nil) for a phase mode that ships no
// example-runs. The seeded outputs only shape phase payloads; the real
// transition guards in simulationTransitionForCompletedRound still decide
// routing, so a preset can shape a branch but never fabricate one the graph
// disallows.
type simPreset struct {
	meta         SimulationPreset
	initiative   *ExampleRunInitiative
	steps        []ExampleRunStep
	expectedPath []string
}

// roundItems returns the seeded backlog items for the preset's sandbox
// initiative, falling back to generic defaults when the example-run declares
// none.
func (p simPreset) roundItems() []RoundItem {
	if p.initiative != nil && len(p.initiative.Items) > 0 {
		return p.initiative.RoundItems()
	}
	return defaultSimulationItems()
}

// criteria returns the seeded acceptance criteria, falling back to a generic
// default when the example-run declares none.
func (p simPreset) criteria() []string {
	if p.initiative != nil && len(p.initiative.Criteria) > 0 {
		return append([]string(nil), p.initiative.Criteria...)
	}
	return defaultSimulationCriteria()
}

// simulationPresets projects a mode's loaded example-runs into simulator
// presets. A phase mode with no example-runs gets a single synthesized
// happy-path that walks the graph on canned outputs, so the simulator remains
// usable before a mode authors fixtures.
func simulationPresets(def Definition) []simPreset {
	if len(def.ExampleRuns) == 0 {
		return []simPreset{syntheticHappyPath()}
	}
	out := make([]simPreset, 0, len(def.ExampleRuns))
	for _, run := range def.ExampleRuns {
		out = append(out, simPreset{
			meta: SimulationPreset{
				ID:          run.ID,
				Label:       defaultString(run.Label, humanizeToken(run.ID)),
				Description: run.Description,
				Branch:      run.Branch,
				Scenario:    run.Scenario,
			},
			initiative:   run.Initiative,
			steps:        run.Steps,
			expectedPath: run.ExpectedPath,
		})
	}
	return out
}

// syntheticHappyPath is the fallback preset for a phase mode that has not
// authored example-runs yet: no seeded steps, so every phase completes on canned
// outputs and the walk reaches the terminal phase. It declares no expected_path,
// so the walk is not asserted against a fixture.
func syntheticHappyPath() simPreset {
	return simPreset{
		meta: SimulationPreset{
			ID:          happyPathPresetID,
			Label:       "Happy path",
			Description: "Every phase completes cleanly and the mode reaches its terminal phase.",
			Branch:      "Straight-through completion with no loop-backs.",
			Scenario:    "A well-scoped initiative where the first pass through every phase succeeds.",
		},
	}
}

func presetMetadata(presets []simPreset) []SimulationPreset {
	out := make([]SimulationPreset, 0, len(presets))
	for _, preset := range presets {
		out = append(out, preset.meta)
	}
	return out
}

// resolveSimulationPreset returns the requested preset by id, or the first
// (happy-path) preset when the id is empty or unknown. Presets always has at
// least one entry for phase modes.
func resolveSimulationPreset(presets []simPreset, id string) simPreset {
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

// mergeSeededResult overlays an example-run step's declared output onto the
// contract-derived canned base, deep-merging nested objects (so a seeded
// `progress.decision` overrides just that leaf, not the whole progress object).
// The merge is generic — whatever fields the example-run declares override the
// scaffolding — so no branch-specific Go logic is needed.
//
// It returns both the typed result (for trace display) and the merged generic
// map (for envelope encoding). The map is the fidelity-preserving shape: a
// seeded field the runtime does not model — e.g. a routing value carried
// inline on the handoff for a classified transition — survives in the map even
// though the typed round-trip drops it, matching the runtime path where the
// raw envelope preserves emitted fields the typed PhaseResult does not.
func mergeSeededResult(base PhaseResult, seeded map[string]any) (PhaseResult, map[string]any, error) {
	baseBytes, err := json.Marshal(base)
	if err != nil {
		return PhaseResult{}, nil, fmt.Errorf("marshal canned result: %w", err)
	}
	baseMap := map[string]any{}
	if err := json.Unmarshal(baseBytes, &baseMap); err != nil {
		return PhaseResult{}, nil, fmt.Errorf("decode canned result: %w", err)
	}
	if len(seeded) == 0 {
		return base, baseMap, nil
	}
	deepMergeMap(baseMap, seeded)
	mergedBytes, err := json.Marshal(baseMap)
	if err != nil {
		return PhaseResult{}, nil, fmt.Errorf("marshal merged result: %w", err)
	}
	var result PhaseResult
	if err := json.Unmarshal(mergedBytes, &result); err != nil {
		return PhaseResult{}, nil, fmt.Errorf("decode merged result: %w", err)
	}
	return result, baseMap, nil
}

// deepMergeMap recursively overlays src onto dst: matching object keys merge,
// every other key (scalar/array) overwrites.
func deepMergeMap(dst, src map[string]any) {
	for key, value := range src {
		if srcMap, ok := value.(map[string]any); ok {
			if dstMap, ok := dst[key].(map[string]any); ok {
				deepMergeMap(dstMap, srcMap)
				continue
			}
		}
		dst[key] = value
	}
}

// assertSimulatedPath verifies the walked trace consumed every seeded step and,
// when the preset declares an expected_path, matches it exactly. A synthetic
// preset (no expected_path) is not asserted. This is the runtime twin of the
// load-time WalkExampleRun assertion, guarding against a fixture drifting from
// the guards it is meant to exercise.
func assertSimulatedPath(preset simPreset, trace []SimulationStep, stepIdx int) error {
	if stepIdx != len(preset.steps) {
		return fmt.Errorf("simulate preset %q: %d seeded step(s) unconsumed; steps must match the walked path", preset.meta.ID, len(preset.steps)-stepIdx)
	}
	if len(preset.expectedPath) == 0 {
		return nil
	}
	if len(trace) != len(preset.expectedPath) {
		return fmt.Errorf("simulate preset %q: walked %d phases, expected_path declares %d", preset.meta.ID, len(trace), len(preset.expectedPath))
	}
	for i, step := range trace {
		if step.Phase != preset.expectedPath[i] {
			return fmt.Errorf("simulate preset %q: walked phase[%d]=%q, expected_path declares %q", preset.meta.ID, i, step.Phase, preset.expectedPath[i])
		}
	}
	return nil
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

func simulationInitiative(def Definition, preset simPreset) InitiativeSnapshot {
	title, description := "", ""
	if preset.initiative != nil {
		title = preset.initiative.Title
		description = preset.initiative.Description
	}
	if strings.TrimSpace(title) == "" {
		title = def.Label + " Simulation"
	}
	if strings.TrimSpace(description) == "" {
		description = "Ephemeral sandbox initiative used to preview operating-mode flow."
	}
	items := preset.roundItems()
	itemRefs := make([]string, 0, len(items))
	for _, item := range items {
		itemRefs = append(itemRefs, item.Ref)
	}
	return InitiativeSnapshot{
		Name:               simulationInitiativeName,
		Title:              title,
		Description:        description,
		Mode:               string(def.Mode),
		Items:              itemRefs,
		AcceptanceCriteria: preset.criteria(),
	}
}

func cannedSimulationResult(def Definition, phaseDef PhaseDefinition, items []RoundItem) PhaseResult {
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
		result.BacklogSync = simulationBacklogSync(items)
	}
	return result
}

// simulationBacklogSync builds a realistic reconcile proposal from the seeded
// items: the primary item is marked completed and a scoped follow-up is
// proposed, so the reconcile step reads as a real backlog alignment rather than
// a no-op.
func simulationBacklogSync(items []RoundItem) *BacklogSyncPlan {
	plan := &BacklogSyncPlan{
		Rationale: "Align the backlog with the work the drained cycle just completed.",
	}
	if len(items) > 0 {
		plan.CompletedItems = []string{items[0].Ref}
	}
	if len(items) > 1 {
		plan.UpdatedItems = []string{items[1].Ref}
	}
	plan.CreatedItems = []string{"fix/simulation-followup"}
	return plan
}

// encodeEnvelopeMap encodes a generic result map (typed fields plus any seeded
// fields the runtime does not model) as the structured-result envelope string.
func encodeEnvelopeMap(result map[string]any) (string, error) {
	body, err := json.Marshal(map[string]any{resultEnvelopeKey: result})
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func simulationTransitionForCompletedRound(def Definition, round RoundEnvelope) *SimulationTransition {
	from := Phase(round.Phase)
	if guards := def.PhaseGraph.Guards[from]; len(guards) > 0 {
		lookup := RoundPayload(round.Payload).ResultFieldLookup()
		for _, gt := range guards {
			if !gt.When.Eval(lookup) {
				continue
			}
			transition := simulationTransitionFromGuard(from, gt.When)
			if len(gt.To) > 0 {
				transition.To = string(gt.To[0])
			}
			return &transition
		}
		return nil
	}
	next := def.PhaseGraph.Transitions[from]
	if len(next) == 0 {
		return nil
	}
	return &SimulationTransition{
		From:          string(from),
		To:            string(next[0]),
		ConditionKind: GuardOpAlways,
		Label:         GuardLabel(Guard{Op: GuardOpAlways}),
	}
}

func simulationTransitionFromGuard(from Phase, guard Guard) SimulationTransition {
	return SimulationTransition{
		From:          string(from),
		ConditionKind: GuardKind(guard),
		Label:         GuardLabel(guard),
		Field:         guard.Field,
		Value:         renderGuardValue(guard.Value),
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
