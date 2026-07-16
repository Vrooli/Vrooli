package runmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Executor is the suite engine the manager drives. It is satisfied by
// *execution.SuiteExecutionService. A nil emit runs silently; the manager
// always supplies an emit that fans events into the run's broadcaster.
type Executor interface {
	ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error)
}

// Heartbeat cadence (quiet-phase keep-alive). Tunable lever, shared with the
// SSE gateway: TEST_GENIE_HEARTBEAT_SECONDS (default 30s, floor 5s).
const (
	defaultHeartbeatInterval = 30 * time.Second
	minHeartbeatInterval     = 5 * time.Second
)

// retireGrace is how long a terminal run lingers in the in-memory registry
// after completion so immediately-following Status/Wait/Follow calls read the
// authoritative live snapshot before falling back to the durable index. The
// run-admission guard deliberately ignores these lingering terminal runs: a
// just-finished run never blocks or coalesces a fresh start.
const retireGrace = 60 * time.Second

// defaultMaxRunsPerScenario is the per-scenario in-progress run cap. It is 1
// because concurrent runs of the SAME scenario are not merely wasteful but
// INCORRECT: every run brings the target up via targetruntime.EnsureRunning,
// which shares one live instance (ports, DB, fixtures) with no mutual
// exclusion, and one run's Cleanup can tear the scenario down out from under
// another. Serialization — not isolation — is the invariant. The lever
// TEST_GENIE_MAX_RUNS_PER_SCENARIO exists only as a future escape hatch for
// when per-run isolation lands; >1 is documented-unsafe until then.
const defaultMaxRunsPerScenario = 1

// maxRunsPerScenarioFromEnv resolves the configured per-scenario cap, never
// returning less than 1.
func maxRunsPerScenarioFromEnv() int {
	if raw := strings.TrimSpace(os.Getenv("TEST_GENIE_MAX_RUNS_PER_SCENARIO")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			return n
		}
	}
	return defaultMaxRunsPerScenario
}

// defaultMaxConcurrentRuns is the GLOBAL cap on simultaneously-executing runs
// across ALL scenarios. Unlike the per-scenario cap (which is a correctness
// invariant), this is a load governor: every run brings a scenario's full stack
// up and drives a suite, so N comprehensive suites at once saturate a shared
// host (the 2026-06-21 incident: three concurrent baseline suites on a box also
// running another agent stalled one executor). Requests beyond the cap are not
// rejected — they are admitted as StatusQueued and promoted FIFO as slots free.
//
// The default is intentionally conservative (2): this host routinely also runs
// background agent sessions, the autoheal loop, and a supervisor. It is the one
// shared budget for BOTH manually-started runs and the background fleet sweep,
// because the fleet scheduler launches through this same Manager.Start path.
const (
	defaultMaxConcurrentRuns   = 2
	defaultMaxQueuedRuns       = 16
	defaultMaxPreviewRuns      = 2
	defaultMaxQueuedPerCaller  = 4
	defaultMaxPreviewPerCaller = 1
)

// maxConcurrentRunsFromEnv resolves the configured global concurrency cap from
// TEST_GENIE_MAX_CONCURRENT_RUNS (the env lever; a settings-domain UI control is
// a planned follow-on), never returning less than 1.
func maxConcurrentRunsFromEnv() int {
	if raw := strings.TrimSpace(os.Getenv("TEST_GENIE_MAX_CONCURRENT_RUNS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			return n
		}
	}
	return defaultMaxConcurrentRuns
}

func boundedIntFromEnv(name string, fallback int) int {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			return n
		}
	}
	return fallback
}

// HeartbeatInterval resolves the configured quiet-phase heartbeat cadence.
func HeartbeatInterval() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("TEST_GENIE_HEARTBEAT_SECONDS")); raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs >= int(minHeartbeatInterval.Seconds()) {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultHeartbeatInterval
}

// Manager owns durable run execution decoupled from any client request.
type Manager struct {
	base                context.Context
	cancelBase          context.CancelFunc
	exec                Executor
	scenariosRoot       string
	heartbeat           time.Duration
	maxRunsPerScenario  int
	maxConcurrentRuns   int
	maxQueuedRuns       int
	maxQueuedPerCaller  int
	maxPreviewPerCaller int
	previewTokens       chan struct{}
	previewByCaller     map[string]int

	// wg tracks live drive goroutines so Shutdown can wait for in-flight runs to
	// finalize their durable records before returning.
	wg sync.WaitGroup

	mu   sync.Mutex
	runs map[string]*activeRun

	previewRejectedTotal  uint64
	queueRejectedTotal    uint64
	scenarioRejectedTotal uint64
	coalescedTotal        uint64
}

// activeRun is the in-memory state for a run currently tracked by the manager.
type activeRun struct {
	runID    string
	scenario string
	preset   string
	// admissionKey is the deterministic coalescing identity for this run (see
	// admissionKey). Immutable after construction, so it is safe to read under
	// the manager lock during admission.
	admissionKey string
	caller       string
	startedAt    time.Time
	cancel       context.CancelFunc
	bc           *broadcaster
	done         chan struct{}
	// runCtx and input are retained so a run admitted in StatusQueued can be
	// driven later by the dispatcher when a global concurrency slot frees. They
	// are unused for runs that start immediately.
	runCtx context.Context
	input  execution.SuiteExecutionInput

	mu          sync.Mutex
	status      string
	activePhase string
	phaseIndex  int
	phaseTotal  int
	lastEventAt time.Time
	etaTotal    int
	result      *orchestrator.SuiteExecutionResult
	err         error
}

// LiveStatus is a point-in-time snapshot of a run, sourced from the in-memory
// registry when active or the durable index otherwise.
type LiveStatus struct {
	RunID                       string
	Scenario                    string
	Status                      string
	ActivePhase                 string
	PhaseIndex                  int
	PhaseTotal                  int
	StartedAt                   time.Time
	ElapsedSeconds              float64
	EstimatedTotalSeconds       int
	EstimatedRemainingSeconds   int
	ETAKnown                    bool
	RecommendedNextCheckSeconds int
	Verdict                     string
	Success                     bool
	Error                       string
	// Active reports whether the snapshot came from the live registry (true) or
	// the durable index (false).
	Active bool
	// Result is the full terminal result when known (live terminal snapshot).
	Result *orchestrator.SuiteExecutionResult
	// TerminalPresentations is populated from Result.Phases on terminal live
	// snapshots, preserving the provider-computed standing for agent wait paths.
	TerminalPresentations     []*commonv1.PhasePresentation
	TerminalFindingsSummaries []*runspb.PhaseFindingsSummary
	// TerminalRecord is the canonical compact projection stored inside the
	// terminal snapshot. It is populated only when that snapshot validated.
	TerminalRecord                *sharedruns.RunRecord
	TerminalSnapshotSchemaVersion int
	DescriptorSnapshot            *sharedruns.DescriptorSnapshot
	// DegradedReasons explains why a durable terminal projection is incomplete.
	// It is empty for canonical snapshots and populated for legacy/corrupt data;
	// wire consumers must preserve it as UNKNOWN rather than zero-fill fields.
	DegradedReasons []string
}

// StartOptions configures a run start.
type StartOptions struct {
	Input execution.SuiteExecutionInput
	// Caller is a bounded transport identity used only for admission fairness
	// and aggregate telemetry; it is never persisted in run artifacts.
	Caller string
	// EstimatedTotalSeconds is the summed plan-preview estimate, used for the
	// remaining-ETA and recommended-next-check fields. Zero means unknown.
	EstimatedTotalSeconds int
}

// AdmissionSnapshot is the bounded admission state exposed to operators. The
// counters are process-lifetime totals while the occupancy fields are sampled
// at the instant this snapshot is read.
type AdmissionSnapshot struct {
	Running                 int    `json:"running"`
	Queued                  int    `json:"queued"`
	MaxConcurrentRuns       int    `json:"maxConcurrentRuns"`
	MaxQueuedRuns           int    `json:"maxQueuedRuns"`
	MaxQueuedRunsPerCaller  int    `json:"maxQueuedRunsPerCaller"`
	PreviewInFlight         int    `json:"previewInFlight"`
	MaxPreviewRuns          int    `json:"maxPreviewRuns"`
	MaxPreviewRunsPerCaller int    `json:"maxPreviewRunsPerCaller"`
	PreviewRejectedTotal    uint64 `json:"previewRejectedTotal"`
	QueueRejectedTotal      uint64 `json:"queueRejectedTotal"`
	ScenarioRejectedTotal   uint64 `json:"scenarioRejectedTotal"`
	CoalescedTotal          uint64 `json:"coalescedTotal"`
}

// New constructs a Manager driving exec, resolving scenario indexes under
// scenariosRoot. Runs execute under a server-lifetime context independent of
// any request.
func New(exec Executor, scenariosRoot string) *Manager {
	base, cancel := context.WithCancel(context.Background())
	return &Manager{
		base:                base,
		cancelBase:          cancel,
		exec:                exec,
		scenariosRoot:       strings.TrimSpace(scenariosRoot),
		heartbeat:           HeartbeatInterval(),
		maxRunsPerScenario:  maxRunsPerScenarioFromEnv(),
		maxConcurrentRuns:   maxConcurrentRunsFromEnv(),
		maxQueuedRuns:       boundedIntFromEnv("TEST_GENIE_MAX_QUEUED_RUNS", defaultMaxQueuedRuns),
		maxQueuedPerCaller:  boundedIntFromEnv("TEST_GENIE_MAX_QUEUED_RUNS_PER_CALLER", defaultMaxQueuedPerCaller),
		maxPreviewPerCaller: boundedIntFromEnv("TEST_GENIE_MAX_PREVIEW_RUNS_PER_CALLER", defaultMaxPreviewPerCaller),
		previewTokens:       make(chan struct{}, boundedIntFromEnv("TEST_GENIE_MAX_PREVIEW_RUNS", defaultMaxPreviewRuns)),
		previewByCaller:     make(map[string]int),
		runs:                make(map[string]*activeRun),
	}
}

type SaturatedError struct{ Limit string }

func (e *SaturatedError) Error() string {
	return fmt.Sprintf("test-genie admission is saturated (%s); retry after an active run advances or completes", e.Limit)
}

// TryAcquirePreview is a non-blocking gate before plan preview. It prevents a
// burst from creating unbounded preview work or waiting goroutines.
func (m *Manager) TryAcquirePreview() (func(), error) {
	return m.TryAcquirePreviewFor("")
}

// TryAcquirePreviewFor applies both the global and caller-specific preview
// bounds. Empty caller identity intentionally uses the shared "anonymous"
// bucket, which fails closed instead of granting unlimited work to callers
// whose transport cannot supply attribution.
func (m *Manager) TryAcquirePreviewFor(caller string) (func(), error) {
	caller = normalizedCaller(caller)
	m.mu.Lock()
	if m.previewByCaller[caller] >= m.maxPreviewPerCaller {
		m.mu.Unlock()
		atomic.AddUint64(&m.previewRejectedTotal, 1)
		return nil, &SaturatedError{Limit: "caller preview capacity"}
	}
	select {
	case m.previewTokens <- struct{}{}:
		m.previewByCaller[caller]++
		m.mu.Unlock()
		var once sync.Once
		return func() {
			once.Do(func() {
				<-m.previewTokens
				m.mu.Lock()
				m.previewByCaller[caller]--
				if m.previewByCaller[caller] == 0 {
					delete(m.previewByCaller, caller)
				}
				m.mu.Unlock()
			})
		}, nil
	default:
		m.mu.Unlock()
		atomic.AddUint64(&m.previewRejectedTotal, 1)
		return nil, &SaturatedError{Limit: "preview capacity"}
	}
}

func normalizedCaller(caller string) string {
	caller = strings.TrimSpace(caller)
	if caller == "" {
		return "anonymous"
	}
	if len(caller) > 128 {
		return caller[:128]
	}
	return caller
}

// AdmissionStatus reports both the configured bounds and their live usage.
// It intentionally contains no request payload or caller data, so it is safe
// to return from an operator-facing endpoint.
func (m *Manager) AdmissionStatus() AdmissionSnapshot {
	m.mu.Lock()
	snapshot := AdmissionSnapshot{
		Running:                 m.runningCountLocked(),
		Queued:                  m.queuedCountLocked(),
		MaxConcurrentRuns:       m.maxConcurrentRuns,
		MaxQueuedRuns:           m.maxQueuedRuns,
		MaxQueuedRunsPerCaller:  m.maxQueuedPerCaller,
		PreviewInFlight:         len(m.previewTokens),
		MaxPreviewRuns:          cap(m.previewTokens),
		MaxPreviewRunsPerCaller: m.maxPreviewPerCaller,
	}
	m.mu.Unlock()
	snapshot.PreviewRejectedTotal = atomic.LoadUint64(&m.previewRejectedTotal)
	snapshot.QueueRejectedTotal = atomic.LoadUint64(&m.queueRejectedTotal)
	snapshot.ScenarioRejectedTotal = atomic.LoadUint64(&m.scenarioRejectedTotal)
	snapshot.CoalescedTotal = atomic.LoadUint64(&m.coalescedTotal)
	return snapshot
}

// CoalescedRunID performs the cheap identity portion of admission before a
// caller spends a preview token. It preserves identical-request coalescing
// under preview saturation; divergent requests still go through the bounded
// preview path and later full admission.
func (m *Manager) CoalescedRunID(req orchestrator.SuiteExecutionRequest) string {
	scenario := strings.TrimSpace(req.ScenarioName)
	key := admissionKey(req)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, run := range m.runs {
		if run.scenario == scenario && !isTerminal(run.currentStatus()) && run.admissionKey == key {
			atomic.AddUint64(&m.coalescedTotal, 1)
			return run.runID
		}
	}
	return ""
}

func runKey(scenario, runID string) string { return scenario + "\x00" + runID }

// admissionKey is the deterministic coalescing identity of a run request:
// two requests with the same key are the *same* logical run, so a second one
// can ride the first instead of stacking a duplicate suite. The baseline name
// is deliberately NOT part of it (see §6.3 of the plan: many diffs of one
// scenario share a single comprehensive run). FailFast/diagnostics are excluded
// — they don't change which suite executes against which tree.
func admissionKey(req orchestrator.SuiteExecutionRequest) string {
	phases := append([]string(nil), req.Phases...)
	sort.Strings(phases)
	skip := append([]string(nil), req.Skip...)
	sort.Strings(skip)
	parts := []string{
		"scenario=" + strings.TrimSpace(req.ScenarioName),
		"preset=" + strings.TrimSpace(req.Preset),
		"capture=" + strings.TrimSpace(req.CaptureProfile),
		"phases=" + strings.Join(phases, ","),
		"skip=" + strings.Join(skip, ","),
		"scenarioPath=" + strings.TrimSpace(req.ScenarioPath),
		"logicalRepoRoot=" + strings.TrimSpace(req.LogicalRepoRoot),
		"logicalScenarioRelPath=" + strings.TrimSpace(req.LogicalScenarioRelPath),
	}
	return strings.Join(parts, "\x1f")
}

// StartResult reports the outcome of an admitted run.
type StartResult struct {
	// RunID is the run to observe — the freshly-started run, or the in-flight
	// run a coalesced request attached to.
	RunID string
	// Coalesced is true when the request matched an already-in-flight run of the
	// same scenario+key and rode it instead of starting a second suite.
	Coalesced bool
}

// BusyError is returned when a DIVERGENT run (different key) is requested for a
// scenario that already has the maximum in-progress runs. It carries the
// in-flight run so callers can render wait/abort guidance without parsing
// strings. StartRun maps it to connect.CodeFailedPrecondition.
type BusyError struct {
	Scenario string
	RunID    string
	Preset   string
}

func (e *BusyError) Error() string {
	preset := e.Preset
	if preset == "" {
		preset = "(default)"
	}
	return fmt.Sprintf("scenario %s already has an in-progress run %s (preset %s); wait for it or abort it before starting a different run", e.Scenario, e.RunID, preset)
}

// currentStatus reads the run's status under its own lock. Safe to call while
// holding the manager lock (lock order is always m.mu → ar.mu).
func (ar *activeRun) currentStatus() string {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	return ar.status
}

func (m *Manager) lookup(scenario, runID string) *activeRun {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runs[runKey(scenario, runID)]
}

// Start admits a run under the one-run-per-scenario invariant, registers it, and
// drives it in a server-lifetime goroutine, returning a StartResult
// synchronously. The run survives cancellation of the request that initiated it.
//
// Admission (atomic under m.mu — scan + decide + insert, so two concurrent
// Starts cannot both miss):
//   - an identical in-flight request (same admissionKey) COALESCES onto the
//     existing run (StartResult.Coalesced=true); no second suite is driven.
//   - a divergent request, when the scenario is already at its in-progress cap,
//     is REJECTED with *BusyError carrying the in-flight run id + preset.
//   - otherwise a fresh run is minted, registered, and driven.
//
// Terminal runs lingering within retireGrace are ignored: a just-finished run
// never blocks or coalesces a fresh start.
func (m *Manager) Start(opts StartOptions) (StartResult, error) {
	scenario := strings.TrimSpace(opts.Input.Request.ScenarioName)
	if scenario == "" {
		return StartResult{}, fmt.Errorf("scenarioName is required")
	}
	runID := strings.TrimSpace(opts.Input.Request.RunID)
	if runID == "" {
		runID = sharedruns.NewRunID()
	}
	opts.Input.Request.RunID = runID
	preset := strings.TrimSpace(opts.Input.Request.Preset)
	key := admissionKey(opts.Input.Request)
	caller := normalizedCaller(opts.Caller)

	now := time.Now().UTC()
	ar := &activeRun{
		runID:        runID,
		scenario:     scenario,
		preset:       preset,
		admissionKey: key,
		caller:       caller,
		startedAt:    now,
		bc:           newBroadcaster(),
		done:         make(chan struct{}),
		status:       sharedruns.StatusInProgress,
		lastEventAt:  now,
		etaTotal:     opts.EstimatedTotalSeconds,
	}
	runCtx, cancel := context.WithCancel(m.base)
	ar.cancel = cancel
	ar.runCtx = runCtx
	ar.input = opts.Input

	m.mu.Lock()
	// Enumerate in-progress runs of this scenario (ignoring terminal lingerers).
	// Queued runs count as in-flight: a scenario may hold at most one pending OR
	// running run, so the per-scenario serialization invariant holds end to end.
	var inFlight []*activeRun
	for _, other := range m.runs {
		if other.scenario != scenario {
			continue
		}
		if isTerminal(other.currentStatus()) {
			continue
		}
		inFlight = append(inFlight, other)
	}
	// Coalesce onto an identical in-flight request.
	for _, other := range inFlight {
		if other.admissionKey == key {
			m.mu.Unlock()
			cancel()
			atomic.AddUint64(&m.coalescedTotal, 1)
			return StartResult{RunID: other.runID, Coalesced: true}, nil
		}
	}
	// Reject a divergent request once the scenario is at its in-progress cap.
	if len(inFlight) >= m.maxRunsPerScenario {
		busy := oldestRun(inFlight)
		m.mu.Unlock()
		cancel()
		atomic.AddUint64(&m.scenarioRejectedTotal, 1)
		return StartResult{}, &BusyError{Scenario: scenario, RunID: busy.runID, Preset: busy.preset}
	}
	// Defensive: an explicitly pre-minted run id must be unique.
	if _, exists := m.runs[runKey(scenario, runID)]; exists {
		m.mu.Unlock()
		cancel()
		return StartResult{}, fmt.Errorf("run %s is already active", runID)
	}
	// Global concurrency gate: if the host is already running the maximum number
	// of suites (across ALL scenarios, including the background fleet sweep),
	// admit this one as queued rather than rejecting it. The dispatcher promotes
	// it FIFO when a slot frees. Otherwise it starts immediately.
	queued := m.runningCountLocked() >= m.maxConcurrentRuns
	if queued && m.queuedCountLocked() >= m.maxQueuedRuns {
		m.mu.Unlock()
		cancel()
		atomic.AddUint64(&m.queueRejectedTotal, 1)
		return StartResult{}, &SaturatedError{Limit: "queued run capacity"}
	}
	if queued && m.queuedCountForCallerLocked(caller) >= m.maxQueuedPerCaller {
		m.mu.Unlock()
		cancel()
		atomic.AddUint64(&m.queueRejectedTotal, 1)
		return StartResult{}, &SaturatedError{Limit: "caller queued run capacity"}
	}
	if queued {
		ar.setStatus(sharedruns.StatusQueued)
	}
	m.runs[runKey(scenario, runID)] = ar
	m.mu.Unlock()

	if queued {
		// Persist a placeholder so the run is visible in `runs list` while it
		// waits; the executor's own Append replaces it with the full in_progress
		// record on promotion (Append is upsert-by-run-id).
		if err := sharedruns.NewIndex(m.scenarioDir(scenario)).Append(sharedruns.RunRecord{
			RunID:     runID,
			Scenario:  scenario,
			StartedAt: now,
			Status:    sharedruns.StatusQueued,
			Preset:    preset,
		}); err != nil {
			log.Printf("[runmanager] persist queued record %s/%s: %v", scenario, runID, err)
		}
		ar.bc.publish(Event{Kind: EventRunQueued, RunID: runID, Scenario: scenario, Preset: preset})
		return StartResult{RunID: runID}, nil
	}

	// run_started boundary so a follower subscribing immediately learns the id.
	ar.bc.publish(Event{Kind: EventRunStarted, RunID: runID, Scenario: scenario, Preset: preset})

	m.wg.Add(1)
	go m.drive(runCtx, ar, opts.Input)
	return StartResult{RunID: runID}, nil
}

// runningCountLocked returns the number of runs currently executing (status
// in_progress). Queued and terminal-lingering runs are excluded. Caller holds
// m.mu.
func (m *Manager) runningCountLocked() int {
	n := 0
	for _, ar := range m.runs {
		if ar.currentStatus() == sharedruns.StatusInProgress {
			n++
		}
	}
	return n
}

func (m *Manager) queuedCountLocked() int {
	n := 0
	for _, ar := range m.runs {
		if ar.currentStatus() == sharedruns.StatusQueued {
			n++
		}
	}
	return n
}

func (m *Manager) queuedCountForCallerLocked(caller string) int {
	n := 0
	for _, ar := range m.runs {
		if ar.caller == caller && ar.currentStatus() == sharedruns.StatusQueued {
			n++
		}
	}
	return n
}

// setStatus updates the run status under the run lock.
func (ar *activeRun) setStatus(status string) {
	ar.mu.Lock()
	ar.status = status
	ar.mu.Unlock()
}

// dispatch promotes queued runs to in_progress while global slots are free,
// oldest-first (FIFO). A queued run's scenario never has a concurrent running
// run (the per-scenario cap is enforced at admission), so promotion cannot
// violate the per-scenario invariant. Drives are started outside the lock to
// preserve the m.mu -> ar.mu lock order. Called whenever a slot may have freed
// (a run reached a terminal state).
func (m *Manager) dispatch() {
	m.mu.Lock()
	var toStart []*activeRun
	for m.runningCountLocked()+len(toStart) < m.maxConcurrentRuns {
		next := m.oldestQueuedLocked(toStart)
		if next == nil {
			break
		}
		next.mu.Lock()
		next.status = sharedruns.StatusInProgress
		// Re-stamp so elapsed/ETA measure execution, not the time spent queued.
		next.startedAt = time.Now().UTC()
		next.lastEventAt = next.startedAt
		next.mu.Unlock()
		toStart = append(toStart, next)
	}
	m.mu.Unlock()

	for _, ar := range toStart {
		ar.bc.publish(Event{Kind: EventRunStarted, RunID: ar.runID, Scenario: ar.scenario, Preset: ar.preset})
		m.wg.Add(1)
		go m.drive(ar.runCtx, ar, ar.input)
	}
}

// oldestQueuedLocked returns the earliest-started run still in StatusQueued that
// is not already selected for promotion this pass, or nil. Caller holds m.mu.
func (m *Manager) oldestQueuedLocked(exclude []*activeRun) *activeRun {
	var oldest *activeRun
	for _, ar := range m.runs {
		if ar.currentStatus() != sharedruns.StatusQueued {
			continue
		}
		if containsRun(exclude, ar) {
			continue
		}
		if oldest == nil || ar.startedAt.Before(oldest.startedAt) {
			oldest = ar
		}
	}
	return oldest
}

func containsRun(runs []*activeRun, target *activeRun) bool {
	for _, ar := range runs {
		if ar == target {
			return true
		}
	}
	return false
}

// oldestRun returns the earliest-started run in a non-empty slice (the one a
// busy-rejection points the caller at to wait on or abort).
func oldestRun(runs []*activeRun) *activeRun {
	oldest := runs[0]
	for _, ar := range runs[1:] {
		if ar.startedAt.Before(oldest.startedAt) {
			oldest = ar
		}
	}
	return oldest
}

// drive runs the suite to completion, fanning events to followers, then writes
// the terminal boundary and retires the run from the registry.
func (m *Manager) drive(ctx context.Context, ar *activeRun, input execution.SuiteExecutionInput) {
	defer m.wg.Done()
	defer close(ar.done)

	stopHB := make(chan struct{})
	go m.heartbeatLoop(ar, stopHB)

	result, err := m.exec.ExecuteWithEvents(ctx, input, func(ev orchestrator.ExecutionEvent) {
		m.onOrchestratorEvent(ar, ev)
	})
	close(stopHB)

	aborted := ctx.Err() != nil

	ar.mu.Lock()
	ar.result = result
	ar.err = err
	switch {
	case aborted:
		ar.status = sharedruns.StatusAborted
	case err != nil:
		ar.status = sharedruns.StatusFailed
	case result != nil && !result.Success:
		ar.status = sharedruns.StatusFailed
	default:
		ar.status = sharedruns.StatusPassed
	}
	status := ar.status
	elapsed := time.Since(ar.startedAt).Seconds()
	ar.mu.Unlock()

	if aborted {
		// The orchestrator's finalize wrote passed/failed from partial results;
		// override the durable record to aborted (the explicit terminal state).
		m.markIndexAborted(ar.scenario, ar.runID, result)
	} else {
		// Reconcile the durable record to the terminal outcome. Normally the
		// orchestrator's finalize already wrote passed/failed; but when
		// ExecuteWithEvents returns early (e.g. the target scenario never came up,
		// "timeout waiting for runtime URLs"), finalize is skipped and the record
		// is left at in_progress. Without this it would linger as an orphan after
		// the in-memory run retires — the orphan factory behind the 2026-06-21
		// stalled-baseline incident.
		m.reconcileDurableTerminal(ar.scenario, ar.runID, status, result)
	}

	term := Event{
		Kind:           EventRunCompleted,
		ElapsedSeconds: round1(elapsed),
		RunID:          ar.runID,
		Scenario:       ar.scenario,
		Status:         status,
	}
	switch {
	case result != nil:
		term.Success = result.Success && !aborted
		term.Verdict = result.Verdict
		term.ArtifactDir = result.ArtifactDir
	default:
		term.Success = false
	}
	if aborted {
		term.Success = false
		if term.Verdict == "" {
			term.Verdict = "ABORTED"
		}
	}
	if err != nil {
		term.Error = err.Error()
		term.Success = false
	}
	ar.bc.publish(term)
	ar.bc.close()

	m.retire(ar)
	// This run released its concurrency slot; promote any waiters.
	m.dispatch()
}

// onOrchestratorEvent translates a low-level orchestrator event into the
// canonical vocabulary, updates live state, and publishes to followers.
func (m *Manager) onOrchestratorEvent(ar *activeRun, ev orchestrator.ExecutionEvent) {
	ar.mu.Lock()
	if ev.Phase != "" {
		ar.activePhase = ev.Phase
	}
	if ev.Type == orchestrator.EventPhaseStart {
		ar.phaseIndex = ev.PhaseIndex
		ar.phaseTotal = ev.PhaseTotal
	}
	ar.lastEventAt = time.Now()
	elapsed := round1(time.Since(ar.startedAt).Seconds())
	ar.mu.Unlock()

	switch ev.Type {
	case orchestrator.EventPhaseStart:
		ar.bc.publish(Event{Kind: EventPhaseStarted, ElapsedSeconds: elapsed, Phase: ev.Phase, PhaseIndex: ev.PhaseIndex, PhaseTotal: ev.PhaseTotal})
	case orchestrator.EventPhaseEnd:
		kind := EventPhaseCompleted
		if ev.Status != "passed" && ev.Status != "skipped" {
			kind = EventPhaseFailed
		}
		ar.bc.publish(Event{Kind: kind, ElapsedSeconds: elapsed, Phase: ev.Phase, Status: ev.Status, DurationSeconds: ev.DurationSeconds, Error: ev.Error, PhasePresentation: ev.PhasePresentation, FindingsSummary: ev.FindingsSummary})
	case orchestrator.EventObservation, orchestrator.EventProgress:
		if strings.TrimSpace(ev.Message) == "" {
			return
		}
		ar.bc.publish(Event{Kind: EventPhaseProgress, ElapsedSeconds: elapsed, Phase: ev.Phase, Message: ev.Message})
	case orchestrator.EventComplete:
		// The terminal boundary is synthesized in drive() once ExecuteWithEvents
		// returns, so the full result (incl. abort override) is reflected.
	}
}

// heartbeatLoop emits a phase-aware keep-alive when the active phase has gone
// quiet for longer than the heartbeat interval.
func (m *Manager) heartbeatLoop(ar *activeRun, stop <-chan struct{}) {
	if m.heartbeat <= 0 {
		return
	}
	ticker := time.NewTicker(m.heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			ar.mu.Lock()
			phase := ar.activePhase
			quiet := time.Since(ar.lastEventAt)
			elapsed := round1(time.Since(ar.startedAt).Seconds())
			ar.mu.Unlock()
			if quiet < m.heartbeat {
				continue
			}
			ar.bc.publish(Event{Kind: EventPhaseHeartbeat, ElapsedSeconds: elapsed, Phase: phase, QuietSeconds: round1(quiet.Seconds())})
		}
	}
}

// retire drops the run from the registry after a grace window so late callers
// still observe the live terminal snapshot before falling back to the index.
func (m *Manager) retire(ar *activeRun) {
	time.AfterFunc(retireGrace, func() {
		m.mu.Lock()
		delete(m.runs, runKey(ar.scenario, ar.runID))
		m.mu.Unlock()
	})
}

// Follow returns the replayed history plus a live channel of subsequent events.
// Cancelling ctx (client disconnect / detach) stops delivery WITHOUT aborting
// the run. For a run that is no longer active, a terminal snapshot is
// synthesized from the durable index.
func (m *Manager) Follow(ctx context.Context, scenario, runID string) (replay []Event, ch <-chan Event, err error) {
	ar := m.lookup(scenario, runID)
	if ar == nil {
		return m.terminalReplay(scenario, runID)
	}

	hist, live, cancel := ar.bc.subscribe()
	out := make(chan Event, subscriberBuffer)
	go func() {
		defer close(out)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-live:
				if !ok {
					return
				}
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return hist, out, nil
}

// terminalReplay synthesizes a closed event stream for a run that is no longer
// tracked in memory, from its durable index record.
func (m *Manager) terminalReplay(scenario, runID string) ([]Event, <-chan Event, error) {
	rec, err := sharedruns.NewIndex(m.scenarioDir(scenario)).Find(runID)
	if err != nil {
		return nil, nil, err
	}
	closed := make(chan Event)
	close(closed)
	elapsed := 0.0
	if !rec.CompletedAt.IsZero() {
		elapsed = round1(rec.CompletedAt.Sub(rec.StartedAt).Seconds())
	}
	replay := []Event{
		{Kind: EventRunStarted, RunID: rec.RunID, Scenario: rec.Scenario},
		{
			Kind:           EventRunCompleted,
			ElapsedSeconds: elapsed,
			RunID:          rec.RunID,
			Scenario:       rec.Scenario,
			Status:         rec.Status,
			Success:        rec.Status == sharedruns.StatusPassed,
		},
	}
	return replay, closed, nil
}

// Wait blocks until the run reaches a terminal state or ctx is done. On ctx
// timeout/cancel it returns the current (possibly non-terminal) snapshot plus
// ctx.Err(); the run keeps executing.
func (m *Manager) Wait(ctx context.Context, scenario, runID string) (LiveStatus, error) {
	ar := m.lookup(scenario, runID)
	if ar == nil {
		ls, err := m.statusFromIndex(scenario, runID)
		return ls, err
	}
	select {
	case <-ar.done:
		// Terminal reads always come back through the persisted snapshot, even
		// during the retirement grace window. This makes the immediate and
		// post-restart projections identical and catches incomplete finalization.
		return m.terminalStatusFromIndexOrLive(ar), nil
	case <-ctx.Done():
		return m.snapshot(ar), ctx.Err()
	}
}

// Status returns the current snapshot, preferring the live registry and falling
// back to the durable index.
func (m *Manager) Status(scenario, runID string) (LiveStatus, error) {
	if ar := m.lookup(scenario, runID); ar != nil {
		if isTerminal(ar.currentStatus()) {
			return m.terminalStatusFromIndexOrLive(ar), nil
		}
		return m.snapshot(ar), nil
	}
	return m.statusFromIndex(scenario, runID)
}

// Abort cancels an active run and blocks (bounded) until it reaches its
// terminal aborted state, returning the final snapshot. Aborting a run that is
// already terminal is a no-op that returns its snapshot.
func (m *Manager) Abort(scenario, runID string) (LiveStatus, error) {
	// Decide under m.mu so the queued->aborted transition cannot interleave with
	// dispatch promoting the same run (which would race to start a drive and
	// double-close ar.done).
	m.mu.Lock()
	ar := m.runs[runKey(scenario, runID)]
	if ar == nil {
		m.mu.Unlock()
		// No live executor. If the durable record is still non-terminal it is an
		// orphan left by a prior crash/restart whose registry goroutine is gone —
		// downgrade it to aborted here rather than reading back a stale
		// in_progress forever (the startup Sweep only runs at boot). Without this,
		// an orphaned run is unabortable until the next server restart.
		m.downgradeOrphanIndex(scenario, runID)
		return m.statusFromIndex(scenario, runID)
	}
	if ar.currentStatus() == sharedruns.StatusQueued {
		// A queued run has no drive goroutine to honor context cancellation, so
		// retire it directly. The status flip happens under m.mu (so dispatch
		// won't also promote it); the durable write, broadcast, and done-close
		// happen after the unlock. No concurrency slot was held.
		ar.setStatus(sharedruns.StatusAborted)
		m.mu.Unlock()
		ar.cancel()
		m.markIndexAborted(ar.scenario, ar.runID, nil)
		ar.bc.publish(Event{Kind: EventRunCompleted, RunID: ar.runID, Scenario: ar.scenario, Status: sharedruns.StatusAborted, Verdict: "ABORTED"})
		ar.bc.close()
		close(ar.done)
		m.retire(ar)
		return m.terminalStatusFromIndexOrLive(ar), nil
	}
	m.mu.Unlock()
	ar.cancel()
	select {
	case <-ar.done:
	case <-time.After(2 * time.Minute):
	}
	return m.terminalStatusFromIndexOrLive(ar), nil
}

// terminalStatusFromIndexOrLive prefers the canonical persisted snapshot but
// preserves a terminal live result when persistence itself is unavailable.
// The fallback is explicitly degraded; it exists for storage failures and
// narrow executor seams, never as a success-shaped substitute for durability.
func (m *Manager) terminalStatusFromIndexOrLive(ar *activeRun) LiveStatus {
	status, err := m.statusFromIndex(ar.scenario, ar.runID)
	if err == nil {
		return status
	}
	status = m.snapshot(ar)
	status.DegradedReasons = append(status.DegradedReasons, "canonical terminal persistence unavailable: "+err.Error())
	return status
}

// downgradeOrphanIndex marks a non-terminal durable record (in_progress or
// queued) with no live registry counterpart as aborted. It is the same
// transition the startup Sweep applies, exposed so Abort can recover an orphan
// without waiting for a restart. No-op if the record is missing or already
// terminal.
func (m *Manager) downgradeOrphanIndex(scenario, runID string) {
	err := sharedruns.NewIndex(m.scenarioDir(scenario)).Update(runID, func(r *sharedruns.RunRecord) error {
		if isTerminal(r.Status) {
			return nil
		}
		r.Status = sharedruns.StatusAborted
		if r.CompletedAt.IsZero() {
			r.CompletedAt = time.Now().UTC()
		}
		return nil
	})
	if err != nil && err != sharedruns.ErrRunNotFound {
		log.Printf("[runmanager] downgrade orphan %s/%s: %v", scenario, runID, err)
	}
}

// Sweep marks every non-terminal index entry (in_progress or queued) that has no
// live registry counterpart as aborted. Run at startup it downgrades runs
// orphaned by a prior crash/restart — including runs that were still queued
// behind the concurrency cap when the process died. It is idempotent and safe to
// call repeatedly. Returns the number swept.
func (m *Manager) Sweep() (int, error) {
	if m.scenariosRoot == "" {
		return 0, nil
	}
	entries, err := os.ReadDir(m.scenariosRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	swept := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		dir := m.scenarioDir(scenario)
		idx := sharedruns.NewIndex(dir)
		records, err := idx.List()
		if err != nil {
			log.Printf("[runmanager] sweep: list %s: %v", scenario, err)
			continue
		}
		for _, rec := range records {
			if isTerminal(rec.Status) {
				continue
			}
			if m.lookup(scenario, rec.RunID) != nil {
				continue // genuinely active in this process
			}
			runID := rec.RunID
			if err := idx.Update(runID, func(r *sharedruns.RunRecord) error {
				if isTerminal(r.Status) {
					return nil
				}
				r.Status = sharedruns.StatusAborted
				if r.CompletedAt.IsZero() {
					r.CompletedAt = time.Now().UTC()
				}
				return nil
			}); err != nil {
				log.Printf("[runmanager] sweep: mark aborted %s/%s: %v", scenario, runID, err)
				continue
			}
			swept++
		}
	}
	return swept, nil
}

// Shutdown cancels every active run and the manager base context, then waits for
// all drive goroutines to finalize their durable records before returning, so a
// caller (or test) can rely on no further index writes after Shutdown returns.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	runs := make([]*activeRun, 0, len(m.runs))
	for _, ar := range m.runs {
		runs = append(runs, ar)
	}
	m.mu.Unlock()
	for _, ar := range runs {
		ar.cancel()
	}
	m.cancelBase()
	m.wg.Wait()
}

func (m *Manager) markIndexAborted(scenario, runID string, result *orchestrator.SuiteExecutionResult) {
	idx := sharedruns.NewIndex(m.scenarioDir(scenario))
	rec, err := idx.Find(runID)
	if err != nil {
		if err != sharedruns.ErrRunNotFound {
			log.Printf("[runmanager] find aborted %s/%s: %v", scenario, runID, err)
		}
		return
	}
	completedAt := time.Now().UTC()
	if result == nil {
		result = &orchestrator.SuiteExecutionResult{
			RunID:         runID,
			ScenarioName:  scenario,
			StartedAt:     rec.StartedAt,
			CompletedAt:   completedAt,
			Success:       false,
			Verdict:       "ABORTED",
			PlannedPhases: append([]string(nil), rec.PlannedPhases...),
		}
	} else {
		result.Success = false
		result.Verdict = "ABORTED"
		if result.CompletedAt.IsZero() {
			result.CompletedAt = completedAt
		}
	}
	if err := ensureArtifactCatalog(m.scenarioDir(scenario), runID, result.CompletedAt); err != nil {
		result.Warnings = append(result.Warnings, "artifact catalog unavailable: "+err.Error())
	}
	err = idx.Finalize(runID, result, func(r *sharedruns.RunRecord) error {
		r.Status = sharedruns.StatusAborted
		if r.CompletedAt.IsZero() {
			r.CompletedAt = result.CompletedAt
		}
		if len(r.Phases) == 0 {
			r.Phases = compactPhaseRecords(result.Phases)
		}
		return nil
	})
	if err != nil {
		log.Printf("[runmanager] mark aborted %s/%s: %v", scenario, runID, err)
	}
}

// reconcileDurableTerminal sets the durable record to a terminal status when it
// is still non-terminal — the safety net for runs whose executor returned before
// the orchestrator finalized the record (e.g. the target scenario failed to
// start). It republishes through Finalize even when the orchestrator already
// wrote the compact terminal record, ensuring the canonical snapshot and index
// agree. It is a no-op if the record is missing or status is not terminal.
func (m *Manager) reconcileDurableTerminal(scenario, runID, status string, result *orchestrator.SuiteExecutionResult) {
	if !isTerminal(status) {
		return
	}
	idx := sharedruns.NewIndex(m.scenarioDir(scenario))
	rec, findErr := idx.Find(runID)
	if findErr != nil {
		if findErr != sharedruns.ErrRunNotFound {
			log.Printf("[runmanager] find terminal %s/%s: %v", scenario, runID, findErr)
		}
		return
	}
	if result == nil {
		verdict := "FAIL"
		if status == sharedruns.StatusPassed {
			verdict = "PASS"
		}
		result = &orchestrator.SuiteExecutionResult{
			RunID:         runID,
			ScenarioName:  scenario,
			StartedAt:     rec.StartedAt,
			CompletedAt:   time.Now().UTC(),
			Success:       status == sharedruns.StatusPassed,
			Verdict:       verdict,
			PlannedPhases: append([]string(nil), rec.PlannedPhases...),
		}
	} else {
		if result.RunID == "" {
			result.RunID = runID
		}
		if result.ScenarioName == "" {
			result.ScenarioName = scenario
		}
		if result.StartedAt.IsZero() {
			result.StartedAt = rec.StartedAt
		}
		if result.CompletedAt.IsZero() {
			result.CompletedAt = time.Now().UTC()
		}
		if status == sharedruns.StatusFailed {
			result.Success = false
			result.Verdict = "FAIL"
		}
	}
	if err := ensureArtifactCatalog(m.scenarioDir(scenario), runID, result.CompletedAt); err != nil {
		result.Warnings = append(result.Warnings, "artifact catalog unavailable: "+err.Error())
	}
	err := idx.Finalize(runID, result, func(r *sharedruns.RunRecord) error {
		r.Status = status
		if r.CompletedAt.IsZero() {
			r.CompletedAt = result.CompletedAt
			if r.CompletedAt.IsZero() {
				r.CompletedAt = time.Now().UTC()
			}
		}
		if len(r.Phases) == 0 {
			r.Phases = compactPhaseRecords(result.Phases)
		}
		if len(r.PlannedPhases) == 0 {
			r.PlannedPhases = append([]string(nil), result.PlannedPhases...)
		}
		return nil
	})
	if err != nil && err != sharedruns.ErrRunNotFound {
		// Do not fall back to a terminal index-only update: a missing snapshot
		// is incomplete evidence and must not look successfully finalized.
		log.Printf("[runmanager] reconcile terminal snapshot %s/%s: %v", scenario, runID, err)
	}
}

func ensureArtifactCatalog(scenarioDir, runID string, generatedAt time.Time) error {
	if _, err := sharedartifacts.ReadArtifactCatalog(scenarioDir, runID); err == nil {
		return nil
	} else if !errors.Is(err, sharedartifacts.ErrArtifactCatalogNotFound) {
		return err
	}
	var declarations []sharedartifacts.ArtifactPhaseDeclaration
	if snapshot, err := sharedruns.ReadDescriptorSnapshot(scenarioDir, runID); err == nil {
		declarations = make([]sharedartifacts.ArtifactPhaseDeclaration, 0, len(snapshot.Phases))
		for _, descriptor := range snapshot.Phases {
			declarations = append(declarations, sharedartifacts.ArtifactPhaseDeclaration{
				Phase: descriptor.Phase, EvidenceKinds: append([]string(nil), descriptor.EvidenceKinds...),
			})
		}
	}
	_, err := sharedartifacts.RefreshArtifactCatalog(scenarioDir, runID, declarations, generatedAt)
	return err
}

func compactPhaseRecords(results []orchestrator.PhaseExecutionResult) []sharedruns.PhaseRecord {
	records := make([]sharedruns.PhaseRecord, 0, len(results))
	for _, phase := range results {
		records = append(records, sharedruns.PhaseRecord{
			Name: phase.Name, Status: phase.Status, DurationSeconds: phase.DurationSeconds, Comparable: true,
		})
	}
	return records
}

func (m *Manager) scenarioDir(scenario string) string {
	return filepath.Join(m.scenariosRoot, strings.TrimSpace(scenario))
}

func (m *Manager) snapshot(ar *activeRun) LiveStatus {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	terminal := isTerminal(ar.status)
	elapsed := time.Since(ar.startedAt).Seconds()
	if terminal && ar.result != nil && !ar.result.CompletedAt.IsZero() {
		elapsed = ar.result.CompletedAt.Sub(ar.startedAt).Seconds()
	}
	etaKnown := ar.etaTotal > 0
	remaining := 0
	if etaKnown && !terminal {
		remaining = ar.etaTotal - int(elapsed)
		if remaining < 0 {
			remaining = 0
		}
	}
	ls := LiveStatus{
		RunID:                       ar.runID,
		Scenario:                    ar.scenario,
		Status:                      ar.status,
		ActivePhase:                 ar.activePhase,
		PhaseIndex:                  ar.phaseIndex,
		PhaseTotal:                  ar.phaseTotal,
		StartedAt:                   ar.startedAt,
		ElapsedSeconds:              round1(elapsed),
		EstimatedTotalSeconds:       ar.etaTotal,
		EstimatedRemainingSeconds:   remaining,
		ETAKnown:                    etaKnown,
		RecommendedNextCheckSeconds: recommendedNextCheck(ar.status, remaining, etaKnown),
		Active:                      true,
		Result:                      ar.result,
	}
	if ar.result != nil {
		ls.Verdict = ar.result.Verdict
		ls.Success = ar.result.Success && !terminalAborted(ar.status)
		if terminal {
			ls.TerminalPresentations, ls.TerminalFindingsSummaries = terminalMaturity(ar.result)
		}
	}
	if ar.err != nil {
		ls.Error = ar.err.Error()
	}
	return ls
}

func terminalMaturity(result *orchestrator.SuiteExecutionResult) ([]*commonv1.PhasePresentation, []*runspb.PhaseFindingsSummary) {
	if result == nil {
		return nil, nil
	}
	standings := make([]*commonv1.PhasePresentation, 0, len(result.Phases))
	summaries := make([]*runspb.PhaseFindingsSummary, 0, len(result.Phases))
	for _, phase := range result.Phases {
		if phase.PhasePresentation != nil {
			standings = append(standings, phase.PhasePresentation)
		}
		if phase.FindingsSummary != nil {
			summaries = append(summaries, phase.FindingsSummary)
		}
	}
	return standings, summaries
}

func (m *Manager) statusFromIndex(scenario, runID string) (LiveStatus, error) {
	idx := sharedruns.NewIndex(m.scenarioDir(scenario))
	rec, err := idx.Find(runID)
	if err != nil {
		return LiveStatus{}, err
	}
	elapsed := 0.0
	if !rec.CompletedAt.IsZero() {
		elapsed = rec.CompletedAt.Sub(rec.StartedAt).Seconds()
	}
	ls := LiveStatus{
		RunID:                       rec.RunID,
		Scenario:                    rec.Scenario,
		Status:                      rec.Status,
		StartedAt:                   rec.StartedAt,
		ElapsedSeconds:              round1(elapsed),
		Success:                     rec.Status == sharedruns.StatusPassed,
		RecommendedNextCheckSeconds: recommendedNextCheck(rec.Status, 0, false),
		Active:                      false,
	}
	descriptorSnapshot, descriptorErr := sharedruns.ReadDescriptorSnapshot(m.scenarioDir(scenario), runID)
	if descriptorErr == nil {
		ls.DescriptorSnapshot = &descriptorSnapshot
		if rec.DescriptorSnapshotSchemaVersion != 0 && rec.DescriptorSnapshotSchemaVersion != descriptorSnapshot.SchemaVersion {
			ls.DegradedReasons = append(ls.DegradedReasons, "descriptor snapshot schema does not match the run index")
		}
		if rec.DescriptorSnapshotDigest != "" && rec.DescriptorSnapshotDigest != descriptorSnapshot.Digest {
			ls.DegradedReasons = append(ls.DegradedReasons, "descriptor snapshot digest does not match the run index")
		}
	}
	if !isTerminal(rec.Status) {
		return ls, nil
	}
	if descriptorErr != nil {
		if errors.Is(descriptorErr, sharedruns.ErrDescriptorSnapshotNotFound) {
			ls.DegradedReasons = append(ls.DegradedReasons, "legacy run predates descriptor snapshots")
		} else {
			ls.DegradedReasons = append(ls.DegradedReasons, "descriptor snapshot unavailable: "+descriptorErr.Error())
		}
	}
	snapshot, snapshotErr := idx.ReadTerminalSnapshot(runID)
	if snapshotErr != nil {
		reason := "terminal snapshot unavailable: " + snapshotErr.Error()
		if errors.Is(snapshotErr, sharedruns.ErrSnapshotNotFound) {
			reason = "legacy run predates canonical terminal snapshots"
		}
		ls.DegradedReasons = append(ls.DegradedReasons, reason)
		return ls, nil
	}
	var result orchestrator.SuiteExecutionResult
	if err := json.Unmarshal(snapshot.Result, &result); err != nil {
		ls.DegradedReasons = append(ls.DegradedReasons, "terminal snapshot result is corrupt: "+err.Error())
		return ls, nil
	}
	ls.Result = &result
	terminalRecord := snapshot.Run
	// Pins are mutable retention metadata rather than terminal test evidence.
	// Merge the current index generation so a snapshot read never resurrects a
	// stale pin set while every immutable terminal field stays snapshot-owned.
	terminalRecord.Pins = append([]sharedruns.PinRecord(nil), rec.Pins...)
	ls.TerminalRecord = &terminalRecord
	ls.TerminalSnapshotSchemaVersion = snapshot.SchemaVersion
	ls.Verdict = result.Verdict
	ls.Success = result.Success && rec.Status == sharedruns.StatusPassed
	ls.TerminalPresentations, ls.TerminalFindingsSummaries = terminalMaturity(&result)
	return ls, nil
}

// recommendedNextCheck returns the backoff (seconds) a poller should wait before
// re-checking: 0 when terminal, 30 when the ETA is unknown, otherwise the
// remaining estimate clamped to [5, 60].
func recommendedNextCheck(status string, remaining int, etaKnown bool) int {
	if isTerminal(status) {
		return 0
	}
	if !etaKnown {
		return 30
	}
	switch {
	case remaining < 5:
		return 5
	case remaining > 60:
		return 60
	default:
		return remaining
	}
}

func isTerminal(status string) bool {
	switch status {
	case sharedruns.StatusPassed, sharedruns.StatusFailed, sharedruns.StatusAborted:
		return true
	default:
		return false
	}
}

func terminalAborted(status string) bool { return status == sharedruns.StatusAborted }

func round1(v float64) float64 {
	if v < 0 {
		v = 0
	}
	return float64(int64(v*10+0.5)) / 10
}
