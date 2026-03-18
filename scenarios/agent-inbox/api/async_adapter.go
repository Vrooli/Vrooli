package main

import (
	"context"
	"encoding/json"
	"time"

	"agent-inbox/persistence"
	"agent-inbox/services"
)

// =============================================================================
// Async Repository Adapter
// =============================================================================

// asyncRepoAdapter bridges persistence.Repository to services.AsyncOperationRepository.
// It converts between persistence and services record types.
type asyncRepoAdapter struct {
	repo *persistence.Repository
}

func newAsyncRepoAdapter(repo *persistence.Repository) *asyncRepoAdapter {
	return &asyncRepoAdapter{repo: repo}
}

// Convert persistence record to services record
func toServiceRecord(rec *persistence.AsyncOperationRecord) *services.AsyncOperationRecord {
	if rec == nil {
		return nil
	}
	return &services.AsyncOperationRecord{
		ToolCallID:    rec.ToolCallID,
		ChatID:        rec.ChatID,
		ToolName:      rec.ToolName,
		ScenarioName:  rec.ScenarioName,
		OperationID:   rec.OperationID,
		Status:        rec.Status,
		Progress:      rec.Progress,
		Message:       rec.Message,
		Phase:         rec.Phase,
		Result:        json.RawMessage(rec.Result),
		Error:         rec.Error,
		AsyncBehavior: json.RawMessage(rec.AsyncBehavior),
		StartedAt:     rec.StartedAt,
		UpdatedAt:     rec.UpdatedAt,
		CompletedAt:   rec.CompletedAt,
	}
}

// Convert services record to persistence record
func toPersistenceRecord(rec *services.AsyncOperationRecord) *persistence.AsyncOperationRecord {
	if rec == nil {
		return nil
	}
	return &persistence.AsyncOperationRecord{
		ToolCallID:    rec.ToolCallID,
		ChatID:        rec.ChatID,
		ToolName:      rec.ToolName,
		ScenarioName:  rec.ScenarioName,
		OperationID:   rec.OperationID,
		Status:        rec.Status,
		Progress:      rec.Progress,
		Message:       rec.Message,
		Phase:         rec.Phase,
		Result:        json.RawMessage(rec.Result),
		Error:         rec.Error,
		AsyncBehavior: json.RawMessage(rec.AsyncBehavior),
		StartedAt:     rec.StartedAt,
		UpdatedAt:     rec.UpdatedAt,
		CompletedAt:   rec.CompletedAt,
	}
}

// Convert persistence event to services event
func toServiceEvent(ev *persistence.AsyncCompletionEventRecord) *services.AsyncCompletionEventRecord {
	if ev == nil {
		return nil
	}
	return &services.AsyncCompletionEventRecord{
		ID:         ev.ID,
		ChatID:     ev.ChatID,
		ToolCallID: ev.ToolCallID,
		ToolName:   ev.ToolName,
		Status:     ev.Status,
		Result:     json.RawMessage(ev.Result),
		Error:      ev.Error,
		CreatedAt:  ev.CreatedAt,
	}
}

// Convert services event to persistence event
func toPersistenceEvent(ev *services.AsyncCompletionEventRecord) *persistence.AsyncCompletionEventRecord {
	if ev == nil {
		return nil
	}
	return &persistence.AsyncCompletionEventRecord{
		ID:         ev.ID,
		ChatID:     ev.ChatID,
		ToolCallID: ev.ToolCallID,
		ToolName:   ev.ToolName,
		Status:     ev.Status,
		Result:     json.RawMessage(ev.Result),
		Error:      ev.Error,
		CreatedAt:  ev.CreatedAt,
	}
}

// Implement services.AsyncOperationRepository interface

func (a *asyncRepoAdapter) CreateAsyncOperation(ctx context.Context, op *services.AsyncOperationRecord) error {
	return a.repo.CreateAsyncOperation(ctx, toPersistenceRecord(op))
}

func (a *asyncRepoAdapter) UpdateAsyncOperation(ctx context.Context, op *services.AsyncOperationRecord) error {
	return a.repo.UpdateAsyncOperation(ctx, toPersistenceRecord(op))
}

func (a *asyncRepoAdapter) GetAsyncOperationByToolCallID(ctx context.Context, toolCallID string) (*services.AsyncOperationRecord, error) {
	rec, err := a.repo.GetAsyncOperationByToolCallID(ctx, toolCallID)
	if err != nil {
		return nil, err
	}
	return toServiceRecord(rec), nil
}

func (a *asyncRepoAdapter) GetActiveAsyncOperationsByChatID(ctx context.Context, chatID string) ([]*services.AsyncOperationRecord, error) {
	recs, err := a.repo.GetActiveAsyncOperationsByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	result := make([]*services.AsyncOperationRecord, len(recs))
	for i, rec := range recs {
		result[i] = toServiceRecord(rec)
	}
	return result, nil
}

func (a *asyncRepoAdapter) GetAllActiveAsyncOperations(ctx context.Context) ([]*services.AsyncOperationRecord, error) {
	recs, err := a.repo.GetAllActiveAsyncOperations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*services.AsyncOperationRecord, len(recs))
	for i, rec := range recs {
		result[i] = toServiceRecord(rec)
	}
	return result, nil
}

func (a *asyncRepoAdapter) DeleteAsyncOperation(ctx context.Context, toolCallID string) error {
	return a.repo.DeleteAsyncOperation(ctx, toolCallID)
}

func (a *asyncRepoAdapter) CleanupCompletedAsyncOperations(ctx context.Context, olderThan time.Duration) (int64, error) {
	return a.repo.CleanupCompletedAsyncOperations(ctx, olderThan)
}

func (a *asyncRepoAdapter) UpdateAsyncOperationStatus(ctx context.Context, toolCallID, status string) error {
	return a.repo.UpdateAsyncOperationStatus(ctx, toolCallID, status)
}

func (a *asyncRepoAdapter) GetCompletedAsyncOperationsByChatID(ctx context.Context, chatID string, limit, offset int) ([]*services.AsyncOperationRecord, int, error) {
	recs, total, err := a.repo.GetCompletedAsyncOperationsByChatID(ctx, chatID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*services.AsyncOperationRecord, len(recs))
	for i, rec := range recs {
		result[i] = toServiceRecord(rec)
	}
	return result, total, nil
}

func (a *asyncRepoAdapter) CreateCompletionEvent(ctx context.Context, event *services.AsyncCompletionEventRecord) error {
	return a.repo.CreateCompletionEvent(ctx, toPersistenceEvent(event))
}

func (a *asyncRepoAdapter) GetCompletionEventsSince(ctx context.Context, chatID string, since time.Time) ([]*services.AsyncCompletionEventRecord, error) {
	events, err := a.repo.GetCompletionEventsSince(ctx, chatID, since)
	if err != nil {
		return nil, err
	}
	result := make([]*services.AsyncCompletionEventRecord, len(events))
	for i, ev := range events {
		result[i] = toServiceEvent(ev)
	}
	return result, nil
}

func (a *asyncRepoAdapter) CleanupOldCompletionEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	return a.repo.CleanupOldCompletionEvents(ctx, olderThan)
}
