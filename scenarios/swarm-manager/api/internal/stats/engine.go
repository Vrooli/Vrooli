package stats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/workshop"
)

// indexByteFast returns the index of the first occurrence of c in s,
// or -1 if absent. Used to parse kind from entity IDs of the form
// "<kind>/<name>" without dragging in the strings package for one call.
func indexByteFast(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

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
	initiativeMode    map[string]string          // initiative → current operating mode
	modeSwitchCount   int
	itemStatus        map[string]string // entity_id → current status

	// Operating-mode tracking.
	modePhaseRuns map[string]map[string]int
	// modePhaseRunsByLane counts started phase events grouped by lane
	// (PhaseKind from the payload). Empty-key bucket catches legacy
	// events written before P2 wired phase_kind on the payload.
	modePhaseRunsByLane     map[string]int
	modeCompleted           map[string]int
	modeFailed              map[string]int
	modeCanceled            map[string]int
	modeReplanNumerator     map[string]int
	modeReplanDenominator   map[string]int
	modeAcceptanceNumerator map[string]int
	modeAcceptanceDenom     map[string]int
	modeDurationSums        map[string]map[string]float64
	modeDurationCounts      map[string]map[string]int
	modeCompletedScopes     map[string]map[string]bool
	modeProfileUsage        map[string]int
	modeProfilePhaseRuns    map[string]map[string]int
	modeBacklogSync         map[string]*BacklogSyncStats

	// Execution tracking.
	//
	// execOutcome captures the *current* terminal outcome for each execution
	// id, overwritten if the execution later transitions (e.g. failed →
	// manually_accepted). Counters like "completed" and "failed" are derived
	// from this map at read time so manual overrides do not double-count.
	execTotal             int
	execOutcome           map[string]string // exec_id → "completed" | "failed" | "canceled" | "manually_accepted"
	execDurations         []float64         // in minutes, captured at each terminal transition
	execHasFixup          map[string]bool   // exec_id → had fixups
	earliestEventAt       time.Time         // timestamp of the earliest observed event
	earliestEventRecorded bool

	// Workshop tracking.
	workshopRounds map[string]int // entity_id → max round number

	// Decision recommendation tracking.
	//
	// Aggregated from decision.workshop_round_completed payloads. Per-kind
	// counters use the same fields and are keyed by the entity kind ("idea",
	// "research", "fix", "execute", "chore"). An "unknown" bucket catches
	// any kind not in the BoostN map so we notice if a new kind appears
	// without map updates (logged at WARN by the engine).
	decisionItemsTotal             int
	decisionItemsAnswered          int
	decisionItemsRecommendedChosen int
	decisionItemsFreeformChosen    int
	decisionByKind                 map[string]*decisionKindCounters

	// Review evidence tracking.
	reviewRoundsCompleted  int
	reviewEvidenceCounts   []int
	reviewEvidenceVerified int
	reviewRequestsCreated  int
	reviewDurations        []float64 // in seconds

	// Native Agent Session tracking.
	sessionKind                   map[string]string
	sessionStatus                 map[string]string
	sessionCreatedAt              map[string]time.Time
	sessionMessageCount           map[string]int
	sessionFirstProposalRecorded  map[string]bool
	sessionFirstProposalSeconds   []float64
	sessionProposalCreatedByKind  map[string]int
	sessionProposalAppliedByKind  map[string]int
	sessionArtifactsCreatedByKind map[string]int
	sessionArtifactsByType        map[string]int
	sessionCreatedBacklogItems    int
	sessionCreatedInitiatives     int

	// Record tracking. Counters fold EventRecordCreated and
	// EventRecordSuperseded. There is no stub-fill event, so "filled stub"
	// state is not derivable from the event log; stubs counts created-as-stub
	// only.
	recordTotal            int
	recordCreatedAt        []time.Time
	recordsByKind          map[string]int
	recordsByScenario      map[string]int
	recordsWithBacklogRef  int
	recordsStubs           int
	recordsSupersedeCount  int
}

func newAggregateState() *aggregateState {
	return &aggregateState{
		now:                           time.Now,
		currentBacklog:                make(map[string]bool),
		createdAt:                     make(map[string]time.Time),
		inProgressAt:                  make(map[string]time.Time),
		queuedAt:                      make(map[string]time.Time),
		blockedItems:                  make(map[string]time.Time),
		blockReasons:                  make(map[string]int),
		initiativeItems:               make(map[string]map[string]bool),
		initiativeInitial:             make(map[string]int),
		initiativeCreated:             make(map[string]bool),
		initiativeMode:                make(map[string]string),
		itemStatus:                    make(map[string]string),
		modePhaseRuns:                 make(map[string]map[string]int),
		modePhaseRunsByLane:           make(map[string]int),
		modeCompletedScopes:           make(map[string]map[string]bool),
		modeCompleted:                 make(map[string]int),
		modeFailed:                    make(map[string]int),
		modeCanceled:                  make(map[string]int),
		modeReplanNumerator:           make(map[string]int),
		modeReplanDenominator:         make(map[string]int),
		modeAcceptanceNumerator:       make(map[string]int),
		modeAcceptanceDenom:           make(map[string]int),
		modeDurationSums:              make(map[string]map[string]float64),
		modeDurationCounts:            make(map[string]map[string]int),
		modeProfileUsage:              make(map[string]int),
		modeProfilePhaseRuns:          make(map[string]map[string]int),
		modeBacklogSync:               make(map[string]*BacklogSyncStats),
		execHasFixup:                  make(map[string]bool),
		workshopRounds:                make(map[string]int),
		execOutcome:                   make(map[string]string),
		decisionByKind:                make(map[string]*decisionKindCounters),
		sessionKind:                   make(map[string]string),
		sessionStatus:                 make(map[string]string),
		sessionCreatedAt:              make(map[string]time.Time),
		sessionMessageCount:           make(map[string]int),
		sessionFirstProposalRecorded:  make(map[string]bool),
		sessionProposalCreatedByKind:  make(map[string]int),
		sessionProposalAppliedByKind:  make(map[string]int),
		sessionArtifactsCreatedByKind: make(map[string]int),
		sessionArtifactsByType:        make(map[string]int),
		recordsByKind:                 make(map[string]int),
		recordsByScenario:             make(map[string]int),
	}
}

// decisionKindCounters holds per-kind decision counters used for the
// per-kind breakdown of recommendation acceptance.
type decisionKindCounters struct {
	itemsTotal             int
	itemsAnswered          int
	itemsRecommendedChosen int
	itemsFreeformChosen    int
}

type agentSessionStatsPayload struct {
	SessionKind string `json:"session_kind"`
	Status      string `json:"status"`
}

type agentSessionProposalStatsPayload struct {
	SessionKind  string `json:"session_kind"`
	ProposalKind string `json:"proposal_kind"`
}

type agentSessionArtifactStatsPayload struct {
	SessionKind  string `json:"session_kind"`
	ArtifactType string `json:"artifact_type"`
	Action       string `json:"action"`
}

func (s *aggregateState) recordModePhaseStarted(p eventlog.OperatingModePhasePayload) {
	if p.Mode == "" || p.Phase == "" {
		return
	}
	incrementNested(s.modePhaseRuns, p.Mode, p.Phase, 1)
	// Count by lane regardless of mode so the Operations Center can show
	// historical pressure across all initiative-shaped work.
	s.modePhaseRunsByLane[p.PhaseKind]++
	if p.AgentProfileKey != "" {
		s.modeProfileUsage[p.AgentProfileKey]++
		incrementNested(s.modeProfilePhaseRuns, p.AgentProfileKey, p.Phase, 1)
	}
}

func (s *aggregateState) recordModePhaseTerminal(p eventlog.OperatingModePhasePayload, outcome string) {
	if p.Mode == "" {
		return
	}
	switch outcome {
	case "completed":
		s.modeCompleted[p.Mode]++
		if p.ScopeID != "" {
			if s.modeCompletedScopes[p.Mode] == nil {
				s.modeCompletedScopes[p.Mode] = make(map[string]bool)
			}
			s.modeCompletedScopes[p.Mode][p.ScopeID] = true
		}
	case "failed":
		s.modeFailed[p.Mode]++
	case "canceled":
		s.modeCanceled[p.Mode]++
	}
	if p.DurationSeconds > 0 && p.Phase != "" {
		addNestedFloat(s.modeDurationSums, p.Mode, p.Phase, p.DurationSeconds)
		incrementNested(s.modeDurationCounts, p.Mode, p.Phase, 1)
	}
	if outcome == "completed" {
		policy, ok := operatingModeMetricsPolicy(p.Mode)
		if !ok {
			return
		}
		s.recordOperatingModePolicyMetrics(p, policy)
	}
}

func (s *aggregateState) recordOperatingModePolicyMetrics(p eventlog.OperatingModePhasePayload, policy operatingmode.MetricsPolicy) {
	phase := operatingmode.Phase(p.Phase)
	if policy.CountsReplanSample(phase) {
		s.modeReplanDenominator[p.Mode]++
		if p.ReplanNeeded {
			s.modeReplanNumerator[p.Mode]++
		}
	}
	if policy.CountsAcceptanceSample(phase) && p.Verdict != "" {
		s.modeAcceptanceDenom[p.Mode]++
		if policy.IsAcceptedVerdict(p.Verdict) {
			s.modeAcceptanceNumerator[p.Mode]++
		}
	}
}

func operatingModeMetricsPolicy(mode string) (operatingmode.MetricsPolicy, bool) {
	def, err := operatingmode.DefinitionFor(operatingmode.Mode(mode))
	if err != nil {
		return operatingmode.MetricsPolicy{}, false
	}
	return def.Metrics, true
}

func incrementNested(m map[string]map[string]int, outer, inner string, delta int) {
	if m[outer] == nil {
		m[outer] = make(map[string]int)
	}
	m[outer][inner] += delta
}

func addNestedFloat(m map[string]map[string]float64, outer, inner string, delta float64) {
	if m[outer] == nil {
		m[outer] = make(map[string]float64)
	}
	m[outer][inner] += delta
}

// countExecOutcomes returns (completed, failed, manuallyAccepted) from
// execOutcome. Canceled and non-terminal outcomes are excluded from both
// numerator and denominator of success-rate math.
func (s *aggregateState) countExecOutcomes() (completed, failed, manuallyAccepted int) {
	for _, outcome := range s.execOutcome {
		switch outcome {
		case "completed":
			completed++
		case "manually_accepted":
			completed++
			manuallyAccepted++
		case "failed":
			failed++
		}
	}
	return
}

func (s *aggregateState) processEvent(e *eventlog.Event) {
	s.totalEvents++
	if !s.earliestEventRecorded || e.Timestamp.Before(s.earliestEventAt) {
		s.earliestEventAt = e.Timestamp
		s.earliestEventRecorded = true
	}

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
		if s.initiativeMode[e.EntityID] == "" {
			s.initiativeMode[e.EntityID] = "item-level"
		}
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

	case eventlog.EventInitiativeModeChanged:
		var p eventlog.InitiativeModeChangePayload
		if !unmarshalMeta(e.Metadata, &p) {
			return
		}
		s.modeSwitchCount++
		if p.To != "" {
			s.initiativeMode[e.EntityID] = p.To
		}

	case eventlog.EventOperatingModePhaseStarted:
		var p eventlog.OperatingModePhasePayload
		if unmarshalMeta(e.Metadata, &p) {
			s.recordModePhaseStarted(p)
		}

	case eventlog.EventOperatingModePhaseCompleted:
		var p eventlog.OperatingModePhasePayload
		if unmarshalMeta(e.Metadata, &p) {
			s.recordModePhaseTerminal(p, "completed")
		}

	case eventlog.EventOperatingModePhaseFailed:
		var p eventlog.OperatingModePhasePayload
		if unmarshalMeta(e.Metadata, &p) {
			s.recordModePhaseTerminal(p, "failed")
		}

	case eventlog.EventOperatingModePhaseCanceled:
		var p eventlog.OperatingModePhasePayload
		if unmarshalMeta(e.Metadata, &p) {
			s.recordModePhaseTerminal(p, "canceled")
		}

	case eventlog.EventOperatingModeBacklogSynced:
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

	// --- Execution ---
	case eventlog.EventExecutionCreated:
		s.execTotal++
		var p eventlog.ExecutionCreatedPayload
		_ = unmarshalMeta(e.Metadata, &p)

	case eventlog.EventExecutionCompleted:
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

	case eventlog.EventExecutionFailed:
		s.execOutcome[e.EntityID] = "failed"
		var p eventlog.ExecutionFailedPayload
		if unmarshalMeta(e.Metadata, &p) {
			s.execDurations = append(s.execDurations, p.DurationSeconds/60.0)
		}

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

	case eventlog.EventAgentSessionStarted, eventlog.EventAgentSessionContinued, eventlog.EventAgentSessionCompleted,
		eventlog.EventAgentSessionFailed, eventlog.EventAgentSessionCanceled:
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

	case eventlog.EventAgentSessionProposalCreated:
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

	case eventlog.EventAgentSessionProposalApplied:
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

	case eventlog.EventAgentSessionArtifactLinked:
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

	// --- Workshop ---
	case eventlog.EventWorkshopRoundCompleted:
		var p eventlog.WorkshopRoundPayload
		if unmarshalMeta(e.Metadata, &p) {
			if p.RoundNumber > s.workshopRounds[e.EntityID] {
				s.workshopRounds[e.EntityID] = p.RoundNumber
			}
			// Per-item decision counters. Pre-schema events leave these
			// zero; we just skip the contribution and continue.
			if p.ItemsTotal > 0 {
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

	// --- Records ---
	case eventlog.EventRecordCreated:
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

	case eventlog.EventRecordSuperseded:
		s.recordsSupersedeCount++
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
