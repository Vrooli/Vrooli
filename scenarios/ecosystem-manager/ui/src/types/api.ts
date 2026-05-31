/**
 * API Type Definitions for Ecosystem Manager
 */

// ==================== Task Types ====================

export type TaskStatus =
  | 'pending'
  | 'in-progress'
  | 'completed'
  | 'completed-finalized'
  | 'failed'
  | 'failed-blocked'
  | 'archived';

export type TaskSort =
  | 'updated_desc'
  | 'updated_asc'
  | 'created_desc'
  | 'created_asc';

export type TaskType = 'resource' | 'scenario';

export type OperationType = 'generator' | 'improver';

export type Priority = 'critical' | 'high' | 'medium' | 'low';

// Steering strategy types for unified steering configuration
export type SteeringStrategy = 'profile' | 'queue' | 'manual' | 'none';

export interface SteeringConfig {
  strategy: SteeringStrategy;
  profileId?: string; // Profile strategy - ID of the Auto Steer profile
  queue?: string[][]; // Queue strategy - ordered list of skill sets
  manualSet?: string[]; // Manual strategy - skill set that repeats
}

export interface Task {
  id: string;
  title: string;
  type: TaskType;
  operation: OperationType;
  priority: Priority;
  status: TaskStatus;
  target?: string[];
  notes?: string;
  steer_set?: string[];
  auto_steer_profile_id?: string;
  auto_steer_phase_index?: number;
  steering_queue?: string[][]; // Ordered list of skill sets for queue steering strategy
  steering_queue_index?: number; // Current position in the queue (0-indexed)
  steering_queue_set_label?: string; // Current set summary being executed from the queue
  steering_queue_total?: number; // Total items in the queue
  steering_queue_exhausted?: boolean; // Whether the queue has been fully processed
  auto_requeue?: boolean;
  created_at: string;
  updated_at: string;
  execution_count?: number;
  completion_count?: number;
  last_completed_at?: string;
  current_process?: ProcessInfo;
  cooldown_until?: string;
}

export interface ProcessInfo {
  process_id: string;
  agent_id: string;
  start_time: string;
  elapsed_seconds?: number;
}

export interface TaskFilters {
  search?: string;
  status?: TaskStatus | '';
  type?: TaskType | '';
  operation?: OperationType | '';
  priority?: Priority | '';
  sort?: TaskSort;
}

export interface CreateTaskInput {
  type: TaskType;
  operation: OperationType;
  priority: Priority;
  steer_set?: string[];
  target?: string[];
  notes?: string;
  auto_steer_profile_id?: string;
  steering_queue?: string[][]; // Ordered list of skill sets for queue steering strategy
  auto_requeue?: boolean;
}

export interface UpdateTaskInput {
  priority?: Priority;
  notes?: string;
  steer_set?: string[];
  auto_steer_profile_id?: string;
  steering_queue?: string[][]; // Ordered list of skill sets for queue steering strategy
  auto_requeue?: boolean;
  target?: string[];
}

// ==================== Queue Types ====================

export interface QueueStatus {
  active: boolean;
  slots_used: number;
  max_concurrent: number;
  available_slots?: number;
  tasks_remaining: number;
  cooldown_seconds?: number;
  rate_limited?: boolean;
  rate_limit_retry_after?: number;
  rate_limit_pause_until?: string;
  executions_completed?: number;
  execution_limit?: number;
  execution_limit_reached?: boolean;
}

export interface RunningProcess {
  task_id: string;
  task_title: string;
  process_id: string;
  process_type: 'task' | 'insight';
  agent_id: string;
  start_time: string;
  elapsed_seconds: number;
}

// ==================== Settings Constraints ====================

export interface ConstraintRange {
  min: number;
  max: number;
}

export interface SettingsConstraints {
  slots: ConstraintRange;
  cooldown_seconds: ConstraintRange;
  execution_limit: ConstraintRange;
  max_turns: ConstraintRange;
  task_timeout: ConstraintRange;
  idle_timeout_cap: ConstraintRange;
  recycler: {
    interval_seconds: ConstraintRange;
    max_retries: ConstraintRange;
    retry_delay_seconds: ConstraintRange;
    completion_threshold: ConstraintRange;
    failure_threshold: ConstraintRange;
  };
}

// ==================== Settings Types ====================

export interface Settings {
  processor: ProcessorSettings;
  agent: AgentSettings;
  display: DisplaySettings;
  recycler: RecyclerSettings;
}

export interface ProcessorSettings {
  concurrent_slots: number;
  cooldown_seconds: number;
  execution_limit: number;
  active: boolean;
}

export interface AgentSettings {
  max_turns: number;
  allowed_tools?: string;
  skip_permissions: boolean;
  task_timeout_minutes: number;
  idle_timeout_cap_minutes: number;
  runner_type: 'claude-code' | 'codex' | 'opencode';
}

export interface DisplaySettings {
  theme: 'light' | 'dark' | 'auto';
  condensed_mode: boolean;
}

export interface RecyclerSettings {
  enabled_for: 'off' | 'resources' | 'scenarios' | 'both';
  recycle_interval: number;
  max_retries: number;
  retry_delay_seconds: number;
  model_provider: 'ollama' | 'openrouter';
  model_name: string;
  completion_threshold: number;
  failure_threshold: number;
}

// ==================== Auto Steer Types ====================

/**
 * An Auto Steer profile is the controller's *objective function*, not a script.
 * It declares which improvement dimensions matter (weights), what "done" looks
 * like (targets), which skills the controller may select (allowed_skills), and
 * the loop budget. The controller derives the path; the profile no longer
 * sequences phases. See docs/concepts/CONTROL-MODEL.md.
 */
export interface AutoSteerProfile {
  id: string;
  name: string;
  description?: string;
  tags?: string[];
  objective: AutoSteerObjective;
  allowed_skills: string[];
  budget: AutoSteerBudget;
  audit_preset?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AutoSteerObjective {
  /** Per-dimension importance. Higher weight = the controller prioritizes closing that dimension. */
  dimension_weights: Record<string, number>;
  targets: AutoSteerTargets;
}

export interface AutoSteerTargets {
  /** Highest severity allowed to remain open at termination (e.g. "low"). Empty = no severity gate. */
  max_open_severity?: string;
  /** Required operational-target completion percentage (0-100). */
  operational_targets_pct?: number;
}

export interface AutoSteerBudget {
  /** Layer-3 backstop: hard iteration cap. */
  max_iterations: number;
  /** Stop when marginal weighted-score improvement falls below this floor. */
  diminishing_returns_floor: number;
  /** Run a full comprehensive re-audit every N iterations (targeted re-audit otherwise). */
  reaudit_cadence?: number;
}

/**
 * One iteration of the closed-loop controller's reasoning (DIAGNOSE → SELECT →
 * MEASURE). Surfaced by GET /api/auto-steer/execution/:taskId/trace so the
 * controller is a glass box. See docs/concepts/CONTROL-MODEL.md.
 */
export interface DecisionTraceEntry {
  iteration: number;
  timestamp?: string;
  dimension_scores?: Record<string, number>;
  heaviest_dimension?: string;
  chosen_skill?: string;
  rationale?: string;
  fingerprint?: string;
  score_before?: number;
  score_after?: number;
  realized_delta?: number;
  /** Agent run token cost for this iteration (0 = unknown). */
  tokens_used?: number;
  /** Per-dimension findings flow this iteration produced. */
  closed_by_dimension?: Record<string, number>;
  introduced_by_dimension?: Record<string, number>;
  /** Net weighted score went up this iteration. */
  regressed?: boolean;
  /** The profile's regression veto fired this iteration. */
  veto_applied?: boolean;
  /** Terminal halt reason, set on the final iteration. */
  halt_reason?: string;
  /** DTV transparency (P2). The chosen skill's fitness verdict. */
  dtv_verdict?: string;
  /** The cold-start trust/cost prior DTV seeded for the chosen skill. */
  dtv_prior?: number;
  /** Skills the Layer-1 gate denied for the chosen dimension → reason. */
  dtv_excluded?: Record<string, string>;
  /** The gate would have emptied the dimension; fell back to allow-all. */
  dtv_gate_override?: boolean;
  /** DTV was unreachable; this selection used fail-open (P1) behavior. */
  dtv_degraded?: boolean;
}

export interface DecisionTraceResponse {
  task_id: string;
  trace: DecisionTraceEntry[];
  count: number;
}

/** One row of the per-(skill, dimension) effectiveness ledger. */
export interface EffectivenessRow {
  skill_id: string;
  dimension: string;
  closed_count: number;
  introduced_count: number;
  net_closed: number;
  total_runs: number;
  total_tokens: number;
  avg_tokens_per_run: number;
  observed_efficacy_per_ktok: number;
  expected_efficacy_per_ktok: number;
  last_run_at?: string;
}

export interface EffectivenessResponse {
  effectiveness: EffectivenessRow[];
  count: number;
  prior: number;
  shrinkage_k: number;
}

/** A built-in objective profile shipped with the system; same shape as a saved profile. */
export type AutoSteerTemplate = AutoSteerProfile;

/** One canonical improvement dimension (served from the dimensions SSOT). */
export interface DimensionInfo {
  id: string;
  description: string;
}

/** A single open finding produced by a test-genie audit, mapped to a dimension. */
export interface Finding {
  id: string;
  dimension: string;
  /** architecture.v1.FindingSeverity enum value. */
  severity: number;
  location?: string;
  code?: string;
  message?: string;
  phase?: string;
}

/**
 * The controller's diagnosis at a point in time: open findings bucketed by
 * dimension with weighted scores and a set fingerprint. This is the controller's
 * primary state (the gap it is closing), not a phase cursor.
 */
export interface FindingsState {
  findings: Finding[];
  dimensionScore: Record<string, number>;
  dimensionCount: Record<string, number>;
  totalScore: number;
  fingerprint: string;
}

export interface AutoSteerExecutionState {
  task_id: string;
  profile_id: string;
  /** Completed controller iterations. */
  iteration: number;
  /** Skill the controller selected for the current/next run. */
  current_skill?: string;
  /** Why that skill was chosen (heaviest open dimension, etc.). */
  current_rationale?: string;
  findings?: FindingsState;
  /** Total weighted score after each iteration (the convergence curve). */
  score_history?: number[];
  trace?: DecisionTraceEntry[];
  metrics?: MetricsSnapshot;
  started_at?: string;
  last_updated?: string;
}

export interface MetricsSnapshot {
  timestamp: string;
  phase_loops: number;
  total_loops: number;
  build_status: number;
  operational_targets_total: number;
  operational_targets_passing: number;
  operational_targets_percentage: number;
  ux?: {
    accessibility_score: number;
    ui_test_coverage: number;
    responsive_breakpoints: number;
    user_flows_implemented: number;
    loading_states_count: number;
    error_handling_coverage: number;
  };
  refactor?: {
    cyclomatic_complexity_avg: number;
    duplication_percentage: number;
    standards_violations: number;
    tidiness_score: number;
    tech_debt_items: number;
  };
  test?: {
    unit_test_coverage: number;
    integration_test_coverage: number;
    ui_test_coverage: number;
    edge_cases_covered: number;
    flaky_tests: number;
    test_quality_score: number;
  };
  performance?: {
    bundle_size_kb: number;
    initial_load_time_ms: number;
    lcp_ms: number;
    fid_ms: number;
    cls_score: number;
  };
  security?: {
    vulnerability_count: number;
    input_validation_coverage: number;
    auth_implementation_score: number;
    security_scan_score: number;
  };
}

/**
 * One skill's contribution within a completed controller run, derived from the
 * decision trace. Replaces the old per-phase breakdown.
 */
export interface SkillPerformance {
  skill_name: string;
  iterations: number;
  /** Total realized weighted-score reduction attributed to this skill. */
  weighted_delta: number;
}

export interface UserFeedback {
  rating: number;
  comments?: string;
  submitted_at: string;
}

export interface ProfilePerformance {
  id: string;
  profile_id: string;
  scenario_name: string;
  execution_id: string;
  start_metrics: MetricsSnapshot;
  end_metrics: MetricsSnapshot;
  phase_breakdown: SkillPerformance[];
  total_iterations: number;
  total_duration: number;
  user_feedback?: UserFeedback;
  executed_at: string;
  feedback_entries?: ExecutionFeedbackEntry[];
}

export interface ExecutionFeedbackEntry {
  id: string;
  category: string;
  severity: 'low' | 'medium' | 'high' | 'critical' | string;
  suggested_action?: string;
  comments?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface ExecutionFeedbackEntryPayload {
  category: string;
  severity: 'low' | 'medium' | 'high' | 'critical' | string;
  suggested_action?: string;
  comments?: string;
  metadata?: Record<string, unknown>;
}

// ==================== Discovery Types ====================

export interface Resource {
  name: string;
  display_name?: string;
  description?: string;
  path?: string;
  port?: number;
  category?: string;
  version?: string;
  healthy?: boolean;
  status?: string;
}

export interface Scenario {
  name: string;
  display_name?: string;
  status?: string;
  description?: string;
  path?: string;
  category?: string;
  version?: string;
}

export interface Operation {
  name: string;
  display_name: string;
  description?: string;
}

export interface Category {
  name: string;
  display_name: string;
}

export interface ActiveTarget {
  target: string;
  task_id: string;
  status: TaskStatus;
  title?: string;
}

// ==================== Logs Types ====================

export interface LogEntry {
  timestamp: string;
  level: 'info' | 'warning' | 'error';
  message: string;
  context?: Record<string, unknown>;
}

// ==================== Execution Types ====================

export interface ExecutionHistory {
  id: string;
  task_id: string;
  task_title?: string;
  task_type?: TaskType;
  task_operation?: OperationType;
  agent_tag?: string;
  process_id?: number;
  start_time: string;
  end_time?: string;
  duration?: string;
  status: 'running' | 'completed' | 'failed' | 'rate_limited';
  exit_code?: number;
  exit_reason?: string;
  prompt_size?: number;
  prompt_path?: string;
  output_path?: string;
  clean_output_path?: string;
  last_message_path?: string;
  transcript_path?: string;
  auto_steer_profile_id?: string;
  auto_steer_iteration?: number;
  steer_skill_ids?: string[];
  steer_set_label?: string;
  steer_phase_index?: number;
  steer_phase_iteration?: number;
  steering_source?: string;
  timeout_allowed?: string;
  rate_limited?: boolean;
  retry_after?: number;
  success?: boolean;
  metadata?: Record<string, unknown>;
}

export interface ExecutionPrompt {
  prompt?: string;
  content?: string;
  size?: number;
  task_id?: string;
  execution_id?: string;
  metadata?: Record<string, unknown>;
}

export interface ExecutionOutput {
  output?: string;
  content?: string;
  size?: number;
  source?: string;
  task_id?: string;
  execution_id?: string;
  metadata?: Record<string, unknown>;
}

// ==================== WebSocket Types ====================

export type WebSocketMessageType =
  | 'connected'
  | 'task_status_changed'
  | 'task_status_updated'
  | 'task_updated'
  | 'task_recycled'
  | 'task_deleted'
  | 'task_started'
  | 'task_executing'
  | 'task_progress'
  | 'task_completed'
  | 'task_failed'
  | 'claude_execution_complete'
  | 'process_terminated'
  | 'settings_updated'
  | 'settings_reset'
  | 'rate_limit_pause'
  | 'rate_limit_pause_started'
  | 'rate_limit_resume'
  | 'rate_limit_manual_reset'
  | 'rate_limit_hit'
  | 'execution_limit_reached'
  | 'log_entry'
  | string;

export interface WebSocketMessage {
  type: WebSocketMessageType;
  data?: Record<string, unknown>;
  message?: string;
  timestamp?: number;
  [key: string]: unknown;
}

// ==================== Health Check ====================

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export interface PromptPreviewConfig {
  type: TaskType;
  operation: OperationType;
  title: string;
  priority: Priority;
  notes?: string;
  target?: string;
  targets?: string[];
  steer_set?: string[];
  steering_queue?: string[][];
  auto_steer_profile_id?: string;
  auto_steer_phase_index?: number;
}

export interface PromptPreviewResult {
  prompt: string;
  token_count?: number;
  sections?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}

export interface PromptFileInfo {
  id: string;
  path: string;
  display_name?: string;
  type?: string;
  size?: number;
  modified_at?: string;
}

export interface PromptFile {
  id: string;
  path: string;
  content: string;
  size: number;
  modified_at?: string;
}

export interface PhaseInfo {
  id: string;
  name: string;
  description?: string;
  modes?: string[];
  source?: 'prompt-manager' | 'builtin';
}

// ==================== Skills Types (from prompt-manager) ====================

export interface SkillResponse {
  id: string;
  name: string;
  description: string;
  content: string;
  modes: string[];
  tags: string[];
  icon?: string;
  targetToolId?: string;
  draft: boolean;
  folder: string;
  createdAt: string;
  updatedAt: string;
  usageCount: number;
  lastUsed?: string;
  effectivenessRating?: number;
}

export interface SkillsSyncResult {
  success: boolean;
  available: boolean;
  skillCount?: number;
  error?: string;
}

// ==================== API Response Wrappers ====================

export interface ApiResponse<T> {
  data: T;
  message?: string;
}

export interface ApiError {
  error: string;
  message: string;
  status: number;
}

// ==================== Visited Tracker Types ====================

export interface Campaign {
  id: string;
  name: string;
  from_agent: string;
  description?: string;
  patterns: string[];
  location?: string;
  tag?: string;
  notes?: string;
  max_files?: number;
  exclude_patterns?: string[];
  created_at: string;
  updated_at: string;
  status: string;
  metadata?: Record<string, unknown>;
  tracked_files?: TrackedFile[];
  visits?: Visit[];
  structure_snapshots?: StructureSnapshot[];
  total_files?: number;
  visited_files?: number;
  coverage_percent?: number;
}

export interface TrackedFile {
  id: string;
  file_path: string;
  absolute_path: string;
  visit_count: number;
  first_seen: string;
  last_visited?: string;
  last_modified: string;
  content_hash?: string;
  size_bytes: number;
  staleness_score: number;
  deleted: boolean;
  notes?: string;
  priority_weight?: number;
  excluded?: boolean;
  metadata?: Record<string, unknown>;
}

export interface Visit {
  id: string;
  file_id: string;
  timestamp: string;
  context?: string;
  agent?: string;
  conversation_id?: string;
  duration_ms?: number;
  findings?: Record<string, unknown>;
}

export interface StructureSnapshot {
  id: string;
  timestamp: string;
  total_files: number;
  new_files: string[];
  deleted_files: string[];
  moved_files: Record<string, string>;
  snapshot_data?: Record<string, unknown>;
}

// ==================== Insights Types ====================

export type PatternType = 'failure_mode' | 'timeout' | 'rate_limit' | 'stuck_state';
export type PatternSeverity = 'critical' | 'high' | 'medium' | 'low';
export type SuggestionType = 'prompt' | 'timeout' | 'code' | 'autosteer_profile';
export type SuggestionPriority = 'critical' | 'high' | 'medium' | 'low';
export type SuggestionStatus = 'pending' | 'applied' | 'rejected' | 'superseded';
export type ConfidenceLevel = 'high' | 'medium' | 'low';

export interface AnalysisWindow {
  start_time: string;
  end_time: string;
  limit: number;
  status_filter?: string;
}

export interface Pattern {
  id: string;
  type: PatternType;
  frequency: number;
  severity: PatternSeverity;
  description: string;
  examples: string[];
  evidence: string[];
}

export interface ProposedChange {
  file: string;
  type: 'edit' | 'create' | 'config_update';
  description: string;
  before?: string;
  after?: string;
  content?: string;
  config_path?: string;
  config_value?: unknown;
}

export interface ImpactEstimate {
  success_rate_improvement: string;
  time_reduction?: string;
  confidence: ConfidenceLevel;
  rationale: string;
}

export interface Suggestion {
  id: string;
  pattern_id: string;
  type: SuggestionType;
  priority: SuggestionPriority;
  title: string;
  description: string;
  changes: ProposedChange[];
  impact: ImpactEstimate;
  status: SuggestionStatus;
  applied_at?: string;
}

export interface ExecutionStatistics {
  total_executions: number;
  success_count: number;
  failure_count: number;
  timeout_count: number;
  rate_limit_count: number;
  success_rate: number;
  avg_duration: string;
  median_duration: string;
  most_common_exit_reason: string;
}

export interface InsightReport {
  id: string;
  task_id: string;
  generated_at: string;
  analysis_window: AnalysisWindow;
  execution_count: number;
  patterns: Pattern[];
  suggestions: Suggestion[];
  statistics: ExecutionStatistics;
  generated_by: string;
}

export interface CrossTaskPattern extends Pattern {
  affected_tasks: string[];
  task_types: string[];
}

export interface TaskTypeStats {
  count: number;
  success_rate: number;
  avg_duration: string;
  top_pattern: string;
}

export interface SystemInsightReport {
  id: string;
  generated_at: string;
  time_window: AnalysisWindow;
  task_count: number;
  total_executions: number;
  cross_task_patterns: CrossTaskPattern[];
  system_suggestions: Suggestion[];
  by_task_type: Record<string, TaskTypeStats>;
  by_operation: Record<string, TaskTypeStats>;
}

export interface GenerateInsightOptions {
  limit?: number;
  status_filter?: string;
  include_files?: string[];
}

export interface ApplySuggestionResult {
  success: boolean;
  message: string;
  files_changed?: string[];
  backup_path?: string;
}
