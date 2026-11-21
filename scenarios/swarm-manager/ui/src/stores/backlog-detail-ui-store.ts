/**
 * Zustand store for BacklogDetailsPage transient UI state.
 *
 * Centralises dialog open/close flags, selection sets, and review-mode
 * toggle so that deeply-nested child components can read/write them
 * without prop drilling through the page component.
 *
 * Keeps transient detail-page state local to this store.
 */

import { create } from "zustand";
import type { ExecutionRecord, ArchiveRequirementRecord, ArchiveTarget } from "../types";

// ---------------------------------------------------------------------------
// Dialog state (mirrors useDialogState shape for CRUD dialogs)
// ---------------------------------------------------------------------------

interface DialogState<T> {
  isOpen: boolean;
  mode: "create" | "edit";
  editing: T | null;
}

const closedDialog = <T,>(): DialogState<T> => ({ isOpen: false, mode: "create", editing: null });

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

interface BacklogDetailUIState {
  // Simple open/close dialogs
  showEdit: boolean;
  showDelete: boolean;
  showRunModal: boolean;
  showGlobDialog: boolean;
  // Parameterised dialog state
  followUpTarget: ExecutionRecord | null;
  followUpContext?: string;
  roundToDelete: number | null;

  // CRUD dialogs (mirror useDialogState shape)
  reqDialog: DialogState<{ groupId: string; req?: ArchiveRequirementRecord }>;
  moduleDialog: DialogState<string>;
  targetDialog: DialogState<ArchiveTarget>;

  // Selection & review
  selectedTargetIds: Set<string>;
  selectedRequirementIds: Set<string>;
  reviewMode: boolean;

  // --- Actions ---
  openEdit: () => void;
  closeEdit: () => void;
  openDelete: () => void;
  closeDelete: () => void;
  openRunModal: () => void;
  closeRunModal: () => void;
  openGlob: () => void;
  closeGlob: () => void;
  setFollowUpTarget: (target: ExecutionRecord | null) => void;
  setFollowUpContext: (context?: string) => void;
  setRoundToDelete: (round: number | null) => void;

  // CRUD dialog actions
  openReqCreate: (groupId: string) => void;
  openReqEdit: (data: { groupId: string; req: ArchiveRequirementRecord }) => void;
  closeReqDialog: () => void;
  openModuleCreate: () => void;
  openModuleEdit: (moduleId: string) => void;
  closeModuleDialog: () => void;
  openTargetCreate: () => void;
  openTargetEdit: (target: ArchiveTarget) => void;
  closeTargetDialog: () => void;

  // Selection
  toggleTargetId: (id: string) => void;
  toggleRequirementId: (id: string) => void;
  clearSelections: () => void;
  toggleReviewMode: () => void;

  /** Reset all UI state (call when navigating between backlog items). */
  reset: () => void;
}

const INITIAL_STATE = {
  showEdit: false,
  showDelete: false,
  showRunModal: false,
  showGlobDialog: false,
  followUpTarget: null,
  followUpContext: undefined,
  roundToDelete: null,
  reqDialog: closedDialog<{ groupId: string; req?: ArchiveRequirementRecord }>(),
  moduleDialog: closedDialog<string>(),
  targetDialog: closedDialog<ArchiveTarget>(),
  selectedTargetIds: new Set<string>(),
  selectedRequirementIds: new Set<string>(),
  reviewMode: false,
} as const;

export const useBacklogDetailUIStore = create<BacklogDetailUIState>((set) => ({
  ...INITIAL_STATE,

  // Simple toggles
  openEdit: () => set({ showEdit: true }),
  closeEdit: () => set({ showEdit: false }),
  openDelete: () => set({ showDelete: true }),
  closeDelete: () => set({ showDelete: false }),
  openRunModal: () => set({ showRunModal: true }),
  closeRunModal: () => set({ showRunModal: false }),
  openGlob: () => set({ showGlobDialog: true }),
  closeGlob: () => set({ showGlobDialog: false }),
  setFollowUpTarget: (target) => set({ followUpTarget: target, ...(target ? {} : { followUpContext: undefined }) }),
  setFollowUpContext: (context) => set({ followUpContext: context }),
  setRoundToDelete: (round) => set({ roundToDelete: round }),

  // CRUD dialogs
  openReqCreate: (groupId) => set({ reqDialog: { isOpen: true, mode: "create", editing: { groupId } } }),
  openReqEdit: (data) => set({ reqDialog: { isOpen: true, mode: "edit", editing: data } }),
  closeReqDialog: () => set({ reqDialog: closedDialog() }),
  openModuleCreate: () => set({ moduleDialog: { isOpen: true, mode: "create", editing: null } }),
  openModuleEdit: (moduleId) => set({ moduleDialog: { isOpen: true, mode: "edit", editing: moduleId } }),
  closeModuleDialog: () => set({ moduleDialog: closedDialog() }),
  openTargetCreate: () => set({ targetDialog: { isOpen: true, mode: "create", editing: null } }),
  openTargetEdit: (target) => set({ targetDialog: { isOpen: true, mode: "edit", editing: target } }),
  closeTargetDialog: () => set({ targetDialog: closedDialog() }),

  // Selection
  toggleTargetId: (id) =>
    set((s) => {
      const next = new Set(s.selectedTargetIds);
      if (next.has(id)) next.delete(id); else next.add(id);
      return { selectedTargetIds: next };
    }),
  toggleRequirementId: (id) =>
    set((s) => {
      const next = new Set(s.selectedRequirementIds);
      if (next.has(id)) next.delete(id); else next.add(id);
      return { selectedRequirementIds: next };
    }),
  clearSelections: () => set({ selectedTargetIds: new Set(), selectedRequirementIds: new Set() }),
  toggleReviewMode: () => set((s) => ({ reviewMode: !s.reviewMode })),

  reset: () => set({
    ...INITIAL_STATE,
    // Create fresh Set instances to avoid sharing references
    selectedTargetIds: new Set<string>(),
    selectedRequirementIds: new Set<string>(),
    reqDialog: closedDialog(),
    moduleDialog: closedDialog(),
    targetDialog: closedDialog(),
  }),
}));
