package stats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"swarm-manager/internal/eventlog"
)

const refreshBatchSize = 5000

// Engine incrementally aggregates events into metrics using a watermark pattern.
type Engine struct {
	mu        sync.RWMutex
	watermark int64
	repo      eventlog.Repository
	state     *aggregateState
}

// NewEngine creates a stats engine backed by the given event repository.
func NewEngine(repo eventlog.Repository) *Engine {
	return &Engine{
		repo:  repo,
		state: newAggregateState(),
	}
}

// Rebuild replays all events from scratch. Called once at startup.
func (e *Engine) Rebuild(ctx context.Context) error {
	events, err := e.repo.All(ctx)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	e.state = newAggregateState()
	for i := range events {
		e.state.processEvent(&events[i])
	}

	maxID, err := e.repo.MaxID(ctx)
	if err != nil {
		return err
	}
	e.watermark = maxID
	return nil
}

// Refresh incrementally processes events appended since the last watermark.
func (e *Engine) Refresh(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for {
		events, err := e.repo.Since(ctx, e.watermark, refreshBatchSize)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		for i := range events {
			e.state.processEvent(&events[i])
		}
		e.watermark = events[len(events)-1].ID
		if len(events) < refreshBatchSize {
			return nil
		}
	}
}

// GetStats returns the current computed metrics. Callers should call Refresh first.
func (e *Engine) GetStats() StatsResponse {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.buildResponse()
}

// aggregateState holds all running counters and maps needed for metric computation.
type aggregateState struct {
	now func() time.Time // seam for testing

	// Event counter.
	totalEvents int64

	// Backlog tracking.
	createdEvents    []time.Time     // timestamps of backlog.created events
	completedEvents  []time.Time     // timestamps of backlog.status_changed to completed
	currentBacklog   map[string]bool // entity IDs of non-completed/non-archived backlog items
	completedAllTime int

	// Timing tracking.
	createdAt    map[string]time.Time // entity_id → created timestamp
	inProgressAt map[string]time.Time // entity_id → when moved to in_progress
	queuedAt     map[string]time.Time // entity_id → when queued
	cycleTimesH  []float64            // completed cycle times in hours
	leadTimesH   []float64            // completed lead times in hours
	queueWaitH   []float64            // completed queue wait times in hours

	// Blocking tracking.
	blockedItems   map[string]time.Time // entity_id → when blocked
	blockReasons   map[string]int       // reason → count
	blockDurations []float64            // resolved block durations in hours

	// Initiative tracking.
	initiativeItems   map[string]map[string]bool // initiative → set of items
	initiativeInitial map[string]int             // initiative → item count at creation
	initiativeCreated map[string]bool            // initiatives that exist
	itemStatus        map[string]string          // entity_id → current status

	// Execution tracking.
	execTotal     int
	execCompleted int
	execFailed    int
	execDurations []float64       // in minutes
	execHasFixup  map[string]bool // exec_id → had fixups

	// Workshop tracking.
	workshopRounds map[string]int // entity_id → max round number

	// Review evidence tracking.
	reviewRoundsCompleted  int
	reviewEvidenceCounts   []int
	reviewEvidenceVerified int
	reviewRequestsCreated  int
	reviewDurations        []float64 // in seconds
}

func newAggregateState() *aggregateState {
	return &aggregateState{
		now:               time.Now,
		currentBacklog:    make(map[string]bool),
		createdAt:         make(map[string]time.Time),
		inProgressAt:      make(map[string]time.Time),
		queuedAt:          make(map[string]time.Time),
		blockedItems:      make(map[string]time.Time),
		blockReasons:      make(map[string]int),
		initiativeItems:   make(map[string]map[string]bool),
		initiativeInitial: make(map[string]int),
		initiativeCreated: make(map[string]bool),
		itemStatus:        make(map[string]string),
		execHasFixup:      make(map[string]bool),
		workshopRounds:    make(map[string]int),
	}
}

func (s *aggregateState) processEvent(e *eventlog.Event) {
	s.totalEvents++

	switch e.EventType {
	// --- Backlog ---
	case eventlog.EventBacklogCreated:
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

	case eventlog.EventBacklogStatusChanged:
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

	case eventlog.EventBacklogArchived:
		delete(s.currentBacklog, e.EntityID)
		var p eventlog.ArchivePayload
		if unmarshalMeta(e.Metadata, &p) && p.PreviousStatus != "" {
			s.itemStatus[e.EntityID] = p.PreviousStatus
		} else {
			// Historical events before the migration may have nil metadata.
			s.itemStatus[e.EntityID] = "archived"
		}

	case eventlog.EventBacklogUnarchived:
		// Restore item to active backlog using whatever status we have recorded.
		s.currentBacklog[e.EntityID] = true

	case eventlog.EventBacklogDeleted:
		delete(s.currentBacklog, e.EntityID)
		delete(s.itemStatus, e.EntityID)

	case eventlog.EventBacklogBlocked:
		s.blockedItems[e.EntityID] = e.Timestamp
		var p eventlog.BlockPayload
		if unmarshalMeta(e.Metadata, &p) && p.Reason != "" {
			s.blockReasons[p.Reason]++
		}

	case eventlog.EventBacklogUnblocked:
		if blockedAt, ok := s.blockedItems[e.EntityID]; ok {
			s.blockDurations = append(s.blockDurations, e.Timestamp.Sub(blockedAt).Hours())
			delete(s.blockedItems, e.EntityID)
		}

	case eventlog.EventBacklogInitiativeChanged:
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

	// --- Initiative ---
	case eventlog.EventInitiativeCreated:
		s.initiativeCreated[e.EntityID] = true
		if s.initiativeItems[e.EntityID] == nil {
			s.initiativeItems[e.EntityID] = make(map[string]bool)
		}

	case eventlog.EventInitiativeItemAdded:
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

	case eventlog.EventInitiativeItemRemoved:
		var p eventlog.InitiativeItemPayload
		if unmarshalMeta(e.Metadata, &p) {
			if items := s.initiativeItems[e.EntityID]; items != nil {
				delete(items, p.Item)
			}
		}

	// --- Execution ---
	case eventlog.EventExecutionCreated:
		s.execTotal++
		var p eventlog.ExecutionCreatedPayload
		// Track if this is a follow-up (indicates follow-up rate).
		// We use a simple heuristic: if the backlog item already has an execution, it's a follow-up.
		// TODO: implement follow-up tracking once execution history is available.
		_ = unmarshalMeta(e.Metadata, &p)

	case eventlog.EventExecutionCompleted:
		s.execCompleted++
		var p eventlog.ExecutionCompletedPayload
		if unmarshalMeta(e.Metadata, &p) {
			s.execDurations = append(s.execDurations, p.DurationSeconds/60.0)
			s.execHasFixup[e.EntityID] = p.HadFixups
		}

	case eventlog.EventExecutionFailed:
		s.execFailed++
		var p eventlog.ExecutionFailedPayload
		if unmarshalMeta(e.Metadata, &p) {
			s.execDurations = append(s.execDurations, p.DurationSeconds/60.0)
		}

	case eventlog.EventExecutionCanceled:
		// Cancellations don't count toward success or failure rate.

	// --- Workshop ---
	case eventlog.EventWorkshopRoundCompleted:
		var p eventlog.WorkshopRoundPayload
		if unmarshalMeta(e.Metadata, &p) {
			if p.RoundNumber > s.workshopRounds[e.EntityID] {
				s.workshopRounds[e.EntityID] = p.RoundNumber
			}
		}

	// --- Review evidence ---
	case eventlog.EventReviewRoundCompleted:
		s.reviewRoundsCompleted++
		var p eventlog.ReviewRoundCompletedPayload
		if unmarshalMeta(e.Metadata, &p) {
			s.reviewEvidenceCounts = append(s.reviewEvidenceCounts, p.EvidenceCount)
			s.reviewDurations = append(s.reviewDurations, p.DurationSecs)
		}

	case eventlog.EventReviewEvidenceVerified:
		s.reviewEvidenceVerified++

	case eventlog.EventReviewRequestCreated:
		s.reviewRequestsCreated++
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
