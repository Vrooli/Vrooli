import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Code, ConnectError } from "@connectrpc/connect";

import {
  StateStatus,
  WorkspaceProvenance,
  deleteWorkspaceAsset,
  getWorkspace,
  listWorkspaceAssets,
  saveWorkspace,
  uploadWorkspaceAsset,
  workspaceAssetUrl,
  type Workspace,
  type WorkspaceAsset,
  type WorkspaceAssetFidelity,
  type WorkspaceDisplay,
  type WorkspacePage,
  type WorkspaceState,
} from "../../api/workspaces";
import type { DesignDevice, DesignDocument, DesignFrame, DesignOptions } from "../designs";
import {
  emptyStack,
  jumpCommand,
  pushCommand,
  redoCommand,
  undoCommand,
  type CommandStack,
} from "../studio/history";
import { DEVICE_NAME, docSignature, rekeySpec } from "../studio/model";
import { assertImageUpload } from "./assets";
import type { TemplateNode, TemplatePalette } from "../templates/spec";
import {
  activeStateOf,
  displaySizes,
  editableSpec,
  pageDocument,
  protoDevice,
  readStateSpec,
  seedSpec,
  writeStateSpec,
} from "../workspaces/spec";
import { applyPatch, PatchError } from "./patch";
import type { ProposalApplication, StudioProposal } from "./proposal";
import type { SavedState } from "./useStudio";

const AUTOSAVE_MS = 900;

function withoutKey(record: Record<string, string>, key: string): Record<string, string> {
  return Object.fromEntries(Object.entries(record).filter(([entry]) => entry !== key));
}

function clone<T>(value: T): T {
  return structuredClone(value);
}

function localId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}${Date.now().toString(36).slice(-6)}`;
}

function makeDisplay(device: DesignDevice, spec: TemplateNode): WorkspaceDisplay {
  const size = displaySizes(device);
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
    id: localId("dp"),
    device: protoDevice(device),
    width: size.width,
    height: size.height,
    sortOrder: 0,
    states: [makeState("Default", spec, 0)],
  };
}

function makeState(name: string, spec: TemplateNode, order: number): WorkspaceState {
  const state: WorkspaceState = {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState",
    id: localId("st"),
    name,
    argsJson: "{}",
    scenarioDataJson: "{}",
    status: StateStatus.DRAFT,
    captureId: "",
    sortOrder: order,
  };
  writeStateSpec(state, spec);
  return state;
}

function makePage(name: string, spec: TemplateNode, order: number): WorkspacePage {
  const display = makeDisplay("desktop", spec);
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
    id: localId("pg"),
    name,
    route: "",
    pageTypeId: "",
    sortOrder: order,
    specJson: JSON.stringify(spec),
    displays: [display],
  };
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
  updatePage: (patch: Partial<Pick<WorkspacePage, "name" | "route" | "pageTypeId">>, label: string, mergeKey?: string) => void;
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
  resolveConflict: (choice: "keep-mine" | "take-theirs") => void;
}

interface StudioModel {
  workspace: Workspace | null;
  history: CommandStack<Workspace>;
}

/**
 * One workspace studio state owner. It projects the active page into the
 * document shape the mature studio components already render, and writes every
 * edit back through the revision-guarded workspace aggregate. Autosave is
 * serialized and never replaces newer edits with an older response.
 */
export function useWorkspaceStudio(workspaceId: string): WorkspaceStudioApi {
  const [model, setModel] = useState<StudioModel>({ workspace: null, history: emptyStack() });
  const [ready, setReady] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pageId, setPageId] = useState("");
  const [displayId, setDisplayId] = useState("");
  const [stateByDisplay, setStateByDisplay] = useState<Record<string, string>>({});
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [baseRevision, setBaseRevision] = useState(0);
  const [saved, setSaved] = useState<SavedState | undefined>(undefined);
  const [saving, setSaving] = useState(false);
  const [saveFailed, setSaveFailed] = useState(false);
  const [conflict, setConflict] = useState<{ current: Workspace } | null>(null);
  const [assets, setAssets] = useState<WorkspaceAsset[]>([]);
  const [assetsLoading, setAssetsLoading] = useState(false);
  const [assetError, setAssetError] = useState<string | null>(null);
  // A slow upload list for one workspace must never overwrite the list for
  // another after navigation. The token makes late results ignorable.
  const assetRequest = useRef(0);

  const workspaceRef = useRef<Workspace | null>(null);
  workspaceRef.current = model.workspace;
  const pageIdRef = useRef(pageId);
  pageIdRef.current = pageId;
  const statesRef = useRef(stateByDisplay);
  statesRef.current = stateByDisplay;
  const revisionRef = useRef(baseRevision);
  revisionRef.current = baseRevision;
  const inFlight = useRef<Promise<void> | null>(null);
  const conflictRef = useRef<{ current: Workspace } | null>(null);
  conflictRef.current = conflict;
  const readOnlyRef = useRef(false);
  const dirtyRef = useRef(false);
  const saveRef = useRef<() => Promise<void>>(() => Promise.resolve());
  // A failed write is not retried on a timer: a down API should not be polled
  // every second. The next edit or the explicit retry clears this signature.
  const failedSignature = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setReady(false);
    setLoadError(null);
    setConflict(null);
    setAssets([]);
    setAssetError(null);
    const request = ++assetRequest.current;
    setAssetsLoading(true);
    void listWorkspaceAssets(workspaceId)
      .then((list) => {
        if (!cancelled && assetRequest.current === request) setAssets(list);
      })
      .catch((cause: unknown) => {
        if (!cancelled && assetRequest.current === request) {
          setAssetError(cause instanceof Error ? cause.message : "assets-unavailable");
        }
      })
      .finally(() => {
        if (!cancelled && assetRequest.current === request) setAssetsLoading(false);
      });
    getWorkspace(workspaceId)
      .then((loaded) => {
        if (cancelled) return;
        if (!loaded) {
          setLoadError("not-found");
          setReady(true);
          return;
        }
        const firstPage = loaded.pages[0];
        setModel({ workspace: loaded, history: emptyStack() });
        setBaseRevision(loaded.revision);
        setPageId(firstPage?.id ?? "");
        setDisplayId(firstPage?.displays[0]?.id ?? "");
        setStateByDisplay({});
        setSelectedId(null);
        setSaved({
          revision: loaded.revision,
          at: Date.now(),
          signature: workspaceSignature(loaded),
        });
        setReady(true);
      })
      .catch((cause: unknown) => {
        if (cancelled) return;
        setLoadError(cause instanceof Error ? cause.message : "load-failed");
        setReady(true);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId]);

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
  readOnlyRef.current = readOnly;
  const dirty = !readOnly && Boolean(saved && workspace && signature !== saved.signature);
  dirtyRef.current = dirty;

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

  const selectPage = useCallback(
    (id: string) => {
      const target = workspace?.pages.find((entry) => entry.id === id);
      if (!target) return;
      setPageId(id);
      setDisplayId(target.displays[0]?.id ?? "");
      setSelectedId(null);
    },
    [workspace],
  );

  const reloadAssets = useCallback(async () => {
    const request = ++assetRequest.current;
    setAssetsLoading(true);
    try {
      const list = await listWorkspaceAssets(workspaceId);
      if (assetRequest.current === request) {
        setAssets(list);
        setAssetError(null);
      }
    } catch (cause) {
      if (assetRequest.current === request) {
        setAssetError(cause instanceof Error ? cause.message : "assets-unavailable");
      }
      throw cause;
    } finally {
      if (assetRequest.current === request) setAssetsLoading(false);
    }
  }, [workspaceId]);

  const importAsset = useCallback(
    async (file: File, fidelity: WorkspaceAssetFidelity = "reference") => {
      // The server is authoritative, but rejecting here turns a slow 8 MiB
      // upload into an immediate message.
      assertImageUpload(file);
      const asset = await uploadWorkspaceAsset(workspaceId, file, fidelity);
      // Any list requested before this upload must not overwrite the new row
      // with a snapshot that predates it.
      assetRequest.current += 1;
      setAssets((current) => [asset, ...current.filter((entry) => entry.id !== asset.id)]);
      setAssetError(null);
    },
    [workspaceId],
  );

  const removeAsset = useCallback(
    async (assetId: string) => {
      await deleteWorkspaceAsset(workspaceId, assetId);
      assetRequest.current += 1;
      setAssets((current) => current.filter((entry) => entry.id !== assetId));
    },
    [workspaceId],
  );

  const selectAddress = useCallback((nextPage: string, nextDisplay: string, nextState: string) => {
    if (workspaceRef.current) {
      const target = workspaceRef.current.pages.find((entry) => entry.id === nextPage);
      if (target) setPageId(target.id);
    }
    setDisplayId(nextDisplay);
    if (nextDisplay && nextState) {
      setStateByDisplay((current) => ({ ...current, [nextDisplay]: nextState }));
    }
    setSelectedId(null);
  }, []);

  const selectDisplay = useCallback((id: string) => {
    setDisplayId(id);
    setSelectedId(null);
  }, []);

  const selectFrame = useCallback(
    (id: string) => {
      const frame = page?.displays.find((entry) => entry.id === id);
      if (!frame) return;
      setDisplayId(id);
      setSelectedId(null);
    },
    [page],
  );

  const selectState = useCallback(
    (id: string) => {
      if (!display) return;
      setStateByDisplay((current) => ({ ...current, [display.id]: id }));
      setSelectedId(null);
    },
    [display],
  );

  const updateFrame = useCallback(
    (frameId: string, mutate: (spec: TemplateNode) => TemplateNode, label: string, mergeKey?: string) => {
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

  const addPage = useCallback(
    (name?: string, spec?: TemplateNode) => {
      const draftPage = makePage(name?.trim() || "New page", spec ?? seedSpec("page"), 0);
      apply("Add page", undefined, (draft) => {
        draftPage.sortOrder = draft.pages.length;
        draft.pages.push(draftPage);
      });
      setPageId(draftPage.id);
      setDisplayId(draftPage.displays[0]?.id ?? "");
      setSelectedId(null);
    },
    [apply],
  );

  const duplicatePage = useCallback(() => {
    if (!page) return;
    const copy = clone(page);
    copy.id = localId("pg");
    copy.name = `${page.name} copy`;
    copy.sortOrder = page.sortOrder + 1;
    copy.displays = copy.displays.map((entry) => {
      const displayId = localId("dp");
      const states = entry.states.map((stateEntry) => ({
        ...stateEntry,
        id: localId("st"),
      }));
      const first = states[0];
      if (first) {
        const spec = readStateSpec(first);
        if (spec) writeStateSpec(first, rekeySpec(spec, displayId));
      }
      return { ...entry, id: displayId, states };
    });
    apply("Duplicate page", undefined, (draft) => {
      draft.pages.push(copy);
    });
    setPageId(copy.id);
    setDisplayId(copy.displays[0]?.id ?? "");
  }, [apply, page]);

  const removePage = useCallback(() => {
    if (!page || !workspace || workspace.pages.length <= 1) return;
    const removed = page.id;
    apply("Remove page", undefined, (draft) => {
      draft.pages = draft.pages.filter((entry) => entry.id !== removed);
      draft.pages.forEach((entry, index) => {
        entry.sortOrder = index;
      });
    });
    const next = workspace.pages.find((entry) => entry.id !== removed);
    setPageId(next?.id ?? "");
    setDisplayId(next?.displays[0]?.id ?? "");
    setSelectedId(null);
  }, [apply, page, workspace]);

  const addFrame = useCallback(
    (device: DesignDevice) => {
      if (!page) return;
      const display = makeDisplay(device, seedSpec(`${page.id}:${localId("seed")}`));
      display.sortOrder = page.displays.length;
      apply(`Add ${DEVICE_NAME[device]} display`, undefined, (draft) => {
        const target = draft.pages.find((entry) => entry.id === page.id);
        if (target) target.displays.push(display);
      });
      setDisplayId(display.id);
      setSelectedId(null);
    },
    [apply, page],
  );

  const removeFrame = useCallback(
    (id: string) => {
      if (!page || page.displays.length <= 1) return;
      apply("Remove display", undefined, (draft) => {
        const target = draft.pages.find((entry) => entry.id === page.id);
        if (!target) return;
        target.displays = target.displays.filter((entry) => entry.id !== id);
        target.displays.forEach((entry, index) => {
          entry.sortOrder = index;
        });
      });
      setStateByDisplay((current) => withoutKey(current, id));
      if (displayId === id) {
        const next = page.displays.find((entry) => entry.id !== id);
        setDisplayId(next?.id ?? "");
        setSelectedId(null);
      }
    },
    [apply, displayId, page],
  );

  const addState = useCallback(() => {
    if (!page || !display) return;
    const current = activeStateOf(display, stateByDisplay);
    const spec = readStateSpec(current) ?? readStateSpec(display.states[0]) ?? seedSpec(display.id);
    const stateEntry = makeState(`State ${display.states.length + 1}`, spec, display.states.length);
    apply("Add state", undefined, (draft) => {
      const target = draft.pages
        .find((entry) => entry.id === page.id)
        ?.displays.find((entry) => entry.id === display.id);
      target?.states.push(stateEntry);
    });
    setStateByDisplay((values) => ({ ...values, [display.id]: stateEntry.id }));
    setSelectedId(null);
  }, [apply, display, page, stateByDisplay]);

  const renameState = useCallback(
    (name: string) => {
      if (!page || !display || !state) return;
      apply("Rename state", `state-name:${state.id}`, (draft) => {
        const target = draft.pages
          .find((entry) => entry.id === page.id)
          ?.displays.find((entry) => entry.id === display.id)
          ?.states.find((entry) => entry.id === state.id);
        if (target) target.name = name;
      });
    },
    [apply, display, page, state],
  );

  const removeState = useCallback(() => {
    if (!page || !display || !state || display.states.length <= 1) return;
    const removed = state.id;
    apply("Remove state", undefined, (draft) => {
      const target = draft.pages
        .find((entry) => entry.id === page.id)
        ?.displays.find((entry) => entry.id === display.id);
      if (!target) return;
      target.states = target.states.filter((entry) => entry.id !== removed);
      target.states.forEach((entry, index) => {
        entry.sortOrder = index;
      });
    });
    setStateByDisplay((current) => withoutKey(current, display.id));
  }, [apply, display, page, state]);

  const applyProposal = useCallback(
    (proposal: StudioProposal): ProposalApplication => {
      const current = workspaceRef.current;
      const activePage = current?.pages.find((entry) => entry.id === pageIdRef.current) ?? current?.pages[0];
      if (!current || !activePage) {
        return { ok: false, reason: "invalid", detail: "There is no active page." };
      }
      const doc = pageDocument(current, activePage, statesRef.current);
      // The design id, page projection and signature prove the proposal was
      // computed against what is on screen; the revision is not enough because
      // autosave bumps it without changing the composition.
      if (proposal.designId !== current.id || proposal.signature !== docSignature(doc)) {
        return {
          ok: false,
          reason: "stale",
          detail: "The workspace changed after this proposal was made.",
        };
      }
      let applied;
      try {
        applied = applyPatch(doc, proposal.operations);
      } catch (error) {
        return {
          ok: false,
          reason: "invalid",
          detail: error instanceof PatchError ? error.reason : "The patch could not be applied.",
        };
      }
      apply(`AI · ${proposal.summary}`, undefined, (draft) => {
        const targetPage = draft.pages.find((entry) => entry.id === activePage.id);
        if (!targetPage) return;
        for (const frame of applied.doc.frames) {
          const display = targetPage.displays.find((entry) => entry.id === frame.id);
          if (!display) continue;
          const state = activeStateOf(display, statesRef.current);
          if (state) writeStateSpec(state, frame.spec);
        }
        // A text or layer patch must not rewrite a theme the person never
        // touched; only a setTheme operation owns themeJson.
        if (proposal.operations.some((operation) => operation.kind === "setTheme")) {
          draft.themeJson = JSON.stringify(applied.doc.theme);
        }
      });
      return { ok: true };
    },
    [apply],
  );

  const undo = useCallback(() => {
    setModel((current) => {
      const result = undoCommand(current.history);
      return result ? { workspace: clone(result.state), history: result.stack } : current;
    });
    setSelectedId(null);
  }, []);

  const redo = useCallback(() => {
    setModel((current) => {
      const result = redoCommand(current.history);
      return result ? { workspace: clone(result.state), history: result.stack } : current;
    });
    setSelectedId(null);
  }, []);

  const jump = useCallback((position: number) => {
    setModel((current) => {
      const result = jumpCommand(current.history, position);
      return result ? { workspace: clone(result.state), history: result.stack } : current;
    });
    setSelectedId(null);
  }, []);

  const save = useCallback(async () => {
    if (inFlight.current) return inFlight.current;
    // A conflict is a decision point. Overwriting the other writer silently is
    // exactly what the revision guard exists to prevent.
    if (conflictRef.current || readOnlyRef.current) return;
    const snapshot = workspaceRef.current;
    if (!snapshot) return;
    const revision = revisionRef.current;
    const signatureSent = workspaceSignature(snapshot);
    const run = (async () => {
      setSaving(true);
      setSaveFailed(false);
      try {
        const next = await saveWorkspace(snapshot, revision);
        revisionRef.current = next.revision;
        setBaseRevision(next.revision);
        setSaved({ revision: next.revision, at: Date.now(), signature: signatureSent });
        failedSignature.current = null;
        setConflict(null);
        setModel((current) => {
          if (!current.workspace) return current;
          if (current.workspace === snapshot) {
            return { workspace: next, history: current.history };
          }
          // A newer edit arrived while this save was in flight. Keep the newer
          // bytes, record the revision the server accepted, and let the
          // autosave effect write the remainder.
          const merged = { ...current.workspace, revision: next.revision };
          return { workspace: merged, history: current.history };
        });
      } catch (cause) {
        const error = ConnectError.from(cause);
        if (error.code === Code.Aborted) {
          setConflict({ current: snapshot });
          try {
            const latest = await getWorkspace(snapshot.id);
            if (latest) setConflict({ current: latest });
          } catch {
            // The conflict is still reported; the reload is a convenience.
          }
        } else {
          failedSignature.current = signatureSent;
          setSaveFailed(true);
        }
      } finally {
        setSaving(false);
        inFlight.current = null;
      }
    })();
    inFlight.current = run;
    return run;
  }, []);

  // Debounced autosave. Every edit creates a new workspace object, so the
  // effect restarts the timer and one quiet interval produces one save.
  useEffect(() => {
    if (!ready || conflict || readOnly || !dirty) return;
    if (failedSignature.current === signature) return;
    const timer = globalThis.setTimeout(() => {
      void save();
    }, AUTOSAVE_MS);
    return () => globalThis.clearTimeout(timer);
  }, [ready, conflict, readOnly, dirty, signature, model.workspace, save]);

  // Navigation must not discard the last edits. The debounce is cleared on
  // unmount, so flush once while the request can still finish.
  useEffect(() => {
    saveRef.current = save;
    const flush = () => {
      if (dirtyRef.current && !conflictRef.current) void saveRef.current();
    };
    const onHidden = () => {
      if (document.visibilityState === "hidden") flush();
    };
    window.addEventListener("pagehide", flush);
    document.addEventListener("visibilitychange", onHidden);
    return () => {
      window.removeEventListener("pagehide", flush);
      document.removeEventListener("visibilitychange", onHidden);
      flush();
    };
  }, [save]);

  const resolveConflict = useCallback(
    (choice: "keep-mine" | "take-theirs") => {
      const current = conflict?.current;
      if (!current) return;
      if (choice === "take-theirs") {
        setModel({ workspace: current, history: emptyStack() });
        setBaseRevision(current.revision);
        revisionRef.current = current.revision;
        setPageId(current.pages[0]?.id ?? "");
        setDisplayId(current.pages[0]?.displays[0]?.id ?? "");
        setSaved({
          revision: current.revision,
          at: Date.now(),
          signature: workspaceSignature(current),
        });
        setConflict(null);
        return;
      }
      setBaseRevision(current.revision);
      revisionRef.current = current.revision;
      setModel((model) =>
        model.workspace ? { ...model, workspace: { ...model.workspace, revision: current.revision } } : model,
      );
      setConflict(null);
    },
    [conflict],
  );

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
    resolveConflict,
  };
}

/**
 * Content signature of the whole aggregate, independent of which page, display
 * or state is currently on screen. Using the visible document instead would
 * mark a workspace dirty every time someone changed state without editing.
 */
function workspaceSignature(workspace: Workspace): string {
  return JSON.stringify({
    name: workspace.name,
    summary: workspace.summary,
    provenance: workspace.provenance,
    tags: workspace.tags,
    bindingScenario: workspace.bindingScenario,
    bindingPage: workspace.bindingPage,
    themeJson: workspace.themeJson,
    optionsJson: workspace.optionsJson,
    pages: workspace.pages.map((page) => ({
      id: page.id,
      name: page.name,
      route: page.route,
      pageTypeId: page.pageTypeId,
      sortOrder: page.sortOrder,
      specJson: page.specJson,
      displays: page.displays.map((display) => ({
        id: display.id,
        device: display.device,
        width: display.width,
        height: display.height,
        sortOrder: display.sortOrder,
        states: display.states.map((state) => ({
          id: state.id,
          name: state.name,
          argsJson: state.argsJson,
          scenarioDataJson: state.scenarioDataJson,
          status: state.status,
          captureId: state.captureId,
          sortOrder: state.sortOrder,
        })),
      })),
    })),
  });
}

