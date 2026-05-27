// ============================================================================
// Baselines API (Plan B §4.2) — typed wrappers over BaselinesService
// ============================================================================
//
// Thin async functions over the generated Connect client. They exist so the
// React Query hooks (and tests) depend on a small, stable surface instead of
// the raw proto client, and so repoId string→bigint conversion happens in one
// place. No transformation logic beyond that — the proto messages are already
// the UI's data model.

import { baselinesClient } from "./connect";
import type {
  BaselineManifest,
  DiffBaselineResponse,
  SnapshotForBaselineResponse,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

// The full set of surfaces a baseline can pin, in display order.
export const BASELINE_SURFACES = ["workflows", "tests", "structure", "visuals", "rules"] as const;
export type BaselineSurface = (typeof BASELINE_SURFACES)[number];

// repoId is a numeric string in the UI (X-Repo-Id); the proto wants int64
// (bigint in connect-es). 0n means "active repo" server-side.
function toRepoId(repoId?: string | null): bigint {
  if (!repoId) return 0n;
  try {
    return BigInt(repoId);
  } catch {
    return 0n;
  }
}

export interface ListBaselinesParams {
  scenario: string;
  allBranches?: boolean;
  branch?: string;
  repoId?: string | null;
}

export async function listBaselines(params: ListBaselinesParams): Promise<BaselineManifest[]> {
  const res = await baselinesClient.listBaselines({
    scenario: params.scenario,
    branch: params.branch ?? "",
    allBranches: params.allBranches ?? false,
    repoId: toRepoId(params.repoId),
  });
  return res.baselines;
}

export interface GetBaselineParams {
  scenario: string;
  name: string;
  branch?: string;
  repoId?: string | null;
}

export async function getBaseline(params: GetBaselineParams): Promise<BaselineManifest | undefined> {
  const res = await baselinesClient.getBaseline({
    scenario: params.scenario,
    name: params.name,
    branch: params.branch ?? "",
    repoId: toRepoId(params.repoId),
  });
  return res.baseline;
}

export interface DiffBaselineParams {
  scenario: string;
  name: string;
  branch?: string;
  surface?: string;
  repoId?: string | null;
}

export async function diffBaseline(params: DiffBaselineParams): Promise<DiffBaselineResponse> {
  return baselinesClient.diffBaseline({
    scenario: params.scenario,
    name: params.name,
    branch: params.branch ?? "",
    surface: params.surface ?? "",
    repoId: toRepoId(params.repoId),
  });
}

export interface SnapshotBaselineParams {
  scenario: string;
  name: string;
  include: string[];
  fast: boolean;
  branch?: string;
  reason?: string;
  createdBy?: string;
  repoId?: string | null;
}

// snapshotForBaseline captures every requested surface and writes the manifest
// — the "Capture" action in SetBaselineModal. This is a long-running call
// (BAS workflows + test-genie + visuals); callers should expect minutes.
export async function snapshotForBaseline(
  params: SnapshotBaselineParams,
): Promise<SnapshotForBaselineResponse> {
  return baselinesClient.snapshotForBaseline({
    scenario: params.scenario,
    name: params.name,
    include: params.include,
    fast: params.fast,
    branch: params.branch ?? "",
    reason: params.reason ?? "",
    createdBy: params.createdBy ?? "ui",
    repoId: toRepoId(params.repoId),
  });
}

export interface DeleteBaselineParams {
  scenario: string;
  name: string;
  branch?: string;
  repoId?: string | null;
}

export async function deleteBaseline(params: DeleteBaselineParams): Promise<boolean> {
  const res = await baselinesClient.deleteBaseline({
    scenario: params.scenario,
    name: params.name,
    branch: params.branch ?? "",
    repoId: toRepoId(params.repoId),
  });
  return res.deleted;
}

export interface EditBaselineParams {
  scenario: string;
  name: string;
  surface: string;
  pinRunId: string;
  branch?: string;
  reason?: string;
  repoId?: string | null;
}

export async function editBaseline(params: EditBaselineParams): Promise<BaselineManifest | undefined> {
  const res = await baselinesClient.editBaseline({
    scenario: params.scenario,
    name: params.name,
    surface: params.surface,
    pinRunId: params.pinRunId,
    branch: params.branch ?? "",
    reason: params.reason ?? "",
    repoId: toRepoId(params.repoId),
  });
  return res.baseline;
}
