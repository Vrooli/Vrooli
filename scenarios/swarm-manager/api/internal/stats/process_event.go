package stats

import (
	"encoding/json"
	"log/slog"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/workshop"
)

func (s *aggregateState) processEvent(e *eventlog.Event) {
	s.totalEvents++
	if !s.earliestEventRecorded || e.Timestamp.Before(s.earliestEventAt) {
		s.earliestEventAt = e.Timestamp
		s.earliestEventRecorded = true
	}

	switch e.EventType {
	// --- Backlog ---
	case eventlog.EventBacklogCreated:
		s.handleBacklogCreated(e)
	case eventlog.EventBacklogStatusChanged:
		s.handleBacklogStatusChanged(e)
	case eventlog.EventBacklogArchived:
		s.handleBacklogArchived(e)
	case eventlog.EventBacklogUnarchived:
		// Restore item to active backlog using whatever status we have recorded.
		s.currentBacklog[e.EntityID] = true
	case eventlog.EventBacklogDeleted:
		delete(s.currentBacklog, e.EntityID)
		delete(s.itemStatus, e.EntityID)
	case eventlog.EventBacklogBlocked:
		s.handleBacklogBlocked(e)
	case eventlog.EventBacklogUnblocked:
		s.handleBacklogUnblocked(e)
	case eventlog.EventBacklogInitiativeChanged:
		s.handleBacklogInitiativeChanged(e)

	// --- Initiative ---
	case eventlog.EventInitiativeCreated:
		s.handleInitiativeCreated(e)
	case eventlog.EventInitiativeItemAdded:
		s.handleInitiativeItemAdded(e)
	case eventlog.EventInitiativeItemRemoved:
		s.handleInitiativeItemRemoved(e)
	case eventlog.EventInitiativeModeChanged:
		s.handleInitiativeModeChanged(e)
	case eventlog.EventOperatingModePhaseStarted:
		s.handleOperatingModePhaseStarted(e)
	case eventlog.EventOperatingModePhaseCompleted:
		s.handleOperatingModePhaseTerminal(e, "completed")
	case eventlog.EventOperatingModePhaseFailed:
		s.handleOperatingModePhaseTerminal(e, "failed")
	case eventlog.EventOperatingModePhaseCanceled:
		s.handleOperatingModePhaseTerminal(e, "canceled")
	case eventlog.EventOperatingModeBacklogSynced:
		s.handleOperatingModeBacklogSynced(e)

	// --- Execution ---
	case eventlog.EventExecutionCreated:
		s.execTotal++
	case eventlog.EventExecutionCompleted:
		s.handleExecutionCompleted(e)
	case eventlog.EventExecutionFailed:
		s.handleExecutionFailed(e)
	case eventlog.EventExecutionCanceled:
		s.execOutcome[e.EntityID] = "canceled"
	case eventlog.EventExecutionManuallyAccepted:
		// Manual acceptance is emitted in addition to execution.completed, so
		// the completion itself will also be recorded. Marking the outcome
		// here (and preserving it under EventExecutionCompleted) guarantees
		// a manually-accepted run overrides any earlier "failed" outcome,
		// without double-counting.
		s.execOutcome[e.EntityID] = "manually_accepted"

	// --- Native Agent Sessions ---
	case eventlog.EventAgentSessionCreated:
		s.handleAgentSessionCreated(e)
	case eventlog.EventAgentSessionStarted, eventlog.EventAgentSessionContinued, eventlog.EventAgentSessionCompleted,
		eventlog.EventAgentSessionFailed, eventlog.EventAgentSessionCanceled:
		s.handleAgentSessionLifecycle(e)
	case eventlog.EventAgentSessionProposalCreated:
		s.handleAgentSessionProposalCreated(e)
	case eventlog.EventAgentSessionProposalApplied:
		s.handleAgentSessionProposalApplied(e)
	case eventlog.EventAgentSessionArtifactLinked:
		s.handleAgentSessionArtifactLinked(e)

	// --- Workshop ---
	case eventlog.EventWorkshopRoundCompleted:
		s.handleWorkshopRoundCompleted(e)

	// --- Review evidence ---
	case eventlog.EventReviewRoundCompleted:
		s.handleReviewRoundCompleted(e)
	case eventlog.EventReviewEvidenceVerified:
		s.reviewEvidenceVerified++
	case eventlog.EventReviewRequestCreated:
		s.reviewRequestsCreated++

	// --- Records ---
	case eventlog.EventRecordCreated:
		s.handleRecordCreated(e)
	case eventlog.EventRecordSuperseded:
		s.recordsSupersedeCount++
	}
}

// --- Backlog handlers ---

func (s *aggregateState) handleBacklogCreated(e *eventlog.Event) {
	s.createdEvents = append(s.createdEvents, e.Timestamp)
	s.currentBacklog[e.EntityID] = true
	s.createdAt[e.EntityID] = e.Timestamp
	s.itemStatus[e.EntityID] = "backlog"

	var p eventlog.BacklogCreatedPayload
	if unmarshalMeta(e.Metadata, &p) {
		s.itemStatus[e.EntityID] = p.Status
		if p.Initiative != "" {
			if s.initiativeItems[p.Initiative] == nil {
				s.initiativeItems[p.Initiative] = make(map[string]bool)
			}
			s.initiativeItems[p.Initiative][e.EntityID] = true
		}
	}
}

func (s *aggregateState) handleBacklogStatusChanged(e *eventlog.Event) {
	var p eventlog.StatusChangePayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	s.itemStatus[e.EntityID] = p.To

	if p.To == "in_progress" {
		s.inProgressAt[e.EntityID] = e.Timestamp
	}
	if p.To == "queued" {
		s.queuedAt[e.EntityID] = e.Timestamp
	}
	if p.To == "in_progress" {
		if qt, ok := s.queuedAt[e.EntityID]; ok {
			s.queueWaitH = append(s.queueWaitH, e.Timestamp.Sub(qt).Hours())
			delete(s.queuedAt, e.EntityID)
		}
	}
	if p.To == "completed" {
		s.completedEvents = append(s.completedEvents, e.Timestamp)
		s.completedAllTime++
		delete(s.currentBacklog, e.EntityID)

		if start, ok := s.inProgressAt[e.EntityID]; ok {
			s.cycleTimesH = append(s.cycleTimesH, e.Timestamp.Sub(start).Hours())
			delete(s.inProgressAt, e.EntityID)
		}
		if created, ok := s.createdAt[e.EntityID]; ok {
			s.leadTimesH = append(s.leadTimesH, e.Timestamp.Sub(created).Hours())
		}
	}
}

func (s *aggregateState) handleBacklogArchived(e *eventlog.Event) {
	delete(s.currentBacklog, e.EntityID)
	var p eventlog.ArchivePayload
	if unmarshalMeta(e.Metadata, &p) && p.PreviousStatus != "" {
		s.itemStatus[e.EntityID] = p.PreviousStatus
	} else {
		// Historical events before the migration may have nil metadata.
		s.itemStatus[e.EntityID] = "archived"
	}
}

func (s *aggregateState) handleBacklogBlocked(e *eventlog.Event) {
	s.blockedItems[e.EntityID] = e.Timestamp
	var p eventlog.BlockPayload
	if unmarshalMeta(e.Metadata, &p) && p.Reason != "" {
		s.blockReasons[p.Reason]++
	}
}

func (s *aggregateState) handleBacklogUnblocked(e *eventlog.Event) {
	if blockedAt, ok := s.blockedItems[e.EntityID]; ok {
		s.blockDurations = append(s.blockDurations, e.Timestamp.Sub(blockedAt).Hours())
		delete(s.blockedItems, e.EntityID)
	}
}

func (s *aggregateState) handleBacklogInitiativeChanged(e *eventlog.Event) {
	var p eventlog.InitiativeChangePayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	if p.From != "" {
		if items := s.initiativeItems[p.From]; items != nil {
			delete(items, e.EntityID)
		}
	}
	if p.To != "" {
		if s.initiativeItems[p.To] == nil {
			s.initiativeItems[p.To] = make(map[string]bool)
		}
		s.initiativeItems[p.To][e.EntityID] = true
	}
}

// --- Initiative handlers ---

func (s *aggregateState) handleInitiativeCreated(e *eventlog.Event) {
	s.initiativeCreated[e.EntityID] = true
	if s.initiativeMode[e.EntityID] == "" {
		s.initiativeMode[e.EntityID] = "item-level"
	}
	if s.initiativeItems[e.EntityID] == nil {
		s.initiativeItems[e.EntityID] = make(map[string]bool)
	}
}

func (s *aggregateState) handleInitiativeItemAdded(e *eventlog.Event) {
	var p eventlog.InitiativeItemPayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	if s.initiativeItems[e.EntityID] == nil {
		s.initiativeItems[e.EntityID] = make(map[string]bool)
	}
	s.initiativeItems[e.EntityID][p.Item] = true
	// Track initial count: if this is the first time, record it.
	if _, exists := s.initiativeInitial[e.EntityID]; !exists {
		s.initiativeInitial[e.EntityID] = 0
	}
}

func (s *aggregateState) handleInitiativeItemRemoved(e *eventlog.Event) {
	var p eventlog.InitiativeItemPayload
	if unmarshalMeta(e.Metadata, &p) {
		if items := s.initiativeItems[e.EntityID]; items != nil {
			delete(items, p.Item)
		}
	}
}

func (s *aggregateState) handleInitiativeModeChanged(e *eventlog.Event) {
	var p eventlog.InitiativeModeChangePayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	s.modeSwitchCount++
	if p.To != "" {
		s.initiativeMode[e.EntityID] = p.To
	}
}

func (s *aggregateState) handleOperatingModePhaseStarted(e *eventlog.Event) {
	var p eventlog.OperatingModePhasePayload
	if unmarshalMeta(e.Metadata, &p) {
		s.recordModePhaseStarted(p)
	}
}

func (s *aggregateState) handleOperatingModePhaseTerminal(e *eventlog.Event, status string) {
	var p eventlog.OperatingModePhasePayload
	if unmarshalMeta(e.Metadata, &p) {
		s.recordModePhaseTerminal(p, status)
	}
}

func (s *aggregateState) handleOperatingModeBacklogSynced(e *eventlog.Event) {
	var p eventlog.OperatingModeBacklogSyncPayload
	if unmarshalMeta(e.Metadata, &p) && p.Mode != "" {
		bucket := s.modeBacklogSync[p.Mode]
		if bucket == nil {
			bucket = &BacklogSyncStats{}
			s.modeBacklogSync[p.Mode] = bucket
		}
		bucket.Events++
		bucket.ItemsCompleted += p.BacklogItemsCompleted
		bucket.ItemsCreated += p.BacklogItemsCreated
		bucket.ItemsUpdated += p.BacklogItemsUpdated
	}
}

// --- Execution handlers ---

func (s *aggregateState) handleExecutionCompleted(e *eventlog.Event) {
	// Preserve a manually_accepted marker so a failed-then-accepted run
	// stays categorized as a manual acceptance rather than being
	// demoted back to plain completed.
	if s.execOutcome[e.EntityID] != "manually_accepted" {
		s.execOutcome[e.EntityID] = "completed"
	}
	var p eventlog.ExecutionCompletedPayload
	if unmarshalMeta(e.Metadata, &p) {
		s.execDurations = append(s.execDurations, p.DurationSeconds/60.0)
		s.execHasFixup[e.EntityID] = p.HadFixups
	}
}

func (s *aggregateState) handleExecutionFailed(e *eventlog.Event) {
	s.execOutcome[e.EntityID] = "failed"
	var p eventlog.ExecutionFailedPayload
	if unmarshalMeta(e.Metadata, &p) {
		s.execDurations = append(s.execDurations, p.DurationSeconds/60.0)
	}
}

// --- Native Agent Session handlers ---

func (s *aggregateState) handleAgentSessionCreated(e *eventlog.Event) {
	var p agentSessionStatsPayload
	if unmarshalMeta(e.Metadata, &p) {
		s.sessionKind[e.EntityID] = p.SessionKind
		s.sessionStatus[e.EntityID] = p.Status
	}
	if s.sessionKind[e.EntityID] == "" {
		s.sessionKind[e.EntityID] = "unknown"
	}
	if s.sessionStatus[e.EntityID] == "" {
		s.sessionStatus[e.EntityID] = "starting"
	}
	s.sessionCreatedAt[e.EntityID] = e.Timestamp
	s.sessionMessageCount[e.EntityID]++
}

func (s *aggregateState) handleAgentSessionLifecycle(e *eventlog.Event) {
	var p agentSessionStatsPayload
	if unmarshalMeta(e.Metadata, &p) {
		if p.SessionKind != "" {
			s.sessionKind[e.EntityID] = p.SessionKind
		}
		if p.Status != "" {
			s.sessionStatus[e.EntityID] = p.Status
		}
	}
	if e.EventType == eventlog.EventAgentSessionContinued {
		s.sessionMessageCount[e.EntityID]++
	}
}

func (s *aggregateState) handleAgentSessionProposalCreated(e *eventlog.Event) {
	var p agentSessionProposalStatsPayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	kind := p.SessionKind
	if kind == "" {
		kind = s.sessionKind[e.EntityID]
	}
	if kind == "" {
		kind = "unknown"
	}
	s.sessionProposalCreatedByKind[kind]++
	if !s.sessionFirstProposalRecorded[e.EntityID] {
		if createdAt, ok := s.sessionCreatedAt[e.EntityID]; ok {
			s.sessionFirstProposalSeconds = append(s.sessionFirstProposalSeconds, e.Timestamp.Sub(createdAt).Seconds())
		}
		s.sessionFirstProposalRecorded[e.EntityID] = true
	}
}

func (s *aggregateState) handleAgentSessionProposalApplied(e *eventlog.Event) {
	var p agentSessionProposalStatsPayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	kind := p.SessionKind
	if kind == "" {
		kind = s.sessionKind[e.EntityID]
	}
	if kind == "" {
		kind = "unknown"
	}
	s.sessionProposalAppliedByKind[kind]++
}

func (s *aggregateState) handleAgentSessionArtifactLinked(e *eventlog.Event) {
	var p agentSessionArtifactStatsPayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	kind := p.SessionKind
	if kind == "" {
		kind = s.sessionKind[e.EntityID]
	}
	if kind == "" {
		kind = "unknown"
	}
	if p.ArtifactType != "" {
		s.sessionArtifactsByType[p.ArtifactType]++
	}
	if p.Action == "created" {
		s.sessionArtifactsCreatedByKind[kind]++
		switch p.ArtifactType {
		case "backlog_item":
			s.sessionCreatedBacklogItems++
		case "initiative":
			s.sessionCreatedInitiatives++
		}
	}
}

// --- Workshop handlers ---

func (s *aggregateState) handleWorkshopRoundCompleted(e *eventlog.Event) {
	var p eventlog.WorkshopRoundPayload
	if !unmarshalMeta(e.Metadata, &p) {
		return
	}
	if p.RoundNumber > s.workshopRounds[e.EntityID] {
		s.workshopRounds[e.EntityID] = p.RoundNumber
	}
	// Per-item decision counters. Pre-schema events leave these
	// zero; we just skip the contribution and continue.
	if p.ItemsTotal <= 0 {
		return
	}
	s.decisionItemsTotal += p.ItemsTotal
	s.decisionItemsAnswered += p.ItemsAnswered
	s.decisionItemsRecommendedChosen += p.ItemsRecommendedChosen
	s.decisionItemsFreeformChosen += p.ItemsFreeformChosen

	kind := p.Kind
	if kind == "" {
		if idx := indexByteFast(e.EntityID, '/'); idx > 0 {
			kind = e.EntityID[:idx]
		}
	}
	if kind == "" {
		kind = "unknown"
	}
	if _, known := workshop.BoostN[kind]; !known && kind != "unknown" {
		slog.Warn("recommendation-acceptance stats: unknown kind", "kind", kind, "entity", e.EntityID)
	}
	bucket, ok := s.decisionByKind[kind]
	if !ok {
		bucket = &decisionKindCounters{}
		s.decisionByKind[kind] = bucket
	}
	bucket.itemsTotal += p.ItemsTotal
	bucket.itemsAnswered += p.ItemsAnswered
	bucket.itemsRecommendedChosen += p.ItemsRecommendedChosen
	bucket.itemsFreeformChosen += p.ItemsFreeformChosen
}

// --- Review evidence handlers ---

func (s *aggregateState) handleReviewRoundCompleted(e *eventlog.Event) {
	s.reviewRoundsCompleted++
	var p eventlog.ReviewRoundCompletedPayload
	if unmarshalMeta(e.Metadata, &p) {
		s.reviewEvidenceCounts = append(s.reviewEvidenceCounts, p.EvidenceCount)
		s.reviewDurations = append(s.reviewDurations, p.DurationSecs)
	}
}

// --- Record handlers ---

func (s *aggregateState) handleRecordCreated(e *eventlog.Event) {
	s.recordTotal++
	s.recordCreatedAt = append(s.recordCreatedAt, e.Timestamp)
	var p eventlog.RecordCreatedPayload
	if unmarshalMeta(e.Metadata, &p) {
		if p.Kind != "" {
			s.recordsByKind[p.Kind]++
		}
		if p.Scenario != "" {
			s.recordsByScenario[p.Scenario]++
		}
		if p.BacklogRef != "" {
			s.recordsWithBacklogRef++
		}
		if p.Stub {
			s.recordsStubs++
		}
	}
}

func unmarshalMeta(data json.RawMessage, v any) bool {
	if len(data) == 0 {
		return false
	}
	if err := json.Unmarshal(data, v); err != nil {
		slog.Warn("unmarshal metadata failed", "error", err)
		return false
	}
	return true
}
