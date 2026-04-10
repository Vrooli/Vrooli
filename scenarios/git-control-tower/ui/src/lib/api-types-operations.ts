// ============================================================================
// Diff, Stage, Commit, File & Search Type Definitions
// ============================================================================

import type { DiffHunk, DiffStats } from "./api-types-repo";

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
  agentManagerRunId?: string;
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

// Provenance Types
export interface ProvenanceFile {
  filePath: string;
  relativePath: string;
  changeType: string;
  appliedAt: string;
}

export interface ProvenanceRunGroup {
  runId: string;
  sandboxId: string;
  sandboxOwner: string;
  files: ProvenanceFile[];
  latestAppliedAt: string;
}

export interface ProvenanceResponse {
  available: boolean;
  runGroups: ProvenanceRunGroup[];
  warning?: string;
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
