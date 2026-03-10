import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });
const REPO_HEADER = "X-Repo-Id";

function buildRepoHeaders(repoId?: string) {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (repoId) {
    headers[REPO_HEADER] = repoId;
  }
  return headers;
}

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

export interface RepoRecord {
  id: number;
  path: string;
  name: string;
  remote_url?: string;
  added_at: string;
  last_opened_at?: string;
  favorite?: boolean;
}

export interface RepoListResponse {
  repos: RepoRecord[];
  active_id?: number;
  timestamp: string;
}

export interface RepoActiveResponse {
  repo?: RepoRecord;
  timestamp: string;
}

export interface RepoOpenRequest {
  path: string;
}

export interface RepoCloneRequest {
  url: string;
  destination: string;
}

export interface RepoActiveRequest {
  id: number;
}

export interface RepoMutationResponse {
  repo?: RepoRecord;
  timestamp: string;
}

export interface RepoRemoveResponse {
  removed: boolean;
  timestamp: string;
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

export interface BranchInfo {
  name: string;
  upstream?: string;
  oid?: string;
  last_commit_at?: string;
  ahead?: number;
  behind?: number;
  is_current?: boolean;
}

export interface RepoBranchesResponse {
  current: string;
  locals: BranchInfo[];
  remotes: BranchInfo[];
  timestamp: string;
}

export interface BranchWarning {
  message: string;
  requires_confirmation?: boolean;
  requires_tracking?: boolean;
  requires_fetch?: boolean;
  dirty_summary?: RepoStatusSummary;
}

export interface CreateBranchRequest {
  name: string;
  from?: string;
  checkout?: boolean;
  allow_dirty?: boolean;
}

export interface BranchCreateResponse {
  success: boolean;
  branch?: BranchInfo;
  warning?: BranchWarning;
  error?: string;
  validation_errors?: string[];
  timestamp: string;
}

export interface SwitchBranchRequest {
  name: string;
  allow_dirty?: boolean;
  track_remote?: boolean;
}

export interface BranchSwitchResponse {
  success: boolean;
  branch?: BranchInfo;
  warning?: BranchWarning;
  error?: string;
  timestamp: string;
}

export interface PublishBranchRequest {
  remote?: string;
  branch?: string;
  set_upstream?: boolean;
  fetch?: boolean;
}

export interface BranchPublishResponse {
  success: boolean;
  remote: string;
  branch: string;
  warning?: BranchWarning;
  error?: string;
  timestamp: string;
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
  net_lines?: number;
  hunk_count?: number;
  largest_hunk?: number;
  density?: number;
  is_binary?: boolean;
  is_rename?: boolean;
  old_path?: string;
}

/** View mode for the diff viewer */
export type ViewMode = "diff" | "full_diff" | "source" | "preview";

/** View mode for the file list */
export type FileViewMode = "flat" | "grouped" | "tree";

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
  content_hash?: string;
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
  warnings?: string[];
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
  amend?: boolean;
  author_name?: string;
  author_email?: string;
}

export interface CommitResponse {
  success: boolean;
  hash?: string;
  amended?: boolean;
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
  level?: "project" | "group";
  group_dir?: string;
}

export interface IgnoreResponse {
  success: boolean;
  ignored: string[];
  failed?: string[];
  errors?: string[];
  gitignore_path?: string;
  timestamp: string;
}

// Grouping rules types
export interface GroupingRulesConfig {
  enabled: boolean;
  rules: GroupingRuleAPI[];
}

export interface GroupingRuleAPI {
  id: string;
  label: string;
  prefixes: string[];
  mode: string; // "prefix" | "segment"
}

// Gitignore health types
export interface GitignoreHealthResponse {
  root_entry_count: number;
  suggestions: GitignoreSuggestion[];
}

export interface GitignoreSuggestion {
  line: number;
  pattern: string;
  type: "single_group" | "cross_group";
  group_label: string;
  group_dir: string;
  target_pattern: string;
  has_gitignore: boolean;
}

export interface GitignoreMoveRequest {
  line: number;
  pattern: string;
  group_dir: string;
  target_pattern: string;
}

export interface GitignoreMoveResponse {
  success: boolean;
  removed_from?: string;
  added_to?: string;
  error?: string;
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
  pushed?: boolean;
  up_to_date?: boolean;
  verified?: boolean;
  verification_error?: string;
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

export type UpstreamActionType = "fetch" | "push_set_upstream" | "set_upstream";

export interface UpstreamActionRequest {
  action: UpstreamActionType;
  remote?: string;
  branch?: string;
  upstream?: string;
}

export interface UpstreamActionResponse {
  success: boolean;
  action: UpstreamActionType | string;
  remote?: string;
  branch?: string;
  upstream?: string;
  error?: string;
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

// File Search Types
export type FileStatus = "tracked" | "untracked" | "ignored";

export interface FileInfo {
  path: string;
  language?: string;
  status?: FileStatus;
}

export interface FileTreeResponse {
  files: FileInfo[];
  truncated: boolean;
  cancelled: boolean;
  search_mode: "default" | "deep";
  timestamp: string;
}

// Directory Listing Types (for lazy loading)
export interface DirEntry {
  name: string;
  path: string;
  is_dir: boolean;
  language?: string;
  tracked: boolean; // True if tracked by git (files only; folders true if they contain tracked files)
}

export interface DirListResponse {
  path: string;
  entries: DirEntry[];
  timestamp: string;
}

// Related Files Types
export type RelationType = "imports" | "imported_by" | "test" | "index" | "types";

export interface RelatedFile {
  path: string;
  relation_type: RelationType;
}

export interface RelatedFilesResponse {
  path: string;
  related: RelatedFile[];
  timestamp: string;
}

// Content Search Types
export interface ContentSearchRequest {
  query: string;
  case_sensitive?: boolean;
  whole_word?: boolean;
  regex?: boolean;
  include?: string; // Comma-separated globs
  exclude?: string; // Comma-separated globs
  context_lines?: number;
  limit?: number;
  timeout?: number;
}

export interface ContentSearchMatch {
  path: string;
  line_number: number;
  content: string;
  context_before?: string;
  context_after?: string;
}

export interface ContentSearchResponse {
  matches: ContentSearchMatch[];
  total: number;
  truncated: boolean;
  cancelled: boolean;
  query: string;
  timestamp: string;
}

// ============================================================================
// Credentials Types
// ============================================================================

export type CredentialType = "https" | "ssh";

export interface Credential {
  id: string;
  remote: string;
  url: string;
  type: CredentialType;
  username?: string;
  token_masked?: string;
  ssh_key_path?: string;
  is_configured: boolean;
  created_at: string;
  updated_at: string;
}

export interface CredentialsListResponse {
  credentials: Credential[];
  timestamp: string;
}

export interface CredentialSaveRequest {
  remote: string;
  url?: string;
  username?: string;
  token?: string;
  ssh_key_path?: string;
}

export interface CredentialSaveResponse {
  success: boolean;
  credential?: Credential;
  error?: string;
  timestamp: string;
}

export interface CredentialDeleteResponse {
  success: boolean;
  error?: string;
  timestamp: string;
}

export interface CredentialTestRequest {
  remote: string;
  use_stored?: boolean;
}

export interface CredentialTestResponse {
  success: boolean;
  reachable: boolean;
  authorized: boolean;
  error?: string;
  timestamp: string;
}

export interface RemoteURLUpdateRequest {
  remote: string;
  url: string;
}

export interface RemoteURLUpdateResponse {
  success: boolean;
  old_url?: string;
  new_url?: string;
  error?: string;
  timestamp: string;
}

// ============================================================================
// API Functions
// ============================================================================

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text();
    let message = text;
    if (text) {
      try {
        const parsed = JSON.parse(text) as { error?: string };
        if (parsed?.error) {
          message = parsed.error;
        }
      } catch {
        // Ignore JSON parse errors; fall back to raw text.
      }
    }
    throw new Error(message || `Request failed: ${res.status}`);
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

export async function fetchRepoStatus(repoId?: string): Promise<RepoStatus> {
  const url = buildApiUrl("/repo/status", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<RepoStatus>(res);
}

export async function fetchRepoHistory(
  limit = 30,
  includeFiles = false,
  repoId?: string
): Promise<RepoHistoryResponse> {
  const params = new URLSearchParams();
  if (limit > 0) params.set("limit", String(limit));
  if (includeFiles) params.set("include", "files");

  const url = buildApiUrl(`/repo/history?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<RepoHistoryResponse>(res);
}

export async function fetchDiff(
  path?: string,
  staged = false,
  untracked = false,
  commit?: string,
  mode: ViewMode = "diff",
  any = false,
  repoId?: string
): Promise<DiffResponse> {
  const params = new URLSearchParams();
  if (path) params.set("path", path);
  if (staged) params.set("staged", "true");
  if (untracked) params.set("untracked", "true");
  if (commit) params.set("commit", commit);
  if (mode && mode !== "diff") params.set("mode", mode);
  if (any) params.set("any", "true");

  const url = buildApiUrl(`/repo/diff?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<DiffResponse>(res);
}

export async function stageFiles(request: StageRequest, repoId?: string): Promise<StageResponse> {
  const url = buildApiUrl("/repo/stage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<StageResponse>(res);
}

export async function unstageFiles(request: UnstageRequest, repoId?: string): Promise<UnstageResponse> {
  const url = buildApiUrl("/repo/unstage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<UnstageResponse>(res);
}

export async function createCommit(request: CommitRequest, repoId?: string): Promise<CommitResponse> {
  const url = buildApiUrl("/repo/commit", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<CommitResponse>(res);
}

export async function discardFiles(request: DiscardRequest, repoId?: string): Promise<DiscardResponse> {
  const url = buildApiUrl("/repo/discard", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<DiscardResponse>(res);
}

export async function ignoreFile(request: IgnoreRequest, repoId?: string): Promise<IgnoreResponse> {
  const url = buildApiUrl("/repo/ignore", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<IgnoreResponse>(res);
}

export async function pushToRemote(
  request: PushRequest = {},
  repoId?: string
): Promise<PushResponse> {
  const url = buildApiUrl("/repo/push", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<PushResponse>(res);
}

export async function pullFromRemote(
  request: PullRequest = {},
  repoId?: string
): Promise<PullResponse> {
  const url = buildApiUrl("/repo/pull", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<PullResponse>(res);
}

export async function runUpstreamAction(
  request: UpstreamActionRequest,
  repoId?: string
): Promise<UpstreamActionResponse> {
  const url = buildApiUrl("/repo/upstream-action", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<UpstreamActionResponse>(res);
}

export async function fetchSyncStatus(
  doFetch = false,
  repoId?: string
): Promise<SyncStatusResponse> {
  const params = new URLSearchParams();
  if (doFetch) params.set("fetch", "true");

  const url = buildApiUrl(`/repo/sync-status?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<SyncStatusResponse>(res);
}

export async function fetchApprovedChanges(repoId?: string): Promise<ApprovedChangesResponse> {
  const url = buildApiUrl("/repo/approved-changes", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<ApprovedChangesResponse>(res);
}

export async function fetchApprovedChangesPreview(
  request: ApprovedChangesPreviewRequest,
  repoId?: string
): Promise<ApprovedChangesResponse> {
  const url = buildApiUrl("/repo/approved-changes/preview", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<ApprovedChangesResponse>(res);
}

export async function fetchBranches(repoId?: string): Promise<RepoBranchesResponse> {
  const url = buildApiUrl("/repo/branches", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<RepoBranchesResponse>(res);
}

export async function createBranch(
  request: CreateBranchRequest,
  repoId?: string
): Promise<BranchCreateResponse> {
  const url = buildApiUrl("/repo/branch/create", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<BranchCreateResponse>(res);
}

export async function switchBranch(
  request: SwitchBranchRequest,
  repoId?: string
): Promise<BranchSwitchResponse> {
  const url = buildApiUrl("/repo/branch/switch", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<BranchSwitchResponse>(res);
}

export async function publishBranch(
  request: PublishBranchRequest = {},
  repoId?: string
): Promise<BranchPublishResponse> {
  const url = buildApiUrl("/repo/branch/publish", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<BranchPublishResponse>(res);
}

export async function fetchFiles(
  pattern?: string,
  limit = 1000,
  deep = false,
  timeout = 5000,
  repoId?: string
): Promise<FileTreeResponse> {
  const params = new URLSearchParams();
  if (pattern) params.set("pattern", pattern);
  if (limit > 0) params.set("limit", String(limit));
  if (deep) params.set("deep", "true");
  if (timeout > 0) params.set("timeout", String(timeout));

  const url = buildApiUrl(`/repo/files?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<FileTreeResponse>(res);
}

export async function fetchRelatedFiles(
  path: string,
  repoId?: string
): Promise<RelatedFilesResponse> {
  const params = new URLSearchParams();
  params.set("path", path);

  const url = buildApiUrl(`/repo/related?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<RelatedFilesResponse>(res);
}

export async function searchContent(
  request: ContentSearchRequest,
  repoId?: string
): Promise<ContentSearchResponse> {
  const params = new URLSearchParams();
  params.set("query", request.query);
  if (request.case_sensitive) params.set("case_sensitive", "true");
  if (request.whole_word) params.set("whole_word", "true");
  if (request.regex) params.set("regex", "true");
  if (request.include) params.set("include", request.include);
  if (request.exclude) params.set("exclude", request.exclude);
  if (request.context_lines !== undefined) params.set("context_lines", String(request.context_lines));
  if (request.limit !== undefined) params.set("limit", String(request.limit));
  if (request.timeout !== undefined) params.set("timeout", String(request.timeout));

  const url = buildApiUrl(`/repo/search/content?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<ContentSearchResponse>(res);
}

export async function fetchDirectoryContents(path = "", repoId?: string): Promise<DirListResponse> {
  const params = new URLSearchParams();
  if (path) params.set("path", path);

  const url = buildApiUrl(`/repo/files/dir?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<DirListResponse>(res);
}

// Delete Path Types
export interface DeletePathRequest {
  path: string;
}

export interface DeletePathResponse {
  success: boolean;
  path: string;
  is_dir: boolean;
  error?: string;
  timestamp: string;
}

export interface SaveFileContentRequest {
  path: string;
  content: string;
  expected_hash?: string;
}

export interface SaveFileContentResponse {
  success: boolean;
  path: string;
  content_hash: string;
  bytes_written: number;
  timestamp: string;
}

export interface SaveFileContentConflictResponse {
  error: string;
  path: string;
  current_hash: string;
  timestamp: string;
}

export class FileContentConflictError extends Error {
  readonly path: string;
  readonly currentHash: string;

  constructor(message: string, path: string, currentHash: string) {
    super(message);
    this.name = "FileContentConflictError";
    this.path = path;
    this.currentHash = currentHash;
  }
}

export async function deletePath(
  request: DeletePathRequest,
  repoId?: string
): Promise<DeletePathResponse> {
  const url = buildApiUrl("/repo/files/delete", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<DeletePathResponse>(res);
}

export async function saveFileContent(
  request: SaveFileContentRequest,
  repoId?: string
): Promise<SaveFileContentResponse> {
  const url = buildApiUrl("/repo/files/content", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });

  if (res.status === 409) {
    const payload = (await res.json()) as SaveFileContentConflictResponse;
    throw new FileContentConflictError(
      payload.error || "File changed on disk",
      payload.path,
      payload.current_hash
    );
  }

  return handleResponse<SaveFileContentResponse>(res);
}

// ============================================================================
// Grouping Rules API Functions
// ============================================================================

export async function fetchGroupingRules(repoId?: string): Promise<GroupingRulesConfig> {
  const url = buildApiUrl("/repo/grouping-rules", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "GET",
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<GroupingRulesConfig>(res);
}

export async function saveGroupingRules(config: GroupingRulesConfig, repoId?: string): Promise<GroupingRulesConfig> {
  const url = buildApiUrl("/repo/grouping-rules", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(config),
  });
  return handleResponse<GroupingRulesConfig>(res);
}

// ============================================================================
// Gitignore Health API Functions
// ============================================================================

export async function fetchGitignoreHealth(repoId?: string): Promise<GitignoreHealthResponse> {
  const url = buildApiUrl("/repo/gitignore/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "GET",
    headers: buildRepoHeaders(repoId),
  });
  return handleResponse<GitignoreHealthResponse>(res);
}

export async function moveGitignoreEntry(request: GitignoreMoveRequest, repoId?: string): Promise<GitignoreMoveResponse> {
  const url = buildApiUrl("/repo/gitignore/move", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<GitignoreMoveResponse>(res);
}

// ============================================================================
// Credentials API Functions
// ============================================================================

export async function fetchCredentials(repoId?: string): Promise<CredentialsListResponse> {
  const url = buildApiUrl("/credentials", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<CredentialsListResponse>(res);
}

export async function saveCredential(
  request: CredentialSaveRequest,
  repoId?: string
): Promise<CredentialSaveResponse> {
  const url = buildApiUrl("/credentials", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<CredentialSaveResponse>(res);
}

export async function deleteCredential(
  id: string,
  repoId?: string
): Promise<CredentialDeleteResponse> {
  const url = buildApiUrl(`/credentials/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: buildRepoHeaders(repoId)
  });
  return handleResponse<CredentialDeleteResponse>(res);
}

export async function testCredential(
  request: CredentialTestRequest,
  repoId?: string
): Promise<CredentialTestResponse> {
  const url = buildApiUrl("/credentials/test", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<CredentialTestResponse>(res);
}

export async function updateRemoteURL(
  request: RemoteURLUpdateRequest,
  repoId?: string
): Promise<RemoteURLUpdateResponse> {
  const url = buildApiUrl("/repo/remote/url", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request)
  });
  return handleResponse<RemoteURLUpdateResponse>(res);
}

// ============================================================================
// Repo Registry API Functions
// ============================================================================

export async function fetchRepos(): Promise<RepoListResponse> {
  const url = buildApiUrl("/repos", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<RepoListResponse>(res);
}

export async function fetchActiveRepo(): Promise<RepoActiveResponse> {
  const url = buildApiUrl("/repos/active", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<RepoActiveResponse>(res);
}

export async function openRepo(request: RepoOpenRequest): Promise<RepoMutationResponse> {
  const url = buildApiUrl("/repos/open", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<RepoMutationResponse>(res);
}

export async function cloneRepo(request: RepoCloneRequest): Promise<RepoMutationResponse> {
  const url = buildApiUrl("/repos/clone", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<RepoMutationResponse>(res);
}

export async function setActiveRepo(request: RepoActiveRequest): Promise<RepoMutationResponse> {
  const url = buildApiUrl("/repos/active", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<RepoMutationResponse>(res);
}

export async function removeRepo(id: number): Promise<RepoRemoveResponse> {
  const url = buildApiUrl(`/repos/${encodeURIComponent(String(id))}`, {
    baseUrl: API_BASE
  });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });
  return handleResponse<RepoRemoveResponse>(res);
}

// ============================================================================
// SSH Key Management Types
// ============================================================================

export type SSHKeyType = "ed25519" | "rsa" | "ecdsa" | "dsa" | "unknown";

export interface SSHKeyInfo {
  path: string;
  filename: string;
  type: SSHKeyType;
  bits?: number;
  fingerprint: string;
  comment?: string;
  created_at?: string;
  has_public: boolean;
}

export interface SSHListKeysResponse {
  keys: SSHKeyInfo[];
  ssh_dir: string;
  timestamp: string;
}

export interface SSHGenerateKeyRequest {
  type: "ed25519" | "rsa";
  bits?: number;
  comment?: string;
  filename?: string;
}

export interface SSHGenerateKeyResponse {
  success: boolean;
  key?: SSHKeyInfo;
  public_key?: string;
  error?: string;
  timestamp: string;
}

export interface SSHGetPublicKeyRequest {
  key_path: string;
}

export interface SSHGetPublicKeyResponse {
  success: boolean;
  public_key?: string;
  fingerprint?: string;
  error?: string;
  timestamp: string;
}

export interface SSHTestConnectionRequest {
  key_path: string;
}

export interface SSHTestConnectionResponse {
  success: boolean;
  status: string;
  message?: string;
  hint?: string;
  github_user?: string;
  fingerprint?: string;
  latency_ms?: number;
  timestamp: string;
}

export interface SSHDeleteKeyRequest {
  key_path: string;
}

export interface SSHDeleteKeyResponse {
  success: boolean;
  message?: string;
  error?: string;
  private_deleted: boolean;
  public_deleted: boolean;
  timestamp: string;
}

// ============================================================================
// SSH Key Management API Functions
// ============================================================================

export async function fetchSSHKeys(): Promise<SSHListKeysResponse> {
  const url = buildApiUrl("/ssh/keys", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<SSHListKeysResponse>(res);
}

export async function generateSSHKey(request: SSHGenerateKeyRequest): Promise<SSHGenerateKeyResponse> {
  const url = buildApiUrl("/ssh/keys/generate", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHGenerateKeyResponse>(res);
}

export async function getSSHPublicKey(request: SSHGetPublicKeyRequest): Promise<SSHGetPublicKeyResponse> {
  const url = buildApiUrl("/ssh/keys/public", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHGetPublicKeyResponse>(res);
}

export async function testSSHConnection(request: SSHTestConnectionRequest): Promise<SSHTestConnectionResponse> {
  const url = buildApiUrl("/ssh/keys/test", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHTestConnectionResponse>(res);
}

export async function deleteSSHKey(request: SSHDeleteKeyRequest): Promise<SSHDeleteKeyResponse> {
  const url = buildApiUrl("/ssh/keys", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request)
  });
  return handleResponse<SSHDeleteKeyResponse>(res);
}
