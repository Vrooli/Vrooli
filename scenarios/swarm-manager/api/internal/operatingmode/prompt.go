package operatingmode

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"swarm-manager/internal/promptmanager"
)

// ErrPromptRenderUnavailable signals the no-spawn render seam cannot run
// because the prompt client or catalog resolver is not wired. Callers that can
// degrade gracefully (the render-preview endpoint) map it to a typed response
// so the UI can fall back to the resolved variable map instead of surfacing a
// 500.
var ErrPromptRenderUnavailable = errors.New("operating-mode prompt render seam unavailable")

// renderedPhasePrompt is the fully substituted agent prompt together with the
// inputs that produced it. Both the real spawn path (buildPrompt) and the
// render-preview endpoint build it from renderPhasePrompt so a preview is
// byte-identical to what an agent actually receives.
type renderedPhasePrompt struct {
	SkillID    string
	ProfileKey string
	Variables  map[string]string
	Prompt     string
	Source     *PinnedPromptSource
}

// templateSlotRE matches an unsubstituted {{VARIABLE}} template slot left in a
// rendered prompt. Slots use SCREAMING_SNAKE names, matching the read
// vocabulary.
var templateSlotRE = regexp.MustCompile(`\{\{([A-Z][A-Z0-9_]*)\}\}`)

func promptSourceKey(mode Mode, phase Phase) string {
	return string(mode) + "/" + string(phase)
}

func (s *Service) pinReachablePromptSources(ctx context.Context, def Definition) (map[string]PinnedPromptSource, error) {
	bundle, _, err := pinDefinitionBundle(def, DefinitionFor)
	if err != nil {
		return nil, err
	}
	return s.pinPromptSourcesFromBundle(ctx, bundle)
}

func (s *Service) pinPromptSourcesFromBundle(ctx context.Context, bundle DefinitionBundle) (map[string]PinnedPromptSource, error) {
	sourceClient, ok := s.prompts.(promptmanager.SourceClient)
	if !ok {
		return nil, fmt.Errorf("%w: prompt client cannot snapshot immutable skill sources", ErrPromptRenderUnavailable)
	}
	root, err := bundle.RootDefinition()
	if err != nil {
		return nil, err
	}
	compiled, err := CompileInputContract(bundle.Definitions, root)
	if err != nil {
		return nil, err
	}

	modes := make([]string, 0, len(bundle.Definitions))
	for mode := range bundle.Definitions {
		modes = append(modes, string(mode))
	}
	sort.Strings(modes)
	pinned := map[string]PinnedPromptSource{}
	for _, modeName := range modes {
		mode := Mode(modeName)
		modeDef := bundle.Definitions[mode]
		phases := make([]string, 0, len(modeDef.PhaseGraph.Phases))
		for phase := range modeDef.PhaseGraph.Phases {
			phases = append(phases, string(phase))
		}
		sort.Strings(phases)
		for _, phaseName := range phases {
			phase := Phase(phaseName)
			phaseDef := modeDef.PhaseGraph.Phases[phase]
			if phaseDef.Delegated() {
				continue
			}
			expected, err := compiledPhaseTemplateVariables(compiled, mode, phase)
			if err != nil {
				return nil, err
			}
			snapshot, err := sourceClient.ReadSkillSource(ctx, phaseDef.SkillID, expected)
			if err != nil {
				return nil, fmt.Errorf("pin prompt source for mode %q phase %q: %w", mode, phase, err)
			}
			if strings.TrimSpace(snapshot.SkillID) != phaseDef.SkillID {
				return nil, fmt.Errorf("pin prompt source for mode %q phase %q: skill mismatch %q != %q", mode, phase, snapshot.SkillID, phaseDef.SkillID)
			}
			if snapshot.Revision <= 0 || strings.TrimSpace(snapshot.Content) == "" || strings.TrimSpace(snapshot.ContentHash) == "" {
				return nil, fmt.Errorf("pin prompt source for mode %q phase %q: incomplete immutable source metadata", mode, phase)
			}
			actualDigest := sha256.Sum256([]byte(snapshot.Content))
			if want := fmt.Sprintf("sha256:%x", actualDigest[:]); snapshot.ContentHash != want {
				return nil, fmt.Errorf("pin prompt source for mode %q phase %q: content hash %q does not match %q", mode, phase, snapshot.ContentHash, want)
			}
			actualVariables := append([]string(nil), snapshot.TemplateVariables...)
			sort.Strings(actualVariables)
			if strings.Join(actualVariables, "\x00") != strings.Join(expected, "\x00") {
				return nil, fmt.Errorf("pin prompt source for mode %q phase %q: template variables %v do not exactly match compiled bindings %v", mode, phase, actualVariables, expected)
			}
			pinned[promptSourceKey(mode, phase)] = PinnedPromptSource{
				Mode: modeName, Phase: phaseName, SkillID: phaseDef.SkillID,
				Revision: strconv.Itoa(snapshot.Revision), Variant: snapshot.SelectedVariantID,
				Content: snapshot.Content, ContentHash: snapshot.ContentHash,
				TemplateVariables: actualVariables, Retention: string(InputRetentionValue),
			}
		}
	}
	return pinned, nil
}

func compiledPhaseTemplateVariables(compiled CompiledInputContract, mode Mode, phase Phase) ([]string, error) {
	for _, modeContract := range compiled.Modes {
		if modeContract.Mode != mode {
			continue
		}
		for _, phaseContract := range modeContract.Phases {
			if phaseContract.Phase != phase {
				continue
			}
			variables := make([]string, 0, len(phaseContract.Bindings))
			for _, binding := range phaseContract.Bindings {
				variables = append(variables, binding.Variable)
			}
			sort.Strings(variables)
			return variables, nil
		}
		return nil, fmt.Errorf("compiled input contract for mode %q has no phase %q", mode, phase)
	}
	return nil, fmt.Errorf("compiled input contract has no mode %q", mode)
}

// renderPhasePrompt is the single seam that resolves a phase's skill, validates
// it against the prompt catalog, substitutes the phase's declared reads into
// the template, and returns the literal prompt with its inputs. Substitution
// uses exactly the declared read contract (RunContext.DeclaredReads); a
// template slot the declared reads do not satisfy fails loudly rather than
// rendering an empty value. It performs no spawning, locking, or persistence,
// so it is safe to call from both the live phase runner and the
// simulation-preview endpoint.
func (s *Service) renderPhasePrompt(ctx context.Context, rc RunContext, round RoundEnvelope, note string) (renderedPhasePrompt, error) {
	if s.prompts == nil {
		return renderedPhasePrompt{}, fmt.Errorf("%w: prompt client not wired", ErrPromptRenderUnavailable)
	}
	if s.promptCatalog == nil {
		return renderedPhasePrompt{}, fmt.Errorf("%w: prompt catalog resolver not wired", ErrPromptRenderUnavailable)
	}
	skillID := rc.PhaseDef.SkillID
	entry, ok := s.promptCatalog(string(rc.Def.Mode), string(rc.PhaseDef.Phase))
	if !ok {
		return renderedPhasePrompt{}, fmt.Errorf("prompt catalog missing entry for mode %q phase %q", rc.Def.Mode, rc.PhaseDef.Phase)
	}
	if strings.TrimSpace(entry.CatalogID) != rc.PhaseDef.CatalogID {
		return renderedPhasePrompt{}, fmt.Errorf("prompt catalog ID mismatch for mode %q phase %q: registry=%q catalog=%q", rc.Def.Mode, rc.PhaseDef.Phase, rc.PhaseDef.CatalogID, entry.CatalogID)
	}
	if strings.TrimSpace(entry.SkillID) != skillID {
		return renderedPhasePrompt{}, fmt.Errorf("prompt catalog skill mismatch for mode %q phase %q: registry=%q catalog=%q", rc.Def.Mode, rc.PhaseDef.Phase, skillID, entry.SkillID)
	}
	variables, err := rc.DeclaredReads(round, note)
	if err != nil {
		return renderedPhasePrompt{}, err
	}
	var prompt string
	if rc.Execution != nil {
		source, ok := rc.Execution.ReachablePromptSources[promptSourceKey(rc.Def.Mode, rc.PhaseDef.Phase)]
		if !ok {
			return renderedPhasePrompt{}, fmt.Errorf("execution %q has no pinned prompt source for mode %q phase %q", rc.Execution.ExecutionID, rc.Def.Mode, rc.PhaseDef.Phase)
		}
		actualDigest := sha256.Sum256([]byte(source.Content))
		if want := fmt.Sprintf("sha256:%x", actualDigest[:]); source.ContentHash != want {
			return renderedPhasePrompt{}, fmt.Errorf("execution %q pinned prompt source hash mismatch for mode %q phase %q", rc.Execution.ExecutionID, rc.Def.Mode, rc.PhaseDef.Phase)
		}
		prompt = source.Content
		sourceCopy := source
		keys := make([]string, 0, len(variables))
		for name := range variables {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			prompt = strings.ReplaceAll(prompt, "{{"+name+"}}", variables[name])
		}
		return validateRenderedPhasePrompt(skillID, rc, variables, prompt, &sourceCopy)
	} else {
		prompt, err = s.prompts.ReadSkill(ctx, skillID, variables, false)
		if err != nil {
			return renderedPhasePrompt{}, err
		}
	}
	return validateRenderedPhasePrompt(skillID, rc, variables, prompt, nil)
}

func validateRenderedPhasePrompt(skillID string, rc RunContext, variables map[string]string, prompt string, source *PinnedPromptSource) (renderedPhasePrompt, error) {
	if strings.TrimSpace(prompt) == "" {
		return renderedPhasePrompt{}, fmt.Errorf("prompt skill %q rendered empty content", skillID)
	}
	if unsatisfied := unsatisfiedTemplateSlots(prompt); len(unsatisfied) > 0 {
		return renderedPhasePrompt{}, fmt.Errorf("prompt skill %q for mode %q phase %q has unsatisfied template slot(s) %v: declare them in the phase reads and ensure target %q provides them", skillID, rc.Def.Mode, rc.PhaseDef.Phase, unsatisfied, rc.Def.Target.Kind)
	}
	return renderedPhasePrompt{
		SkillID:    skillID,
		ProfileKey: rc.PhaseDef.ProfileKey,
		Variables:  variables,
		Prompt:     prompt,
		Source:     source,
	}, nil
}

func promptRenderTrace(rendered renderedPhasePrompt, execution OperatingModeExecution) (*PromptRenderTrace, error) {
	if rendered.Source == nil {
		return nil, fmt.Errorf("pinned prompt source is required for a durable round trace")
	}
	variablesJSON, err := json.Marshal(rendered.Variables)
	if err != nil {
		return nil, fmt.Errorf("marshal rendered prompt variables: %w", err)
	}
	variablesDigest, err := canonicalJSONDigest(variablesJSON)
	if err != nil {
		return nil, fmt.Errorf("digest rendered prompt variables: %w", err)
	}
	promptDigest := sha256.Sum256([]byte(rendered.Prompt))
	inputRetention, err := promptInputRetentionTrace(execution, *rendered.Source, rendered.Variables)
	if err != nil {
		return nil, err
	}
	return &PromptRenderTrace{
		SkillID: rendered.SkillID, SourceRevision: rendered.Source.Revision,
		SourceVariant: rendered.Source.Variant, SourceHash: rendered.Source.ContentHash,
		VariablesHash: variablesDigest, RenderedPromptHash: fmt.Sprintf("sha256:%x", promptDigest[:]),
		DefinitionDigest: execution.DefinitionDigest, InputContractDigest: execution.InputContractDigest,
		RedactionMetadata: map[string]any{
			"variables_persisted": false, "rendered_prompt_persisted": false,
			"policy": "hashes_only", "inputs": inputRetention,
		},
	}, nil
}

func promptInputRetentionTrace(execution OperatingModeExecution, source PinnedPromptSource, variables map[string]string) (map[string]any, error) {
	var compiled CompiledInputContract
	if len(execution.CompiledInputContract) > 0 {
		if err := json.Unmarshal(execution.CompiledInputContract, &compiled); err != nil {
			return nil, fmt.Errorf("decode execution input contract for prompt trace: %w", err)
		}
	} else {
		// Adopted pre-Phase-4 executions have an immutable definition bundle but
		// no compiled-input slot. Derive the trace metadata from that pinned
		// bundle, never from the mutable live registry.
		root, err := execution.DefinitionBundle.RootDefinition()
		if err != nil {
			return nil, fmt.Errorf("resolve pinned definition for legacy prompt trace: %w", err)
		}
		compiled, err = CompileInputContract(execution.DefinitionBundle.Definitions, root)
		if err != nil {
			return nil, fmt.Errorf("compile pinned legacy input contract for prompt trace: %w", err)
		}
	}
	for _, mode := range compiled.Modes {
		if mode.Mode != Mode(source.Mode) {
			continue
		}
		inputs := make(map[string]CompiledInput, len(mode.Inputs))
		for _, input := range mode.Inputs {
			inputs[input.Spec.ID] = input
		}
		for _, phase := range mode.Phases {
			if phase.Phase != Phase(source.Phase) {
				continue
			}
			out := make(map[string]any, len(phase.Bindings))
			for _, binding := range phase.Bindings {
				input, ok := inputs[binding.InputID]
				if !ok {
					return nil, fmt.Errorf("prompt trace binding %q references missing input %q", binding.Variable, binding.InputID)
				}
				entry := map[string]any{
					"input_id": input.Spec.ID, "sensitivity": input.Spec.Sensitivity,
					"retention": input.Spec.Retention, "value_persisted": false,
				}
				if input.Spec.Retention == InputRetentionDigest {
					valueDigest := sha256.Sum256([]byte(variables[binding.Variable]))
					entry["value_digest"] = fmt.Sprintf("sha256:%x", valueDigest[:])
				}
				out[binding.Variable] = entry
			}
			return out, nil
		}
		return nil, fmt.Errorf("prompt trace input contract for mode %q has no phase %q", source.Mode, source.Phase)
	}
	return nil, fmt.Errorf("prompt trace input contract has no mode %q", source.Mode)
}

// unsatisfiedTemplateSlots returns the sorted, deduplicated {{VARIABLE}} slots
// still present in a rendered prompt.
func unsatisfiedTemplateSlots(prompt string) []string {
	matches := templateSlotRE.FindAllStringSubmatch(prompt, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if _, ok := seen[m[1]]; ok {
			continue
		}
		seen[m[1]] = struct{}{}
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

func (s *Service) buildPrompt(ctx context.Context, rc RunContext, round RoundEnvelope, note string) (string, error) {
	rendered, err := s.renderPhasePrompt(ctx, rc, round, note)
	if err != nil {
		return "", err
	}
	return rendered.Prompt, nil
}

// promptVariables resolves the phase's declared reads for display surfaces
// (render previews, simulation traces). Resolution failures degrade to an
// empty map rather than failing the surface; the render path itself surfaces
// the typed error.
func promptVariables(rc RunContext, round RoundEnvelope, note string) map[string]string {
	reads, err := rc.DeclaredReads(round, note)
	if err != nil {
		return map[string]string{}
	}
	return reads
}
