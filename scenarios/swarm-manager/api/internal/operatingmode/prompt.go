package operatingmode

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
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
}

// templateSlotRE matches an unsubstituted {{VARIABLE}} template slot left in a
// rendered prompt. Slots use SCREAMING_SNAKE names, matching the read
// vocabulary.
var templateSlotRE = regexp.MustCompile(`\{\{([A-Z][A-Z0-9_]*)\}\}`)

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
	prompt, err := s.prompts.ReadSkill(ctx, skillID, variables, false)
	if err != nil {
		return renderedPhasePrompt{}, err
	}
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
	}, nil
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
