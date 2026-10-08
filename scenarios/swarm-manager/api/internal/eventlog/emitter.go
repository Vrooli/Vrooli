package eventlog

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/vrooli/api-core/provenance"
)

// Emitter provides typed methods for emitting events. All methods are
// fire-and-forget: errors are logged but never returned, so event logging
// cannot block or fail the calling mutation. Production writes still go
// through AppendAttributed, which rejects an event with no actor identity.
type Emitter struct {
	repo Repository
}

// NewEmitter creates an Emitter backed by the given repository.
func NewEmitter(repo Repository) *Emitter {
	return &Emitter{repo: repo}
}

// --- Backlog events ---

func (e *Emitter) EmitBacklogCreated(entityID, kind, status string, priority int, initiative, effort string) {
	// This compatibility entrypoint predates request provenance. Keep its
	// origin explicit as a system writer; request-aware callers must use the
	// context/source variants so the operator or agent identity is preserved.
	e.emitWithActor(EntityBacklogItem, entityID, EventBacklogCreated, "system", systemEmitterActor, BacklogCreatedPayload{
		Kind:      kind,
		Status:    status,
		Priority:  priority,
		Milestone: initiative,
		Effort:    effort,
	})
}

// EmitBacklogCreatedFromSource records a creation with explicit actor
// attribution. Use this from the unified backlog.Service so the actor
// reflects the originating surface (HTTP user, batch, feedback round,
// review round) instead of the default "user".
func (e *Emitter) EmitBacklogCreatedFromSource(entityID, kind, status string, priority int, initiative, effort, actorType, actorID string) {
	if actorType == "" {
		actorType = "user"
	}
	e.emitWithActor(EntityBacklogItem, entityID, EventBacklogCreated, actorType, actorID, BacklogCreatedPayload{
		Kind:      kind,
		Status:    status,
		Priority:  priority,
		Milestone: initiative,
		Effort:    effort,
	})
}

// EmitBacklogStatusChanged records a status transition. This is a durability
// pushback signal, so it takes the request context: an item bounced back to
// failed or needs_followup is only attributable if the row records who did it.
func (e *Emitter) EmitBacklogStatusChanged(ctx context.Context, entityID, from, to string) {
	e.emitContext(ctx, EntityBacklogItem, entityID, EventBacklogStatusChanged, StatusChangePayload{From: from, To: to})
}

func (e *Emitter) EmitBacklogStatusChangedFromSource(ctx context.Context, entityID, from, to string, source BacklogMutationSourcePayload, itemRefs []string) {
	payload := StatusChangePayload{
		From:     from,
		To:       to,
		Source:   &source,
		ItemRefs: append([]string(nil), itemRefs...),
	}
	actorType := "user"
	actorID := ""
	if source.Mode != "" || source.Entrypoint != "" {
		actorType = "operating_mode"
		actorID = source.InitiativeName
		if source.RunID != "" {
			actorID = source.RunID
		}
	}
	e.emitWithActorContext(ctx, EntityBacklogItem, entityID, EventBacklogStatusChanged, actorType, actorID, payload)
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

// EmitBacklogMilestoneChanged records a goal-owned milestone reference
// change. The initiative method above is retained solely for replaying older
// event streams.
func (e *Emitter) EmitBacklogMilestoneChanged(entityID, from, to string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogMilestoneChanged, MilestoneChangePayload{From: from, To: to})
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

// --- Goal events ---

func (e *Emitter) EmitGoalCreated(name string, payload GoalCreatedPayload) {
	e.emit(EntityGoal, name, EventGoalCreated, payload)
}

func (e *Emitter) EmitGoalUpdated(name string) {
	e.emit(EntityGoal, name, EventGoalUpdated, nil)
}

func (e *Emitter) EmitGoalTargetAdded(name, target string) {
	e.emit(EntityGoal, name, EventGoalTargetAdded, GoalTargetPayload{Target: target})
}

func (e *Emitter) EmitGoalTargetRemoved(name, target string) {
	e.emit(EntityGoal, name, EventGoalTargetRemoved, GoalTargetPayload{Target: target})
}

func (e *Emitter) EmitGoalPriorityChanged(name string, from, to int) {
	e.emit(EntityGoal, name, EventGoalPriorityChanged, PriorityChangePayload{From: from, To: to})
}

func (e *Emitter) EmitGoalArchived(name, previousStatus, archivedAt string) {
	e.emit(EntityGoal, name, EventGoalArchived, ArchivePayload{
		PreviousStatus: previousStatus,
		ArchivedAt:     archivedAt,
	})
}

func (e *Emitter) EmitGoalArchivedWithDisposition(name, previousStatus, archivedAt, actor string, droppedItems []string) {
	e.emit(EntityGoal, name, EventGoalArchived, ArchivePayload{PreviousStatus: previousStatus, ArchivedAt: archivedAt, Actor: actor, DroppedItems: droppedItems})
}

func (e *Emitter) EmitGoalUnarchived(name, archivedAt string) {
	e.emit(EntityGoal, name, EventGoalUnarchived, UnarchivePayload{ArchivedAt: archivedAt})
}

func (e *Emitter) EmitGoalScopeSnapshot(name string, payload GoalScopeSnapshotPayload) {
	e.emit(EntityGoal, name, EventGoalScopeSnapshot, payload)
}

func (e *Emitter) EmitMilestoneCreated(goal, milestone string, payload MilestonePayload) {
	e.emit(EntityGoal, goal+"/"+milestone, EventMilestoneCreated, payload)
}

func (e *Emitter) EmitMilestoneUpdated(goal, milestone string, payload MilestonePayload) {
	e.emit(EntityGoal, goal+"/"+milestone, EventMilestoneUpdated, payload)
}

func (e *Emitter) EmitMilestoneItemsAssigned(goal, milestone string, payload MilestonePayload) {
	e.emit(EntityGoal, goal+"/"+milestone, EventMilestoneItemsAssigned, payload)
}

func (e *Emitter) EmitMilestoneItemsUnassigned(goal, milestone string, payload MilestonePayload) {
	e.emit(EntityGoal, goal+"/"+milestone, EventMilestoneItemsUnassigned, payload)
}

func (e *Emitter) EmitMilestoneArchived(goal, milestone string, payload MilestonePayload) {
	e.emit(EntityGoal, goal+"/"+milestone, EventMilestoneArchived, payload)
}

// --- Calibration events ---

// EmitBacklogDurationSample records one coarse lead-time observation for a
// completed backlog item. itemRef is "<kind>/<name>". Consumed by the ETA
// engine to build per-effort-class duration distributions.
func (e *Emitter) EmitBacklogDurationSample(itemRef string, payload DurationSamplePayload) {
	e.emit(EntityBacklogItem, itemRef, EventBacklogDurationSample, payload)
}

// --- Record events ---

// EmitRecordCreated records that a new records.Record was persisted. Actor is
// "user" by default; pass actorType="agent" or similar for non-human authors.
// The stats engine consumes both this and EmitRecordSuperseded to compute
// records-per-window and regression-rate.
func (e *Emitter) EmitRecordCreated(recordID, kind, scenario, backlogRef string, stub bool) {
	e.emit(EntityRecord, recordID, EventRecordCreated, RecordCreatedPayload{
		Kind:       kind,
		Scenario:   scenario,
		BacklogRef: backlogRef,
		Stub:       stub,
	})
}

// EmitRecordSuperseded records that a record was superseded by another. The
// successor record's id goes in Event.EntityID; the predecessor goes in the
// payload, giving stats consumers a directional link without joining tables.
func (e *Emitter) EmitRecordSuperseded(ctx context.Context, successorID, supersededID, reason string) {
	e.emitContext(ctx, EntityRecord, successorID, EventRecordSuperseded, RecordSupersededPayload{
		SupersededID: supersededID,
		Reason:       reason,
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

// --- Agent session events ---

func (e *Emitter) EmitAgentSessionCreated(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionCreated, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionStarted(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionStarted, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionContinued(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionContinued, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionCompleted(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionCompleted, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionFailed(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionFailed, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionCanceled(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionCanceled, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionDeleted(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionDeleted, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionProposalCreated(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionProposalCreated, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionProposalApplied(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionProposalApplied, "agent_session", sessionID, payload)
}

func (e *Emitter) EmitAgentSessionArtifactLinked(sessionID string, payload any) {
	e.emitWithActor(EntityAgentSession, sessionID, EventAgentSessionArtifactLinked, "agent_session", sessionID, payload)
}

// --- Decision/workshop events ---

func (e *Emitter) EmitWorkshopRoundCompleted(entityID string, payload WorkshopRoundPayload) {
	e.emit(EntityBacklogItem, entityID, EventWorkshopRoundCompleted, payload)
}

func (e *Emitter) EmitAutonomyGateModeChanged(gateID, from, to, actorID string) {
	e.emitWithActor(EntitySystem, gateID, EventAutonomyGateModeChanged, "operator", actorID, AutonomyGateModeChangedPayload{GateID: gateID, From: from, To: to})
}

// --- View events ---

func (e *Emitter) EmitBacklogViewed(entityID, kind string) {
	e.emit(EntityBacklogItem, entityID, EventBacklogViewed, ViewPayload{Kind: kind})
}

func (e *Emitter) EmitExecutionViewed(execID string) {
	e.emit(EntityExecution, execID, EventExecutionViewed, nil)
}

func (e *Emitter) EmitCaptureViewed(captureID string) {
	e.emit(EntityCapture, captureID, EventCaptureViewed, nil)
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

func (e *Emitter) EmitReviewFailed(ctx context.Context, executionID, reason string, durationSecs float64) {
	e.emitContext(ctx, EntityExecution, executionID, EventReviewFailed, ReviewFailedPayload{
		ExecutionID:  executionID,
		Reason:       reason,
		DurationSecs: durationSecs,
	})
}

// EmitBacklogProposalApplied records that a single proposal mutation
// landed on a backlog item. Actor is set from the originating surface
// (feedback round or milestone review) so consumers can group changes
// by the round that caused them.
func (e *Emitter) EmitBacklogProposalApplied(entityID string, payload ProposalAppliedPayload) {
	actorType, actorID := proposalActor(payload)
	e.emitWithActor(EntityBacklogItem, entityID, EventBacklogProposalApplied, actorType, actorID, payload)
}

// proposalActor picks the durable actor identity for a proposal mutation.
// Review rounds take precedence when both IDs are present (review-applied
// follow-ups are attributed to the review, not to any feedback that
// preceded it).
func proposalActor(p ProposalAppliedPayload) (actorType, actorID string) {
	switch {
	case p.ReviewRoundID != "":
		return "milestone_review", p.ReviewRoundID
	case p.FeedbackRoundID != "":
		return "feedback_round", p.FeedbackRoundID
	default:
		return "proposal", p.InitiativeName
	}
}

// emit is the internal helper that marshals metadata and appends the event
// with the default "user" actor.
//
// It carries no request context, so provenance cannot be resolved and the row
// records verification_status=absent. Any event type read by a durability or
// attribution consumer must use emitContext instead — see
// docs/internal/PROVENANCE.md § Event seams.
func (e *Emitter) emit(entityType EntityType, entityID string, eventType EventType, payload any) {
	// Compatibility methods without a context identify the actual writer as
	// this Swarm Manager system process. They must not manufacture a user or
	// unattributed actor. New mutation paths should use emitContext.
	e.emitWithActor(entityType, entityID, eventType, "system", systemEmitterActor, payload)
}

const systemEmitterActor = "system/swarm-manager/compatibility-emitter"

// emitContext is the request-aware counterpart to emit. The caller's context
// carries the verified provenance resolved by api-core's middleware, so the
// appended row can record who actually performed the mutation.
func (e *Emitter) emitContext(ctx context.Context, entityType EntityType, entityID string, eventType EventType, payload any) {
	e.emitWithActorContext(ctx, entityType, entityID, eventType, "user", "", payload)
}

func (e *Emitter) emitWithActor(entityType EntityType, entityID string, eventType EventType, actorType, actorID string, payload any) {
	e.emitWithActorContext(context.Background(), entityType, entityID, eventType, actorType, actorID, payload)
}

// EmitBacklogCreatedFromContext is the request-aware seam used by the
// backlog mutation path. It keeps the legacy typed emitter API intact while
// allowing the shared verifier result to reach the append-only event row.
func (e *Emitter) EmitBacklogCreatedFromContext(ctx context.Context, entityID, kind, status string, priority int, milestone, effort, actorType, actorID string) {
	e.emitWithActorContext(ctx, EntityBacklogItem, entityID, EventBacklogCreated, actorType, actorID, BacklogCreatedPayload{
		Kind: kind, Status: status, Priority: priority, Milestone: milestone, Effort: effort,
	})
}

func (e *Emitter) emitWithActorContext(ctx context.Context, entityType EntityType, entityID string, eventType EventType, actorType, actorID string, payload any) {
	prov := provenance.FromContext(ctx)
	verifiedActorID, verifiedActorType, _, status, verifiedRunID, _ := prov.WriteFields()
	sessionID, sessionKind := prov.ObservationFields()
	if prov.IsVerifiedAgent() {
		actorType, actorID = verifiedActorType, verifiedActorID
	}
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
		Timestamp:          time.Now().UTC(),
		EntityType:         entityType,
		EntityID:           entityID,
		EventType:          eventType,
		ActorType:          actorType,
		ActorID:            actorID,
		RunID:              verifiedRunID,
		VerificationStatus: status,
		HarnessSessionID:   sessionID,
		HarnessKind:        sessionKind,
		Metadata:           metadata,
	}

	if attributed, ok := e.repo.(interface {
		AppendAttributed(context.Context, Event) (int64, error)
	}); ok {
		if _, err := attributed.AppendAttributed(context.Background(), event); err != nil {
			slog.Error("failed to append event", "event_type", eventType, "entity_type", entityType, "entity_id", entityID, "error", err)
		}
		return
	}
	if _, err := e.repo.Append(context.Background(), event); err != nil {
		slog.Error("failed to append event", "event_type", eventType, "entity_type", entityType, "entity_id", entityID, "error", err)
	}
}
