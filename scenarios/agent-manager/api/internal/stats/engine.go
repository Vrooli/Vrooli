// Stats aggregation engine.
//
// The Engine reads typed-operational records from internal/eventlog and
// folds them into an in-memory aggregateState that the public Get*
// methods read from. State is rebuildable from scratch (Rebuild) and
// can be advanced incrementally (Refresh) using a persisted watermark
// (CheckpointStore) so a crash mid-rebuild resumes from the last
// checkpoint, not from zero.
//
// Concurrency model:
//   - One Engine per process (constructed in main.go and shared).
//   - Reads (GetSummary, GetFallback, GetHealth, …) take the RLock.
//   - Refresh / Rebuild take the write lock for the full operation.
//   - Refresh is safe to call concurrently with reads (the lock pattern
//     guarantees a consistent snapshot per read).

package stats

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"agent-manager/internal/eventlog"
)

// refreshBatchSize bounds each batched read out of the repository so a
// large catch-up does not allocate a single giant slice. The number is
// the same as swarm-manager's, which was load-tested at the time of
// the swarm-manager port.
const refreshBatchSize = 5000

// CheckpointStore persists the engine's last-processed rowid so a
// crash or restart resumes from the checkpoint instead of replaying
// every event from the beginning.
//
// The "name" disambiguates engines that might one day share the same
// store (e.g., a future per-tenant stats engine); for now main.go
// passes a single fixed name.
type CheckpointStore interface {
	Load(ctx context.Context, name string) (int64, error)
	Save(ctx context.Context, name string, rowid int64) error
}

// Engine incrementally aggregates typed-operational events into the
// metrics consumed by handlers and CLI.
type Engine struct {
	mu             sync.RWMutex
	repo           eventlog.Repository
	checkpoint     CheckpointStore
	checkpointName string

	watermark int64
	state     *aggregateState
	now       func() time.Time
}

// NewEngine wires an engine to an event repository and a checkpoint
// store. The checkpoint may be nil for tests or in-memory engines; the
// engine then replays from rowid 0 each Rebuild and persists nothing.
func NewEngine(repo eventlog.Repository, checkpoint CheckpointStore, checkpointName string) *Engine {
	return &Engine{
		repo:           repo,
		checkpoint:     checkpoint,
		checkpointName: checkpointName,
		state:          newAggregateState(),
		now:            time.Now,
	}
}

// Rebuild resumes from the saved checkpoint (if any) and processes
// events forward in batches. Used at boot and as the recovery path
// after a crash.
//
// The "from-checkpoint" semantics mean Rebuild is the same operation
// as Refresh from the engine's point of view; the difference is only
// that Rebuild resets in-memory state first, so a corrupted snapshot
// can be recovered without restarting the process.
func (e *Engine) Rebuild(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	startRowid := int64(0)
	if e.checkpoint != nil {
		saved, err := e.checkpoint.Load(ctx, e.checkpointName)
		if err != nil {
			return fmt.Errorf("stats rebuild: load checkpoint: %w", err)
		}
		startRowid = saved
	}

	e.state = newAggregateState()
	e.watermark = startRowid

	return e.advanceLocked(ctx)
}

// Refresh advances the watermark by processing events appended since
// the last Refresh / Rebuild. Safe to call frequently; it short-circuits
// when no new events are present.
func (e *Engine) Refresh(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.advanceLocked(ctx)
}

// advanceLocked is the inner loop shared by Rebuild and Refresh.
// Caller must hold e.mu (write lock).
func (e *Engine) advanceLocked(ctx context.Context) error {
	for {
		records, err := e.repo.SinceID(ctx, e.watermark, refreshBatchSize)
		if err != nil {
			return fmt.Errorf("stats advance: %w", err)
		}
		if len(records) == 0 {
			return nil
		}
		for i := range records {
			rec := records[i]
			processor := lookupProcessor(rec.EventType, rec.SchemaVersion)
			if processor == nil {
				slog.Warn("stats: no processor for event",
					"event_type", rec.EventType,
					"schema_version", rec.SchemaVersion,
					"event_id", rec.ID,
				)
				continue
			}
			processor(e.state, rec)
			e.state.totalEvents++
			if !e.state.earliestRecorded || rec.Timestamp.Before(e.state.earliestEventAt) {
				e.state.earliestEventAt = rec.Timestamp
				e.state.earliestRecorded = true
			}
		}
		// run_events.rowid is monotonic for the SQLite backend, so the
		// last record in the batch carries the new watermark. The
		// repository's SinceID query orders by rowid asc, so this is
		// safe even with mixed-event-type batches.
		e.watermark = records[len(records)-1].Rowid

		if e.checkpoint != nil {
			if err := e.checkpoint.Save(ctx, e.checkpointName, e.watermark); err != nil {
				// A failed checkpoint write is not fatal — we'll redo
				// these events on next boot — but it is operator-
				// significant.
				slog.Warn("stats: checkpoint save failed",
					"name", e.checkpointName,
					"watermark", e.watermark,
					"error", err.Error(),
				)
			}
		}

		if len(records) < refreshBatchSize {
			return nil
		}
	}
}

// Watermark returns the engine's current cursor. Exposed for tests.
func (e *Engine) Watermark() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.watermark
}

// EventCount returns the total number of processed events. Exposed
// primarily for tests; production reads should consume one of the
// Get* methods.
func (e *Engine) EventCount() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.totalEvents
}

// ----- aggregateState -----

type fallbackPairKey struct {
	From   string
	To     string
	Reason string
}

type modelKey struct {
	Runner string
	Model  string
}

// aggregateState holds every counter/map the processors mutate. The
// fields are flat by design — each (event, schema_version) processor
// owns one or more fields explicitly, no shared "metadata bag".
type aggregateState struct {
	totalEvents      int64
	earliestEventAt  time.Time
	earliestRecorded bool

	// Runner fallback.
	runnerFallbackAttempts int
	runnerExhausted        int
	runnerByReason         map[string]int
	runnerPair             map[fallbackPairKey]int
	runnerChainDepth       map[int]int

	// Model fallback.
	modelFallbackAttempts int
	modelExhausted        int
	modelByReason         map[string]int
	modelPair             map[fallbackPairKey]int
	modelChainDepth       map[int]int
	modelByPreset         map[string]int

	// Health (current snapshot derived from transitions).
	modelHealth  map[modelKey]ModelHealthEntry
	runnerHealth map[string]RunnerHealthEntry

	// Sandbox.
	sandboxTotal         int
	sandboxSuccess       int
	sandboxByOp          map[string]OperationCount
	sandboxDurationSum   float64
	sandboxDurationCount int

	// Heartbeat.
	heartbeatMisses   int
	heartbeatByTarget map[string]int

	// Checkpoint.
	checkpointFailures int
	checkpointByStep   map[string]int
	checkpointByPhase  map[string]int

	// Retry.
	retryAttempts    int
	retryByOperation map[string]int
	retryByReason    map[string]int
}

func newAggregateState() *aggregateState {
	return &aggregateState{
		runnerByReason:    make(map[string]int),
		runnerPair:        make(map[fallbackPairKey]int),
		runnerChainDepth:  make(map[int]int),
		modelByReason:     make(map[string]int),
		modelPair:         make(map[fallbackPairKey]int),
		modelChainDepth:   make(map[int]int),
		modelByPreset:     make(map[string]int),
		modelHealth:       make(map[modelKey]ModelHealthEntry),
		runnerHealth:      make(map[string]RunnerHealthEntry),
		sandboxByOp:       make(map[string]OperationCount),
		heartbeatByTarget: make(map[string]int),
		checkpointByStep:  make(map[string]int),
		checkpointByPhase: make(map[string]int),
		retryByOperation:  make(map[string]int),
		retryByReason:     make(map[string]int),
	}
}

