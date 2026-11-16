import { StagedMessageMode, WorktreeSuggestionQueueState, type WorktreeSuggestionQueueEntry } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";

export interface QueuedStageSelection {
  id: string;
  snapshotDigest?: string;
  selectionId?: string;
  paths: string[];
  queuedAt: string;
  state: "queued" | "awaiting_unstage" | "staged" | "stage_failed" | "stale" | "commit_pending";
  suggestionMode?: "quick" | "deep";
  suggestionState?: "starting" | "running" | "succeeded" | "failed";
  requestKey?: string;
  jobId?: string;
  suggestionError?: string;
  stageError?: string;
}

export interface QueuedCommitTarget {
  entryId: string;
  snapshotDigest: string;
}

function queueViewState(state: WorktreeSuggestionQueueState): QueuedStageSelection["state"] {
  switch (state) {
    case WorktreeSuggestionQueueState.STAGED: return "staged";
    case WorktreeSuggestionQueueState.FAILED: return "stage_failed";
    case WorktreeSuggestionQueueState.STALE: return "stale";
    case WorktreeSuggestionQueueState.COMMIT_PENDING: return "commit_pending";
    case WorktreeSuggestionQueueState.AWAITING_UNSTAGE: return "awaiting_unstage";
    case WorktreeSuggestionQueueState.QUEUED:
    case WorktreeSuggestionQueueState.UNSPECIFIED:
    default: return "queued";
  }
}

/** The server queue is authoritative. This is only the view model used by the UI. */
export function toQueuedStageSelection(entry: WorktreeSuggestionQueueEntry): QueuedStageSelection {
  return {
    id: entry.id,
    snapshotDigest: entry.snapshotDigest,
    paths: [...entry.paths],
    queuedAt: entry.createdAt,
    state: queueViewState(entry.state),
    selectionId: entry.selectionId || entry.id,
    ...(entry.mode === StagedMessageMode.DEEP ? { suggestionMode: "deep" as const } : entry.mode === StagedMessageMode.QUICK ? { suggestionMode: "quick" as const } : {}),
    ...(entry.requestKey ? { requestKey: entry.requestKey } : {}),
    ...(entry.jobId ? { jobId: entry.jobId } : {}),
    ...(entry.failureReason ? { stageError: entry.failureReason } : {}),
  };
}

/** Keeps a just-succeeded commit observable until its asynchronously persisted
 * first follow-up selection becomes available. */
export class SuccessfulCommitQueueGate {
  private readonly awaitingSelection = new Set<string>();

  markSuccessful(repo: string) { this.awaitingSelection.add(repo); }

  canDispatch(repo: string, hasSelection: boolean) {
    return hasSelection && this.awaitingSelection.has(repo);
  }

  consume(repo: string) { this.awaitingSelection.delete(repo); }
}

export function appendStageSelection(
  queue: readonly QueuedStageSelection[],
  paths: readonly string[],
  id: string,
  queuedAt = new Date().toISOString(),
  suggestionMode?: "quick" | "deep",
) {
  const queuedPaths = new Set(queue.flatMap(item => item.paths));
  const uniquePaths = [...new Set(paths)].filter(path => path && !path.includes("\0") && !queuedPaths.has(path));
  if (uniquePaths.length === 0) return [...queue];
  return [...queue, { id, paths: uniquePaths, queuedAt, state: "queued" as const, ...(suggestionMode ? { suggestionMode, suggestionState: "starting" as const, requestKey: id } : {}) }];
}

export function updateStageSelection(queue: readonly QueuedStageSelection[], id: string, patch: Partial<QueuedStageSelection>) {
  return queue.map(item => {
    if (item.id !== id) return item;
    const updated = { ...item, ...patch, id: item.id, paths: item.paths, queuedAt: item.queuedAt };
    if (patch.state === "staged") delete updated.stageError;
    return updated;
  });
}

export function removeStageSelection(queue: readonly QueuedStageSelection[], id: string) {
  return queue.filter(item => item.id !== id);
}

export function isStageSelectionComplete(selection: QueuedStageSelection, response: { success: boolean; staged: readonly string[] }) {
  if (!response.success) return false;
  const staged = new Set(response.staged);
  return selection.paths.every(path => staged.has(path));
}

export function reconcileStageQueueWithIndex(queue: readonly QueuedStageSelection[], stagedPaths: readonly string[]) {
  const staged = new Set(stagedPaths);
  return queue.map(selection => {
    if (selection.state === "awaiting_unstage" || selection.state === "stale" || selection.state === "commit_pending") return selection;
    const fullyStaged = selection.paths.every(path => staged.has(path));
    if (fullyStaged && selection.state !== "staged") return { ...selection, state: "staged" as const, stageError: undefined };
    if (!fullyStaged && selection.state === "staged") return { ...selection, state: "queued" as const, stageError: "Some queued files are no longer staged. They remain in the follow-up queue." };
    return selection;
  });
}

export function queuedHeadIncludedInIndex(queue: readonly QueuedStageSelection[], stagedPaths: readonly string[] | undefined): QueuedCommitTarget | null {
  if (!stagedPaths) return null;
  const head = queue[0];
  return head && head.snapshotDigest && head.paths.every(path => stagedPaths.includes(path))
    ? { entryId: head.id, snapshotDigest: head.snapshotDigest }
    : null;
}

/** An incomplete staged-to-queue handoff must never enter the current commit. */
export function queuedSelectionAwaitingUnstage(queue: readonly QueuedStageSelection[], stagedPaths: readonly string[] | undefined) {
  if (!stagedPaths?.length) return undefined;
  const staged = new Set(stagedPaths);
  return queue.find(selection => selection.state === "awaiting_unstage" && selection.paths.some(path => staged.has(path)));
}

export function retireCommittedQueueHead(queue: readonly QueuedStageSelection[], committedSelectionId: string | null) {
  return committedSelectionId && queue[0]?.id === committedSelectionId ? queue.slice(1) : [...queue];
}

export type StageQueueLock = <Result>(name: string, update: () => Result | Promise<Result>) => Promise<Result>;

async function withBrowserStageQueueLock<Result>(name: string, update: () => Result | Promise<Result>): Promise<Result> {
  const locks = typeof navigator !== "undefined" ? navigator.locks : undefined;
  if (!locks) throw new Error("This browser does not support coordinated updates to the follow-up queue.");
  return locks.request(name, { mode: "exclusive" }, update);
}

/** Keeps one repository's automatic queue-head stage request exclusive across tabs. */
export function withStageQueueDispatchLock<Result>(
  repo: string,
  dispatch: () => Result | Promise<Result>,
  lock: StageQueueLock = withBrowserStageQueueLock,
) {
  return lock(`gct.stageQueue.${repo || "default"}.dispatch`, dispatch);
}
