// ============================================================================
// Core Git/Repo API Functions
// ============================================================================

import { API_BASE, buildRepoHeaders, handleResponse, buildApiUrl } from "./api-internals";
import type { HealthResponse, RepoStatus, RepoHistoryResponse, RepoBranchesResponse, BranchCreateResponse, BranchSwitchResponse, BranchPublishResponse, CreateBranchRequest, SwitchBranchRequest, PublishBranchRequest } from "./api-types-repo";
import {
  FileContentConflictError,
  type ViewMode,
  type DiffResponse,
  type StageRequest,
  type StageResponse,
  type UnstageRequest,
  type UnstageResponse,
  type CommitRequest,
  type CommitResponse,
  type DiscardRequest,
  type DiscardResponse,
  type IgnoreRequest,
  type IgnoreResponse,
  type PushRequest,
  type PushResponse,
  type PullRequest,
  type PullResponse,
  type UpstreamActionRequest,
  type UpstreamActionResponse,
  type SyncStatusResponse,
  type ApprovedChangesResponse,
  type ApprovedChangesPreviewRequest,
  type ProvenanceResponse,
  type FileTreeResponse,
  type RelatedFilesResponse,
  type ContentSearchRequest,
  type ContentSearchResponse,
  type DirListResponse,
  type DeletePathRequest,
  type DeletePathResponse,
  type SaveFileContentRequest,
  type SaveFileContentResponse,
  type SaveFileContentConflictResponse,
} from "./api-types-operations";

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  return handleResponse<HealthResponse>(res);
}

export async function fetchRepoStatus(repoId?: string, hotspots = false): Promise<RepoStatus> {
  const path = hotspots ? "/repo/status?hotspots=true" : "/repo/status";
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<RepoStatus>(res);
}

export async function fetchRepoHistory(
  limit = 30,
  includeFiles = false,
  repoId?: string,
  grep?: string
): Promise<RepoHistoryResponse> {
  const params = new URLSearchParams();
  if (limit > 0) params.set("limit", String(limit));
  if (includeFiles) params.set("include", "files");
  if (grep) params.set("grep", grep);

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

export async function fetchProvenance(repoId?: string): Promise<ProvenanceResponse> {
  const url = buildApiUrl("/repo/provenance", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store"
  });
  return handleResponse<ProvenanceResponse>(res);
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
