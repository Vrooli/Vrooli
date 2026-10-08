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
  DiffResult,
  SnapshotForBaselineResponse,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

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
  repoId?: string | null;
}

// diffBaseline starts a durable diff then resolves its verdict with a
// server-side wait (no client polling). The diff verdict is computed and cached
// server-side; for the UI's one-shot query this reads as a single async call.
export async function diffBaseline(params: DiffBaselineParams): Promise<DiffResult | undefined> {
  const started = await baselinesClient.startDiff({
    scenario: params.scenario,
    name: params.name,
    branch: params.branch ?? "",
    repoId: toRepoId(params.repoId),
  });
  const res = await baselinesClient.getDiffResult({
    scenario: params.scenario,
    name: params.name,
    branch: params.branch ?? "",
    runId: started.runId,
    repoId: toRepoId(params.repoId),
    wait: true,
  });
  return res.diff;
}

export interface SnapshotBaselineParams {
  scenario: string;
  name: string;
  branch?: string;
  reason?: string;
  createdBy?: string;
  repoId?: string | null;
}

// snapshotForBaseline always captures one comprehensive baseline-profile run.
export async function snapshotForBaseline(
  params: SnapshotBaselineParams,
): Promise<SnapshotForBaselineResponse> {
  return baselinesClient.snapshotForBaseline({
    scenario: params.scenario,
    name: params.name,
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
