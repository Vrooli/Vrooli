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
  ignored?: string[];
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
  scopes: Record<string, string[]>;
  summary: RepoStatusSummary;
  timestamp: string;
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

export interface DiffResponse {
  repo_dir: string;
  path?: string;
  staged: boolean;
  base?: string;
  has_diff: boolean;
  hunks?: DiffHunk[];
  stats: DiffStats;
  raw?: string;
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

export async function fetchDiff(
  path?: string,
  staged = false
): Promise<DiffResponse> {
  const params = new URLSearchParams();
  if (path) params.set("path", path);
  if (staged) params.set("staged", "true");

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
