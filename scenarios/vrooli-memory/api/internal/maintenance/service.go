// Package maintenance owns the in-process projection/import refresh loop.
// It deliberately calls the harness services directly: lifecycle commands are
// supervisor-facing and must never be used as an internal scheduler API.
package maintenance

import (
	"context"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"vrooli-memory/internal/harness"
	"vrooli-memory/internal/ledgerclient"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

const (
	IntervalEnv     = "VROOLI_MEMORY_MAINTENANCE_INTERVAL"
	DefaultInterval = 6 * time.Hour
	RuntimeTimeout  = 2 * time.Minute

	// CompactLimitEnv bounds how many clusters one scheduled pass compacts.
	// Compaction is the only maintenance step whose cost scales with the
	// backlog rather than with the number of runtimes, so it needs its own
	// bound and its own timeout. Zero disables scheduled compaction.
	CompactLimitEnv = "VROOLI_MEMORY_MAINTENANCE_COMPACT_LIMIT"
	// The source-ledger API uses the shared api-core 30-second write timeout.
	// A small default keeps one synchronous bounded pass inside that transport
	// envelope while callers may raise it explicitly when they own a longer
	// request path.
	DefaultCompactLimit = 2
	CompactTimeout      = 30 * time.Minute
)

type (
	Outcome struct {
		Runtime, ImportStatus, ImportError, ProjectionStatus, ProjectionError string
		ProjectionSizeBytes, ProjectionSizeLines                              int64
		ProjectionByteCap, ProjectionLineCap                                  int64
		StartedAt, CompletedAt                                                time.Time
	}
	// Compaction is run-level, not per-runtime: the canopy belongs to the
	// corpus, not to any coding harness that feeds it.
	Compaction struct {
		Status, Error                                    string
		Compacted, FrontierBefore, FrontierAfter, Target int
	}
	CompactionResult = ledgerclient.CompactionResult
	Run              struct {
		ID                     string
		StartedAt, CompletedAt time.Time
		Outcomes               []Outcome
		Compaction             Compaction
	}
	Store interface {
		Begin(context.Context, Run) error
		PutOutcome(context.Context, string, Outcome) error
		PutCompaction(context.Context, string, Compaction) error
		Complete(context.Context, string, time.Time) error
		Latest(context.Context) (Run, error)
	}
	// Compactor is the forest seam. The maintenance loop never reaches into
	// the forest repository directly.
	Compactor interface {
		RunBounded(context.Context, int) (CompactionResult, error)
	}
	Importer interface {
		Runtimes() []string
		Import(context.Context, string, bool) (harness.ImportResult, error)
	}
	Projector interface {
		Runtimes() []string
		Project(context.Context, string, bool) (harness.ProjectionResult, error)
	}
	Service struct {
		store        Store
		importer     Importer
		projector    Projector
		compactor    Compactor
		clock        schedule.Clock
		interval     time.Duration
		compactLimit int
		running      atomic.Bool
	}
)

func NewService(store Store, importer Importer, projector Projector, clk schedule.Clock, interval time.Duration) *Service {
	if clk == nil {
		clk = schedule.System()
	}
	return &Service{store: store, importer: importer, projector: projector, clock: clk, interval: interval, compactLimit: DefaultCompactLimit}
}

// WithCompaction adds the scheduled compaction step. It is separate from
// NewService so a caller that only refreshes harness stores stays unchanged.
func (s *Service) WithCompaction(compactor Compactor, limit int) *Service {
	s.compactor = compactor
	s.compactLimit = limit
	return s
}

func CompactLimitFromEnv(getenv func(string) (string, bool)) (int, error) {
	raw, ok := getenv(CompactLimitEnv)
	if !ok || strings.TrimSpace(raw) == "" {
		return DefaultCompactLimit, nil
	}
	return strconv.Atoi(strings.TrimSpace(raw))
}

func CompactLimitFromOS() (int, error) { return CompactLimitFromEnv(os.LookupEnv) }

func IntervalFromEnv(getenv func(string) (string, bool)) (time.Duration, error) {
	raw, ok := getenv(IntervalEnv)
	if !ok || strings.TrimSpace(raw) == "" {
		return DefaultInterval, nil
	}
	if strings.TrimSpace(raw) == "0" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}

func IntervalFromOS() (time.Duration, error) { return IntervalFromEnv(os.LookupEnv) }

func (s *Service) Start(ctx context.Context) {
	if s.interval <= 0 {
		return
	}
	go func() {
		// A restart refreshes stores immediately; subsequent runs are ticker-led.
		_, _ = s.RunOnce(ctx)
		ticker := s.clock.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
				_, _ = s.RunOnce(ctx)
			}
		}
	}()
}

func (s *Service) RunOnce(ctx context.Context) (bool, error) {
	if !s.running.CompareAndSwap(false, true) {
		return false, nil
	}
	defer s.running.Store(false)

	started := s.clock.Now().UTC()
	run := Run{ID: uuid.NewString(), StartedAt: started}
	if err := s.store.Begin(ctx, run); err != nil {
		return true, err
	}
	runtimes := union(s.importer.Runtimes(), s.projector.Runtimes())
	outcomes := make(map[string]Outcome, len(runtimes))
	for _, runtime := range runtimes {
		outcomes[runtime] = Outcome{Runtime: runtime, ImportStatus: "not_configured", ProjectionStatus: "not_configured", StartedAt: started}
	}
	for _, runtime := range s.importer.Runtimes() {
		o := outcomes[runtime]
		runCtx, cancel := context.WithTimeout(ctx, RuntimeTimeout)
		_, err := s.importer.Import(runCtx, runtime, false)
		cancel()
		if err != nil {
			o.ImportStatus, o.ImportError = "failed", err.Error()
		} else {
			o.ImportStatus = "completed"
		}
		outcomes[runtime] = o
		if err := s.store.PutOutcome(ctx, run.ID, o); err != nil {
			return true, err
		}
	}
	for _, runtime := range s.projector.Runtimes() {
		o := outcomes[runtime]
		runCtx, cancel := context.WithTimeout(ctx, RuntimeTimeout)
		projection, err := s.projector.Project(runCtx, runtime, false)
		cancel()
		o.ProjectionSizeBytes = projection.SizeBytes
		o.ProjectionSizeLines = projection.SizeLines
		o.ProjectionByteCap = projection.ByteCap
		o.ProjectionLineCap = projection.LineCap
		if err != nil {
			o.ProjectionStatus, o.ProjectionError = "failed", err.Error()
		} else {
			o.ProjectionStatus = "completed"
		}
		o.CompletedAt = s.clock.Now().UTC()
		outcomes[runtime] = o
		if err := s.store.PutOutcome(ctx, run.ID, o); err != nil {
			return true, err
		}
	}
	// Compaction runs last. It is the only step whose duration scales with the
	// backlog, so putting it ahead of projection would hold every harness memory
	// file hostage to a long catch-up pass. The canopy this pass grows is picked
	// up by the next tick's projection; ambient memory is never blocked on it.
	compaction, err := s.compact(ctx, run.ID)
	if err != nil {
		return true, err
	}
	run.Compaction = compaction

	completed := s.clock.Now().UTC()
	for _, runtime := range runtimes {
		o := outcomes[runtime]
		if o.CompletedAt.IsZero() {
			o.CompletedAt = completed
		}
		if err := s.store.PutOutcome(ctx, run.ID, o); err != nil {
			return true, err
		}
	}
	return true, s.store.Complete(ctx, run.ID, completed)
}

// compact runs one bounded pass and records its outcome. A compaction failure
// never fails the maintenance run: import and projection are independent of the
// canopy, and a provider outage must not stop ambient memory from refreshing.
// The error is recorded so a stalled canopy is visible instead of silent.
func (s *Service) compact(ctx context.Context, runID string) (Compaction, error) {
	if s.compactor == nil || s.compactLimit == 0 {
		out := Compaction{Status: "not_configured"}
		return out, s.store.PutCompaction(ctx, runID, out)
	}
	runCtx, cancel := context.WithTimeout(ctx, CompactTimeout)
	defer cancel()
	result, err := s.compactor.RunBounded(runCtx, s.compactLimit)
	out := Compaction{
		Status:         "completed",
		Compacted:      result.CompactedCount,
		FrontierBefore: result.EligibleFrontierBefore,
		FrontierAfter:  result.EligibleFrontierAfter,
		Target:         result.Target,
	}
	if err != nil {
		out.Status, out.Error = "failed", err.Error()
	}
	return out, s.store.PutCompaction(ctx, runID, out)
}

func (s *Service) Latest(ctx context.Context) (Run, error) { return s.store.Latest(ctx) }

func union(groups ...[]string) []string {
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, item := range group {
			seen[item] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for item := range seen {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
