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
  history: HistoryWindow;
  throughput: ThroughputStats;
  timing: TimingStats;
  scope: ScopeStats;
  blocking: BlockingStats;
  agent: AgentStats;
  dashboard: DashboardStats;
  mode: ModeStats;
  session: SessionStats;
}

/** Span of event-log history observed by the engine. */
export interface HistoryWindow {
  earliest_event_at: string;
  history_days: number;
  has_history: boolean;
  min_sample_meaningful: number;
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
  avg_lead_time_hours: number;
  median_lead_time_hours: number;
  lead_time_sample_size: number;
  avg_execution_minutes: number;
  median_execution_minutes: number;
  execution_duration_samples: number;
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
  completed_count: number;
  failed_count: number;
  manually_accepted_count: number;
  success_rate: number;
  failure_rate: number;
  manual_accept_rate: number;
  follow_up_rate: number;
  avg_execution_minutes: number;
  avg_workshop_rounds: number;
  success_rate_sample_size: number;
  execution_duration_samples: number;
  workshop_rounds_sample_size: number;
  recommendation_acceptance_rate: number;
  recommendation_acceptance_sample_size: number;
  freeform_override_rate: number;
  decision_items_total: number;
  decision_items_answered: number;
  recommendation_acceptance_by_kind: Record<string, KindRate>;
}

/** Per-kind breakdown of recommendation acceptance. */
export interface KindRate {
  rate: number;
  sample_size: number;
}

/** Top-level summary numbers for the dashboard view. */
export interface DashboardStats {
  total_backlog_size: number;
  total_completed_all_time: number;
  velocity_trend: VelocityPoint[];
  estimated_weeks_remaining: number;
  velocity_weeks_covered: number;
}

/** Operating-mode adoption and phase-run metrics. */
export interface ModeStats {
  usage_by_mode: Record<string, number>;
  mode_switch_count: number;
  phase_runs_by_mode: Record<string, Record<string, number>>;
  completed_by_mode: Record<string, number>;
  failed_by_mode: Record<string, number>;
  canceled_by_mode: Record<string, number>;
  replan_rate_by_mode: Record<string, KindRate>;
  acceptance_rate_by_mode: Record<string, KindRate>;
  avg_phase_duration_seconds: Record<string, Record<string, number>>;
  avg_runs_per_completed_scope: Record<string, number>;
  backlog_sync_by_mode: Record<string, BacklogSyncStats>;
  usage_by_profile: Record<string, number>;
  phase_runs_by_profile: Record<string, Record<string, number>>;
}

/** Backlog reconciliation summary emitted by operating-mode phases. */
export interface BacklogSyncStats {
  events: number;
  items_completed: number;
  items_created: number;
  items_updated: number;
}

/** Native Agent Session adoption and outcome metrics. */
export interface SessionStats {
  total_sessions: number;
  active_sessions: number;
  sessions_by_kind: Record<string, number>;
  sessions_by_status: Record<string, number>;
  proposal_created_by_kind: Record<string, number>;
  proposal_applied_by_kind: Record<string, number>;
  proposal_apply_rate_by_kind: Record<string, KindRate>;
  artifacts_created_by_kind: Record<string, number>;
  artifacts_by_type: Record<string, number>;
  avg_messages_per_session: number;
  avg_time_to_first_proposal_seconds: number;
  first_proposal_sample_size: number;
  failed_session_rate: number;
  failed_session_sample_size: number;
  session_created_backlog_items: number;
  session_created_initiatives: number;
}

/** Completions in a calendar week. */
export interface VelocityPoint {
  week_start: string;
  completed: number;
}

/** The stat categories, matching the API's ?category= filter values. */
export type StatsCategory = "dashboard" | "throughput" | "agent" | "timing" | "blocking" | "scope" | "modes" | "sessions";
