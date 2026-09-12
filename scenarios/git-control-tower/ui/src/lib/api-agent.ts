// ============================================================================
// Agent Manager — Types + API Functions
// ============================================================================

import { API_BASE, REPO_HEADER, buildRepoHeaders, handleResponse, buildApiUrl } from "./api-internals";

// ── Agent Manager Types ────────────────────────────────────────────────

export const RUN_STATUS = {
  PENDING: "pending",
  STARTING: "starting",
  RUNNING: "running",
  NEEDS_REVIEW: "needs_review",
  COMPLETE: "complete",
  FAILED: "failed",
  CANCELLED: "cancelled",
} as const;
export type AgentRunStatus = typeof RUN_STATUS[keyof typeof RUN_STATUS];
export const ACTIVE_STATUSES: AgentRunStatus[] = [RUN_STATUS.PENDING, RUN_STATUS.STARTING, RUN_STATUS.RUNNING];
export const TERMINAL_STATUSES: AgentRunStatus[] = [RUN_STATUS.COMPLETE, RUN_STATUS.FAILED, RUN_STATUS.CANCELLED];

export interface AgentProfile {
  id: string;
  key?: string;
  name: string;
  description?: string;
  model?: string;
  runnerType?: string;
}

export interface AgentProfileListResponse {
  profiles: AgentProfile[];
  total: number;
}

export interface AgentRunRequest {
  scenarioSlug: string;
  prompt: string;
  profileId?: string;
  profileKey?: string;
  attachmentIds?: string[];
}

export interface AgentRunCreateResponse {
  runId: string;
  taskId: string;
}

export interface AgentRunSummary {
  filesModified?: string[];
  filesCreated?: string[];
  filesDeleted?: string[];
  tokensUsed?: number;
  turnsUsed?: number;
  costEstimate?: number;
}

export interface AgentRunActions {
  canInvestigate?: boolean;
  canApplyInvestigation?: boolean;
  canDelete?: boolean;
  canStop: boolean;
  canRetry: boolean;
  canContinue: boolean;
  canApprove: boolean;
  canReject: boolean;
  canReview?: boolean;
  canExtractRecommendations?: boolean;
  canRegenerateRecommendations?: boolean;
}

export interface AgentRun {
  id: string;
  taskId?: string;
  sessionId?: string;
  status: AgentRunStatus;
  phase?: string;
  progressPercent?: number;
  errorMsg?: string;
  approvalState?: string;
  promptPreview?: string;
  sandboxId?: string;
  summary?: AgentRunSummary;
  actions?: AgentRunActions;
  createdAt: string;
  startedAt?: string;
  endedAt?: string;
}

export interface AgentRunListResponse {
  runs: AgentRun[];
  total: number;
}

export type AgentEventType = "message" | "tool_call" | "tool_result" | "error" | "status_change" | "log" | "progress";

export interface AgentRunEvent {
  id: string;
  runId: string;
  sequence: number;
  eventType: AgentEventType;
  timestamp: string;
  data?: unknown;
}

export interface AgentRunEventsResponse {
  events: AgentRunEvent[];
}

export interface AgentRunDiffFile {
  path: string;
  changeType: string;
  additions: number;
  deletions: number;
  isBinary?: boolean;
  patch?: string;
}

export interface AgentRunDiffResponse {
  runId: string;
  content?: string;
  files: AgentRunDiffFile[];
}

export interface AgentContinueRequest {
  message: string;
  attachment_ids?: string[];
}

export interface AttachmentUploadResponse {
  id: string;
  fileName: string;
  contentType: string;
  fileSize: number;
}

export interface AgentContinueResponse {
  success: boolean;
  run?: AgentRun;
}

export interface AgentApproveRequest {
  actor?: string;
  commitMsg?: string;
}

export interface AgentApproveResponse {
  success: boolean;
  filesApplied?: number;
  commitHash?: string;
  message?: string;
}

export interface AgentRejectRequest {
  actor?: string;
  reason?: string;
}

export interface AgentRejectResponse {
  status: string;
}

export interface AgentStopResponse {
  status: string;
}

export type AgentContextKind = "test-failure" | "screenshot" | "change-summary" | "scenario-quality";

export interface AgentContextItem {
  kind: AgentContextKind;
  id: string;
  label: string;
  markdown: string;
  /** Absolute filesystem paths for screenshot images (resolved at send time). */
  screenshotPaths?: string[];
}

// ── Agent Manager API Functions ────────────────────────────────────────

export async function uploadAgentAttachment(
  file: File,
  repoId?: string
): Promise<AttachmentUploadResponse> {
  const url = buildApiUrl("/agent/attachments/upload", { baseUrl: API_BASE });
  const formData = new FormData();
  formData.append("file", file);
  const headers: Record<string, string> = {};
  if (repoId) headers[REPO_HEADER] = repoId;
  const res = await fetch(url, {
    method: "POST",
    headers,
    body: formData,
  });
  return handleResponse<AttachmentUploadResponse>(res);
}

export async function fetchAgentProfiles(repoId?: string): Promise<AgentProfileListResponse> {
  const url = buildApiUrl("/agent/profiles", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<AgentProfileListResponse>(res);
}

export async function createAgentRun(
  request: AgentRunRequest,
  repoId?: string
): Promise<AgentRunCreateResponse> {
  const url = buildApiUrl("/agent/run", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<AgentRunCreateResponse>(res);
}

export async function fetchAgentRuns(
  scenarioSlug: string,
  limit?: number,
  repoId?: string
): Promise<AgentRunListResponse> {
  const params = new URLSearchParams({ scenarioSlug });
  if (limit) params.set("limit", String(limit));
  const url = buildApiUrl(`/agent/runs?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<AgentRunListResponse>(res);
}

export async function fetchAgentRun(runId: string, repoId?: string): Promise<AgentRun> {
  const url = buildApiUrl(`/agent/runs/${runId}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<AgentRun>(res);
}

export async function fetchAgentRunEvents(
  runId: string,
  afterSequence?: number,
  repoId?: string
): Promise<AgentRunEventsResponse> {
  const params = new URLSearchParams();
  if (afterSequence != null) params.set("afterSequence", String(afterSequence));
  const qs = params.toString();
  const url = buildApiUrl(`/agent/runs/${runId}/events${qs ? `?${qs}` : ""}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<AgentRunEventsResponse>(res);
}

export async function fetchAgentRunDiff(runId: string, repoId?: string): Promise<AgentRunDiffResponse> {
  const url = buildApiUrl(`/agent/runs/${runId}/diff`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<AgentRunDiffResponse>(res);
}

export async function continueAgentRun(
  runId: string,
  request: AgentContinueRequest,
  repoId?: string
): Promise<AgentContinueResponse> {
  const url = buildApiUrl(`/agent/runs/${runId}/continue`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<AgentContinueResponse>(res);
}

export async function approveAgentRun(
  runId: string,
  request: AgentApproveRequest,
  repoId?: string
): Promise<AgentApproveResponse> {
  const url = buildApiUrl(`/agent/runs/${runId}/approve`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<AgentApproveResponse>(res);
}

export async function rejectAgentRun(
  runId: string,
  request: AgentRejectRequest,
  repoId?: string
): Promise<AgentRejectResponse> {
  const url = buildApiUrl(`/agent/runs/${runId}/reject`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<AgentRejectResponse>(res);
}

export async function stopAgentRun(runId: string, repoId?: string): Promise<AgentStopResponse> {
  const url = buildApiUrl(`/agent/runs/${runId}/stop`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify({}),
  });
  return handleResponse<AgentStopResponse>(res);
}
