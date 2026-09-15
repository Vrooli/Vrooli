package stats

import (
	"time"

	"swarm-manager/internal/eventlog"
)

// aggregateState holds all running counters and maps needed for metric computation.
type aggregateState struct {
	now func() time.Time // seam for testing

	// Event counter.
	totalEvents int64

	// Backlog tracking.
	createdEvents    []time.Time      // timestamps of backlog.created events
	completedEvents  []completedEvent // backlog.status_changed events to completed
	currentBacklog   map[string]bool  // entity IDs of non-completed/non-archived backlog items
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
	decisionByGate                 map[string]*decisionKindCounters

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
	recordTotal           int
	recordCreatedAt       []time.Time
	recordsByKind         map[string]int
	recordsByScenario     map[string]int
	recordsWithBacklogRef int
	recordsStubs          int
	recordsSupersedeCount int
}

type completedEvent struct {
	Timestamp time.Time
	Kind      string
	Name      string
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
		decisionByGate:                make(map[string]*decisionKindCounters),
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

// countExecOutcomes returns terminal execution outcomes. Abstentions and
// budget exhaustion are separate from ordinary failures.
func (s *aggregateState) countExecOutcomes() (completed, failed, abstained, budgetExhausted, manuallyAccepted int) {
	for _, outcome := range s.execOutcome {
		switch outcome {
		case "completed":
			completed++
		case "manually_accepted":
			completed++
			manuallyAccepted++
		case "failed":
			failed++
		case "abstained":
			abstained++
		case "budget_exhausted":
			budgetExhausted++
		}
	}
	return
}
