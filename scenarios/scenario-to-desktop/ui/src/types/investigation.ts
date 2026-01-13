/**
 * Types for pipeline investigation and fix tasks.
 * Mirrors the backend domain types in api/domain/task.go and api/domain/investigation.go.
 */

export type TaskType = "investigate" | "fix";

export type EffortLevel = "checks" | "logs" | "trace";

export type InvestigationStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface TaskFocus {
  harness: boolean;
  subject: boolean;
}

export interface FixPermissions {
  immediate: boolean;
  permanent: boolean;
  prevention: boolean;
}

export interface CreateTaskRequest {
  task_type: TaskType;
  focus: TaskFocus;
  effort?: EffortLevel;
  permissions?: FixPermissions;
  note?: string;
  include_contexts?: string[];
  source_investigation_id?: string;
  max_iterations?: number;
}

export interface FixIterationRecord {
  number: number;
  started_at: string;
  ended_at?: string;
  diagnosis_summary: string;
  changes_summary: string;
  rebuild_triggered: boolean;
  verify_result: string;
  outcome: string;
}

export interface FixIterationState {
  current_iteration: number;
  max_iterations: number;
  iterations: FixIterationRecord[];
  final_status?: string;
}

export interface InvestigationDetails {
  task_type?: TaskType;
  effort?: EffortLevel;
  focus?: TaskFocus;
  permissions?: FixPermissions;
  source_investigation_id?: string;
  included_contexts?: string[];
  fix_state?: FixIterationState;
}

export interface Investigation {
  id: string;
  pipeline_id: string;
  status: InvestigationStatus;
  findings?: string;
  progress: number;
  details?: InvestigationDetails;
  agent_run_id?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface InvestigationSummary {
  id: string;
  pipeline_id: string;
  status: InvestigationStatus;
  task_type: TaskType;
  progress: number;
  created_at: string;
  completed_at?: string;
}

export interface AgentManagerStatus {
  available: boolean;
  url?: string;
  reason?: string;
}

export interface CreateTaskResponse {
  task: Investigation;
}

export interface ListTasksResponse {
  tasks: InvestigationSummary[];
}

export interface GetTaskResponse {
  task: Investigation;
}

export interface StopTaskResponse {
  success: boolean;
  message: string;
}
