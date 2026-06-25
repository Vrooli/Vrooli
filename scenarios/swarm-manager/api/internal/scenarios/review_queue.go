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

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
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
	// hasMaintenanceItems is true when at least one pending item is a
	// maintenance kind (fix or chore) rather than a creation kind
	// (execute, idea, research).  Used by the greenfield-only heuristic.
	hasMaintenanceItems bool
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

// isCreationKind returns true for backlog kinds that represent planned work
// to create or research a scenario, as opposed to maintaining existing code.
func isCreationKind(k backlog.BacklogKind) bool {
	return k == backlog.KindExecute || k == backlog.KindIdea || k == backlog.KindResearch
}

// computeReviewQueue is a pure function that computes the ranked review queue.
// It takes pre-loaded data to keep the function testable without I/O.
//
// existingScenarios, when non-nil, is the set of scenario names that actually
// exist on disk.  Scenarios not in this set are filtered out.  When nil the
// check is skipped (graceful degradation if the scenario source is unavailable)
// and a fallback heuristic excludes scenarios whose only pending items are
// creation-oriented kinds (execute/idea/research) with no review history.
func computeReviewQueue(
	items []backlog.BacklogItem,
	records []execution.Record,
	excludeTag string,
	limit int,
	now time.Time,
	existingScenarios map[string]bool,
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

	tallyPendingItems(entries, items, excludeTag)
	tallyExecutionActivity(entries, records, now)

	totalScenarios = len(entries)
	results, excludedCount = scoreReviewQueueEntries(entries, now, existingScenarios)

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

// tallyPendingItems counts pending backlog items per scenario and records
// exclusion / maintenance flags (Pass 1).
func tallyPendingItems(entries map[string]*scenarioQueueEntry, items []backlog.BacklogItem, excludeTag string) {
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
			if !isCreationKind(item.Kind) {
				entry.hasMaintenanceItems = true
			}
		}
	}
}

// tallyExecutionActivity walks execution records to record last-review state
// and recent execution counts per scenario (Pass 2).
func tallyExecutionActivity(entries map[string]*scenarioQueueEntry, records []execution.Record, now time.Time) {
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
}

// scoreReviewQueueEntries scores and ranks the accumulated entries, applying
// exclusion filters (Pass 3). Returns the unsorted results and excluded count.
func scoreReviewQueueEntries(entries map[string]*scenarioQueueEntry, now time.Time, existingScenarios map[string]bool) (results []reviewQueueResult, excludedCount int) {
	for name, entry := range entries {
		if entry.excluded {
			excludedCount++
			continue
		}

		// Skip scenarios that don't exist on disk when the set is provided.
		if existingScenarios != nil && !existingScenarios[name] {
			excludedCount++
			continue
		}

		workloadScore := float64(entry.pendingCount) * workloadWeight
		activityScore := float64(entry.recentExecutionCount) * activityWeight
		stalenessScore := computeStalenessScore(entry.lastReviewAt, now)

		composite := workloadScore + activityScore + stalenessScore
		primary := primarySignalFor(workloadScore, activityScore, stalenessScore)
		cooldown := computeCooldown(entry.lastReviewAt, now)

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
	return results, excludedCount
}

// computeStalenessScore returns the staleness component of the composite score.
func computeStalenessScore(lastReviewAt, now time.Time) float64 {
	if lastReviewAt.IsZero() {
		return stalenessWeight // max staleness if never reviewed
	}
	hours := now.Sub(lastReviewAt).Hours()
	if hours > maxStalenessHours {
		hours = maxStalenessHours
	}
	return (hours / maxStalenessHours) * stalenessWeight
}

// primarySignalFor reports which scoring component dominates.
func primarySignalFor(workloadScore, activityScore, stalenessScore float64) string {
	primary := "staleness"
	maxComponent := stalenessScore
	if workloadScore > maxComponent {
		primary = "workload"
		maxComponent = workloadScore
	}
	if activityScore > maxComponent {
		primary = "recent_activity"
	}
	return primary
}

// computeCooldown returns the cooldown expiry, or the zero time when no
// cooldown is active.
func computeCooldown(lastReviewAt, now time.Time) time.Time {
	if lastReviewAt.IsZero() {
		return time.Time{}
	}
	cd := lastReviewAt.Add(defaultCooldownHours * time.Hour)
	if cd.After(now) {
		return cd
	}
	return time.Time{}
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

// applyGreenfieldFallback removes results whose pending backlog items are
// exclusively creation kinds (execute/idea/research) and that have no review
// history.  This is a best-effort heuristic used when the authoritative
// scenario source is unavailable.
func applyGreenfieldFallback(items []backlog.BacklogItem, results []reviewQueueResult, excludedCount int) ([]reviewQueueResult, int) {
	// Pre-compute per-scenario maintenance flag from backlog items.
	hasMaint := make(map[string]bool)
	for _, item := range items {
		if item.ArchivedAt != nil || terminalBacklogStatus(item.Status) {
			continue
		}
		if !isCreationKind(item.Kind) {
			for _, sc := range pathutil.ScenariosFromGlobs(item.AcceptanceAllow) {
				hasMaint[sc] = true
			}
		}
	}

	filtered := results[:0]
	for _, r := range results {
		if !hasMaint[r.scenarioName] && r.lastReviewAt.IsZero() {
			excludedCount++
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered, excludedCount
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

	// Build the set of scenarios that actually exist on disk so the queue
	// can filter out planned-but-not-yet-created scenarios.  If the source
	// is unavailable we degrade gracefully (nil set → fallback heuristic).
	var existingScenarios map[string]bool
	if h.source != nil {
		sources, err := h.source.List(r.Context())
		if err != nil {
			slog.Warn("review-queue: failed to load scenario source, using fallback heuristic", "error", err)
		} else {
			existingScenarios = make(map[string]bool, len(sources))
			for _, s := range sources {
				existingScenarios[s.Name] = true
			}
		}
	}

	results, totalScenarios, excludedCount := computeReviewQueue(items, records, excludeTag, limit, time.Now(), existingScenarios)

	// Fallback heuristic (Rec #2): when the scenario source was unavailable,
	// drop results that look like planned-but-not-built work — every pending
	// item is a creation kind (execute/idea/research) and no review history.
	if existingScenarios == nil {
		results, excludedCount = applyGreenfieldFallback(items, results, excludedCount)
	}

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
