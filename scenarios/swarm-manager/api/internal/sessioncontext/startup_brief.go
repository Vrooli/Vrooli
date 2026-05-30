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
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/operatingmode"
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
	RankedInitiatives      []rankedInitiativeBrief `json:"ranked_initiatives,omitempty"`
	RecommendedNextActions []briefAction           `json:"recommended_next_actions,omitempty"`
	DrillDownCommands      []briefDrillDownCommand `json:"drill_down_commands,omitempty"`
	Warnings               []string                `json:"warnings,omitempty"`
}

// rankedInitiativeBrief is the compact per-initiative ranking row embedded in
// the operations startup brief metadata. Ref carries the typed reference
// (`initiative:<name>`) the agent should echo verbatim so the UI can linkify
// it; the remaining fields are the signals the snapshot ranked on.
type rankedInitiativeBrief struct {
	Ref                string `json:"ref"`
	Name               string `json:"name"`
	Title              string `json:"title"`
	Priority           int    `json:"priority"`
	Readiness          string `json:"readiness"`
	DownstreamUnblocks int    `json:"downstream_unblocks"`
}

// maxStartupBriefRankedInitiatives bounds how many ranked initiatives land in
// the brief so a large portfolio doesn't blow the session context budget. The
// agent drills the long tail via `swarm-manager initiatives list`.
const maxStartupBriefRankedInitiatives = 8

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
	case agentsessions.KindOperatingModeAuthoring:
		return r.operatingModeStartupBrief(limits)
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
	case agentsessions.StartupBriefOperatingModeAuthoringRef:
		return agentsessions.KindOperatingModeAuthoring, nil
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
	// Augment the activity briefing with the ranked initiative snapshot so
	// the agent receives deterministic rankings instead of re-deriving them
	// per turn. The ranked section is prepended: it is the headline value of
	// this brief, and the summary is truncated to a rune budget — leading with
	// rankings keeps them from being clipped when the activity detail is long.
	// The full structured ranking always rides in metadata regardless of
	// truncation. The snapshot is optional: when unavailable the brief degrades
	// to the activity briefing alone with a recorded warning.
	if r.snapshots != nil {
		if snap, snapErr := r.snapshots.GetSnapshot(ctx); snapErr != nil {
			metadata.Warnings = append(metadata.Warnings, "ranked initiatives unavailable: "+snapErr.Error())
		} else {
			ranked := rankedInitiativeBriefs(snap)
			metadata.RankedInitiatives = ranked
			metadata.SourceCounts["ranked_initiatives"] = len(snap.Initiatives)
			summary = formatRankedInitiatives(snap, ranked) + summary
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

// rankedInitiativeBriefs projects the bounded head of the snapshot's ranked
// initiatives into the compact brief rows, stamping each with its typed
// `initiative:<name>` reference for downstream linkification.
func rankedInitiativeBriefs(snap *operations.OperationsSnapshot) []rankedInitiativeBrief {
	limit := maxStartupBriefRankedInitiatives
	if len(snap.Initiatives) < limit {
		limit = len(snap.Initiatives)
	}
	out := make([]rankedInitiativeBrief, 0, limit)
	for _, ri := range snap.Initiatives[:limit] {
		out = append(out, rankedInitiativeBrief{
			Ref:                "initiative:" + ri.Name,
			Name:               ri.Name,
			Title:              ri.Title,
			Priority:           ri.Priority,
			Readiness:          ri.Readiness,
			DownstreamUnblocks: ri.DownstreamUnblocks,
		})
	}
	return out
}

// formatRankedInitiatives renders the ranked head as a human-readable section
// appended to the operations brief summary. Each line leads with the typed
// `initiative:<name>` reference so an agent quoting it produces a span the UI
// linkifies.
func formatRankedInitiatives(snap *operations.OperationsSnapshot, rows []rankedInitiativeBrief) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Ranked initiatives (%d ready, %d blocked of %d total):\n",
		snap.Summary.ReadyInitiatives, snap.Summary.BlockedInitiatives, snap.Summary.TotalInitiatives)
	for _, row := range rows {
		priority := "unprioritized"
		if row.Priority > 0 {
			priority = fmt.Sprintf("P%d", row.Priority)
		}
		fmt.Fprintf(&b, "- `initiative:%s` [%s %s]: %s (unblocks %d)\n",
			row.Name, priority, row.Readiness, firstNonEmpty(row.Title, row.Name), row.DownstreamUnblocks)
	}
	return b.String()
}

func (r *Resolver) portfolioStartupBrief(limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	backlogStore := backlog.NewFileStore(r.scenarioRoot)
	initService := initiatives.NewService(initiatives.NewStore(r.scenarioRoot), backlogStore)
	inits, initErr := initService.List()
	items, itemErr := backlogStore.LoadAll(nil)
	warnings := warningStrings(initErr, itemErr)
	now := time.Now().UTC()

	statusCounts := map[string]int{}
	for _, item := range items {
		statusCounts[string(item.Status)]++
	}
	initStatusCounts := map[string]int{}
	for _, item := range inits {
		initStatusCounts[item.Initiative.Status]++
	}

	sort.Slice(inits, func(i, j int) bool {
		if inits[i].Initiative.Priority == inits[j].Initiative.Priority {
			return inits[i].Initiative.Updated > inits[j].Initiative.Updated
		}
		return inits[i].Initiative.Priority > inits[j].Initiative.Priority
	})
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].Updated > items[j].Updated
		}
		return items[i].Priority > items[j].Priority
	})

	var b strings.Builder
	fmt.Fprintf(&b, "Generated: %s. Initiatives: %d. Backlog items: %d.\n", now.Format(time.RFC3339), len(inits), len(items))
	writeCountMap(&b, "Initiative status counts", initStatusCounts)
	writeCountMap(&b, "Backlog status counts", statusCounts)
	if len(inits) > 0 {
		b.WriteString("Top initiatives:\n")
		for _, item := range take(inits, startupBriefItemLimit) {
			init := item.Initiative
			fmt.Fprintf(&b, "- %s [%s priority=%d mode=%s]: %s. Rollup total=%d completed=%d failed=%d in_progress=%d pending=%d\n",
				init.Name, init.Status, init.Priority, firstNonEmpty(init.Mode, "item-level"), init.Title,
				item.Rollup.Total, item.Rollup.Completed, item.Rollup.Failed, item.Rollup.InProgress, item.Rollup.Pending)
		}
	}
	if len(items) > 0 {
		b.WriteString("High-priority backlog candidates:\n")
		for _, item := range take(items, startupBriefItemLimit) {
			fmt.Fprintf(&b, "- %s/%s [%s priority=%d]: %s", item.Kind, item.Name, item.Status, item.Priority, item.Title)
			if item.Initiative != "" {
				fmt.Fprintf(&b, " initiative=%s", item.Initiative)
			}
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
			"initiatives":   len(inits),
			"backlog_items": len(items),
		},
		RecommendedNextActions: []briefAction{
			{ID: "plan-from-brief", Label: "Plan from brief", Reason: "Use this bounded portfolio snapshot before scanning broad lists."},
			{ID: "check-candidates", Label: "Check next-action candidates", Reason: "Use a targeted initiative or backlog command only after identifying a likely scope.", Command: "swarm-manager initiatives list --json"},
		},
		DrillDownCommands: []briefDrillDownCommand{
			{Label: "Initiatives", Command: "swarm-manager initiatives list --json"},
			{Label: "Pending questions", Command: "swarm-manager backlog pending-questions --brief --json"},
			{Label: "Stats", Command: "swarm-manager stats summary --json"},
		},
		Warnings: warnings,
	}
	return startupContextItem(agentsessions.KindMetaOrchestration, "Portfolio startup brief", b.String(), "/initiatives", metadata, limits)
}

func (r *Resolver) operatingModeStartupBrief(limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	now := time.Now().UTC()
	if err := operatingmode.ValidateRegistry(); err != nil {
		return agentsessions.ContextItem{}, err
	}
	modes := operatingmode.Modes()
	var b strings.Builder
	fmt.Fprintf(&b, "Generated: %s. Registered operating modes: %d.\n", now.Format(time.RFC3339), len(modes))
	for _, mode := range modes {
		def, err := operatingmode.DefinitionFor(mode)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s. Scope=%s strategy=%s phases=%d.\n", def.Mode, def.Label, def.Scope.Kind, def.RunStrategy.Kind, len(def.PhaseGraph.Phases))
		if len(def.BestFor) > 0 {
			fmt.Fprintf(&b, "  Best for: %s\n", strings.Join(def.BestFor, "; "))
		}
		if len(def.NotFor) > 0 {
			fmt.Fprintf(&b, "  Not for: %s\n", strings.Join(def.NotFor, "; "))
		}
		if def.WhenInDoubtPickInstead != "" {
			fmt.Fprintf(&b, "  When in doubt pick instead: %s\n", def.WhenInDoubtPickInstead)
		}
	}
	metadata := startupBriefMetadata{
		Kind:             string(agentsessions.KindOperatingModeAuthoring),
		GeneratedAt:      now.Format(time.RFC3339),
		StaleAfter:       now.Add(startupBriefFreshnessModeSeconds * time.Second).Format(time.RFC3339),
		FreshnessSeconds: startupBriefFreshnessModeSeconds,
		SourceCounts:     map[string]int{"operating_modes": len(modes)},
		RecommendedNextActions: []briefAction{
			{ID: "classify-first", Label: "Classify before authoring", Reason: "Compare the requested workflow with existing modes before proposing a new mode."},
			{ID: "reuse-existing", Label: "Prefer reuse", Reason: "Recommend an existing mode unless the workflow needs a distinct phase graph, artifact contract, or governance policy."},
		},
		DrillDownCommands: []briefDrillDownCommand{
			{Label: "Mode catalog", Command: "swarm-manager operating-mode list --json"},
			{Label: "Mode detail", Command: "swarm-manager operating-mode get --mode <mode> --json"},
		},
	}
	return startupContextItem(agentsessions.KindOperatingModeAuthoring, "Operating mode authoring startup brief", b.String(), "/operating-modes", metadata, limits)
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
