package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// StatsResponse mirrors the API stats endpoint payload.
type StatsResponse struct {
	GeneratedAt string          `json:"generated_at"`
	EventCount  int64           `json:"event_count"`
	Throughput  ThroughputStats `json:"throughput"`
	Timing      TimingStats     `json:"timing"`
	Scope       ScopeStats      `json:"scope"`
	Blocking    BlockingStats   `json:"blocking"`
	Agent       AgentStats      `json:"agent"`
	Dashboard   DashboardStats  `json:"dashboard"`
}

type ThroughputStats struct {
	CompletedLast7Days  int `json:"completed_last_7_days"`
	CompletedLast30Days int `json:"completed_last_30_days"`
	CreatedLast7Days    int `json:"created_last_7_days"`
	CreatedLast30Days   int `json:"created_last_30_days"`
	NetDelta7Days       int `json:"net_delta_7_days"`
	NetDelta30Days      int `json:"net_delta_30_days"`
}

type TimingStats struct {
	AvgCycleTimeHours    float64 `json:"avg_cycle_time_hours"`
	AvgLeadTimeHours     float64 `json:"avg_lead_time_hours"`
	AvgQueueWaitHours    float64 `json:"avg_queue_wait_hours"`
	MedianCycleTimeHours float64 `json:"median_cycle_time_hours"`
	MedianLeadTimeHours  float64 `json:"median_lead_time_hours"`
}

type ScopeStats struct {
	Initiatives        []InitiativeHealth `json:"initiatives"`
	MaxDependencyDepth int                `json:"max_dependency_depth"`
}

type InitiativeHealth struct {
	Name       string  `json:"name"`
	Total      int     `json:"total"`
	Completed  int     `json:"completed"`
	InProgress int     `json:"in_progress"`
	Blocked    int     `json:"blocked"`
	ScopeCreep float64 `json:"scope_creep"`
}

type BlockingStats struct {
	CurrentlyBlocked int           `json:"currently_blocked"`
	BlockedRatio     float64       `json:"blocked_ratio"`
	TopReasons       []ReasonCount `json:"top_reasons"`
	AvgBlockHours    float64       `json:"avg_block_hours"`
}

type ReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type AgentStats struct {
	TotalExecutions                    int                 `json:"total_executions"`
	SuccessRate                        float64             `json:"success_rate"`
	FailureRate                        float64             `json:"failure_rate"`
	FollowUpRate                       float64             `json:"follow_up_rate"`
	AvgExecutionMinutes                float64             `json:"avg_execution_minutes"`
	AvgWorkshopRounds                  float64             `json:"avg_workshop_rounds"`
	RecommendationAcceptanceRate       float64             `json:"recommendation_acceptance_rate"`
	RecommendationAcceptanceSampleSize int                 `json:"recommendation_acceptance_sample_size"`
	FreeformOverrideRate               float64             `json:"freeform_override_rate"`
	RecommendationAcceptanceByKind     map[string]KindRate `json:"recommendation_acceptance_by_kind"`
}

type KindRate struct {
	Rate       float64 `json:"rate"`
	SampleSize int     `json:"sample_size"`
}

type DashboardStats struct {
	TotalBacklogSize        int             `json:"total_backlog_size"`
	TotalCompletedAllTime   int             `json:"total_completed_all_time"`
	VelocityTrend           []VelocityPoint `json:"velocity_trend"`
	EstimatedWeeksRemaining float64         `json:"estimated_weeks_remaining"`
}

type VelocityPoint struct {
	WeekStart string `json:"week_start"`
	Completed int    `json:"completed"`
}

func (a *App) fetchStats(category string) ([]byte, error) {
	var q url.Values
	if category != "" {
		q = url.Values{"category": {category}}
	}
	return a.core.Get("/stats", q)
}

func (a *App) cmdStatsSummary(args []string) error {
	fs := flag.NewFlagSet("stats summary", flag.ContinueOnError)
	formatFlag := fs.String("format", "markdown", "Output format: json or markdown")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.fetchStats("")
	if err != nil {
		return err
	}

	// --json is a shortcut for --format json
	if *jsonOut || *formatFlag == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp StatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse stats: %w", err)
	}
	printStatsSummaryMarkdown(resp)
	return nil
}

func (a *App) cmdStatsThroughput(args []string) error {
	return a.statsCategoryCommand("throughput", args)
}

func (a *App) cmdStatsBlocking(args []string) error {
	return a.statsCategoryCommand("blocking", args)
}

func (a *App) cmdStatsInitiatives(args []string) error {
	return a.statsCategoryCommand("scope", args)
}

func (a *App) cmdStatsAgent(args []string) error {
	return a.statsCategoryCommand("agent", args)
}

func (a *App) statsCategoryCommand(category string, args []string) error {
	fs := flag.NewFlagSet("stats "+category, flag.ContinueOnError)
	formatFlag := fs.String("format", "markdown", "Output format: json or markdown")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.fetchStats(category)
	if err != nil {
		return err
	}

	// --json is a shortcut for --format json
	if *jsonOut || *formatFlag == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp StatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse stats: %w", err)
	}

	switch category {
	case "throughput":
		printThroughputMarkdown(resp.Throughput)
	case "blocking":
		printBlockingMarkdown(resp.Blocking)
	case "scope":
		printScopeMarkdown(resp.Scope)
	case "agent":
		printAgentMarkdown(resp.Agent)
	default:
		printStatsSummaryMarkdown(resp)
	}
	return nil
}

func printStatsSummaryMarkdown(resp StatsResponse) {
	fmt.Println("## Swarm Manager Stats")
	fmt.Printf("  Generated: %s | Events: %d\n\n", resp.GeneratedAt, resp.EventCount)

	printDashboardMarkdown(resp.Dashboard)
	fmt.Println()
	printThroughputMarkdown(resp.Throughput)
	fmt.Println()
	printTimingMarkdown(resp.Timing)
	fmt.Println()
	printBlockingMarkdown(resp.Blocking)
	fmt.Println()
	printAgentMarkdown(resp.Agent)
	fmt.Println()
	printScopeMarkdown(resp.Scope)
}

func printDashboardMarkdown(d DashboardStats) {
	fmt.Println("### Dashboard")
	fmt.Printf("  Backlog size: %d | Completed all time: %d\n", d.TotalBacklogSize, d.TotalCompletedAllTime)
	if d.EstimatedWeeksRemaining > 0 {
		fmt.Printf("  Estimated weeks remaining: %.1f\n", d.EstimatedWeeksRemaining)
	}
	if len(d.VelocityTrend) > 0 {
		parts := make([]string, 0, len(d.VelocityTrend))
		for _, v := range d.VelocityTrend {
			parts = append(parts, fmt.Sprintf("%s:%d", v.WeekStart, v.Completed))
		}
		fmt.Printf("  Velocity trend: %s\n", strings.Join(parts, " "))
	}
}

func printThroughputMarkdown(t ThroughputStats) {
	fmt.Println("### Throughput")
	fmt.Printf("  Created (7d/30d): %d / %d\n", t.CreatedLast7Days, t.CreatedLast30Days)
	fmt.Printf("  Completed (7d/30d): %d / %d\n", t.CompletedLast7Days, t.CompletedLast30Days)
	fmt.Printf("  Net delta (7d/30d): %+d / %+d\n", t.NetDelta7Days, t.NetDelta30Days)
}

func printTimingMarkdown(t TimingStats) {
	fmt.Println("### Timing")
	fmt.Printf("  Avg cycle time: %.1fh | Median: %.1fh\n", t.AvgCycleTimeHours, t.MedianCycleTimeHours)
	fmt.Printf("  Avg lead time: %.1fh | Median: %.1fh\n", t.AvgLeadTimeHours, t.MedianLeadTimeHours)
	fmt.Printf("  Avg queue wait: %.1fh\n", t.AvgQueueWaitHours)
}

func printBlockingMarkdown(b BlockingStats) {
	fmt.Println("### Blocking")
	fmt.Printf("  Currently blocked: %d (%.0f%%)\n", b.CurrentlyBlocked, b.BlockedRatio*100)
	if b.AvgBlockHours > 0 {
		fmt.Printf("  Avg block duration: %.1fh\n", b.AvgBlockHours)
	}
	if len(b.TopReasons) > 0 {
		fmt.Println("  Top reasons:")
		for _, r := range b.TopReasons {
			fmt.Printf("    - %s (%d)\n", r.Reason, r.Count)
		}
	}
}

func printAgentMarkdown(a AgentStats) {
	fmt.Println("### Agent Efficiency")
	fmt.Printf("  Total executions: %d\n", a.TotalExecutions)
	fmt.Printf("  Success rate: %.0f%% | Failure rate: %.0f%%\n", a.SuccessRate*100, a.FailureRate*100)
	fmt.Printf("  Follow-up rate: %.0f%%\n", a.FollowUpRate*100)
	if a.AvgExecutionMinutes > 0 {
		fmt.Printf("  Avg execution time: %.1f min\n", a.AvgExecutionMinutes)
	}
	if a.AvgWorkshopRounds > 0 {
		fmt.Printf("  Avg workshop rounds: %.1f\n", a.AvgWorkshopRounds)
	}
	if a.RecommendationAcceptanceSampleSize > 0 {
		fmt.Printf("  Recommendation acceptance: %.1f%% (n=%d)\n",
			a.RecommendationAcceptanceRate*100, a.RecommendationAcceptanceSampleSize)
		fmt.Printf("  Freeform override: %.1f%% (n=%d)\n",
			a.FreeformOverrideRate*100, a.RecommendationAcceptanceSampleSize)
		if len(a.RecommendationAcceptanceByKind) > 0 {
			kinds := make([]string, 0, len(a.RecommendationAcceptanceByKind))
			for k := range a.RecommendationAcceptanceByKind {
				kinds = append(kinds, k)
			}
			sort.Strings(kinds)
			fmt.Println("    By kind:")
			for _, k := range kinds {
				kr := a.RecommendationAcceptanceByKind[k]
				if kr.SampleSize == 0 {
					continue
				}
				fmt.Printf("      %s: %.1f%% (n=%d)\n", k, kr.Rate*100, kr.SampleSize)
			}
		}
	}
}

func printScopeMarkdown(s ScopeStats) {
	fmt.Println("### Initiative Health")
	if len(s.Initiatives) == 0 {
		fmt.Println("  No initiatives tracked")
		return
	}
	for _, ih := range s.Initiatives {
		pct := 0.0
		if ih.Total > 0 {
			pct = float64(ih.Completed) / float64(ih.Total) * 100
		}
		fmt.Printf("  %s: %d/%d (%.0f%%) | In progress: %d | Blocked: %d",
			ih.Name, ih.Completed, ih.Total, pct, ih.InProgress, ih.Blocked)
		if ih.ScopeCreep > 0 {
			fmt.Printf(" | Scope creep: +%.0f%%", ih.ScopeCreep*100)
		}
		fmt.Println()
	}
}
