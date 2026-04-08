// ============================================================================
// Visual Capture & Workflow Capture — Types + API Functions
// ============================================================================

import { API_BASE, buildRepoHeaders, handleResponse, buildApiUrl } from "./api-internals";

// ── Visual Capture Types ───────────────────────────────────────────────

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

// ── Visual Capture API Functions ───────────────────────────────────────

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

// ── Workflow Capture Types ─────────────────────────────────────────────

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

// ── Workflow Capture API Functions ──────────────────────────────────────

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
