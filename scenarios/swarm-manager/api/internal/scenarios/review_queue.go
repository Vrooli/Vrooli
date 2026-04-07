package scenarios

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"
)

// Scoring weights for composite review queue priority.
const (
	workloadWeight  = 3.0
	activityWeight  = 1.5
	stalenessWeight = 0.5

	// maxStalenessHours caps the staleness component at 30 days.
	maxStalenessHours = 720.0
	// defaultCooldownHours sets cooldown after a recent review.
	defaultCooldownHours = 12
	// recentActivityDays is the lookback window for counting recent executions.
	recentActivityDays = 30

	defaultLimit = 5
	maxLimit     = 50
	defaultTag   = "preemptive-qa"
)

// BacklogLister loads backlog items for review queue computation.
type BacklogLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// ExecutionLister loads execution records for review queue computation.
type ExecutionLister interface {
	List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error)
}

// scenarioQueueEntry accumulates per-scenario stats during scoring.
type scenarioQueueEntry struct {
	pendingCount         int
	recentExecutionCount int
	lastReviewClass      string
	lastReviewAt         time.Time
	excluded             bool
}

// reviewQueueResult is the internal representation before proto conversion.
type reviewQueueResult struct {
	scenarioName         string
	pendingCount         int
	lastReviewClass      string
	lastReviewAt         time.Time
	recentExecutionCount int
	compositeScore       float64
	primarySignal        string
	cooldownUntil        time.Time
}

// terminalBacklogStatus returns true for statuses that should not count as pending.
func terminalBacklogStatus(s backlog.BacklogStatus) bool {
	return s == backlog.StatusCompleted || s == backlog.StatusFailed
}

// computeReviewQueue is a pure function that computes the ranked review queue.
// It takes pre-loaded data to keep the function testable without I/O.
func computeReviewQueue(
	items []backlog.BacklogItem,
	records []execution.Record,
	excludeTag string,
	limit int,
	now time.Time,
) (results []reviewQueueResult, totalScenarios, excludedCount int) {
	if excludeTag == "" {
		excludeTag = defaultTag
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	entries := make(map[string]*scenarioQueueEntry)

	// Pass 1: Count pending backlog items per scenario and detect exclusions.
	for _, item := range items {
		if item.ArchivedAt != nil {
			continue
		}
		if terminalBacklogStatus(item.Status) {
			continue
		}
		scenarios := pathutil.ScenariosFromGlobs(item.AcceptanceAllow)
		if len(scenarios) == 0 {
			continue
		}

		isExclusionCandidate := (item.Kind == backlog.KindFix || item.Kind == backlog.KindChore) &&
			hasTag(item.Tags, excludeTag)

		for _, sc := range scenarios {
			entry := getOrCreate(entries, sc)
			entry.pendingCount++
			if isExclusionCandidate {
				entry.excluded = true
			}
		}
	}

	// Pass 2: Walk execution records for recent activity and last review.
	reviewSummaries := ComputeReviewSummaries(records)
	for name, summary := range reviewSummaries {
		entry := getOrCreate(entries, name)
		entry.lastReviewAt = summary.LastReviewAt
		entry.lastReviewClass = summary.LastReviewClassification
	}

	cutoff := now.Add(-recentActivityDays * 24 * time.Hour)
	for _, rec := range records {
		if rec.Finalization == nil {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
		if err != nil {
			continue
		}
		if !createdAt.After(cutoff) {
			continue
		}
		for _, sf := range rec.Finalization.Scenarios {
			entry := getOrCreate(entries, sf.ScenarioName)
			entry.recentExecutionCount++
		}
	}

	// Pass 3: Score and rank.
	totalScenarios = len(entries)
	for name, entry := range entries {
		if entry.excluded {
			excludedCount++
			continue
		}

		workloadScore := float64(entry.pendingCount) * workloadWeight
		activityScore := float64(entry.recentExecutionCount) * activityWeight

		var stalenessScore float64
		if entry.lastReviewAt.IsZero() {
			stalenessScore = stalenessWeight // max staleness if never reviewed
		} else {
			hours := now.Sub(entry.lastReviewAt).Hours()
			if hours > maxStalenessHours {
				hours = maxStalenessHours
			}
			stalenessScore = (hours / maxStalenessHours) * stalenessWeight
		}

		composite := workloadScore + activityScore + stalenessScore

		// Determine primary signal.
		primary := "staleness"
		maxComponent := stalenessScore
		if workloadScore > maxComponent {
			primary = "workload"
			maxComponent = workloadScore
		}
		if activityScore > maxComponent {
			primary = "recent_activity"
		}

		var cooldown time.Time
		if !entry.lastReviewAt.IsZero() {
			cd := entry.lastReviewAt.Add(defaultCooldownHours * time.Hour)
			if cd.After(now) {
				cooldown = cd
			}
		}

		results = append(results, reviewQueueResult{
			scenarioName:         name,
			pendingCount:         entry.pendingCount,
			lastReviewClass:      entry.lastReviewClass,
			lastReviewAt:         entry.lastReviewAt,
			recentExecutionCount: entry.recentExecutionCount,
			compositeScore:       math.Round(composite*100) / 100,
			primarySignal:        primary,
			cooldownUntil:        cooldown,
		})
	}

	// Sort descending by score, then alphabetically for stability.
	sort.Slice(results, func(i, j int) bool {
		if results[i].compositeScore != results[j].compositeScore {
			return results[i].compositeScore > results[j].compositeScore
		}
		return results[i].scenarioName < results[j].scenarioName
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, totalScenarios, excludedCount
}

func getOrCreate(m map[string]*scenarioQueueEntry, name string) *scenarioQueueEntry {
	if e, ok := m[name]; ok {
		return e
	}
	e := &scenarioQueueEntry{}
	m[name] = e
	return e
}

func hasTag(tags []string, target string) bool {
	lower := strings.ToLower(target)
	for _, t := range tags {
		if strings.ToLower(t) == lower {
			return true
		}
	}
	return false
}

// ReviewQueue handles GET /api/v1/scenarios/review-queue.
func (h *Handler) ReviewQueue(w http.ResponseWriter, r *http.Request) {
	if h.backlogLister == nil || h.executionLister == nil {
		apierr.MapError(w, "[scenarios] review-queue", apierr.Internal("review queue dependencies not configured"))
		return
	}

	// Parse query params.
	query := r.URL.Query()
	limit := defaultLimit
	if v := query.Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 1 || parsed > maxLimit {
			apierr.MapError(w, "[scenarios] review-queue", apierr.BadRequest("limit must be 1-50"))
			return
		}
		limit = parsed
	}
	excludeTag := query.Get("exclude_tag")

	// Load data.
	items, err := h.backlogLister.LoadAll(nil)
	if err != nil {
		slog.Error("review-queue: failed to load backlog", "error", err)
		apierr.MapError(w, "[scenarios] review-queue", apierr.Internal("failed to load backlog items"))
		return
	}

	records, err := h.executionLister.List(r.Context(), execution.ListFilters{})
	if err != nil {
		slog.Error("review-queue: failed to load executions", "error", err)
		apierr.MapError(w, "[scenarios] review-queue", apierr.Internal("failed to load execution records"))
		return
	}

	results, totalScenarios, excludedCount := computeReviewQueue(items, records, excludeTag, limit, time.Now())

	// Convert to proto response.
	protoItems := make([]*apipb.ScenarioReviewQueueItem, 0, len(results))
	for _, res := range results {
		item := &apipb.ScenarioReviewQueueItem{
			ScenarioName:         res.scenarioName,
			PendingBacklogCount:  int32(res.pendingCount),
			RecentExecutionCount: int32(res.recentExecutionCount),
			CompositeScore:       res.compositeScore,
			PrimarySignal:        res.primarySignal,
		}
		if res.lastReviewClass != "" {
			item.LastReviewClassification = &res.lastReviewClass
		}
		if !res.lastReviewAt.IsZero() {
			ts := res.lastReviewAt.Format(time.RFC3339)
			item.LastReviewAt = &ts
		}
		if !res.cooldownUntil.IsZero() {
			cd := res.cooldownUntil.Format(time.RFC3339)
			item.CooldownUntil = &cd
		}
		protoItems = append(protoItems, item)
	}

	resp := &apipb.ScenarioReviewQueueResponse{
		Items:          protoItems,
		TotalScenarios: int32(totalScenarios),
		ExcludedCount:  int32(excludedCount),
	}

	slog.Info("review-queue", "total", totalScenarios, "excluded", excludedCount, "returned", len(protoItems))
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] review-queue", apierr.Internal("failed to encode response"))
	}
}
