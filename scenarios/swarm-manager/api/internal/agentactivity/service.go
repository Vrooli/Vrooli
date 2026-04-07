package agentactivity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/idgen"
)

var errNotFound = errors.New("agent activity not found")

type rawAgentService interface {
	IsEnabled() bool
	IsAvailable(ctx context.Context) bool
	ResolveURL(ctx context.Context) (string, error)
	GetProfileID() string
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
	StopRun(ctx context.Context, runID string) error
}

type runContinuer interface {
	ContinueRun(ctx context.Context, runID string, message string) error
}

type ServiceConfig struct {
	StorePath    string
	AgentService rawAgentService
}

// Service is the single Swarm Manager seam for tracked agent-manager usage.
// All backlog/capture/scenario work spawns and continuations must flow through
// this service so durable activity records remain complete.
type Service struct {
	store           Store
	agentService    rawAgentService
	continuer       runContinuer
	eventDispatcher dispatch.NodeDispatcher
	mu              sync.Mutex
}

func NewService(cfg ServiceConfig) *Service {
	svc := &Service{
		store:        NewStore(cfg.StorePath),
		agentService: cfg.AgentService,
	}
	if continuer, ok := cfg.AgentService.(runContinuer); ok {
		svc.continuer = continuer
	}
	return svc
}

func (s *Service) SetEventDispatcher(d dispatch.NodeDispatcher) {
	s.eventDispatcher = d
}

func (s *Service) IsEnabled() bool {
	return s.agentService != nil && s.agentService.IsEnabled()
}

func (s *Service) IsAvailable(ctx context.Context) bool {
	if s.agentService == nil {
		return false
	}
	return s.agentService.IsAvailable(ctx)
}

func (s *Service) ResolveURL(ctx context.Context) (string, error) {
	if s.agentService == nil {
		return "", agentmanager.ErrNotAvailable
	}
	return s.agentService.ResolveURL(ctx)
}

func (s *Service) GetProfileID() string {
	if s.agentService == nil {
		return ""
	}
	return s.agentService.GetProfileID()
}

func (s *Service) SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	spec, err := specFromContext(ctx)
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	return s.spawnTracked(ctx, spec, req)
}

func (s *Service) ContinueRun(ctx context.Context, runID string, message string) error {
	spec, err := specFromContext(ctx)
	if err != nil {
		return err
	}
	return s.continueTracked(ctx, spec, runID, message)
}

func (s *Service) GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error) {
	if s.agentService == nil {
		return agentmanager.RunState{}, agentmanager.ErrNotAvailable
	}
	return s.agentService.GetRunState(ctx, runID)
}

func (s *Service) StopRun(ctx context.Context, runID string) error {
	if s.agentService == nil {
		return agentmanager.ErrNotAvailable
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return fmt.Errorf("run id is required")
	}
	if err := s.agentService.StopRun(ctx, trimmed); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return err
	}
	now := nowRFC3339()
	changed := false
	for i := range records {
		if strings.TrimSpace(records[i].RunID) != trimmed || !isActiveStatus(records[i].Status) {
			continue
		}
		records[i].Status = StatusCancelled
		records[i].FinishedAt = now
		records[i].UpdatedAt = now
		changed = true
	}
	if !changed {
		return nil
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	for _, record := range records {
		if record.RunID == trimmed && record.Status == StatusCancelled {
			s.dispatchStatusUpdate(record)
		}
	}
	return nil
}

func (s *Service) Get(ctx context.Context, activityID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.refreshActiveLocked(ctx); err != nil {
		return Record{}, err
	}
	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
	}
	trimmed := strings.TrimSpace(activityID)
	for _, record := range records {
		if record.ActivityID == trimmed {
			return record, nil
		}
	}
	return Record{}, errNotFound
}

func (s *Service) List(ctx context.Context, filters ListFilters) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.refreshActiveLocked(ctx); err != nil {
		return nil, err
	}
	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		if matchesFilters(record, filters) {
			filtered = append(filtered, record)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].RequestedAt == filtered[j].RequestedAt {
			return filtered[i].ActivityID > filtered[j].ActivityID
		}
		return filtered[i].RequestedAt > filtered[j].RequestedAt
	})

	return filtered, nil
}

// HasActiveAgent checks whether a backlog item has an active agent
// (pending, starting, running, or needs_review). Used by non-spawn guards
// like WorkshopDeleteRound and WorkshopReset to prevent mutations while
// an agent is working. Returns false on store errors (safe-side: allow
// operations to proceed if the store is broken).
func (s *Service) HasActiveAgent(ctx context.Context, ownerKind, ownerName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return false
	}
	_ = s.refreshActiveForOwnerLocked(ctx, records, ownerKind, ownerName)
	records, _ = s.store.Load()

	for _, rec := range records {
		if rec.OwnerType != OwnerBacklog || rec.OwnerKind != ownerKind || rec.OwnerName != ownerName {
			continue
		}
		if isPendingStale(rec) {
			continue
		}
		if isActiveStatus(rec.Status) {
			return true
		}
	}
	return false
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
		// Best-effort refresh of this item's active records to clear stale state.
		if refreshErr := s.refreshActiveForOwnerLocked(ctx, records, spec.OwnerKind, spec.OwnerName); refreshErr != nil {
			slog.Warn("backlog item guard: refresh failed, proceeding with stale data", "err", refreshErr)
		}
		// Re-load after refresh may have saved updated statuses.
		records, err = s.store.Load()
		if err != nil {
			s.mu.Unlock()
			return agentmanager.RunResult{}, err
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
				s.mu.Unlock()
				return agentmanager.RunResult{}, fmt.Errorf("%w: %s/%s already has an active agent (activity=%s, status=%s, purpose=%s)",
					ErrBacklogItemBusy, rec.OwnerKind, rec.OwnerName, rec.ActivityID, rec.Status, rec.Purpose)
			}
		}
		if staleCleaned {
			if err := s.store.Save(records); err != nil {
				s.mu.Unlock()
				return agentmanager.RunResult{}, err
			}
		}
	}

	now := nowRFC3339()
	record := Record{
		ActivityID:      idgen.Generate(),
		OwnerType:       spec.OwnerType,
		OwnerKind:       spec.OwnerKind,
		OwnerName:       spec.OwnerName,
		OwnerTitle:      spec.OwnerTitle,
		ExecutionID:     spec.ExecutionID,
		Purpose:         spec.Purpose,
		InteractionType: InteractionSpawn,
		Status:          StatusPending,
		RequestedAt:     now,
		RequestedBy:     spec.RequestedBy,
		Metadata:        spec.Metadata,
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
