package operations

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"swarm-manager/internal/goals"
)

// DefaultSnapshotTTL bounds how stale a cached OperationsSnapshot may be.
const DefaultSnapshotTTL = 120 * time.Second

const (
	ReadinessReady      = "ready"
	ReadinessBlocked    = "blocked"
	ReadinessInProgress = "in_progress"
	ReadinessComplete   = "complete"
)

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

// RankedGoal is an active or archived goal annotated with derived scope
// signals. Goals do not have an explicit dependency graph; blocking and
// readiness come directly from their prerequisite closure.
type RankedGoal struct {
	Name               string      `json:"name"`
	Title              string      `json:"title"`
	Status             string      `json:"status"`
	Priority           int         `json:"priority"`
	Readiness          string      `json:"readiness"`
	BlockedMemberItems []string    `json:"blocked_member_items,omitempty"`
	Scope              goals.Scope `json:"scope"`
}

type SnapshotSummary struct {
	TotalGoals        int `json:"total_goals"`
	ActiveGoals       int `json:"active_goals"`
	ReadyGoals        int `json:"ready_goals"`
	BlockedGoals      int `json:"blocked_goals"`
	TotalBacklogItems int `json:"total_backlog_items"`
	BlockedItems      int `json:"blocked_items"`
}

// OperationsSnapshot is the cached goal-ranked view fed into the
// swarm_operations session context.
type OperationsSnapshot struct {
	GeneratedAt time.Time       `json:"generated_at"`
	TTLSeconds  int             `json:"ttl_seconds"`
	Goals       []RankedGoal    `json:"goals"`
	Summary     SnapshotSummary `json:"summary"`
	Warnings    []string        `json:"warnings,omitempty"`
}

type SnapshotBuilder struct {
	overview OverviewReader
	ttl      time.Duration
	now      func() time.Time

	mu       sync.RWMutex
	cached   *OperationsSnapshot
	cachedAt time.Time
}

type SnapshotBuilderConfig struct {
	Overview OverviewReader
	TTL      time.Duration
	Now      func() time.Time
}

func NewSnapshotBuilder(cfg SnapshotBuilderConfig) (*SnapshotBuilder, error) {
	if cfg.Overview == nil {
		return nil, fmt.Errorf("operations: SnapshotBuilderConfig.Overview is required")
	}
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultSnapshotTTL
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &SnapshotBuilder{overview: cfg.Overview, ttl: cfg.TTL, now: cfg.Now}, nil
}

func (b *SnapshotBuilder) GetSnapshot(_ context.Context) (*OperationsSnapshot, error) {
	now := b.now().UTC()
	b.mu.RLock()
	if b.cached != nil && now.Sub(b.cachedAt) < b.ttl {
		snapshot := b.cached
		b.mu.RUnlock()
		return snapshot, nil
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cached != nil && now.Sub(b.cachedAt) < b.ttl {
		return b.cached, nil
	}
	snapshot, err := b.build()
	if err != nil {
		return nil, err
	}
	b.cached, b.cachedAt = snapshot, now
	return snapshot, nil
}

func (b *SnapshotBuilder) Invalidate() {
	b.mu.Lock()
	b.cached, b.cachedAt = nil, time.Time{}
	b.mu.Unlock()
}

func (b *SnapshotBuilder) build() (*OperationsSnapshot, error) {
	ov, err := b.overview.GetOverview()
	if err != nil {
		return nil, fmt.Errorf("operations snapshot: load overview: %w", err)
	}
	ranked := rankGoals(ov.Goals)
	summary := SnapshotSummary{
		TotalGoals: len(ov.Goals), ActiveGoals: ov.Summary.ActiveGoals,
		TotalBacklogItems: ov.Summary.TotalItems, BlockedItems: len(ov.DependencyGraph.Blocked),
	}
	for _, goal := range ranked {
		switch goal.Readiness {
		case ReadinessReady:
			summary.ReadyGoals++
		case ReadinessBlocked:
			summary.BlockedGoals++
		}
	}
	return &OperationsSnapshot{GeneratedAt: b.now().UTC(), TTLSeconds: int(b.ttl / time.Second), Goals: ranked, Summary: summary}, nil
}

// rankGoals prioritizes explicit priority, then scope readiness, then name.
func rankGoals(goalList []goals.GoalWithScope) []RankedGoal {
	ranked := make([]RankedGoal, 0, len(goalList))
	for _, goal := range goalList {
		ranked = append(ranked, RankedGoal{
			Name: goal.Goal.Name, Title: goal.Goal.Title, Status: goal.Goal.Status,
			Priority: goal.Goal.Priority, Readiness: classifyGoalReadiness(goal),
			BlockedMemberItems: nilIfEmpty(goal.Scope.Blocked), Scope: goal.Scope,
		})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		aPrioritized, bPrioritized := a.Priority > 0, b.Priority > 0
		if aPrioritized != bPrioritized {
			return aPrioritized
		}
		if aPrioritized && a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if readinessRank(a.Readiness) != readinessRank(b.Readiness) {
			return readinessRank(a.Readiness) < readinessRank(b.Readiness)
		}
		return a.Name < b.Name
	})
	return ranked
}

func classifyGoalReadiness(goal goals.GoalWithScope) string {
	if goal.Goal.Status == goals.StatusArchived || (goal.Scope.Total > 0 && goal.Scope.CompletedCount == goal.Scope.Total) {
		return ReadinessComplete
	}
	if len(goal.Scope.Blocked) > 0 && len(goal.Scope.Ready) == 0 {
		return ReadinessBlocked
	}
	if len(goal.Scope.Ready) > 0 {
		return ReadinessReady
	}
	return ReadinessInProgress
}

func nilIfEmpty(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return in
}
