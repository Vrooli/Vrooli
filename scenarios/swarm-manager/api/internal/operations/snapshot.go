package operations

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"swarm-manager/internal/initiatives"
)

// DefaultSnapshotTTL bounds how stale a cached OperationsSnapshot may be
// before the next reader triggers a rebuild. A cockpit view of initiative
// rankings tolerates up to two minutes of staleness — the rankings change
// only when an initiative's status, priority, dependency set, or member
// rollup changes, all of which are operator-paced rather than per-second.
// TTL is the floor guarantee; event-based invalidation can layer on top
// later without changing this contract.
const DefaultSnapshotTTL = 120 * time.Second

// Readiness classifies an initiative's actionability for ranking. The four
// states are mutually exclusive and evaluated in priority order by
// classifyReadiness: complete first, then blocked, then in_progress, else
// ready.
const (
	ReadinessReady      = "ready"
	ReadinessBlocked    = "blocked"
	ReadinessInProgress = "in_progress"
	ReadinessComplete   = "complete"
)

// readinessRank gives the sort weight for each readiness state — lower
// sorts first, so a "ready" initiative outranks a "blocked" one at equal
// priority. Unknown states fall to the back alongside complete.
func readinessRank(readiness string) int {
	switch readiness {
	case ReadinessReady:
		return 0
	case ReadinessInProgress:
		return 1
	case ReadinessBlocked:
		return 2
	case ReadinessComplete:
		return 3
	default:
		return 4
	}
}

// RankedInitiative is a single initiative annotated with the signals the
// operations cockpit ranks on. It is a transport-free domain value; the
// Connect layer (added with the operations migration) maps it to proto.
type RankedInitiative struct {
	Name               string                   `json:"name"`
	Title              string                   `json:"title"`
	Status             string                   `json:"status"`
	Mode               string                   `json:"mode,omitempty"`
	Priority           int                      `json:"priority"`
	Readiness          string                   `json:"readiness"`
	IncompleteDeps     []string                 `json:"incomplete_deps,omitempty"`
	BlockedMemberItems []string                 `json:"blocked_member_items,omitempty"`
	DownstreamUnblocks int                      `json:"downstream_unblocks"`
	Rollup             initiatives.RollupStatus `json:"rollup"`
}

// SnapshotSummary captures the portfolio-level counts a reader needs to
// frame the ranked list without re-deriving them from the slice.
type SnapshotSummary struct {
	TotalInitiatives   int `json:"total_initiatives"`
	ActiveInitiatives  int `json:"active_initiatives"`
	ReadyInitiatives   int `json:"ready_initiatives"`
	BlockedInitiatives int `json:"blocked_initiatives"`
	TotalBacklogItems  int `json:"total_backlog_items"`
	BlockedItems       int `json:"blocked_items"`
}

// OperationsSnapshot is the cached, ranked initiative view fed into the
// swarm_operations session context. Initiatives is pre-sorted by the
// ranking contract in rankInitiatives; consumers render in slice order.
type OperationsSnapshot struct {
	GeneratedAt time.Time          `json:"generated_at"`
	TTLSeconds  int                `json:"ttl_seconds"`
	Initiatives []RankedInitiative `json:"initiatives"`
	Summary     SnapshotSummary    `json:"summary"`
	Warnings    []string           `json:"warnings,omitempty"`
}

// SnapshotBuilder produces OperationsSnapshots from overview data and caches
// the result for TTL. It is safe for concurrent use.
type SnapshotBuilder struct {
	overview OverviewReader
	ttl      time.Duration
	now      func() time.Time

	mu       sync.RWMutex
	cached   *OperationsSnapshot
	cachedAt time.Time
}

// SnapshotBuilderConfig wires a SnapshotBuilder. Overview is required; TTL
// and Now default to DefaultSnapshotTTL and time.Now.
type SnapshotBuilderConfig struct {
	Overview OverviewReader
	TTL      time.Duration
	Now      func() time.Time
}

// NewSnapshotBuilder validates config and returns a builder. A nil Overview
// is a programming error and fails fast at construction.
func NewSnapshotBuilder(cfg SnapshotBuilderConfig) (*SnapshotBuilder, error) {
	if cfg.Overview == nil {
		return nil, fmt.Errorf("operations: SnapshotBuilderConfig.Overview is required")
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultSnapshotTTL
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &SnapshotBuilder{overview: cfg.Overview, ttl: ttl, now: now}, nil
}

// GetSnapshot returns a cached snapshot when one is still fresh, otherwise
// rebuilds it. The cache is read under an RLock and the rebuild is gated by
// a write lock so concurrent readers share one rebuild's result rather than
// each issuing their own overview load.
func (b *SnapshotBuilder) GetSnapshot(_ context.Context) (*OperationsSnapshot, error) {
	now := b.now().UTC()

	b.mu.RLock()
	if b.cached != nil && now.Sub(b.cachedAt) < b.ttl {
		snap := b.cached
		b.mu.RUnlock()
		return snap, nil
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	// Re-check under the write lock: another goroutine may have rebuilt
	// while we waited for the lock.
	if b.cached != nil && now.Sub(b.cachedAt) < b.ttl {
		return b.cached, nil
	}

	snap, err := b.build()
	if err != nil {
		return nil, err
	}
	b.cached = snap
	b.cachedAt = now
	return snap, nil
}

// Invalidate drops the cached snapshot so the next GetSnapshot rebuilds.
// Wired to backlog/initiative mutation events as an enhancement over TTL.
func (b *SnapshotBuilder) Invalidate() {
	b.mu.Lock()
	b.cached = nil
	b.cachedAt = time.Time{}
	b.mu.Unlock()
}

// build loads overview data and assembles the ranked snapshot. It never
// caches — GetSnapshot owns the cache lifecycle.
func (b *SnapshotBuilder) build() (*OperationsSnapshot, error) {
	ov, err := b.overview.GetOverview()
	if err != nil {
		return nil, fmt.Errorf("operations snapshot: load overview: %w", err)
	}

	blockedItems := make(map[string]bool, len(ov.DependencyGraph.Blocked))
	for _, key := range ov.DependencyGraph.Blocked {
		blockedItems[key] = true
	}

	ranked := rankInitiatives(ov.Initiatives, blockedItems)

	summary := SnapshotSummary{
		TotalInitiatives:  len(ov.Initiatives),
		ActiveInitiatives: ov.Summary.ActiveInitiatives,
		TotalBacklogItems: ov.Summary.TotalItems,
		BlockedItems:      len(ov.DependencyGraph.Blocked),
	}
	for _, ri := range ranked {
		switch ri.Readiness {
		case ReadinessReady:
			summary.ReadyInitiatives++
		case ReadinessBlocked:
			summary.BlockedInitiatives++
		}
	}

	return &OperationsSnapshot{
		GeneratedAt: b.now().UTC(),
		TTLSeconds:  int(b.ttl / time.Second),
		Initiatives: ranked,
		Summary:     summary,
	}, nil
}

// rankInitiatives annotates each initiative with readiness, incomplete
// dependencies, blocked member items, and downstream-unblocks count, then
// sorts by the ranking contract:
//
//  1. prioritized (priority > 0) before unprioritized;
//  2. among prioritized, ascending priority (P1 is the highest priority —
//     matching the backlog query/export convention);
//  3. then readiness (ready first, complete last);
//  4. then downstream-unblocks descending (unblocking more work first);
//  5. then name ascending for a stable order.
func rankInitiatives(inits []initiatives.InitiativeWithRollup, blockedItems map[string]bool) []RankedInitiative {
	// statusByName lets dependency evaluation look up an initiative's
	// status without a second pass. Missing deps are fail-open (treated as
	// complete) so a dangling reference never wedges a whole initiative.
	statusByName := make(map[string]string, len(inits))
	for _, iw := range inits {
		statusByName[iw.Initiative.Name] = iw.Initiative.Status
	}

	// downstreamCount[name] = how many not-complete initiatives depend on
	// `name`. Completing `name` is what would unblock them, so this is the
	// "unblock leverage" signal.
	downstreamCount := make(map[string]int, len(inits))
	for _, iw := range inits {
		if iw.Initiative.Status == initiatives.InitiativeStatusCompleted {
			continue
		}
		for _, dep := range iw.Initiative.DependsOn {
			downstreamCount[dep]++
		}
	}

	ranked := make([]RankedInitiative, 0, len(inits))
	for _, iw := range inits {
		init := iw.Initiative

		incompleteDeps := make([]string, 0)
		for _, dep := range init.DependsOn {
			depStatus, found := statusByName[dep]
			if !found {
				continue // fail-open: unknown dependency presumed complete
			}
			if depStatus != initiatives.InitiativeStatusCompleted {
				incompleteDeps = append(incompleteDeps, dep)
			}
		}

		blockedMembers := make([]string, 0)
		for _, member := range init.Items {
			if blockedItems[member] {
				blockedMembers = append(blockedMembers, member)
			}
		}

		readiness := classifyReadiness(init, iw.Rollup, len(incompleteDeps) > 0 || len(blockedMembers) > 0)

		ranked = append(ranked, RankedInitiative{
			Name:               init.Name,
			Title:              init.Title,
			Status:             init.Status,
			Mode:               initiatives.NormalizeMode(init.Mode),
			Priority:           init.Priority,
			Readiness:          readiness,
			IncompleteDeps:     nilIfEmpty(incompleteDeps),
			BlockedMemberItems: nilIfEmpty(blockedMembers),
			DownstreamUnblocks: downstreamCount[init.Name],
			Rollup:             iw.Rollup,
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		aPrioritized, bPrioritized := a.Priority > 0, b.Priority > 0
		if aPrioritized != bPrioritized {
			return aPrioritized // prioritized initiatives first
		}
		if aPrioritized && a.Priority != b.Priority {
			return a.Priority < b.Priority // ascending: P1 outranks P10
		}
		if readinessRank(a.Readiness) != readinessRank(b.Readiness) {
			return readinessRank(a.Readiness) < readinessRank(b.Readiness)
		}
		if a.DownstreamUnblocks != b.DownstreamUnblocks {
			return a.DownstreamUnblocks > b.DownstreamUnblocks
		}
		return a.Name < b.Name
	})

	return ranked
}

// classifyReadiness maps an initiative to one of the four readiness states.
// Evaluation order matters: a completed initiative is complete regardless of
// stale dependency metadata; an initiative with any incomplete dependency or
// blocked member item is blocked; one with active member work or in a review
// phase is in_progress; otherwise it is ready to act on.
func classifyReadiness(init initiatives.Initiative, rollup initiatives.RollupStatus, hasBlockers bool) string {
	if init.Status == initiatives.InitiativeStatusCompleted {
		return ReadinessComplete
	}
	if hasBlockers {
		return ReadinessBlocked
	}
	if rollup.InProgress > 0 || initiatives.IsReviewInitiativeStatus(init.Status) {
		return ReadinessInProgress
	}
	return ReadinessReady
}

// nilIfEmpty normalizes an accumulator slice to nil when empty so JSON
// omitempty drops the field rather than emitting "[]".
func nilIfEmpty(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return in
}
