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
	Throughput  ThroughputStats `json:"throughput"`
	Timing      TimingStats     `json:"timing"`
	Scope       ScopeStats      `json:"scope"`
	Blocking    BlockingStats   `json:"blocking"`
	Agent       AgentStats      `json:"agent"`
	Dashboard   DashboardStats  `json:"dashboard"`
	Review      ReviewStats     `json:"review"`
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
type TimingStats struct {
	AvgCycleTimeHours    float64 `json:"avg_cycle_time_hours"`
	AvgLeadTimeHours     float64 `json:"avg_lead_time_hours"`
	AvgQueueWaitHours    float64 `json:"avg_queue_wait_hours"`
	MedianCycleTimeHours float64 `json:"median_cycle_time_hours"`
	MedianLeadTimeHours  float64 `json:"median_lead_time_hours"`
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
type AgentStats struct {
	TotalExecutions     int     `json:"total_executions"`
	SuccessRate         float64 `json:"success_rate"`
	FailureRate         float64 `json:"failure_rate"`
	FollowUpRate        float64 `json:"follow_up_rate"`
	AvgExecutionMinutes float64 `json:"avg_execution_minutes"`
	AvgWorkshopRounds   float64 `json:"avg_workshop_rounds"`
}

// DashboardStats provides top-level summary numbers.
type DashboardStats struct {
	TotalBacklogSize        int             `json:"total_backlog_size"`
	TotalCompletedAllTime   int             `json:"total_completed_all_time"`
	VelocityTrend           []VelocityPoint `json:"velocity_trend"`
	EstimatedWeeksRemaining float64         `json:"estimated_weeks_remaining"`
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
