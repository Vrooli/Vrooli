package operatingmode

import (
	"encoding/json"
	"strings"
	"time"
)

func (s *Service) applyPhaseResult(round *RoundEnvelope, output string) error {
	result, ok, err := ParsePhaseResult(output)
	if err != nil || !ok {
		return err
	}
	now := s.clock().UTC().Format(time.RFC3339)
	round.Payload = ensurePayload(round.Payload)
	round.Payload[resultEnvelopeKey] = result
	if strings.TrimSpace(result.Verdict) != "" {
		round.Payload["verdict"] = strings.TrimSpace(result.Verdict)
	}
	if result.ReplanNeeded {
		round.Payload["replan_needed"] = true
	}
	if result.Readiness != nil {
		round.Readiness = result.Readiness
	}
	if result.Progress != nil {
		result.Progress.UpdatedAt = defaultString(result.Progress.UpdatedAt, now)
		round.Payload["progress"] = *result.Progress
		if round.Mode == string(ModePhasedPlanDrain) {
			data, err := json.MarshalIndent(result.Progress, "", "  ")
			if err != nil {
				return err
			}
			clean, err := s.store.WriteArtifact(round.InitiativeName, Mode(round.Mode), "modes/phased-plan-drain/progress.json", data)
			if err != nil {
				return err
			}
			round.ArtifactUpdates = append(round.ArtifactUpdates, ArtifactUpdate{Path: clean, ContentType: "application/json", Required: true, UpdatedAt: now, Source: string(resultEnvelopeKey)})
		}
	}
	for _, artifact := range result.Artifacts {
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			continue
		}
		clean, err := s.store.WriteArtifact(round.InitiativeName, Mode(round.Mode), path, []byte(artifact.Content))
		if err != nil {
			return err
		}
		declaration := artifactDeclaration(MustDefinition(Mode(round.Mode)), clean)
		round.ArtifactUpdates = append(round.ArtifactUpdates, ArtifactUpdate{
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
		round.Handoffs = append(round.Handoffs, handoff)
	}
	for _, handoff := range result.Handoffs {
		handoff.CreatedAt = defaultString(handoff.CreatedAt, now)
		round.Handoffs = append(round.Handoffs, handoff)
	}
	if result.BacklogSync != nil {
		round.Payload["backlog_sync_plan"] = result.BacklogSync
	}
	return nil
}
