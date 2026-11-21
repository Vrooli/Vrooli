import { createClient } from "@connectrpc/connect";
import {
  DisplayDevice,
  StateStatus,
  WorkspaceProvenance,
  WorkspaceService,
  type ListWorkspacesResponse,
  type Workspace,
  type WorkspaceDisplay,
  type WorkspacePage,
  type WorkspaceState,
  type WorkspaceSummary,
} from "@vrooli/proto-types/react-component-library/v1/workspace/workspace_pb";

import { API_BASE, AUTHORING_TIMEOUT_MS, createBoundedFetch, transport } from "./client";

export const workspacesClient = createClient(WorkspaceService, transport);

export { DisplayDevice, StateStatus, WorkspaceProvenance };
export type {
  ListWorkspacesResponse,
  Workspace,
  WorkspaceDisplay,
  WorkspacePage,
  WorkspaceState,
  WorkspaceSummary,
};

export async function listWorkspaces(input: {
  provenance?: WorkspaceProvenance;
  query?: string;
  limit?: number;
} = {}): Promise<WorkspaceSummary[]> {
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

/* ── Reference assets ─────────────────────────────────────────────────────
 *
 * Uploads are the one workspace concept that does not travel over Connect: the
 * bytes are rendered by the browser and uploaded as a data URL. The server
 * answers failures in the same JSON envelope, so `requestAsset` below reads it
 * exactly like the Connect client reads `ApiError`.
 */

export type WorkspaceAssetFidelity = "reference" | "captured";

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
  /** Server path for the bytes; pass through `workspaceAssetUrl`. */
  url: string;
}

/** The server rejects anything larger; checking here turns a slow upload into
 * an immediate, actionable message. */
export const MAX_WORKSPACE_ASSET_BYTES = 8 * 1024 * 1024;

const assetFetch = createBoundedFetch(AUTHORING_TIMEOUT_MS);

function assetEndpoint(path: string): string {
  return `${API_BASE.replace(/\/$/, "")}${path}`;
}

export function workspaceAssetUrl(asset: { url: string }): string {
  if (/^https?:\/\//i.test(asset.url)) return asset.url;
  return assetEndpoint(asset.url);
}

async function requestAsset(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("accept", "application/json");
  if (init.body !== undefined) headers.set("content-type", "application/json");
  try {
    return await assetFetch(assetEndpoint(path), { ...init, headers });
  } catch (cause) {
    throw new Error(cause instanceof Error ? cause.message : "The asset API is unreachable.");
  }
}

async function assetFailure(response: Response): Promise<Error> {
  try {
    const payload = (await response.json()) as { error?: { message?: string } };
    if (payload.error?.message) return new Error(payload.error.message);
  } catch {
    /* fall through to the status message */
  }
  return new Error(`The asset API responded ${response.status}.`);
}

async function assetJson<T>(response: Response): Promise<T> {
  if (!response.ok) throw await assetFailure(response);
  return (await response.json()) as T;
}

export async function listWorkspaceAssets(workspaceId: string): Promise<WorkspaceAsset[]> {
  const response = await requestAsset(`/api/workspaces/${encodeURIComponent(workspaceId)}/assets`);
  const payload = await assetJson<{ assets?: WorkspaceAsset[] }>(response);
  return payload.assets ?? [];
}

export async function listAllWorkspaceAssets(): Promise<WorkspaceAsset[]> {
  const response = await requestAsset("/api/workspace-assets");
  const payload = await assetJson<{ assets?: WorkspaceAsset[] }>(response);
  return payload.assets ?? [];
}

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result !== "string") {
        reject(new Error("The selected file could not be read."));
        return;
      }
      const comma = reader.result.indexOf(",");
      resolve(comma >= 0 ? reader.result.slice(comma + 1) : reader.result);
    };
    reader.onerror = () => reject(reader.error ?? new Error("The selected file could not be read."));
    reader.readAsDataURL(file);
  });
}

export async function uploadWorkspaceAsset(
  workspaceId: string,
  file: File,
  fidelity: WorkspaceAssetFidelity = "reference",
): Promise<WorkspaceAsset> {
  const dataBase64 = await readFileAsBase64(file);
  const response = await requestAsset(
    `/api/workspaces/${encodeURIComponent(workspaceId)}/assets`,
    {
      method: "POST",
      body: JSON.stringify({ name: file.name, mime: file.type, dataBase64, fidelity }),
    },
  );
  const payload = await assetJson<{ asset?: WorkspaceAsset }>(response);
  if (!payload.asset) throw new Error("The asset API returned no asset.");
  return payload.asset;
}

export async function deleteWorkspaceAsset(workspaceId: string, assetId: string): Promise<void> {
  const response = await requestAsset(
    `/api/workspaces/${encodeURIComponent(workspaceId)}/assets/${encodeURIComponent(assetId)}`,
    { method: "DELETE" },
  );
  // A repeat delete is success: the desired state is "absent".
  if (response.ok || response.status === 404) return;
  throw await assetFailure(response);
}
