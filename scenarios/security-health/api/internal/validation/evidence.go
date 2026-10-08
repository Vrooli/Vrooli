package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// EvidenceKey identifies one scanner result completely. Fingerprint is a
// digest of every input that can affect the scanner's normalized findings,
// including the scanner/policy identity and external advisory freshness when
// applicable.
type EvidenceKey struct {
	Scenario    string
	Scanner     string
	Fingerprint string
}

// Validate rejects incomplete correctness identities. Persistence adapters call
// the same guard so direct store use cannot create unreachable rows.
func (k EvidenceKey) Validate() error {
	if strings.TrimSpace(k.Scenario) == "" {
		return errors.New("evidence scenario is required")
	}
	if strings.TrimSpace(k.Scanner) == "" {
		return errors.New("evidence scanner is required")
	}
	if strings.TrimSpace(k.Fingerprint) == "" {
		return errors.New("evidence fingerprint is required")
	}
	return nil
}

func (k EvidenceKey) flightKey() string {
	return k.Scenario + "\x00" + k.Scanner + "\x00" + k.Fingerprint
}

// EvidenceRecord is the only data eligible for persistence. Findings have
// already crossed the scanner's normalization/redaction boundary; raw command
// output and scanner-native match objects are intentionally absent.
type EvidenceRecord struct {
	Key       EvidenceKey
	Findings  []Finding
	ExpiresAt time.Time
}

// EvidenceStore is the persistence seam for normalized scanner evidence.
// Implementations must treat corrupt or expired entries as misses.
type EvidenceStore interface {
	Load(ctx context.Context, key EvidenceKey, now time.Time) (EvidenceRecord, bool, error)
	Store(ctx context.Context, record EvidenceRecord, now time.Time) error
	DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error)
}

// EvidenceSource describes how a scanner result was obtained.
type EvidenceSource string

const (
	EvidenceSourceCache     EvidenceSource = "cache"
	EvidenceSourceExecution EvidenceSource = "execution"
	EvidenceSourceCoalesced EvidenceSource = "coalesced"
	EvidenceSourceUncached  EvidenceSource = "uncached"
)

// EvidenceOutcome is request-local execution evidence. It is safe to expose
// through metrics because it contains no target content or scanner output.
type EvidenceOutcome struct {
	Source        EvidenceSource
	AdmissionWait time.Duration
	ExecutionTime time.Duration
	CacheError    bool
}

// ExecuteUncached runs evidence through the same admission budget without
// persistence or coalescing. It is the fail-safe path for scanners without a
// trustworthy fingerprint and for fingerprint computation failures.
func (c *EvidenceCoordinator) ExecuteUncached(ctx context.Context, scanner string, weight int64, run EvidenceRun) ([]Finding, EvidenceOutcome, error) {
	if strings.TrimSpace(scanner) == "" {
		return nil, EvidenceOutcome{}, errors.New("scanner name is required")
	}
	if run == nil {
		return nil, EvidenceOutcome{}, errors.New("evidence run function is required")
	}
	if weight <= 0 {
		return nil, EvidenceOutcome{}, errors.New("scanner weight must be positive")
	}
	waitStarted := c.clock()
	if err := c.gate.Acquire(ctx, weight); err != nil {
		return nil, EvidenceOutcome{}, err
	}
	outcome := EvidenceOutcome{Source: EvidenceSourceUncached, AdmissionWait: c.clock().Sub(waitStarted)}
	defer c.gate.Release(weight)
	runStarted := c.clock()
	findings, err := run(ctx)
	outcome.ExecutionTime = c.clock().Sub(runStarted)
	c.record(scanner, func(m *ScannerEvidenceMetrics) {
		m.UncachedExecutions++
		m.AdmissionWait += outcome.AdmissionWait
		m.ExecutionTime += outcome.ExecutionTime
		if err != nil {
			m.Failures++
		} else {
			m.Executions++
		}
	})
	return cloneFindings(findings), outcome, err
}

// EvidenceRun is the cache-miss operation. It must return normalized,
// redacted findings rather than scanner-native output.
type EvidenceRun func(context.Context) ([]Finding, error)

// DOC: docs/internal/PERFORMANCE.md#incremental-validation-model
// EvidenceCoordinator owns cache lookup, identical-request coalescing, and the
// one shared resource budget used by every scanner subprocess.
type EvidenceCoordinator struct {
	store EvidenceStore
	clock func() time.Time
	gate  *WeightedGate

	mu      sync.Mutex
	flights map[string]*evidenceFlight
	metrics map[string]ScannerEvidenceMetrics
}

type evidenceFlight struct {
	done     chan struct{}
	findings []Finding
	outcome  EvidenceOutcome
	err      error
}

// EvidenceCoordinatorDeps makes time, persistence, and capacity explicit.
// Capacity is a relative scanner-cost budget, not a goroutine count.
type EvidenceCoordinatorDeps struct {
	Store    EvidenceStore
	Clock    func() time.Time
	Capacity int64
}

// NewEvidenceCoordinator constructs the shared scanner execution controller.
func NewEvidenceCoordinator(deps EvidenceCoordinatorDeps) *EvidenceCoordinator {
	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	capacity := deps.Capacity
	if capacity <= 0 {
		capacity = 1
	}
	return &EvidenceCoordinator{
		store:   deps.Store,
		clock:   clock,
		gate:    NewWeightedGate(capacity),
		flights: make(map[string]*evidenceFlight),
		metrics: make(map[string]ScannerEvidenceMetrics),
	}
}

// Execute returns fresh cached evidence or runs one cache miss. Successful
// results are persisted; scanner failures and cancellations never are. Cache
// failures degrade to a real scan because stale or unavailable optimization
// state must not weaken validation.
func (c *EvidenceCoordinator) Execute(
	ctx context.Context,
	key EvidenceKey,
	weight int64,
	ttl time.Duration,
	run EvidenceRun,
) ([]Finding, EvidenceOutcome, error) {
	if err := key.Validate(); err != nil {
		return nil, EvidenceOutcome{}, err
	}
	if run == nil {
		return nil, EvidenceOutcome{}, errors.New("evidence run function is required")
	}
	if ttl <= 0 {
		return nil, EvidenceOutcome{}, errors.New("evidence ttl must be positive")
	}
	if weight <= 0 {
		return nil, EvidenceOutcome{}, errors.New("scanner weight must be positive")
	}

	now := c.clock()
	cacheDegraded := false
	if c.store != nil {
		record, ok, err := c.store.Load(ctx, key, now)
		if err != nil {
			cacheDegraded = true
			c.record(key.Scanner, func(m *ScannerEvidenceMetrics) { m.CacheErrors++ })
		} else if ok {
			c.record(key.Scanner, func(m *ScannerEvidenceMetrics) { m.Hits++ })
			return cloneFindings(record.Findings), EvidenceOutcome{Source: EvidenceSourceCache}, nil
		}
	}
	c.record(key.Scanner, func(m *ScannerEvidenceMetrics) { m.Misses++ })

	flightKey := key.flightKey()
	c.mu.Lock()
	if existing, ok := c.flights[flightKey]; ok {
		c.metrics[key.Scanner] = incrementCoalesced(c.metrics[key.Scanner])
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, EvidenceOutcome{}, ctx.Err()
		case <-existing.done:
			outcome := existing.outcome
			outcome.Source = EvidenceSourceCoalesced
			return cloneFindings(existing.findings), outcome, existing.err
		}
	}
	flight := &evidenceFlight{done: make(chan struct{})}
	flight.outcome.CacheError = cacheDegraded
	c.flights[flightKey] = flight
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.flights, flightKey)
		close(flight.done)
		c.mu.Unlock()
	}()

	waitStarted := c.clock()
	if err := c.gate.Acquire(ctx, weight); err != nil {
		flight.err = err
		return nil, EvidenceOutcome{}, err
	}
	flight.outcome.AdmissionWait = c.clock().Sub(waitStarted)
	defer c.gate.Release(weight)

	runStarted := c.clock()
	findings, err := run(ctx)
	flight.outcome.ExecutionTime = c.clock().Sub(runStarted)
	flight.outcome.Source = EvidenceSourceExecution
	if err != nil {
		flight.err = err
		c.record(key.Scanner, func(m *ScannerEvidenceMetrics) {
			m.Failures++
			m.AdmissionWait += flight.outcome.AdmissionWait
			m.ExecutionTime += flight.outcome.ExecutionTime
		})
		return nil, flight.outcome, err
	}

	flight.findings = cloneFindings(findings)
	c.record(key.Scanner, func(m *ScannerEvidenceMetrics) {
		m.Executions++
		m.AdmissionWait += flight.outcome.AdmissionWait
		m.ExecutionTime += flight.outcome.ExecutionTime
	})
	if c.store != nil {
		record := EvidenceRecord{
			Key:       key,
			Findings:  cloneFindings(findings),
			ExpiresAt: c.clock().Add(ttl),
		}
		if err := c.store.Store(ctx, record, c.clock()); err != nil {
			flight.outcome.CacheError = true
			c.record(key.Scanner, func(m *ScannerEvidenceMetrics) { m.CacheErrors++ })
		}
	}
	return cloneFindings(findings), flight.outcome, nil
}

func incrementCoalesced(m ScannerEvidenceMetrics) ScannerEvidenceMetrics {
	m.Coalesced++
	return m
}

// ScannerEvidenceMetrics is a stable aggregate for one scanner identity.
type ScannerEvidenceMetrics struct {
	Hits               uint64
	Misses             uint64
	Coalesced          uint64
	Executions         uint64
	Failures           uint64
	CacheErrors        uint64
	UncachedExecutions uint64
	AdmissionWait      time.Duration
	ExecutionTime      time.Duration
}

// EvidenceMetricsSnapshot is a point-in-time copy safe for status rendering.
type EvidenceMetricsSnapshot struct {
	Scanners map[string]ScannerEvidenceMetrics
	Capacity int64
	InUse    int64
	PeakUse  int64
}

// Metrics returns a copy; callers cannot mutate coordinator state.
func (c *EvidenceCoordinator) Metrics() EvidenceMetricsSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	scanners := make(map[string]ScannerEvidenceMetrics, len(c.metrics))
	for name, metric := range c.metrics {
		scanners[name] = metric
	}
	capacity, inUse, peak := c.gate.Snapshot()
	return EvidenceMetricsSnapshot{Scanners: scanners, Capacity: capacity, InUse: inUse, PeakUse: peak}
}

func (c *EvidenceCoordinator) record(scanner string, update func(*ScannerEvidenceMetrics)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	metric := c.metrics[scanner]
	update(&metric)
	c.metrics[scanner] = metric
}

func cloneFindings(in []Finding) []Finding {
	if in == nil {
		return nil
	}
	return append([]Finding(nil), in...)
}

// WeightedGate is a cancellation-aware FIFO resource budget. FIFO admission
// avoids partial-token deadlocks and prevents light scanners from starving a
// queued heavy scanner.
type WeightedGate struct {
	mu       sync.Mutex
	capacity int64
	inUse    int64
	peakUse  int64
	waiters  []*gateWaiter
}

type gateWaiter struct {
	weight  int64
	ready   chan struct{}
	granted bool
}

// NewWeightedGate constructs a gate with a positive capacity.
func NewWeightedGate(capacity int64) *WeightedGate {
	if capacity <= 0 {
		panic("weighted gate capacity must be positive")
	}
	return &WeightedGate{capacity: capacity}
}

// Acquire waits until weight fits in the shared budget or ctx is cancelled.
func (g *WeightedGate) Acquire(ctx context.Context, weight int64) error {
	if weight <= 0 {
		return errors.New("weight must be positive")
	}
	if weight > g.capacity {
		return fmt.Errorf("weight %d exceeds capacity %d", weight, g.capacity)
	}
	waiter := &gateWaiter{weight: weight, ready: make(chan struct{})}
	g.mu.Lock()
	g.waiters = append(g.waiters, waiter)
	g.grantLocked()
	g.mu.Unlock()

	select {
	case <-waiter.ready:
		return nil
	case <-ctx.Done():
		g.mu.Lock()
		if waiter.granted {
			g.inUse -= waiter.weight
		} else {
			for i, candidate := range g.waiters {
				if candidate == waiter {
					g.waiters = append(g.waiters[:i], g.waiters[i+1:]...)
					break
				}
			}
		}
		g.grantLocked()
		g.mu.Unlock()
		return ctx.Err()
	}
}

// Release returns weight to the gate. A mismatch is a programming error.
func (g *WeightedGate) Release(weight int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if weight <= 0 || weight > g.inUse {
		panic("weighted gate release exceeds acquired weight")
	}
	g.inUse -= weight
	g.grantLocked()
}

// Snapshot returns capacity, current usage, and process-lifetime peak usage.
func (g *WeightedGate) Snapshot() (capacity, inUse, peakUse int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.capacity, g.inUse, g.peakUse
}

func (g *WeightedGate) grantLocked() {
	for len(g.waiters) > 0 {
		waiter := g.waiters[0]
		if g.inUse+waiter.weight > g.capacity {
			return
		}
		g.waiters = g.waiters[1:]
		g.inUse += waiter.weight
		if g.inUse > g.peakUse {
			g.peakUse = g.inUse
		}
		waiter.granted = true
		close(waiter.ready)
	}
}
