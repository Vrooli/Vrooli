import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  DesignConflictError,
  createDesign,
  deleteDesign,
  deleteDesignAsset,
  designsUsingFallback,
  getDesign,
  listDesignAssets,
  listDesigns,
  putDesign,
  uploadDesignAsset,
} from "../designs";
import type { DesignAsset, DesignDocument, DesignFrame, DesignOptions } from "../designs";
import type { Template, TemplateNode, TemplatePalette } from "../templates/spec";
import { findNode } from "../templates/spec";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { assertImageUpload } from "./assets";
import {
  anonymousFrame,
  docSignature,
  duplicateInput,
  seedInput,
  type DesignBinding,
} from "./model";
import {
  emptyStack,
  jumpCommand,
  pushCommand,
  redoCommand,
  undoCommand,
  type CommandStack,
} from "./history";
import { createSaveQueue } from "./saveQueue";
import { applyPatch, PatchError } from "./patch";
import type { StudioProposal } from "./proposal";
import { useStudioSync, type StudioSyncApi } from "./useStudioSync";

const AUTOSAVE_DELAY_MS = 700;
/** Remember the design the person was last editing (tab restore, not data). */
const ACTIVE_DESIGN_KEY = "rcl.studio.active-design";
/** Mirrors the API limit so an oversized file never leaves the browser. */


export interface SavedState {
  revision: number;
  at: number;
  signature: string;
  /** True when the last write only reached this browser. */
  local?: boolean;
}

/** Kept as an alias so the Assets pane and callers share one asset shape. */
export type ReferenceAsset = DesignAsset;

interface StudioSeed {
  doc: DesignDocument;
  signature: string;
}

function provisionalDesign(binding: DesignBinding): DesignDocument {
  const input = seedInput(DEFAULT_TEMPLATE, undefined, binding);
  return {
    id: "studio-draft",
    name: input.name,
    updatedAt: new Date().toISOString(),
    revision: 1,
    frames: input.frames,
    theme: input.theme,
    options: input.options,
    scenario: input.scenario ?? null,
    page: input.page ?? null,
  };
}

/**
 * The first paint must already show a real composition. The provisional
 * document is replaced by the stored one as soon as the async designs client
 * answers, so there is no empty flash and the page story still finds Aperture.
 */
function initialSeed(binding: DesignBinding): StudioSeed {
  const doc = provisionalDesign(binding);
  return { doc, signature: docSignature(doc) };
}

interface LoadedDesigns {
  docs: DesignDocument[];
  /** Designs whose document could not be read; they stay listed server-side. */
  failed: number;
}

let loadPromise: Promise<LoadedDesigns> | null = null;

function loadDesigns(): Promise<LoadedDesigns> {
  loadPromise ??= (async (): Promise<LoadedDesigns> => {
    const summaries = await listDesigns();
    if (summaries.length === 0) {
      const created = await createDesign(seedInput(DEFAULT_TEMPLATE));
      return { docs: [created], failed: 0 };
    }
    const settled = await Promise.allSettled(summaries.map((summary) => getDesign(summary.id)));
    const docs = settled.flatMap((result) => (result.status === "fulfilled" ? [result.value] : []));
    if (docs.length === 0) {
      const created = await createDesign(seedInput(DEFAULT_TEMPLATE));
      return { docs: [created], failed: settled.length };
    }
    return {
      docs: docs.sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)),
      failed: settled.length - docs.length,
    };
  })().catch((error: unknown) => {
    loadPromise = null;
    throw error;
  });
  return loadPromise;
}

function savedFromDoc(doc: DesignDocument): SavedState {
  return {
    revision: doc.revision,
    at: Date.parse(doc.updatedAt) || Date.now(),
    signature: docSignature(doc),
    local: false,
  };
}

import type { ProposalApplication } from "./proposal";

export type { ProposalApplication } from "./proposal";

export interface StudioApi {
  ready: boolean;
  docs: DesignDocument[];
  openDocs: DesignDocument[];
  activeDoc: DesignDocument | undefined;
  activeFrame: DesignFrame | undefined;
  selectedId: string | null;
  saving: boolean;
  saveFailed: boolean;
  /** Designs the server listed but the client could not read. */
  unavailable: number;
  /** True when storage fell back to this browser (API unreachable). */
  fallback: boolean;
  /** A stale-revision save that another tab won. */
  conflict: { designId: string; current: DesignDocument } | null;
  saved: Record<string, SavedState>;
  history: CommandStack<DesignDocument>;
  selectDesign: (id: string) => void;
  openDesign: (id: string) => void;
  closeDesign: (id: string) => void;
  duplicateActive: (name: string) => Promise<void>;
  deleteActive: () => Promise<void>;
  retrySave: () => void;
  /** Apply a model proposal as one undoable command, or explain why not. */
  applyProposal: (proposal: StudioProposal) => ProposalApplication;
  createFromTemplate: (
    template: Template,
    name?: string,
    binding?: DesignBinding,
  ) => Promise<void>;
  selectFrame: (id: string) => void;
  selectNode: (id: string | null) => void;
  undo: () => void;
  redo: () => void;
  jump: (position: number) => void;
  updateFrame: (
    frameId: string,
    mutate: (spec: TemplateNode) => TemplateNode,
    label: string,
    mergeKey?: string,
  ) => void;
  updateTheme: (patch: Partial<TemplatePalette>, label: string, mergeKey?: string) => void;
  updateOptions: (patch: Partial<DesignOptions>, label: string) => void;
  renameDesign: (name: string, label: string) => void;
  addFrame: (device: DesignFrame["device"], name: string, label: string) => void;
  removeFrame: (frameId: string, label: string) => void;
  applyTemplate: (template: Template, label: string) => void;
  assetsFor: (designId: string) => ReferenceAsset[];
  importAsset: (file: File, fidelity?: "reference" | "captured") => Promise<void>;
  removeAsset: (assetId: string) => Promise<void>;
  resolveConflict: (choice: "keep-mine" | "take-theirs") => void;
}

export function useStudio(binding: DesignBinding = {}): StudioApi {
  // Only the first render's binding seeds the provisional document; changing
  // the route afterwards must not swap the design under the editor.
  const bindingRef = useRef(binding);
  const seed = useMemo(() => initialSeed(bindingRef.current), []);
  const [docs, setDocs] = useState<DesignDocument[]>([seed.doc]);
  const [openIds, setOpenIds] = useState<string[]>([seed.doc.id]);
  const [activeId, setActiveId] = useState(seed.doc.id);
  const [activeFrameId, setActiveFrameId] = useState(seed.doc.frames[0]?.id ?? "");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveFailed, setSaveFailed] = useState(false);
  const [unavailable, setUnavailable] = useState(0);
  const [fallback, setFallback] = useState(false);
  const [conflict, setConflict] = useState<{ designId: string; current: DesignDocument } | null>(
    null,
  );
  const syncRef = useRef<StudioSyncApi | null>(null);
  const [saved, setSaved] = useState<Record<string, SavedState>>({
    [seed.doc.id]: { revision: seed.doc.revision, at: Date.now(), signature: seed.signature },
  });
  const [historyMap, setHistoryMap] = useState<Record<string, CommandStack<DesignDocument>>>({});
  const [assets, setAssets] = useState<Record<string, DesignAsset[]>>({});
  const assetRequests = useRef(new Set<string>());

  const docsRef = useRef(docs);
  docsRef.current = docs;
  const activeIdRef = useRef(activeId);
  activeIdRef.current = activeId;
  const activeFrameRef = useRef(activeFrameId);
  activeFrameRef.current = activeFrameId;
  const selectedRef = useRef(selectedId);
  selectedRef.current = selectedId;
  const historyRef = useRef(historyMap);
  historyRef.current = historyMap;
  const savedRef = useRef(saved);
  savedRef.current = saved;
  const openIdsRef = useRef(openIds);
  openIdsRef.current = openIds;

  useEffect(() => {
    let cancelled = false;
    void loadDesigns()
      .then((loaded) => {
        if (cancelled) return;
        const remembered = (() => {
          try {
            return window.localStorage.getItem(ACTIVE_DESIGN_KEY);
          } catch {
            return null;
          }
        })();
        const first = loaded.docs.find((doc) => doc.id === remembered) ?? loaded.docs[0];
        if (!first) return;
        setDocs(loaded.docs);
        setOpenIds(loaded.docs.map((doc) => doc.id));
        setActiveId(first.id);
        setActiveFrameId(first.frames[0]?.id ?? "");
        setSelectedId(null);
        setSaved(Object.fromEntries(loaded.docs.map((doc) => [doc.id, savedFromDoc(doc)])));
        setUnavailable(loaded.failed);
        setFallback(designsUsingFallback());
        setReady(true);
      })
      .catch(() => {
        if (!cancelled) setReady(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const commit = useCallback(
    (
      designId: string,
      label: string,
      mutate: (current: DesignDocument) => DesignDocument,
      mergeKey?: string,
    ) => {
      const current = docsRef.current.find((doc) => doc.id === designId);
      if (!current) return;
      const next = mutate(current);
      if (docSignature(next) === docSignature(current)) return;
      const stack = pushCommand(historyRef.current[designId] ?? emptyStack<DesignDocument>(), {
        label,
        before: current,
        after: next,
        mergeKey,
      });
      const nextDocs = docsRef.current.map((doc) => (doc.id === designId ? next : doc));
      const nextHistory = { ...historyRef.current, [designId]: stack };
      docsRef.current = nextDocs;
      historyRef.current = nextHistory;
      setDocs(nextDocs);
      setHistoryMap(nextHistory);
      setSaveFailed(false);
    },
    [],
  );

  const applyProposal = useCallback(
    (proposal: StudioProposal): ProposalApplication => {
      const current = docsRef.current.find((doc) => doc.id === activeIdRef.current);
      if (!current) {
        return { ok: false, reason: "invalid", detail: "There is no active design." };
      }
      // The design id and signature are the staleness proof, not the revision:
      // an autosave of an unchanged document bumps the revision without
      // changing a layer, but a tab switch changes the design entirely.
      if (proposal.designId !== current.id || proposal.signature !== docSignature(current)) {
        return {
          ok: false,
          reason: "stale",
          detail: "The design changed after this proposal was made.",
        };
      }
      try {
        const applied = applyPatch(current, proposal.operations);
        commit(current.id, `AI · ${proposal.summary}`, () => applied.doc);
        return { ok: true };
      } catch (error) {
        return {
          ok: false,
          reason: "invalid",
          detail: error instanceof PatchError ? error.reason : "The patch could not be applied.",
        };
      }
    },
    [commit],
  );

  const activeDoc = docs.find((doc) => doc.id === activeId) ?? docs[0];
  const activeFrame = activeDoc
    ? (activeDoc.frames.find((frame) => frame.id === activeFrameId) ?? activeDoc.frames[0])
    : undefined;
  const openDocs = useMemo(
    () =>
      openIds.flatMap((id) => {
        const doc = docs.find((entry) => entry.id === id);
        return doc ? [doc] : [];
      }),
    [docs, openIds],
  );

  useEffect(() => {
    // Wait for the stored designs: the provisional "studio-draft" id must not
    // overwrite the remembered design before the load reads it.
    if (!ready || !activeId) return;
    try {
      window.localStorage.setItem(ACTIVE_DESIGN_KEY, activeId);
    } catch {
      /* private mode: remembering the tab is a convenience, not state */
    }
  }, [activeId, ready]);

  const selectDesign = useCallback((id: string) => {
    const doc = docsRef.current.find((entry) => entry.id === id);
    if (!doc) return;
    setOpenIds((prev) => (prev.includes(id) ? prev : [...prev, id]));
    setActiveId(id);
    setActiveFrameId((prev) => (doc.frames.some((frame) => frame.id === prev) ? prev : (doc.frames[0]?.id ?? "")));
    setSelectedId(null);
  }, []);

  const openDesign = useCallback(
    (id: string) => {
      selectDesign(id);
    },
    [selectDesign],
  );

  const closeDesign = useCallback((id: string) => {
    if (openIdsRef.current.length <= 1) return;
    const next = openIdsRef.current.filter((entry) => entry !== id);
    openIdsRef.current = next;
    setOpenIds(next);
    if (id === activeIdRef.current) {
      const fallback = next[0];
      const doc = docsRef.current.find((entry) => entry.id === fallback);
      if (fallback && doc) {
        setActiveId(fallback);
        setActiveFrameId(doc.frames[0]?.id ?? "");
        setSelectedId(null);
      }
    }
  }, []);

  const duplicateActive = useCallback(async (name: string) => {
    const current = docsRef.current.find((doc) => doc.id === activeIdRef.current);
    if (!current) return;
    const created = await createDesign(duplicateInput(current, name));
    setDocs((prev) => [...prev, created]);
    setOpenIds((prev) => [...prev, created.id]);
    setActiveId(created.id);
    setActiveFrameId(created.frames[0]?.id ?? "");
    setSelectedId(null);
    setSaved((prev) => ({ ...prev, [created.id]: savedFromDoc(created) }));
  }, []);

  const deleteActive = useCallback(async () => {
    const current = docsRef.current.find((doc) => doc.id === activeIdRef.current);
    if (!current || docsRef.current.length <= 1) return;
    await deleteDesign(current.id);
    const remaining = docsRef.current.filter((doc) => doc.id !== current.id);
    const stillOpen = openIdsRef.current.filter((id) => id !== current.id);
    const fallback = remaining.find((doc) => stillOpen.includes(doc.id)) ?? remaining[0];
    if (!fallback) return;
    const nextOpen = stillOpen.length > 0 ? stillOpen : [fallback.id];
    docsRef.current = remaining;
    openIdsRef.current = nextOpen;
    activeIdRef.current = fallback.id;
    activeFrameRef.current = fallback.frames[0]?.id ?? "";
    selectedRef.current = null;
    setDocs(remaining);
    setOpenIds(nextOpen);
    setActiveId(fallback.id);
    setActiveFrameId(fallback.frames[0]?.id ?? "");
    setSelectedId(null);
    const without = <T,>(record: Record<string, T>): Record<string, T> =>
      Object.fromEntries(Object.entries(record).filter(([key]) => key !== current.id));
    setSaved(without);
    setHistoryMap(without);
    setAssets(without);
    syncRef.current?.notifyDesignsChanged();
  }, []);

  const createFromTemplate = useCallback(
    async (template: Template, name?: string, target: DesignBinding = bindingRef.current) => {
      const created = await createDesign(seedInput(template, name, target));
    setDocs((prev) => [...prev, created]);
    setOpenIds((prev) => [...prev, created.id]);
    setActiveId(created.id);
    setActiveFrameId(created.frames[0]?.id ?? "");
    setSelectedId(null);
      setSaved((prev) => ({ ...prev, [created.id]: savedFromDoc(created) }));
      syncRef.current?.notifyDesignsChanged();
    },
    [],
  );

  const selectFrame = useCallback((id: string) => {
    if (activeFrameRef.current !== id) setSelectedId(null);
    activeFrameRef.current = id;
    setActiveFrameId(id);
  }, []);

  const selectNode = useCallback((id: string | null) => {
    setSelectedId(id);
  }, []);

  const applyState = useCallback((designId: string, state: DesignDocument, stack: CommandStack<DesignDocument>) => {
    setDocs((prev) => prev.map((doc) => (doc.id === designId ? state : doc)));
    setHistoryMap((prev) => ({ ...prev, [designId]: stack }));
    const nextFrame =
      state.frames.find((frame) => frame.id === activeFrameRef.current) ?? state.frames[0];
    if (nextFrame) {
      activeFrameRef.current = nextFrame.id;
      setActiveFrameId(nextFrame.id);
    }
    const previous = selectedRef.current;
    const keep = previous && nextFrame ? findNode(nextFrame.spec, previous) : undefined;
    setSelectedId(keep ? keep.id : null);
    setSaveFailed(false);
  }, []);

  const undo = useCallback(() => {
    const designId = activeIdRef.current;
    const stack = historyRef.current[designId];
    if (!stack) return;
    const result = undoCommand(stack);
    if (!result) return;
    applyState(designId, result.state, result.stack);
  }, [applyState]);

  const redo = useCallback(() => {
    const designId = activeIdRef.current;
    const stack = historyRef.current[designId];
    if (!stack) return;
    const result = redoCommand(stack);
    if (!result) return;
    applyState(designId, result.state, result.stack);
  }, [applyState]);

  const jump = useCallback(
    (position: number) => {
      const designId = activeIdRef.current;
      const stack = historyRef.current[designId];
      if (!stack) return;
      const result = jumpCommand(stack, position);
      if (!result) return;
      applyState(designId, result.state, result.stack);
    },
    [applyState],
  );

  const updateFrame = useCallback(
    (
      frameId: string,
      mutate: (spec: TemplateNode) => TemplateNode,
      label: string,
      mergeKey?: string,
    ) => {
      const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
      if (!doc) return;
      commit(
        doc.id,
        label,
        (current) => ({
          ...current,
          frames: current.frames.map((frame) =>
            frame.id === frameId ? { ...frame, spec: mutate(frame.spec) } : frame,
          ),
        }),
        mergeKey,
      );
    },
    [commit],
  );

  const updateTheme = useCallback(
    (patch: Partial<TemplatePalette>, label: string, mergeKey?: string) => {
      const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
      if (!doc) return;
      commit(
        doc.id,
        label,
        (current) => ({ ...current, theme: { ...current.theme, ...patch } }),
        mergeKey,
      );
    },
    [commit],
  );

  const updateOptions = useCallback(
    (patch: Partial<DesignOptions>, label: string) => {
      const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
      if (!doc) return;
      // Consecutive edits of the same option are one undo step; two different
      // options are two, because they read as separate decisions.
      const mergeKey = `options:${Object.keys(patch).sort().join(",")}`;
      commit(
        doc.id,
        label,
        (current) => ({
          ...current,
          options: { ...(current.options ?? {}), ...patch },
        }),
        mergeKey,
      );
    },
    [commit],
  );

  const renameDesign = useCallback(
    (name: string, label: string) => {
      const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
      if (!doc) return;
      commit(doc.id, label, (current) => ({ ...current, name }), "design-name");
    },
    [commit],
  );

  const addFrame = useCallback(
    (device: DesignFrame["device"], name: string, label: string) => {
      const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
      if (!doc) return;
      const source = doc.frames.find((frame) => frame.id === activeFrameRef.current) ?? doc.frames[0];
      if (!source) return;
      const frame = anonymousFrame(source, device, name);
      commit(doc.id, label, (current) => ({ ...current, frames: [...current.frames, frame] }));
      setActiveFrameId(frame.id);
      setSelectedId(null);
    },
    [commit],
  );

  const removeFrame = useCallback(
    (frameId: string, label: string) => {
      const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
      if (!doc || doc.frames.length <= 1) return;
      commit(doc.id, label, (current) => ({
        ...current,
        frames: current.frames.filter((frame) => frame.id !== frameId),
      }));
      setActiveFrameId((prev) => {
        if (prev !== frameId) return prev;
        return doc.frames.find((frame) => frame.id !== frameId)?.id ?? "";
      });
      setSelectedId(null);
    },
    [commit],
  );

  const applyTemplate = useCallback(
    (template: Template, label: string) => {
      const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
      if (!doc) return;
      const input = seedInput(template);
      commit(doc.id, label, (current) => ({
        ...current,
        name: input.name,
        theme: input.theme,
        frames: input.frames,
        // A template owns its chrome too; keeping the old radius while the
        // frames change would mix two compositions.
        options: { radius: template.radius, display: template.display },
      }));
      setActiveFrameId(input.frames[0]?.id ?? "");
      setSelectedId(null);
    },
    [commit],
  );

  const adoptRemote = useCallback((remote: DesignDocument) => {
    setDocs((prev) => prev.map((doc) => (doc.id === remote.id ? remote : doc)));
    setSaved((prev) => ({ ...prev, [remote.id]: savedFromDoc(remote) }));
    // The old command history describes a document that no longer exists on
    // the server, so keeping it would let Undo resurrect stale content.
    setHistoryMap((prev) => ({ ...prev, [remote.id]: emptyStack<DesignDocument>() }));
    setSelectedId(null);
    setSaveFailed(false);
  }, []);

  const reloadActive = useCallback(async () => {
    const id = activeIdRef.current;
    if (!id) return;
    try {
      adoptRemote(await getDesign(id));
      setConflict(null);
    } catch {
      // Stay on the local copy; the next broadcast or save will try again.
    }
  }, [adoptRemote]);

  const openExternalConflict = useCallback(async () => {
    const id = activeIdRef.current;
    if (!id) return;
    try {
      setConflict({ designId: id, current: await getDesign(id) });
    } catch {
      // Offline: the next save is still guarded by the revision.
    }
  }, []);

  const refreshSummaries = useCallback(() => {
    void listDesigns()
      .then(async (summaries) => {
        if (designsUsingFallback()) {
          // The local cache is not the server's list. Treating it as one would
          // delete every design the cache has not seen, so stop here.
          setFallback(true);
          return;
        }
        const listed = new Set(summaries.map((summary) => summary.id));
        // A document the server no longer lists was deleted elsewhere. Drop it
        // only when it has no unsaved edits: a dirty document is evidence the
        // person is still working, in which case a later save surfaces the
        // loss instead of the tab silently discarding the work.
        const removable = docsRef.current.filter(
          (doc) => !listed.has(doc.id) && savedRef.current[doc.id]?.signature === docSignature(doc),
        );
        if (removable.length > 0) {
          const gone = new Set(removable.map((doc) => doc.id));
          const remaining = docsRef.current.filter((doc) => !gone.has(doc.id));
          // Never empty the studio on one signal; a list response that is
          // briefly partial must not delete the only design on screen.
          if (remaining.length > 0) {
            docsRef.current = remaining;
            setDocs(remaining);
            setOpenIds((prev) => prev.filter((id) => !gone.has(id)));
            setSaved((prev) => Object.fromEntries(Object.entries(prev).filter(([id]) => !gone.has(id))));
            setHistoryMap((prev) => Object.fromEntries(Object.entries(prev).filter(([id]) => !gone.has(id))));
            setAssets((prev) => Object.fromEntries(Object.entries(prev).filter(([id]) => !gone.has(id))));
            const fallback = remaining[0];
            if (gone.has(activeIdRef.current) && fallback) {
              activeIdRef.current = fallback.id;
              activeFrameRef.current = fallback.frames[0]?.id ?? "";
              setActiveId(fallback.id);
              setActiveFrameId(fallback.frames[0]?.id ?? "");
              setSelectedId(null);
            }
          }
        }

        const known = new Set(docsRef.current.map((doc) => doc.id));
        const missing = summaries.filter((summary) => !known.has(summary.id));
        if (missing.length === 0) return;
        const settled = await Promise.allSettled(missing.map((summary) => getDesign(summary.id)));
        const added = settled.flatMap((result) => (result.status === "fulfilled" ? [result.value] : []));
        if (added.length === 0) return;
        const next = [...docsRef.current, ...added].sort((left, right) =>
          right.updatedAt.localeCompare(left.updatedAt),
        );
        docsRef.current = next;
        setDocs(next);
        setSaved((prev) => ({
          ...prev,
          ...Object.fromEntries(added.map((doc) => [doc.id, savedFromDoc(doc)])),
        }));
        // New designs appear in the studio's list but do not steal the tab the
        // person is editing.
      })
      .catch(() => {
        // A failed list refetch is not worth a banner; the next event retries.
      });
  }, []);

  const activeForSync = docs.find((doc) => doc.id === activeId) ?? docs[0];
  const activeSaved = activeForSync ? saved[activeForSync.id] : undefined;
  // A browser-only write is not durable, even though its signature matches.
  // Treat it as dirty for reconciliation so a remote save cannot silently
  // replace the only copy that exists.
  const activeDirty = activeForSync
    ? activeSaved?.signature !== docSignature(activeForSync) || activeSaved.local === true
    : false;
  const sync = useStudioSync({
    activeDesignId: activeForSync?.id ?? null,
    activeRevision: activeForSync?.revision ?? 0,
    dirty: activeDirty,
    saving,
    onReloadDesign: () => {
      void reloadActive();
    },
    onConflict: () => {
      void openExternalConflict();
    },
    onDesignsChanged: refreshSummaries,
    onAssetsChanged: (designId) => {
      if (designId !== activeIdRef.current) return;
      void listDesignAssets(designId)
        .then((list) => setAssets((prev) => ({ ...prev, [designId]: list })))
        .catch(() => {
          // Keep the current list; the pane's refresh action still works.
        });
    },
  });
  syncRef.current = sync;

  const persist = useCallback(async (designId: string) => {
    const doc = docsRef.current.find((entry) => entry.id === designId);
    if (!doc) return;
    const signature = docSignature(doc);
    const previous = savedRef.current[designId];
    // A local-only draft has the same signature as the document, so the
    // signature alone would stop the reconnect retry before it ran.
    if (previous?.signature === signature && !previous.local) return;
    try {
      const write = await putDesign(doc);
      const persisted = write.design;
      // Adopt the server-owned revision so the next save is based on the row
      // that actually exists. docSignature ignores revision, so this does not
      // mark the design dirty again.
      setDocs((prev) =>
        prev.map((entry) =>
          entry.id === designId
            ? { ...entry, revision: persisted.revision, updatedAt: persisted.updatedAt }
            : entry,
        ),
      );
      setSaved((prev) => ({
        ...prev,
        [designId]: { revision: persisted.revision, at: Date.now(), signature, local: write.local },
      }));
      if (!write.local) {
        // The revision the server assigned is what another tab compares
        // against; a signal for a browser-only write would be a lie.
        syncRef.current?.notifyDesignsChanged({ designId, revision: persisted.revision });
      }
      setSaveFailed(false);
      setFallback(write.local || designsUsingFallback());
    } catch (error) {
      if (error instanceof DesignConflictError) {
        setConflict({ designId, current: error.current });
        // Do not keep retrying a save the server will reject; the conflict
        // resolver decides which document wins.
        throw new Error("save conflict");
      }
      setSaveFailed(true);
      setFallback(designsUsingFallback());
      throw error instanceof Error ? error : new Error("save failed");
    }
  }, []);

  const saveQueue = useMemo(() => createSaveQueue(persist, setSaving), [persist]);

  useEffect(() => {
    if (!ready) return;
    // Autosave is per dirty document, not per "last edited" document:
    // switching design tabs before the debounce fired must not strand the
    // previous design's unsaved edits until it is opened again.
    const dirty = docs.filter(
      (doc) =>
        doc.id !== conflict?.designId && saved[doc.id]?.signature !== docSignature(doc),
    );
    if (dirty.length === 0) return;
    const timer = window.setTimeout(() => {
      for (const doc of dirty) saveQueue.run(doc.id);
    }, AUTOSAVE_DELAY_MS);
    return () => {
      window.clearTimeout(timer);
    };
  }, [docs, ready, saved, saveQueue, conflict?.designId]);

  useEffect(() => () => saveQueue.dispose(), [saveQueue]);

  // A write that only reached this browser is retried when the network comes
  // back. Signature equality must not block it: `persist` knows about `local`.
  useEffect(() => {
    const retryLocal = () => {
      for (const [designId, state] of Object.entries(savedRef.current)) {
        if (state.local) saveQueue.run(designId);
      }
    };
    window.addEventListener("online", retryLocal);
    return () => window.removeEventListener("online", retryLocal);
  }, [saveQueue]);

  const retrySave = useCallback(() => {
    saveQueue.run(activeIdRef.current);
  }, [saveQueue]);

  const assetsFor = useCallback(
    (designId: string) => assets[designId] ?? [],
    [assets],
  );

  // Assets live on the server with the design. Load them the first time a
  // design becomes active; the request set prevents a refetch loop when the
  // state update re-renders the pane.
  useEffect(() => {
    if (!ready) return;
    if (assetRequests.current.has(activeId)) return;
    assetRequests.current.add(activeId);
    void listDesignAssets(activeId)
      .then((list) => setAssets((prev) => ({ ...prev, [activeId]: list })))
      .catch(() => {
        // Leave the list empty; the pane shows its empty state and the next
        // successful action will refill it.
        assetRequests.current.delete(activeId);
      });
  }, [activeId, ready]);

  const importAsset = useCallback(async (file: File, fidelity: "reference" | "captured" = "reference") => {
    assertImageUpload(file);
    const designId = activeIdRef.current;
    const asset = await uploadDesignAsset(designId, file, fidelity);
    setAssets((prev) => ({ ...prev, [designId]: [asset, ...(prev[designId] ?? [])] }));
    syncRef.current?.notifyAssetsChanged(designId);
  }, []);

  const removeAsset = useCallback(async (assetId: string) => {
    const designId = activeIdRef.current;
    await deleteDesignAsset(designId, assetId);
    setAssets((prev) => ({
      ...prev,
      [designId]: (prev[designId] ?? []).filter((asset) => asset.id !== assetId),
    }));
    syncRef.current?.notifyAssetsChanged(designId);
  }, []);

  const resolveConflict = useCallback(
    (choice: "keep-mine" | "take-theirs") => {
      if (!conflict) return;
      const { designId, current } = conflict;
      if (choice === "take-theirs") {
        setDocs((prev) => prev.map((doc) => (doc.id === designId ? current : doc)));
        setSaved((prev) => ({ ...prev, [designId]: savedFromDoc(current) }));
        setHistoryMap((prev) => ({ ...prev, [designId]: emptyStack<DesignDocument>() }));
        setSelectedId(null);
      } else {
        // Keep the local document but rebase it on the server's revision, then
        // save immediately so the resolution is durable. The saved signature
        // must be cleared in the same step: without it `persist` sees an
        // unchanged document and returns before the queued save runs.
        const local = docsRef.current.find((doc) => doc.id === designId);
        if (local) {
          const rebased = { ...local, revision: current.revision };
          const nextDocs = docsRef.current.map((doc) => (doc.id === designId ? rebased : doc));
          docsRef.current = nextDocs;
          setDocs(nextDocs);
        }
        const entry = savedRef.current[designId];
        if (entry) {
          const rebasedEntry = { ...entry, revision: current.revision, signature: "" };
          savedRef.current = { ...savedRef.current, [designId]: rebasedEntry };
          setSaved((prev) => ({ ...prev, [designId]: rebasedEntry }));
        }
        window.setTimeout(() => saveQueue.run(designId), 0);
      }
      setConflict(null);
      setSaveFailed(false);
    },
    [conflict, saveQueue],
  );

  return {
    ready,
    docs,
    openDocs,
    activeDoc,
    activeFrame,
    selectedId,
    saving,
    saveFailed,
    unavailable,
    fallback,
    conflict,
    saved,
    history: historyMap[activeId] ?? emptyStack<DesignDocument>(),
    selectDesign,
    openDesign,
    closeDesign,
    duplicateActive,
    deleteActive,
    retrySave,
    applyProposal,
    resolveConflict,
    createFromTemplate,
    selectFrame,
    selectNode,
    undo,
    redo,
    jump,
    updateFrame,
    updateTheme,
    updateOptions,
    renameDesign,
    addFrame,
    removeFrame,
    applyTemplate,
    assetsFor,
    importAsset,
    removeAsset,
  };
}
