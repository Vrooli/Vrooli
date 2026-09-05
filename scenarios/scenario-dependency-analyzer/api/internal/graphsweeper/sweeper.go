package graphsweeper

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/graphingest"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/freshness-go/treedigest"
)

// Ingestor re-ingests one scenario's edges into the unified store.
type Ingestor interface {
	IngestScenario(ctx context.Context, repoRoot, scenario string, apply bool) (graphingest.ScenarioResult, error)
}

// DigestStore persists per-scenario ingest digests for freshness gating.
type DigestStore interface {
	GetIngestDigest(scenario string) (string, bool, error)
	SetIngestDigest(scenario, digest string) error
}

// Scenario is one sweepable unit.
type Scenario struct {
	Name string
	Root string
}

// Lister enumerates sweepable scenarios under a root.
type Lister interface {
	ListScenarios(root string) ([]Scenario, error)
}

// Digester computes a scenario's working-tree content digest.
type Digester interface {
	ComputeDigest(root string) (string, error)
}

// DirectoryLister lists scenarios that carry a .vrooli/service.json.
type DirectoryLister struct{}

// ListScenarios returns each immediate subdirectory of root that is a scenario.
func (DirectoryLister) ListScenarios(root string) ([]Scenario, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]Scenario, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "service.json")); err != nil {
			continue
		}
		out = append(out, Scenario{Name: entry.Name(), Root: dir})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// TreeDigester computes digests via freshness-go's treedigest.
type TreeDigester struct{}

// ComputeDigest delegates to treedigest.Compute.
func (TreeDigester) ComputeDigest(root string) (string, error) { return treedigest.Compute(root) }

// Report is the outcome of one sweep cycle.
type Report struct {
	StartedAt       time.Time `json:"started_at"`
	DurationMs      int64     `json:"duration_ms"`
	Scanned         int       `json:"scanned"`
	SkippedFresh    int       `json:"skipped_fresh"`
	Ingested        int       `json:"ingested"`
	Degraded        int       `json:"degraded"`
	Failed          int       `json:"failed"`
	BreakerSkipped  int       `json:"breaker_skipped"`
	BudgetHit       bool      `json:"budget_hit"`
	BreakerState    string    `json:"breaker_state"`
	DegradedSources []string  `json:"degraded_sources,omitempty"`
}

// Sweeper owns the freshness-gated ingest loop.
type Sweeper struct {
	cfg      Config
	ingestor Ingestor
	digests  DigestStore
	lister   Lister
	digester Digester
	clock    Clock
	breaker  *breaker

	mu         sync.Mutex
	running    bool
	lastReport *Report
	lastRunAt  time.Time
}

// Option customizes a Sweeper.
type Option func(*Sweeper)

// WithClock overrides the time seam.
func WithClock(c Clock) Option { return func(s *Sweeper) { s.clock = c } }

// WithLister overrides the scenario lister.
func WithLister(l Lister) Option { return func(s *Sweeper) { s.lister = l } }

// WithDigester overrides the digester.
func WithDigester(d Digester) Option { return func(s *Sweeper) { s.digester = d } }

// New constructs a Sweeper with explicit seams.
func New(cfg Config, ingestor Ingestor, digests DigestStore, opts ...Option) *Sweeper {
	s := &Sweeper{
		cfg:      cfg,
		ingestor: ingestor,
		digests:  digests,
		lister:   DirectoryLister{},
		digester: TreeDigester{},
		clock:    systemClock{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	s.breaker = newBreaker(cfg.BreakerThreshold, cfg.BreakerCooldown, s.clock)
	return s
}

// RunLoop runs the sweeper on its configured cadence until ctx is cancelled.
// Cycles are single-flighted; a slow cycle never overlaps the next tick.
func (s *Sweeper) RunLoop(ctx context.Context) {
	if !s.cfg.Enabled {
		return
	}
	if s.cfg.StartJitter > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.cfg.StartJitter):
		}
	}
	s.runOnceGuarded(ctx)
	if s.cfg.Interval <= 0 {
		return
	}
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnceGuarded(ctx)
		}
	}
}

func (s *Sweeper) runOnceGuarded(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	report := s.RunOnce(ctx)
	s.mu.Lock()
	s.lastReport = &report
	s.lastRunAt = report.StartedAt
	s.mu.Unlock()
}

// RunOnce executes a single sweep cycle: list scenarios, skip the fresh ones,
// re-ingest the changed ones under the concurrency cap, wall-clock budget, and
// circuit breaker. It is safe to call directly (used by tests and the status RPC).
func (s *Sweeper) RunOnce(ctx context.Context) Report {
	start := s.clock.Now()
	report := Report{StartedAt: start}
	collector := metrics.Start()

	scenarios, err := s.lister.ListScenarios(s.cfg.ScenariosRoot)
	if err != nil {
		log.Printf("graph-sweeper: list scenarios: %v", err)
		report.DurationMs = s.clock.Now().Sub(start).Milliseconds()
		report.BreakerState = s.breaker.State()
		collector.Stop()
		return report
	}

	concurrency := s.cfg.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		degraded = map[string]struct{}{}
	)

	for _, sc := range scenarios {
		if ctx.Err() != nil {
			break
		}
		if s.cfg.CycleBudget > 0 && s.clock.Now().Sub(start) >= s.cfg.CycleBudget {
			mu.Lock()
			report.BudgetHit = true
			mu.Unlock()
			break
		}
		mu.Lock()
		report.Scanned++
		mu.Unlock()

		wg.Add(1)
		sem <- struct{}{}
		go func(sc Scenario) {
			defer wg.Done()
			defer func() { <-sem }()

			digest, _ := s.digester.ComputeDigest(sc.Root)
			if digest != "" {
				if last, ok, _ := s.digests.GetIngestDigest(sc.Name); ok && last == digest {
					mu.Lock()
					report.SkippedFresh++
					mu.Unlock()
					return
				}
			}
			if !s.breaker.Allow() {
				mu.Lock()
				report.BreakerSkipped++
				mu.Unlock()
				return
			}
			res, err := s.ingestor.IngestScenario(ctx, s.cfg.RepoRoot, sc.Name, true)
			if err != nil {
				s.breaker.Failure()
				mu.Lock()
				report.Failed++
				if res.Degraded {
					report.Degraded++
					degraded["upstream"] = struct{}{}
				}
				mu.Unlock()
				return
			}
			s.breaker.Success()
			if digest != "" {
				if setErr := s.digests.SetIngestDigest(sc.Name, digest); setErr != nil {
					log.Printf("graph-sweeper: set ingest digest for %s: %v", sc.Name, setErr)
				}
			}
			mu.Lock()
			report.Ingested++
			mu.Unlock()
		}(sc)
	}
	wg.Wait()

	report.BreakerState = s.breaker.State()
	report.DegradedSources = sortedKeys(degraded)
	report.DurationMs = s.clock.Now().Sub(start).Milliseconds()

	collector.Gauge("scanned", float64(report.Scanned))
	collector.Gauge("skipped_fresh", float64(report.SkippedFresh))
	collector.Gauge("ingested", float64(report.Ingested))
	collector.Gauge("degraded", float64(report.Degraded))
	collector.Stop()
	return report
}

// Status is the sweeper observability surface.
type Status struct {
	Enabled      bool          `json:"enabled"`
	Interval     time.Duration `json:"interval"`
	Concurrency  int           `json:"concurrency"`
	CycleBudget  time.Duration `json:"cycle_budget"`
	BreakerState string        `json:"breaker_state"`
	LastRunAt    *time.Time    `json:"last_run_at,omitempty"`
	LastCycle    *Report       `json:"last_cycle,omitempty"`
}

// Status reports the sweeper's current configuration and last-cycle outcome.
func (s *Sweeper) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := Status{
		Enabled:      s.cfg.Enabled,
		Interval:     s.cfg.Interval,
		Concurrency:  s.cfg.Concurrency,
		CycleBudget:  s.cfg.CycleBudget,
		BreakerState: s.breaker.State(),
		LastCycle:    s.lastReport,
	}
	if !s.lastRunAt.IsZero() {
		t := s.lastRunAt
		status.LastRunAt = &t
	}
	return status
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
