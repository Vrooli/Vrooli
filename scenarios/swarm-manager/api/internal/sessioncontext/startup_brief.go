package sessioncontext

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/operations"
)

const (
	startupBriefFreshnessOperationsSeconds = 120
	startupBriefFreshnessPortfolioSeconds  = 600
	startupBriefFreshnessModeSeconds       = 1800
	startupBriefItemLimit                  = 6
)

type startupBriefMetadata struct {
	Kind                   string                  `json:"kind"`
	GeneratedAt            string                  `json:"generated_at"`
	StaleAfter             string                  `json:"stale_after"`
	FreshnessSeconds       int                     `json:"freshness_seconds"`
	SourceCounts           map[string]int          `json:"source_counts,omitempty"`
	RankedGoals            []rankedGoalBrief       `json:"ranked_goals,omitempty"`
	RecommendedNextActions []briefAction           `json:"recommended_next_actions,omitempty"`
	DrillDownCommands      []briefDrillDownCommand `json:"drill_down_commands,omitempty"`
	Warnings               []string                `json:"warnings,omitempty"`
}

// rankedGoalBrief is the compact per-goal ranking row embedded in
// the operations startup brief metadata. Ref carries the typed reference
// (`goal:<name>`) the agent should echo verbatim so the UI can linkify
// it; the remaining fields are the signals the snapshot ranked on.
type rankedGoalBrief struct {
	Ref       string `json:"ref"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	Priority  int    `json:"priority"`
	Readiness string `json:"readiness"`
}

// maxStartupBriefRankedGoals bounds how many ranked goals land in the brief.
const maxStartupBriefRankedGoals = 8

type briefAction struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Reason  string `json:"reason"`
	Command string `json:"command,omitempty"`
	UIPath  string `json:"ui_path,omitempty"`
}

type briefDrillDownCommand struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

func (r *Resolver) ResolveSessionStartupBrief(ctx context.Context, kind agentsessions.Kind, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	switch kind {
	case agentsessions.KindSwarmOperations:
		return r.operationsStartupBrief(ctx, limits)
	case agentsessions.KindMetaOrchestration:
		return r.portfolioStartupBrief(limits)
	case agentsessions.KindWorkflowAuthoring:
		return r.workflowAuthoringStartupBrief(limits)
	default:
		return agentsessions.ContextItem{}, fmt.Errorf("%w: unsupported startup brief kind", agentsessions.ErrValidation)
	}
}

func kindForStartupBriefRef(ref string) (agentsessions.Kind, error) {
	switch strings.TrimSpace(ref) {
	case agentsessions.StartupBriefSwarmOperationsRef:
		return agentsessions.KindSwarmOperations, nil
	case agentsessions.StartupBriefMetaOrchestrationRef:
		return agentsessions.KindMetaOrchestration, nil
	case agentsessions.StartupBriefWorkflowAuthoringRef:
		return agentsessions.KindWorkflowAuthoring, nil
	default:
		return "", fmt.Errorf("%w: unknown startup brief ref %q", agentsessions.ErrValidation, ref)
	}
}

func (r *Resolver) operationsStartupBrief(ctx context.Context, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	if r.briefings == nil {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: operations startup brief is unavailable", agentsessions.ErrValidation)
	}
	briefing, err := r.briefings.Build(ctx, operations.Filters{})
	if err != nil {
		return agentsessions.ContextItem{}, err
	}
	generated := briefing.GeneratedAt.UTC()
	metadata := startupBriefMetadata{
		Kind:             string(agentsessions.KindSwarmOperations),
		GeneratedAt:      generated.Format(time.RFC3339),
		StaleAfter:       generated.Add(time.Duration(briefing.FreshnessSeconds) * time.Second).Format(time.RFC3339),
		FreshnessSeconds: briefing.FreshnessSeconds,
		SourceCounts: map[string]int{
			"active_work":        len(briefing.ActiveWork),
			"needs_attention":    len(briefing.NeedsAttention),
			"recent_completions": len(briefing.RecentCompletions),
			"director_handoffs":  len(briefing.DirectorHandoffs),
		},
		Warnings: briefing.Warnings,
	}
	for _, action := range briefing.RecommendedNextActions {
		metadata.RecommendedNextActions = append(metadata.RecommendedNextActions, briefAction(action))
	}
	for _, command := range briefing.DrillDownCommands {
		metadata.DrillDownCommands = append(metadata.DrillDownCommands, briefDrillDownCommand(command))
	}

	summary := formatOperationsBriefingSummary(briefing)
	// Augment the activity briefing with the ranked goal snapshot so
	// the agent receives deterministic rankings instead of re-deriving them
	// per turn. The ranked section is prepended: it is the headline value of
	// this brief, and the summary is truncated to a rune budget — leading with
	// rankings keeps them from being clipped when the activity detail is long.
	// The full structured ranking always rides in metadata regardless of
	// truncation. The snapshot is optional: when unavailable the brief degrades
	// to the activity briefing alone with a recorded warning.
	if r.snapshots != nil {
		if snap, snapErr := r.snapshots.GetSnapshot(ctx); snapErr != nil {
			metadata.Warnings = append(metadata.Warnings, "ranked goals unavailable: "+snapErr.Error())
		} else {
			ranked := rankedGoalBriefs(snap)
			metadata.RankedGoals = ranked
			metadata.SourceCounts["ranked_goals"] = len(snap.Goals)
			summary = formatRankedGoals(snap, ranked) + summary
		}
	}

	return startupContextItem(
		agentsessions.KindSwarmOperations,
		"Swarm operations startup brief",
		summary,
		"/operations",
		metadata,
		limits,
	)
}

// rankedGoalBriefs stamps compact goal ranking rows with typed goal refs.
func rankedGoalBriefs(snap *operations.OperationsSnapshot) []rankedGoalBrief {
	limit := maxStartupBriefRankedGoals
	if len(snap.Goals) < limit {
		limit = len(snap.Goals)
	}
	out := make([]rankedGoalBrief, 0, limit)
	for _, goal := range snap.Goals[:limit] {
		out = append(out, rankedGoalBrief{
			Ref: "goal:" + goal.Name, Name: goal.Name, Title: goal.Title,
			Priority: goal.Priority, Readiness: goal.Readiness,
		})
	}
	return out
}

// formatRankedGoals renders the ranked head as a human-readable section
// appended to the operations brief summary. Each line leads with the typed
// `goal:<name>` reference so an agent quoting it produces a span the UI
// linkifies.
func formatRankedGoals(snap *operations.OperationsSnapshot, rows []rankedGoalBrief) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Ranked goals (%d ready, %d blocked of %d total):\n",
		snap.Summary.ReadyGoals, snap.Summary.BlockedGoals, snap.Summary.TotalGoals)
	for _, row := range rows {
		priority := "unprioritized"
		if row.Priority > 0 {
			priority = fmt.Sprintf("P%d", row.Priority)
		}
		fmt.Fprintf(&b, "- `goal:%s` [%s %s]: %s\n", row.Name, priority, row.Readiness, firstNonEmpty(row.Title, row.Name))
	}
	return b.String()
}

func (r *Resolver) portfolioStartupBrief(limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	backlogStore := backlog.NewFileStore(r.scenarioRoot)
	goalService := goals.NewService(goals.NewStore(r.scenarioRoot), backlogStore)
	goalList, goalErr := goalService.List()
	items, itemErr := backlogStore.LoadAll(nil)
	warnings := warningStrings(goalErr, itemErr)
	now := time.Now().UTC()

	statusCounts := map[string]int{}
	for _, item := range items {
		statusCounts[string(item.Status)]++
	}
	goalStatusCounts := map[string]int{}
	for _, item := range goalList {
		goalStatusCounts[item.Goal.Status]++
	}

	sort.Slice(goalList, func(i, j int) bool {
		if goalList[i].Goal.Priority == goalList[j].Goal.Priority {
			return goalList[i].Goal.Updated > goalList[j].Goal.Updated
		}
		return goalList[i].Goal.Priority > goalList[j].Goal.Priority
	})
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].Updated > items[j].Updated
		}
		return items[i].Priority > items[j].Priority
	})

	var b strings.Builder
	fmt.Fprintf(&b, "Generated: %s. Goals: %d. Backlog items: %d.\n", now.Format(time.RFC3339), len(goalList), len(items))
	writeCountMap(&b, "Goal status counts", goalStatusCounts)
	writeCountMap(&b, "Backlog status counts", statusCounts)
	if len(goalList) > 0 {
		b.WriteString("Top goals:\n")
		for _, item := range take(goalList, startupBriefItemLimit) {
			goal := item.Goal
			fmt.Fprintf(&b, "- %s [%s priority=%d]: %s. Scope total=%d completed=%d ready=%d blocked=%d\n",
				goal.Name, goal.Status, goal.Priority, goal.Title,
				item.Scope.Total, item.Scope.CompletedCount, len(item.Scope.Ready), item.Scope.BlockedCount)
		}
	}
	if len(items) > 0 {
		b.WriteString("High-priority backlog candidates:\n")
		for _, item := range take(items, startupBriefItemLimit) {
			fmt.Fprintf(&b, "- %s/%s [%s priority=%d]: %s", item.Kind, item.Name, item.Status, item.Priority, item.Title)
			if len(item.DependsOn) > 0 {
				fmt.Fprintf(&b, " depends_on=%s", strings.Join(item.DependsOn, ","))
			}
			b.WriteString("\n")
		}
	}
	metadata := startupBriefMetadata{
		Kind:             string(agentsessions.KindMetaOrchestration),
		GeneratedAt:      now.Format(time.RFC3339),
		StaleAfter:       now.Add(startupBriefFreshnessPortfolioSeconds * time.Second).Format(time.RFC3339),
		FreshnessSeconds: startupBriefFreshnessPortfolioSeconds,
		SourceCounts: map[string]int{
			"goals":         len(goalList),
			"backlog_items": len(items),
		},
		RecommendedNextActions: []briefAction{
			{ID: "plan-from-brief", Label: "Plan from brief", Reason: "Use this bounded portfolio snapshot before scanning broad lists."},
			{ID: "check-candidates", Label: "Check next-action candidates", Reason: "Use a targeted goal or backlog command only after identifying a likely scope.", Command: "swarm-manager goals list --json"},
		},
		DrillDownCommands: []briefDrillDownCommand{
			{Label: "Goals", Command: "swarm-manager goals list --json"},
			{Label: "Pending questions", Command: "swarm-manager backlog pending-questions --brief --json"},
			{Label: "Stats", Command: "swarm-manager stats summary --json"},
		},
		Warnings: warnings,
	}
	return startupContextItem(agentsessions.KindMetaOrchestration, "Portfolio startup brief", b.String(), "/goals", metadata, limits)
}

// workflowAuthoringStartupBrief gives the authoring conversation the durable
// operating model and the current declared catalog, without asking the agent
// to infer its authority from a broad project snapshot.
func (r *Resolver) workflowAuthoringStartupBrief(limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	now := time.Now().UTC()
	metadata := startupBriefMetadata{
		Kind:             string(agentsessions.KindWorkflowAuthoring),
		GeneratedAt:      now.Format(time.RFC3339),
		StaleAfter:       now.Add(startupBriefFreshnessModeSeconds * time.Second).Format(time.RFC3339),
		FreshnessSeconds: startupBriefFreshnessModeSeconds,
		RecommendedNextActions: []briefAction{
			{ID: "read-operating-model", Label: "Read the target operating model", Reason: "Establish the session/workflow boundary before proposing a change.", Command: "sed -n '1,220p' docs/concepts/TARGET-OPERATING-MODEL.md"},
			{ID: "inspect-transition-catalog", Label: "Inspect registered transitions", Reason: "Prefer improving an existing declared transition over inventing a parallel method.", Command: "cat .vrooli/swarm-transitions/registry.json"},
		},
		DrillDownCommands: []briefDrillDownCommand{
			{Label: "Target operating model", Command: "sed -n '1,220p' docs/concepts/TARGET-OPERATING-MODEL.md"},
			{Label: "Transition registry", Command: "cat .vrooli/swarm-transitions/registry.json"},
			{Label: "Workflow declarations", Command: "find .vrooli/agent-manager -maxdepth 1 -name '*.json' -print"},
		},
	}
	summary := "Workflow authoring is a human-led design conversation. Use it to compare a natural agent-working method with Swarm's target operating model and declared transition catalog. Propose an existing-workflow improvement, a new reviewed transition/workflow, or a backlog item when Swarm lacks the required deterministic domain adapter. Do not treat this session as permission to execute or silently mutate workflow declarations.\n\nAuthoritative references:\n- docs/concepts/TARGET-OPERATING-MODEL.md\n- .vrooli/swarm-transitions/registry.json\n- .vrooli/agent-manager/*.json\n"
	return startupContextItem(agentsessions.KindWorkflowAuthoring, "Workflow authoring startup brief", summary, "/sessions", metadata, limits)
}

func startupContextItem(kind agentsessions.Kind, title, summary, nodeID string, metadata startupBriefMetadata, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return agentsessions.ContextItem{}, err
	}
	return agentsessions.ContextItem{
		Type:         agentsessions.ContextStartupBrief,
		Ref:          agentsessions.StartupBriefRefForKind(kind),
		Title:        title,
		Summary:      truncate(summary, limits.MaxSummaryRunes),
		NodeID:       nodeID,
		MetadataJSON: string(data),
	}, nil
}

func warningStrings(errs ...error) []string {
	var warnings []string
	for _, err := range errs {
		if err != nil {
			warnings = append(warnings, err.Error())
		}
	}
	return warnings
}

func writeCountMap(b *strings.Builder, label string, counts map[string]int) {
	if len(counts) == 0 {
		return
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	fmt.Fprintf(b, "%s: %s.\n", label, strings.Join(parts, ", "))
}

func take[T any](items []T, limit int) []T {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}
