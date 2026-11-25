import { advisoryClient } from "../../lib/connect";
import { QueuedWorktreeSelectionState, StagedMessageMode, WorktreeSelectionSource, WorktreeSuggestionQueueDisposition, WorktreeSuggestionQueueState } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
export type {
  StagedMessageCollection,
  StagedMessageJob,
  StagedMessageSuggestion,
  StagedMessageContext,
  StagedMessageProviderReadiness,
  StagedMessageContinuation,
  WorktreeSuggestionQueue,
  WorktreeSuggestionQueueEntry,
  QueuedWorktreeSelection,
} from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
export { QueuedWorktreeSelectionState, StagedMessageMode, WorktreeSelectionSource, WorktreeSuggestionQueueDisposition, WorktreeSuggestionQueueState };

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
  mode?: StagedMessageMode;
}) {
  return advisoryClient.startStagedMessage({ ...input, mode: input.mode ?? StagedMessageMode.QUICK });
}

export function retryMessage(input: { repositoryId: string; jobId: string; snapshotDigest: string; requestKey: string; mode: StagedMessageMode }) {
  return advisoryClient.retryStagedMessage(input);
}

export function getMessageProviderReadiness(signal?: AbortSignal) {
  return advisoryClient.getStagedMessageProviderReadiness({}, { signal });
}

export function startWorktreeMessage(input: {
  repositoryId: string;
  paths: string[];
  requestKey: string;
  operatorNote?: string;
  mode: StagedMessageMode;
  replaceQueueEntryId?: string;
  expectedQueueRevision?: bigint;
}) {
  return advisoryClient.startWorktreeStagedMessage({
    repositoryId: input.repositoryId,
    paths: input.paths,
    requestKey: input.requestKey,
    operatorNote: input.operatorNote ?? "",
    mode: input.mode,
    replaceQueueEntryId: input.replaceQueueEntryId ?? "",
    expectedQueueRevision: input.expectedQueueRevision ?? 0n,
  });
}

export function queueWorktreeSelection(input: {
  repositoryId: string;
  selectionId: string;
  source: WorktreeSelectionSource;
  paths: string[];
}) {
  return advisoryClient.queueWorktreeSelection({
    repositoryId: input.repositoryId,
    selectionId: input.selectionId,
    source: input.source,
    paths: input.paths,
  });
}

export function extendQueuedSelection(input: {
  repositoryId: string;
  selectionId: string;
  expectedSnapshotDigest: string;
  paths: string[];
}) {
  return advisoryClient.extendQueuedSelection({
    repositoryId: input.repositoryId,
    selectionId: input.selectionId,
    expectedSnapshotDigest: input.expectedSnapshotDigest,
    paths: input.paths,
  });
}

export function finalizeQueuedSelection(input: { repositoryId: string; selectionId: string; expectedSnapshotDigest: string }) {  return advisoryClient.finalizeQueuedSelection({
    repositoryId: input.repositoryId,
    selectionId: input.selectionId,
    expectedSnapshotDigest: input.expectedSnapshotDigest,
  });
}

export function startQueuedSelection(input: {
  repositoryId: string;
  selectionId: string;
  snapshotDigest: string;
  requestKey: string;
  mode: StagedMessageMode;
  operatorNote?: string;
}) {
  return advisoryClient.startQueuedSelection({
    repositoryId: input.repositoryId,
    selectionId: input.selectionId,
    snapshotDigest: input.snapshotDigest,
    requestKey: input.requestKey,
    mode: input.mode,
    operatorNote: input.operatorNote ?? "",
  });
}

export function listWorktreeSuggestionQueue(repositoryId: string) {
  return advisoryClient.listWorktreeSuggestionQueue({ repositoryId });
}

export function updateWorktreeSuggestionQueueEntry(input: {
  repositoryId: string;
  entryId: string;
  expectedHeadEntryId?: string;
  expectedRevision: bigint;
  state: WorktreeSuggestionQueueState;
  failureReason?: string;
}) {
  return advisoryClient.updateWorktreeSuggestionQueueEntry({
    repositoryId: input.repositoryId,
    entryId: input.entryId,
    expectedHeadEntryId: input.expectedHeadEntryId ?? "",
    expectedRevision: input.expectedRevision,
    state: input.state,
    failureReason: input.failureReason ?? "",
  });
}

export function retireWorktreeSuggestionQueueHead(input: { repositoryId: string; expectedHeadEntryId: string; expectedRevision: bigint }) {
  return advisoryClient.retireWorktreeSuggestionQueueHead({
    repositoryId: input.repositoryId,
    expectedHeadEntryId: input.expectedHeadEntryId,
    expectedRevision: input.expectedRevision,
    disposition: WorktreeSuggestionQueueDisposition.COMMITTED,
  });
}

export function removeWorktreeSuggestionQueueEntry(input: { repositoryId: string; entryId: string; expectedRevision: bigint }) {
  return advisoryClient.removeWorktreeSuggestionQueueEntry(input);
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
