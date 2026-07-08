package operatingmode

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// echoRender deterministically substitutes the resolved variables into a fixed
// template so tests can (a) assert fixture data reaches the prompt and (b) prove
// the spawn path and preview path pass a byte-identical variable map.
func echoRender(skillID string, variables map[string]string) string {
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "SKILL=%s\n", skillID)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s=%s\n", key, variables[key])
	}
	return b.String()
}

// TestRenderPhasePromptParityWithBuildPrompt is the parity guard: the shared
// renderPhasePrompt seam must produce the exact prompt the real spawn path
// (buildPrompt) produces for equal inputs, so a preview never diverges from what
// an agent receives.
func TestRenderPhasePromptParityWithBuildPrompt(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})

	def, err := DefinitionFor(ModeHolisticLoop)
	if err != nil {
		t.Fatalf("DefinitionFor: %v", err)
	}
	sim, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "")
	if err != nil {
		t.Fatalf("SimulateMode: %v", err)
	}
	for _, step := range sim.Trace {
		pd, err := def.PhaseDefinition(Phase(step.Phase))
		if err != nil {
			t.Fatalf("PhaseDefinition %q: %v", step.Phase, err)
		}
		stepCtx := simulationStepContext(def, pd, step.Inputs)

		rendered, err := svc.renderPhasePrompt(context.Background(), stepCtx, step.Round, "")
		if err != nil {
			t.Fatalf("renderPhasePrompt %q: %v", step.Phase, err)
		}
		built, err := svc.buildPrompt(context.Background(), stepCtx, step.Round, "")
		if err != nil {
			t.Fatalf("buildPrompt %q: %v", step.Phase, err)
		}
		if rendered.Prompt != built {
			t.Fatalf("phase %q: renderPhasePrompt != buildPrompt\nrender:\n%s\nbuild:\n%s", step.Phase, rendered.Prompt, built)
		}
		if rendered.SkillID != pd.SkillID || rendered.ProfileKey != pd.ProfileKey {
			t.Fatalf("phase %q: rendered skill/profile = %q/%q, want %q/%q", step.Phase, rendered.SkillID, rendered.ProfileKey, pd.SkillID, pd.ProfileKey)
		}
	}
}

// TestRenderSimulationPromptSubstitutesFixtureData covers the render endpoint's
// core promise: every step of both phase modes across every preset renders a
// prompt containing the substituted initiative title, member-item refs, and
// acceptance criteria.
func TestRenderSimulationPromptSubstitutesFixtureData(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})

	modes := []Mode{ModeHolisticLoop, ModePhasedPlanDrain}
	for _, mode := range modes {
		sim, err := svc.SimulateMode(context.Background(), mode, "")
		if err != nil {
			t.Fatalf("SimulateMode %q: %v", mode, err)
		}
		for _, preset := range sim.Presets {
			presetSim, err := svc.SimulateMode(context.Background(), mode, preset.ID)
			if err != nil {
				t.Fatalf("SimulateMode %q/%q: %v", mode, preset.ID, err)
			}
			for i := range presetSim.Trace {
				resp, err := svc.RenderSimulationPrompt(context.Background(), mode, preset.ID, i)
				if err != nil {
					t.Fatalf("RenderSimulationPrompt %q/%q/%d: %v", mode, preset.ID, i, err)
				}
				if resp.Degraded {
					t.Fatalf("RenderSimulationPrompt %q/%q/%d unexpectedly degraded: %s", mode, preset.ID, i, resp.DegradedReason)
				}
				if resp.Prompt == "" {
					t.Fatalf("RenderSimulationPrompt %q/%q/%d empty prompt", mode, preset.ID, i)
				}
				// Substituted initiative title.
				if title := presetSim.Initiative.Title; title != "" && !strings.Contains(resp.Prompt, title) {
					t.Fatalf("%q/%q/%d prompt missing title %q:\n%s", mode, preset.ID, i, title, resp.Prompt)
				}
				// Substituted member-item refs (carried in MEMBER_ITEMS_JSON).
				for _, ref := range presetSim.Initiative.Items {
					if !strings.Contains(resp.Prompt, ref) {
						t.Fatalf("%q/%q/%d prompt missing item ref %q:\n%s", mode, preset.ID, i, ref, resp.Prompt)
					}
				}
				// Substituted acceptance criteria.
				for _, criterion := range presetSim.Initiative.AcceptanceCriteria {
					if !strings.Contains(resp.Prompt, criterion) {
						t.Fatalf("%q/%q/%d prompt missing criterion %q:\n%s", mode, preset.ID, i, criterion, resp.Prompt)
					}
				}
				if resp.SkillID != presetSim.Trace[i].SkillID {
					t.Fatalf("%q/%q/%d skill = %q, want %q", mode, preset.ID, i, resp.SkillID, presetSim.Trace[i].SkillID)
				}
			}
		}
	}
}

// TestRenderSimulationPromptDegradesWhenPromptClientUnavailable asserts an
// unwired prompt client yields a typed degraded response (Variables populated,
// no error) rather than a 500, so the UI can fall back to the variable map.
func TestRenderSimulationPromptDegradesWhenPromptClientUnavailable(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})
	svc.prompts = nil // simulate prompt-manager seam not wired

	resp, err := svc.RenderSimulationPrompt(context.Background(), ModeHolisticLoop, "", 0)
	if err != nil {
		t.Fatalf("RenderSimulationPrompt degraded path returned error: %v", err)
	}
	if !resp.Degraded {
		t.Fatalf("resp.Degraded = false, want true")
	}
	if resp.Prompt != "" {
		t.Fatalf("degraded prompt = %q, want empty", resp.Prompt)
	}
	if len(resp.Variables) == 0 {
		t.Fatalf("degraded response has no variables to fall back to")
	}
	if resp.Variables["INITIATIVE_TITLE"] == "" {
		t.Fatalf("degraded variables missing INITIATIVE_TITLE: %+v", resp.Variables)
	}
}

// TestRenderSimulationPromptRejectsOutOfRangeStep confirms an out-of-range step
// index is a caller error, not a rendered prompt.
func TestRenderSimulationPromptRejectsOutOfRangeStep(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})

	if _, err := svc.RenderSimulationPrompt(context.Background(), ModeHolisticLoop, "", 999); err == nil {
		t.Fatal("RenderSimulationPrompt out-of-range step error = nil, want error")
	} else if !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error = %v, want out-of-range", err)
	}
}

// TestRenderLivePromptSubstitutesInitiativeData renders a live initiative's
// phase prompt through the shared seam and asserts real initiative data is
// substituted.
func TestRenderLivePromptSubstitutesInitiativeData(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})

	resp, err := svc.RenderLivePrompt(context.Background(), "init-a", "investigate", 0, "focus on the mic path")
	if err != nil {
		t.Fatalf("RenderLivePrompt: %v", err)
	}
	if resp.Degraded {
		t.Fatalf("live render degraded unexpectedly: %s", resp.DegradedReason)
	}
	if resp.Prompt == "" || resp.SkillID == "" {
		t.Fatalf("live render missing prompt/skill: %+v", resp)
	}
	// The seeded fake initiative "init-a" carries this title and item ref.
	if !strings.Contains(resp.Prompt, "Init A") {
		t.Fatalf("live prompt missing initiative title:\n%s", resp.Prompt)
	}
	if !strings.Contains(resp.Prompt, "execute/do-thing") {
		t.Fatalf("live prompt missing member item ref:\n%s", resp.Prompt)
	}
	if !strings.Contains(resp.Prompt, "focus on the mic path") {
		t.Fatalf("live prompt missing operator note:\n%s", resp.Prompt)
	}
}

// TestRenderLivePromptDegradesWhenPromptClientUnavailable asserts the live path
// degrades the same way the simulation path does.
func TestRenderLivePromptDegradesWhenPromptClientUnavailable(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})
	svc.prompts = nil

	resp, err := svc.RenderLivePrompt(context.Background(), "init-a", "investigate", 0, "")
	if err != nil {
		t.Fatalf("RenderLivePrompt degraded path returned error: %v", err)
	}
	if !resp.Degraded || resp.Prompt != "" || len(resp.Variables) == 0 {
		t.Fatalf("degraded live response = %+v, want degraded with variables", resp)
	}
}

// TestBuildPromptReportsUnavailableSeam ensures the typed sentinel surfaces
// through the live spawn path too, so the runner's error handling stays intact.
func TestBuildPromptReportsUnavailableSeam(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})
	svc.prompts = nil

	def, err := DefinitionFor(ModeHolisticLoop)
	if err != nil {
		t.Fatalf("DefinitionFor: %v", err)
	}
	sim, err := newTestServiceWithOptions(t, t.TempDir(), serviceOptions{prompts: &fakePrompts{render: echoRender}}).
		SimulateMode(context.Background(), ModeHolisticLoop, "")
	if err != nil {
		t.Fatalf("SimulateMode: %v", err)
	}
	pd, err := def.PhaseDefinition(Phase(sim.Trace[0].Phase))
	if err != nil {
		t.Fatalf("PhaseDefinition: %v", err)
	}
	stepCtx := simulationStepContext(def, pd, sim.Trace[0].Inputs)
	if _, err := svc.buildPrompt(context.Background(), stepCtx, sim.Trace[0].Round, ""); !errors.Is(err, ErrPromptRenderUnavailable) {
		t.Fatalf("buildPrompt error = %v, want ErrPromptRenderUnavailable", err)
	}
}
