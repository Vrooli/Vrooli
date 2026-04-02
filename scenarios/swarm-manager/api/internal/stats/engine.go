package stats

import (
	"context"
	"encoding/json"
	"log"
	"sort"
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
	reviewRoundsCompleted   int
	reviewEvidenceCounts    []int
	reviewEvidenceVerified  int
	reviewRequestsCreated   int
	reviewDurations         []float64 // in seconds
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
		s.itemStatus[e.EntityID] = "archived"

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

func (s *aggregateState) buildResponse() StatsResponse {
	now := s.now()
	return StatsResponse{
		GeneratedAt: now,
		EventCount:  s.totalEvents,
		Throughput:  s.buildThroughput(now),
		Timing:      s.buildTiming(),
		Scope:       s.buildScope(),
		Blocking:    s.buildBlocking(),
		Agent:       s.buildAgent(),
		Dashboard:   s.buildDashboard(now),
		Review:      s.buildReview(),
	}
}

func (s *aggregateState) buildReview() ReviewStats {
	var avgEvidence float64
	if len(s.reviewEvidenceCounts) > 0 {
		total := 0
		for _, c := range s.reviewEvidenceCounts {
			total += c
		}
		avgEvidence = float64(total) / float64(len(s.reviewEvidenceCounts))
	}

	totalEvidence := 0
	for _, c := range s.reviewEvidenceCounts {
		totalEvidence += c
	}

	var verificationRate float64
	if totalEvidence > 0 {
		verificationRate = float64(s.reviewEvidenceVerified) / float64(totalEvidence)
	}

	var requestMoreRate float64
	if s.reviewRoundsCompleted > 0 {
		requestMoreRate = float64(s.reviewRequestsCreated) / float64(s.reviewRoundsCompleted)
	}

	var avgDuration float64
	if len(s.reviewDurations) > 0 {
		total := 0.0
		for _, d := range s.reviewDurations {
			total += d
		}
		avgDuration = total / float64(len(s.reviewDurations))
	}

	return ReviewStats{
		RoundsCompleted:         s.reviewRoundsCompleted,
		AverageEvidencePerRound: avgEvidence,
		VerificationRate:        verificationRate,
		RequestMoreRate:         requestMoreRate,
		AverageReviewDuration:   avgDuration,
	}
}

func (s *aggregateState) buildThroughput(now time.Time) ThroughputStats {
	d7 := now.Add(-7 * 24 * time.Hour)
	d30 := now.Add(-30 * 24 * time.Hour)

	created7 := countAfter(s.createdEvents, d7)
	created30 := countAfter(s.createdEvents, d30)
	completed7 := countAfter(s.completedEvents, d7)
	completed30 := countAfter(s.completedEvents, d30)

	return ThroughputStats{
		CompletedLast7Days:  completed7,
		CompletedLast30Days: completed30,
		CreatedLast7Days:    created7,
		CreatedLast30Days:   created30,
		NetDelta7Days:       created7 - completed7,
		NetDelta30Days:      created30 - completed30,
	}
}

func (s *aggregateState) buildTiming() TimingStats {
	return TimingStats{
		AvgCycleTimeHours:    avgFloat(s.cycleTimesH),
		AvgLeadTimeHours:     avgFloat(s.leadTimesH),
		AvgQueueWaitHours:    avgFloat(s.queueWaitH),
		MedianCycleTimeHours: medianFloat(s.cycleTimesH),
		MedianLeadTimeHours:  medianFloat(s.leadTimesH),
	}
}

func (s *aggregateState) buildScope() ScopeStats {
	var inits []InitiativeHealth
	for name := range s.initiativeCreated {
		items := s.initiativeItems[name]
		ih := InitiativeHealth{
			Name:  name,
			Total: len(items),
		}
		for itemID := range items {
			switch s.itemStatus[itemID] {
			case "completed":
				ih.Completed++
			case "in_progress", "queued":
				ih.InProgress++
			}
			if _, blocked := s.blockedItems[itemID]; blocked {
				ih.Blocked++
			}
		}
		initial := s.initiativeInitial[name]
		if initial > 0 {
			ih.ScopeCreep = float64(len(items)-initial) / float64(initial)
		}
		inits = append(inits, ih)
	}
	sort.Slice(inits, func(i, j int) bool { return inits[i].Name < inits[j].Name })

	return ScopeStats{
		Initiatives: inits,
	}
}

func (s *aggregateState) buildBlocking() BlockingStats {
	blocked := len(s.blockedItems)
	total := len(s.currentBacklog)
	var ratio float64
	if total > 0 {
		ratio = float64(blocked) / float64(total)
	}

	// Top reasons sorted by count descending.
	var reasons []ReasonCount
	for reason, count := range s.blockReasons {
		reasons = append(reasons, ReasonCount{Reason: reason, Count: count})
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i].Count > reasons[j].Count })
	if len(reasons) > 10 {
		reasons = reasons[:10]
	}

	return BlockingStats{
		CurrentlyBlocked: blocked,
		BlockedRatio:     ratio,
		TopReasons:       reasons,
		AvgBlockHours:    avgFloat(s.blockDurations),
	}
}

func (s *aggregateState) buildAgent() AgentStats {
	var successRate, failureRate float64
	finished := s.execCompleted + s.execFailed
	if finished > 0 {
		successRate = float64(s.execCompleted) / float64(finished)
		failureRate = float64(s.execFailed) / float64(finished)
	}

	var followUpRate float64
	fixupCount := 0
	for _, had := range s.execHasFixup {
		if had {
			fixupCount++
		}
	}
	if s.execCompleted > 0 {
		followUpRate = float64(fixupCount) / float64(s.execCompleted)
	}

	var avgRounds float64
	if len(s.workshopRounds) > 0 {
		total := 0
		for _, r := range s.workshopRounds {
			total += r
		}
		avgRounds = float64(total) / float64(len(s.workshopRounds))
	}

	return AgentStats{
		TotalExecutions:     s.execTotal,
		SuccessRate:         successRate,
		FailureRate:         failureRate,
		FollowUpRate:        followUpRate,
		AvgExecutionMinutes: avgFloat(s.execDurations),
		AvgWorkshopRounds:   avgRounds,
	}
}

func (s *aggregateState) buildDashboard(now time.Time) DashboardStats {
	// Velocity trend: weekly completions over trailing 8 weeks.
	var trend []VelocityPoint
	for i := 7; i >= 0; i-- {
		weekStart := now.Add(-time.Duration(i*7*24) * time.Hour).Truncate(24 * time.Hour)
		weekEnd := weekStart.Add(7 * 24 * time.Hour)
		count := 0
		for _, t := range s.completedEvents {
			if !t.Before(weekStart) && t.Before(weekEnd) {
				count++
			}
		}
		trend = append(trend, VelocityPoint{
			WeekStart: weekStart.Format("2006-01-02"),
			Completed: count,
		})
	}

	// Avg velocity from last 4 full weeks (indices 4-7 in the 9 weeks).
	avgVelocity := 0.0
	if len(trend) >= 5 {
		total := 0
		weeks := 0
		for i := len(trend) - 5; i < len(trend)-1; i++ {
			total += trend[i].Completed
			weeks++
		}
		if weeks > 0 {
			avgVelocity = float64(total) / float64(weeks)
		}
	}

	var weeksRemaining float64
	if avgVelocity > 0 {
		weeksRemaining = float64(len(s.currentBacklog)) / avgVelocity
	}

	return DashboardStats{
		TotalBacklogSize:        len(s.currentBacklog),
		TotalCompletedAllTime:   s.completedAllTime,
		VelocityTrend:           trend,
		EstimatedWeeksRemaining: weeksRemaining,
	}
}

// --- helpers ---

func countAfter(timestamps []time.Time, after time.Time) int {
	count := 0
	for _, t := range timestamps {
		if t.After(after) {
			count++
		}
	}
	return count
}

func avgFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func medianFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func unmarshalMeta(data json.RawMessage, v any) bool {
	if len(data) == 0 {
		return false
	}
	if err := json.Unmarshal(data, v); err != nil {
		log.Printf("[stats] unmarshal metadata: %v", err)
		return false
	}
	return true
}
