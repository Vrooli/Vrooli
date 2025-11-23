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
  | 'archived'
  | 'review';

export type TaskType = 'resource' | 'scenario';

export type OperationType = 'generator' | 'improver';

export type Priority = 'critical' | 'high' | 'medium' | 'low';

export interface Task {
  id: string;
  title: string;
  type: TaskType;
  operation: OperationType;
  priority: Priority;
  status: TaskStatus;
  target?: string[];
  notes?: string;
  auto_steer_profile_id?: string;
  auto_requeue?: boolean;
  created_at: string;
  updated_at: string;
  execution_count?: number;
  current_process?: ProcessInfo;
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
}

export interface CreateTaskInput {
  title: string;
  type: TaskType;
  operation: OperationType;
  priority: Priority;
  target?: string[];
  notes?: string;
  auto_steer_profile_id?: string;
  auto_requeue?: boolean;
}

export interface UpdateTaskInput {
  title?: string;
  priority?: Priority;
  notes?: string;
  auto_steer_profile_id?: string;
  auto_requeue?: boolean;
}

// ==================== Queue Types ====================

export interface QueueStatus {
  active: boolean;
  slots_used: number;
  max_concurrent: number;
  tasks_remaining: number;
  next_refresh_in?: number;
  rate_limited?: boolean;
  rate_limit_retry_after?: number;
}

export interface RunningProcess {
  task_id: string;
  task_title: string;
  process_id: string;
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
  refresh_interval: number;
  max_tasks?: number;
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
  mode: string;
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
  logic_operator?: 'AND' | 'OR';
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

// ==================== Discovery Types ====================

export interface Resource {
  name: string;
  display_name: string;
  type: string;
  status: 'running' | 'stopped' | 'unknown';
  description?: string;
}

export interface Scenario {
  name: string;
  display_name: string;
  status: 'running' | 'stopped' | 'unknown';
  description?: string;
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
  start_time: string;
  end_time?: string;
  status: 'running' | 'completed' | 'failed';
  exit_code?: number;
  metadata?: Record<string, unknown>;
}

export interface ExecutionPrompt {
  content: string;
  metadata?: Record<string, unknown>;
}

export interface ExecutionOutput {
  content: string;
  metadata?: Record<string, unknown>;
}

// ==================== WebSocket Types ====================

export type WebSocketMessageType =
  | 'task_status_update'
  | 'process_started'
  | 'process_completed'
  | 'queue_status_update'
  | 'rate_limit_notification';

export interface WebSocketMessage {
  type: WebSocketMessageType;
  task_id?: string;
  status?: TaskStatus;
  process_id?: string;
  agent_id?: string;
  start_time?: string;
  active?: boolean;
  slots_used?: number;
  retry_after?: number;
  [key: string]: unknown;
}

// ==================== Health Check ====================

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

// ==================== Recycler Types ====================

export interface RecyclerTestPayload {
  output_text: string;
  expected_status?: TaskStatus;
}

export interface RecyclerTestResult {
  suggested_status: TaskStatus;
  confidence: number;
  reasoning?: string;
}

export interface PromptPreviewConfig {
  type: TaskType;
  operation: OperationType;
  title: string;
  priority: Priority;
  notes?: string;
  target?: string[];
}

export interface PromptPreviewResult {
  prompt: string;
  token_count?: number;
  sections?: Record<string, string>;
  metadata?: Record<string, unknown>;
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
