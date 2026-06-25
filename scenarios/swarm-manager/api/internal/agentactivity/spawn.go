package agentactivity

import (
	"context"
	"fmt"
	"log/slog"
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

func (s *Service) spawnTracked(
	ctx context.Context,
	spec Spec,
	req agentmanager.BacklogSpawnRequest,
) (agentmanager.RunResult, error) {
	if s.agentService == nil || !s.agentService.IsEnabled() {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	// Guard: at most one active agent per backlog item.
	if spec.OwnerType == OwnerBacklog {
		records, err = s.guardSingleActiveBacklogAgent(ctx, records, spec)
		if err != nil {
			s.mu.Unlock()
			return agentmanager.RunResult{}, err
		}
	}

	// Lane gate. Backlog spawns predate P2 — if PhaseKind is unset, fall
	// back to the per-Purpose default lane via LaneOf. Saturation returns
	// ErrLaneSaturated so QueueBacklog can leave the queued item pending.
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
		Metadata:        metadataWithProfileKey(spec.Metadata, req.ProfileKey),
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	runResult, spawnErr := s.agentService.SpawnBacklog(ctx, req)

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

// guardSingleActiveBacklogAgent enforces at most one active agent per backlog
// item: it refreshes the item's records, auto-fails stale pending spawns, and
// returns ErrBacklogItemBusy if an active agent already exists. Returns the
// (possibly reloaded) records for the caller to continue with. Must be called
// with s.mu held.
func (s *Service) guardSingleActiveBacklogAgent(ctx context.Context, records []Record, spec Spec) ([]Record, error) {
	// Best-effort refresh of this item's active records to clear stale state.
	if refreshErr := s.refreshActiveForOwnerLocked(ctx, records, spec.OwnerKind, spec.OwnerName); refreshErr != nil {
		slog.Warn("backlog item guard: refresh failed, proceeding with stale data", "err", refreshErr)
	}
	// Re-load after refresh may have saved updated statuses.
	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	staleCleaned := false
	for i, rec := range records {
		if rec.OwnerType != OwnerBacklog || rec.OwnerKind != spec.OwnerKind || rec.OwnerName != spec.OwnerName {
			continue
		}
		// Auto-fail stale pending records (no RunID, older than TTL).
		if isPendingStale(rec) {
			records[i].Status = StatusFailed
			records[i].FailureReason = "pending spawn timed out"
			records[i].FinishedAt = nowRFC3339()
			records[i].UpdatedAt = records[i].FinishedAt
			staleCleaned = true
			continue
		}
		if isActiveStatus(rec.Status) {
			return nil, fmt.Errorf("%w: %s/%s already has an active agent (activity=%s, status=%s, purpose=%s)",
				ErrBacklogItemBusy, rec.OwnerKind, rec.OwnerName, rec.ActivityID, rec.Status, rec.Purpose)
		}
	}
	if staleCleaned {
		if err := s.store.Save(records); err != nil {
			return nil, err
		}
	}
	return records, nil
}

// spawnInitiativeTracked is the initiative analogue of spawnTracked. There
// is no per-owner exclusivity guard — feedback rounds gate themselves via
// the on-disk feedback lock, which is checked before this is called.
func (s *Service) spawnInitiativeTracked(
	ctx context.Context,
	spec Spec,
	req agentmanager.InitiativeSpawnRequest,
) (agentmanager.RunResult, error) {
	if s.initiativeSpawner == nil {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}
	if s.agentService == nil || !s.agentService.IsEnabled() {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	// Lane gate. Initiative-scoped phase runs declare PhaseKind explicitly
	// (the phase runner reads it from PhaseDefinition.Kind), so saturation
	// here surfaces ErrLaneSaturated to the operating-mode round refresher,
	// which defers via the pending_auto_start sidecar (P4) on retry.
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
		Metadata:        metadataWithProfileKey(spec.Metadata, req.ProfileKey),
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	runResult, spawnErr := s.initiativeSpawner.SpawnInitiative(ctx, req)

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

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	// Lane gate. Session-scoped spawns (interactive operator sessions)
	// honor the same lanes — without phaseKind they fall back to the
	// per-Purpose default; saturation returns ErrLaneSaturated to the
	// caller (no enqueue path).
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
		Metadata:        metadataWithProfileKey(spec.Metadata, req.ProfileKey),
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	runResult, spawnErr := s.sessionSpawner.SpawnSession(ctx, req)

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
