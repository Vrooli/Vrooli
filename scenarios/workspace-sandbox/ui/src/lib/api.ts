import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// --- Types matching the Go API ---

export type Status =
  | "creating"
  | "active"
  | "stopped"
  | "checkpointing"
  | "checkpointed"
  | "approved"
  | "rejected"
  | "deleted"
  | "error";
export type OwnerType = "agent" | "user" | "task" | "system";
export type ChangeType = "added" | "modified" | "deleted";
export type ApprovalStatus = "pending" | "approved" | "rejected";
export type ViewMode = "diff" | "full_diff" | "source";
export type LineChange = "" | "added" | "deleted";

/** Active-tab statuses: operationally interesting, restartable, actionable. */
export const ACTIVE_STATUSES: readonly Status[] = ["creating", "active", "stopped", "checkpointing", "checkpointed", "error"] as const;

/** History-tab statuses: terminal, audit-only. */
export const HISTORY_STATUSES: readonly Status[] = ["approved", "rejected", "deleted"] as const;

/** True when the sandbox lives in the History tab (terminal status). */
export function isHistoryStatus(s: Status): boolean {
  return HISTORY_STATUSES.includes(s);
}

/** Durability state of a diff snapshot. See ARCHIVE_DESIGN.md §3. */
export type ArchiveState = "complete" | "not_captured";

export interface AnnotatedLine {
  number: number;
  content: string;
  change?: LineChange;
  oldNumber?: number;
}

export interface FileViewData {
  fullContent?: string;
  annotatedLines?: AnnotatedLine[];
}

export interface MountHealth {
  healthy: boolean;
  verified: boolean;
  error?: string;
  hint?: string;
}

export interface Sandbox {
  id: string;
  name?: string;
  scopePath: string;
  reservedPath: string;
  reservedPaths?: string[];
  noLock?: boolean;
  projectRoot: string;
  owner?: string;
  ownerType: OwnerType;
  status: Status;
  errorMessage?: string;
  mountHealth?: MountHealth;
  createdAt: string;
  lastUsedAt: string;
  stoppedAt?: string;
  approvedAt?: string;
  deletedAt?: string;
  driverId: string;
  driverVersion: string;
  lowerDir?: string;
  upperDir?: string;
  workDir?: string;
  mergedDir?: string;
  sizeBytes: number;
  fileCount: number;
  activePids: number[];
  sessionCount: number;
  tags?: string[];
  metadata?: Record<string, unknown>;
}

export interface FileChange {
  id: string;
  sandboxId: string;
  filePath: string;
  changeType: ChangeType;
  fileSize: number;
  fileMode: number;
  detectedAt: string;
  approvalStatus: ApprovalStatus;
  approvedAt?: string;
  approvedBy?: string;
  acceptance?: {
    status: "accepted" | "ignored" | "denied" | "binary_ignored";
    reason?: string;
    rule?: string;
  };
}

export interface DiffStats {
  filesChanged: number;
  filesAdded: number;
  filesModified: number;
  filesDeleted: number;
  linesAdded: number;
  linesRemoved: number;
  totalBytes: number;
}

export interface DiffResult {
  sandboxId: string;
  files: FileChange[];
  unifiedDiff: string;
  generated: string;
  stats: DiffStats;
  // View mode support
  mode?: ViewMode;
  fileContents?: Record<string, FileViewData>;
  /**
   * Distinguishes archive-sourced responses from live ones. Empty (zero
   * value) means the diff was served from the live overlay; non-empty
   * values only appear when the response came from a durable archive.
   *   - undefined      → live overlay (Active/Stopped/Creating/Error)
   *   - "complete"     → archived snapshot with content blobs
   *   - "not_captured" → archived row, no blobs (e.g. Error → Deleted)
   */
  archiveState?: ArchiveState;
}

export interface ArchivedFileEntry {
  path: string;
  changeType: ChangeType;
  size: number;
  fileMode?: number;
  approvalStatus?: ApprovalStatus;
  blobSha256?: string;
}

/** Wire shape for an archived (terminal-status) sandbox diff. */
export interface DiffArchive {
  sandboxId: string;
  snapshotAt: string;
  archiveState: ArchiveState;
  /** Terminal status the sandbox transitioned to at snapshot time. */
  sandboxStatus: Status;
  files: ArchivedFileEntry[];
  stats: DiffStats;
  unifiedDiffSha256?: string;
  totalBlobBytes: number;
  projectRoot: string;
  owner?: string;
  agentManagerRunId?: string;
}

export interface HealthResponse {
  status: string;
  service: string;
  version: string;
  readiness: boolean;
  timestamp: string;
  dependencies: {
    database: string;
    driver: string;
  };
  config?: {
    projectRoot?: string;
  };
}

export interface DriverInfo {
  driver: {
    id: string;
    version: string;
    description: string;
    available: boolean;
  };
  available: boolean;
  error?: string;
}

// Driver Options types for Settings dialog
export interface DriverRequirement {
  name: string;
  met: boolean;
  current?: string;
  howToFix?: string;
  optional?: boolean;
}

export interface DriverCapabilities {
  filesystemIsolation: boolean;
  processIsolation: boolean;
  networkIsolation: boolean;
  directAccess: boolean;
  notes?: string;
}

export interface DriverOption {
  id: string;
  name: string;
  description: string;
  directAccess: boolean;
  capabilities: DriverCapabilities;
  requirements: DriverRequirement[];
  available: boolean;
  recommended?: boolean;
}

export interface DriverOptionsResponse {
  os: string;
  kernel?: string;
  currentDriver: string;
  inUserNamespace: boolean;
  options: DriverOption[];
}

export interface ListResult {
  sandboxes: Sandbox[];
  totalCount: number;
  limit: number;
  offset: number;
}

export interface ApprovalResult {
  success: boolean;
  applied: number;
  failed: number;
  commitHash?: string;
  error?: string;
  appliedAt: string;
}

export interface DiscardResult {
  success: boolean;
  discarded: number;
  remaining: number;
  error?: string;
  files?: string[];
}

export interface CreateRequest {
  name?: string;
  scopePath: string;
  reservedPath?: string;
  reservedPaths?: string[];
  noLock?: boolean;
  projectRoot?: string;
  owner?: string;
  ownerType?: OwnerType;
  tags?: string[];
  metadata?: Record<string, unknown>;
}

export interface ApprovalRequest {
  sandboxId: string;
  mode: "all" | "files" | "hunks";
  fileIds?: string[];
  hunkRanges?: Array<{
    fileId: string;
    startLine: number;
    endLine: number;
  }>;
  actor?: string;
  commitMessage?: string;
  overrideAcceptance?: boolean;
}

export interface ListFilter {
  name?: string;
  status?: Status[];
  owner?: string;
  projectRoot?: string;
  scopePath?: string;
  limit?: number;
  offset?: number;
}

/** Filter shape accepted by GET /sandboxes/history. */
export interface HistoryFilter {
  /** Subset of {approved, rejected, deleted}; empty means all three. */
  statuses?: Status[];
  projectRoot?: string;
  owner?: string;
  agentManagerRunId?: string;
  /** Free-text substring across owner / runId / sandboxId. */
  search?: string;
  /** RFC3339 lower bound on snapshot_at. */
  snapshotAtFrom?: string;
  /** RFC3339 upper bound on snapshot_at. */
  snapshotAtTo?: string;
  /** "snapshot_at" (default) or "total_blob_bytes". */
  sortBy?: "snapshot_at" | "total_blob_bytes";
  /** Default true (newest first) when sortBy is snapshot_at. */
  sortDesc?: boolean;
  limit?: number;
  offset?: number;
}

export interface HistoryListResult {
  archives: DiffArchive[];
  totalCount: number;
  limit: number;
  offset: number;
}

// --- API Functions ---

async function apiRequest<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const res = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(errorBody.error || `Request failed: ${res.status}`);
  }

  // Handle 204 No Content
  if (res.status === 204) {
    return {} as T;
  }

  return res.json();
}

// Health check
export async function fetchHealth(): Promise<HealthResponse> {
  return apiRequest<HealthResponse>("/health");
}

// Driver info
export async function fetchDriverInfo(): Promise<DriverInfo> {
  return apiRequest<DriverInfo>("/driver/info");
}

// Driver options (for Settings dialog)
export async function fetchDriverOptions(): Promise<DriverOptionsResponse> {
  return apiRequest<DriverOptionsResponse>("/driver/options");
}

// Driver selection types
export interface SelectDriverRequest {
  driverId: string;
}

export interface SelectDriverResponse {
  success: boolean;
  selectedDriver: string;
  requiresRestart: boolean;
  message: string;
}

export interface DriverPreferenceResponse {
  preference: string;
  currentDriver: string;
  isActive: boolean;
}

// Select driver
export async function selectDriver(driverId: string): Promise<SelectDriverResponse> {
  return apiRequest<SelectDriverResponse>("/driver/select", {
    method: "POST",
    body: JSON.stringify({ driverId }),
  });
}

// Get driver preference
export async function fetchDriverPreference(): Promise<DriverPreferenceResponse> {
  return apiRequest<DriverPreferenceResponse>("/driver/preference");
}

// List sandboxes
export async function listSandboxes(filter?: ListFilter): Promise<ListResult> {
  const params = new URLSearchParams();
  if (filter?.status) {
    filter.status.forEach((s) => params.append("status", s));
  }
  if (filter?.name) params.set("name", filter.name);
  if (filter?.owner) params.set("owner", filter.owner);
  if (filter?.projectRoot) params.set("projectRoot", filter.projectRoot);
  if (filter?.scopePath) params.set("scopePath", filter.scopePath);
  if (filter?.limit) params.set("limit", String(filter.limit));
  if (filter?.offset) params.set("offset", String(filter.offset));

  const query = params.toString();
  return apiRequest<ListResult>(`/sandboxes${query ? `?${query}` : ""}`);
}

// Get single sandbox
export async function getSandbox(id: string): Promise<Sandbox> {
  return apiRequest<Sandbox>(`/sandboxes/${id}`);
}

// List archived (History tab) sandboxes.
export async function listHistory(filter?: HistoryFilter): Promise<HistoryListResult> {
  const params = new URLSearchParams();
  filter?.statuses?.forEach((s) => params.append("status", s));
  if (filter?.projectRoot) params.set("projectRoot", filter.projectRoot);
  if (filter?.owner) params.set("owner", filter.owner);
  if (filter?.agentManagerRunId) params.set("agentManagerRunId", filter.agentManagerRunId);
  if (filter?.search) params.set("search", filter.search);
  if (filter?.snapshotAtFrom) params.set("snapshotAtFrom", filter.snapshotAtFrom);
  if (filter?.snapshotAtTo) params.set("snapshotAtTo", filter.snapshotAtTo);
  if (filter?.sortBy) params.set("sortBy", filter.sortBy);
  if (filter?.sortDesc) params.set("sortDesc", "true");
  if (filter?.limit) params.set("limit", String(filter.limit));
  if (filter?.offset) params.set("offset", String(filter.offset));

  const query = params.toString();
  return apiRequest<HistoryListResult>(`/sandboxes/history${query ? `?${query}` : ""}`);
}

// Create sandbox
export async function createSandbox(req: CreateRequest): Promise<Sandbox> {
  return apiRequest<Sandbox>("/sandboxes", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

// Delete sandbox
export async function deleteSandbox(id: string): Promise<void> {
  await apiRequest<void>(`/sandboxes/${id}`, { method: "DELETE" });
}

// Stop sandbox
export async function stopSandbox(id: string): Promise<Sandbox> {
  return apiRequest<Sandbox>(`/sandboxes/${id}/stop`, { method: "POST" });
}

// Start sandbox (remount a stopped sandbox)
export async function startSandbox(id: string): Promise<Sandbox> {
  return apiRequest<Sandbox>(`/sandboxes/${id}/start`, { method: "POST" });
}

// Resume sandbox (remount a checkpointed sandbox for the next turn)
export async function resumeSandbox(id: string): Promise<Sandbox> {
  return apiRequest<Sandbox>(`/sandboxes/${id}/resume`, { method: "POST" });
}

// Get diff
export async function getDiff(id: string, mode: ViewMode = "diff"): Promise<DiffResult> {
  const params = new URLSearchParams();
  if (mode !== "diff") {
    params.set("mode", mode);
  }
  const query = params.toString();
  return apiRequest<DiffResult>(`/sandboxes/${id}/diff${query ? `?${query}` : ""}`);
}

// Fetch the raw bytes of a single archived file blob, addressed by path
// inside the archive's index. Used by DiffViewer for on-demand expansion
// of large diffs whose per-file content was not embedded in the listing
// response.
export async function getArchivedFileContent(
  sandboxId: string,
  path: string
): Promise<string> {
  const params = new URLSearchParams({ path });
  const url = buildApiUrl(`/sandboxes/${sandboxId}/diff/file?${params.toString()}`, {
    baseUrl: API_BASE,
  });
  const res = await fetch(url);
  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(errorBody.error || `Request failed: ${res.status}`);
  }
  return res.text();
}

// Approve changes
export async function approveSandbox(
  id: string,
  options?: Partial<ApprovalRequest>
): Promise<ApprovalResult> {
  return apiRequest<ApprovalResult>(`/sandboxes/${id}/approve`, {
    method: "POST",
    body: JSON.stringify({
      mode: options?.mode || "all",
      fileIds: options?.fileIds,
      hunkRanges: options?.hunkRanges,
      actor: options?.actor,
      commitMessage: options?.commitMessage,
      overrideAcceptance: options?.overrideAcceptance,
    }),
  });
}

// Reject changes
export async function rejectSandbox(
  id: string,
  actor?: string
): Promise<Sandbox> {
  return apiRequest<Sandbox>(`/sandboxes/${id}/reject`, {
    method: "POST",
    body: JSON.stringify({ actor }),
  });
}

// Discard specific files
export async function discardFiles(
  sandboxId: string,
  options: {
    fileIds?: string[];
    filePaths?: string[];
    actor?: string;
  }
): Promise<DiscardResult> {
  return apiRequest<DiscardResult>(`/sandboxes/${sandboxId}/discard`, {
    method: "POST",
    body: JSON.stringify({
      fileIds: options.fileIds,
      filePaths: options.filePaths,
      actor: options.actor,
    }),
  });
}

// Get workspace path
export async function getWorkspacePath(
  id: string
): Promise<{ path: string }> {
  return apiRequest<{ path: string }>(`/sandboxes/${id}/workspace`);
}

// --- Derived Stats ---

export interface SandboxStats {
  total: number;
  active: number;
  stopped: number;
  checkpointing: number;
  checkpointed: number;
  approved: number;
  rejected: number;
  error: number;
  totalSizeBytes: number;
}

export function computeStats(sandboxes: Sandbox[]): SandboxStats {
  return sandboxes.reduce(
    (acc, sb) => {
      // Skip deleted sandboxes from stats - they shouldn't be shown in UI
      if (sb.status === "deleted") {
        return acc;
      }
      acc.total++;
      acc.totalSizeBytes += sb.sizeBytes || 0;
      switch (sb.status) {
        case "active":
        case "creating":
          acc.active++;
          break;
        case "stopped":
          acc.stopped++;
          break;
        case "checkpointing":
          acc.checkpointing++;
          break;
        case "checkpointed":
          acc.checkpointed++;
          break;
        case "approved":
          acc.approved++;
          break;
        case "rejected":
          acc.rejected++;
          break;
        case "error":
          acc.error++;
          break;
      }
      return acc;
    },
    {
      total: 0,
      active: 0,
      stopped: 0,
      checkpointing: 0,
      checkpointed: 0,
      approved: 0,
      rejected: 0,
      error: 0,
      totalSizeBytes: 0,
    }
  );
}

// Format file size
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

// Format relative time
export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}

// --- Path Validation ---

export interface PathValidationResult {
  path: string;
  projectRoot?: string;
  valid: boolean;
  exists?: boolean;
  isDirectory?: boolean;
  withinProjectRoot?: boolean;
  error?: string;
}

// Validate a path on the server
export async function validatePath(
  path: string,
  projectRoot?: string
): Promise<PathValidationResult> {
  const params = new URLSearchParams({ path });
  if (projectRoot) {
    params.set("projectRoot", projectRoot);
  }
  return apiRequest<PathValidationResult>(`/validate-path?${params.toString()}`);
}

// --- Process Execution Types ---

export interface ExecRequest {
  command: string;
  args?: string[];
  allowNetwork?: boolean;
  env?: Record<string, string>;
  workingDir?: string;
  sessionId?: string;
  isolationLevel?: string; // Profile ID (e.g., "full", "vrooli-aware", or custom profile IDs)
  memoryLimitMB?: number;
  cpuTimeSec?: number;
  timeoutSec?: number;
  maxProcesses?: number;
  maxOpenFiles?: number;
}

export interface ExecResponse {
  exitCode: number;
  stdout: string;
  stderr: string;
  pid?: number;
  timedOut?: boolean;
}

export interface StartProcessRequest {
  command: string;
  args?: string[];
  name?: string;
  allowNetwork?: boolean;
  env?: Record<string, string>;
  workingDir?: string;
  isolationLevel?: string; // Profile ID (e.g., "full", "vrooli-aware", or custom profile IDs)
  memoryLimitMB?: number;
  cpuTimeSec?: number;
  maxProcesses?: number;
  maxOpenFiles?: number;
}

export interface ProcessInfo {
  pid: number;
  sandboxId: string;
  command: string;
  name?: string;
  startedAt: string;
  logPath?: string;
}

export interface ProcessListResponse {
  processes: ProcessInfo[];
}

export interface LogResponse {
  pid: number;
  sandboxId: string;
  path: string;
  sizeBytes: number;
  isActive: boolean;
  content?: string;
}

export interface LogListResponse {
  sandboxId: string;
  logs: Array<{
    pid: number;
    path: string;
    sizeBytes: number;
    isActive: boolean;
  }>;
}

// --- Process Execution Functions ---

// Execute a command synchronously
export async function execCommand(
  sandboxId: string,
  req: ExecRequest
): Promise<ExecResponse> {
  return apiRequest<ExecResponse>(`/sandboxes/${sandboxId}/exec`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

// Start a background process
export async function startProcess(
  sandboxId: string,
  req: StartProcessRequest
): Promise<ProcessInfo> {
  return apiRequest<ProcessInfo>(`/sandboxes/${sandboxId}/processes`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}

// List processes in a sandbox
export async function listProcesses(
  sandboxId: string,
  runningOnly?: boolean
): Promise<ProcessListResponse> {
  const params = runningOnly ? "?running=true" : "";
  return apiRequest<ProcessListResponse>(`/sandboxes/${sandboxId}/processes${params}`);
}

// Kill a process
export async function killProcess(
  sandboxId: string,
  pid: number
): Promise<void> {
  await apiRequest<void>(`/sandboxes/${sandboxId}/processes/${pid}`, {
    method: "DELETE",
  });
}

// Kill all processes in a sandbox
export async function killAllProcesses(sandboxId: string): Promise<void> {
  await apiRequest<void>(`/sandboxes/${sandboxId}/processes`, {
    method: "DELETE",
  });
}

// Get process logs
export async function getProcessLogs(
  sandboxId: string,
  pid: number,
  options?: { tail?: number; offset?: number }
): Promise<LogResponse> {
  const params = new URLSearchParams();
  if (options?.tail) params.set("tail", String(options.tail));
  if (options?.offset) params.set("offset", String(options.offset));
  const query = params.toString();
  return apiRequest<LogResponse>(
    `/sandboxes/${sandboxId}/processes/${pid}/logs${query ? `?${query}` : ""}`
  );
}

// List all logs for a sandbox
export async function listLogs(sandboxId: string): Promise<LogListResponse> {
  return apiRequest<LogListResponse>(`/sandboxes/${sandboxId}/logs`);
}

// --- Execution Config and Isolation Profiles ---

export interface ResourceLimitsConfig {
  memoryLimitMB: number;
  cpuTimeSec: number;
  maxProcesses: number;
  maxOpenFiles: number;
  timeoutSec: number;
}

export interface ExecutionConfig {
  defaultResourceLimits: ResourceLimitsConfig;
  maxResourceLimits: ResourceLimitsConfig;
  defaultIsolationProfile: string;
  defaultNoLock: boolean;
}

export type NetworkAccess = "none" | "localhost" | "full";

export interface IsolationProfile {
  id: string;
  name: string;
  description: string;
  builtin: boolean;
  networkAccess: NetworkAccess;
  readOnlyBinds: Record<string, string>;
  readWriteBinds: Record<string, string>;
  environment: Record<string, string>;
  hostname: string;
}

// Get execution config
export async function fetchExecutionConfig(): Promise<ExecutionConfig> {
  return apiRequest<ExecutionConfig>("/config/execution");
}

// Update execution config
export async function updateExecutionConfig(
  config: ExecutionConfig
): Promise<ExecutionConfig> {
  return apiRequest<ExecutionConfig>("/config/execution", {
    method: "PUT",
    body: JSON.stringify(config),
  });
}

// List all isolation profiles
export async function fetchProfiles(): Promise<IsolationProfile[]> {
  return apiRequest<IsolationProfile[]>("/config/profiles");
}

// Get a single profile
export async function fetchProfile(id: string): Promise<IsolationProfile> {
  return apiRequest<IsolationProfile>(`/config/profiles/${id}`);
}

// Save (create/update) a profile
export async function saveProfile(
  profile: IsolationProfile
): Promise<IsolationProfile> {
  return apiRequest<IsolationProfile>(`/config/profiles/${profile.id}`, {
    method: "PUT",
    body: JSON.stringify(profile),
  });
}

// Delete a profile
export async function deleteProfile(id: string): Promise<void> {
  await apiRequest<void>(`/config/profiles/${id}`, {
    method: "DELETE",
  });
}

// --- Diff-archive retention configuration ---

export interface RetentionConfig {
  /** 0 disables age-based eviction. */
  maxArchiveAgeDays: number;
  /** 0 disables global size budget. */
  maxArchiveSizeBytes: number;
  /** 0 disables per-project cap. */
  maxArchivesPerProject: number;
}

/** Partial PUT — omit a field to leave it unchanged. */
export type RetentionUpdate = Partial<RetentionConfig>;

export async function fetchRetentionConfig(): Promise<RetentionConfig> {
  return apiRequest<RetentionConfig>("/config/retention");
}

export async function updateRetentionConfig(
  update: RetentionUpdate
): Promise<RetentionConfig> {
  return apiRequest<RetentionConfig>("/config/retention", {
    method: "PUT",
    body: JSON.stringify(update),
  });
}

// --- Commit Preview and Pending Types ---

export interface CommitPreviewFile {
  filePath: string;
  relativePath: string;
  changeType: string;
  sandboxId: string;
  sandboxOwner: string;
  appliedAt: string;
  // "pending" = still uncommitted, "already_committed" = committed externally
  status: "pending" | "already_committed";
}

export interface CommitPreviewSandboxGroup {
  sandboxId: string;
  sandboxOwner: string;
  fileCount: number;
  added: number;
  modified: number;
  deleted: number;
}

export interface CommitPreviewResult {
  files: CommitPreviewFile[];
  committableFiles: number;
  alreadyCommittedFiles: number;
  suggestedMessage: string;
  groupedBySandbox: CommitPreviewSandboxGroup[];
}

export interface CommitPendingRequest {
  projectRoot?: string;
  sandboxIds?: string[];
  commitMessage?: string;
  actor?: string;
}

export interface CommitPendingResult {
  success: boolean;
  filesCommitted: number;
  commitHash?: string;
  error?: string;
}

// --- Commit Preview and Pending Functions ---

// Get commit preview with reconciliation
export async function fetchCommitPreview(projectRoot?: string): Promise<CommitPreviewResult> {
  const params = new URLSearchParams();
  if (projectRoot) {
    params.set("projectRoot", projectRoot);
  }
  const query = params.toString();
  return apiRequest<CommitPreviewResult>(`/commit-preview${query ? `?${query}` : ""}`);
}

// Commit pending changes
export async function commitPending(req: CommitPendingRequest): Promise<CommitPendingResult> {
  return apiRequest<CommitPendingResult>("/commit-pending", {
    method: "POST",
    body: JSON.stringify(req),
  });
}
