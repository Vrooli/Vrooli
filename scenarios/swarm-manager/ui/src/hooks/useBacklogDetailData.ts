/**
 * useBacklogDetailData
 *
 * Custom hook that encapsulates ALL data-fetching (useQuery), mutations (useMutation),
 * and derived computations (useMemo) for the BacklogDetailsPage.
 *
 * This keeps the page component focused on UI state and JSX composition while
 * centralising cache-invalidation logic (e.g. archive mutations sharing a single
 * query key) in one place.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  getItemActions,
  scenariosFromGlobs,
} from "../lib";
import type { ItemActions, ResolvedDependencyActivity } from "../lib/backlog-queue-utils";
import { computeDependencyRelations } from "../lib/backlog-queue-utils";
import type { FeedbackItem, MaturityItem } from "../lib/attention";
import { isAgentActivityBlocking } from "../lib/agent-activity-utils";
import type {
  ArchiveRequirementRecord,
  ArchiveTargetFormValues,
  BacklogFile,
  BacklogKind,
  BacklogStatus,
  ModuleFormValues,
  ReviewUpdate,
} from "../types";
import type { ReviewRound } from "../services/review-service";
import { backlogService } from "../services";
import { useAgentActivitiesStore, useBacklogStore } from "../stores";
import { useBacklogQueries } from "./useBacklogQueries";
import { useBacklogMutations } from "./useBacklogMutations";
import type { FileActionType } from "./useBacklogMutations";

// ---------------------------------------------------------------------------
// Options interface
// ---------------------------------------------------------------------------

export interface UseBacklogDetailDataOptions {
  backlogKind: BacklogKind | null;
  name: string | undefined;
  agentRunIsExecuting: boolean;
  agentRunIsBlocking: boolean;
}

// ---------------------------------------------------------------------------
// Hook implementation
// ---------------------------------------------------------------------------

export function useBacklogDetailData({
  backlogKind,
  name,
  agentRunIsExecuting,
  agentRunIsBlocking,
}: UseBacklogDetailDataOptions) {
  const allBacklogItems = useBacklogStore((state) => state.items);
  const blockingMap = useBacklogStore((state) => state.blockingMap);

  // -----------------------------------------------------------------------
  // Queries
  // -----------------------------------------------------------------------

  const queries = useBacklogQueries({ backlogKind, name, agentRunIsBlocking });

  const {
    item,
    isLoadingItem,
    itemError,
    refetchItem,
    spawnedItems,
    files,
    isLoadingFiles,
    filesError,
    refetchFiles,
    executionHistory,
    reviewRounds,
    isGatheringEvidence,
    isAwaitingManualReview,
    workshopDir,
    workshopRoundPaths,
    workshopRounds,
    refetchWorkshopRounds,
    maturitySummaryData,
    readinessData,
    archiveTargets,
  } = queries;

  // -----------------------------------------------------------------------
  // Mutations
  // -----------------------------------------------------------------------

  const {
    updateMutation,
    statusMutation,
    depStatusMutation,
    acceptanceGlobMutation,
    deleteMutation,
    agentMutation,
    workshopSaveMutation,
    updateReqsMutation,
    createModuleMutation,
    updateModuleMetaMutation,
    deleteModuleMutation,
    createTargetMutation,
    updateTargetMutation,
    deleteTargetMutation,
    archiveMutation,
    unarchiveMutation,
    batchReviewMutation,
    workshopDeleteRoundMutation,
    workshopResetMutation,
    fileActionMutation,
    updateError,
    archiveError,
    deleteError,
    agentErrorMsg,
    invalidateFiles,
    invalidateItem,
  } = useBacklogMutations({
    backlogKind,
    name,
    refetchFiles: () => void refetchFiles(),
    refetchWorkshopRounds: () => void refetchWorkshopRounds(),
  });

  // -----------------------------------------------------------------------
  // Computed values
  // -----------------------------------------------------------------------

  // Backlog summary cache (shared with useCommandPostBadgeCount / useNodeActionContext).
  // Powers the attentionReasons overlay on dependency chips.
  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
  });

  const feedbackMap = useMemo(() => new Map<string, FeedbackItem>(), []);

  const maturityMap = useMemo(() => {
    const map = new Map<string, MaturityItem>();
    for (const entry of summaryQuery.data?.maturity?.items ?? []) {
      map.set(`${entry.kind}/${entry.name}`, {
        kind: entry.kind,
        name: entry.name,
        ready: entry.ready ?? false,
        pendingItems: entry.pending_items ?? 0,
      });
    }
    return map;
  }, [summaryQuery.data?.maturity]);

  const activities = useAgentActivitiesStore((s) => s.activities);
  const activityByKey = useMemo(() => {
    const map = new Map<string, ResolvedDependencyActivity>();
    for (const activity of activities) {
      if (activity.ownerType !== "backlog") continue;
      if (!isAgentActivityBlocking(activity.status)) continue;
      const key = `${activity.ownerKind}/${activity.ownerName}`;
      // Activities are sorted newest-first; keep the first (latest) per key.
      if (!map.has(key)) {
        map.set(key, { purpose: activity.purpose, status: activity.status });
      }
    }
    return map;
  }, [activities]);

  const depRelations = useMemo(
    () =>
      item
        ? computeDependencyRelations(item, allBacklogItems, {
            activityByKey,
            feedbackMap,
            maturityMap,
          })
        : { parents: [], children: [] },
    [item, allBacklogItems, activityByKey, feedbackMap, maturityMap],
  );

  const deliverableLabel = backlogKind === "research" ? "Conclusion" : "Plan";
  const deliverableLabelLower = deliverableLabel.toLowerCase();
  const workshopActionLabel = workshopRounds.length > 0 ? "Next Round" : "Workshop";
  const isWorkshopFinalized = workshopRounds.some((r) => r.mode === "finalize")
    && !(readinessData?.pendingSynthesis ?? false);

  const itemActions: ItemActions | null = useMemo(() => {
    if (!item) return null;
    const itemKey = `${item.kind}/${item.name}`;
    return getItemActions({
      item,
      blockingInfo: blockingMap[itemKey] ?? null,
      readinessReady: readinessData ? readinessData.ready : null,
      pendingSynthesis: readinessData?.pendingSynthesis ?? false,
      agentRunning: agentRunIsBlocking,
      agentExecuting: agentRunIsExecuting,
      // Only the LATEST round gates the CTAs: each workshop round supersedes
      // the previous one, so decisions left unanswered in an early round no
      // longer block once a later round (or finalize) exists. This mirrors the
      // server's pending_items computation in the maturity summary.
      hasPendingDecisions: workshopRounds.length > 0
        ? (workshopRounds[workshopRounds.length - 1]?.items?.some(
            (wi) => wi.type === "decision" && wi.selected == null,
          ) ?? false)
        : false,
      hasExecutionHistory: (executionHistory?.length ?? 0) > 0,
      hasTerminalExecution: (executionHistory ?? []).some(
        (e) => e.status === "completed" || e.status === "failed" || e.status === "canceled" || e.status === "needs_fixup",
      ),
    });
  }, [item, blockingMap, readinessData, agentRunIsBlocking, agentRunIsExecuting, workshopRounds, executionHistory]);

  const isLocked = itemActions?.locked ?? false;
  const isTerminal = itemActions?.terminal ?? false;
  const workshopBlockedDeps = itemActions?.blockingDepKeys ?? [];

  const reqModuleMap = useMemo(() => {
    const map = new Map<string, string>();
    if (!archiveTargets) return map;
    const walk = (groups: typeof archiveTargets.requirements) => {
      for (const g of groups) {
        for (const r of g.requirements) {
          map.set(r.id, g.id);
        }
        walk(g.children);
      }
    };
    walk(archiveTargets.requirements);
    return map;
  }, [archiveTargets]);

  const targetIdSet = useMemo(
    () => new Set(archiveTargets?.targets.map((t) => t.id) ?? []),
    [archiveTargets],
  );

  const targetScenarios = useMemo(
    () => scenariosFromGlobs(item?.acceptanceAllow),
    [item?.acceptanceAllow],
  );

  const getAgentDialogTargetIds = useMemo(() => {
    return (selectedTargetIds: Set<string>) => {
      const merged = new Set(selectedTargetIds);
      for (const t of archiveTargets?.targets ?? []) {
        if (t.review_status === "flagged") merged.add(t.id);
      }
      return merged;
    };
  }, [archiveTargets]);

  const getAgentDialogRequirementIds = useMemo(() => {
    return (selectedRequirementIds: Set<string>) => {
      const merged = new Set(selectedRequirementIds);
      if (!archiveTargets) return merged;
      const walkReqs = (groups: typeof archiveTargets.requirements) => {
        for (const g of groups) {
          for (const r of g.requirements) {
            if (r.review_status === "flagged") merged.add(r.id);
          }
          walkReqs(g.children);
        }
      };
      walkReqs(archiveTargets.requirements);
      return merged;
    };
  }, [archiveTargets]);

  // -----------------------------------------------------------------------
  // Return
  // -----------------------------------------------------------------------

  return {
    // Query data
    item,
    isLoadingItem,
    itemError,
    refetchItem: () => void refetchItem(),

    spawnedItems,

    files,
    isLoadingFiles,
    filesError,
    refetchFiles: () => void refetchFiles(),

    executionHistory,

    reviewRounds: reviewRounds ?? ([] as ReviewRound[]),
    isGatheringEvidence,
    isAwaitingManualReview,

    workshopRounds,
    refetchWorkshopRounds: () => void refetchWorkshopRounds(),

    maturitySummaryData,
    readinessData,

    archiveTargets,

    // Computed
    depRelations,
    workshopDir,
    workshopRoundPaths,
    itemActions,
    reqModuleMap,
    targetIdSet,
    agentDialogTargetIds: getAgentDialogTargetIds,
    agentDialogRequirementIds: getAgentDialogRequirementIds,
    targetScenarios,

    deliverableLabel,
    deliverableLabelLower,
    workshopActionLabel,
    isWorkshopFinalized,
    isLocked,
    isTerminal,
    workshopBlockedDeps,

    // Mutations — stable callbacks
    updateItem: (values: { title: string; description: string; status: BacklogStatus; priority: number; tags: string[] }) =>
      updateMutation.mutate(values),
    isUpdating: updateMutation.isPending,
    updateError,
    resetUpdateMutation: () => updateMutation.reset(),

    updateStatus: (newStatus: BacklogStatus) => statusMutation.mutate(newStatus),
    isUpdatingStatus: statusMutation.isPending,

    updateDepStatus: (args: { kind: string; depName: string; newStatus: BacklogStatus }) =>
      depStatusMutation.mutate(args),

    updateAcceptanceGlob: (values: { acceptanceAllow: string[]; acceptanceDeny: string[] }) =>
      acceptanceGlobMutation.mutate(values),
    isUpdatingGlob: acceptanceGlobMutation.isPending,
    resetGlobMutation: () => acceptanceGlobMutation.reset(),

    deleteItem: () => deleteMutation.mutate(),
    isDeleting: deleteMutation.isPending,
    deleteError,
    resetDeleteMutation: () => deleteMutation.reset(),

    isArchiving: archiveMutation.isPending || unarchiveMutation.isPending,
    archiveError,

    runAgent: (payload: { mode?: string; prompt: string; contextPaths?: string[]; contextTargetIds?: string[]; contextRequirementIds?: string[]; confirm?: boolean; force?: boolean }) =>
      agentMutation.mutate(payload),
    isRunningAgent: agentMutation.isPending,
    agentError: agentErrorMsg,
    resetAgentMutation: () => agentMutation.reset(),

    saveWorkshopRound: (roundNumber: number, content: string) =>
      workshopSaveMutation.mutate({ roundNumber, content }),
    isSavingWorkshop: workshopSaveMutation.isPending,

    updateRequirements: (args: { moduleId: string; requirements: ArchiveRequirementRecord[] }) =>
      updateReqsMutation.mutate(args),
    isUpdatingReqs: updateReqsMutation.isPending,
    updateReqsError: updateReqsMutation.error instanceof Error ? updateReqsMutation.error.message : null,
    resetUpdateReqsMutation: () => updateReqsMutation.reset(),

    createModule: (payload: ModuleFormValues & { position?: number }) =>
      createModuleMutation.mutate(payload),
    isCreatingModule: createModuleMutation.isPending,
    createModuleError: createModuleMutation.error instanceof Error ? createModuleMutation.error.message : null,
    resetCreateModuleMutation: () => createModuleMutation.reset(),

    updateModuleMeta: (args: { moduleId: string; payload: { title: string; description: string } }) =>
      updateModuleMetaMutation.mutate(args),
    isUpdatingModuleMeta: updateModuleMetaMutation.isPending,
    updateModuleMetaError: updateModuleMetaMutation.error instanceof Error ? updateModuleMetaMutation.error.message : null,
    resetUpdateModuleMetaMutation: () => updateModuleMetaMutation.reset(),

    deleteModule: (moduleId: string) => deleteModuleMutation.mutate(moduleId),

    createTarget: (target: ArchiveTargetFormValues) => createTargetMutation.mutate(target),
    isCreatingTarget: createTargetMutation.isPending,
    createTargetError: createTargetMutation.error instanceof Error ? createTargetMutation.error.message : null,
    resetCreateTargetMutation: () => createTargetMutation.reset(),

    updateTarget: (args: { targetId: string; target: ArchiveTargetFormValues }) =>
      updateTargetMutation.mutate(args),
    isUpdatingTarget: updateTargetMutation.isPending,
    updateTargetError: updateTargetMutation.error instanceof Error ? updateTargetMutation.error.message : null,
    resetUpdateTargetMutation: () => updateTargetMutation.reset(),

    deleteTarget: (targetId: string) => deleteTargetMutation.mutate(targetId),

    batchReview: (items: ReviewUpdate[]) => batchReviewMutation.mutate(items),
    isBatchReviewing: batchReviewMutation.isPending,
    batchReviewError: batchReviewMutation.error instanceof Error ? batchReviewMutation.error.message : null,

    deleteWorkshopRound: (roundNumber: number) => workshopDeleteRoundMutation.mutate({ roundNumber }),
    isDeletingWorkshopRound: workshopDeleteRoundMutation.isPending,
    isResettingWorkshop: workshopResetMutation.isPending,
    resetWorkshopResetMutation: () => workshopResetMutation.reset(),

    fileAction: (args: { action: FileActionType; target: BacklogFile; destinationPath?: string }) =>
      fileActionMutation.mutate(args),
    isFileActionPending: fileActionMutation.isPending,

    // Invalidation helpers
    invalidateFiles,
    invalidateItem,

    // Raw mutations for advanced wiring (onSuccess with closures in the page)
    _mutations: {
      update: updateMutation,
      status: statusMutation,
      acceptanceGlob: acceptanceGlobMutation,
      delete: deleteMutation,
      archiveMutation,
      unarchiveMutation,
      agent: agentMutation,
      workshopSave: workshopSaveMutation,
      updateReqs: updateReqsMutation,
      createModule: createModuleMutation,
      updateModuleMeta: updateModuleMetaMutation,
      deleteModule: deleteModuleMutation,
      createTarget: createTargetMutation,
      updateTarget: updateTargetMutation,
      deleteTarget: deleteTargetMutation,
      batchReview: batchReviewMutation,
      workshopDeleteRound: workshopDeleteRoundMutation,
      workshopReset: workshopResetMutation,
      fileAction: fileActionMutation,
    },
  } as const;
}
