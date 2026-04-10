export type InvestigationStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

// New unified task types
export type TaskType = 'investigate' | 'fix';
export type EffortLevel = 'checks' | 'logs' | 'trace';
export type FocusType = 'harness' | 'subject';
export type PermissionType = 'immediate' | 'permanent' | 'prevention';

export interface TaskFocus {
  harness: boolean;
  subject: boolean;
}

export interface FixPermissions {
  immediate: boolean;
  permanent: boolean;
  prevention: boolean;
}

export interface FixIterationRecord {
  number: number;
  started_at: string;
  ended_at?: string;
  diagnosis_summary?: string;
  changes_summary?: string;
  deploy_triggered: boolean;
  verify_result?: string;
  outcome?: string;
  agent_run_id?: string;
}

export interface FixIterationState {
  current_iteration: number;
  max_iterations: number;
  iterations: FixIterationRecord[];
  final_status?: string;
}

export interface Investigation {
  id: string;
  deployment_id: string;
  deployment_run_id?: string;
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

export interface InvestigationDetails {
  source: string;
  run_id?: string;
  duration_seconds?: number;
  tokens_used?: number;
  cost_estimate?: number;
  operation_mode: string;
  trigger_reason: string;
  deployment_step?: string;
  source_investigation_id?: string;
  source_findings?: string;
  // New task-specific fields
  task_type?: TaskType;
  focus?: TaskFocus;
  effort?: EffortLevel;
  permissions?: FixPermissions;
  fix_state?: FixIterationState;
}

export interface InvestigationSummary {
  id: string;
  deployment_id: string;
  deployment_run_id?: string;
  status: InvestigationStatus;
  progress: number;
  has_findings: boolean;
  error_message?: string;
  source_investigation_id?: string;
  created_at: string;
  completed_at?: string;
}

export interface CreateInvestigationRequest {
  auto_fix?: boolean;
  note?: string;
  include_contexts?: string[];
}

// New unified task request
export interface CreateTaskRequest {
  task_type: TaskType;
  focus: TaskFocus;
  note?: string;
  // Investigate-only fields
  effort?: EffortLevel;
  // Fix-only fields
  permissions?: FixPermissions;
  source_investigation_id?: string;
  max_iterations?: number;
  // Context selection
  include_contexts?: string[];
}

export interface ApplyFixesRequest {
  immediate: boolean;
  permanent: boolean;
  prevention: boolean;
  note?: string;
}

export interface AgentManagerStatus {
  enabled: boolean;
  available: boolean;
  message?: string;
}
