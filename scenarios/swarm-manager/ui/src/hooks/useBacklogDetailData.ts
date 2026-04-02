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
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import {
  defaultQueryOptions,
  getItemActions,
  scenariosFromGlobs,
} from "../lib";
import type { ItemActions } from "../lib/backlog-queue-utils";
import { computeDependencyRelations } from "../lib/backlog-queue-utils";
import { parseWorkshopRound, WORKSHOP_FILE_PATHS, findBacklogFileByPath } from "../lib/workshop-files";
import { buildReadinessData } from "../lib/maturity";
import type { ReadinessIndicatorData } from "../lib/maturity";
import { backlogService, executionService } from "../services";
import type {
  ArchiveRequirementRecord,
  ArchiveTargetFormValues,
  BacklogFile,
  BacklogKind,
  BacklogStatus,
  ExecutionRecord,
  ModuleFormValues,
  ResearchResponse,
  ReviewUpdate,
} from "../types";
import type { BacklogItem, MaturityItemSummary, WorkshopRound } from "../types/domain";
import { useBacklogStore } from "../stores";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/** How often to poll for fresh data while an agent run is active (ms). */
const AGENT_RUN_REFRESH_MS = 6000;

type FileActionType = "rename" | "move" | "copy" | "delete";

// ---------------------------------------------------------------------------
// Options interface
// ---------------------------------------------------------------------------

export interface UseBacklogDetailDataOptions {
  backlogKind: BacklogKind | null;
  name: string | undefined;
  agentRunIsActive: boolean;
}

// ---------------------------------------------------------------------------
// Hook implementation
// ---------------------------------------------------------------------------

export function useBacklogDetailData({
  backlogKind,
  name,
  agentRunIsActive,
}: UseBacklogDetailDataOptions) {
  const queryClient = useQueryClient();
  const upsertItem = useBacklogStore((state) => state.upsertItem);
  const allBacklogItems = useBacklogStore((state) => state.items);
  const removeItem = useBacklogStore((state) => state.removeItem);

  const cachedItem = useMemo(
    () => allBacklogItems.find((i) => i.kind === backlogKind && i.name === name),
    [allBacklogItems, backlogKind, name],
  );

  // -----------------------------------------------------------------------
  // Queries
  // -----------------------------------------------------------------------

  const {
    data: item,
    isLoading: isLoadingItem,
    error: itemError,
    refetch: refetchItem,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.get(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    placeholderData: cachedItem,
    ...defaultQueryOptions,
  });

  const spawnRef = item ? `${item.kind}/${item.name}` : "";
  const { data: spawnedItems } = useQuery({
    queryKey: ["backlog", "spawned-from", spawnRef],
    queryFn: () => backlogService.listBySpawnedFrom(spawnRef),
    enabled: !!spawnRef,
  });

  const {
    data: files,
    isLoading: isLoadingFiles,
    error: filesQueryError,
    refetch: refetchFiles,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "files"],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getFiles(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const { data: executionHistory } = useQuery({
    queryKey: ["executions", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return executionService.list({ backlogKind: backlogKind as BacklogKind, backlogName: name });
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: 10_000,
  });

  // Workshop round files
  const workshopDir = useMemo(
    () => findBacklogFileByPath(files ?? [], WORKSHOP_FILE_PATHS.workshopDir.replace(/\/$/, "")),
    [files],
  );
  const workshopRoundPaths = useMemo(() => {
    if (!workshopDir?.children) return [];
    return workshopDir.children
      .filter((f) => f.type === "file" && /^round-\d+\.json$/.test(f.name))
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true }))
      .map((f) => f.path);
  }, [workshopDir]);

  const {
    data: workshopRoundContents,
    refetch: refetchWorkshopRounds,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "workshop-rounds", workshopRoundPaths],
    queryFn: async () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const contents = await Promise.all(
        workshopRoundPaths.map((p) => backlogService.getFileContent(backlogKind, name, p)),
      );
      return contents;
    },
    enabled: !!backlogKind && !!name && workshopRoundPaths.length > 0,
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const workshopRounds = useMemo(() => {
    if (!workshopRoundContents) return [];
    return workshopRoundContents
      .map((content) => parseWorkshopRound(content))
      .filter((r): r is { round: WorkshopRound; error?: string } => r.round !== null)
      .map((r) => r.round);
  }, [workshopRoundContents]);

  // Maturity / readiness
  const { data: maturitySummaryData } = useQuery({
    queryKey: ["backlog-maturity-summary"],
    queryFn: () => backlogService.getMaturitySummary(),
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const readinessData = useMemo<ReadinessIndicatorData | null>(() => {
    if (!maturitySummaryData || !backlogKind || !name) return null;
    const match = (maturitySummaryData.items ?? []).find(
      (i: MaturityItemSummary) => i.kind === backlogKind && i.name === name,
    );
    return match ? buildReadinessData(match) : null;
  }, [maturitySummaryData, backlogKind, name]);

  // Archive targets
  const { data: archiveTargets } = useQuery({
    queryKey: ["backlog", backlogKind, name, "archive-targets"],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getArchiveTargets(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    ...defaultQueryOptions,
  });

  // -----------------------------------------------------------------------
  // Computed values
  // -----------------------------------------------------------------------

  const depRelations = useMemo(
    () => item ? computeDependencyRelations(item, allBacklogItems) : { parents: [], children: [] },
    [item, allBacklogItems],
  );

  const deliverableLabel = backlogKind === "research" ? "Conclusion" : "Plan";
  const deliverableLabelLower = deliverableLabel.toLowerCase();
  const workshopActionLabel = workshopRounds.length > 0 ? "Next Round" : "Workshop";
  const isWorkshopFinalized = workshopRounds.some((r) => r.mode === "finalize")
    && !(readinessData?.pendingSynthesis ?? false);

  const itemActions: ItemActions | null = useMemo(() => {
    if (!item) return null;
    return getItemActions({
      item,
      allItems: allBacklogItems,
      readinessReady: readinessData ? readinessData.ready : null,
      pendingSynthesis: readinessData?.pendingSynthesis ?? false,
      agentRunning: agentRunIsActive,
      hasPendingDecisions: workshopRounds.some(
        (r) => r.items?.some((wi) => wi.type === "decision" && wi.selected == null),
      ),
      hasExecutionHistory: (executionHistory?.length ?? 0) > 0,
    });
  }, [item, allBacklogItems, readinessData, agentRunIsActive, workshopRounds, executionHistory]);

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

  // Merge functions for flagged items (returned as stable callbacks)
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
  // Mutations
  // -----------------------------------------------------------------------

  const archiveTargetsQueryKey = ["backlog", backlogKind, name, "archive-targets"];

  const updateMutation = useMutation({
    mutationFn: (values: {
      title: string;
      description: string;
      status: BacklogStatus;
      priority: number;
      tags: string[];
    }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.update(backlogKind, name, {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        tags: values.tags,
      });
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      upsertItem(updatedItem);
    },
  });

  const statusMutation = useMutation({
    mutationFn: (newStatus: BacklogStatus) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.update(backlogKind, name, { status: newStatus });
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      upsertItem(updatedItem);
    },
  });

  const depStatusMutation = useMutation({
    mutationFn: ({ kind, depName, newStatus }: { kind: string; depName: string; newStatus: BacklogStatus }) =>
      backlogService.update(kind as BacklogKind, depName, { status: newStatus }),
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
    },
  });

  const acceptanceGlobMutation = useMutation({
    mutationFn: (values: { acceptanceAllow: string[]; acceptanceDeny: string[] }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.update(backlogKind, name, {
        acceptanceAllow: values.acceptanceAllow,
        acceptanceDeny: values.acceptanceDeny,
      });
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      upsertItem(updatedItem);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.delete(backlogKind, name);
    },
    onSuccess: () => {
      if (backlogKind && name) {
        removeItem(name, backlogKind);
      }
    },
  });

  const agentMutation = useMutation({
    mutationFn: ({ mode, prompt, contextPaths, contextTargetIds, contextRequirementIds }: {
      mode?: string;
      prompt: string;
      contextPaths?: string[];
      contextTargetIds?: string[];
      contextRequirementIds?: string[];
    }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.research(backlogKind, name, {
        mode,
        prompt,
        contextPaths,
        contextTargetIds,
        contextRequirementIds,
      });
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      void queryClient.invalidateQueries({ queryKey: ["backlog-maturity-summary"] });
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    },
  });

  const workshopSaveMutation = useMutation({
    mutationFn: async ({ roundNumber, content }: { roundNumber: number; content: string }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.workshopSave(backlogKind, name, roundNumber, content);
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "workshop-rounds"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-maturity-summary"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
      void refetchFiles();
      void refetchWorkshopRounds();
    },
  });

  const updateReqsMutation = useMutation({
    mutationFn: ({ moduleId, requirements }: { moduleId: string; requirements: ArchiveRequirementRecord[] }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateModuleRequirements(backlogKind, name, moduleId, requirements);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const createModuleMutation = useMutation({
    mutationFn: (payload: ModuleFormValues & { position?: number }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.createModule(backlogKind, name, payload);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const updateModuleMetaMutation = useMutation({
    mutationFn: ({ moduleId, payload }: { moduleId: string; payload: { title: string; description: string } }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateModuleMeta(backlogKind, name, moduleId, payload);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const deleteModuleMutation = useMutation({
    mutationFn: (moduleId: string) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.deleteModule(backlogKind, name, moduleId);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const createTargetMutation = useMutation({
    mutationFn: (target: ArchiveTargetFormValues) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.createArchiveTarget(backlogKind, name, target);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const updateTargetMutation = useMutation({
    mutationFn: ({ targetId, target }: { targetId: string; target: ArchiveTargetFormValues }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateArchiveTarget(backlogKind, name, targetId, target);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const deleteTargetMutation = useMutation({
    mutationFn: (targetId: string) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.deleteArchiveTarget(backlogKind, name, targetId);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const batchReviewMutation = useMutation({
    mutationFn: (items: ReviewUpdate[]) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.batchReview(backlogKind, name, items);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey });
    },
  });

  const workshopDeleteRoundMutation = useMutation({
    mutationFn: async ({ roundNumber }: { roundNumber: number }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.workshopDeleteRound(backlogKind, name, roundNumber);
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "workshop-rounds"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-maturity-summary"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
      void refetchFiles();
      void refetchWorkshopRounds();
    },
  });

  const fileActionMutation = useMutation({
    mutationFn: async ({ action, target, destinationPath }: { action: FileActionType; target: BacklogFile; destinationPath?: string }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      if (action === "rename") {
        if (!destinationPath) throw new Error("Destination path is required");
        return backlogService.renameFile(backlogKind, name, target.path, destinationPath);
      }
      if (action === "move") {
        if (!destinationPath) throw new Error("Destination path is required");
        return backlogService.moveFile(backlogKind, name, target.path, destinationPath);
      }
      if (action === "copy") {
        if (!destinationPath) throw new Error("Destination path is required");
        return backlogService.copyFile(backlogKind, name, target.path, destinationPath);
      }
      return backlogService.deleteFile(backlogKind, name, target.path);
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
    },
  });

  // -----------------------------------------------------------------------
  // Convenience error derivations
  // -----------------------------------------------------------------------

  const updateError = updateMutation.isError
    ? updateMutation.error instanceof Error ? updateMutation.error.message : "Failed to update backlog item. Please try again."
    : null;
  const deleteError = deleteMutation.isError
    ? deleteMutation.error instanceof Error ? deleteMutation.error.message : "Failed to delete backlog item. Please try again."
    : null;
  const agentErrorMsg = agentMutation.isError
    ? agentMutation.error instanceof Error ? agentMutation.error.message : "Failed to start the agent. Make sure agent-manager is running."
    : null;

  const filesError = filesQueryError instanceof Error ? filesQueryError : null;

  // -----------------------------------------------------------------------
  // Return
  // -----------------------------------------------------------------------

  return {
    // Query data
    item,
    isLoadingItem,
    itemError: itemError as Error | null,
    refetchItem: () => void refetchItem(),

    spawnedItems,

    files,
    isLoadingFiles,
    filesError,
    refetchFiles: () => void refetchFiles(),

    executionHistory,

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

    runAgent: (payload: { mode?: string; prompt: string; contextPaths?: string[]; contextTargetIds?: string[]; contextRequirementIds?: string[] }) =>
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

    fileAction: (args: { action: FileActionType; target: BacklogFile; destinationPath?: string }) =>
      fileActionMutation.mutate(args),
    isFileActionPending: fileActionMutation.isPending,

    // Invalidation helpers
    invalidateFiles: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
    },
    invalidateItem: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
    },

    // Raw mutations for advanced wiring (onSuccess with closures in the page)
    _mutations: {
      update: updateMutation,
      status: statusMutation,
      acceptanceGlob: acceptanceGlobMutation,
      delete: deleteMutation,
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
      fileAction: fileActionMutation,
    },
  } as const;
}
