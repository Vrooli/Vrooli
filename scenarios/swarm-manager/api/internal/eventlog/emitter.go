package eventlog

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// Emitter provides typed methods for emitting events. All methods are
// fire-and-forget: errors are logged but never returned, so event logging
// cannot block or fail the calling mutation.
type Emitter struct {
	repo Repository
}

// NewEmitter creates an Emitter backed by the given repository.
func NewEmitter(repo Repository) *Emitter {
	return &Emitter{repo: repo}
}

// --- Backlog events ---

func (e *Emitter) EmitBacklogCreated(entityID, kind, status string, priority int, initiative, effort string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogCreated, BacklogCreatedPayload{
		Kind:       kind,
		Status:     status,
		Priority:   priority,
		Initiative: initiative,
		Effort:     effort,
	})
}

func (e *Emitter) EmitBacklogStatusChanged(entityID, from, to string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogStatusChanged, StatusChangePayload{From: from, To: to})
}

func (e *Emitter) EmitBacklogPriorityChanged(entityID string, from, to int) {
	e.emit(EntityBacklogItem, entityID, EventBacklogPriorityChanged, PriorityChangePayload{From: from, To: to})
}

func (e *Emitter) EmitBacklogEffortChanged(entityID, from, to string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogEffortChanged, EffortChangePayload{From: from, To: to})
}

func (e *Emitter) EmitBacklogDependencyAdded(entityID, target string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogDependencyAdded, DependencyPayload{Target: target})
}

func (e *Emitter) EmitBacklogDependencyRemoved(entityID, target string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogDependencyRemoved, DependencyPayload{Target: target})
}

func (e *Emitter) EmitBacklogInitiativeChanged(entityID, from, to string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogInitiativeChanged, InitiativeChangePayload{From: from, To: to})
}

func (e *Emitter) EmitBacklogBlocked(entityID, reason string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogBlocked, BlockPayload{Reason: reason})
}

func (e *Emitter) EmitBacklogUnblocked(entityID, reason string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogUnblocked, BlockPayload{Reason: reason})
}

func (e *Emitter) EmitBacklogArchived(entityID, previousStatus, archivedAt string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogArchived, ArchivePayload{
		PreviousStatus: previousStatus,
		ArchivedAt:     archivedAt,
	})
}

func (e *Emitter) EmitBacklogUnarchived(entityID, archivedAt string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogUnarchived, UnarchivePayload{
		ArchivedAt: archivedAt,
	})
}

func (e *Emitter) EmitBacklogDeleted(entityID string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogDeleted, nil)
}

// --- Execution events ---

func (e *Emitter) EmitExecutionCreated(execID, backlogKind, backlogName, mode string) {
	e.emit(EntityExecution, execID, EventExecutionCreated, ExecutionCreatedPayload{
		BacklogKind: backlogKind,
		BacklogName: backlogName,
		Mode:        mode,
	})
}

func (e *Emitter) EmitExecutionStatusChanged(execID, from, to string) {
	e.emit(EntityExecution, execID, EventExecutionStatusChanged, StatusChangePayload{From: from, To: to})
}

func (e *Emitter) EmitExecutionCompleted(execID string, durationSecs float64, hadFixups bool) {
	e.emit(EntityExecution, execID, EventExecutionCompleted, ExecutionCompletedPayload{
		DurationSeconds: durationSecs,
		HadFixups:       hadFixups,
	})
}

func (e *Emitter) EmitExecutionFailed(execID, reason string, durationSecs float64) {
	e.emit(EntityExecution, execID, EventExecutionFailed, ExecutionFailedPayload{
		Reason:          reason,
		DurationSeconds: durationSecs,
	})
}

func (e *Emitter) EmitExecutionCanceled(execID, reason string) {
	e.emit(EntityExecution, execID, EventExecutionCanceled, ExecutionCanceledPayload{Reason: reason})
}

func (e *Emitter) EmitExecutionManuallyAccepted(execID, acceptedBy, reason, previousStatus string) {
	e.emit(EntityExecution, execID, EventExecutionManuallyAccepted, ExecutionManuallyAcceptedPayload{
		AcceptedBy:         acceptedBy,
		Reason:             reason,
		PreviousExecStatus: previousStatus,
	})
}

// --- Initiative events ---

func (e *Emitter) EmitInitiativeCreated(name string) {
	e.emit(EntityInitiative, name, EventInitiativeCreated, nil)
}

func (e *Emitter) EmitInitiativeItemAdded(name, item string) {
	e.emit(EntityInitiative, name, EventInitiativeItemAdded, InitiativeItemPayload{Item: item})
}

func (e *Emitter) EmitInitiativeItemRemoved(name, item string) {
	e.emit(EntityInitiative, name, EventInitiativeItemRemoved, InitiativeItemPayload{Item: item})
}

func (e *Emitter) EmitInitiativeStatusChanged(name, from, to string) {
	e.emit(EntityInitiative, name, EventInitiativeStatusChanged, StatusChangePayload{From: from, To: to})
}

func (e *Emitter) EmitInitiativeArchived(name, previousStatus, archivedAt string) {
	e.emit(EntityInitiative, name, EventInitiativeArchived, ArchivePayload{
		PreviousStatus: previousStatus,
		ArchivedAt:     archivedAt,
	})
}

func (e *Emitter) EmitInitiativeUnarchived(name, archivedAt string) {
	e.emit(EntityInitiative, name, EventInitiativeUnarchived, UnarchivePayload{
		ArchivedAt: archivedAt,
	})
}

// --- Queue events ---

func (e *Emitter) EmitQueued(backlogKind, backlogName string, position int) {
	e.emit(EntityQueue, backlogKind+"/"+backlogName, EventQueued, QueuePayload{
		BacklogKind: backlogKind,
		BacklogName: backlogName,
		Position:    position,
	})
}

func (e *Emitter) EmitDequeued(backlogKind, backlogName, reason string) {
	e.emit(EntityQueue, backlogKind+"/"+backlogName, EventDequeued, QueuePayload{
		BacklogKind: backlogKind,
		BacklogName: backlogName,
		Reason:      reason,
	})
}

// --- System/migration events ---

// EmitMigrationApplied records that a one-time migration has finished. Callers
// should check for an existing sentinel (via Repository.All or similar) before
// running the migration body to avoid re-applying.
func (e *Emitter) EmitMigrationApplied(name, description string, affectedIDs int) {
	e.emit(EntitySystem, name, EventSystemMigrationApplied, MigrationAppliedPayload{
		Name:        name,
		Description: description,
		AffectedIDs: affectedIDs,
	})
}

// --- Decision/workshop events ---

func (e *Emitter) EmitWorkshopRoundCompleted(entityID string, roundNumber int) {
	e.emit(EntityBacklogItem, entityID, EventWorkshopRoundCompleted, WorkshopRoundPayload{
		RoundNumber: roundNumber,
	})
}

// --- View events ---

func (e *Emitter) EmitBacklogViewed(entityID, kind string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogViewed, ViewPayload{Kind: kind})
}

func (e *Emitter) EmitExecutionViewed(execID string) {
	e.emit(EntityExecution, execID, EventExecutionViewed, nil)
}

func (e *Emitter) EmitInitiativeViewed(name string) {
	e.emit(EntityInitiative, name, EventInitiativeViewed, nil)
}

func (e *Emitter) EmitCaptureViewed(captureID string) {
	e.emit(EntityCapture, captureID, EventCaptureViewed, nil)
}

func (e *Emitter) EmitClarificationStarted(entityID string, roundNumber int, itemID string, hasMessage bool) {
	e.emit(EntityBacklogItem, entityID, EventClarificationStarted, ClarificationStartedPayload{
		RoundNumber: roundNumber,
		ItemID:      itemID,
		HasMessage:  hasMessage,
	})
}

func (e *Emitter) EmitClarificationResolved(entityID string, roundNumber int, itemID string, messageCount int, impactLevel string) {
	e.emit(EntityBacklogItem, entityID, EventClarificationResolved, ClarificationResolvedPayload{
		RoundNumber:  roundNumber,
		ItemID:       itemID,
		MessageCount: messageCount,
		ImpactLevel:  impactLevel,
	})
}

func (e *Emitter) EmitClarificationAction(entityID string, roundNumber int, itemID string, action string) {
	e.emit(EntityBacklogItem, entityID, EventClarificationAction, ClarificationActionPayload{
		RoundNumber: roundNumber,
		ItemID:      itemID,
		Action:      action,
	})
}

// --- Review evidence events ---

func (e *Emitter) EmitReviewStarted(executionID string, roundNumber int) {
	e.emit(EntityExecution, executionID, EventReviewStarted, ReviewStartedPayload{
		ExecutionID: executionID,
		RoundNumber: roundNumber,
	})
}

func (e *Emitter) EmitReviewEvidenceAdded(executionID, evidenceID, evidenceType string) {
	e.emit(EntityExecution, executionID, EventReviewEvidenceAdded, ReviewEvidencePayload{
		ExecutionID:  executionID,
		EvidenceID:   evidenceID,
		EvidenceType: evidenceType,
	})
}

func (e *Emitter) EmitReviewEvidenceVerified(executionID, evidenceID string) {
	e.emit(EntityExecution, executionID, EventReviewEvidenceVerified, ReviewVerifiedPayload{
		ExecutionID: executionID,
		EvidenceID:  evidenceID,
	})
}

func (e *Emitter) EmitReviewRequestCreated(executionID, requestID, description string) {
	e.emit(EntityExecution, executionID, EventReviewRequestCreated, ReviewRequestPayload{
		ExecutionID: executionID,
		RequestID:   requestID,
		Description: description,
	})
}

func (e *Emitter) EmitReviewRequestFulfilled(executionID, requestID, evidenceID string) {
	e.emit(EntityExecution, executionID, EventReviewRequestFulfilled, ReviewRequestPayload{
		ExecutionID: executionID,
		RequestID:   requestID,
		EvidenceID:  evidenceID,
	})
}

func (e *Emitter) EmitReviewRoundCompleted(executionID string, roundNumber, evidenceCount int, classification string, durationSecs float64) {
	e.emit(EntityExecution, executionID, EventReviewRoundCompleted, ReviewRoundCompletedPayload{
		ExecutionID:    executionID,
		RoundNumber:    roundNumber,
		EvidenceCount:  evidenceCount,
		Classification: classification,
		DurationSecs:   durationSecs,
	})
}

func (e *Emitter) EmitReviewFailed(executionID, reason string, durationSecs float64) {
	e.emit(EntityExecution, executionID, EventReviewFailed, ReviewFailedPayload{
		ExecutionID:  executionID,
		Reason:       reason,
		DurationSecs: durationSecs,
	})
}

// emit is the internal helper that marshals metadata and appends the event.
func (e *Emitter) emit(entityType EntityType, entityID string, eventType EventType, payload any) {
	var metadata json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			slog.Error("failed to marshal payload", "event_type", eventType, "error", err)
			return
		}
		metadata = data
	}

	event := Event{
		Timestamp:  time.Now().UTC(),
		EntityType: entityType,
		EntityID:   entityID,
		EventType:  eventType,
		ActorType:  "user",
		Metadata:   metadata,
	}

	if _, err := e.repo.Append(context.Background(), event); err != nil {
		slog.Error("failed to append event", "event_type", eventType, "entity_type", entityType, "entity_id", entityID, "error", err)
	}
}
