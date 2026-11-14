package database

import (
	"context"
	"encoding/json"
	"fmt"

	"agent-manager/internal/domain"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

func activeEffortOrchestratorIDs(candidate *domain.Run) []string {
	if candidate == nil {
		return nil
	}
	effortIDs := make([]string, 0, len(candidate.WorkReferences))
	for _, ref := range candidate.WorkReferences {
		if domain.IsPublicActiveEffortOrchestratorReference(ref) {
			effortIDs = append(effortIDs, ref.GetId())
		}
	}
	return effortIDs
}

func activeEffortOrchestratorConflictForIDs(ctx context.Context, tx *Tx, candidateID string, effortIDs []string) (bool, error) {
	if len(effortIDs) == 0 {
		return false, nil
	}
	args, err := effortOrchestratorOwnerArgs(effortIDs)
	if err != nil {
		return false, err
	}
	var conflict bool
	queryArgs := []any{args[0], candidateID}
	queryArgs = append(queryArgs, args[1:]...)
	err = tx.QueryRowContext(ctx, `SELECT EXISTS (`+effortOrchestratorOwnerSubquery("?")+`)`, queryArgs...).Scan(&conflict)
	if err != nil {
		return false, fmt.Errorf("check active effort orchestrator owner: %w", err)
	}
	return conflict, nil
}

func effortOrchestratorOwnerSubquery(excludedRunID string) string {
	return `SELECT 1 FROM runs existing,
		json_each(CASE WHEN json_valid(existing.work_references) THEN existing.work_references ELSE '[]' END) declared,
		json_each(?) candidate
	WHERE existing.id <> ` + excludedRunID + `
		AND existing.status IN (?, ?, ?, ?, ?)
		AND json_extract(declared.value, '$.kind') = 'effort'
		AND json_extract(declared.value, '$.id') = candidate.value
		AND json_extract(declared.value, '$.relationship') = 'orchestrator'
		AND json_extract(declared.value, '$.verified') = 1
		AND json_extract(declared.value, '$.visibility') = ?
		AND json_extract(declared.value, '$.state') = ?`
}

func effortOrchestratorOwnerArgs(effortIDs []string) ([]any, error) {
	if len(effortIDs) == 0 {
		return nil, nil
	}
	candidateIDs, err := json.Marshal(effortIDs)
	if err != nil {
		return nil, err
	}
	return []any{
		string(candidateIDs),
		string(domain.RunStatusPending), string(domain.RunStatusStarting), string(domain.RunStatusRunning), string(domain.RunStatusNeedsReview), string(domain.RunStatusParked),
		int32(eventdomain.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC),
		int32(eventdomain.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE),
	}, nil
}

func effortOrchestratorAdmissionGuard(runID string, effortIDs []string) (string, []any, error) {
	if len(effortIDs) == 0 {
		return "", nil, nil
	}
	args, err := effortOrchestratorOwnerArgs(effortIDs)
	if err != nil {
		return "", nil, err
	}
	queryArgs := []any{args[0], runID}
	queryArgs = append(queryArgs, args[1:]...)
	return ` AND NOT EXISTS (` + effortOrchestratorOwnerSubquery("?") + `)`, queryArgs, nil
}

func effortOrchestratorUpdateGuard(effortIDs []string) (string, []any, error) {
	if len(effortIDs) == 0 {
		return "", nil, nil
	}
	args, err := effortOrchestratorOwnerArgs(effortIDs)
	if err != nil {
		return "", nil, err
	}
	args = append([]any{
		string(domain.RunStatusPending), string(domain.RunStatusStarting), string(domain.RunStatusRunning),
	}, args...)
	clause := ` AND (runs.status IN (?, ?, ?) OR NOT EXISTS (` + effortOrchestratorOwnerSubquery("runs.id") + `))`
	return clause, args, nil
}
