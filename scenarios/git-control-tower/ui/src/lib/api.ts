import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// ============================================================================
// Type Definitions
// ============================================================================

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  readiness: boolean;
  checks: {
    database: { status: string; message?: string };
    git: { status: string; message?: string };
    repo: { status: string; message?: string };
  };
}

export interface RepoBranchStatus {
  head: string;
  upstream?: string;
  ahead?: number;
  behind?: number;
  oid?: string;
}

export interface RepoFilesStatus {
  staged: string[];
  unstaged: string[];
  untracked: string[];
  conflicts: string[];
  binary?: string[];
  ignored?: string[];
  statuses?: Record<string, string>;
}

export interface RepoFileStats {
  staged?: Record<string, DiffStats>;
  unstaged?: Record<string, DiffStats>;
  untracked?: Record<string, DiffStats>;
}

export interface RepoStatusSummary {
  staged: number;
  unstaged: number;
  untracked: number;
  conflicts: number;
  ignored?: number;
}

export interface RepoStatus {
  repo_dir: string;
  branch: RepoBranchStatus;
  files: RepoFilesStatus;
  file_stats?: RepoFileStats;
  scopes: Record<string, string[]>;
  summary: RepoStatusSummary;
  author: {
    name?: string;
    email?: string;
  };
  timestamp: string;
}

export interface RepoHistoryResponse {
  repo_dir: string;
  lines: string[];
  entries?: RepoHistoryEntry[];
  limit: number;
  timestamp: string;
}

export interface RepoHistoryEntry {
  hash: string;
  author?: string;
  date?: string;
  subject: string;
  files: string[];
}

export interface DiffHunk {
  old_start: number;
  old_count: number;
  new_start: number;
  new_count: number;
  header: string;
  lines: string[];
}

export interface DiffStats {
  additions: number;
  deletions: number;
  files: number;
}

/** View mode for the diff viewer */
export type ViewMode = "diff" | "full_diff" | "source";

/** Type of change on a line */
export type LineChange = "" | "added" | "deleted" | "modified";

/** A single line with change annotation */
export interface AnnotatedLine {
  number: number;
  content: string;
  change?: LineChange;
  old_number?: number;
}

export interface DiffResponse {
  repo_dir: string;
  path?: string;
  staged: boolean;
  untracked?: boolean;
  base?: string;
  has_diff: boolean;
  hunks?: DiffHunk[];
  stats: DiffStats;
  raw?: string;
  full_content?: string;
  annotated_lines?: AnnotatedLine[];
  mode?: ViewMode;
  timestamp: string;
}

export interface StageRequest {
  paths: string[];
  scope?: string;
}

export interface StageResponse {
  success: boolean;
  staged: string[];
  failed?: string[];
  errors?: string[];
  timestamp: string;
}

export interface UnstageRequest {
  paths: string[];
  scope?: string;
}

export interface UnstageResponse {
  success: boolean;
  unstaged: string[];
  failed?: string[];
  errors?: string[];
  timestamp: string;
}

export interface CommitRequest {
  message: string;
  validate_conventional?: boolean;
  author_name?: string;
  author_email?: string;
}

export interface CommitResponse {
  success: boolean;
  hash?: string;
  error?: string;
  validation_errors?: string[];
  timestamp: string;
}

export interface DiscardRequest {
  paths: string[];
  untracked?: boolean;
}

export interface DiscardResponse {
  success: boolean;
  discarded: string[];
  failed?: string[];
  errors?: string[];
  timestamp: string;
}

export interface IgnoreRequest {
  path: string;
}

export interface IgnoreResponse {
  success: boolean;
  ignored: string[];
  failed?: string[];
  errors?: string[];
  gitignore_path?: string;
  timestamp: string;
}

export interface PushRequest {
  remote?: string;
  branch?: string;
  set_upstream?: boolean;
}

export interface PushResponse {
  success: boolean;
  remote: string;
  branch: string;
  error?: string;
  timestamp: string;
}

export interface PullRequest {
  remote?: string;
  branch?: string;
}

export interface PullResponse {
  success: boolean;
  remote: string;
  branch: string;
  error?: string;
  has_conflicts?: boolean;
  timestamp: string;
}

export interface SyncStatusResponse {
  branch: string;
  upstream?: string;
  remote_url?: string;
  ahead: number;
  behind: number;
  has_upstream: boolean;
  can_push: boolean;
  can_pull: boolean;
  needs_push: boolean;
  needs_pull: boolean;
  has_uncommitted_changes: boolean;
  safety_warnings?: string[];
  recommendations?: string[];
  fetched: boolean;
  fetch_error?: string;
  timestamp: string;
}

export interface ApprovedChangeFile {
  relativePath: string;
  status: string;
  sandboxId?: string;
  sandboxOwner?: string;
  changeType?: string;
}

export interface ApprovedChangesResponse {
  available: boolean;
  committableFiles: number;
  suggestedMessage?: string;
  files?: ApprovedChangeFile[];
  warning?: string;
}

export interface ApprovedChangesPreviewRequest {
  paths: string[];
}

// ============================================================================
// API Functions
// ============================================================================

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const error = await res.text();
    throw new Error(error || `Request failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<HealthResponse>(res);
}

export async function fetchRepoStatus(): Promise<RepoStatus> {
  const url = buildApiUrl("/repo/status", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<RepoStatus>(res);
}

export async function fetchRepoHistory(
  limit = 30,
  includeFiles = false
): Promise<RepoHistoryResponse> {
  const params = new URLSearchParams();
  if (limit > 0) params.set("limit", String(limit));
  if (includeFiles) params.set("include", "files");

  const url = buildApiUrl(`/repo/history?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<RepoHistoryResponse>(res);
}

export async function fetchDiff(
  path?: string,
  staged = false,
  untracked = false,
  commit?: string,
  mode: ViewMode = "diff"
): Promise<DiffResponse> {
  const params = new URLSearchParams();
  if (path) params.set("path", path);
  if (staged) params.set("staged", "true");
  if (untracked) params.set("untracked", "true");
  if (commit) params.set("commit", commit);
  if (mode && mode !== "diff") params.set("mode", mode);

  const url = buildApiUrl(`/repo/diff?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<DiffResponse>(res);
}

export async function stageFiles(request: StageRequest): Promise<StageResponse> {
  const url = buildApiUrl("/repo/stage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<StageResponse>(res);
}

export async function unstageFiles(request: UnstageRequest): Promise<UnstageResponse> {
  const url = buildApiUrl("/repo/unstage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<UnstageResponse>(res);
}

export async function createCommit(request: CommitRequest): Promise<CommitResponse> {
  const url = buildApiUrl("/repo/commit", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<CommitResponse>(res);
}

export async function discardFiles(request: DiscardRequest): Promise<DiscardResponse> {
  const url = buildApiUrl("/repo/discard", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<DiscardResponse>(res);
}

export async function ignoreFile(request: IgnoreRequest): Promise<IgnoreResponse> {
  const url = buildApiUrl("/repo/ignore", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<IgnoreResponse>(res);
}

export async function pushToRemote(request: PushRequest = {}): Promise<PushResponse> {
  const url = buildApiUrl("/repo/push", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<PushResponse>(res);
}

export async function pullFromRemote(request: PullRequest = {}): Promise<PullResponse> {
  const url = buildApiUrl("/repo/pull", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<PullResponse>(res);
}

export async function fetchSyncStatus(doFetch = false): Promise<SyncStatusResponse> {
  const params = new URLSearchParams();
  if (doFetch) params.set("fetch", "true");

  const url = buildApiUrl(`/repo/sync-status?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<SyncStatusResponse>(res);
}

export async function fetchApprovedChanges(): Promise<ApprovedChangesResponse> {
  const url = buildApiUrl("/repo/approved-changes", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<ApprovedChangesResponse>(res);
}

export async function fetchApprovedChangesPreview(
  request: ApprovedChangesPreviewRequest
): Promise<ApprovedChangesResponse> {
  const url = buildApiUrl("/repo/approved-changes/preview", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<ApprovedChangesResponse>(res);
}
