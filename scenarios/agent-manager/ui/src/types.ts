// Agent Manager domain types - mirrors the Go domain types

export type RunnerType = "claude-code" | "codex" | "opencode";

export type TaskStatus = 
  | "queued"
  | "running"
  | "needs_review"
  | "approved"
  | "rejected"
  | "failed"
  | "cancelled";

export type RunStatus =
  | "pending"
  | "starting"
  | "running"
  | "needs_review"
  | "complete"
  | "failed"
  | "cancelled";

export type ApprovalState =
  | "none"
  | "pending"
  | "partially_approved"
  | "approved"
  | "rejected";

export type RunMode = "sandboxed" | "in_place";

export type RunPhase =
  | "queued"
  | "initializing"
  | "sandbox_creating"
  | "runner_acquiring"
  | "executing"
  | "collecting_results"
  | "awaiting_review"
  | "applying"
  | "cleaning_up"
  | "completed";

export type RunEventType =
  | "log"
  | "message"
  | "tool_call"
  | "tool_result"
  | "status"
  | "metric"
  | "artifact"
  | "error";

// Entity types
export interface AgentProfile {
  id: string;
  name: string;
  description?: string;
  runnerType: RunnerType;
  model?: string;
  maxTurns?: number;
  timeout?: number; // in nanoseconds (Go time.Duration)
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  requiresSandbox: boolean;
  requiresApproval: boolean;
  allowedPaths?: string[];
  deniedPaths?: string[];
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ContextAttachment {
  type: "file" | "link" | "note";
  path?: string;
  url?: string;
  content?: string;
}

export interface Task {
  id: string;
  title: string;
  description?: string;
  scopePath: string;
  projectRoot?: string;
  phasePromptIds?: string[];
  contextAttachments?: ContextAttachment[];
  status: TaskStatus;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RunSummary {
  description?: string;
  filesModified?: string[];
  filesCreated?: string[];
  filesDeleted?: string[];
  tokensUsed?: number;
  turnsUsed?: number;
  costEstimate?: number;
}

export interface Run {
  id: string;
  taskId: string;
  agentProfileId: string;
  sandboxId?: string;
  runMode: RunMode;
  status: RunStatus;
  startedAt?: string;
  endedAt?: string;
  phase: RunPhase;
  lastCheckpointId?: string;
  lastHeartbeat?: string;
  progressPercent: number;
  idempotencyKey?: string;
  summary?: RunSummary;
  errorMsg?: string;
  exitCode?: number;
  approvalState: ApprovalState;
  approvedBy?: string;
  approvedAt?: string;
  diffPath?: string;
  logPath?: string;
  changedFiles: number;
  totalSizeBytes: number;
  createdAt: string;
  updatedAt: string;
}

// Conflict information for sandbox scope conflicts
export interface ConflictingSandbox {
  sandboxId: string;
  scope: string;
  conflictType: string;
}

// Structured error details (e.g., for sandbox conflicts)
export interface ErrorDetails {
  operation?: string;
  is_transient?: boolean;
  can_retry?: boolean;
  sandbox_id?: string;
  cause?: string;
  conflicts?: ConflictingSandbox[];
}

export interface RunEventData {
  // Log events
  level?: string;
  message?: string;
  // Message events
  role?: string;
  content?: string;
  // Tool call events
  toolName?: string;
  input?: Record<string, unknown>;
  toolCallId?: string;
  // Tool result events
  output?: string;
  error?: string;
  success?: boolean;
  // Status events
  oldStatus?: string;
  newStatus?: string;
  reason?: string;
  // Metric events
  name?: string;
  value?: number;
  unit?: string;
  tags?: Record<string, string>;
  // Artifact events
  type?: string;
  path?: string;
  size?: number;
  mimeType?: string;
  // Error events
  code?: string;
  retryable?: boolean;
  recovery?: string;
  stackTrace?: string;
  details?: ErrorDetails; // Structured error details (e.g., conflicting sandboxes)
}

export interface RunEvent {
  id: string;
  runId: string;
  sequence: number;
  eventType: RunEventType;
  timestamp: string;
  data: RunEventData;
}

export interface DiffResult {
  unified: string;
  files: DiffFile[];
  stats: DiffStats;
}

export interface DiffFile {
  path: string;
  status: "added" | "modified" | "deleted";
  additions: number;
  deletions: number;
}

export interface DiffStats {
  filesChanged: number;
  additions: number;
  deletions: number;
  totalBytes: number;
}

// API response types
export interface HealthStatus {
  status: "healthy" | "degraded" | "unhealthy";
  service: string;
  timestamp: string;
  readiness: boolean;
  dependencies: {
    sandbox: {
      connected: boolean;
      error?: string;
    };
    runners: Record<string, {
      connected: boolean;
      error?: string;
    }>;
  };
  activeRuns: number;
  queuedTasks: number;
}

export interface RunnerStatus {
  type: RunnerType;
  available: boolean;
  message?: string;
  capabilities?: {
    SupportsMessages: boolean;
    SupportsToolEvents: boolean;
    SupportsCostTracking: boolean;
    SupportsStreaming: boolean;
    SupportsCancellation: boolean;
    SupportedModels: string[];
    MaxTurns: number;
  };
}

export interface ProbeResult {
  runnerType: RunnerType;
  success: boolean;
  message: string;
  response?: string;
  durationMs: number;
}

export interface ApprovalResult {
  success: boolean;
  commitHash?: string;
  message?: string;
}

// List/response wrappers (API v1)
export interface ListProfilesResponse {
  profiles: AgentProfile[];
  total: number;
}

export interface ListTasksResponse {
  tasks: Task[];
  total: number;
}

export interface ListRunsResponse {
  runs: Run[];
  total: number;
}

export interface GetRunnerStatusResponse {
  runners: RunnerStatus[];
}

export interface GetRunEventsResponse {
  events: RunEvent[];
}

export interface GetRunDiffResponse {
  diff: DiffResult;
}

export interface ProbeRunnerResponse {
  result: ProbeResult;
}

// Form types
export interface CreateProfileRequest {
  name: string;
  description?: string;
  runnerType: RunnerType;
  model?: string;
  maxTurns?: number;
  timeout?: number;
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  requiresSandbox?: boolean;
  requiresApproval?: boolean;
  allowedPaths?: string[];
  deniedPaths?: string[];
}

export interface CreateTaskRequest {
  title: string;
  description?: string;
  scopePath: string;
  projectRoot?: string;
  contextAttachments?: ContextAttachment[];
}

export interface CreateRunRequest {
  taskId: string;
  // Profile-based config (optional - can be omitted if inline config provided)
  agentProfileId?: string;
  // Custom tag for identification
  tag?: string;
  // Inline config (optional - used if no profile, or overrides profile)
  runnerType?: RunnerType;
  model?: string;
  maxTurns?: number;
  timeout?: number;
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  // Execution options
  prompt?: string;
  runMode?: RunMode;
  forceInPlace?: boolean;
  idempotencyKey?: string;
}

export interface ApproveRequest {
  actor: string;
  commitMsg?: string;
  force?: boolean;
}

export interface RejectRequest {
  actor: string;
  reason?: string;
}
