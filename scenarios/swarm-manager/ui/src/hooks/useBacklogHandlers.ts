/**
 * useBacklogHandlers
 *
 * Encapsulates all useCallback handlers for BacklogDetailsPage, keeping the
 * page component focused on layout and composition.
 *
 * Dialog/selection state is read from backlog-detail-ui-store via getState()
 * (stable, no re-renders).
 */

import { useCallback } from "react";
import { type SetURLSearchParams } from "react-router-dom";
import { findRequirementGroup } from "../lib/archive-utils";
import { useBacklogDetailUIStore } from "../stores";
import type { useBacklogDetailData } from "./useBacklogDetailData";
import type {
  ArchiveRequirement,
  ArchiveRequirementRecord,
  ArchiveTarget,
  ArchiveTargetFormValues,
  BacklogFile,
  BacklogKind,
  BacklogStatus,
  ModuleFormValues,
  ReviewAction,
  ReviewUpdate,
} from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type DetailsTab = "info" | "prompt" | "files";
type FileActionType = "rename" | "move" | "copy" | "delete";

/** The subset of useBacklogDetailData return used by handlers. */
type BacklogDetailData = ReturnType<typeof useBacklogDetailData>;

export interface UseBacklogHandlersOptions {
  data: BacklogDetailData;
  backlogKind: BacklogKind | null;
  name: string | undefined;
  // UI state setters that remain local (URL-synced)
  setSelectedFile: (v: BacklogFile | null) => void;
  setActiveTab: (v: DetailsTab) => void;
  setSearchParams: SetURLSearchParams;
  selectedFile: BacklogFile | null;
  // Navigation
  closeDetail: () => void;
  refreshActivities: (force: boolean) => Promise<void>;
}

const remapSelectedPath = (
  currentPath: string,
  target: BacklogFile,
  destinationPath: string,
): string | null => {
  if (target.type === "file") {
    return currentPath === target.path ? destinationPath : currentPath;
  }
  const prefix = `${target.path}/`;
  if (currentPath === target.path) return destinationPath;
  if (currentPath.startsWith(prefix)) {
    return `${destinationPath}/${currentPath.slice(prefix.length)}`;
  }
  return currentPath;
};

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useBacklogHandlers(opts: UseBacklogHandlersOptions) {
  const {
    data,
    backlogKind,
    name,
    setSelectedFile,
    setActiveTab,
    setSearchParams,
    selectedFile,
    closeDetail,
    refreshActivities,
  } = opts;

  const { _mutations, archiveTargets, targetIdSet, reqModuleMap, deliverableLabelLower } = data;

  // --- Item CRUD ---

  const handleUpdateItem = useCallback((values: {
    title: string;
    description: string;
    status: BacklogStatus;
    priority: number;
    tags: string[];
  }) => {
    _mutations.update.mutate(values, {
      onSuccess: () => { useBacklogDetailUIStore.getState().closeEdit(); },
    });
  }, [_mutations.update]);

  const handleAcceptanceGlobSave = useCallback((allow: string[], deny: string[]) => {
    _mutations.acceptanceGlob.mutate({ acceptanceAllow: allow, acceptanceDeny: deny }, {
      onSuccess: () => { useBacklogDetailUIStore.getState().closeGlob(); },
    });
  }, [_mutations.acceptanceGlob]);

  const handleDeleteConfirm = useCallback(() => {
    _mutations.delete.mutate(undefined, {
      onSuccess: () => { closeDetail(); },
    });
  }, [_mutations.delete, closeDetail]);

  // --- Agent ---

  const handleAgentSubmit = useCallback((payload: {
    mode?: string;
    prompt: string;
    contextPaths?: string[];
    contextTargetIds?: string[];
    contextRequirementIds?: string[];
  }) => {
    _mutations.agent.mutate(payload, {
      onSuccess: () => {
        useBacklogDetailUIStore.getState().closeAgent();
        void refreshActivities(true);
      },
    });
  }, [_mutations.agent, refreshActivities]);

  // --- Workshop ---

  const handleSaveRound = useCallback((roundNumber: number, content: string) => {
    _mutations.workshopSave.mutate({ roundNumber, content }, {
      onSuccess: (result) => {
        if (result.autoAdvance?.triggered && result.autoAdvance?.runId) {
          void refreshActivities(true);
        }
      },
    });
  }, [_mutations.workshopSave, refreshActivities]);

  const handleDeleteWorkshopRound = useCallback(() => {
    const roundToDelete = useBacklogDetailUIStore.getState().roundToDelete;
    if (roundToDelete !== null) {
      _mutations.workshopDeleteRound.mutate({ roundNumber: roundToDelete }, {
        onSuccess: () => { useBacklogDetailUIStore.getState().setRoundToDelete(null); },
      });
    }
  }, [_mutations.workshopDeleteRound]);

  const startWorkshopMode = useCallback((mode: "workshop" | "finalize", prompt: string) => {
    if (!backlogKind || !name) return;
    handleAgentSubmit({ mode, prompt });
  }, [backlogKind, name, handleAgentSubmit]);

  const handleRunWorkshop = useCallback(() => {
    startWorkshopMode("workshop", "Run the next workshop round for this backlog item.");
  }, [startWorkshopMode]);

  const handleFinalizeWorkshop = useCallback(() => {
    startWorkshopMode(
      "finalize",
      `Finalize the latest workshop answers into the ${deliverableLabelLower} for this backlog item.`,
    );
  }, [deliverableLabelLower, startWorkshopMode]);

  // --- Files ---

  const handleFileSelect = useCallback((file: BacklogFile) => {
    if (file.type === "file") {
      setSelectedFile(file);
      setActiveTab("files");
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("file", file.path);
        return next;
      }, { replace: true });
    }
  }, [setSelectedFile, setActiveTab, setSearchParams]);

  const handleUploadComplete = useCallback(() => {
    data.invalidateFiles();
  }, [data]);

  const handleFileAction = useCallback((action: FileActionType, target: BacklogFile, destinationPath?: string) => {
    _mutations.fileAction.mutate({ action, target, destinationPath }, {
      onSuccess: (_result, variables) => {
        const currentSelectedPath = selectedFile?.path;
        if (!currentSelectedPath) return;

        if (variables.action === "delete") {
          const affectedPath =
            currentSelectedPath === variables.target.path ||
            (variables.target.type === "directory" && currentSelectedPath.startsWith(`${variables.target.path}/`));
          if (affectedPath) {
            setSelectedFile(null);
            setSearchParams((prev) => {
              const next = new URLSearchParams(prev);
              next.delete("file");
              return next;
            }, { replace: true });
          }
          return;
        }

        if (!variables.destinationPath) return;
        const remapped = remapSelectedPath(currentSelectedPath, variables.target, variables.destinationPath);
        if (!remapped || remapped === currentSelectedPath) return;
        setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          next.set("file", remapped);
          return next;
        }, { replace: true });
      },
    });
  }, [_mutations.fileAction, selectedFile?.path, setSelectedFile, setSearchParams]);

  // --- Review ---

  const handleReviewAction = useCallback((id: string, _type: "target" | "requirement", action: ReviewAction) => {
    let reviewItem: ReviewUpdate;
    if (targetIdSet.has(id)) {
      reviewItem = { id, type: "target", ...action };
    } else {
      const moduleId = reqModuleMap.get(id);
      if (!moduleId) return;
      reviewItem = { id, type: "requirement", module_id: moduleId, ...action };
    }
    data.batchReview([reviewItem]);
  }, [targetIdSet, reqModuleMap, data]);

  const handleBulkApprove = useCallback(() => {
    const { selectedTargetIds, selectedRequirementIds } = useBacklogDetailUIStore.getState();
    const items: ReviewUpdate[] = [];
    for (const id of selectedTargetIds) {
      items.push({ id, type: "target", review_status: "approved" });
    }
    for (const id of selectedRequirementIds) {
      const moduleId = reqModuleMap.get(id);
      if (moduleId) items.push({ id, type: "requirement", module_id: moduleId, review_status: "approved" });
    }
    if (items.length > 0) data.batchReview(items);
  }, [reqModuleMap, data]);

  const handleBulkFlag = useCallback(() => {
    const { selectedTargetIds, selectedRequirementIds } = useBacklogDetailUIStore.getState();
    const items: ReviewUpdate[] = [];
    for (const id of selectedTargetIds) {
      items.push({ id, type: "target", review_status: "flagged" });
    }
    for (const id of selectedRequirementIds) {
      const moduleId = reqModuleMap.get(id);
      if (moduleId) items.push({ id, type: "requirement", module_id: moduleId, review_status: "flagged" });
    }
    if (items.length > 0) data.batchReview(items);
  }, [reqModuleMap, data]);

  // --- Targets ---

  const handleCreateTarget = useCallback(() => {
    useBacklogDetailUIStore.getState().openTargetCreate();
  }, []);

  const handleEditTarget = useCallback((target: ArchiveTarget) => {
    useBacklogDetailUIStore.getState().openTargetEdit(target);
  }, []);

  const handleDeleteTarget = useCallback((targetId: string) => {
    if (!window.confirm(`Delete target "${targetId}"?`)) return;
    data.deleteTarget(targetId);
  }, [data]);

  const handleTargetDialogSubmit = useCallback((values: ArchiveTargetFormValues) => {
    const { targetDialog } = useBacklogDetailUIStore.getState();
    if (targetDialog.mode === "create") {
      _mutations.createTarget.mutate(values, {
        onSuccess: () => { useBacklogDetailUIStore.getState().closeTargetDialog(); },
      });
    } else if (targetDialog.editing) {
      _mutations.updateTarget.mutate({ targetId: targetDialog.editing.id, target: values }, {
        onSuccess: () => { useBacklogDetailUIStore.getState().closeTargetDialog(); },
      });
    }
  }, [_mutations.createTarget, _mutations.updateTarget]);

  // --- Requirements ---

  const handleCreateRequirement = useCallback((groupId: string) => {
    useBacklogDetailUIStore.getState().openReqCreate(groupId);
  }, []);

  const handleEditRequirement = useCallback((groupId: string, requirement: ArchiveRequirement) => {
    useBacklogDetailUIStore.getState().openReqEdit({ groupId, req: requirement as ArchiveRequirementRecord });
  }, []);

  const handleDeleteRequirement = useCallback((groupId: string, requirementId: string) => {
    if (!window.confirm(`Delete requirement "${requirementId}"?`)) return;
    if (!archiveTargets) return;
    const group = findRequirementGroup(archiveTargets.requirements, groupId);
    if (!group) return;
    const updated = group.requirements.filter((r) => r.id !== requirementId) as ArchiveRequirementRecord[];
    data.updateRequirements({ moduleId: groupId, requirements: updated });
  }, [archiveTargets, data]);

  const handleReorderRequirement = useCallback((groupId: string, requirementId: string, direction: "up" | "down") => {
    if (!archiveTargets) return;
    const group = findRequirementGroup(archiveTargets.requirements, groupId);
    if (!group) return;
    const reqs = [...group.requirements] as ArchiveRequirementRecord[];
    const idx = reqs.findIndex((r) => r.id === requirementId);
    if (idx < 0) return;
    const swapIdx = direction === "up" ? idx - 1 : idx + 1;
    if (swapIdx < 0 || swapIdx >= reqs.length) return;
    const tmp = reqs[idx] as typeof reqs[number];
    reqs[idx] = reqs[swapIdx] as typeof reqs[number];
    reqs[swapIdx] = tmp;
    data.updateRequirements({ moduleId: groupId, requirements: reqs });
  }, [archiveTargets, data]);

  const handleReqDialogSubmit = useCallback((values: ArchiveRequirementRecord) => {
    const { reqDialog } = useBacklogDetailUIStore.getState();
    if (!reqDialog.editing || !archiveTargets) return;
    const { groupId } = reqDialog.editing;
    const group = findRequirementGroup(archiveTargets.requirements, groupId);
    if (!group) return;
    let updated: ArchiveRequirementRecord[];
    if (reqDialog.editing?.req) {
      updated = group.requirements.map((r) => r.id === values.id ? values : r) as ArchiveRequirementRecord[];
    } else {
      updated = [...group.requirements as ArchiveRequirementRecord[], values];
    }
    _mutations.updateReqs.mutate({ moduleId: groupId, requirements: updated }, {
      onSuccess: () => { useBacklogDetailUIStore.getState().closeReqDialog(); },
    });
  }, [archiveTargets, _mutations.updateReqs]);

  // --- Modules ---

  const handleCreateModule = useCallback(() => {
    useBacklogDetailUIStore.getState().openModuleCreate();
  }, []);

  const handleEditModule = useCallback((groupId: string) => {
    useBacklogDetailUIStore.getState().openModuleEdit(groupId);
  }, []);

  const handleDeleteModule = useCallback((groupId: string) => {
    if (!window.confirm(`Delete module "${groupId}" and all its requirements?`)) return;
    data.deleteModule(groupId);
  }, [data]);

  const handleModuleDialogSubmit = useCallback((values: ModuleFormValues) => {
    const { moduleDialog } = useBacklogDetailUIStore.getState();
    if (moduleDialog.mode === "create") {
      _mutations.createModule.mutate(values, {
        onSuccess: () => { useBacklogDetailUIStore.getState().closeModuleDialog(); },
      });
    } else if (moduleDialog.editing) {
      _mutations.updateModuleMeta.mutate({
        moduleId: moduleDialog.editing,
        payload: { title: values.title, description: values.description },
      }, {
        onSuccess: () => { useBacklogDetailUIStore.getState().closeModuleDialog(); },
      });
    }
  }, [_mutations.createModule, _mutations.updateModuleMeta]);

  // --- Selection toggles (delegated to store) ---

  const handleTargetToggle = useCallback((id: string) => {
    useBacklogDetailUIStore.getState().toggleTargetId(id);
  }, []);

  const handleRequirementToggle = useCallback((id: string) => {
    useBacklogDetailUIStore.getState().toggleRequirementId(id);
  }, []);

  return {
    handleUpdateItem,
    handleAcceptanceGlobSave,
    handleDeleteConfirm,
    handleAgentSubmit,
    handleSaveRound,
    handleDeleteWorkshopRound,
    handleRunWorkshop,
    handleFinalizeWorkshop,
    handleFileSelect,
    handleUploadComplete,
    handleFileAction,
    handleReviewAction,
    handleBulkApprove,
    handleBulkFlag,
    handleCreateTarget,
    handleEditTarget,
    handleDeleteTarget,
    handleTargetDialogSubmit,
    handleCreateRequirement,
    handleEditRequirement,
    handleDeleteRequirement,
    handleReorderRequirement,
    handleReqDialogSubmit,
    handleCreateModule,
    handleEditModule,
    handleDeleteModule,
    handleModuleDialogSubmit,
    handleTargetToggle,
    handleRequirementToggle,
  } as const;
}
