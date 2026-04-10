// ============================================================================
// Core Repo & Branch Type Definitions
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

export interface RepoAuthorStatus {
  name?: string;
  email?: string;
}

export interface RepoStatus {
  repo_dir: string;
  branch: RepoBranchStatus;
  files: RepoFilesStatus;
  file_stats?: RepoFileStats;
  file_hotspots?: Record<string, number>;
  scopes?: Record<string, string[]>;
  summary: RepoStatusSummary;
  author: RepoAuthorStatus;
  timestamp: string;
  stash_count?: number;
  submodules?: string[];
}

export interface RepoHistoryResponse {
  repo_dir: string;
  lines: string[];
  entries: RepoHistoryEntry[];
  limit: number;
  grep_pattern?: string;
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
  header: string;
  old_start: number;
  old_lines: number;
  new_start: number;
  new_lines: number;
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
