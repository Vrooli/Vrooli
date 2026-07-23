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
import {
  getItemActions,
  scenariosFromGlobs,
} from "../lib";
import type { ItemActions, ResolvedDependencyActivity } from "../lib/backlog-queue-utils";
import { computeDependencyRelations } from "../lib/backlog-queue-utils";
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
    nextAction,
    reviewRounds,
    isGatheringEvidence,
    isAwaitingManualReview,
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
    descriptionMutation,
    deleteMutation,
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
    fileActionMutation,
    updateError,
    archiveError,
    deleteError,
    invalidateFiles,
    invalidateItem,
  } = useBacklogMutations({ backlogKind, name });

  // -----------------------------------------------------------------------
  // Computed values
  // -----------------------------------------------------------------------

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
        ? computeDependencyRelations(item, allBacklogItems, { activityByKey })
        : { parents: [], children: [] },
    [item, allBacklogItems, activityByKey],
  );

  // Default client-side CTA funnel, then gated by the canonical workflow
  // projection's legal_actions when a workflow document exists for the item.
  // No workflow (found=false → the item has not run an operation yet) → the
  // client funnel applies unchanged. See lib/workflow-legal-actions.ts.
  const itemActions: ItemActions | null = useMemo(() => {
    if (!item) return null;
    const itemKey = `${item.kind}/${item.name}`;
    const clientActions = getItemActions({
      item,
      blockingInfo: blockingMap[itemKey] ?? null,
      agentRunning: agentRunIsBlocking,
      agentExecuting: agentRunIsExecuting,
      hasPendingDecisions: false,
      hasExecutionHistory: (executionHistory?.length ?? 0) > 0,
      hasTerminalExecution: (executionHistory ?? []).some(
        (e) => e.status === "completed" || e.status === "failed" || e.status === "canceled" || e.status === "needs_fixup",
      ),
    });
    return clientActions;
  }, [item, blockingMap, agentRunIsBlocking, agentRunIsExecuting, executionHistory]);

  const isLocked = itemActions?.locked ?? false;
  const isTerminal = itemActions?.terminal ?? false;

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
    nextAction,

    reviewRounds: reviewRounds ?? [],
    isGatheringEvidence,
    isAwaitingManualReview,

    archiveTargets,

    // Computed
    depRelations,
    itemActions,
    reqModuleMap,
    targetIdSet,
    targetScenarios,

    isLocked,
    isTerminal,

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

    updateDescription: (description: string) => descriptionMutation.mutateAsync(description),
    isUpdatingDescription: descriptionMutation.isPending,
    updateDescriptionError: descriptionMutation.error instanceof Error ? descriptionMutation.error.message : null,
    resetDescriptionMutation: () => descriptionMutation.reset(),

    deleteItem: () => deleteMutation.mutate(),
    isDeleting: deleteMutation.isPending,
    deleteError,
    resetDeleteMutation: () => deleteMutation.reset(),

    isArchiving: archiveMutation.isPending || unarchiveMutation.isPending,
    archiveError,


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
      description: descriptionMutation,
      delete: deleteMutation,
      archiveMutation,
      unarchiveMutation,
      updateReqs: updateReqsMutation,
      createModule: createModuleMutation,
      updateModuleMeta: updateModuleMetaMutation,
      deleteModule: deleteModuleMutation,
      createTarget: createTargetMutation,
      updateTarget: updateTargetMutation,
      deleteTarget: deleteTargetMutation,
      batchReview: batchReviewMutation,
      fileAction: fileActionMutation,
    },
  } as const;
}
