import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// --- Types matching the Go API ---

export type Status = "creating" | "active" | "stopped" | "approved" | "rejected" | "deleted" | "error";
export type OwnerType = "agent" | "user" | "task" | "system";
export type ChangeType = "added" | "modified" | "deleted";
export type ApprovalStatus = "pending" | "approved" | "rejected";

export interface Sandbox {
  id: string;
  scopePath: string;
  projectRoot: string;
  owner?: string;
  ownerType: OwnerType;
  status: Status;
  errorMessage?: string;
  createdAt: string;
  lastUsedAt: string;
  stoppedAt?: string;
  approvedAt?: string;
  deletedAt?: string;
  driver: string;
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
}

export interface DiffResult {
  sandboxId: string;
  files: FileChange[];
  unifiedDiff: string;
  generated: string;
  totalAdded: number;
  totalDeleted: number;
  totalModified: number;
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
    type: string;
    version: string;
    description: string;
    available: boolean;
  };
  available: boolean;
  error?: string;
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

export interface CreateRequest {
  scopePath: string;
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
}

export interface ListFilter {
  status?: Status[];
  owner?: string;
  projectRoot?: string;
  scopePath?: string;
  limit?: number;
  offset?: number;
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
  return apiRequest<DriverInfo>("/driver");
}

// List sandboxes
export async function listSandboxes(filter?: ListFilter): Promise<ListResult> {
  const params = new URLSearchParams();
  if (filter?.status) {
    filter.status.forEach((s) => params.append("status", s));
  }
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

// Get diff
export async function getDiff(id: string): Promise<DiffResult> {
  return apiRequest<DiffResult>(`/sandboxes/${id}/diff`);
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
  approved: number;
  rejected: number;
  error: number;
  totalSizeBytes: number;
}

export function computeStats(sandboxes: Sandbox[]): SandboxStats {
  return sandboxes.reduce(
    (acc, sb) => {
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
