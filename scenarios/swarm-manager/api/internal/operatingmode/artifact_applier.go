package operatingmode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// applyPhaseResult resolves the completed round's agent output through the
// resolution ladder and, when a contract-satisfying result is recovered, applies
// it to the round (artifacts, payload, handoffs) and persists artifacts. The
// returned ResolvedPhaseResult carries the outcome the caller uses to decide the
// round's lifecycle: a resolved/recovered result completes the round; an abstain
// does not (the round's structured output could not be resolved). An error is
// returned only for an infrastructure or post-apply contract failure (e.g. a
// required artifact was not produced), never for imperfect model output.
func (s *Service) applyPhaseResult(ctx context.Context, round *RoundEnvelope, messages []string) (ResolvedPhaseResult, error) {
	def, err := DefinitionFor(Mode(round.Mode))
	if err != nil {
		return ResolvedPhaseResult{}, err
	}
	return s.applyPhaseResultWithPersistence(ctx, def, round, messages, true)
}

// applyPhaseResultInMemory resolves a round against an explicitly-provided
// Definition rather than re-fetching it from the process registry. Simulation
// (including authoring simulation of a draft mode loaded fresh from disk that is
// not yet registered) drives this path, so the Definition must flow in from the
// caller's walk.
func (s *Service) applyPhaseResultInMemory(ctx context.Context, def Definition, round *RoundEnvelope, output string) (ResolvedPhaseResult, error) {
	return s.applyPhaseResultWithPersistence(ctx, def, round, []string{output}, false)
}

func (s *Service) applyPhaseResultWithPersistence(ctx context.Context, def Definition, round *RoundEnvelope, messages []string, persistArtifacts bool) (ResolvedPhaseResult, error) {
	// A delegated round resolves against the sub-mode's phase contract (the
	// sub-phase's declared output steers the ladder and the applied-result
	// checks); artifact paths keep resolving against the parent mode's
	// artifact policy because the round lives in the parent run's tree.
	_, phaseDef, err := effectiveRoundExecution(def, *round)
	if err != nil {
		return ResolvedPhaseResult{}, err
	}

	resolved := resolvePhaseOutput(ctx, phaseDef, messages, s.classifier)
	// The resolution record is durable regardless of outcome so operators and the
	// UI can see which ladder rung resolved the round (or why it abstained).
	MutableRoundPayload(round).SetResolution(resolved.Record())
	if !resolved.Resolved() {
		// Honest abstain: the required structured output could not be resolved or
		// reconstructed. Do not apply a partial result — the caller keeps the
		// round out of clean completion so nothing auto-progresses on absent data.
		return resolved, nil
	}

	result := resolved.Result
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
	if result.PlanRef != nil {
		ref := normalizePlanRef(result.PlanRef)
		if err := validateOperatingModePlanRef(ref); err != nil {
			return resolved, err
		}
		payload.SetPlanRef(*ref)
		if persistArtifacts {
			if s.planRefBinder == nil {
				return resolved, fmt.Errorf("phase %q produced plan_ref but no initiative plan_ref binder is configured", phaseDef.Phase)
			}
			if _, err := s.planRefBinder.BindInitiativePlanRef(round.InitiativeName, *ref); err != nil {
				return resolved, err
			}
		}
	}
	writes := make([]stagedArtifactWrite, 0, len(result.Artifacts)+1)
	if result.Progress != nil {
		result.Progress.UpdatedAt = defaultString(result.Progress.UpdatedAt, now)
		payload.SetProgress(*result.Progress)
	}
	bindingWrites, bindingUpdates, err := resultBindingArtifacts(def, phaseDef, result, now)
	if err != nil {
		return resolved, err
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
			return resolved, err
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
		return resolved, err
	}
	if persistArtifacts {
		for _, write := range writes {
			if _, err := s.store.WriteArtifact(firstNonEmpty(staged.ScopeID, staged.InitiativeName), Mode(staged.Mode), write.Path, write.Content); err != nil {
				return resolved, err
			}
		}
	}
	*round = staged
	return resolved, nil
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
	if contract.RequiresPlanRef && result.PlanRef == nil {
		return fmt.Errorf("phase %q requires plan_ref", phaseDef.Phase)
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
