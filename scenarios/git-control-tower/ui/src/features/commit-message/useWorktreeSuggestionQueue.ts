import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QueuedWorktreeSelectionState, StagedMessageMode, WorktreeSelectionSource, WorktreeSuggestionQueueState } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { cancelMessageJob, finalizeQueuedSelection as finalizeQueuedSelectionApi, getMessageJob, listWorktreeSuggestionQueue, queueWorktreeSelection as queueWorktreeSelectionApi, removeWorktreeSuggestionQueueEntry, retireWorktreeSuggestionQueueHead, startQueuedSelection as startQueuedSelectionApi, updateWorktreeSuggestionQueueEntry } from "./api";
import { stagedSuggestionHold, type StagedSuggestionHold } from "./messageSession";
import { isStageSelectionComplete, reconcileStageQueueWithIndex, staleSelectionRecovery, SuccessfulCommitQueueGate, withStageQueueDispatchLock, toQueuedStageSelection, type QueuedCommitTarget, type QueuedStageSelection } from "./stageQueue";
import type { StageResponse } from "../../lib/api";
import { useRepoStatus } from "../../lib/hooks";
import type { StageBatcher } from "../../lib/stage-batcher";
import { checkStageConfirmation, isSameQueuedSelection, removeQueueSelectionIfUnchanged } from "./stageConfirmation";

export interface QueueNotice { tone: "success" | "warning" | "error"; title: string; message: string }
export type PendingSuggestionStage = { repoId: string; hold: StagedSuggestionHold | null; paths: string[]; queuedConflicts: QueuedStageSelection[]; retireSelection?: QueuedStageSelection; allowQueueInstead?: boolean; onSuccess: (data: StageResponse) => void; afterStage?: (data: StageResponse) => void | Promise<void> } | null;

export function useWorktreeSuggestionQueue({ repoId, notify, stageBatcher, pendingSuggestionStage, setPendingSuggestionStage, statusQuery, refreshRepoStatus }: {
  repoId?: string;
  notify: (notice: QueueNotice) => void;
  stageBatcher: StageBatcher<StageResponse>;
  pendingSuggestionStage: PendingSuggestionStage;
  setPendingSuggestionStage: (next: PendingSuggestionStage) => void;
  statusQuery: ReturnType<typeof useRepoStatus>;
  refreshRepoStatus: ReturnType<typeof useRepoStatus>["refetch"];
}) {
  const queryClient = useQueryClient();
  const stageQueueRepo = repoId ?? "";
  const stageQueueKey = useMemo(() => ["worktree-suggestion-queue", stageQueueRepo] as const, [stageQueueRepo]);
  const publishStageQueue = useCallback((targetRepo: string, incoming: Awaited<ReturnType<typeof listWorktreeSuggestionQueue>>) => {
    const key = ["worktree-suggestion-queue", targetRepo] as const;
    const current = queryClient.getQueryData<typeof incoming>(key);
    // Queue revisions advance on every durable change. Reuse the current object
    // at equal revisions so polling does not rerender the commit surface.
    if (current && current.revision >= incoming.revision) return current;
    queryClient.setQueryData(key, incoming);
    return incoming;
  }, [queryClient]);
  const stageQueueQuery = useQuery({
    queryKey: stageQueueKey,
    queryFn: async () => publishStageQueue(stageQueueRepo, await listWorktreeSuggestionQueue(stageQueueRepo)),
    enabled: Boolean(stageQueueRepo),
    refetchInterval: query => query.state.data?.entries.some(entry =>
      entry.state === WorktreeSuggestionQueueState.AWAITING_UNSTAGE ||
      entry.state === WorktreeSuggestionQueueState.COMMIT_PENDING,
    ) ? 5_000 : 30_000,
  });
  const suggestionStageQueue = useMemo(() => stageQueueQuery.data?.entries.map(toQueuedStageSelection) ?? [], [stageQueueQuery.data?.entries]);
  const successfulCommitsAwaitingQueue = useRef(new SuccessfulCommitQueueGate());
  const [queueRecoveryPendingId, setQueueRecoveryPendingId] = useState<string | null>(null);
  const pendingCommitQueueSelectionId = useRef<QueuedCommitTarget | null>(null);
  const dispatchQueuedStageForRepo = useRef<((targetRepo: string) => void) | null>(null);
  const unresolvedQueueConflicts = useRef(new Map<string, Set<string>>());
  const activeStageQueueRepo = useRef(stageQueueRepo);
  activeStageQueueRepo.current = stageQueueRepo;
  const refreshStageQueue = useCallback(async (targetRepo: string) => {
    const queue = await listWorktreeSuggestionQueue(targetRepo);
    if (activeStageQueueRepo.current === targetRepo) return publishStageQueue(targetRepo, queue);
    return queue;
  }, [publishStageQueue]);
  const mutateQueueForRepo = useCallback(async (targetRepo: string, update: (current: QueuedStageSelection[]) => QueuedStageSelection[]) => {
    let queue = await listWorktreeSuggestionQueue(targetRepo);
    const proposed = update(queue.entries.map(toQueuedStageSelection));
    const proposedById = new Map(proposed.map(entry => [entry.id, entry]));
    try {
      for (const entry of queue.entries) {
        const next = proposedById.get(entry.id);
        if (!next) {
          const response = await removeWorktreeSuggestionQueueEntry({ repositoryId: targetRepo, entryId: entry.id, expectedRevision: queue.revision });
          if (response.queue) queue = response.queue;
          continue;
        }
        if (next.state === toQueuedStageSelection(entry).state && next.stageError === toQueuedStageSelection(entry).stageError) continue;
        const state = next.state === "staged" ? WorktreeSuggestionQueueState.STAGED
          : next.state === "queued" ? WorktreeSuggestionQueueState.QUEUED
            : next.state === "stale" ? WorktreeSuggestionQueueState.STALE
              : next.state === "commit_pending" ? WorktreeSuggestionQueueState.COMMIT_PENDING : WorktreeSuggestionQueueState.FAILED;
        const response = await updateWorktreeSuggestionQueueEntry({
          repositoryId: targetRepo, entryId: entry.id, expectedHeadEntryId: queue.entries[0]?.id,
          expectedRevision: queue.revision, state,
          failureReason: next.state === "stale" ? next.stageError ?? "The queued files changed after capture." : next.stageError ?? "",
        });
        if (response.queue) queue = response.queue;
      }
      queue = await listWorktreeSuggestionQueue(targetRepo);
      if (activeStageQueueRepo.current === targetRepo) queue = publishStageQueue(targetRepo, queue);
      return { persisted: true, queue: queue.entries.map(toQueuedStageSelection), previous: [] as QueuedStageSelection[] };
    } catch (error) {
      queue = await listWorktreeSuggestionQueue(targetRepo).catch(() => queue);
      if (activeStageQueueRepo.current === targetRepo) queue = publishStageQueue(targetRepo, queue);
      return { persisted: false, queue: queue.entries.map(toQueuedStageSelection), previous: queue.entries.map(toQueuedStageSelection), error: error instanceof Error ? error.message : String(error) };
    }
  }, [publishStageQueue]);
  const completeQueuedStageSelectionFor = useCallback(async (id: string, targetRepo: string, expected?: QueuedStageSelection) => {
    const initial = await listWorktreeSuggestionQueue(targetRepo);
    const selection = initial.entries.map(toQueuedStageSelection).find(item => item.id === id);
    if (!selection) return true;
    if (expected && !isSameQueuedSelection(selection, expected)) return false;
    if (selection.requestKey && (selection.jobId || selection.suggestionMode)) {
      try { await cancelMessageJob(targetRepo, selection.jobId ?? "", selection.requestKey); }
      catch (error) { notify({ tone: "warning", title: "Queue entry removed; suggestion may continue", message: error instanceof Error ? error.message : "The suggestion owner did not confirm cancellation." }); }
    }
    let lastQueue: Awaited<ReturnType<typeof listWorktreeSuggestionQueue>> | undefined;
    const removed = await removeQueueSelectionIfUnchanged(expected ?? selection, async () => {
      lastQueue = await listWorktreeSuggestionQueue(targetRepo);
      return { revision: lastQueue.revision, entries: lastQueue.entries.map(toQueuedStageSelection) };
    }, async revision => {
      const response = await removeWorktreeSuggestionQueueEntry({ repositoryId: targetRepo, entryId: id, expectedRevision: revision });
      lastQueue = response.queue ?? lastQueue;
      if (response.queue && activeStageQueueRepo.current === targetRepo) publishStageQueue(targetRepo, response.queue);
    });
    if (removed) {
      const blocked = unresolvedQueueConflicts.current.get(targetRepo);
      blocked?.delete(id);
      if (blocked?.size === 0) {
        unresolvedQueueConflicts.current.delete(targetRepo);
        if (successfulCommitsAwaitingQueue.current.canDispatch(targetRepo, Boolean(lastQueue?.entries.length))) dispatchQueuedStageForRepo.current?.(targetRepo);
      }
      return true;
    }
    notify({ tone: "warning", title: "Queued selection changed", message: "The queue entry changed before removal. It remains saved for review." });
    return false;
  }, [notify, publishStageQueue]);
  const completeQueuedStageSelection = useCallback((id: string) => {
    void completeQueuedStageSelectionFor(id, stageQueueRepo).then(persisted => {
      if (!persisted) notify({ tone: "warning", title: "Queue change was not saved", message: "The server could not confirm the queue change. The saved queue remains visible." });
    }).catch(error => notify({ tone: "error", title: "Could not update queued selection", message: error instanceof Error ? error.message : String(error) }));
  }, [completeQueuedStageSelectionFor, notify, stageQueueRepo]);
  const retireConfirmedQueuedSuggestions = useCallback(async (response: StageResponse, targetRepo: string, confirmed: QueuedStageSelection[]) => {
    if (!response.success || confirmed.length === 0) return;
    const stagedPaths = response.staged;
    if (stagedPaths.length === 0) return;
    const staged = new Set(stagedPaths);
    const removed: string[] = [];
    const unresolved: string[] = [];
    const queue = await listWorktreeSuggestionQueue(targetRepo);
    for (const selection of confirmed) {
      if (!selection.paths.every(path => staged.has(path))) continue;
      const current = queue.entries.map(toQueuedStageSelection).find(item => item.id === selection.id);
      // A replaced entry can reuse its queue id. Its changed digest or paths
      // make it a new user decision, so leave it for another confirmation.
      if (!current || !isSameQueuedSelection(current, selection)) continue;
      (await completeQueuedStageSelectionFor(selection.id, targetRepo, selection) ? removed : unresolved).push(selection.id);
    }
    const laterConflicts = (await listWorktreeSuggestionQueue(targetRepo)).entries.map(toQueuedStageSelection)
      .filter(selection => !removed.includes(selection.id) && selection.paths.some(path => staged.has(path)));
    if (laterConflicts.length) {
      const blocked = unresolvedQueueConflicts.current.get(targetRepo) ?? new Set<string>();
      laterConflicts.forEach(selection => { blocked.add(selection.id); unresolved.push(selection.id); });
      unresolvedQueueConflicts.current.set(targetRepo, blocked);
    }
    if (removed.length) notify({ tone: "warning", title: "Overlapping queued suggestions removed", message: `${removed.length} queued selection${removed.length === 1 ? " was" : "s were"} removed because staged files are now part of the current commit.` });
    if (unresolved.length) {
      const blocked = unresolvedQueueConflicts.current.get(targetRepo) ?? new Set<string>();
      unresolved.forEach(id => blocked.add(id));
      unresolvedQueueConflicts.current.set(targetRepo, blocked);
      notify({ tone: "error", title: "Resolve overlapping queued selections", message: `${unresolved.length} selection${unresolved.length === 1 ? " still overlaps" : "s still overlap"} the staged files. The follow-up queue is paused until ${unresolved.length === 1 ? "it is removed" : "they are removed"}.` });
    }
  }, [completeQueuedStageSelectionFor, notify]);
  const startQueuedSuggestion = useCallback(async (selection: QueuedStageSelection, mode: StagedMessageMode, targetRepo = stageQueueRepo, requestKey = `${selection.selectionId}:${mode}`) => {
    if (!selection.selectionId) return;
    try {
      // Re-admit from the immutable queue selection. A job id can outlive the
      // job row (or become unreachable); retrying by that id cannot recover it.
      await startQueuedSelectionApi({ repositoryId: targetRepo, selectionId: selection.selectionId, snapshotDigest: selection.snapshotDigest ?? "", requestKey, mode });
      await refreshStageQueue(targetRepo);
    } catch (error) {
      notify({ tone: "error", title: "Could not start queued suggestion", message: error instanceof Error ? error.message : "The suggestion request could not be confirmed. Reconnect to check its saved state." });
    }
  }, [notify, refreshStageQueue, stageQueueRepo]);
  const retryQueuedSuggestion = useCallback((selection: QueuedStageSelection) => {
    const mode = selection.suggestionMode === "deep" ? StagedMessageMode.DEEP : StagedMessageMode.QUICK;
    void startQueuedSuggestion(selection, mode, stageQueueRepo, `${selection.selectionId}:retry:${crypto.randomUUID()}`);
  }, [stageQueueRepo, startQueuedSuggestion]);
  const refreshQueuedSuggestion = useCallback(async (selection: QueuedStageSelection) => {
    try {
      if (!selection.selectionId || !selection.suggestionMode) throw new Error("Choose a quick or deep suggestion before refreshing this selection.");
      await startQueuedSelectionApi({ repositoryId: stageQueueRepo, selectionId: selection.selectionId, snapshotDigest: selection.snapshotDigest ?? "", requestKey: `${selection.selectionId}:retry:${crypto.randomUUID()}`, mode: selection.suggestionMode === "deep" ? StagedMessageMode.DEEP : StagedMessageMode.QUICK });
      await refreshStageQueue(stageQueueRepo);
      if (successfulCommitsAwaitingQueue.current.canDispatch(stageQueueRepo, true)) dispatchQueuedStageForRepo.current?.(stageQueueRepo);
    } catch (error) {
      notify({ tone: "error", title: "Could not refresh queued suggestion", message: error instanceof Error ? error.message : "The queued snapshot could not be replaced. The existing entry remains saved." });
    }
  }, [notify, refreshStageQueue, stageQueueRepo]);
  const queueSelection = useCallback(async (paths: string[], source: WorktreeSelectionSource) => {
    const targetRepo = stageQueueRepo;
    try {
      const before = await listWorktreeSuggestionQueue(targetRepo);
      const queuedPaths = new Set(before.entries.flatMap(entry => entry.paths));
      const uniquePaths = [...new Set(paths)];
      if (uniquePaths.some(path => queuedPaths.has(path))) {
        notify({ tone: "warning", title: "Files already queued", message: "Each file can belong to only one queued commit selection." });
        return "already-queued";
      }
      const selectionId = crypto.randomUUID();
      const selection = await queueWorktreeSelectionApi({ repositoryId: targetRepo, selectionId, source, paths: uniquePaths });
      let after: Awaited<ReturnType<typeof listWorktreeSuggestionQueue>> | undefined;
      try { after = await refreshStageQueue(targetRepo); }
      catch { /* The admission response is authoritative; refresh can recover independently. */ }
      if (activeStageQueueRepo.current === targetRepo) setPendingSuggestionStage(null);
      notify({ tone: after ? "success" : "warning", title: after ? "Files queued" : "Files saved; queue status reconnecting", message: `${selection.paths.length} file${selection.paths.length === 1 ? "" : "s"} saved as a separate commit selection. Choose quick or deep later if you want a suggestion.` });
      if (after && successfulCommitsAwaitingQueue.current.canDispatch(targetRepo, after.entries.length > 0)) dispatchQueuedStageForRepo.current?.(targetRepo);
      return selection;
    } catch (error) {
      notify({ tone: "error", title: "Could not save queued selection", message: error instanceof Error ? error.message : String(error) });
      void refreshStageQueue(targetRepo);
      return undefined;
    }
  }, [notify, refreshStageQueue, setPendingSuggestionStage, stageQueueRepo]);
  const finalizeQueuedSelection = useCallback(async (selectionId: string, snapshotDigest: string, targetRepo = stageQueueRepo) => {
    const finalized = await finalizeQueuedSelectionApi({ repositoryId: targetRepo, selectionId, expectedSnapshotDigest: snapshotDigest });
    await refreshStageQueue(targetRepo);
    return finalized.state === QueuedWorktreeSelectionState.QUEUED;
  }, [refreshStageQueue, stageQueueRepo]);
  const markQueuedSelectionStale = useCallback(async (selection: QueuedStageSelection, reason: string) => {
    const result = await mutateQueueForRepo(stageQueueRepo, current => current.map(item =>
      item.id === selection.id && item.snapshotDigest === selection.snapshotDigest
        ? { ...item, state: "stale", stageError: reason }
        : item,
    ));
    if (!result.persisted) throw new Error(result.error ?? "The queue could not save the stale selection state.");
  }, [mutateQueueForRepo, stageQueueRepo]);
  const stageNextQueuedSelectionForRepo = useCallback((targetRepo: string) => {
    void withStageQueueDispatchLock(targetRepo, async () => {
      if (unresolvedQueueConflicts.current.get(targetRepo)?.size) return;
      const serverQueue = await listWorktreeSuggestionQueue(targetRepo);
      const next = serverQueue.entries[0] ? toQueuedStageSelection(serverQueue.entries[0]) : undefined;
      if (!next || next.state === "staged" || next.state === "stale" || next.state === "commit_pending" || next.state === "awaiting_unstage") return;
      const markFailed = async (message: string, stale: boolean) => {
        await mutateQueueForRepo(targetRepo, current => current.map(item => item.id === next.id ? { ...item, state: stale ? "stale" : "stage_failed", stageError: message } : item));
        notify({ tone: "error", title: stale ? "Queued selection is stale" : "Queued selection needs review", message });
      };
      if (next.selectionId && next.snapshotDigest) {
        try {
          await finalizeQueuedSelection(next.selectionId, next.snapshotDigest, targetRepo);
        } catch (error) {
          await markFailed(error instanceof Error ? error.message : "The queued files changed after capture. Refresh or remove this selection before staging.", true);
          return;
        }
      } else if (next.suggestionMode && next.requestKey) {
        try {
          const job = await getMessageJob(targetRepo, next.jobId ?? "", next.requestKey);
          if (!job.applicable) {
            await markFailed(job.applicabilityReason || "The queued files changed after their suggestion was captured. Refresh the suggestion before staging.", true);
            return;
          }
        } catch (error) {
          await markFailed(error instanceof Error ? error.message : "Could not verify the queued suggestion against current files.", /changed|stale|no longer match/i.test(String(error)));
          return;
        }
      }
      if (successfulCommitsAwaitingQueue.current.canDispatch(targetRepo, true)) successfulCommitsAwaitingQueue.current.consume(targetRepo);
      await new Promise<void>(resolve => {
        let updateState: Promise<unknown> | undefined;
        stageBatcher.enqueue(next.paths, {
          onSuccess: result => {
            if (isStageSelectionComplete(next, result)) {
              updateState = mutateQueueForRepo(targetRepo, current => current.map(item => item.id === next.id ? { ...item, state: "staged", stageError: undefined } : item)).then(update => {
                const stagedEntry = update.queue.find(item => item.id === next.id);
                const accepted = update.persisted && stagedEntry?.state === "staged";
                notify({
                  tone: accepted ? "success" : "warning",
                  title: accepted ? "Next commit selection staged" : stagedEntry?.state === "stale" ? "Queued files changed" : "Selection staged; queue needs review",
                  message: accepted
                    ? `${next.paths.length} captured files are staged and ready for review.`
                    : stagedEntry?.state === "stale"
                      ? "The staged contents no longer match the saved selection. Keep these staged versions and clear the stale entry, or remove it before committing."
                      : "Files are staged, but GCT has not confirmed that the queue selection matches the index. Review or remove the queue entry before committing.",
                });
              });
              return;
            }
            const message = (result.errors ?? result.failed ?? []).join("\n") || "The queued selection was not staged completely. It remains in the queue.";
            updateState = mutateQueueForRepo(targetRepo, current => current.map(item => item.id === next.id ? { ...item, state: "stage_failed", stageError: message } : item));
            notify({ tone: "error", title: "Queued staging needs attention", message });
          },
          onError: error => {
            updateState = mutateQueueForRepo(targetRepo, current => current.map(item => item.id === next.id ? { ...item, state: "stage_failed", stageError: error instanceof Error ? error.message : String(error) } : item));
            console.error("Queued file staging failed; selection remains queued for retry.", error);
          },
          onSettled: () => { void (updateState ?? Promise.resolve()).finally(resolve); },
        });
      });
    }).catch(error => notify({ tone: "error", title: "Could not coordinate queued staging", message: error instanceof Error ? error.message : String(error) }));
  }, [finalizeQueuedSelection, mutateQueueForRepo, notify, stageBatcher]);
  dispatchQueuedStageForRepo.current = stageNextQueuedSelectionForRepo;
  const stageNextQueuedSelection = useCallback(() => {
    stageNextQueuedSelectionForRepo(stageQueueRepo);
  }, [stageNextQueuedSelectionForRepo, stageQueueRepo]);
  const stageQueueAfterCommit = useCallback((committedSelection: QueuedCommitTarget | null) => {
    const targetRepo = stageQueueRepo;
    void (async () => {
      if (committedSelection) {
        let retired = false;
        for (let attempt = 0; attempt < 4 && !retired; attempt++) {
          const queue = await listWorktreeSuggestionQueue(targetRepo);
          const head = queue.entries[0];
          if (head?.id !== committedSelection.entryId || head.snapshotDigest !== committedSelection.snapshotDigest) { retired = true; break; }
          try {
            await retireWorktreeSuggestionQueueHead({ repositoryId: targetRepo, expectedHeadEntryId: committedSelection.entryId, expectedRevision: queue.revision });
            retired = true;
          } catch (error) {
            if (attempt === 3) throw error;
            await new Promise(resolve => setTimeout(resolve, 50 * (attempt + 1)));
          }
        }
        await refreshStageQueue(targetRepo);
      }
      successfulCommitsAwaitingQueue.current.markSuccessful(targetRepo);
      stageNextQueuedSelectionForRepo(targetRepo);
    })().catch(error => notify({ tone: "error", title: "Could not advance the follow-up queue", message: error instanceof Error ? error.message : String(error) }));
  }, [notify, refreshStageQueue, stageNextQueuedSelectionForRepo, stageQueueRepo]);
  const markQueuedCommitPending = useCallback(async (targetRepo: string, target: QueuedCommitTarget, stagedPaths: readonly string[], allowAlreadyPending = false) => {
    const queue = await listWorktreeSuggestionQueue(targetRepo);
    const head = queue.entries[0] ? toQueuedStageSelection(queue.entries[0]) : undefined;
    if (!head || head.id !== target.entryId || head.snapshotDigest !== target.snapshotDigest || !head.paths.every(path => stagedPaths.includes(path))) {
      throw new Error("The queued selection is no longer the staged queue head. Review the index before committing.");
    }
    if (head.state === "commit_pending") {
      if (allowAlreadyPending) return;
      throw new Error("This queued selection already has a commit in progress. Resolve that commit before retrying.");
    }
    if (head.state !== "staged" && head.state !== "queued") throw new Error("The queued selection is not ready to commit. Resolve its staging state first.");
    await updateWorktreeSuggestionQueueEntry({
      repositoryId: targetRepo, entryId: target.entryId, expectedHeadEntryId: target.entryId, expectedRevision: queue.revision,
      state: WorktreeSuggestionQueueState.COMMIT_PENDING,
    });
    await refreshStageQueue(targetRepo);
  }, [refreshStageQueue]);
  const restoreQueuedCommitAfterFailure = useCallback(async (targetRepo: string, target: QueuedCommitTarget) => {
    const [status, queue] = await Promise.all([refreshRepoStatus(), listWorktreeSuggestionQueue(targetRepo)]);
    const head = queue.entries[0] ? toQueuedStageSelection(queue.entries[0]) : undefined;
    if (!head || head.id !== target.entryId || head.snapshotDigest !== target.snapshotDigest || head.state !== "commit_pending") return;
    const stillStaged = head.paths.every(path => status.data?.files.staged.includes(path));
    await updateWorktreeSuggestionQueueEntry({
      repositoryId: targetRepo, entryId: target.entryId, expectedHeadEntryId: target.entryId, expectedRevision: queue.revision,
      state: stillStaged ? WorktreeSuggestionQueueState.STAGED : WorktreeSuggestionQueueState.QUEUED,
    });
    await refreshStageQueue(targetRepo);
  }, [refreshRepoStatus, refreshStageQueue]);
  const resolveQueuedCommit = useCallback((entryId: string, committed: boolean) => {
    void listWorktreeSuggestionQueue(stageQueueRepo).then(queue => {
      const head = queue.entries[0];
      if (!head || head.id !== entryId) throw new Error("The queued entry is no longer the queue head.");
      const target = { entryId: head.id, snapshotDigest: head.snapshotDigest };
      if (committed) stageQueueAfterCommit(target);
      else return restoreQueuedCommitAfterFailure(stageQueueRepo, target);
    }).catch(error => notify({ tone: "error", title: committed ? "Could not resolve committed selection" : "Could not restore queued selection", message: error instanceof Error ? error.message : String(error) }));
  }, [notify, restoreQueuedCommitAfterFailure, stageQueueAfterCommit, stageQueueRepo]);
  useEffect(() => {
    const stagedPaths = statusQuery.data?.files.staged;
    const head = suggestionStageQueue[0];
    if (!stagedPaths || !head) return;
    const [reconciled] = reconcileStageQueueWithIndex([head], stagedPaths);
    if (reconciled && (reconciled.state !== head.state || reconciled.stageError !== head.stageError)) {
      void mutateQueueForRepo(stageQueueRepo, current => current.map(item => item.id === head.id ? reconciled : item));
    }
  }, [mutateQueueForRepo, stageQueueRepo, statusQuery.data?.files.staged, suggestionStageQueue]);
  useEffect(() => {
    const blocked = unresolvedQueueConflicts.current.get(stageQueueRepo);
    if (!blocked?.size) return;
    const remaining = new Set(suggestionStageQueue.map(selection => selection.id));
    for (const id of blocked) if (!remaining.has(id)) blocked.delete(id);
    if (blocked.size === 0) {
      unresolvedQueueConflicts.current.delete(stageQueueRepo);
      if (successfulCommitsAwaitingQueue.current.canDispatch(stageQueueRepo, suggestionStageQueue.length > 0)) dispatchQueuedStageForRepo.current?.(stageQueueRepo);
    }
  }, [stageQueueRepo, suggestionStageQueue]);
  // Every stage entry point confirms while a commit suggestion describes the
  // current staged selection, so it is not invalidated by accident.
  const requestStage = useCallback(
    (paths: string[], onSuccess: (data: StageResponse) => void) => {
      const uniquePaths = [...new Set(paths)];
      if (uniquePaths.length === 0) return;
      if (!statusQuery.data) {
        notify({ tone: "warning", title: "Cannot confirm staged selection", message: "Repository status is unavailable. Wait for it to load before staging so an active suggestion is not changed unexpectedly." });
        return;
      }
      const hold = stagedSuggestionHold(queryClient, repoId ?? "");
      const staged = new Set(uniquePaths);
      const queuedConflicts = suggestionStageQueue.filter(selection => selection.paths.some(path => staged.has(path)));
      if (hold || queuedConflicts.length > 0) {
        setPendingSuggestionStage({ repoId: repoId ?? "", hold: hold ?? null, paths: uniquePaths, queuedConflicts, onSuccess });
      }
      else stageBatcher.enqueue(uniquePaths, { onSuccess });
    },
    [notify, queryClient, repoId, setPendingSuggestionStage, stageBatcher, statusQuery.data, suggestionStageQueue],
  );
  const stageQueuedSelectionNow = useCallback((selection: QueuedStageSelection) => {
    if (queueRecoveryPendingId === selection.id) return;
    const targetRepo = stageQueueRepo;
    setQueueRecoveryPendingId(selection.id);
    void (async () => {
      const freshStatus = await refreshRepoStatus();
      const freshStagedPaths = freshStatus.data?.files.staged;
      if (!freshStatus.data || !freshStagedPaths) {
        notify({ tone: "warning", title: "Cannot confirm staged selection", message: "Repository status is unavailable. Wait for it to load before staging." });
        return;
      }
      const latest = await listWorktreeSuggestionQueue(targetRepo);
      if (activeStageQueueRepo.current !== targetRepo) throw new Error("The repository changed before staging began.");
      const entries = latest.entries.map(toQueuedStageSelection);
      const current = entries.find(item => item.id === selection.id);
      if (!current || !isSameQueuedSelection(current, selection)) {
        notify({ tone: "warning", title: "Queued selection changed", message: "Reload the queue card and review the current selection before staging." });
        return;
      }
      const staged = new Set(freshStagedPaths);
      const alreadyStaged = selection.paths.filter(path => staged.has(path));
      const worktreeChanged = new Set([...(freshStatus.data.files.unstaged ?? []), ...(freshStatus.data.files.untracked ?? [])]);
      const recovery = staleSelectionRecovery(selection, freshStagedPaths, [...worktreeChanged]);
      if (recovery.kind === "keep-staged") {
        const removed = await completeQueuedStageSelectionFor(selection.id, targetRepo, selection);
        if (!removed) return;
        await refreshStageQueue(targetRepo);
        notify({ tone: "success", title: "Staged files kept", message: `${selection.paths.length} files remain staged for this commit. The stale queue entry was cleared.` });
        return;
      }
      if (recovery.kind === "remove-or-review") {
        notify({ tone: "warning", title: "Selection cannot be staged as a whole", message: "At least one selected path is neither staged nor changed in the worktree. Remove this stale queue entry or review the files again; GCT left the index unchanged." });
        return;
      }
      const pathsToStage = recovery.paths;
      const hold = stagedSuggestionHold(queryClient, targetRepo);
      const paths = pathsToStage;
      const selectedPaths = new Set(paths);
      const queuedConflicts = entries.filter(item => item.id !== selection.id && item.paths.some(path => selectedPaths.has(path)));
      const onSuccess = (data: StageResponse) => {
        if (data.warnings?.length) notify({ tone: "warning", title: "Some files were skipped", message: data.warnings.join("\n") });
      };
      const afterStage = async (data: StageResponse) => {
        if (!data.success) {
          notify({ tone: "error", title: "Selection remains in queue", message: (data.errors ?? data.failed ?? []).join("\n") || "GCT could not stage the remaining files. The saved selection is still in the queue." });
          return;
        }
        const afterStatus = await refreshRepoStatus();
        const afterStaged = new Set(afterStatus.data?.files.staged ?? []);
        const complete = isStageSelectionComplete(selection, { success: true, staged: [...alreadyStaged, ...data.staged, ...afterStaged] });
        if (!complete) {
          notify({ tone: "error", title: "Selection remains in queue", message: "Some selected files are still unstaged. Review the file list and retry; the staged files were left untouched." });
          return;
        }
        const removed = await completeQueuedStageSelectionFor(selection.id, targetRepo, selection);
        if (!removed) notify({ tone: "error", title: "Files staged; queue still needs cleanup", message: "The files are staged, but the saved selection changed before GCT could remove it. Remove or review that queue entry before committing." });
      };
      if (hold || queuedConflicts.length) {
        setPendingSuggestionStage({ repoId: targetRepo, hold: hold ?? null, paths, queuedConflicts, retireSelection: selection, allowQueueInstead: false, onSuccess, afterStage });
        return;
      }
      stageBatcher.enqueue(paths, {
        onSuccess: data => { onSuccess(data); void afterStage(data).catch(error => notify({ tone: "error", title: "Files staged; queue cleanup failed", message: error instanceof Error ? error.message : String(error) })); },
        onError: error => notify({ tone: "error", title: "Could not stage queued files", message: error instanceof Error ? error.message : String(error) }),
      });
    })().catch(error => notify({ tone: "error", title: "Could not stage queued files", message: error instanceof Error ? error.message : String(error) }))
      .finally(() => setQueueRecoveryPendingId(current => current === selection.id ? null : current));
  }, [completeQueuedStageSelectionFor, notify, queryClient, queueRecoveryPendingId, refreshRepoStatus, refreshStageQueue, setPendingSuggestionStage, stageBatcher, stageQueueRepo]);
  const handleConfirmSuggestionStage = useCallback(() => {
    const pending = pendingSuggestionStage;
    if (!pending) return;
    if (checkStageConfirmation(pending.repoId, repoId ?? "", [], []) === "stale-repository" || stageBatcher.scopeKey !== (repoId ?? "default")) {
      setPendingSuggestionStage(null);
      notify({ tone: "warning", title: "Repository changed", message: "This staging confirmation belongs to a different repository. Review the selection again before staging." });
      return;
    }
    void (async () => {
      const latestEntries = (await listWorktreeSuggestionQueue(pending.repoId)).entries.map(toQueuedStageSelection);
      const currentOwnedSelection = pending.retireSelection && latestEntries.find(selection => selection.id === pending.retireSelection?.id);
      const current = latestEntries.filter(selection => selection.id !== pending.retireSelection?.id && selection.paths.some(path => pending.paths.includes(path)));
      if (activeStageQueueRepo.current !== pending.repoId) {
        setPendingSuggestionStage(null);
        notify({ tone: "warning", title: "Repository changed", message: "The repository changed while checking the queue. Review the selection again before staging." });
        return;
      }
      if (pending.retireSelection && (!currentOwnedSelection || !isSameQueuedSelection(currentOwnedSelection, pending.retireSelection))) {
        setPendingSuggestionStage(null);
        notify({ tone: "warning", title: "Queued selection changed", message: "This saved selection changed while staging was being confirmed. Reload the queue card and review it again." });
        return;
      }
      if (checkStageConfirmation(pending.repoId, repoId ?? "", pending.queuedConflicts, current) !== "confirmed") {
        setPendingSuggestionStage({ ...pending, queuedConflicts: current });
        notify({ tone: "warning", title: "Queue changed; review again", message: "The overlapping queued selections changed while confirmation was open. Review the updated list before staging." });
        return;
      }
      const recoveryId = pending.retireSelection?.id;
      if (recoveryId) setQueueRecoveryPendingId(recoveryId);
      stageBatcher.enqueue(pending.paths, { onSuccess: data => {
        pending.onSuccess(data);
        void (async () => {
          try {
            await retireConfirmedQueuedSuggestions(data, pending.repoId, pending.queuedConflicts);
          } catch (error) {
            notify({ tone: "error", title: "Could not update overlapping queued selections", message: error instanceof Error ? error.message : "The saved queue could not be reconciled with staged files." });
          }
          try {
            await pending.afterStage?.(data);
          } catch (error) {
            notify({ tone: "error", title: "Files staged; queue cleanup failed", message: error instanceof Error ? error.message : String(error) });
          }
        })().finally(() => {
          if (recoveryId) setQueueRecoveryPendingId(id => id === recoveryId ? null : id);
        });
      }, onError: error => {
        if (recoveryId) setQueueRecoveryPendingId(id => id === recoveryId ? null : id);
        notify({ tone: "error", title: "Could not stage these files", message: error instanceof Error ? error.message : "The index was not changed. The saved queue entry remains available." });
      }});
      setPendingSuggestionStage(null);
    })().catch(error => notify({ tone: "error", title: "Could not verify queued selections", message: error instanceof Error ? error.message : String(error) }));
  }, [pendingSuggestionStage, repoId, retireConfirmedQueuedSuggestions, setPendingSuggestionStage, setQueueRecoveryPendingId, stageBatcher, notify]);
  const handleQueueSuggestionStage = useCallback(() => {
    if (!pendingSuggestionStage) return;
    if (pendingSuggestionStage.repoId !== stageQueueRepo) {
      setPendingSuggestionStage(null);
      notify({ tone: "warning", title: "Repository changed", message: "This selection belongs to another repository. Review the selection again before queueing it." });
      return;
    }
    void queueSelection(pendingSuggestionStage.paths, WorktreeSelectionSource.INDEX_TO_WORKTREE);
  }, [pendingSuggestionStage, queueSelection, setPendingSuggestionStage, stageQueueRepo, notify]);

  return { stageQueueRepo, stageQueueReady: stageQueueQuery.isSuccess, suggestionStageQueue, queueRecoveryPendingId, pendingCommitQueueSelectionId, completeQueuedStageSelection, resolveQueuedCommit, stageNextQueuedSelection, retryQueuedSuggestion, refreshQueuedSuggestion, queueSelection, finalizeQueuedSelection, markQueuedSelectionStale, startQueuedSuggestion, requestStage, stageQueuedSelectionNow, handleConfirmSuggestionStage, handleQueueSuggestionStage, markQueuedCommitPending, restoreQueuedCommitAfterFailure, stageQueueAfterCommit };
}
