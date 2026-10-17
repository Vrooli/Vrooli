// ============================================================================
// Diff, Stage, Commit, File & Search Type Definitions
// ============================================================================

import type { DiffHunk, DiffStats } from "./api-types-repo";
import type {
  AuthorityStatus as ProtoAuthorityStatus,
  ConfirmMutationRequest,
  MutationIntent,
  MutationPreview,
  PrepareMutationRequest,
} from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import type { CreateCommitResponse as ProtoCreateCommitResponse } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";

// Human-control contracts are proto-owned. Keep these aliases only for the
// existing API barrel names while consumers migrate to generated fields and
// Connect-Web transport.
export type AuthorityStatus = ProtoAuthorityStatus;
export type MutationPreviewRequest = Pick<PrepareMutationRequest, "repositoryId" | "operation"> & { subjectContext?: string };
export type MutationPreviewResponse = MutationPreview;
export type MutationIntentRequest = Pick<ConfirmMutationRequest, "repositoryId" | "operation" | "expectedRevision" | "subjectDigest"> & { subjectContext?: string };
export type MutationIntentResponse = MutationIntent;

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
  intent_id?: string;
  validate_conventional?: boolean;
  amend?: boolean;
  author_name?: string;
  author_email?: string;
  skip_precommit_once?: boolean;
}

export interface PrecommitRunResult {
  status: string;
  command?: string;
  exit_code: number;
  summary: string;
  stdout?: string;
  stderr?: string;
  duration_ms: number;
  override_allowed: boolean;
  timestamp: string;
}

export type CommitResponse = ProtoCreateCommitResponse;

export interface PrecommitConfig {
  enabled: boolean;
  command: string;
  working_directory: string;
  timeout_seconds: number;
  run_before_commit: boolean;
  allow_override: boolean;
  last_result?: PrecommitRunResult;
  hook?: PrecommitHookState;
}

export interface PrecommitHookState {
  status: "installed" | "fallback" | "uninstalled" | string;
  reason?: string;
  existing_kind?: "user" | "framework" | "gct" | "none" | string;
  existing_hook_preview?: string;
  path?: string;
  hooks_path?: string;
  installed_at?: string;
}

export interface PrecommitRunRequest {
  command?: string;
  working_directory?: string;
  timeout_seconds?: number;
}

export interface PrecommitRunResponse {
  success: boolean;
  result: PrecommitRunResult;
}

export type PrecommitStreamEventType = "started" | "progress" | "finished" | "error";

export interface PrecommitStreamEvent {
  type: PrecommitStreamEventType;
  elapsed_ms: number;
  command?: string;
  tail?: string[];
  result?: PrecommitRunResult;
  error?: string;
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

export interface ChangeGroupAPI {
  key: string;
  kind?: string;
  id?: string;
  label: string;
  root?: string;
  source: "manual" | "contract" | "builtin" | string;
  files: string[];
}

export interface RepoGroupsResponse {
  groups: ChangeGroupAPI[];
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

// Tracked-binary health types
export interface TrackedBinariesResponse {
  binaries: TrackedBinary[];
  total_bytes: number;
  /** Stated plainly so the UI never implies untracking reclaims repo size. */
  history_warning?: string;
}

export interface TrackedBinary {
  path: string;
  bytes: number;
  format: "elf" | "mach-o" | "pe";
  /** Scenario/resource dir that should ignore this path; "" means repo root. */
  owner_dir: string;
  ignore_pattern: string;
  already_ignored: boolean;
}

export interface UntrackBinaryRequest {
  path: string;
  owner_dir: string;
  ignore_pattern: string;
}

export interface UntrackBinaryResponse {
  success: boolean;
  removed_from_index: boolean;
  ignore_added_to?: string;
  error?: string;
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
  visibility?: string;
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

export type ProvenanceStanding = "exact_content" | "commit_file" | "run_file" | "work_reference" | "asserted" | "stale" | "private" | "unavailable" | "unknown" | string;

export interface ProvenanceWorkReference {
  kind: string;
  id: string;
  revision?: string;
  relationship?: string;
  verified?: boolean;
  visibility?: string;
  state?: string;
  unavailableReason?: string;
}

export interface BlameLine {
  line: number;
  content: string;
  commit?: string;
  author?: string;
  authorTime?: string;
  subject?: string;
}

export interface BlameEvidence {
  runId?: string;
  sandboxId?: string;
  contentDigest?: string;
  commitId?: string;
  visibility?: string;
  commitState?: string;
  runOutcome?: string;
  conversationId?: string;
  costUsd?: number;
  committedAt?: string;
  unavailable?: string[];
  workReferences?: ProvenanceWorkReference[];
}

export interface BlameFile {
  path: string;
  status: string;
  contentDigest?: string;
  reason?: string;
  standing?: ProvenanceStanding;
  downgradeReasons?: string[];
  lines: BlameLine[];
  evidence: BlameEvidence[];
}

export interface ProvenanceChangeBundle {
  runId?: string;
  sandboxId?: string;
  files: string[];
  runOutcome?: string;
  conversationId?: string;
  costUsd?: number;
  workReferences: ProvenanceWorkReference[];
  gaps: string[];
}

export interface BlameResponse {
  revision: string;
  files: BlameFile[];
  truncated: boolean;
  warnings: string[];
  changeBundles: ProvenanceChangeBundle[];
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

/**
 * Raised when the server accepted the request but git reported failure, e.g. a push
 * that was rejected, timed out, or left the remote ref unmoved. The response is kept
 * so callers can name the target ref without re-deriving it.
 */
export class RemoteOperationError extends Error {
  readonly operation: "push" | "pull";
  readonly result: PushResponse | PullResponse;

  constructor(
    operation: "push" | "pull",
    message: string | undefined,
    result: PushResponse | PullResponse
  ) {
    super(message?.trim() || `git ${operation} failed without reporting a reason`);
    this.name = "RemoteOperationError";
    this.operation = operation;
    this.result = result;
  }
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
