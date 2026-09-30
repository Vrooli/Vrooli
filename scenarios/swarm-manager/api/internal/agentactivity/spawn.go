package agentactivity

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/idgen"
)

func metadataWithProfileKey(metadata map[string]string, profileKey string) map[string]string {
	key := strings.TrimSpace(profileKey)
	if key == "" {
		return metadata
	}
	out := make(map[string]string, len(metadata)+1)
	for k, v := range metadata {
		out[k] = v
	}
	out["agent_profile_key"] = key
	return out
}

func (s *Service) spawnSessionTracked(
	ctx context.Context,
	spec Spec,
	req agentmanager.SessionSpawnRequest,
) (agentmanager.RunResult, error) {
	if s.sessionSpawner == nil {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}
	if s.agentService == nil || !s.agentService.IsEnabled() {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}
	// Lane gate. Session-scoped spawns (interactive operator sessions)
	// honor the same lanes — without phaseKind they fall back to the
	// per-Purpose default; saturation returns ErrLaneSaturated to the
	// caller (no enqueue path).
	return s.spawnWithCallback(spec, req.ProfileKey, func() (agentmanager.RunResult, error) {
		return s.sessionSpawner.SpawnSession(ctx, req)
	})
}

// spawnWithCallback handles the record lifecycle for human session spawns:
// acquire the lock, check lane capacity, write a pending record, release the
// lock, call spawnFn, then reacquire the lock to record the outcome.
func (s *Service) spawnWithCallback(
	spec Spec,
	profileKey string,
	spawnFn func() (agentmanager.RunResult, error),
) (agentmanager.RunResult, error) {
	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	if err := s.checkLaneCapacityLocked(spec, records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	now := nowRFC3339()
	record := Record{
		ActivityID:      idgen.Generate(),
		OwnerType:       spec.OwnerType,
		OwnerKind:       spec.OwnerKind,
		OwnerName:       spec.OwnerName,
		OwnerTitle:      spec.OwnerTitle,
		ExecutionID:     spec.ExecutionID,
		PhaseKind:       spec.PhaseKind,
		Purpose:         spec.Purpose,
		InteractionType: InteractionSpawn,
		Status:          StatusPending,
		RequestedAt:     now,
		RequestedBy:     spec.RequestedBy,
		Metadata:        metadataWithProfileKey(spec.Metadata, profileKey),
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	runResult, spawnErr := spawnFn()

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err = s.store.Load()
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	idx := indexByID(records, record.ActivityID)
	if idx < 0 {
		return agentmanager.RunResult{}, fmt.Errorf("agent activity disappeared during spawn")
	}

	updated := records[idx]
	updated.UpdatedAt = nowRFC3339()
	if spawnErr != nil {
		updated.Status = StatusFailed
		updated.FailureReason = spawnErr.Error()
		updated.FinishedAt = updated.UpdatedAt
		records[idx] = updated
		if err := s.store.Save(records); err != nil {
			return agentmanager.RunResult{}, err
		}
		s.dispatchStatusUpdate(updated)
		return agentmanager.RunResult{}, spawnErr
	}

	updated.TaskID = strings.TrimSpace(runResult.TaskID)
	updated.RunID = strings.TrimSpace(runResult.RunID)
	updated.Status = StatusStarting
	updated.StartedAt = updated.UpdatedAt
	records[idx] = updated
	if err := s.store.Save(records); err != nil {
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(updated)
	return runResult, nil
}

func (s *Service) continueTracked(ctx context.Context, spec Spec, runID string, message string) error {
	if s.continuer == nil {
		return fmt.Errorf("run continuation not available")
	}
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return fmt.Errorf("run id is required")
	}

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	now := nowRFC3339()
	record := Record{
		ActivityID:      idgen.Generate(),
		OwnerType:       spec.OwnerType,
		OwnerKind:       spec.OwnerKind,
		OwnerName:       spec.OwnerName,
		OwnerTitle:      spec.OwnerTitle,
		ExecutionID:     spec.ExecutionID,
		PhaseKind:       spec.PhaseKind,
		Purpose:         spec.Purpose,
		InteractionType: InteractionContinue,
		RunID:           trimmedRunID,
		Status:          StatusPending,
		RequestedAt:     now,
		RequestedBy:     spec.RequestedBy,
		Metadata:        spec.Metadata,
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	continueErr := s.continuer.ContinueRun(ctx, trimmedRunID, message)

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err = s.store.Load()
	if err != nil {
		return err
	}
	idx := indexByID(records, record.ActivityID)
	if idx < 0 {
		return fmt.Errorf("agent activity disappeared during continuation")
	}
	updated := records[idx]
	updated.UpdatedAt = nowRFC3339()
	if continueErr != nil {
		updated.Status = StatusFailed
		updated.FailureReason = continueErr.Error()
		updated.FinishedAt = updated.UpdatedAt
	} else {
		updated.Status = StatusRunning
		updated.StartedAt = updated.UpdatedAt
	}
	records[idx] = updated
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(updated)
	return continueErr
}
