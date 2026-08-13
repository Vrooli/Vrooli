package routing

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProviderDemotionState is the durable, queryable portion of zero-yield
// evidence. It contains no corpus data and survives a Search Hub restart.
type ProviderDemotionState struct {
	ProviderID    string
	Routed        int64
	Hits          int64
	EmptyStreak   int64
	Demoted       bool
	Probation     bool
	DecayDeadline time.Time
	Trigger       string
}

type DemotionStore interface {
	Load(context.Context) ([]ProviderDemotionState, error)
	Save(context.Context, ProviderDemotionState) error
}

// defaultZeroYieldMinimumRoutes requires repeated successful empty responses
// before demotion; transport failures are deliberately excluded.
const defaultZeroYieldMinimumRoutes int64 = 5

// defaultZeroYieldDemotionWindow bounds how long a provider stays out of
// automatic routing before a shadow/probation request can test recovery.
const defaultZeroYieldDemotionWindow = 15 * time.Minute

// DefaultRecoveryProbeInterval keeps unattended recovery traffic low-rate:
// the loop checks for expired demotions once per minute, while a provider still
// has a 15-minute decay window before it can be probed.
const DefaultRecoveryProbeInterval = time.Minute

// DefaultRecoveryProbeQuery is intentionally generic and bounded. Provider
// owners remain responsible for corpus quality; a hit proves only that the
// provider has become useful enough to re-enter automatic routing.
const DefaultRecoveryProbeQuery = "recovery probe"

// providerBreakers keeps only short-lived availability state. It deliberately
// never caches provider results or corpus content: Search Hub remains a thin
// router and providers stay authoritative for every returned hit.
type providerBreakers struct {
	mu                     sync.Mutex
	config                 rerankBreakerConfig
	byID                   map[string]*rerankBreaker
	stats                  map[string]*providerYieldStats
	zeroYieldMinimumRoutes int64
	demotionWindow         time.Duration
	store                  DemotionStore
	restored               bool
}

type providerYieldStats struct {
	routed        int64
	hits          int64
	demoted       bool
	emptyStreak   int64
	decayDeadline time.Time
	probation     bool
	trigger       string
}

func newProviderBreakers(cfg rerankBreakerConfig, stores ...DemotionStore) *providerBreakers {
	if cfg.FailureThreshold <= 0 || cfg.Cooldown <= 0 {
		return nil
	}
	minimumRoutes := cfg.ZeroYieldMinimumRoutes
	if minimumRoutes <= 0 {
		minimumRoutes = defaultZeroYieldMinimumRoutes
	}
	demotionWindow := cfg.DemotionWindow
	if demotionWindow <= 0 {
		demotionWindow = defaultZeroYieldDemotionWindow
	}
	var store DemotionStore
	if len(stores) > 0 {
		store = stores[0]
	}
	return &providerBreakers{
		config: cfg, byID: make(map[string]*rerankBreaker), stats: make(map[string]*providerYieldStats),
		zeroYieldMinimumRoutes: minimumRoutes, demotionWindow: demotionWindow,
		store: store,
	}
}

func (p *providerBreakers) restore(ctx context.Context) {
	if p == nil || p.store == nil {
		return
	}
	p.mu.Lock()
	if p.restored {
		p.mu.Unlock()
		return
	}
	p.restored = true
	p.mu.Unlock()
	states, err := p.store.Load(ctx)
	if err != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, state := range states {
		p.stats[state.ProviderID] = &providerYieldStats{
			routed: state.Routed, hits: state.Hits, emptyStreak: state.EmptyStreak,
			demoted: state.Demoted, probation: state.Probation,
			decayDeadline: state.DecayDeadline, trigger: state.Trigger,
		}
	}
}

func (p *providerBreakers) persist(id string, stats providerYieldStats) {
	if p == nil || p.store == nil {
		return
	}
	_ = p.store.Save(context.Background(), ProviderDemotionState{
		ProviderID: id, Routed: stats.routed, Hits: stats.hits, EmptyStreak: stats.emptyStreak,
		Demoted: stats.demoted, Probation: stats.probation, DecayDeadline: stats.decayDeadline,
		Trigger: stats.trigger,
	})
}

func (p *providerBreakers) breaker(id string) *rerankBreaker {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if b := p.byID[id]; b != nil {
		return b
	}
	b := newRerankBreaker(p.config)
	p.byID[id] = b
	return b
}

func (p *providerBreakers) yieldStats(id string) *providerYieldStats {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if s := p.stats[id]; s != nil {
		return s
	}
	s := &providerYieldStats{}
	p.stats[id] = s
	return s
}

func (p *providerBreakers) allow(id string, now time.Time) (bool, string) {
	b := p.breaker(id)
	if b == nil {
		return true, ""
	}
	ok, note := b.allow(now)
	if ok {
		return true, ""
	}
	return false, fmt.Sprintf("provider %s circuit unavailable: %s", id, note)
}

func (p *providerBreakers) record(id string, degraded bool, now time.Time) {
	b := p.breaker(id)
	if b == nil {
		return
	}
	if degraded {
		b.recordFailure(now)
		return
	}
	b.recordSuccess()
}

func (p *providerBreakers) recordResult(id string, hitCount int, degraded, explicit bool, now time.Time) {
	s := p.yieldStats(id)
	if s == nil {
		return
	}
	p.mu.Lock()
	defer func() {
		snapshot := *s
		p.mu.Unlock()
		p.persist(id, snapshot)
	}()
	// A transport/dependency failure belongs to the circuit breaker. It is not
	// evidence that a healthy corpus yielded nothing.
	if degraded {
		return
	}
	s.routed++
	s.hits += int64(hitCount)
	if explicit {
		if hitCount > 0 {
			s.demoted = false
			s.probation = false
			s.emptyStreak = 0
			s.decayDeadline = time.Time{}
			s.trigger = "explicit hit recovery"
			s.routed = 0
			s.hits = 0
		} else {
			// Explicit selection is evidence too. Track zero-yield streaks so
			// status and operators can see the corpus is producing no hits, but
			// never demote an explicitly selected provider as a side effect of
			// the caller's intentional choice.
			s.emptyStreak++
		}
		return
	}
	if hitCount > 0 {
		s.demoted = false
		s.probation = false
		s.emptyStreak = 0
		s.decayDeadline = time.Time{}
		s.trigger = "successful hit recovery"
	} else {
		s.emptyStreak++
	}
	if s.emptyStreak >= p.zeroYieldMinimumRoutes {
		s.demoted = true
		s.probation = false
		if s.decayDeadline.IsZero() || !now.Before(s.decayDeadline) {
			s.decayDeadline = now.Add(p.demotionWindow)
		}
		s.trigger = fmt.Sprintf("%d successful empty routes with zero hits", s.emptyStreak)
	}
}

func (p *providerBreakers) eligibleAutomatic(id string, now time.Time) bool {
	if p == nil {
		return true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.stats[id]
	if s == nil || !s.demoted {
		return true
	}
	if now.Before(s.decayDeadline) || s.probation {
		return false
	}
	// The first automatic request after decay is the shadow/probation probe.
	// Its result is fed back through recordResult and either restores eligibility
	// or starts a fresh decay window.
	s.probation = true
	return true
}

// beginRecoveryProbe atomically claims the single probation slot for a
// demoted provider. Returning false means the provider is not demoted, its
// decay window has not elapsed, or another probe already owns the slot.
func (p *providerBreakers) beginRecoveryProbe(id string, now time.Time) bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	s := p.stats[id]
	if s == nil || !s.demoted || now.Before(s.decayDeadline) || s.probation {
		p.mu.Unlock()
		return false
	}
	s.probation = true
	snapshot := *s
	p.mu.Unlock()
	p.persist(id, snapshot)
	return true
}

// beginFailureRecoveryProbe claims the same single probation discipline for a
// transport circuit whose cooldown elapsed. It intentionally does not use
// rerankBreaker.allow: that method marks the circuit as probing and the probe's
// own Query would then reject itself. The recovery query bypasses the normal
// allow gate after this claim and records success/failure through the normal
// breaker path.
func (p *providerBreakers) beginFailureRecoveryProbe(id string, now time.Time) bool {
	if p == nil {
		return false
	}
	b := p.breaker(id)
	if b == nil {
		return false
	}
	b.mu.Lock()
	due := b.open && now.Sub(b.openedAt) >= b.cooldown && !b.probing
	if due {
		b.probing = true
	}
	b.mu.Unlock()
	if !due {
		return false
	}
	p.mu.Lock()
	s := p.stats[id]
	if s == nil {
		s = &providerYieldStats{}
		p.stats[id] = s
	}
	if s.probation {
		p.mu.Unlock()
		b.mu.Lock()
		b.probing = false
		b.mu.Unlock()
		return false
	}
	s.probation = true
	snapshot := *s
	p.mu.Unlock()
	p.persist(id, snapshot)
	return true
}

// finishFailureRecoveryProbe releases the shared probation slot when a probe
// did not reach the normal fan-out result recorder.
func (p *providerBreakers) finishFailureRecoveryProbe(id string, now time.Time, reason string) {
	if p == nil {
		return
	}
	b := p.breaker(id)
	b.mu.Lock()
	if b.open {
		b.openedAt = now
	}
	b.probing = false
	b.mu.Unlock()
	p.mu.Lock()
	if s := p.stats[id]; s != nil {
		s.probation = false
		if strings.TrimSpace(reason) != "" {
			s.trigger = "failure recovery probe unavailable: " + strings.TrimSpace(reason)
		}
	}
	p.mu.Unlock()
}

// recoveryProbeFailed releases a probation slot after a transport/degraded
// failure without counting that failure as a graded empty response.
func (p *providerBreakers) recoveryProbeFailed(id string, now time.Time, reason string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	s := p.stats[id]
	if s == nil || !s.demoted {
		p.mu.Unlock()
		return
	}
	s.probation = false
	s.decayDeadline = now.Add(p.demotionWindow)
	if strings.TrimSpace(reason) == "" {
		s.trigger = "recovery probe unavailable"
	} else {
		s.trigger = "recovery probe unavailable: " + strings.TrimSpace(reason)
	}
	snapshot := *s
	p.mu.Unlock()
	p.persist(id, snapshot)
}

func (p *providerBreakers) demotion(id string) (bool, int64, int64) {
	if p == nil {
		return false, 0, 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.stats[id]
	if s == nil {
		return false, 0, 0
	}
	return s.demoted, s.routed, s.hits
}

func (p *providerBreakers) demotionDeadline(id string) time.Time {
	if p == nil {
		return time.Time{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if s := p.stats[id]; s != nil {
		return s.decayDeadline
	}
	return time.Time{}
}

func (p *providerBreakers) state(id string) *providerYieldStats {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if stats := p.stats[id]; stats != nil {
		copy := *stats
		return &copy
	}
	return nil
}

func (p *providerBreakers) openImmediately(id string, now time.Time) {
	if b := p.breaker(id); b != nil {
		b.openImmediately(now)
	}
}

// repromote clears only the graded-empty demotion evidence. Transport breaker
// state is intentionally untouched: an operator may restore a useful corpus
// to automatic routing without pretending its endpoint is reachable.
func (p *providerBreakers) repromote(id string, now time.Time) bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	stats, ok := p.stats[id]
	if ok {
		stats.demoted = false
		stats.probation = false
		stats.routed = 0
		stats.hits = 0
		stats.emptyStreak = 0
		stats.decayDeadline = time.Time{}
		stats.trigger = "operator repromotion at " + now.UTC().Format(time.RFC3339)
		copy := *stats
		p.mu.Unlock()
		p.persist(id, copy)
		return true
	}
	p.mu.Unlock()
	return false
}

func isScenarioNotRunning(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not running") || strings.Contains(s, "stopped") || strings.Contains(s, "scenario unavailable")
}

func (p *providerBreakers) status(id string, now time.Time) (bool, string) {
	b := p.breaker(id)
	if b == nil {
		return false, ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return false, ""
	}
	if now.Sub(b.openedAt) >= b.cooldown {
		return true, "provider circuit recovery probe is due"
	}
	return true, fmt.Sprintf("provider circuit open after %d consecutive failure(s); retry after %s", b.failures, b.openedAt.Add(b.cooldown).Format(time.RFC3339))
}
