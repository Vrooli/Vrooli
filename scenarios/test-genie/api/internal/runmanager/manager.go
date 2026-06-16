package runmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	sharedruns "test-genie/internal/shared/runs"
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
	base               context.Context
	cancelBase         context.CancelFunc
	exec               Executor
	scenariosRoot      string
	heartbeat          time.Duration
	maxRunsPerScenario int

	mu   sync.Mutex
	runs map[string]*activeRun
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
	startedAt    time.Time
	cancel       context.CancelFunc
	bc           *broadcaster
	done         chan struct{}

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
}

// StartOptions configures a run start.
type StartOptions struct {
	Input execution.SuiteExecutionInput
	// EstimatedTotalSeconds is the summed plan-preview estimate, used for the
	// remaining-ETA and recommended-next-check fields. Zero means unknown.
	EstimatedTotalSeconds int
}

// New constructs a Manager driving exec, resolving scenario indexes under
// scenariosRoot. Runs execute under a server-lifetime context independent of
// any request.
func New(exec Executor, scenariosRoot string) *Manager {
	base, cancel := context.WithCancel(context.Background())
	return &Manager{
		base:               base,
		cancelBase:         cancel,
		exec:               exec,
		scenariosRoot:      strings.TrimSpace(scenariosRoot),
		heartbeat:          HeartbeatInterval(),
		maxRunsPerScenario: maxRunsPerScenarioFromEnv(),
		runs:               make(map[string]*activeRun),
	}
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

	now := time.Now().UTC()
	ar := &activeRun{
		runID:        runID,
		scenario:     scenario,
		preset:       preset,
		admissionKey: key,
		startedAt:    now,
		bc:           newBroadcaster(),
		done:         make(chan struct{}),
		status:       sharedruns.StatusInProgress,
		lastEventAt:  now,
		etaTotal:     opts.EstimatedTotalSeconds,
	}
	runCtx, cancel := context.WithCancel(m.base)
	ar.cancel = cancel

	m.mu.Lock()
	// Enumerate in-progress runs of this scenario (ignoring terminal lingerers).
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
			return StartResult{RunID: other.runID, Coalesced: true}, nil
		}
	}
	// Reject a divergent request once the scenario is at its in-progress cap.
	if len(inFlight) >= m.maxRunsPerScenario {
		busy := oldestRun(inFlight)
		m.mu.Unlock()
		cancel()
		return StartResult{}, &BusyError{Scenario: scenario, RunID: busy.runID, Preset: busy.preset}
	}
	// Defensive: an explicitly pre-minted run id must be unique.
	if _, exists := m.runs[runKey(scenario, runID)]; exists {
		m.mu.Unlock()
		cancel()
		return StartResult{}, fmt.Errorf("run %s is already active", runID)
	}
	m.runs[runKey(scenario, runID)] = ar
	m.mu.Unlock()

	// run_started boundary so a follower subscribing immediately learns the id.
	ar.bc.publish(Event{Kind: EventRunStarted, RunID: runID, Scenario: scenario, Preset: preset})

	go m.drive(runCtx, ar, opts.Input)
	return StartResult{RunID: runID}, nil
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
		m.markIndexAborted(ar.scenario, ar.runID)
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
		ar.bc.publish(Event{Kind: kind, ElapsedSeconds: elapsed, Phase: ev.Phase, Status: ev.Status, DurationSeconds: ev.DurationSeconds, Error: ev.Error})
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
		return m.snapshot(ar), nil
	case <-ctx.Done():
		return m.snapshot(ar), ctx.Err()
	}
}

// Status returns the current snapshot, preferring the live registry and falling
// back to the durable index.
func (m *Manager) Status(scenario, runID string) (LiveStatus, error) {
	if ar := m.lookup(scenario, runID); ar != nil {
		return m.snapshot(ar), nil
	}
	return m.statusFromIndex(scenario, runID)
}

// Abort cancels an active run and blocks (bounded) until it reaches its
// terminal aborted state, returning the final snapshot. Aborting a run that is
// already terminal is a no-op that returns its snapshot.
func (m *Manager) Abort(scenario, runID string) (LiveStatus, error) {
	ar := m.lookup(scenario, runID)
	if ar == nil {
		return m.statusFromIndex(scenario, runID)
	}
	ar.cancel()
	select {
	case <-ar.done:
	case <-time.After(2 * time.Minute):
	}
	return m.snapshot(ar), nil
}

// Sweep marks every in_progress index entry that has no live registry counterpart
// as aborted. Run at startup it downgrades runs orphaned by a prior crash/restart;
// it is idempotent and safe to call repeatedly. Returns the number swept.
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
			if rec.Status != sharedruns.StatusInProgress {
				continue
			}
			if m.lookup(scenario, rec.RunID) != nil {
				continue // genuinely active in this process
			}
			runID := rec.RunID
			if err := idx.Update(runID, func(r *sharedruns.RunRecord) error {
				if r.Status != sharedruns.StatusInProgress {
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

// Shutdown cancels every active run (best-effort) and the manager base context.
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
}

func (m *Manager) markIndexAborted(scenario, runID string) {
	err := sharedruns.NewIndex(m.scenarioDir(scenario)).Update(runID, func(r *sharedruns.RunRecord) error {
		r.Status = sharedruns.StatusAborted
		if r.CompletedAt.IsZero() {
			r.CompletedAt = time.Now().UTC()
		}
		return nil
	})
	if err != nil {
		log.Printf("[runmanager] mark aborted %s/%s: %v", scenario, runID, err)
	}
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
	}
	if ar.err != nil {
		ls.Error = ar.err.Error()
	}
	return ls
}

func (m *Manager) statusFromIndex(scenario, runID string) (LiveStatus, error) {
	rec, err := sharedruns.NewIndex(m.scenarioDir(scenario)).Find(runID)
	if err != nil {
		return LiveStatus{}, err
	}
	elapsed := 0.0
	if !rec.CompletedAt.IsZero() {
		elapsed = rec.CompletedAt.Sub(rec.StartedAt).Seconds()
	}
	return LiveStatus{
		RunID:                       rec.RunID,
		Scenario:                    rec.Scenario,
		Status:                      rec.Status,
		StartedAt:                   rec.StartedAt,
		ElapsedSeconds:              round1(elapsed),
		Success:                     rec.Status == sharedruns.StatusPassed,
		RecommendedNextCheckSeconds: recommendedNextCheck(rec.Status, 0, false),
		Active:                      false,
	}, nil
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
