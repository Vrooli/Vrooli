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
  file_hotspots?: Record<string, number>;
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
  comment_additions?: number;
  comment_deletions?: number;
  is_new_file?: boolean;
  is_deleted_file?: boolean;
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
// Capabilities Types
// ============================================================================

export type DependencyKind = "scenario" | "resource";
export type CapabilityStatus = "available" | "unavailable" | "unknown";

export interface CapabilityState {
  id: string;
  name: string;
  description: string;
  dependencyKind: DependencyKind;
  dependencySlug: string;
  features: string[];
  status: CapabilityStatus;
  message?: string;
  checkedAt?: string;
}

export interface CapabilitiesResponse {
  capabilities: CapabilityState[];
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

/** Extract a short readable message from an HTML error page. */
function extractHtmlErrorMessage(html: string, status: number): string {
  const titleMatch = html.match(/<title[^>]*>([\s\S]*?)<\/title>/i);
  if (titleMatch?.[1]) {
    const title = titleMatch[1].trim();
    if (title && title.length < 200) return title;
  }
  const h1Match = html.match(/<h1[^>]*>([\s\S]*?)<\/h1>/i);
  if (h1Match?.[1]) {
    const heading = h1Match[1].replace(/<[^>]*>/g, "").trim();
    if (heading && heading.length < 200) return heading;
  }
  return `Server returned an HTML error (status ${status})`;
}

const MAX_ERROR_LENGTH = 500;

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text();
    let message = text;
    if (text) {
      // Detect HTML responses (proxies, panic recovery, etc.)
      const isHtml = text.trimStart().startsWith("<") || res.headers.get("content-type")?.includes("text/html");
      if (isHtml) {
        message = extractHtmlErrorMessage(text, res.status);
      } else {
        try {
          const parsed = JSON.parse(text) as { error?: string };
          if (parsed?.error) {
            message = parsed.error;
          }
        } catch {
          // Ignore JSON parse errors; fall back to raw text.
        }
      }
    }
    if (message && message.length > MAX_ERROR_LENGTH) {
      message = message.slice(0, MAX_ERROR_LENGTH) + "…";
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
// Capabilities API Functions
// ============================================================================

export async function fetchCapabilities(): Promise<CapabilitiesResponse> {
  const url = buildApiUrl("/capabilities", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<CapabilitiesResponse>(res);
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

// ============================================================================
// Visual Capture Types
// ============================================================================

export type CaptureTrigger = "periodic" | "post-commit" | "manual";
export type CaptureStatus = "complete" | "failed";
export type SnapshotRole = "baseline" | "capture";
export type CaptureMode = "baseline" | "capture";

export type CaptureTheme = "light" | "dark";

export interface CapturePreset {
  name: string;
  width: number;
  height: number;
  theme: CaptureTheme;
}

export const SIZE_PRESETS: Record<string, { width: number; height: number }> = {
  Desktop: { width: 1440, height: 900 },
  Tablet: { width: 768, height: 1024 },
  Mobile: { width: 390, height: 844 },
};

export const DEFAULT_PRESETS: CapturePreset[] = [
  { name: "Desktop Light", width: 1440, height: 900, theme: "light" },
];

export function presetSuffix(p: CapturePreset): string {
  return `@${p.width}x${p.height}_${p.theme}`;
}

export function presetLabel(p: CapturePreset): string {
  return p.name;
}

export function presetKey(p: CapturePreset): string {
  return `${p.width}x${p.height}_${p.theme}`;
}

export function getCapturePresets(scenarioSlug: string): CapturePreset[] {
  const stored = localStorage.getItem(`gct.capturePresets.${scenarioSlug}`);
  if (stored) {
    try {
      return JSON.parse(stored);
    } catch { /* fall through */ }
  }
  return DEFAULT_PRESETS;
}

export function setCapturePresets(scenarioSlug: string, presets: CapturePreset[]): void {
  localStorage.setItem(`gct.capturePresets.${scenarioSlug}`, JSON.stringify(presets));
}

export interface SnapshotStalenessInfo {
  isStale: boolean;
  lastFileChange?: string;
  captureCreatedAt?: string;
}

export interface SnapshotFile {
  filename: string;
  pagePath?: string;
  pageLabel?: string;
  width?: number;
  height?: number;
  viewportWidth?: number;
  viewportHeight?: number;
  theme?: string;
  sizeBytes: number;
}

export interface SnapshotSetMeta {
  id: string;
  scenarioSlug: string;
  role: SnapshotRole;
  commitHash?: string;
  triggerType: CaptureTrigger;
  pages: string[];
  screenshotCount: number;
  videoCount: number;
  videoStatus?: "not_implemented" | "disabled" | "captured" | "failed";
  createdAt: string;
  sizeBytes: number;
  status: CaptureStatus;
  error?: string;
  presets: CapturePreset[];
  pageDiscoveryMethod?: "lighthouse" | "fallback" | "explicit";
}

export interface SnapshotSetDetail extends SnapshotSetMeta {
  screenshots: SnapshotFile[];
  videos: SnapshotFile[];
}

export interface VisualCaptureListResponse {
  snapshots: SnapshotSetMeta[];
  total: number;
  staleness?: SnapshotStalenessInfo;
}

export interface CaptureStorageStats {
  totalSizeBytes: number;
  snapshotCount: number;
  perScenario: { scenarioSlug: string; snapshotCount: number; sizeBytes: number }[];
}

// ============================================================================
// Visual Capture API Functions
// ============================================================================

export async function triggerVisualCapture(scenarioSlug: string, mode: CaptureMode = "capture", repoId?: string, presets?: CapturePreset[]): Promise<SnapshotSetMeta> {
  const url = buildApiUrl("/repo/visual-capture", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify({ scenarioSlug, mode, presets })
  });
  return handleResponse<SnapshotSetMeta>(res);
}

export async function fetchVisualCaptures(scenarioSlug: string, repoId?: string): Promise<VisualCaptureListResponse> {
  const params = new URLSearchParams({ scenarioSlug });
  const url = buildApiUrl(`/repo/visual-captures?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<VisualCaptureListResponse>(res);
}

export async function fetchVisualCaptureDetail(id: string, scenarioSlug: string, repoId?: string): Promise<SnapshotSetDetail> {
  const params = new URLSearchParams({ scenarioSlug });
  const url = buildApiUrl(`/repo/visual-captures/${encodeURIComponent(id)}?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<SnapshotSetDetail>(res);
}

export async function fetchCaptureStorageStats(repoId?: string): Promise<CaptureStorageStats> {
  const url = buildApiUrl("/repo/visual-capture-storage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<CaptureStorageStats>(res);
}

export async function deleteVisualCapture(id: string, scenarioSlug: string, repoId?: string): Promise<void> {
  const params = new URLSearchParams({ scenarioSlug });
  const url = buildApiUrl(`/repo/visual-captures/${encodeURIComponent(id)}?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: buildRepoHeaders(repoId)
  });
  await handleResponse<unknown>(res);
}

export async function clearAllCaptureStorage(repoId?: string): Promise<void> {
  const url = buildApiUrl("/repo/visual-capture-storage", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: buildRepoHeaders(repoId)
  });
  await handleResponse<unknown>(res);
}

export function buildCaptureScreenshotUrl(captureId: string, scenarioSlug: string, filename: string): string {
  const params = new URLSearchParams({ scenarioSlug });
  return buildApiUrl(`/repo/visual-captures/${encodeURIComponent(captureId)}/screenshot/${encodeURIComponent(filename)}?${params.toString()}`, { baseUrl: API_BASE });
}

export async function fetchScreenshotPath(captureId: string, scenarioSlug: string, filename: string, repoId?: string): Promise<string> {
  const params = new URLSearchParams({ scenarioSlug });
  const url = buildApiUrl(`/repo/visual-captures/${encodeURIComponent(captureId)}/screenshot/${encodeURIComponent(filename)}/path?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, { headers: buildRepoHeaders(repoId) });
  const data = await handleResponse<{ path: string }>(res);
  return data.path;
}

export function buildCaptureVideoUrl(captureId: string, scenarioSlug: string, filename: string): string {
  const params = new URLSearchParams({ scenarioSlug });
  return buildApiUrl(`/repo/visual-captures/${encodeURIComponent(captureId)}/video/${encodeURIComponent(filename)}?${params.toString()}`, { baseUrl: API_BASE });
}

// ============================================================================
// Workflow Capture Types
// ============================================================================

export type ExecutionMode = "observer" | "mutating" | "destructive";

export interface WorkflowExecutionResult {
  workflowName: string;
  executionMode: ExecutionMode;
  executionId?: string;
  status: "passed" | "failed" | "skipped" | "error";
  error?: string;
  durationMs: number;
  videoCount: number;
  videoStatus?: "captured" | "failed" | "none";
}

export interface WorkflowCaptureResult {
  id: string;
  scenarioSlug: string;
  role: SnapshotRole;
  workflowResults: WorkflowExecutionResult[];
  createdAt: string;
  status: "complete" | "failed";
  error?: string;
  sizeBytes: number;
}

export interface WorkflowCaptureListResponse {
  captures: WorkflowCaptureResult[];
  total: number;
  staleness?: SnapshotStalenessInfo;
}

export interface WorkflowCaptureDetailResponse {
  capture: WorkflowCaptureResult;
  videos: SnapshotFile[];
}

// ============================================================================
// Workflow Capture API Functions
// ============================================================================

export async function triggerWorkflowCapture(
  scenarioSlug: string,
  mode: CaptureMode = "capture",
  executionModes: ExecutionMode[] = ["observer"],
  repoId?: string
): Promise<WorkflowCaptureResult> {
  const url = buildApiUrl("/repo/workflow-capture", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify({ scenarioSlug, mode, executionModes })
  });
  return handleResponse<WorkflowCaptureResult>(res);
}

export async function fetchWorkflowCaptures(scenarioSlug: string, repoId?: string): Promise<WorkflowCaptureListResponse> {
  const params = new URLSearchParams({ scenarioSlug });
  const url = buildApiUrl(`/repo/workflow-captures?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<WorkflowCaptureListResponse>(res);
}

export async function fetchWorkflowCaptureDetail(id: string, scenarioSlug: string, repoId?: string): Promise<WorkflowCaptureDetailResponse> {
  const params = new URLSearchParams({ scenarioSlug });
  const url = buildApiUrl(`/repo/workflow-captures/${encodeURIComponent(id)}?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<WorkflowCaptureDetailResponse>(res);
}

export function buildWorkflowVideoUrl(captureId: string, scenarioSlug: string, filename: string): string {
  const params = new URLSearchParams({ scenarioSlug });
  return buildApiUrl(`/repo/workflow-captures/${encodeURIComponent(captureId)}/video/${encodeURIComponent(filename)}?${params.toString()}`, { baseUrl: API_BASE });
}

// ============================================================================
// Test Execution Types (mirrors test-genie API)
// ============================================================================

export interface TestExecutionRequest {
  scenarioName: string;
  preset?: string;
  phases?: string[];
  skip?: string[];
  failFast?: boolean;
}

export interface TestExecutionResult {
  executionId: string;
  scenarioName: string;
  success: boolean;
  startedAt: string;
  completedAt?: string;
  preset?: string;
  phases: TestPhaseResult[];
  phaseSummary: TestPhaseSummary;
  warnings?: string[];
}

export interface TestPhaseResult {
  name: string;
  status: "passed" | "failed";
  durationSeconds: number;
  logPath?: string;
  error?: string;
  classification?: string;
  remediation?: string;
  observations?: TestObservation[];
}

export interface TestPhaseSummary {
  total: number;
  passed: number;
  failed: number;
  durationSeconds: number;
  observationCount: number;
}

export interface TestObservation {
  icon?: string;
  prefix?: string;
  section?: string;
  text: string;
}

export interface TestExecutionListResponse {
  items: TestExecutionResult[];
  count: number;
}

// ============================================================================
// Test Execution API Functions
// ============================================================================

export async function triggerTestExecution(
  request: TestExecutionRequest,
  repoId?: string
): Promise<TestExecutionResult> {
  const url = buildApiUrl("/repo/test-execution", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<TestExecutionResult>(res);
}

export async function fetchTestExecutions(
  scenarioName: string,
  limit = 10,
  repoId?: string
): Promise<TestExecutionListResponse> {
  const params = new URLSearchParams({ scenarioName, limit: String(limit) });
  const url = buildApiUrl(`/repo/test-executions?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TestExecutionListResponse>(res);
}

export async function fetchTestExecution(
  id: string,
  repoId?: string
): Promise<TestExecutionResult> {
  const url = buildApiUrl(`/repo/test-executions/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TestExecutionResult>(res);
}

// ============================================================================
// Tidiness Manager Types
// ============================================================================

export interface TidinessBreakdown {
  lint_issues: number;
  type_issues: number;
  long_files: number;
  complex_functions: number;
  tech_debt_markers: number;
  duplication_issues: number;
}

export interface TidinessMetricsSummary {
  total_files: number;
  total_lines: number;
  avg_file_length: number;
  max_complexity: number;
  avg_complexity: number;
  duplication_pct: number;
}

export interface TidinessScoreResponse {
  scenario: string;
  score: number;
  violations: number;
  last_scan?: string;
  breakdown?: TidinessBreakdown;
  metrics?: TidinessMetricsSummary;
}

export interface TidinessIssue {
  id: number;
  scenario: string;
  file_path: string;
  category: string;
  severity: string;
  title: string;
  description: string;
  line_number?: number;
  column_number?: number;
  agent_notes?: string;
  remediation_steps?: string;
  status: string;
  created_at: string;
}

export interface TidinessStalenessInfo {
  last_scan_at?: string;
  is_stale: boolean;
  modified_files?: number;
  stale_reason?: string;
  rescan_command?: string;
}

export interface TidinessLightScanRequest {
  scenario_name: string;
  timeout_sec?: number;
  incremental?: boolean;
}

export interface TidinessFileMetric {
  path: string;
  lines: number;
  extension: string;
}

export interface TidinessLongFile {
  path: string;
  lines: number;
  threshold: number;
}

export interface TidinessLightScanResult {
  scenario: string;
  started_at: string;
  completed_at: string;
  duration_ms: number;
  file_metrics: TidinessFileMetric[];
  long_files: TidinessLongFile[];
  total_files: number;
  total_lines: number;
  lint_issues: number;
  type_issues: number;
  long_files_count: number;
}

export interface TidinessScenarioFileInfo {
  path: string;
  lines: number;
  totalIssues: number;
  visitCount: number;
}

export interface TidinessScenarioDetail {
  scenario: string;
  lightIssues: number;
  aiIssues: number;
  longFiles: number;
  files: TidinessScenarioFileInfo[];
}

// ============================================================================
// Tidiness Manager API Functions
// ============================================================================

export async function fetchTidinessScore(
  scenarioName: string,
  repoId?: string
): Promise<TidinessScoreResponse> {
  const params = new URLSearchParams({ scenarioName });
  const url = buildApiUrl(`/repo/tidiness-score?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessScoreResponse>(res);
}

export async function fetchTidinessIssues(
  scenarioName: string,
  file?: string,
  category?: string,
  limit?: number,
  repoId?: string
): Promise<TidinessIssue[]> {
  const params = new URLSearchParams({ scenarioName });
  if (file) params.set("file", file);
  if (category) params.set("category", category);
  if (limit) params.set("limit", String(limit));
  const url = buildApiUrl(`/repo/tidiness-issues?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessIssue[]>(res);
}

export async function fetchTidinessStaleness(
  scenarioName: string,
  repoId?: string
): Promise<TidinessStalenessInfo> {
  const params = new URLSearchParams({ scenarioName });
  const url = buildApiUrl(`/repo/tidiness-staleness?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessStalenessInfo>(res);
}

export async function triggerTidinessLightScan(
  request: TidinessLightScanRequest,
  repoId?: string
): Promise<TidinessLightScanResult> {
  const url = buildApiUrl("/repo/tidiness-scan", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(request),
  });
  return handleResponse<TidinessLightScanResult>(res);
}

export async function fetchTidinessScenarioDetail(
  scenarioName: string,
  repoId?: string
): Promise<TidinessScenarioDetail> {
  const params = new URLSearchParams({ scenarioName });
  const url = buildApiUrl(`/repo/tidiness-scenario?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<TidinessScenarioDetail>(res);
}

// ============================================================================
// Auditor Types
// ============================================================================

export interface AuditorViolation {
  id: string;
  scenario_name: string;
  type: string;
  severity: string;
  title: string;
  description: string;
  file_path: string;
  line_number: number;
  code_snippet?: string;
  recommendation: string;
  standard: string;
  discovered_at: string;
  source?: string;
  metadata?: Record<string, unknown>;
}

export interface AuditorCheckResult {
  check_id: string;
  status: string;
  scan_type: string;
  started_at: string;
  completed_at: string;
  duration_seconds: number;
  files_scanned: number;
  violations: AuditorViolation[];
  statistics: Record<string, number>;
  message: string;
  scenario_name?: string;
  summary?: AuditorViolationSummary;
}

export interface AuditorViolationSummary {
  total: number;
  by_severity: Record<string, number>;
  by_rule?: { rule_id: string; count: number }[];
  highest_severity: string;
  top_violations?: { id: string; severity: string; rule_id: string; title: string; file_path: string }[];
  recommended_steps?: string[];
  generated_at: string;
}

export interface AuditorJobStatus {
  id: string;
  scenario: string;
  scan_type: string;
  status: string;
  started_at: string;
  completed_at?: string;
  elapsed_seconds: number;
  total_scenarios: number;
  processed_scenarios: number;
  processed_files: number;
  total_files: number;
  current_scenario?: string;
  current_file?: string;
  message?: string;
  error?: string;
  result?: AuditorCheckResult;
}

export interface AuditorCheckJobResponse {
  job_id: string;
  status: AuditorJobStatus;
}

export interface AuditorRule {
  id: string;
  name: string;
  description: string;
  category: string;
  severity: string;
  enabled: boolean;
  standard: string;
  targets: string[];
}

export interface AuditorRulesListResponse {
  rules: Record<string, AuditorRule>;
  categories?: Record<string, unknown>;
  count: number;
  total: number;
}

export interface AuditorFixRequest {
  scenario_names: string[];
  rule_ids: string[];
  dry_run?: boolean;
}

export interface AuditorFixChange {
  type: string;
  detail: string;
}

export interface AuditorFixResult {
  scenario_name: string;
  rule_id: string;
  fixed: boolean;
  file_path: string;
  changes: AuditorFixChange[];
  error?: string;
}

export interface AuditorFixResponse {
  results: AuditorFixResult[];
  count: number;
  unfixable_rules: string[];
  errors: string[];
}

// ============================================================================
// Auditor API
// ============================================================================

export async function startAuditorCheck(scenarioName: string, checkType = "full", repoId?: string): Promise<AuditorCheckJobResponse> {
  const res = await fetch(buildApiUrl("/repo/rules-run", { baseUrl: API_BASE }), {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify({ scenario_name: scenarioName, check_type: checkType }),
  });
  if (!res.ok) throw new Error(`Start check failed: ${res.statusText}`);
  return res.json();
}

export async function pollAuditorJob(jobId: string, repoId?: string): Promise<AuditorJobStatus> {
  const res = await fetch(buildApiUrl(`/repo/rules-job/${encodeURIComponent(jobId)}`, { baseUrl: API_BASE }), {
    headers: buildRepoHeaders(repoId),
  });
  if (!res.ok) throw new Error(`Poll job failed: ${res.statusText}`);
  return res.json();
}

export async function fetchAuditorRules(repoId?: string): Promise<AuditorRulesListResponse> {
  const res = await fetch(buildApiUrl("/repo/rules", { baseUrl: API_BASE }), {
    headers: buildRepoHeaders(repoId),
  });
  if (!res.ok) throw new Error(`Fetch rules failed: ${res.statusText}`);
  return res.json();
}

export async function applyAuditorFix(req: AuditorFixRequest, repoId?: string): Promise<AuditorFixResponse> {
  const res = await fetch(buildApiUrl("/repo/rules-fix", { baseUrl: API_BASE }), {
    method: "POST",
    headers: buildRepoHeaders(repoId),
    body: JSON.stringify(req),
  });
  if (!res.ok) throw new Error(`Apply fix failed: ${res.statusText}`);
  return res.json();
}

export async function fetchAuditorViolations(scenarioName: string, repoId?: string): Promise<AuditorViolation[]> {
  const params = new URLSearchParams({ scenarioName });
  const res = await fetch(buildApiUrl(`/repo/rules-violations?${params}`, { baseUrl: API_BASE }), {
    headers: buildRepoHeaders(repoId),
  });
  if (!res.ok) throw new Error(`Fetch violations failed: ${res.statusText}`);
  const data = await res.json();
  return data.violations ?? data;
}

// ── Agent Manager types ──────────────────────────────────────────────

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

export type AgentContextKind = "test-failure" | "code-quality-issue" | "screenshot" | "change-summary" | "scenario-quality" | "rule-violation" | "rules-summary";

export interface AgentContextItem {
  kind: AgentContextKind;
  id: string;
  label: string;
  markdown: string;
  /** Absolute filesystem paths for screenshot images (resolved at send time). */
  screenshotPaths?: string[];
}

// ── Agent Manager fetch functions ────────────────────────────────────

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

// ============================================================================
// Scenario Listing
// ============================================================================

export interface ScenarioInfo {
  name: string;
  display_name: string;
  description: string;
  status: "running" | "stopped";
  health_status: string | null;
  tags: string[];
  runtime: string;
}

export async function fetchScenarios(): Promise<ScenarioInfo[]> {
  const url = buildApiUrl("/scenarios", { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" } });
  return handleResponse<ScenarioInfo[]>(res);
}

// ============================================================================
// Scenario Envelope — enriched metadata for agent orientation
// ============================================================================

/** Enriched scenario metadata derived from service.json, used to build the agent envelope. */
export interface ScenarioEnvelopeData {
  name: string;
  displayName: string;
  description: string;
  path: string;
  tags: string[];
  dependencies: {
    scenarios: Record<string, string>;
    resources: Record<string, string>;
  };
  lifecycle: {
    testCommand?: string;
    buildCommand?: string;
  };
}

/** Fetch enriched scenario metadata for the agent envelope. */
export async function fetchScenarioEnvelope(slug: string): Promise<ScenarioEnvelopeData> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(slug)}/envelope`, { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" } });
  return handleResponse<ScenarioEnvelopeData>(res);
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
