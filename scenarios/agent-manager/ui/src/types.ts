import { ModelPreset, RunMode, RunnerType } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";
import type { ContextAttachment } from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";
export type { ContextAttachment } from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";

export {
  RunnerType,
  ModelPreset,
  TaskStatus,
  RunStatus,
  ApprovalState,
  RunMode,
  RunPhase,
  RunEventType,
  RecoveryAction,
} from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

export type {
  AgentProfile,
  RunConfig,
} from "@vrooli/proto-types/agent-manager/v1/domain/profile_pb";

export type {
  Task,
} from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";

export type {
  Run,
  RunSummary,
  RunnerStatus,
  ProbeResult,
  ApproveResult,
  StopAllResult,
  RunDiff,
} from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";

export type {
  RunEvent,
} from "@vrooli/proto-types/agent-manager/v1/domain/events_pb";

export type {
  HealthResponse,
  ErrorResponse,
  JsonValue,
  JsonObject,
} from "@vrooli/proto-types/common/v1/types_pb";

export { HealthStatus } from "@vrooli/proto-types/common/v1/types_pb";

export type ModelOption = string | { id: string; description?: string };

export interface RunnerModelRegistry {
  models: ModelOption[];
  presets: Record<string, string>;
}

export interface ModelRegistry {
  version: number;
  fallbackRunnerTypes?: string[];
  runners: Record<string, RunnerModelRegistry>;
}

export interface ProfileFormData {
  name: string;
  profileKey?: string;
  description?: string;
  runnerType: RunnerType;
  model?: string;
  modelPreset?: ModelPreset;
  maxTurns?: number;
  timeoutMinutes?: number;
  fallbackRunnerTypes?: RunnerType[];
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  requiresSandbox?: boolean;
  requiresApproval?: boolean;
  allowedPaths?: string[];
  deniedPaths?: string[];
}

export interface TaskFormData {
  title: string;
  description?: string;
  scopePath: string;
  projectRoot?: string;
  contextAttachments?: ContextAttachment[];
}

export interface RunFormData {
  taskId: string;
  agentProfileId?: string;
  tag?: string;
  existingSandboxId?: string;
  runnerType?: RunnerType;
  model?: string;
  modelPreset?: ModelPreset;
  maxTurns?: number;
  timeoutMinutes?: number;
  fallbackRunnerTypes?: RunnerType[];
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  requiresSandbox?: boolean;
  requiresApproval?: boolean;
  allowedPaths?: string[];
  deniedPaths?: string[];
  prompt?: string;
  runMode?: RunMode;
  idempotencyKey?: string;
}

export interface ApproveFormData {
  actor?: string;
  commitMsg?: string;
  force?: boolean;
}

export interface RejectFormData {
  actor?: string;
  reason?: string;
}

// Investigation types
export type InvestigationStatus = "pending" | "running" | "completed" | "failed" | "cancelled";

export interface AnalysisType {
  error_diagnosis: boolean;
  efficiency_analysis: boolean;
  tool_usage_patterns: boolean;
}

export interface ReportSections {
  root_cause_evidence: boolean;
  recommendations: boolean;
  metrics_summary: boolean;
}

export interface Evidence {
  run_id: string;
  event_seq?: number;
  description: string;
  snippet?: string;
}

export interface RootCauseAnalysis {
  description: string;
  evidence: Evidence[];
  confidence: "high" | "medium" | "low";
}

export interface Recommendation {
  id: string;
  priority: "critical" | "high" | "medium" | "low";
  title: string;
  description: string;
  action_type: "prompt_change" | "profile_config" | "code_fix";
}

export interface InvestigationReport {
  summary: string;
  root_cause?: RootCauseAnalysis;
  recommendations?: Recommendation[];
}

export interface MetricsData {
  total_runs: number;
  success_rate: number;
  avg_duration_seconds: number;
  total_tokens_used: number;
  total_cost: number;
  tool_usage_breakdown: Record<string, number>;
  error_type_breakdown: Record<string, number>;
  custom_metrics: Record<string, number>;
}

export interface Investigation {
  id: string;
  run_ids: string[];
  status: InvestigationStatus;
  analysis_type: AnalysisType;
  report_sections: ReportSections;
  custom_context?: string;
  progress: number;
  agent_run_id?: string;
  findings?: InvestigationReport;
  metrics?: MetricsData;
  error_message?: string;
  source_investigation_id?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface CreateInvestigationRequest {
  run_ids: string[];
  analysis_type: AnalysisType;
  report_sections: ReportSections;
  custom_context?: string;
}

export interface ApplyFixesRequest {
  recommendation_ids: string[];
  note?: string;
}
