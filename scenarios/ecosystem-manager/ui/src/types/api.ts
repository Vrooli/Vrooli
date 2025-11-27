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

// SteerMode is now dynamic - any phase name from prompts/phases/*.md files
export type SteerMode = string;

export interface Task {
  id: string;
  title: string;
  type: TaskType;
  operation: OperationType;
  priority: Priority;
  status: TaskStatus;
  target?: string[];
  notes?: string;
  steer_mode?: SteerMode;
  auto_steer_profile_id?: string;
  auto_steer_mode?: string;
  auto_steer_phase_index?: number;
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
  steer_mode?: SteerMode;
  target?: string[];
  notes?: string;
  auto_steer_profile_id?: string;
  auto_requeue?: boolean;
}

export interface UpdateTaskInput {
  priority?: Priority;
  notes?: string;
  steer_mode?: SteerMode;
  auto_steer_profile_id?: string;
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
  active: boolean;
}

export interface AgentSettings {
  max_turns: number;
  allowed_tools?: string;
  skip_permissions: boolean;
  task_timeout_minutes: number;
  idle_timeout_cap_minutes: number;
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

export interface AutoSteerProfile {
  id: string;
  name: string;
  description?: string;
  tags?: string[];
  phases: AutoSteerPhase[];
  created_at?: string;
  updated_at?: string;
}

export interface AutoSteerPhase {
  id?: string;
  mode: SteerMode;
  max_iterations: number;
  description?: string;
  stop_conditions?: StopCondition[];
}

export interface StopCondition {
  type: 'simple' | 'compound';
  // Simple condition fields
  metric?: string;
  compare_operator?: string;
  value?: number;
  // Compound condition fields
  operator?: 'AND' | 'OR';
  conditions?: StopCondition[];
}

export interface ConditionNode {
  type: 'and' | 'or' | 'condition';
  field?: string;
  operator?: string;
  value?: string | number | boolean;
  children?: ConditionNode[];
}

export interface AutoSteerTemplate {
  id: string;
  name: string;
  description?: string;
  tags?: string[];
  phases?: AutoSteerPhase[];
}

export interface PhaseExecution {
  phase_id?: string;
  mode?: string;
  iterations: number;
  stop_reason?: string;
  started_at?: string;
  completed_at?: string | null;
}

export interface AutoSteerExecutionState {
  task_id: string;
  profile_id: string;
  current_phase_index: number;
  current_phase_iteration: number;
  auto_steer_iteration: number;
  phase_history?: PhaseExecution[];
  metrics?: MetricsSnapshot;
  phase_start_metrics?: MetricsSnapshot;
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

export interface PhasePerformance {
  mode: string;
  iterations: number;
  metric_deltas?: Record<string, number>;
  duration: number;
  effectiveness: number;
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
  phase_breakdown: PhasePerformance[];
  total_iterations: number;
  total_duration: number;
  user_feedback?: UserFeedback;
  executed_at: string;
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
  steer_mode?: string;
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
  name: string;
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
