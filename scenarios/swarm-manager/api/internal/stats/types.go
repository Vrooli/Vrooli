// Package stats provides an incremental metrics engine that derives analytics
// from the eventlog. It uses a watermark pattern: on startup it replays all
// events to build aggregate state, then on each request only processes events
// appended since the last refresh.
package stats

import "time"

// StatsResponse is the top-level response returned by GET /api/v1/stats.
type StatsResponse struct {
	GeneratedAt time.Time       `json:"generated_at"`
	EventCount  int64           `json:"event_count"`
	History     HistoryWindow   `json:"history"`
	Throughput  ThroughputStats `json:"throughput"`
	Timing      TimingStats     `json:"timing"`
	Scope       ScopeStats      `json:"scope"`
	Blocking    BlockingStats   `json:"blocking"`
	Agent       AgentStats      `json:"agent"`
	Dashboard   DashboardStats  `json:"dashboard"`
	Review      ReviewStats     `json:"review"`
	Mode        ModeStats       `json:"mode"`
}

// HistoryWindow reports the span of event history the engine has observed. The
// UI uses this to decide when to show a "history shorter than window" banner
// instead of assuming 30d aggregates are comparable to long-term numbers.
type HistoryWindow struct {
	EarliestEventAt     time.Time `json:"earliest_event_at"`
	HistoryDays         float64   `json:"history_days"`
	HasHistory          bool      `json:"has_history"`
	MinSampleMeaningful int       `json:"min_sample_meaningful"`
}

// ThroughputStats tracks item creation and completion rates.
type ThroughputStats struct {
	CompletedLast7Days  int `json:"completed_last_7_days"`
	CompletedLast30Days int `json:"completed_last_30_days"`
	CreatedLast7Days    int `json:"created_last_7_days"`
	CreatedLast30Days   int `json:"created_last_30_days"`
	NetDelta7Days       int `json:"net_delta_7_days"`
	NetDelta30Days      int `json:"net_delta_30_days"`
}

// TimingStats tracks how long work takes across lifecycle stages.
//
// Cycle time and queue wait were removed when it became clear that backlog
// items in this scenario never transition through `in_progress` or `queued`
// states, so both metrics were always rendering as empty-sample "<1 min"
// noise. They are now replaced by execution duration (from the
// execution.completed event payload), which has real samples.
type TimingStats struct {
	AvgLeadTimeHours         float64 `json:"avg_lead_time_hours"`
	MedianLeadTimeHours      float64 `json:"median_lead_time_hours"`
	LeadTimeSampleSize       int     `json:"lead_time_sample_size"`
	AvgExecutionMinutes      float64 `json:"avg_execution_minutes"`
	MedianExecutionMinutes   float64 `json:"median_execution_minutes"`
	ExecutionDurationSamples int     `json:"execution_duration_samples"`
}

// ScopeStats tracks initiative health and scope changes.
type ScopeStats struct {
	Initiatives        []InitiativeHealth `json:"initiatives"`
	MaxDependencyDepth int                `json:"max_dependency_depth"`
}

// InitiativeHealth summarizes a single initiative's progress.
type InitiativeHealth struct {
	Name       string  `json:"name"`
	Total      int     `json:"total"`
	Completed  int     `json:"completed"`
	InProgress int     `json:"in_progress"`
	Blocked    int     `json:"blocked"`
	ScopeCreep float64 `json:"scope_creep"`
}

// BlockingStats tracks blocking patterns.
type BlockingStats struct {
	CurrentlyBlocked int           `json:"currently_blocked"`
	BlockedRatio     float64       `json:"blocked_ratio"`
	TopReasons       []ReasonCount `json:"top_reasons"`
	AvgBlockHours    float64       `json:"avg_block_hours"`
}

// ReasonCount pairs a blocking reason with its occurrence count.
type ReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// AgentStats tracks agent execution efficiency.
//
// CompletedCount includes manually-accepted runs, so SuccessRate reflects
// end-to-end success (agent-finished + human-finished). ManualAcceptRate
// exposes the subset so reviewers can see how often the human disagrees
// with the agent's verdict.
type AgentStats struct {
	TotalExecutions          int     `json:"total_executions"`
	CompletedCount           int     `json:"completed_count"`
	FailedCount              int     `json:"failed_count"`
	ManuallyAcceptedCount    int     `json:"manually_accepted_count"`
	SuccessRate              float64 `json:"success_rate"`
	FailureRate              float64 `json:"failure_rate"`
	ManualAcceptRate         float64 `json:"manual_accept_rate"`
	FollowUpRate             float64 `json:"follow_up_rate"`
	AvgExecutionMinutes      float64 `json:"avg_execution_minutes"`
	AvgWorkshopRounds        float64 `json:"avg_workshop_rounds"`
	SuccessRateSampleSize    int     `json:"success_rate_sample_size"`
	ExecutionDurationSamples int     `json:"execution_duration_samples"`
	WorkshopRoundsSampleSize int     `json:"workshop_rounds_sample_size"`

	// Recommendation acceptance: how often the user picked the agent's
	// recommended workshop option, with a "freeform overrides" companion.
	// Freeform answers ("Other") count toward the answered denominator and
	// freeform-override numerator; they never count toward the recommended
	// numerator — picking "Other" rejects the offered option set.
	RecommendationAcceptanceRate       float64             `json:"recommendation_acceptance_rate"`
	RecommendationAcceptanceSampleSize int                 `json:"recommendation_acceptance_sample_size"`
	FreeformOverrideRate               float64             `json:"freeform_override_rate"`
	DecisionItemsTotal                 int                 `json:"decision_items_total"`
	DecisionItemsAnswered              int                 `json:"decision_items_answered"`
	RecommendationAcceptanceByKind     map[string]KindRate `json:"recommendation_acceptance_by_kind"`
}

// KindRate is the per-kind breakdown for recommendation-acceptance stats.
// SampleSize is the number of answered decision items contributing to Rate.
type KindRate struct {
	Rate       float64 `json:"rate"`
	SampleSize int     `json:"sample_size"`
}

// DashboardStats provides top-level summary numbers.
type DashboardStats struct {
	TotalBacklogSize        int             `json:"total_backlog_size"`
	TotalCompletedAllTime   int             `json:"total_completed_all_time"`
	VelocityTrend           []VelocityPoint `json:"velocity_trend"`
	EstimatedWeeksRemaining float64         `json:"estimated_weeks_remaining"`
	VelocityWeeksCovered    int             `json:"velocity_weeks_covered"`
}

// VelocityPoint represents completions in a calendar week.
type VelocityPoint struct {
	WeekStart string `json:"week_start"`
	Completed int    `json:"completed"`
}

// ReviewStats tracks review agent evidence gathering metrics.
type ReviewStats struct {
	RoundsCompleted         int     `json:"rounds_completed"`
	AverageEvidencePerRound float64 `json:"avg_evidence_per_round"`
	VerificationRate        float64 `json:"verification_rate"`
	RequestMoreRate         float64 `json:"request_more_rate"`
	AverageReviewDuration   float64 `json:"avg_review_duration_seconds"`
}

// ModeStats tracks operating-mode adoption and phase-run outcomes.
type ModeStats struct {
	UsageByMode              map[string]int                `json:"usage_by_mode"`
	ModeSwitchCount          int                           `json:"mode_switch_count"`
	PhaseRunsByMode          map[string]map[string]int     `json:"phase_runs_by_mode"`
	CompletedByMode          map[string]int                `json:"completed_by_mode"`
	FailedByMode             map[string]int                `json:"failed_by_mode"`
	CanceledByMode           map[string]int                `json:"canceled_by_mode"`
	ReplanRateByMode         map[string]KindRate           `json:"replan_rate_by_mode"`
	AcceptanceRateByMode     map[string]KindRate           `json:"acceptance_rate_by_mode"`
	AvgPhaseDurationSeconds  map[string]map[string]float64 `json:"avg_phase_duration_seconds"`
	AvgRunsPerCompletedScope map[string]float64            `json:"avg_runs_per_completed_scope"`
	BacklogSyncByMode        map[string]BacklogSyncStats   `json:"backlog_sync_by_mode"`
	UsageByProfile           map[string]int                `json:"usage_by_profile"`
	PhaseRunsByProfile       map[string]map[string]int     `json:"phase_runs_by_profile"`
}

type BacklogSyncStats struct {
	Events         int `json:"events"`
	ItemsCompleted int `json:"items_completed"`
	ItemsCreated   int `json:"items_created"`
	ItemsUpdated   int `json:"items_updated"`
}
