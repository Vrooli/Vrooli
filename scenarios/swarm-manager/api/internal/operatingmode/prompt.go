package operatingmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"swarm-manager/internal/operatingmode/promptcatalog"
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

// renderPhasePrompt is the single seam that resolves a phase's skill, validates
// it against the prompt catalog, substitutes the phase context into the
// template, and returns the literal prompt with its inputs. It performs no
// spawning, locking, or persistence, so it is safe to call from both the live
// phase runner and the simulation-preview endpoint.
func (s *Service) renderPhasePrompt(ctx context.Context, data phaseContext, round RoundEnvelope, note string) (renderedPhasePrompt, error) {
	if s.prompts == nil {
		return renderedPhasePrompt{}, fmt.Errorf("%w: prompt client not wired", ErrPromptRenderUnavailable)
	}
	if s.promptCatalog == nil {
		return renderedPhasePrompt{}, fmt.Errorf("%w: prompt catalog resolver not wired", ErrPromptRenderUnavailable)
	}
	skillID := data.phaseDef.SkillID
	entry, ok := s.promptCatalog(string(data.def.Mode), string(data.phaseDef.Phase))
	if !ok {
		return renderedPhasePrompt{}, fmt.Errorf("prompt catalog missing entry for mode %q phase %q", data.def.Mode, data.phaseDef.Phase)
	}
	if strings.TrimSpace(entry.CatalogID) != data.phaseDef.CatalogID {
		return renderedPhasePrompt{}, fmt.Errorf("prompt catalog ID mismatch for mode %q phase %q: registry=%q catalog=%q", data.def.Mode, data.phaseDef.Phase, data.phaseDef.CatalogID, entry.CatalogID)
	}
	if strings.TrimSpace(entry.SkillID) != skillID {
		return renderedPhasePrompt{}, fmt.Errorf("prompt catalog skill mismatch for mode %q phase %q: registry=%q catalog=%q", data.def.Mode, data.phaseDef.Phase, skillID, entry.SkillID)
	}
	variables := promptVariables(data, round, note)
	prompt, err := s.prompts.ReadSkill(ctx, skillID, variables, false)
	if err != nil {
		return renderedPhasePrompt{}, err
	}
	if strings.TrimSpace(prompt) == "" {
		return renderedPhasePrompt{}, fmt.Errorf("prompt skill %q rendered empty content", skillID)
	}
	return renderedPhasePrompt{
		SkillID:    skillID,
		ProfileKey: data.phaseDef.ProfileKey,
		Variables:  variables,
		Prompt:     prompt,
	}, nil
}

func (s *Service) buildPrompt(ctx context.Context, data phaseContext, round RoundEnvelope, note string) (string, error) {
	rendered, err := s.renderPhasePrompt(ctx, data, round, note)
	if err != nil {
		return "", err
	}
	return rendered.Prompt, nil
}

func promptVariables(data phaseContext, round RoundEnvelope, note string) map[string]string {
	return map[string]string{
		"INITIATIVE_NAME":        data.init.Name,
		"INITIATIVE_TITLE":       data.init.Title,
		"INITIATIVE_DESCRIPTION": data.init.Description,
		"OPERATING_MODE":         string(data.def.Mode),
		"MODE_LABEL":             data.def.Label,
		"PHASE":                  string(data.phaseDef.Phase),
		"RUN_STRATEGY":           string(data.def.RunStrategy.Kind),
		"ROUND_NUMBER":           fmt.Sprintf("%d", round.Round),
		"AGENT_PROFILE_KEY":      data.phaseDef.ProfileKey,
		"ACCEPTANCE_CRITERIA":    strings.Join(data.init.AcceptanceCriteria, "\n"),
		"OPERATOR_NOTE":          strings.TrimSpace(note),
		"MEMBER_ITEMS_JSON":      mustJSON(data.items),
		"MODE_ARTIFACTS_JSON":    mustJSON(data.artifacts),
		"PRIOR_ROUNDS_JSON":      mustJSON(data.rounds),
		promptcatalog.BacklogSyncProposalVariableKey: promptcatalog.BacklogSyncProposalSnippet(),
		promptcatalog.ElasticSliceVariableKey:        promptcatalog.ElasticSliceSnippet(),
	}
}

func mustJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "null"
	}
	return string(data)
}
