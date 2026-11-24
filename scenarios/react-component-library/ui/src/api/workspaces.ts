import { Code, ConnectError, createClient } from "@connectrpc/connect";
import {
  DisplayDevice,
  StateStatus,
  WorkspaceProvenance,
  WorkspaceService,
  WorkspaceDisplaySchema,
  WorkspacePageSchema,
  WorkspaceStateSchema,
  type ListWorkspacesResponse,
  type Workspace,
  type WorkspaceDisplay,
  type WorkspacePage,
  type WorkspaceState,
  type WorkspaceSummary,
  type WorkspaceAsset as ProtoWorkspaceAsset,
  type WorkspaceSourceBaseline as ProtoWorkspaceSourceBaseline,
} from "@vrooli/proto-types/react-component-library/v1/workspace/workspace_pb";

import { API_BASE, authoringTransport, transport } from "./client";
import { WorkspaceVisualComparisonService, type WorkspaceSourceVisualComparison as ProtoWorkspaceSourceVisualComparison } from "@vrooli/proto-types/react-component-library/v1/workspacevisualcomparison/workspace_visual_comparison_pb";

export const workspacesClient = createClient(WorkspaceService, transport);
const workspaceAuthoringClient = createClient(WorkspaceService, authoringTransport);
export const workspaceVisualComparisonClient = createClient(
  WorkspaceVisualComparisonService,
  authoringTransport,
);

export {
  DisplayDevice,
  StateStatus,
  WorkspaceProvenance,
  WorkspaceDisplaySchema,
  WorkspacePageSchema,
  WorkspaceStateSchema,
};
export type {
  ListWorkspacesResponse,
  Workspace,
  WorkspaceDisplay,
  WorkspacePage,
  WorkspaceState,
  WorkspaceSummary,
};

export async function listWorkspaces(
  input: {
    provenance?: WorkspaceProvenance;
    query?: string;
    limit?: number;
  } = {},
): Promise<WorkspaceSummary[]> {
  const response = await workspacesClient.listWorkspaces(input);
  return response.workspaces;
}

export async function getWorkspace(id: string): Promise<Workspace | null> {
  const response = await workspacesClient.getWorkspace({ id });
  return response.workspace ?? null;
}

export async function createWorkspace(input: {
  name: string;
  summary?: string;
  provenance?: WorkspaceProvenance;
  tags?: string[];
  bindingScenario?: string;
  bindingPage?: string;
  pages?: WorkspacePage[];
  themeJson?: string;
  optionsJson?: string;
}): Promise<Workspace> {
  const response = await workspacesClient.createWorkspace({
    name: input.name,
    summary: input.summary ?? "",
    provenance: input.provenance ?? WorkspaceProvenance.AUTHORED,
    tags: input.tags ?? [],
    bindingScenario: input.bindingScenario ?? "",
    bindingPage: input.bindingPage ?? "",
    pages: input.pages ?? [],
    themeJson: input.themeJson ?? "{}",
    optionsJson: input.optionsJson ?? "{}",
  });
  if (!response.workspace) throw new Error("workspace was not returned");
  return response.workspace;
}

export async function saveWorkspace(
  workspace: Workspace,
  baseRevision: number,
  actor = "local",
): Promise<Workspace> {
  const response = await workspacesClient.saveWorkspace({ workspace, baseRevision, actor });
  if (!response.workspace) throw new Error("workspace was not returned");
  return response.workspace;
}

export async function deleteWorkspace(id: string): Promise<boolean> {
  const response = await workspacesClient.deleteWorkspace({ id });
  return response.deleted;
}

export async function forkWorkspace(id: string, name?: string): Promise<Workspace> {
  const response = await workspacesClient.forkWorkspace({ id, name: name ?? "" });
  if (!response.workspace) throw new Error("workspace was not returned");
  return response.workspace;
}

/* ── Reference assets ───────────────────────────────────────────────────── */

export type WorkspaceAssetFidelity = "reference" | "captured";

/** UI byte counts use numbers for display and formatting; Connect models int64 as bigint. */
export interface WorkspaceAsset {
  id: string;
  workspaceId: string;
  name: string;
  mime: string;
  bytes: number;
  width: number;
  height: number;
  fidelity: WorkspaceAssetFidelity;
  addedAt: string;
  url: string;
}

function toWorkspaceAsset(asset: ProtoWorkspaceAsset): WorkspaceAsset {
  return {
    ...asset,
    bytes: Number(asset.bytes),
    fidelity: asset.fidelity as WorkspaceAssetFidelity,
  };
}

/** The server rejects anything larger; checking here turns a slow upload into
 * an immediate, actionable message. */
export const MAX_WORKSPACE_ASSET_BYTES = 8 * 1024 * 1024;

function assetEndpoint(path: string): string {
  return `${API_BASE.replace(/\/$/, "")}${path}`;
}

export function workspaceAssetUrl(asset: { url: string }): string {
  if (/^https?:\/\//i.test(asset.url)) return asset.url;
  return assetEndpoint(asset.url);
}

export async function listWorkspaceAssets(workspaceId: string): Promise<WorkspaceAsset[]> {
  const response = await workspacesClient.listWorkspaceAssets({ workspaceId });
  return response.assets.map(toWorkspaceAsset);
}

export async function listAllWorkspaceAssets(): Promise<WorkspaceAsset[]> {
  const response = await workspacesClient.listAllWorkspaceAssets({});
  return response.assets.map(toWorkspaceAsset);
}

export interface WorkspaceStateCaptureEvidence {
  schemaVersion: "rcl-workspace-state-capture/v1";
  requestToken: string;
  attemptId: string;
  sourceRevision: number;
  pageId: string;
  displayId: string;
  stateId: string;
  executionId: string;
  capturedAt: string;
  screenshot: { reference: string; width: number; height: number; scope: string };
  accessibility: { reference: string; scope: string };
}

export interface WorkspaceStateCaptureResponse {
  workspaceRevision: number;
  captureId: string;
  evidence: WorkspaceStateCaptureEvidence;
}

export class WorkspaceCaptureError extends Error {
  constructor(
    message: string,
    readonly attemptState?: "failed" | "in_progress" | "completed",
  ) {
    super(message);
    this.name = "WorkspaceCaptureError";
  }
}

export async function captureWorkspaceState(
  workspaceId: string,
  input: {
    expectedRevision: number;
    pageId: string;
    displayId: string;
    stateId: string;
    attemptId: string;
  },
): Promise<WorkspaceStateCaptureResponse> {
  try {
    const response = await workspaceAuthoringClient.captureWorkspaceState({
      workspaceId,
      ...input,
    });
    if (!response.evidence) throw new Error("BAS capture returned no evidence.");
    return response as WorkspaceStateCaptureResponse;
  } catch (cause) {
    if (cause instanceof ConnectError) {
      const attemptState = cause.metadata.get("attempt-state");
      throw new WorkspaceCaptureError(
        cause.message,
        isAttemptState(attemptState) ? attemptState : undefined,
      );
    }
    throw cause;
  }
}

function isAttemptState(
  value: string | null,
): value is NonNullable<WorkspaceCaptureError["attemptState"]> {
  return value === "failed" || value === "in_progress" || value === "completed";
}

export async function getWorkspaceStateCapture(
  workspaceId: string,
  stateId: string,
): Promise<WorkspaceStateCaptureResponse> {
  const response = await workspacesClient.getWorkspaceStateCapture({ workspaceId, stateId });
  if (!response.evidence) throw new Error("BAS capture returned no evidence.");
  return response as WorkspaceStateCaptureResponse;
}

export interface WorkspaceCaptureReceipt {
  id: string;
  workspaceId: string;
  stateId: string;
  sourceRevision: number;
  captureId: string;
  attemptId: string;
  capturedAt: string;
  evidence: WorkspaceStateCaptureEvidence;
}

export interface WorkspaceVisualDiffResult {
  jobId: string;
  verdict: string;
  dimensionsMatch: boolean;
  width: number;
  height: number;
  changedPixels: string;
  totalPixels: string;
  changedFraction: number;
  mae: number;
  rmse: number;
  psnr: number;
  phashDistance: number;
  phashSimilarity: number;
  ssim: number;
  heatmapRef: string;
  warnings: string[];
}

export interface WorkspaceVisualComparison {
  comparisonId: string;
  workspaceId: string;
  stateId: string;
  baselineReceiptId: string;
  currentReceiptId: string;
  createdAt: string;
  result: WorkspaceVisualDiffResult;
}

export type WorkspaceSourceBaseline = ProtoWorkspaceSourceBaseline;
export interface WorkspaceSourceBaselineSet {
  candidates: WorkspaceSourceBaseline[];
  accepted: WorkspaceSourceBaseline | null;
}
export interface WorkspaceSourceVisualComparison extends Omit<ProtoWorkspaceSourceVisualComparison, "resultJson"> {
  result: WorkspaceVisualDiffResult;
}

export async function listWorkspaceSourceBaselines(workspaceId: string, stateId: string): Promise<WorkspaceSourceBaselineSet> {
  const response = await workspacesClient.listWorkspaceSourceBaselines({ workspaceId, stateId });
  return { candidates: response.candidates, accepted: response.accepted ?? null };
}

export async function createWorkspaceSourceBaseline(workspaceId: string, stateId: string, assetId: string, sourceLocator = ""): Promise<WorkspaceSourceBaseline> {
  const response = await workspacesClient.createWorkspaceSourceBaseline({ workspaceId, stateId, assetId, sourceLocator });
  return response;
}

export async function acceptWorkspaceSourceBaseline(workspaceId: string, stateId: string, baselineId: string, acceptedBy: string): Promise<WorkspaceSourceBaseline> {
  const response = await workspacesClient.acceptWorkspaceSourceBaseline({ workspaceId, stateId, baselineId, acceptedBy });
  return response;
}

export async function listWorkspaceSourceBaselineComparisons(workspaceId: string, stateId: string): Promise<WorkspaceSourceVisualComparison[]> {
  const response = await workspaceVisualComparisonClient.listWorkspaceSourceBaselineComparisons({ workspaceId, stateId });
  return response.comparisons.map(toWorkspaceSourceVisualComparison);
}

export async function compareWorkspaceSourceBaseline(workspaceId: string, stateId: string, currentReceiptId: string): Promise<WorkspaceSourceVisualComparison> {
  return toWorkspaceSourceVisualComparison(await workspaceVisualComparisonClient.compareWorkspaceSourceBaseline({ workspaceId, stateId, currentReceiptId }));
}

function toWorkspaceSourceVisualComparison(comparison: ProtoWorkspaceSourceVisualComparison): WorkspaceSourceVisualComparison {
  let result: WorkspaceVisualDiffResult;
  try { result = JSON.parse(comparison.resultJson) as WorkspaceVisualDiffResult; }
  catch { throw new Error("The source visual comparison returned invalid result data."); }
  return { ...comparison, result };
}

function workspaceStateEndpoint(workspaceId: string, stateId: string): string {
  return `/api/workspaces/${encodeURIComponent(workspaceId)}/states/${encodeURIComponent(stateId)}`;
}

export async function listWorkspaceCaptureReceipts(
  workspaceId: string,
  stateId: string,
): Promise<WorkspaceCaptureReceipt[]> {
  const response = await workspacesClient.listWorkspaceCaptureReceipts({ workspaceId, stateId });
  return response.receipts as WorkspaceCaptureReceipt[];
}

export function workspaceCaptureScreenshotUrl(
  workspaceId: string,
  stateId: string,
  receiptId: string,
): string {
  return assetEndpoint(
    `${workspaceStateEndpoint(workspaceId, stateId)}/capture-receipts/${encodeURIComponent(receiptId)}/screenshot`,
  );
}

export async function listWorkspaceVisualComparisons(
  workspaceId: string,
  stateId: string,
): Promise<WorkspaceVisualComparison[]> {
  const response = await workspaceVisualComparisonClient.listWorkspaceVisualComparisons({
    workspaceId,
    stateId,
  });
  return response.comparisons.map(toWorkspaceVisualComparison);
}

export async function compareWorkspaceCaptureReceipts(
  workspaceId: string,
  stateId: string,
  baselineReceiptId: string,
  currentReceiptId: string,
): Promise<WorkspaceVisualComparison> {
  const comparison = await workspaceVisualComparisonClient.compareWorkspaceCaptureReceipts({
    workspaceId,
    stateId,
    baselineReceiptId,
    currentReceiptId,
  });
  return toWorkspaceVisualComparison(comparison);
}

function toWorkspaceVisualComparison(comparison: {
  comparisonId: string;
  workspaceId: string;
  stateId: string;
  baselineReceiptId: string;
  currentReceiptId: string;
  createdAt: string;
  resultJson: string;
}): WorkspaceVisualComparison {
  let result: WorkspaceVisualDiffResult;
  try {
    result = JSON.parse(comparison.resultJson) as WorkspaceVisualDiffResult;
  } catch {
    throw new Error("The visual comparison returned invalid result data.");
  }
  return { ...comparison, result };
}

export function workspaceVisualHeatmapUrl(
  workspaceId: string,
  stateId: string,
  comparisonId: string,
): string {
  return assetEndpoint(
    `${workspaceStateEndpoint(workspaceId, stateId)}/visual-comparisons/${encodeURIComponent(comparisonId)}/heatmap`,
  );
}

/** Source-baseline comparisons share the same persisted heatmap retrieval route. */
export const workspaceSourceVisualHeatmapUrl = workspaceVisualHeatmapUrl;

export async function uploadWorkspaceAsset(
  workspaceId: string,
  file: File,
  fidelity: WorkspaceAssetFidelity = "reference",
): Promise<WorkspaceAsset> {
  if (file.size === 0 || file.size > MAX_WORKSPACE_ASSET_BYTES) {
    throw new Error(
      `Choose a non-empty image smaller than ${MAX_WORKSPACE_ASSET_BYTES / 1024 / 1024} MiB.`,
    );
  }
  const response = await workspacesClient.createWorkspaceAsset({
    workspaceId,
    name: file.name,
    mime: file.type,
    data: new Uint8Array(await file.arrayBuffer()),
    fidelity,
  });
  if (!response.asset) throw new Error("The asset API returned no asset.");
  return toWorkspaceAsset(response.asset);
}

export async function deleteWorkspaceAsset(workspaceId: string, assetId: string): Promise<void> {
  // A repeat delete is success: the desired state is "absent".
  try {
    await workspacesClient.deleteWorkspaceAsset({ workspaceId, assetId });
  } catch (error) {
    if (error instanceof ConnectError && error.code === Code.NotFound) return;
    throw error;
  }
}
