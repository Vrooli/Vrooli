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
	History     HistoryWindow   `json:"history"`
	Throughput  ThroughputStats `json:"throughput"`
	Timing      TimingStats     `json:"timing"`
	Scope       ScopeStats      `json:"scope"`
	Blocking    BlockingStats   `json:"blocking"`
	Agent       AgentStats      `json:"agent"`
	Dashboard   DashboardStats  `json:"dashboard"`
	Session     SessionStats    `json:"session"`
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
	AvgLeadTimeHours         float64 `json:"avg_lead_time_hours"`
	MedianLeadTimeHours      float64 `json:"median_lead_time_hours"`
	LeadTimeSampleSize       int     `json:"lead_time_sample_size"`
	AvgExecutionMinutes      float64 `json:"avg_execution_minutes"`
	MedianExecutionMinutes   float64 `json:"median_execution_minutes"`
	ExecutionDurationSamples int     `json:"execution_duration_samples"`
}

type ScopeStats struct {
	Goals              []GoalHealth `json:"goals"`
	MaxDependencyDepth int          `json:"max_dependency_depth"`
}

type GoalHealth struct {
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
	CompletedCount                     int                 `json:"completed_count"`
	FailedCount                        int                 `json:"failed_count"`
	ManuallyAcceptedCount              int                 `json:"manually_accepted_count"`
	SuccessRate                        float64             `json:"success_rate"`
	FailureRate                        float64             `json:"failure_rate"`
	ManualAcceptRate                   float64             `json:"manual_accept_rate"`
	FollowUpRate                       float64             `json:"follow_up_rate"`
	AvgExecutionMinutes                float64             `json:"avg_execution_minutes"`
	AvgWorkshopRounds                  float64             `json:"avg_workshop_rounds"`
	SuccessRateSampleSize              int                 `json:"success_rate_sample_size"`
	ExecutionDurationSamples           int                 `json:"execution_duration_samples"`
	WorkshopRoundsSampleSize           int                 `json:"workshop_rounds_sample_size"`
	RecommendationAcceptanceRate       float64             `json:"recommendation_acceptance_rate"`
	RecommendationAcceptanceSampleSize int                 `json:"recommendation_acceptance_sample_size"`
	FreeformOverrideRate               float64             `json:"freeform_override_rate"`
	DecisionItemsTotal                 int                 `json:"decision_items_total"`
	DecisionItemsAnswered              int                 `json:"decision_items_answered"`
	RecommendationAcceptanceByKind     map[string]KindRate `json:"recommendation_acceptance_by_kind"`
}

type KindRate struct {
	Rate       float64 `json:"rate"`
	SampleSize int     `json:"sample_size"`
}

type DashboardStats struct {
	TotalBacklogSize      int             `json:"total_backlog_size"`
	TotalCompletedAllTime int             `json:"total_completed_all_time"`
	VelocityTrend         []VelocityPoint `json:"velocity_trend"`
	EstimatedRemaining    *ETABand        `json:"estimated_remaining,omitempty"`
	VelocityWeeksCovered  int             `json:"velocity_weeks_covered"`
}

type ETABand struct {
	P50Hours       float64 `json:"p50_hours"`
	P80Hours       float64 `json:"p80_hours"`
	P50Label       string  `json:"p50_label"`
	P80Label       string  `json:"p80_label"`
	Basis          string  `json:"basis"`
	BasisLabel     string  `json:"basis_label"`
	Confidence     string  `json:"confidence"`
	RemainingItems int     `json:"remaining_items"`
	LaneCapacity   int     `json:"lane_capacity"`
}

type SessionStats struct {
	TotalSessions                     int                 `json:"total_sessions"`
	ActiveSessions                    int                 `json:"active_sessions"`
	SessionsByKind                    map[string]int      `json:"sessions_by_kind"`
	SessionsByStatus                  map[string]int      `json:"sessions_by_status"`
	ProposalCreatedByKind             map[string]int      `json:"proposal_created_by_kind"`
	ProposalAppliedByKind             map[string]int      `json:"proposal_applied_by_kind"`
	ProposalApplyRateByKind           map[string]KindRate `json:"proposal_apply_rate_by_kind"`
	ArtifactsCreatedByKind            map[string]int      `json:"artifacts_created_by_kind"`
	ArtifactsByType                   map[string]int      `json:"artifacts_by_type"`
	AverageMessagesPerSession         float64             `json:"avg_messages_per_session"`
	AverageTimeToFirstProposalSeconds float64             `json:"avg_time_to_first_proposal_seconds"`
	FirstProposalSampleSize           int                 `json:"first_proposal_sample_size"`
	FailedSessionRate                 float64             `json:"failed_session_rate"`
	FailedSessionSampleSize           int                 `json:"failed_session_sample_size"`
	SessionCreatedBacklogItems        int                 `json:"session_created_backlog_items"`
	SessionCreatedGoals               int                 `json:"session_created_goals"`
}

type VelocityPoint struct {
	WeekStart      string             `json:"week_start"`
	Completed      int                `json:"completed"`
	CompletedItems []CompletedItemRef `json:"completed_items"`
}

type CompletedItemRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type HistoryWindow struct {
	EarliestEventAt     string  `json:"earliest_event_at"`
	HistoryDays         float64 `json:"history_days"`
	HasHistory          bool    `json:"has_history"`
	MinSampleMeaningful int     `json:"min_sample_meaningful"`
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

func (a *App) cmdStatsMilestones(args []string) error {
	return a.statsCategoryCommand("scope", args)
}

func (a *App) cmdStatsAgent(args []string) error {
	return a.statsCategoryCommand("agent", args)
}

func (a *App) cmdStatsSessions(args []string) error {
	return a.statsCategoryCommand("sessions", args)
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
	case "sessions":
		printSessionsMarkdown(resp.Session)
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
	printSessionsMarkdown(resp.Session)
	fmt.Println()
	printScopeMarkdown(resp.Scope)
}

func printDashboardMarkdown(d DashboardStats) {
	fmt.Println("### Dashboard")
	fmt.Printf("  Backlog size: %d | Completed all time: %d\n", d.TotalBacklogSize, d.TotalCompletedAllTime)
	if d.EstimatedRemaining != nil {
		fmt.Printf(
			"  Estimated remaining: %s-%s (%d items, %s)\n",
			d.EstimatedRemaining.P50Label,
			d.EstimatedRemaining.P80Label,
			d.EstimatedRemaining.RemainingItems,
			d.EstimatedRemaining.BasisLabel,
		)
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
	fmt.Printf("  Avg lead time: %.1fh | Median: %.1fh\n", t.AvgLeadTimeHours, t.MedianLeadTimeHours)
	if t.ExecutionDurationSamples > 0 {
		fmt.Printf("  Avg execution time: %.1f min | Median: %.1f min (n=%d)\n",
			t.AvgExecutionMinutes, t.MedianExecutionMinutes, t.ExecutionDurationSamples)
	}
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
	fmt.Println("### Goal Health")
	if len(s.Goals) == 0 {
		fmt.Println("  No goals tracked")
		return
	}
	for _, ih := range s.Goals {
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

func printSessionsMarkdown(s SessionStats) {
	fmt.Println("### Agent Sessions")
	fmt.Printf("  Sessions: %d | Active: %d\n", s.TotalSessions, s.ActiveSessions)
	if s.TotalSessions > 0 {
		fmt.Printf("  Avg messages/session: %.1f\n", s.AverageMessagesPerSession)
	}
	if s.FirstProposalSampleSize > 0 {
		fmt.Printf("  Avg time to first proposal: %s (n=%d)\n",
			formatDurationSeconds(s.AverageTimeToFirstProposalSeconds), s.FirstProposalSampleSize)
	}
	if s.FailedSessionSampleSize > 0 {
		fmt.Printf("  Failed session rate: %.1f%% (n=%d)\n",
			s.FailedSessionRate*100, s.FailedSessionSampleSize)
	}
	fmt.Printf("  Created artifacts: backlog=%d goals=%d\n",
		s.SessionCreatedBacklogItems, s.SessionCreatedGoals)
	printIntMap("  By kind", s.SessionsByKind)
	printIntMap("  By status", s.SessionsByStatus)
	if len(s.ProposalApplyRateByKind) > 0 {
		kinds := make([]string, 0, len(s.ProposalApplyRateByKind))
		for kind := range s.ProposalApplyRateByKind {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		fmt.Println("  Proposal apply rate:")
		for _, kind := range kinds {
			rate := s.ProposalApplyRateByKind[kind]
			fmt.Printf("    - %s: %.1f%% (n=%d)\n", kind, rate.Rate*100, rate.SampleSize)
		}
	}
	printIntMap("  Artifacts by type", s.ArtifactsByType)
}

func printIntMap(label string, values map[string]int) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Println(label + ":")
	for _, key := range keys {
		fmt.Printf("    - %s: %d\n", key, values[key])
	}
}

func formatDurationSeconds(seconds float64) string {
	if seconds <= 0 {
		return "0m"
	}
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.0fm", seconds/60)
	}
	return fmt.Sprintf("%.1fh", seconds/3600)
}
