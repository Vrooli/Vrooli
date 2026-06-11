package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/freshness-go/treedigest"
)

var ErrSweepInProgress = errors.New("score sweep already in progress")

// ScenarioRef identifies one scenario directory under the fleet root.
type ScenarioRef struct {
	Name string
	Root string
}

// seam: ScenarioLister enumerates scenarios for the score sweeper. Production
// wires DirectoryScenarioLister; tests wire fakes with deterministic fleets.
type ScenarioLister interface {
	ListScenarios(root string) ([]ScenarioRef, error)
}

// DirectoryScenarioLister lists first-level directories under scenariosRoot.
type DirectoryScenarioLister struct{}

var _ ScenarioLister = DirectoryScenarioLister{}

func (DirectoryScenarioLister) ListScenarios(root string) ([]ScenarioRef, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	scenarios := make([]ScenarioRef, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		scenarios = append(scenarios, ScenarioRef{
			Name: name,
			Root: filepath.Join(root, name),
		})
	}
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].Name < scenarios[j].Name
	})
	return scenarios, nil
}

// seam: DigestComputer computes a scenario tree digest before scoring. The
// sweeper uses this to skip unchanged scenarios without running score math.
type DigestComputer interface {
	ComputeDigest(root string) (string, error)
}

type TreeDigestComputer struct{}

var _ DigestComputer = TreeDigestComputer{}

func (TreeDigestComputer) ComputeDigest(root string) (string, error) {
	return treedigest.Compute(root)
}

// seam: SweepScorer computes one score for the sweeper. Production wires a
// scoring Service with optional network enrichment disabled; tests wire fakes.
type SweepScorer interface {
	GetScore(scenario string) (Result, error)
}

// SweeperConfig controls score history collection.
type SweeperConfig struct {
	ScenariosRoot string
	Repository    SnapshotRepository
	Scorer        SweepScorer
	Lister        ScenarioLister
	Digester      DigestComputer
	Now           func() time.Time
	Logger        *log.Logger
	Concurrency   int
	Interval      time.Duration
	InitialJitter time.Duration
}

// SweepReport summarizes one fleet sweep.
type SweepReport struct {
	Scanned   int
	Skipped   int
	Scored    int
	Persisted int
	Failed    int
	Duration  time.Duration
}

// Sweeper writes digest-deduplicated score snapshots in the background.
type Sweeper struct {
	cfg     SweeperConfig
	running atomic.Bool
}

func NewSweeper(cfg SweeperConfig) (*Sweeper, error) {
	if strings.TrimSpace(cfg.ScenariosRoot) == "" {
		return nil, errors.New("scenarios root is required")
	}
	if cfg.Repository == nil {
		return nil, errors.New("snapshot repository is required")
	}
	if cfg.Scorer == nil {
		return nil, errors.New("sweep scorer is required")
	}
	if cfg.Lister == nil {
		cfg.Lister = DirectoryScenarioLister{}
	}
	if cfg.Digester == nil {
		cfg.Digester = TreeDigestComputer{}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	return &Sweeper{cfg: cfg}, nil
}

func (s *Sweeper) RunOnce(ctx context.Context) (SweepReport, error) {
	if !s.running.CompareAndSwap(false, true) {
		return SweepReport{}, ErrSweepInProgress
	}
	defer s.running.Store(false)

	start := s.cfg.Now()
	scenarios, err := s.cfg.Lister.ListScenarios(s.cfg.ScenariosRoot)
	if err != nil {
		return SweepReport{}, fmt.Errorf("list scenarios: %w", err)
	}

	jobs := make(chan ScenarioRef)
	var report sweepCounters
	var wg sync.WaitGroup
	workers := s.cfg.Concurrency
	if workers > len(scenarios) && len(scenarios) > 0 {
		workers = len(scenarios)
	}
	if workers == 0 {
		return SweepReport{Duration: s.cfg.Now().Sub(start)}, nil
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ref := range jobs {
				s.sweepScenario(ctx, ref, &report)
			}
		}()
	}

send:
	for _, ref := range scenarios {
		select {
		case <-ctx.Done():
			break send
		case jobs <- ref:
		}
	}
	close(jobs)
	wg.Wait()

	out := report.snapshot()
	out.Duration = s.cfg.Now().Sub(start)
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	return out, nil
}

func (s *Sweeper) RunLoop(ctx context.Context) {
	interval := s.cfg.Interval
	if interval <= 0 {
		interval = time.Hour
	}
	if s.cfg.InitialJitter > 0 {
		timer := time.NewTimer(s.cfg.InitialJitter)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}

	run := func() {
		report, err := s.RunOnce(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrSweepInProgress) {
				return
			}
			s.cfg.Logger.Printf("score sweep failed: %v", err)
			return
		}
		s.cfg.Logger.Printf("score sweep complete: scanned=%d skipped=%d scored=%d persisted=%d failed=%d duration=%s",
			report.Scanned, report.Skipped, report.Scored, report.Persisted, report.Failed, report.Duration)
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (s *Sweeper) sweepScenario(ctx context.Context, ref ScenarioRef, counters *sweepCounters) {
	if ctx.Err() != nil {
		return
	}
	counters.addScanned()

	digest, err := s.cfg.Digester.ComputeDigest(ref.Root)
	if err != nil || strings.TrimSpace(digest) == "" {
		counters.addFailed()
		s.cfg.Logger.Printf("score sweep digest failed for %s: %v", ref.Name, err)
		return
	}

	latest, ok, err := s.cfg.Repository.LatestFor(ctx, ref.Name)
	if err != nil {
		counters.addFailed()
		s.cfg.Logger.Printf("score sweep latest lookup failed for %s: %v", ref.Name, err)
		return
	}
	if ok && latest.Digest == digest {
		counters.addSkipped()
		return
	}

	result, err := s.cfg.Scorer.GetScore(ref.Name)
	if err != nil {
		counters.addFailed()
		s.cfg.Logger.Printf("score sweep scoring failed for %s: %v", ref.Name, err)
		return
	}
	counters.addScored()

	snap, err := SnapshotFromResult(result, digest, "sweeper")
	if err != nil {
		counters.addFailed()
		s.cfg.Logger.Printf("score sweep snapshot conversion failed for %s: %v", ref.Name, err)
		return
	}
	inserted, err := s.cfg.Repository.UpsertSnapshot(ctx, snap)
	if err != nil {
		counters.addFailed()
		s.cfg.Logger.Printf("score sweep snapshot persist failed for %s: %v", ref.Name, err)
		return
	}
	if inserted {
		counters.addPersisted()
	} else {
		counters.addSkipped()
	}
}

// SnapshotFromResult converts a computed score into the persisted snapshot
// shape used by the sweeper and bounded recompute paths.
func SnapshotFromResult(result Result, fallbackDigest, source string) (Snapshot, error) {
	digest := strings.TrimSpace(result.Freshness.Digest)
	if digest == "" {
		digest = strings.TrimSpace(fallbackDigest)
	}
	if digest == "" {
		return Snapshot{}, errors.New("snapshot digest is required")
	}
	breakdown, err := json.Marshal(result.Composite)
	if err != nil {
		return Snapshot{}, fmt.Errorf("marshal composite breakdown: %w", err)
	}
	var imp *float64
	if result.Importance != nil {
		value := result.Importance.Score
		imp = &value
	}
	createdAt := result.CalculatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return Snapshot{
		Scenario:       result.Scenario,
		Category:       result.Category,
		Digest:         digest,
		Composite:      result.Composite.Score,
		Classification: result.Composite.Classification,
		WorkingRung:    result.Maturity.WorkingRung,
		BreakdownJSON:  string(breakdown),
		Importance:     imp,
		Source:         source,
		CreatedAt:      createdAt,
	}, nil
}

type sweepCounters struct {
	mu        sync.Mutex
	scanned   int
	skipped   int
	scored    int
	persisted int
	failed    int
}

func (c *sweepCounters) addScanned()   { c.add(func() { c.scanned++ }) }
func (c *sweepCounters) addSkipped()   { c.add(func() { c.skipped++ }) }
func (c *sweepCounters) addScored()    { c.add(func() { c.scored++ }) }
func (c *sweepCounters) addPersisted() { c.add(func() { c.persisted++ }) }
func (c *sweepCounters) addFailed()    { c.add(func() { c.failed++ }) }

func (c *sweepCounters) add(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn()
}

func (c *sweepCounters) snapshot() SweepReport {
	c.mu.Lock()
	defer c.mu.Unlock()
	return SweepReport{
		Scanned:   c.scanned,
		Skipped:   c.skipped,
		Scored:    c.scored,
		Persisted: c.persisted,
		Failed:    c.failed,
	}
}
