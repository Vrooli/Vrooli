package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type cliOperationsView struct {
	Lanes            []cliLaneStatus  `json:"lanes"`
	Queue            cliQueueStatus   `json:"queue"`
	Activities       []cliActivityRow `json:"activities"`
	RecentlyFinished []cliActivityRow `json:"recently_finished"`
	GeneratedAt      string           `json:"generated_at"`
	WindowSeconds    int              `json:"window_seconds"`
}

type cliLaneStatus struct {
	Lane     string `json:"lane"`
	Active   int    `json:"active"`
	Capacity int    `json:"capacity"`
	Queue    int    `json:"queue"`
}

type cliQueueStatus struct {
	Depth    int `json:"depth"`
	MaxDepth int `json:"max_depth"`
}

type cliActivityRow struct {
	ActivityID     string `json:"activity_id"`
	RunID          string `json:"run_id,omitempty"`
	OwnerType      string `json:"owner_type"`
	OwnerKind      string `json:"owner_kind,omitempty"`
	OwnerName      string `json:"owner_name"`
	OwnerTitle     string `json:"owner_title,omitempty"`
	Lane           string `json:"lane,omitempty"`
	Status         string `json:"status"`
	Purpose        string `json:"purpose"`
	Mode           string `json:"mode,omitempty"`
	Phase          string `json:"phase,omitempty"`
	Round          int    `json:"round,omitempty"`
	MilestoneName  string `json:"milestone_name,omitempty"`
	RequestedAt    string `json:"requested_at"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
	RuntimeSeconds int64  `json:"runtime_seconds,omitempty"`
	FailureReason  string `json:"failure_reason,omitempty"`
}

type cliOperationsBriefing struct {
	GeneratedAt            string                         `json:"generated_at"`
	FreshnessSeconds       int                            `json:"freshness_seconds"`
	WindowSeconds          int                            `json:"window_seconds"`
	Summary                cliOperationsBriefingSummary   `json:"summary"`
	ActiveWork             []cliActivityRow               `json:"active_work"`
	NeedsAttention         []cliBriefingAttentionItem     `json:"needs_attention"`
	RecentCompletions      []cliActivityRow               `json:"recent_completions"`
	DirectorHandoffs       []cliDirectorHandoff           `json:"director_handoffs"`
	RecommendedNextActions []cliBriefingRecommendedAction `json:"recommended_next_actions"`
	DrillDownCommands      []cliBriefingDrillDownCommand  `json:"drill_down_commands"`
	Warnings               []string                       `json:"warnings"`
}

type cliOperationsBriefingSummary struct {
	ActiveActivityCount   int            `json:"active_activity_count"`
	RecentlyFinishedCount int            `json:"recently_finished_count"`
	QueueDepth            int            `json:"queue_depth"`
	MaxQueueDepth         int            `json:"max_queue_depth"`
	SaturatedLanes        []string       `json:"saturated_lanes"`
	ActiveLaneCountByLane map[string]int `json:"active_lane_count_by_lane"`
	TotalBacklogItems     int            `json:"total_backlog_items"`
	ActiveMilestones      int            `json:"active_milestones"`
	BlockedItems          int            `json:"blocked_items"`
	ActiveSessions        int            `json:"active_sessions"`
}

type cliBriefingAttentionItem struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
	Title    string `json:"title"`
	Status   string `json:"status,omitempty"`
	Lane     string `json:"lane,omitempty"`
	Ref      string `json:"ref,omitempty"`
	Command  string `json:"command,omitempty"`
}

type cliDirectorHandoff struct {
	SourcePath string `json:"source_path"`
	Title      string `json:"title"`
	ObservedAt string `json:"observed_at,omitempty"`
	Excerpt    string `json:"excerpt"`
}

type cliBriefingRecommendedAction struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Reason  string `json:"reason"`
	Command string `json:"command,omitempty"`
	UIPath  string `json:"ui_path,omitempty"`
}

type cliBriefingDrillDownCommand struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

func (a *App) cmdOperationsList(args []string) error {
	query, jsonOut, err := parseOperationsFlags("operations list", true, args)
	if err != nil {
		return err
	}
	body, err := a.core.Get("/operations", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(jsonOut, body) {
		return nil
	}
	var view cliOperationsView
	if err := json.Unmarshal(body, &view); err != nil {
		return fmt.Errorf("failed to parse operations view: %w", err)
	}
	printOperationsView(view)
	return nil
}

func (a *App) cmdOperationsBrief(args []string) error {
	query, jsonOut, err := parseOperationsFlags("operations brief", false, args)
	if err != nil {
		return err
	}
	body, err := a.core.Get("/operations/brief", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(jsonOut, body) {
		return nil
	}
	var briefing cliOperationsBriefing
	if err := json.Unmarshal(body, &briefing); err != nil {
		return fmt.Errorf("failed to parse operations briefing: %w", err)
	}
	printOperationsBriefing(briefing)
	return nil
}

func parseOperationsFlags(name string, includeFilters bool, args []string) (url.Values, bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	windowFlag := fs.String("window", "", "ISO-8601 PT duration window")
	jsonOut := cliutil.JSONFlag(fs)
	var statuses, lanes, modes, ownerTypes stringSlice
	qFlag := fs.String("q", "", "Search text")
	if includeFilters {
		fs.Var(&statuses, "status", "Activity status filter")
		fs.Var(&lanes, "lane", "Lane filter")
		fs.Var(&modes, "mode", "Operating mode filter")
		fs.Var(&ownerTypes, "owner-type", "Owner type filter")
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, false, err
	}
	query := url.Values{}
	if window := strings.TrimSpace(*windowFlag); window != "" {
		query.Set("window", window)
	}
	for _, value := range statuses {
		query.Add("status", value)
	}
	for _, value := range lanes {
		query.Add("lane", value)
	}
	for _, value := range modes {
		query.Add("mode", value)
	}
	for _, value := range ownerTypes {
		query.Add("owner_type", value)
	}
	if includeFilters {
		if q := strings.TrimSpace(*qFlag); q != "" {
			query.Set("q", q)
		}
	}
	return query, *jsonOut, nil
}

func printOperationsView(view cliOperationsView) {
	printSection("Summary")
	fmt.Printf("  Generated: %s | Window: %ds\n", view.GeneratedAt, view.WindowSeconds)
	fmt.Printf("  Active: %d | Recently finished: %d | Queue: %d/%d\n", len(view.Activities), len(view.RecentlyFinished), view.Queue.Depth, view.Queue.MaxDepth)
	printSection("Lanes")
	for _, lane := range view.Lanes {
		fmt.Printf("  %s: %d/%d active, queue %d\n", lane.Lane, lane.Active, lane.Capacity, lane.Queue)
	}
	printActivityRows("Active Work", view.Activities)
	printCommandListSection("Drill Down", []string{cliCommand("operations", "brief", "--json")})
}

func printOperationsBriefing(briefing cliOperationsBriefing) {
	printSection("Summary")
	fmt.Printf("  Generated: %s | Window: %ds\n", briefing.GeneratedAt, briefing.WindowSeconds)
	fmt.Printf("  Active: %d | Recent: %d | Queue: %d/%d | Milestones: %d active | Blocked: %d | Sessions: %d active\n",
		briefing.Summary.ActiveActivityCount,
		briefing.Summary.RecentlyFinishedCount,
		briefing.Summary.QueueDepth,
		briefing.Summary.MaxQueueDepth,
		briefing.Summary.ActiveMilestones,
		briefing.Summary.BlockedItems,
		briefing.Summary.ActiveSessions,
	)
	if len(briefing.Summary.SaturatedLanes) > 0 {
		fmt.Printf("  Saturated lanes: %s\n", strings.Join(briefing.Summary.SaturatedLanes, ", "))
	}
	if len(briefing.NeedsAttention) > 0 {
		printSection("Needs Attention")
		for _, item := range briefing.NeedsAttention {
			fmt.Printf("  [%s] %s — %s\n", item.Severity, item.Title, item.Reason)
			if item.Command != "" {
				fmt.Printf("    %s\n", item.Command)
			}
		}
	}
	printActivityRows("Active Work", briefing.ActiveWork)
	if len(briefing.RecommendedNextActions) > 0 {
		printSection("Next Actions")
		for _, action := range briefing.RecommendedNextActions {
			fmt.Printf("  %s — %s\n", action.Label, action.Reason)
			if action.Command != "" {
				fmt.Printf("    %s\n", action.Command)
			}
		}
	}
	if len(briefing.Warnings) > 0 {
		printSection("Warnings")
		for _, warning := range briefing.Warnings {
			fmt.Printf("  %s\n", warning)
		}
	}
	commands := make([]string, 0, len(briefing.DrillDownCommands))
	for _, command := range briefing.DrillDownCommands {
		commands = append(commands, command.Command)
	}
	printCommandListSection("Drill Down", commands)
}

func printActivityRows(title string, rows []cliActivityRow) {
	if len(rows) == 0 {
		return
	}
	printSection(title)
	for _, row := range rows {
		name := firstNonEmptyString(row.OwnerTitle, row.OwnerName, row.ActivityID)
		fmt.Printf("  [%s] %s  %s  %s\n", row.Status, name, row.Lane, firstNonEmptyString(row.RunID, row.ActivityID))
		if row.Mode != "" || row.Phase != "" || row.FailureReason != "" {
			fmt.Printf("    mode=%s phase=%s failure=%s\n", row.Mode, row.Phase, row.FailureReason)
		}
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
