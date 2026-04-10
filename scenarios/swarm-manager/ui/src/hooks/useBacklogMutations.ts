import { useQueryClient, useMutation } from "@tanstack/react-query";
import { backlogService } from "../services";
import type {
  ArchiveRequirementRecord,
  ArchiveTargetFormValues,
  BacklogFile,
  BacklogKind,
  BacklogStatus,
  ModuleFormValues,
  ReviewUpdate,
} from "../types";
import { useBacklogStore } from "../stores";

export type FileActionType = "rename" | "move" | "copy" | "delete";

export interface UseBacklogMutationsOptions {
  backlogKind: BacklogKind | null;
  name: string | undefined;
  refetchFiles: () => unknown;
  refetchWorkshopRounds: () => unknown;
}

export function useBacklogMutations({
  backlogKind,
  name,
  refetchFiles,
  refetchWorkshopRounds,
}: UseBacklogMutationsOptions) {
  const queryClient = useQueryClient();
  const upsertItem = useBacklogStore((state) => state.upsertItem);
  const removeItem = useBacklogStore((state) => state.removeItem);

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

  const archiveMutation = useMutation({
    mutationFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.archiveItem(backlogKind, name);
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
      upsertItem(updatedItem);
    },
  });

  const unarchiveMutation = useMutation({
    mutationFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.unarchiveItem(backlogKind, name);
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
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
    mutationFn: ({ mode, prompt, contextPaths, contextTargetIds, contextRequirementIds, confirm, force }: {
      mode?: string;
      prompt: string;
      contextPaths?: string[];
      contextTargetIds?: string[];
      contextRequirementIds?: string[];
      confirm?: boolean;
      force?: boolean;
    }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.research(backlogKind, name, {
        mode,
        prompt,
        contextPaths,
        contextTargetIds,
        contextRequirementIds,
        confirm,
        force,
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

  const workshopResetMutation = useMutation({
    mutationFn: async () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.workshopReset(backlogKind, name);
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
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

  const updateError = updateMutation.isError
    ? updateMutation.error instanceof Error ? updateMutation.error.message : "Failed to update backlog item. Please try again."
    : null;
  const deleteError = deleteMutation.isError
    ? deleteMutation.error instanceof Error ? deleteMutation.error.message : "Failed to delete backlog item. Please try again."
    : null;
  const agentErrorMsg = agentMutation.isError
    ? agentMutation.error instanceof Error ? agentMutation.error.message : "Failed to start the agent. Make sure agent-manager is running."
    : null;

  const archiveError = archiveMutation.isError
    ? archiveMutation.error instanceof Error ? archiveMutation.error.message : "Failed to archive item. Please try again."
    : null;

  return {
    updateMutation,
    statusMutation,
    depStatusMutation,
    acceptanceGlobMutation,
    archiveMutation,
    unarchiveMutation,
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
    batchReviewMutation,
    workshopDeleteRoundMutation,
    workshopResetMutation,
    fileActionMutation,

    updateError,
    archiveError,
    deleteError,
    agentErrorMsg,

    invalidateFiles: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
    },
    invalidateItem: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
    },
  };
}
