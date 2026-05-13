package operatingmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"swarm-manager/internal/operatingmode/promptcatalog"
)

func (s *Service) buildPrompt(ctx context.Context, data phaseContext, round RoundEnvelope, note string) (string, error) {
	if s.prompts == nil {
		return "", errors.New("prompt client not wired")
	}
	skillID := data.phaseDef.SkillID
	if s.promptCatalog == nil {
		return "", errors.New("prompt catalog resolver not wired")
	}
	entry, ok := s.promptCatalog(string(data.def.Mode), string(data.phaseDef.Phase))
	if !ok {
		return "", fmt.Errorf("prompt catalog missing entry for mode %q phase %q", data.def.Mode, data.phaseDef.Phase)
	}
	if strings.TrimSpace(entry.CatalogID) != data.phaseDef.CatalogID {
		return "", fmt.Errorf("prompt catalog ID mismatch for mode %q phase %q: registry=%q catalog=%q", data.def.Mode, data.phaseDef.Phase, data.phaseDef.CatalogID, entry.CatalogID)
	}
	if strings.TrimSpace(entry.SkillID) != skillID {
		return "", fmt.Errorf("prompt catalog skill mismatch for mode %q phase %q: registry=%q catalog=%q", data.def.Mode, data.phaseDef.Phase, skillID, entry.SkillID)
	}
	prompt, err := s.prompts.ReadSkill(ctx, skillID, promptVariables(data, round, note), false)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt skill %q rendered empty content", skillID)
	}
	return prompt, nil
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
	}
}

func mustJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "null"
	}
	return string(data)
}
