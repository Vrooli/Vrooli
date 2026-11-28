// Collection reads and diffs remain typed Connect calls. Collection creation is
// owned by Plan Manager's execution flow, so this UI is an operator inspector
// for the durable identity rather than a second policy-authoring surface.

import { baselinesClient } from "./connect";
import { toProtoRepoId } from "./api-baselines";
import type {
  BaselineCollection,
  GetCollectionDiffStatusResponse,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export async function getBaselineCollection(
  name: string,
  branch: string,
  repoId?: string | null,
): Promise<BaselineCollection | undefined> {
  const result = await baselinesClient.getCollectionStatus({ name, branch, repoId: toProtoRepoId(repoId) });
  return result.collection;
}

export async function diffBaselineCollection(
  name: string,
  branch: string,
  repoId?: string | null,
): Promise<GetCollectionDiffStatusResponse> {
  const operationId = globalThis.crypto.randomUUID();
  await baselinesClient.startCollectionDiff({ name, branch, repoId: toProtoRepoId(repoId), scenarios: [], operationId });
  return baselinesClient.getCollectionDiffStatus({ name, branch, repoId: toProtoRepoId(repoId), operationId });
}
