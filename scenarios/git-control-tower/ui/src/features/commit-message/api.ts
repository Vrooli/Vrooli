import { advisoryClient } from "../../lib/connect";
export type {
  StagedMessageCollection,
  StagedMessageJob,
  StagedMessageSuggestion,
  StagedMessageContext,
} from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";

export function collectMessage(repositoryId: string, signal: AbortSignal) {
  return advisoryClient.collectStagedMessage({ repositoryId }, { signal });
}

export function checkMessage(repositoryId: string, snapshotDigest: string) {
  return advisoryClient.checkStagedMessage({ repositoryId, snapshotDigest });
}

export function startMessage(input: {
  repositoryId: string;
  collectionId: string;
  snapshotDigest: string;
  selectedContextIds: string[];
  requestKey: string;
  operatorNote: string;
  continuationOfJobId?: string;
}) {
  return advisoryClient.startStagedMessage(input);
}

export function getMessageJob(repositoryId: string, jobId: string, requestKey = "") {
  return advisoryClient.getStagedMessageJob({ repositoryId, jobId, requestKey });
}

export function cancelMessageJob(repositoryId: string, jobId: string, requestKey = "") {
  return advisoryClient.cancelStagedMessageJob({ repositoryId, jobId, requestKey });
}

export function waitMessageJob(repositoryId: string, jobId: string, requestKey: string, signal: AbortSignal) {
  return advisoryClient.waitStagedMessageJob({ repositoryId, jobId, requestKey, timeoutSeconds: 30 }, { signal });
}
