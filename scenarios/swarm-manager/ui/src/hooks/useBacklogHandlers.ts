/**
 * useBacklogHandlers
 *
 * Thin composition hook that merges CRUD, file, and structure handlers into a
 * single return object for BacklogDetailsPage. Each concern is implemented in
 * its own sub-hook:
 *
 *   - useBacklogCRUDHandlers  -- item CRUD, agent, workshop
 *   - useBacklogFileHandlers  -- file select, upload, rename/move/copy/delete
 *   - (remaining)             -- review, targets, requirements, modules, selection
 *
 * Dialog/selection state is read from backlog-detail-ui-store via getState()
 * (stable, no re-renders).
 */

import { useCallback } from "react";
import { type SetURLSearchParams } from "react-router-dom";
import { findRequirementGroup } from "../lib/archive-utils";
import { useBacklogDetailUIStore } from "../stores";
import { useBacklogCRUDHandlers } from "./useBacklogCRUDHandlers";
import { useBacklogFileHandlers } from "./useBacklogFileHandlers";
import type { useBacklogDetailData } from "./useBacklogDetailData";
import type { WorkshopSaveResponse } from "../services/backlog/types";
import type {
  ArchiveRequirement,
  ArchiveRequirementRecord,
  ArchiveTarget,
  ArchiveTargetFormValues,
  BacklogFile,
  BacklogKind,
  ModuleFormValues,
  ReviewAction,
  ReviewUpdate,
} from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type DetailsTab = "info" | "prompt" | "files";

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
  onWorkshopSaveResult?: (result: WorkshopSaveResponse) => void;
}

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
    onWorkshopSaveResult,
  } = opts;

  const { _mutations, archiveTargets, targetIdSet, reqModuleMap } = data;

  // --- Delegated sub-hooks ---

  const crudHandlers = useBacklogCRUDHandlers({
    data,
    backlogKind,
    name,
    closeDetail,
    refreshActivities,
    onWorkshopSaveResult,
  });

  const fileHandlers = useBacklogFileHandlers({
    data,
    setSelectedFile,
    setActiveTab,
    setSearchParams,
    selectedFile,
  });

  // --- Review ---

  const handleReviewAction = useCallback(
    (id: string, _type: "target" | "requirement", action: ReviewAction) => {
      let reviewItem: ReviewUpdate;
      if (targetIdSet.has(id)) {
        reviewItem = { id, type: "target", ...action };
      } else {
        const moduleId = reqModuleMap.get(id);
        if (!moduleId) return;
        reviewItem = { id, type: "requirement", module_id: moduleId, ...action };
      }
      data.batchReview([reviewItem]);
    },
    [targetIdSet, reqModuleMap, data],
  );

  const handleBulkApprove = useCallback(() => {
    const { selectedTargetIds, selectedRequirementIds } = useBacklogDetailUIStore.getState();
    const items: ReviewUpdate[] = [];
    for (const id of selectedTargetIds) {
      items.push({ id, type: "target", review_status: "approved" });
    }
    for (const id of selectedRequirementIds) {
      const moduleId = reqModuleMap.get(id);
      if (moduleId)
        items.push({ id, type: "requirement", module_id: moduleId, review_status: "approved" });
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
      if (moduleId)
        items.push({ id, type: "requirement", module_id: moduleId, review_status: "flagged" });
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

  const handleDeleteTarget = useCallback(
    (targetId: string) => {
      if (!window.confirm(`Delete target "${targetId}"?`)) return;
      data.deleteTarget(targetId);
    },
    [data],
  );

  const handleTargetDialogSubmit = useCallback(
    (values: ArchiveTargetFormValues) => {
      const { targetDialog } = useBacklogDetailUIStore.getState();
      if (targetDialog.mode === "create") {
        _mutations.createTarget.mutate(values, {
          onSuccess: () => {
            useBacklogDetailUIStore.getState().closeTargetDialog();
          },
        });
      } else if (targetDialog.editing) {
        _mutations.updateTarget.mutate(
          { targetId: targetDialog.editing.id, target: values },
          {
            onSuccess: () => {
              useBacklogDetailUIStore.getState().closeTargetDialog();
            },
          },
        );
      }
    },
    [_mutations.createTarget, _mutations.updateTarget],
  );

  // --- Requirements ---

  const handleCreateRequirement = useCallback((groupId: string) => {
    useBacklogDetailUIStore.getState().openReqCreate(groupId);
  }, []);

  const handleEditRequirement = useCallback(
    (groupId: string, requirement: ArchiveRequirement) => {
      useBacklogDetailUIStore
        .getState()
        .openReqEdit({ groupId, req: requirement as ArchiveRequirementRecord });
    },
    [],
  );

  const handleDeleteRequirement = useCallback(
    (groupId: string, requirementId: string) => {
      if (!window.confirm(`Delete requirement "${requirementId}"?`)) return;
      if (!archiveTargets) return;
      const group = findRequirementGroup(archiveTargets.requirements, groupId);
      if (!group) return;
      const updated = group.requirements.filter(
        (r) => r.id !== requirementId,
      ) as ArchiveRequirementRecord[];
      data.updateRequirements({ moduleId: groupId, requirements: updated });
    },
    [archiveTargets, data],
  );

  const handleReorderRequirement = useCallback(
    (groupId: string, requirementId: string, direction: "up" | "down") => {
      if (!archiveTargets) return;
      const group = findRequirementGroup(archiveTargets.requirements, groupId);
      if (!group) return;
      const reqs = [...group.requirements] as ArchiveRequirementRecord[];
      const idx = reqs.findIndex((r) => r.id === requirementId);
      if (idx < 0) return;
      const swapIdx = direction === "up" ? idx - 1 : idx + 1;
      if (swapIdx < 0 || swapIdx >= reqs.length) return;
      const tmp = reqs[idx] as (typeof reqs)[number];
      reqs[idx] = reqs[swapIdx] as (typeof reqs)[number];
      reqs[swapIdx] = tmp;
      data.updateRequirements({ moduleId: groupId, requirements: reqs });
    },
    [archiveTargets, data],
  );

  const handleReqDialogSubmit = useCallback(
    (values: ArchiveRequirementRecord) => {
      const { reqDialog } = useBacklogDetailUIStore.getState();
      if (!reqDialog.editing || !archiveTargets) return;
      const { groupId } = reqDialog.editing;
      const group = findRequirementGroup(archiveTargets.requirements, groupId);
      if (!group) return;
      let updated: ArchiveRequirementRecord[];
      if (reqDialog.editing?.req) {
        updated = group.requirements.map((r) =>
          r.id === values.id ? values : r,
        ) as ArchiveRequirementRecord[];
      } else {
        updated = [...(group.requirements as ArchiveRequirementRecord[]), values];
      }
      _mutations.updateReqs.mutate(
        { moduleId: groupId, requirements: updated },
        {
          onSuccess: () => {
            useBacklogDetailUIStore.getState().closeReqDialog();
          },
        },
      );
    },
    [archiveTargets, _mutations.updateReqs],
  );

  // --- Modules ---

  const handleCreateModule = useCallback(() => {
    useBacklogDetailUIStore.getState().openModuleCreate();
  }, []);

  const handleEditModule = useCallback((groupId: string) => {
    useBacklogDetailUIStore.getState().openModuleEdit(groupId);
  }, []);

  const handleDeleteModule = useCallback(
    (groupId: string) => {
      if (!window.confirm(`Delete module "${groupId}" and all its requirements?`)) return;
      data.deleteModule(groupId);
    },
    [data],
  );

  const handleModuleDialogSubmit = useCallback(
    (values: ModuleFormValues) => {
      const { moduleDialog } = useBacklogDetailUIStore.getState();
      if (moduleDialog.mode === "create") {
        _mutations.createModule.mutate(values, {
          onSuccess: () => {
            useBacklogDetailUIStore.getState().closeModuleDialog();
          },
        });
      } else if (moduleDialog.editing) {
        _mutations.updateModuleMeta.mutate(
          {
            moduleId: moduleDialog.editing,
            payload: { title: values.title, description: values.description },
          },
          {
            onSuccess: () => {
              useBacklogDetailUIStore.getState().closeModuleDialog();
            },
          },
        );
      }
    },
    [_mutations.createModule, _mutations.updateModuleMeta],
  );

  // --- Selection toggles (delegated to store) ---

  const handleTargetToggle = useCallback((id: string) => {
    useBacklogDetailUIStore.getState().toggleTargetId(id);
  }, []);

  const handleRequirementToggle = useCallback((id: string) => {
    useBacklogDetailUIStore.getState().toggleRequirementId(id);
  }, []);

  return {
    // CRUD & workshop (from sub-hook)
    ...crudHandlers,
    // File operations (from sub-hook)
    ...fileHandlers,
    // Review
    handleReviewAction,
    handleBulkApprove,
    handleBulkFlag,
    // Targets
    handleCreateTarget,
    handleEditTarget,
    handleDeleteTarget,
    handleTargetDialogSubmit,
    // Requirements
    handleCreateRequirement,
    handleEditRequirement,
    handleDeleteRequirement,
    handleReorderRequirement,
    handleReqDialogSubmit,
    // Modules
    handleCreateModule,
    handleEditModule,
    handleDeleteModule,
    handleModuleDialogSubmit,
    // Selection
    handleTargetToggle,
    handleRequirementToggle,
  } as const;
}
