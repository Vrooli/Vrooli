package operations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/overview"
	"swarm-manager/internal/stats"
)

const (
	maxBriefingActiveWork        = 8
	maxBriefingAttention         = 8
	maxBriefingRecentCompletions = 5
	maxBriefingHandoffs          = 4
	maxHandoffExcerptRunes       = 700
)

type OverviewReader interface {
	GetOverview() (*overview.OverviewResponse, error)
}

type StatsReader interface {
	Refresh(context.Context) error
	GetStats() stats.StatsResponse
}

type DirectorHandoffReader interface {
	ReadDirectorHandoffs(ctx context.Context) ([]DirectorHandoff, []string)
}

type BriefingBuilderConfig struct {
	Aggregator *Aggregator
	Overview   OverviewReader
	Stats      StatsReader
	Handoffs   DirectorHandoffReader
	Now        func() time.Time
}

type BriefingBuilder struct {
	cfg BriefingBuilderConfig
}

func NewBriefingBuilder(cfg BriefingBuilderConfig) (*BriefingBuilder, error) {
	if cfg.Aggregator == nil {
		return nil, fmt.Errorf("operations: BriefingBuilderConfig.Aggregator is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &BriefingBuilder{cfg: cfg}, nil
}

func (b *BriefingBuilder) Build(ctx context.Context, filters Filters) (*OperationsBriefing, error) {
	view, err := b.cfg.Aggregator.Aggregate(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("operations briefing: aggregate operations: %w", err)
	}

	now := b.cfg.Now().UTC()
	warnings := []string{}
	summary := OperationsBriefingSummary{
		ActiveActivityCount:   len(view.Activities),
		RecentlyFinishedCount: len(view.RecentlyFinished),
		QueueDepth:            view.Queue.Depth,
		MaxQueueDepth:         view.Queue.MaxDepth,
		SaturatedLanes:        saturatedLanes(view.Lanes),
		ActiveLaneCountByLane: activeLaneCounts(view.Lanes),
	}

	if b.cfg.Overview != nil {
		if ov, err := b.cfg.Overview.GetOverview(); err == nil && ov != nil {
			summary.TotalBacklogItems = ov.Summary.TotalItems
			summary.ActiveInitiatives = ov.Summary.ActiveInitiatives
			summary.BlockedItems = len(ov.DependencyGraph.Blocked)
		} else if err != nil {
			warnings = append(warnings, "overview unavailable: "+err.Error())
		}
	} else {
		warnings = append(warnings, "overview source unavailable")
	}

	if b.cfg.Stats != nil {
		if err := b.cfg.Stats.Refresh(ctx); err != nil {
			warnings = append(warnings, "stats refresh unavailable: "+err.Error())
		} else {
			st := b.cfg.Stats.GetStats()
			summary.ActiveSessions = st.Session.ActiveSessions
		}
	} else {
		warnings = append(warnings, "stats source unavailable")
	}

	handoffs := []DirectorHandoff{}
	if b.cfg.Handoffs != nil {
		var handoffWarnings []string
		handoffs, handoffWarnings = b.cfg.Handoffs.ReadDirectorHandoffs(ctx)
		warnings = append(warnings, handoffWarnings...)
	} else {
		warnings = append(warnings, "director handoff source unavailable")
	}

	active := briefingActivities(view.Activities, maxBriefingActiveWork)
	recent := briefingActivities(view.RecentlyFinished, maxBriefingRecentCompletions)
	attention := needsAttention(view, summary)

	return &OperationsBriefing{
		GeneratedAt:            now,
		FreshnessSeconds:       0,
		WindowSeconds:          view.WindowSeconds,
		Summary:                summary,
		ActiveWork:             active,
		NeedsAttention:         attention,
		RecentCompletions:      recent,
		DirectorHandoffs:       handoffs,
		RecommendedNextActions: recommendedActions(attention, summary),
		DrillDownCommands:      drillDownCommands(),
		Warnings:               uniqueStrings(warnings),
	}, nil
}

func saturatedLanes(lanes []LaneStatus) []string {
	out := []string{}
	for _, lane := range lanes {
		if lane.Capacity > 0 && lane.Active >= lane.Capacity {
			out = append(out, lane.Lane)
		}
	}
	sort.Strings(out)
	return out
}

func activeLaneCounts(lanes []LaneStatus) map[string]int {
	out := make(map[string]int, len(lanes))
	for _, lane := range lanes {
		out[lane.Lane] = lane.Active
	}
	return out
}

func briefingActivities(rows []ActivityRow, limit int) []BriefingActivity {
	if len(rows) == 0 || limit <= 0 {
		return []BriefingActivity{}
	}
	capacity := len(rows)
	if capacity > limit {
		capacity = limit
	}
	out := make([]BriefingActivity, 0, capacity)
	for i, row := range rows {
		if i >= limit {
			break
		}
		out = append(out, BriefingActivity{
			ActivityID:     row.ActivityID,
			RunID:          row.RunID,
			OwnerType:      row.OwnerType,
			OwnerKind:      row.OwnerKind,
			OwnerName:      row.OwnerName,
			OwnerTitle:     row.OwnerTitle,
			Lane:           row.Lane,
			Status:         row.Status,
			Purpose:        row.Purpose,
			Mode:           row.Mode,
			Phase:          row.Phase,
			Round:          row.Round,
			InitiativeName: row.InitiativeName,
			RequestedAt:    row.RequestedAt,
			StartedAt:      row.StartedAt,
			FinishedAt:     row.FinishedAt,
			RuntimeSeconds: row.RuntimeSeconds,
			FailureReason:  row.FailureReason,
		})
	}
	return out
}

func needsAttention(view *OperationsView, summary OperationsBriefingSummary) []BriefingAttentionItem {
	out := []BriefingAttentionItem{}
	for _, lane := range summary.SaturatedLanes {
		out = append(out, BriefingAttentionItem{
			ID:       "lane/" + lane,
			Severity: "medium",
			Reason:   "lane_saturated",
			Title:    fmt.Sprintf("%s lane is at capacity", lane),
			Lane:     lane,
			Command:  "swarm-manager operations list --lane " + lane,
		})
	}
	if view.Queue.MaxDepth > 0 && view.Queue.Depth >= view.Queue.MaxDepth {
		out = append(out, BriefingAttentionItem{
			ID:       "queue/full",
			Severity: "high",
			Reason:   "queue_full",
			Title:    "Execution queue is full",
			Command:  "swarm-manager queue list",
		})
	}
	for _, row := range append(append([]ActivityRow{}, view.Activities...), view.RecentlyFinished...) {
		if len(out) >= maxBriefingAttention {
			break
		}
		if row.Status != "failed" && row.Status != "needs_review" && row.FailureReason == "" {
			continue
		}
		reason := row.Status
		if reason == "" {
			reason = "needs_attention"
		}
		out = append(out, BriefingAttentionItem{
			ID:       firstNonEmpty(row.RunID, row.ActivityID),
			Severity: severityForStatus(row.Status),
			Reason:   reason,
			Title:    activityTitle(row),
			Status:   row.Status,
			Lane:     row.Lane,
			Ref:      firstNonEmpty(row.RunID, row.ActivityID),
			Command:  commandForActivity(row),
		})
	}
	return out
}

func recommendedActions(attention []BriefingAttentionItem, summary OperationsBriefingSummary) []BriefingRecommendedAction {
	out := []BriefingRecommendedAction{}
	if len(attention) > 0 {
		first := attention[0]
		out = append(out, BriefingRecommendedAction{
			ID:      "review-attention",
			Label:   "Review the highest attention item",
			Reason:  first.Title,
			Command: first.Command,
		})
	}
	if summary.BlockedItems > 0 {
		out = append(out, BriefingRecommendedAction{
			ID:      "inspect-blocked",
			Label:   "Inspect blocked backlog dependencies",
			Reason:  fmt.Sprintf("%d backlog item(s) are dependency-blocked", summary.BlockedItems),
			Command: "swarm-manager overview",
		})
	}
	out = append(out, BriefingRecommendedAction{
		ID:      "drill-down-active",
		Label:   "Drill into active operations only if more detail is needed",
		Reason:  "The briefing already includes the bounded current-state packet",
		Command: "swarm-manager operations list --json",
		UIPath:  "/operations",
	})
	if len(out) > 4 {
		return out[:4]
	}
	return out
}

func drillDownCommands() []BriefingDrillDownCommand {
	return []BriefingDrillDownCommand{
		{Label: "Refresh this briefing", Command: "swarm-manager operations brief --json"},
		{Label: "List active operations", Command: "swarm-manager operations list --json"},
		{Label: "Inspect scenario overview", Command: "swarm-manager overview --json"},
		{Label: "Inspect session stats", Command: "swarm-manager stats sessions --json"},
	}
}

type FileDirectorHandoffReader struct {
	ProjectRoot string
	Now         func() time.Time
}

func (r FileDirectorHandoffReader) ReadDirectorHandoffs(_ context.Context) ([]DirectorHandoff, []string) {
	root := strings.TrimSpace(r.ProjectRoot)
	if root == "" {
		root = "."
	}
	base := filepath.Join(root, "scenarios", "prompt-manager", "store", "teams", "director-swarm", "members")
	entries, err := os.ReadDir(base)
	if err != nil {
		return []DirectorHandoff{}, []string{"director handoffs unavailable: " + err.Error()}
	}
	type candidate struct {
		path string
		info os.FileInfo
	}
	candidates := []candidate{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(base, entry.Name(), "last-handoff.md")
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate{path: path, info: info})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].info.ModTime().After(candidates[j].info.ModTime())
	})
	if len(candidates) > maxBriefingHandoffs {
		candidates = candidates[:maxBriefingHandoffs]
	}
	out := make([]DirectorHandoff, 0, len(candidates))
	warnings := []string{}
	for _, item := range candidates {
		data, err := os.ReadFile(item.path)
		if err != nil {
			warnings = append(warnings, "director handoff unreadable: "+item.path)
			continue
		}
		rel, _ := filepath.Rel(root, item.path)
		out = append(out, DirectorHandoff{
			SourcePath: filepath.ToSlash(rel),
			Title:      titleFromPath(item.path),
			ObservedAt: item.info.ModTime().UTC().Format(time.RFC3339),
			Excerpt:    truncateRunes(strings.TrimSpace(string(data)), maxHandoffExcerptRunes),
		})
	}
	return out, warnings
}

func titleFromPath(path string) string {
	member := filepath.Base(filepath.Dir(path))
	title := strings.ReplaceAll(member, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	return strings.TrimSpace(title)
}

func activityTitle(row ActivityRow) string {
	if strings.TrimSpace(row.OwnerTitle) != "" {
		return strings.TrimSpace(row.OwnerTitle)
	}
	if row.OwnerKind != "" && row.OwnerName != "" {
		return row.OwnerKind + "/" + row.OwnerName
	}
	return firstNonEmpty(row.OwnerName, row.RunID, row.ActivityID)
}

func commandForActivity(row ActivityRow) string {
	if strings.TrimSpace(row.RunID) != "" {
		return "swarm-manager agent-manager run-get --id " + row.RunID
	}
	if strings.TrimSpace(row.ActivityID) != "" {
		return "swarm-manager operations list --q " + row.ActivityID
	}
	return "swarm-manager operations list --json"
}

func severityForStatus(status string) string {
	switch status {
	case "failed":
		return "high"
	case "needs_review":
		return "medium"
	default:
		return "low"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
