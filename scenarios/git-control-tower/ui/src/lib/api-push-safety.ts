import { create } from "@bufbuild/protobuf";
import { GetPushRecoveryRequestSchema, InspectPushSafetyRequestSchema, PreparePushRecoveryRequestSchema, type PushSafetyReport } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { repoClient } from "./connect";
import { issueMutationIntentForOperation } from "./api-core";

export async function inspectPushSafety(repoId?: string, signal?: AbortSignal) {
  return repoClient.inspectPushSafety(create(InspectPushSafetyRequestSchema, { repositoryId: repoId ?? "" }), ...(signal ? [{ signal }] : []));
}
export async function preparePushRecovery(report: PushSafetyReport, repoId?: string) {
  const intent = await issueMutationIntentForOperation("repo.recovery.prepare", repoId, report.fingerprint);
  return repoClient.preparePushRecovery(create(PreparePushRecoveryRequestSchema, {
    repositoryId: intent.repositoryId, intentId: intent.intentId,
    remote: report.remote, branch: report.branch, fingerprint: report.fingerprint,
  }), { timeoutMs: 10 * 60 * 1000 });
}

export async function getPushRecovery(fingerprint: string, repoId?: string) {
 return repoClient.getPushRecovery(create(GetPushRecoveryRequestSchema, { repositoryId: repoId ?? "", fingerprint }));
}
