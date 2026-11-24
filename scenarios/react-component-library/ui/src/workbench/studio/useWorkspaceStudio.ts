import { useCallback, useMemo, useRef, useState } from "react";

import {
  WorkspaceProvenance,
  workspaceAssetUrl,
  type Workspace,
  type WorkspaceAsset,
  type WorkspaceAssetFidelity,
  type WorkspaceDisplay,
  type WorkspacePage,
  type WorkspaceState,
} from "../../api/workspaces";
import type { DesignDevice, DesignDocument, DesignFrame, DesignOptions } from "../design-types";
import {
  emptyStack,
  pushCommand,
  type CommandStack,
} from "../studio/history";
import { useWorkspaceAssets } from "./useWorkspaceAssets";
import type { TemplateNode, TemplatePalette } from "../templates/spec";
import { activeStateOf, editableSpec, pageDocument, writeStateSpec } from "../workspaces/spec";
import { useWorkspaceProposal } from "./useWorkspaceProposal";
import { useWorkspaceStructure } from "./useWorkspaceStructure";
import { useWorkspacePersistence, workspaceSignature } from "./useWorkspacePersistence";
import { useWorkspaceHistory } from "./useWorkspaceHistory";
import type { ProposalApplication, StudioProposal } from "./proposal";
import type { SavedState } from "./persistence";

function clone<T>(value: T): T {
  return structuredClone(value);
}

export interface WorkspaceStudioApi {
  ready: boolean;
  loadError: string | null;
  workspace: Workspace | null;
  page: WorkspacePage | null;
  display: WorkspaceDisplay | null;
  state: WorkspaceState | null;
  doc: DesignDocument | null;
  frames: DesignFrame[];
  stateByDisplay: Record<string, string>;
  history: CommandStack<Workspace>;
  dirty: boolean;
  saving: boolean;
  saveFailed: boolean;
  saved: SavedState | undefined;
  conflict: { current: Workspace } | null;
  selectedId: string | null;
  /** True for a curated template: preview only, never writable. */
  readOnly: boolean;
  /** Reference uploads, newest first. */
  assets: WorkspaceAsset[];
  assetsLoading: boolean;
  assetError: string | null;
  assetUrl: (asset: { url: string }) => string;
  importAsset: (file: File, fidelity?: WorkspaceAssetFidelity) => Promise<void>;
  removeAsset: (assetId: string) => Promise<void>;
  reloadAssets: () => Promise<void>;
  /** Apply a validated proposal as one undoable command. */
  applyProposal: (proposal: StudioProposal) => ProposalApplication;
  selectPage: (id: string) => void;
  /** Apply a page/display/state address in one render, used by deep links. */
  selectAddress: (pageId: string, displayId: string, stateId: string) => void;
  selectDisplay: (id: string) => void;
  selectFrame: (id: string) => void;
  selectState: (id: string) => void;
  selectNode: (id: string | null) => void;
  updateFrame: (
    frameId: string,
    mutate: (spec: TemplateNode) => TemplateNode,
    label: string,
    mergeKey?: string,
  ) => void;
  updateTheme: (patch: Partial<TemplatePalette>, label: string, mergeKey?: string) => void;
  updateOptions: (patch: Partial<DesignOptions>, label: string) => void;
  updatePage: (
    patch: Partial<Pick<WorkspacePage, "name" | "route" | "pageTypeId">>,
    label: string,
    mergeKey?: string,
  ) => void;
  renamePage: (name: string, label: string) => void;
  addPage: (name?: string, spec?: TemplateNode) => void;
  duplicatePage: () => void;
  removePage: () => void;
  addFrame: (device: DesignDevice) => void;
  removeFrame: (id: string) => void;
  addState: () => void;
  renameState: (name: string) => void;
  removeState: () => void;
  undo: () => void;
  redo: () => void;
  jump: (position: number) => void;
  save: () => Promise<void>;
  syncRemote: (workspace?: Workspace) => Promise<void>;
  resolveConflict: (choice: "keep-mine" | "take-theirs") => void;
}

interface StudioModel {
  workspace: Workspace | null;
  history: CommandStack<Workspace>;
}

/** Owns workspace projections and composes the private editing hooks. */
export function useWorkspaceStudio(workspaceId: string): WorkspaceStudioApi {
  const [model, setModel] = useState<StudioModel>({ workspace: null, history: emptyStack() });
  const [pageId, setPageId] = useState("");
  const [displayId, setDisplayId] = useState("");
  const [stateByDisplay, setStateByDisplay] = useState<Record<string, string>>({});
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { assets, assetsLoading, assetError, importAsset, removeAsset, reloadAssets } =
    useWorkspaceAssets(workspaceId);

  const workspaceRef = useRef<Workspace | null>(null);
  workspaceRef.current = model.workspace;
  const pageIdRef = useRef(pageId);
  pageIdRef.current = pageId;
  const displayIdRef = useRef(displayId);
  displayIdRef.current = displayId;
  const statesRef = useRef(stateByDisplay);
  statesRef.current = stateByDisplay;
  const workspace = model.workspace;
  const page = useMemo(
    () => workspace?.pages.find((entry) => entry.id === pageId) ?? workspace?.pages[0] ?? null,
    [workspace, pageId],
  );
  const display = useMemo(
    () => page?.displays.find((entry) => entry.id === displayId) ?? page?.displays[0] ?? null,
    [page, displayId],
  );
  const state = useMemo(
    () => activeStateOf(display ?? undefined, stateByDisplay) ?? null,
    [display, stateByDisplay],
  );
  const doc = useMemo(
    () => (workspace && page ? pageDocument(workspace, page, stateByDisplay) : null),
    [workspace, page, stateByDisplay],
  );
  const frames = doc?.frames ?? [];

  const signature = useMemo(() => (workspace ? workspaceSignature(workspace) : ""), [workspace]);
  const readOnly = workspace?.provenance === WorkspaceProvenance.CURATED;
  const {
    ready,
    loadError,
    saved,
    saving,
    saveFailed,
    conflict,
    dirty,
    save,
    syncRemote,
    resolveConflict,
  } = useWorkspacePersistence({
    workspaceId,
    workspace,
    workspaceRef,
    pageIdRef,
    displayIdRef,
    signature,
    readOnly,
    setModel,
    setPageId,
    setDisplayId,
    setStateByDisplay,
    setSelectedId,
  });

  const apply = useCallback(
    (label: string, mergeKey: string | undefined, change: (draft: Workspace) => void) => {
      setModel((current) => {
        if (!current.workspace) return current;
        // A curated template is a reviewed snapshot. Editing it locally would
        // be lost on navigation, so the mutation seam refuses before the UI can
        // offer a control that appears to work.
        if (current.workspace.provenance === WorkspaceProvenance.CURATED) return current;
        const before = clone(current.workspace);
        const after = clone(current.workspace);
        change(after);
        return {
          workspace: after,
          history: pushCommand(current.history, { label, mergeKey, before, after }),
        };
      });
    },
    [],
  );

  const updateFrame = useCallback(
    (
      frameId: string,
      mutate: (spec: TemplateNode) => TemplateNode,
      label: string,
      mergeKey?: string,
    ) => {
      const activePageId = page?.id;
      if (!activePageId) return;
      apply(label, mergeKey, (draft) => {
        const targetPage = draft.pages.find((entry) => entry.id === activePageId);
        const targetDisplay = targetPage?.displays.find((entry) => entry.id === frameId);
        if (!targetPage || !targetDisplay) return;
        const targetState = activeStateOf(targetDisplay, statesRef.current);
        if (!targetState) return;
        writeStateSpec(
          targetState,
          mutate(editableSpec(targetPage, targetDisplay, targetState.id)),
        );
      });
    },
    [apply, page?.id],
  );

  const updateTheme = useCallback(
    (patch: Partial<TemplatePalette>, label: string, mergeKey?: string) => {
      apply(label, mergeKey, (draft) => {
        const current = JSON.parse(draft.themeJson || "{}") as Record<string, unknown>;
        draft.themeJson = JSON.stringify({ ...current, ...patch });
      });
    },
    [apply],
  );

  const updateOptions = useCallback(
    (patch: Partial<DesignOptions>, label: string) => {
      apply(label, undefined, (draft) => {
        const current = JSON.parse(draft.optionsJson || "{}") as Record<string, unknown>;
        draft.optionsJson = JSON.stringify({ ...current, ...patch });
      });
    },
    [apply],
  );

  const updatePage = useCallback(
    (
      patch: Partial<Pick<WorkspacePage, "name" | "route" | "pageTypeId">>,
      label: string,
      mergeKey?: string,
    ) => {
      const activePageId = page?.id;
      if (!activePageId) return;
      apply(label, mergeKey, (draft) => {
        const target = draft.pages.find((entry) => entry.id === activePageId);
        if (!target) return;
        Object.assign(target, patch);
      });
    },
    [apply, page?.id],
  );

  const renamePage = useCallback(
    (name: string, label: string) => {
      const activePageId = page?.id;
      if (!activePageId) return;
      apply(label, `page-name:${activePageId}`, (draft) => {
        const target = draft.pages.find((entry) => entry.id === activePageId);
        if (target) target.name = name;
      });
    },
    [apply, page?.id],
  );

  const {
    selectPage,
    selectAddress,
    selectDisplay,
    selectFrame,
    selectState,
    addPage,
    duplicatePage,
    removePage,
    addFrame,
    removeFrame,
    addState,
    renameState,
    removeState,
  } = useWorkspaceStructure({
    workspace,
    workspaceRef,
    page,
    display,
    state,
    displayId,
    stateByDisplay,
    apply,
    setPageId,
    setDisplayId,
    setStateByDisplay,
    setSelectedId,
  });

  const applyProposal = useWorkspaceProposal({ workspaceRef, pageIdRef, statesRef, apply });

  const { undo, redo, jump } = useWorkspaceHistory(setModel, setSelectedId);

  return {
    ready,
    loadError,
    workspace,
    page,
    display,
    state,
    doc,
    frames,
    stateByDisplay,
    history: model.history,
    dirty,
    saving,
    saveFailed,
    saved,
    conflict,
    selectedId,
    readOnly,
    assets,
    assetsLoading,
    assetError,
    assetUrl: workspaceAssetUrl,
    importAsset,
    removeAsset,
    reloadAssets,
    applyProposal,
    selectPage,
    selectAddress,
    selectDisplay,
    selectFrame,
    selectState,
    selectNode: setSelectedId,
    updateFrame,
    updateTheme,
    updateOptions,
    updatePage,
    renamePage,
    addPage,
    duplicatePage,
    removePage,
    addFrame,
    removeFrame,
    addState,
    renameState,
    removeState,
    undo,
    redo,
    jump,
    save,
    syncRemote,
    resolveConflict,
  };
}
