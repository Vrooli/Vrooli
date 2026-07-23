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
}

export function useBacklogMutations({
  backlogKind,
  name,
}: UseBacklogMutationsOptions) {
  const queryClient = useQueryClient();
  const upsertItem = useBacklogStore((state) => state.upsertItem);
  const removeItem = useBacklogStore((state) => state.removeItem);
  const invalidate = (queryKey: readonly unknown[]) => {
    void queryClient.invalidateQueries({ queryKey });
  };

  const archiveTargetsQueryKey = ["backlog", backlogKind, name, "archive-targets"];
  const invalidateNextAction = () => invalidate(["backlog", backlogKind, name, "next-action"]);

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
      invalidate(["backlog", backlogKind, name]);
      invalidateNextAction();
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
      invalidate(["backlog", backlogKind, name]);
      invalidateNextAction();
      upsertItem(updatedItem);
    },
  });

  const depStatusMutation = useMutation({
    mutationFn: ({ kind, depName, newStatus }: { kind: string; depName: string; newStatus: BacklogStatus }) =>
      backlogService.update(kind as BacklogKind, depName, { status: newStatus }),
    onSuccess: () => {
      if (!backlogKind || !name) return;
      invalidate(["backlog", backlogKind, name]);
      invalidateNextAction();
      invalidate(["backlog-list"]);
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
      invalidate(["backlog", backlogKind, name]);
      invalidateNextAction();
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
      invalidate(["backlog", backlogKind, name]);
      invalidateNextAction();
      invalidate(["backlog-list"]);
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
      invalidate(["backlog", backlogKind, name]);
      invalidateNextAction();
      invalidate(["backlog-list"]);
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

  const updateReqsMutation = useMutation({
    mutationFn: ({ moduleId, requirements }: { moduleId: string; requirements: ArchiveRequirementRecord[] }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateModuleRequirements(backlogKind, name, moduleId, requirements);
    },
    onSuccess: () => { invalidate(archiveTargetsQueryKey); },
  });

  const createModuleMutation = useMutation({
    mutationFn: (payload: ModuleFormValues & { position?: number }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.createModule(backlogKind, name, payload);
    },
    onSuccess: () => { invalidate(archiveTargetsQueryKey); },
  });

  const updateModuleMetaMutation = useMutation({
    mutationFn: ({ moduleId, payload }: { moduleId: string; payload: { title: string; description: string } }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateModuleMeta(backlogKind, name, moduleId, payload);
    },
    onSuccess: () => { invalidate(archiveTargetsQueryKey); },
  });

  const deleteModuleMutation = useMutation({
    mutationFn: (moduleId: string) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.deleteModule(backlogKind, name, moduleId);
    },
    onSuccess: () => { invalidate(archiveTargetsQueryKey); },
  });

  const createTargetMutation = useMutation({
    mutationFn: (target: ArchiveTargetFormValues) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.createArchiveTarget(backlogKind, name, target);
    },
    onSuccess: () => { invalidate(archiveTargetsQueryKey); },
  });

  const updateTargetMutation = useMutation({
    mutationFn: ({ targetId, target }: { targetId: string; target: ArchiveTargetFormValues }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateArchiveTarget(backlogKind, name, targetId, target);
    },
    onSuccess: () => { invalidate(archiveTargetsQueryKey); },
  });

  const deleteTargetMutation = useMutation({
    mutationFn: (targetId: string) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.deleteArchiveTarget(backlogKind, name, targetId);
    },
    onSuccess: () => { invalidate(archiveTargetsQueryKey); },
  });

  const batchReviewMutation = useMutation({
    mutationFn: (items: ReviewUpdate[]) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.batchReview(backlogKind, name, items);
    },
    onSuccess: () => {
      invalidate(archiveTargetsQueryKey);
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
      invalidate(["backlog", backlogKind, name, "files"]);
    },
  });

  const updateError = updateMutation.isError
    ? updateMutation.error instanceof Error ? updateMutation.error.message : "Failed to update backlog item. Please try again."
    : null;
  const deleteError = deleteMutation.isError
    ? deleteMutation.error instanceof Error ? deleteMutation.error.message : "Failed to delete backlog item. Please try again."
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
    updateReqsMutation,
    createModuleMutation,
    updateModuleMetaMutation,
    deleteModuleMutation,
    createTargetMutation,
    updateTargetMutation,
    deleteTargetMutation,
    batchReviewMutation,
    fileActionMutation,

    updateError,
    archiveError,
    deleteError,

    invalidateFiles: () => {
      if (!backlogKind || !name) return;
      invalidate(["backlog", backlogKind, name, "files"]);
    },
    invalidateItem: () => {
      if (!backlogKind || !name) return;
      invalidate(["backlog", backlogKind, name]);
    },
  };
}
