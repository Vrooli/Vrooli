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

export type QueueCommitBlockReason = "awaiting_unstage" | "stale" | "stage_failed" | "partial_selection" | "not_head";
export type QueueCommitReadiness =
  | { ready: true; target: QueuedCommitTarget | null }
  | { ready: false; reason: QueueCommitBlockReason; selection: QueuedStageSelection };

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

export type StaleSelectionRecovery =
  | { kind: "keep-staged" }
  | { kind: "stage-current"; paths: string[] }
  | { kind: "remove-or-review" };

/** Pick only a recovery that can finish the entire saved selection. Staging a
 * partial subset and leaving the stale queue marker behind creates a dead end. */
export function staleSelectionRecovery(selection: QueuedStageSelection, stagedPaths: readonly string[], worktreeChangedPaths: readonly string[]): StaleSelectionRecovery {
  const staged = new Set(stagedPaths);
  const changed = new Set(worktreeChangedPaths);
  const pathsToStage = [...new Set(selection.paths.filter(path => changed.has(path) || !staged.has(path)))];
  if (pathsToStage.length === 0 && selection.paths.length > 0 && selection.paths.every(path => staged.has(path))) {
    return { kind: "keep-staged" };
  }
  if (selection.paths.length > 0 && selection.paths.every(path => staged.has(path) || changed.has(path)) && pathsToStage.length > 0) {
    return { kind: "stage-current", paths: pathsToStage };
  }
  return { kind: "remove-or-review" };
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
  const result = inspectQueueCommitReadiness(queue, stagedPaths);
  return result.ready ? result.target : null;
}

/** Only a complete, eligible queue head may be attached to a commit. Any staged
 * overlap with an unresolved selection blocks commit until its owner resolves it. */
export function inspectQueueCommitReadiness(queue: readonly QueuedStageSelection[], stagedPaths: readonly string[] | undefined): QueueCommitReadiness {
  if (!stagedPaths) return { ready: true, target: null };
  const staged = new Set(stagedPaths);
  for (const [index, selection] of queue.entries()) {
    const stagedCount = selection.paths.reduce((count, path) => count + Number(staged.has(path)), 0);
    if (stagedCount === 0) continue;
    if (selection.state === "awaiting_unstage") return { ready: false, reason: "awaiting_unstage", selection };
    if (selection.state === "stale") return { ready: false, reason: "stale", selection };
    if (selection.state === "stage_failed") return { ready: false, reason: "stage_failed", selection };
    if (stagedCount !== selection.paths.length) return { ready: false, reason: "partial_selection", selection };
    if (index !== 0) return { ready: false, reason: "not_head", selection };
    if (!selection.snapshotDigest || !["queued", "staged", "commit_pending"].includes(selection.state)) {
      return { ready: false, reason: "stage_failed", selection };
    }
    return { ready: true, target: { entryId: selection.id, snapshotDigest: selection.snapshotDigest } };
  }
  return { ready: true, target: null };
}

export function queueCommitBlockMessage(reason: QueueCommitBlockReason) {
  switch (reason) {
    case "awaiting_unstage": return "Finish queueing the saved selection before committing; some of its files are still staged."
    case "stale": return "Staged files overlap a stale queue entry. Use its recovery action to keep the staged files and clear that entry before committing."
    case "stage_failed": return "A queued selection with a staging error overlaps staged files. Retry staging or remove it before committing."
    case "partial_selection": return "Only part of a queued selection is staged. Stage or unstage the whole selection, or remove it before committing."
    case "not_head": return "A later queued selection overlaps staged files. Resolve it before committing the queue head."
  }
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
