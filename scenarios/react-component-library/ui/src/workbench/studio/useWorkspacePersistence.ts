import { useCallback, useEffect, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { Code, ConnectError } from "@connectrpc/connect";

import { getWorkspace, saveWorkspace, type Workspace } from "../../api/workspaces";
import { emptyStack, type CommandStack } from "../studio/history";
import type { SavedState } from "./persistence";

interface StudioModel {
  workspace: Workspace | null;
  history: CommandStack<Workspace>;
}

interface WorkspacePersistenceOptions {
  workspaceId: string;
  workspace: Workspace | null;
  workspaceRef: MutableRefObject<Workspace | null>;
  pageIdRef: MutableRefObject<string>;
  displayIdRef: MutableRefObject<string>;
  signature: string;
  readOnly: boolean;
  setModel: Dispatch<SetStateAction<StudioModel>>;
  setPageId: Dispatch<SetStateAction<string>>;
  setDisplayId: Dispatch<SetStateAction<string>>;
  setStateByDisplay: Dispatch<SetStateAction<Record<string, string>>>;
  setSelectedId: Dispatch<SetStateAction<string | null>>;
}

/** Owns initial loading, serialized persistence, conflict recovery, and flush lifecycle. */
export function useWorkspacePersistence({
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
}: WorkspacePersistenceOptions) {
  const [ready, setReady] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saved, setSaved] = useState<SavedState | undefined>(undefined);
  const [saving, setSaving] = useState(false);
  const [saveFailed, setSaveFailed] = useState(false);
  const [conflict, setConflict] = useState<{ current: Workspace } | null>(null);
  const dirty = !readOnly && Boolean(saved && workspace && signature !== saved.signature);
  const revisionRef = useRef(0);
  const inFlight = useRef<Promise<void> | null>(null);
  const conflictRef = useRef<{ current: Workspace } | null>(null);
  conflictRef.current = conflict;
  const readOnlyRef = useRef(false);
  readOnlyRef.current = readOnly;
  const dirtyRef = useRef(false);
  dirtyRef.current = dirty;
  const saveRef = useRef<() => Promise<void>>(() => Promise.resolve());
  // A failed write is retried only after another edit or explicit save.
  const failedSignature = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setReady(false);
    setLoadError(null);
    setConflict(null);
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
        revisionRef.current = loaded.revision;
        setPageId(firstPage?.id ?? "");
        setDisplayId(firstPage?.displays[0]?.id ?? "");
        setStateByDisplay({});
        setSelectedId(null);
        setSaved({ revision: loaded.revision, at: Date.now(), signature: workspaceSignature(loaded) });
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
  }, [setDisplayId, setModel, setPageId, setSelectedId, setStateByDisplay, workspaceId]);

  const save = useCallback(async () => {
    if (inFlight.current) return inFlight.current;
    // A conflict is a decision point. Never overwrite the other writer silently.
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
        setSaved({ revision: next.revision, at: Date.now(), signature: signatureSent });
        failedSignature.current = null;
        setConflict(null);
        setModel((current) => {
          if (!current.workspace) return current;
          if (current.workspace === snapshot) return { workspace: next, history: current.history };
          // Preserve edits made during this request, while advancing the base revision.
          return {
            workspace: { ...current.workspace, revision: next.revision },
            history: current.history,
          };
        });
      } catch (cause) {
        const error = ConnectError.from(cause);
        if (error.code === Code.Aborted) {
          setConflict({ current: snapshot });
          try {
            const latest = await getWorkspace(snapshot.id);
            if (latest) setConflict({ current: latest });
          } catch {
            // Keep the conflict visible even if refreshing its details fails.
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
  }, [setModel, workspaceRef]);

  const syncRemote = useCallback(
    async (provided?: Workspace) => {
      const current = provided ?? (await getWorkspace(workspaceId));
      if (!current) throw new Error("Workspace could not be reloaded after capture.");
      const nextPage = current.pages.some((entry) => entry.id === pageIdRef.current)
        ? pageIdRef.current
        : (current.pages[0]?.id ?? "");
      const page = current.pages.find((entry) => entry.id === nextPage);
      const nextDisplay = page?.displays.some((entry) => entry.id === displayIdRef.current)
        ? displayIdRef.current
        : (page?.displays[0]?.id ?? "");
      setModel({ workspace: current, history: emptyStack() });
      revisionRef.current = current.revision;
      setPageId(nextPage);
      setDisplayId(nextDisplay);
      setSaved({ revision: current.revision, at: Date.now(), signature: workspaceSignature(current) });
      setConflict(null);
      setSaveFailed(false);
    },
    [displayIdRef, pageIdRef, setDisplayId, setModel, setPageId, workspaceId],
  );

  useEffect(() => {
    if (!ready || conflict || readOnly || !dirty) return;
    if (failedSignature.current === signature) return;
    const timer = globalThis.setTimeout(() => void save(), 900);
    return () => globalThis.clearTimeout(timer);
  }, [conflict, dirty, ready, readOnly, save, signature, workspace]);

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
        revisionRef.current = current.revision;
        setPageId(current.pages[0]?.id ?? "");
        setDisplayId(current.pages[0]?.displays[0]?.id ?? "");
        setSaved({ revision: current.revision, at: Date.now(), signature: workspaceSignature(current) });
        setConflict(null);
        return;
      }
      revisionRef.current = current.revision;
      setModel((model) =>
        model.workspace
          ? { ...model, workspace: { ...model.workspace, revision: current.revision } }
          : model,
      );
      setConflict(null);
    },
    [conflict, setDisplayId, setModel, setPageId],
  );

  return { ready, loadError, saved, saving, saveFailed, conflict, dirty, save, syncRemote, resolveConflict };
}

/** Stable content signature for the whole workspace, excluding revision metadata. */
export function workspaceSignature(workspace: Workspace): string {
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
