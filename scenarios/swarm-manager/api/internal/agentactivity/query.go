package agentactivity

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
)

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
	if records, err := s.store.Load(); err == nil {
		if err := s.reapExpiredNeedsReviewLocked(records, time.Now()); err != nil {
			return nil, err
		}
	}

	if err := s.refreshActiveLocked(ctx); err != nil {
		return nil, err
	}
	return s.listSnapshot(filters)
}

// ListSnapshot returns activity records from the persisted store without
// polling agent-manager. Graph projections use this snapshot path so opening
// the graph does not block on live run-state checks; the normal List endpoint
// keeps the refresh behavior for operators explicitly viewing activity state.
func (s *Service) ListSnapshot(_ context.Context, filters ListFilters) ([]Record, error) {
	return s.listSnapshot(filters)
}

func (s *Service) listSnapshot(filters ListFilters) ([]Record, error) {
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
