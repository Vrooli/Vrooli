package agentactivity

import (
	"context"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
)

func (s *Service) refreshActiveLocked(ctx context.Context) error {
	if s.agentService == nil {
		return nil
	}

	records, err := s.store.Load()
	if err != nil {
		return err
	}

	stateByRunID := make(map[string]agentmanager.RunState)
	changed := false
	changedRecords := make(map[string]Record)

	for i := range records {
		record := &records[i]
		if !isActiveStatus(record.Status) || strings.TrimSpace(record.RunID) == "" {
			continue
		}

		runID := strings.TrimSpace(record.RunID)
		state, ok := stateByRunID[runID]
		if !ok {
			fetched, fetchErr := s.agentService.GetRunState(ctx, runID)
			if fetchErr != nil {
				continue
			}
			state = fetched
			stateByRunID[runID] = state
		}

		nextStatus, reason := mapRunStatus(state.Status, state.ErrorMsg)
		if nextStatus == record.Status &&
			record.StartedAt == strings.TrimSpace(state.StartedAt) &&
			record.FinishedAt == strings.TrimSpace(state.FinishedAt) &&
			record.FailureReason == strings.TrimSpace(reason) {
			continue
		}

		record.Status = nextStatus
		record.FailureReason = strings.TrimSpace(reason)
		record.UpdatedAt = nowRFC3339()
		if record.TaskID == "" {
			record.TaskID = strings.TrimSpace(state.TaskID)
		}
		if strings.TrimSpace(state.StartedAt) != "" {
			record.StartedAt = strings.TrimSpace(state.StartedAt)
		}
		if strings.TrimSpace(state.FinishedAt) != "" {
			record.FinishedAt = strings.TrimSpace(state.FinishedAt)
		} else if !isActiveStatus(nextStatus) {
			record.FinishedAt = record.UpdatedAt
		}
		changed = true
		changedRecords[record.ActivityID] = *record
	}

	if !changed {
		return nil
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	for _, record := range changedRecords {
		s.dispatchStatusUpdate(record)
	}
	return nil
}

func matchesFilters(record Record, filters ListFilters) bool {
	if filters.ActiveOnly && !isActiveStatus(record.Status) {
		return false
	}
	if value := strings.ToLower(strings.TrimSpace(filters.OwnerType)); value != "" && string(record.OwnerType) != value {
		return false
	}
	if value := strings.ToLower(strings.TrimSpace(filters.OwnerKind)); value != "" && strings.ToLower(record.OwnerKind) != value {
		return false
	}
	if value := strings.TrimSpace(filters.OwnerName); value != "" && record.OwnerName != value {
		return false
	}
	if value := strings.TrimSpace(filters.ExecutionID); value != "" && record.ExecutionID != value {
		return false
	}
	if value := strings.ToLower(strings.TrimSpace(filters.Purpose)); value != "" && string(record.Purpose) != value {
		return false
	}
	if value := strings.ToLower(strings.TrimSpace(filters.Status)); value != "" && string(record.Status) != value {
		return false
	}
	if value := strings.TrimSpace(filters.RunID); value != "" && record.RunID != value {
		return false
	}
	return true
}

func indexByID(records []Record, activityID string) int {
	for i := range records {
		if records[i].ActivityID == activityID {
			return i
		}
	}
	return -1
}

func mapRunStatus(status, errorMsg string) (Status, string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(StatusPending):
		return StatusPending, ""
	case string(StatusStarting):
		return StatusStarting, ""
	case string(StatusRunning):
		return StatusRunning, ""
	case string(StatusNeedsReview):
		return StatusNeedsReview, ""
	case string(StatusComplete):
		return StatusComplete, ""
	case string(StatusFailed):
		return StatusFailed, strings.TrimSpace(errorMsg)
	case string(StatusCancelled):
		return StatusCancelled, strings.TrimSpace(errorMsg)
	default:
		return StatusUnspecified, strings.TrimSpace(errorMsg)
	}
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
