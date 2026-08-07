// Package maintenance owns the in-process projection/import refresh loop.
// It deliberately calls the harness services directly: lifecycle commands are
// supervisor-facing and must never be used as an internal scheduler API.
package maintenance

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"vrooli-memory/internal/clock"
	"vrooli-memory/internal/harness"

	"github.com/google/uuid"
)

const (
	IntervalEnv     = "VROOLI_MEMORY_MAINTENANCE_INTERVAL"
	DefaultInterval = 6 * time.Hour
	RuntimeTimeout  = 2 * time.Minute
)

type (
	Outcome struct {
		Runtime, ImportStatus, ImportError, ProjectionStatus, ProjectionError string
		StartedAt, CompletedAt                                                time.Time
	}
	Run struct {
		ID                     string
		StartedAt, CompletedAt time.Time
		Outcomes               []Outcome
	}
	Store interface {
		Begin(context.Context, Run) error
		PutOutcome(context.Context, string, Outcome) error
		Complete(context.Context, string, time.Time) error
		Latest(context.Context) (Run, error)
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
		store     Store
		importer  Importer
		projector Projector
		clock     clock.Clock
		interval  time.Duration
		running   atomic.Bool
	}
)

func NewService(store Store, importer Importer, projector Projector, clk clock.Clock, interval time.Duration) *Service {
	if clk == nil {
		clk = clock.System{}
	}
	return &Service{store: store, importer: importer, projector: projector, clock: clk, interval: interval}
}

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
		_, err := s.projector.Project(runCtx, runtime, false)
		cancel()
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
