package operatingmode

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *Service) applyPhaseResult(round *RoundEnvelope, output string) error {
	parsed, err := ParsePhaseResultDetailed(output)
	if err != nil {
		return err
	}
	def, err := DefinitionFor(Mode(round.Mode))
	if err != nil {
		return err
	}
	phaseDef, err := def.PhaseDefinition(Phase(round.Phase))
	if err != nil {
		return err
	}
	if err := validateParseStatus(phaseDef, parsed.Status); err != nil {
		return err
	}
	result := parsed.Result
	now := s.clock().UTC().Format(time.RFC3339)

	staged := cloneRoundForPhaseResult(*round)
	payload := MutableRoundPayload(&staged)
	payload.SetPhaseResult(result)
	if strings.TrimSpace(result.Verdict) != "" {
		payload.SetVerdict(result.Verdict)
	}
	if result.ReplanNeeded {
		payload.SetReplanNeeded(true)
	}
	if result.Readiness != nil {
		staged.Readiness = result.Readiness
	}
	writes := make([]stagedArtifactWrite, 0, len(result.Artifacts)+1)
	if result.Progress != nil {
		result.Progress.UpdatedAt = defaultString(result.Progress.UpdatedAt, now)
		payload.SetProgress(*result.Progress)
	}
	bindingWrites, bindingUpdates, err := resultBindingArtifacts(def, phaseDef, result, now)
	if err != nil {
		return err
	}
	writes = append(writes, bindingWrites...)
	staged.ArtifactUpdates = append(staged.ArtifactUpdates, bindingUpdates...)
	for _, artifact := range result.Artifacts {
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			continue
		}
		clean, err := cleanModeRelativePath(def, path)
		if err != nil {
			return err
		}
		declaration := artifactDeclaration(def, clean)
		writes = append(writes, stagedArtifactWrite{Path: clean, Content: []byte(artifact.Content)})
		staged.ArtifactUpdates = append(staged.ArtifactUpdates, ArtifactUpdate{
			Path:        clean,
			ContentType: defaultString(artifact.ContentType, declaration.ContentType),
			Required:    declaration.Required,
			UpdatedAt:   now,
			Source:      string(resultEnvelopeKey),
		})
	}
	if result.Handoff != nil {
		handoff := *result.Handoff
		handoff.CreatedAt = defaultString(handoff.CreatedAt, now)
		staged.Handoffs = append(staged.Handoffs, handoff)
	}
	for _, handoff := range result.Handoffs {
		handoff.CreatedAt = defaultString(handoff.CreatedAt, now)
		staged.Handoffs = append(staged.Handoffs, handoff)
	}
	if result.BacklogSync != nil {
		payload.SetBacklogSyncPlan(result.BacklogSync)
	}
	if err := validateAppliedPhaseResult(phaseDef, result, staged); err != nil {
		return err
	}
	for _, write := range writes {
		if _, err := s.store.WriteArtifact(staged.InitiativeName, Mode(staged.Mode), write.Path, write.Content); err != nil {
			return err
		}
	}
	*round = staged
	return nil
}

type stagedArtifactWrite struct {
	Path    string
	Content []byte
}

func resultBindingArtifacts(def Definition, phaseDef PhaseDefinition, result PhaseResult, updatedAt string) ([]stagedArtifactWrite, []ArtifactUpdate, error) {
	writes := make([]stagedArtifactWrite, 0, len(phaseDef.ResultBindings))
	updates := make([]ArtifactUpdate, 0, len(phaseDef.ResultBindings))
	for _, binding := range phaseDef.ResultBindings {
		switch binding.Kind {
		case ResultBindingProgressArtifact:
			if result.Progress == nil {
				continue
			}
			content, err := json.MarshalIndent(result.Progress, "", "  ")
			if err != nil {
				return nil, nil, err
			}
			clean, err := cleanModeRelativePath(def, binding.Artifact.Path)
			if err != nil {
				return nil, nil, err
			}
			declaration := artifactDeclaration(def, clean)
			writes = append(writes, stagedArtifactWrite{Path: clean, Content: content})
			updates = append(updates, ArtifactUpdate{
				Path:        clean,
				ContentType: defaultString(binding.Artifact.ContentType, declaration.ContentType),
				Required:    binding.Artifact.Required || declaration.Required,
				UpdatedAt:   updatedAt,
				Source:      string(resultEnvelopeKey),
			})
		default:
			return nil, nil, fmt.Errorf("phase %q has unsupported result binding %q", phaseDef.Phase, binding.Kind)
		}
	}
	return writes, updates, nil
}

func validateParseStatus(phaseDef PhaseDefinition, status PhaseResultParseStatus) error {
	if !phaseDef.OutputContract.RequiresStructuredResult {
		return nil
	}
	switch status {
	case PhaseResultParseValid:
		return nil
	case PhaseResultParseEmpty:
		return fmt.Errorf("phase %q produced an empty %s payload", phaseDef.Phase, resultEnvelopeKey)
	case PhaseResultParseMalformed:
		return fmt.Errorf("phase %q produced a malformed %s payload", phaseDef.Phase, resultEnvelopeKey)
	default:
		return fmt.Errorf("phase %q requires a structured %s payload", phaseDef.Phase, resultEnvelopeKey)
	}
}

func cloneRoundForPhaseResult(round RoundEnvelope) RoundEnvelope {
	clone := round
	if round.Payload != nil {
		clone.Payload = make(map[string]any, len(round.Payload)+4)
		for key, value := range round.Payload {
			clone.Payload[key] = value
		}
	}
	clone.ArtifactUpdates = append([]ArtifactUpdate(nil), round.ArtifactUpdates...)
	clone.Handoffs = append([]Handoff(nil), round.Handoffs...)
	clone.Items = append([]RoundItem(nil), round.Items...)
	return clone
}

func validateAppliedPhaseResult(phaseDef PhaseDefinition, result PhaseResult, round RoundEnvelope) error {
	contract := phaseDef.OutputContract
	if contract.RequiresProgress && result.Progress == nil {
		return fmt.Errorf("phase %q requires a valid progress decision", phaseDef.Phase)
	}
	if contract.RequiresVerdict && strings.TrimSpace(result.Verdict) == "" {
		return fmt.Errorf("phase %q requires a review verdict", phaseDef.Phase)
	}
	if contract.RequiresHandoff && result.Handoff == nil && len(result.Handoffs) == 0 {
		return fmt.Errorf("phase %q requires a durable handoff", phaseDef.Phase)
	}
	if contract.RequiresBacklogSync && result.BacklogSync == nil {
		return fmt.Errorf("phase %q requires a backlog_sync plan", phaseDef.Phase)
	}
	for _, required := range contract.RequiredArtifacts {
		if strings.TrimSpace(required.Path) == "" {
			continue
		}
		if !roundHasArtifactUpdate(round, required.Path) {
			return fmt.Errorf("phase %q requires artifact %q", phaseDef.Phase, required.Path)
		}
	}
	return nil
}

func roundHasArtifactUpdate(round RoundEnvelope, path string) bool {
	normalized := strings.TrimSpace(path)
	for _, update := range round.ArtifactUpdates {
		if update.Path == normalized {
			return true
		}
	}
	return false
}
