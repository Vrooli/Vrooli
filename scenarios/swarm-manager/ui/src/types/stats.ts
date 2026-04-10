/**
 * Stats Types - TypeScript interfaces mirroring the Go stats response structs.
 *
 * Source of truth: api/internal/stats/types.go
 * Endpoint: GET /api/v1/stats
 */

/** Top-level response from GET /api/v1/stats. */
export interface StatsResponse {
  generated_at: string;
  event_count: number;
  throughput: ThroughputStats;
  timing: TimingStats;
  scope: ScopeStats;
  blocking: BlockingStats;
  agent: AgentStats;
  dashboard: DashboardStats;
}

/** Item creation and completion rates over rolling windows. */
export interface ThroughputStats {
  completed_last_7_days: number;
  completed_last_30_days: number;
  created_last_7_days: number;
  created_last_30_days: number;
  net_delta_7_days: number;
  net_delta_30_days: number;
}

/** How long work takes across lifecycle stages. */
export interface TimingStats {
  avg_cycle_time_hours: number;
  avg_lead_time_hours: number;
  avg_queue_wait_hours: number;
  median_cycle_time_hours: number;
  median_lead_time_hours: number;
}

/** Initiative health and scope changes. */
export interface ScopeStats {
  initiatives: InitiativeHealth[];
  max_dependency_depth: number;
}

/** Progress summary for a single initiative. */
export interface InitiativeHealth {
  name: string;
  total: number;
  completed: number;
  in_progress: number;
  blocked: number;
  scope_creep: number;
}

/** Blocking patterns and reasons. */
export interface BlockingStats {
  currently_blocked: number;
  blocked_ratio: number;
  top_reasons: ReasonCount[];
  avg_block_hours: number;
}

/** A blocking reason paired with its occurrence count. */
export interface ReasonCount {
  reason: string;
  count: number;
}

/** Agent execution efficiency metrics. */
export interface AgentStats {
  total_executions: number;
  success_rate: number;
  failure_rate: number;
  follow_up_rate: number;
  avg_execution_minutes: number;
  avg_workshop_rounds: number;
}

/** Top-level summary numbers for the dashboard view. */
export interface DashboardStats {
  total_backlog_size: number;
  total_completed_all_time: number;
  velocity_trend: VelocityPoint[];
  estimated_weeks_remaining: number;
}

/** Completions in a calendar week. */
export interface VelocityPoint {
  week_start: string;
  completed: number;
}

/** The 6 stat categories, matching the API's ?category= filter values. */
export type StatsCategory = "dashboard" | "throughput" | "agent" | "timing" | "blocking" | "scope";
