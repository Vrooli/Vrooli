import { useQueryClient } from "@tanstack/react-query";
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
import { useActionMutation } from "./useActionMutation";

export type FileActionType = "rename" | "move" | "copy" | "delete";

export interface UseBacklogMutationsOptions {
  backlogKind: BacklogKind | null;
  name: string | undefined;
}

/**
 * Every mutation on this page acts on one backlog item, so every one of them
 * needs the same guard. Hoisting it removes seventeen copies of the same
 * three-line check and gives the failure a single wording.
 */
function requireTarget(backlogKind: BacklogKind | null, name: string | undefined): { kind: BacklogKind; name: string } {
  if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
  return { kind: backlogKind, name };
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
  const target = () => requireTarget(backlogKind, name);

  const archiveTargetsQueryKey = ["backlog", backlogKind, name, "archive-targets"];
  const invalidateNextAction = () => invalidate(["backlog", backlogKind, name, "next-action"]);

  /**
   * Refresh set shared by every mutation that returns an updated item: the
   * item's own query, its next-action projection, and the store copy the
   * lists render from. Missing any one of these leaves a stale surface.
   */
  const onItemUpdated = (updatedItem: Parameters<typeof upsertItem>[0], alsoList = false) => {
    if (!backlogKind || !name) return;
    invalidate(["backlog", backlogKind, name]);
    invalidateNextAction();
    if (alsoList) invalidate(["backlog-list"]);
    upsertItem(updatedItem);
  };

  const updateMutation = useActionMutation({
    mutationFn: (values: {
      title: string;
      description: string;
      status: BacklogStatus;
      priority: number;
      tags: string[];
    }) => {
      const { kind, name: itemName } = target();
      return backlogService.update(kind, itemName, {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        tags: values.tags,
      });
    },
    errorMessage: "Couldn't save this item",
    // The editor renders `updateError` inline so the operator keeps their edits.
    silentError: true,
    source: "useBacklogMutations.update",
    onSuccess: (updatedItem) => onItemUpdated(updatedItem),
  });

  const statusMutation = useActionMutation({
    mutationFn: (newStatus: BacklogStatus) => {
      const { kind, name: itemName } = target();
      return backlogService.update(kind, itemName, { status: newStatus });
    },
    errorMessage: "Couldn't change the status",
    successMessage: (_item, newStatus) => `Status set to ${newStatus.replaceAll("_", " ")}`,
    source: "useBacklogMutations.status",
    onSuccess: (updatedItem) => onItemUpdated(updatedItem),
  });

  const depStatusMutation = useActionMutation({
    mutationFn: ({ kind, depName, newStatus }: { kind: string; depName: string; newStatus: BacklogStatus }) =>
      backlogService.update(kind as BacklogKind, depName, { status: newStatus }),
    errorMessage: "Couldn't change that dependency's status",
    successMessage: (_item, { depName, newStatus }) => `${depName} set to ${newStatus.replaceAll("_", " ")}`,
    source: "useBacklogMutations.depStatus",
    onSuccess: () => {
      if (!backlogKind || !name) return;
      invalidate(["backlog", backlogKind, name]);
      invalidateNextAction();
      invalidate(["backlog-list"]);
    },
  });

  const acceptanceGlobMutation = useActionMutation({
    mutationFn: (values: { acceptanceAllow: string[]; acceptanceDeny: string[] }) => {
      const { kind, name: itemName } = target();
      return backlogService.update(kind, itemName, {
        acceptanceAllow: values.acceptanceAllow,
        acceptanceDeny: values.acceptanceDeny,
      });
    },
    errorMessage: "Couldn't save the acceptance globs",
    successMessage: "Acceptance globs saved",
    source: "useBacklogMutations.acceptanceGlobs",
    onSuccess: (updatedItem) => onItemUpdated(updatedItem),
  });

  const descriptionMutation = useActionMutation({
    mutationFn: (description: string) => {
      const { kind, name: itemName } = target();
      return backlogService.update(kind, itemName, { description });
    },
    errorMessage: "Couldn't save the description",
    silentError: true,
    source: "useBacklogMutations.description",
    onSuccess: (updatedItem) => onItemUpdated(updatedItem),
  });

  const archiveMutation = useActionMutation({
    mutationFn: () => {
      const { kind, name: itemName } = target();
      return backlogService.archiveItem(kind, itemName);
    },
    errorMessage: "Couldn't archive this item",
    successMessage: "Item archived",
    // Raised from a confirm dialog that shows `archiveError` in place.
    silentError: true,
    source: "useBacklogMutations.archive",
    onSuccess: (updatedItem) => onItemUpdated(updatedItem, true),
  });

  const unarchiveMutation = useActionMutation({
    mutationFn: () => {
      const { kind, name: itemName } = target();
      return backlogService.unarchiveItem(kind, itemName);
    },
    errorMessage: "Couldn't restore this item",
    successMessage: "Item restored",
    source: "useBacklogMutations.unarchive",
    onSuccess: (updatedItem) => onItemUpdated(updatedItem, true),
  });

  const deleteMutation = useActionMutation({
    mutationFn: () => {
      const { kind, name: itemName } = target();
      return backlogService.delete(kind, itemName);
    },
    errorMessage: "Couldn't delete this item",
    successMessage: "Item deleted",
    silentError: true,
    source: "useBacklogMutations.delete",
    onSuccess: () => {
      if (backlogKind && name) removeItem(name, backlogKind);
    },
  });

  const updateReqsMutation = useActionMutation({
    mutationFn: ({ moduleId, requirements }: { moduleId: string; requirements: ArchiveRequirementRecord[] }) => {
      const { kind, name: itemName } = target();
      return backlogService.updateModuleRequirements(kind, itemName, moduleId, requirements);
    },
    errorMessage: "Couldn't save those requirements",
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.updateReqs",
  });

  const createModuleMutation = useActionMutation({
    mutationFn: (payload: ModuleFormValues & { position?: number }) => {
      const { kind, name: itemName } = target();
      return backlogService.createModule(kind, itemName, payload);
    },
    errorMessage: "Couldn't create that module",
    successMessage: "Module created",
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.createModule",
  });

  const updateModuleMetaMutation = useActionMutation({
    mutationFn: ({ moduleId, payload }: { moduleId: string; payload: { title: string; description: string } }) => {
      const { kind, name: itemName } = target();
      return backlogService.updateModuleMeta(kind, itemName, moduleId, payload);
    },
    errorMessage: "Couldn't save that module",
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.updateModuleMeta",
  });

  const deleteModuleMutation = useActionMutation({
    mutationFn: (moduleId: string) => {
      const { kind, name: itemName } = target();
      return backlogService.deleteModule(kind, itemName, moduleId);
    },
    errorMessage: "Couldn't delete that module",
    successMessage: "Module deleted",
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.deleteModule",
  });

  const createTargetMutation = useActionMutation({
    mutationFn: (archiveTarget: ArchiveTargetFormValues) => {
      const { kind, name: itemName } = target();
      return backlogService.createArchiveTarget(kind, itemName, archiveTarget);
    },
    errorMessage: "Couldn't create that target",
    successMessage: "Target created",
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.createTarget",
  });

  const updateTargetMutation = useActionMutation({
    mutationFn: ({ targetId, target: archiveTarget }: { targetId: string; target: ArchiveTargetFormValues }) => {
      const { kind, name: itemName } = target();
      return backlogService.updateArchiveTarget(kind, itemName, targetId, archiveTarget);
    },
    errorMessage: "Couldn't save that target",
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.updateTarget",
  });

  const deleteTargetMutation = useActionMutation({
    mutationFn: (targetId: string) => {
      const { kind, name: itemName } = target();
      return backlogService.deleteArchiveTarget(kind, itemName, targetId);
    },
    errorMessage: "Couldn't delete that target",
    successMessage: "Target deleted",
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.deleteTarget",
  });

  const batchReviewMutation = useActionMutation({
    mutationFn: (items: ReviewUpdate[]) => {
      const { kind, name: itemName } = target();
      return backlogService.batchReview(kind, itemName, items);
    },
    errorMessage: "Couldn't save those review decisions",
    successMessage: (_result, items) =>
      items.length === 1 ? "Review decision saved" : `${items.length} review decisions saved`,
    invalidateKeys: [archiveTargetsQueryKey],
    source: "useBacklogMutations.batchReview",
  });

  const fileActionMutation = useActionMutation({
    mutationFn: async ({ action, target: file, destinationPath }: { action: FileActionType; target: BacklogFile; destinationPath?: string }) => {
      const { kind, name: itemName } = target();
      if (action === "delete") return backlogService.deleteFile(kind, itemName, file.path);
      if (!destinationPath) throw new Error("Destination path is required");
      if (action === "rename") return backlogService.renameFile(kind, itemName, file.path, destinationPath);
      if (action === "move") return backlogService.moveFile(kind, itemName, file.path, destinationPath);
      return backlogService.copyFile(kind, itemName, file.path, destinationPath);
    },
    errorMessage: "Couldn't complete that file action",
    successMessage: (_result, { action, target: file }) => `${FILE_ACTION_PAST_TENSE[action]} ${file.path}`,
    source: "useBacklogMutations.fileAction",
    onSuccess: () => {
      if (!backlogKind || !name) return;
      invalidate(["backlog", backlogKind, name, "files"]);
    },
  });

  // These three are rendered inline by the editor and the confirm dialogs,
  // which is why their mutations opt out of the toast.
  const updateError = updateMutation.errorDescription?.message ?? null;
  const deleteError = deleteMutation.errorDescription?.message ?? null;
  const archiveError = archiveMutation.errorDescription?.message ?? null;

  return {
    updateMutation,
    statusMutation,
    depStatusMutation,
    acceptanceGlobMutation,
    descriptionMutation,
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

/** Confirmation wording for file operations, so the toast reports what happened. */
const FILE_ACTION_PAST_TENSE: Record<FileActionType, string> = {
  rename: "Renamed",
  move: "Moved",
  copy: "Copied",
  delete: "Deleted",
};
