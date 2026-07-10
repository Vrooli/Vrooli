package operatingmode

import (
	"strings"
	"time"

	"swarm-manager/internal/eventlog"
)

func (s *Service) emitPhaseStarted(round RoundEnvelope) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseStarted(round.ScopeID, s.phasePayload(round, "started", ""))
	}
}

func (s *Service) emitPhaseCompleted(round RoundEnvelope) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseCompleted(round.ScopeID, s.phasePayload(round, "completed", ""))
	}
}

func (s *Service) emitPhaseFailed(round RoundEnvelope, reason string) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseFailed(round.ScopeID, s.phasePayload(round, "failed", reason))
	}
}

func (s *Service) emitPhaseCanceled(round RoundEnvelope) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseCanceled(round.ScopeID, s.phasePayload(round, "canceled", ""))
	}
}

func (s *Service) emitParsedPhaseSignals(round RoundEnvelope) {
	if s.events == nil || round.Payload == nil {
		return
	}
	payload := RoundPayload(round.Payload)
	if payload.ReplanNeeded() {
		s.events.EmitOperatingModeReplanNeeded(round.ScopeID, s.phasePayload(round, "completed", ""))
	}
	if _, ok := payload.Progress(); ok {
		s.emitBacklogSynced(round, 0, 0, 0)
	}
}

func (s *Service) emitBacklogSynced(round RoundEnvelope, completed, created, updated int) {
	s.emitBacklogSyncedWithSource(round, completed, created, updated, BacklogMutationSource{}, nil)
}

func (s *Service) emitBacklogSyncedWithSource(round RoundEnvelope, completed, created, updated int, source BacklogMutationSource, itemRefs []string) {
	if s.events == nil {
		return
	}
	s.events.EmitOperatingModeBacklogSynced(round.ScopeID, backlogSyncPayload(round, completed, created, updated, source, itemRefs))
}

func backlogSyncPayload(round RoundEnvelope, completed, created, updated int, source BacklogMutationSource, itemRefs []string) eventlog.OperatingModeBacklogSyncPayload {
	payload := eventlog.OperatingModeBacklogSyncPayload{
		Mode:                  round.Mode,
		ScopeKind:             round.ScopeKind,
		ScopeID:               round.ScopeID,
		InitiativeName:        round.InitiativeName,
		Phase:                 round.Phase,
		RunStrategy:           round.RunStrategy,
		AgentProfileKey:       round.AgentProfileKey,
		RoundNumber:           round.Round,
		RunID:                 round.RunID,
		Status:                string(round.Status),
		BacklogItemsCompleted: completed,
		BacklogItemsCreated:   created,
		BacklogItemsUpdated:   updated,
		ItemRefs:              append([]string(nil), itemRefs...),
		ArtifactPaths:         artifactPaths(round.ArtifactUpdates),
	}
	if source.Entrypoint != "" || source.Mode != "" || source.RunID != "" {
		payload.Source = &eventlog.BacklogMutationSourcePayload{
			Entrypoint:     source.Entrypoint,
			InitiativeName: source.InitiativeName,
			Mode:           source.Mode,
			Phase:          source.Phase,
			Round:          source.Round,
			RunID:          source.RunID,
			RequestedBy:    source.RequestedBy,
		}
	}
	return payload
}

func phasePayload(round RoundEnvelope, status, reason string) eventlog.OperatingModePhasePayload {
	payload := eventlog.OperatingModePhasePayload{
		Mode:            round.Mode,
		ScopeKind:       round.ScopeKind,
		ScopeID:         round.ScopeID,
		InitiativeName:  round.InitiativeName,
		Phase:           round.Phase,
		PhaseKind:       string(phaseKindFor(Mode(round.Mode), Phase(round.Phase))),
		RunStrategy:     round.RunStrategy,
		AgentProfileKey: round.AgentProfileKey,
		RoundNumber:     round.Round,
		RunID:           round.RunID,
		Status:          status,
		ArtifactPaths:   artifactPaths(round.ArtifactUpdates),
	}
	if reason != "" {
		payload.Verdict = reason
	} else if round.Payload != nil {
		roundPayload := RoundPayload(round.Payload)
		if verdict := roundPayload.Verdict(); verdict != "" {
			payload.Verdict = verdict
		}
		if roundPayload.ReplanNeeded() {
			payload.ReplanNeeded = true
		}
	}
	payload.DurationSeconds = roundDuration(round)
	return payload
}

func (s *Service) phasePayload(round RoundEnvelope, status, reason string) eventlog.OperatingModePhasePayload {
	payload := phasePayload(round, status, reason)
	_, def, err := s.definitionBundleForRound(round)
	if err != nil {
		return payload
	}
	phaseDef, err := def.PhaseDefinition(Phase(round.Phase))
	if err == nil {
		payload.PhaseKind = string(phaseDef.Kind)
	}
	return payload
}

// phaseKindFor returns the PhaseKind for (mode, phase) by consulting the
// registry. Empty string when the mode or phase is unknown — stats and
// downstream consumers tolerate empty PhaseKind on legacy events written
// before this field landed.
func phaseKindFor(mode Mode, phase Phase) PhaseKind {
	def, err := DefinitionFor(mode)
	if err != nil {
		return ""
	}
	phaseDef, err := def.PhaseDefinition(phase)
	if err != nil {
		return ""
	}
	return phaseDef.Kind
}

func artifactPaths(updates []ArtifactUpdate) []string {
	if len(updates) == 0 {
		return nil
	}
	paths := make([]string, 0, len(updates))
	for _, update := range updates {
		if strings.TrimSpace(update.Path) != "" {
			paths = append(paths, update.Path)
		}
	}
	return paths
}

func roundDuration(round RoundEnvelope) float64 {
	if round.GeneratedAt == "" || round.Payload == nil {
		return 0
	}
	finished := RoundPayload(round.Payload).FinishedAt()
	if finished == "" {
		return 0
	}
	start, err1 := time.Parse(time.RFC3339, round.GeneratedAt)
	end, err2 := time.Parse(time.RFC3339, finished)
	if err1 != nil || err2 != nil {
		return 0
	}
	return end.Sub(start).Seconds()
}
