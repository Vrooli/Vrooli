// Collection reads and diffs remain typed Connect calls. Collection creation is
// owned by Plan Manager's execution flow, so this UI is an operator inspector
// for the durable identity rather than a second policy-authoring surface.

import { baselinesClient } from "./connect";
import type {
  BaselineCollection,
  GetCollectionDiffStatusResponse,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

function toRepoId(repoId?: string | null): bigint {
  if (!repoId) return 0n;
  try {
    return BigInt(repoId);
  } catch {
    return 0n;
  }
}

export async function getBaselineCollection(
  name: string,
  branch: string,
  repoId?: string | null,
): Promise<BaselineCollection | undefined> {
  const result = await baselinesClient.getCollectionStatus({ name, branch, repoId: toRepoId(repoId) });
  return result.collection;
}

export async function diffBaselineCollection(
  name: string,
  branch: string,
  repoId?: string | null,
): Promise<GetCollectionDiffStatusResponse> {
  const operationId = globalThis.crypto.randomUUID();
  await baselinesClient.startCollectionDiff({ name, branch, repoId: toRepoId(repoId), scenarios: [], operationId });
  return baselinesClient.getCollectionDiffStatus({ name, branch, repoId: toRepoId(repoId), operationId });
}
