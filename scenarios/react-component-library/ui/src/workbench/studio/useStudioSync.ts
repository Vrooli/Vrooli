/**
 * Binds the studio to the cross-tab sync channel.
 *
 * The hook knows nothing about `useStudio`: it accepts callbacks and reports
 * what moved. That keeps the reconciliation rule in `sync.ts` testable without
 * a component and lets the coordinator choose how to reload a design (refetch
 * and adopt) or show a conflict (the existing banner).
 *
 * Two properties drive the implementation:
 *
 *  1. Bursts collapse. A save that touches one design and an asset upload can
 *     arrive together; each callback must fire once per 250 ms window, not once
 *     per message. The pending set is accumulated and drained on the timer.
 *  2. Callbacks stay current without resubscribing. Handlers live in a ref, so
 *     a render that changes `dirty` or `saving` does not tear down the channel
 *     and drop a message in the gap.
 */
import { useCallback, useEffect, useRef } from "react";

import {
  createSyncChannel,
  currentTabId,
  reconcileDesign,
  type StudioSyncEvent,
  type SyncChannel,
} from "./sync";

/** Window in which repeated signals collapse into one callback burst. */
export const STUDIO_SYNC_COALESCE_MS = 250;

export interface StudioSyncHandlers {
  activeDesignId: string | null;
  activeRevision: number;
  /** True when the active design has unsaved local edits. */
  dirty: boolean;
  saving: boolean;
  /** Refetch the active document and adopt the remote copy. */
  onReloadDesign: () => void;
  /** Show the existing conflict banner for the active design. */
  onConflict: () => void;
  /** Refetch the design summary list. */
  onDesignsChanged: () => void;
  onAssetsChanged: (designId: string) => void;
}

export interface StudioSyncApi {
  /**
   * Broadcast a design change. Passing the saved design's id and the revision
   * the server returned lets another tab reconcile the open document; omitting
   * them is the coarse "the list moved" signal used by create/delete/rename.
   */
  notifyDesignsChanged: (change?: { designId: string; revision: number }) => void;
  notifyAssetsChanged: (designId: string) => void;
  notifyTemplateChanged: () => void;
}

interface PendingSync {
  /** A coarse list move (create, delete, rename, focus). */
  list: boolean;
  /** Targeted design → highest revision seen in this window. */
  remote: Map<string, number>;
  assets: Set<string>;
}

function emptyPending(): PendingSync {
  return { list: false, remote: new Map(), assets: new Set() };
}

export function useStudioSync(handlers: StudioSyncHandlers): StudioSyncApi {
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;

  const channelRef = useRef<SyncChannel | null>(null);
  const mountedRef = useRef(false);
  const pendingRef = useRef<PendingSync>(emptyPending());
  const timerRef = useRef<number | null>(null);

  const cancelTimer = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const flush = useCallback(() => {
    timerRef.current = null;
    if (!mountedRef.current) return;

    const current = handlersRef.current;
    const pending = pendingRef.current;
    pendingRef.current = emptyPending();

    for (const designId of pending.assets) {
      // Only the active design's assets are loaded by the studio. A signal for
      // another design is not actionable until that design is opened, and
      // opening it refetches anyway.
      if (designId === current.activeDesignId) current.onAssetsChanged(designId);
    }

    const activeId = current.activeDesignId;
    const remoteRevision = activeId === null ? undefined : pending.remote.get(activeId);
    if (activeId !== null && remoteRevision !== undefined) {
      const decision = reconcileDesign({
        activeId,
        activeRevision: current.activeRevision,
        remoteId: activeId,
        remoteRevision,
        dirty: current.dirty,
        saving: current.saving,
      });
      if (decision === "reload") current.onReloadDesign();
      else if (decision === "conflict") current.onConflict();
    }

    const listMoved = pending.list || pending.remote.size > 0;
    // A save in flight owns the next authoritative revision. Refetching now
    // would race it and briefly show a revision the server is about to replace;
    // the save settlement (and its own broadcast) settles the list instead.
    if (listMoved && !current.saving) current.onDesignsChanged();
  }, []);

  const schedule = useCallback(() => {
    // `!mountedRef` keeps a straggling sender from arming a timer after
    // teardown. An already-armed timer means the window is still open.
    if (!mountedRef.current || timerRef.current !== null) return;
    timerRef.current = window.setTimeout(flush, STUDIO_SYNC_COALESCE_MS);
  }, [flush]);

  const handleEvent = useCallback(
    (event: StudioSyncEvent) => {
      if (!mountedRef.current) return;
      const pending = pendingRef.current;
      switch (event.type) {
        case "designs-changed":
          if (event.designId !== undefined && event.revision !== undefined) {
            const known = pending.remote.get(event.designId) ?? 0;
            pending.remote.set(event.designId, Math.max(known, event.revision));
          } else {
            pending.list = true;
          }
          break;
        case "assets-changed":
          pending.assets.add(event.designId);
          break;
        case "template-changed":
          // The template shelf is tab-local in this slice and this hook has no
          // template callback. Acknowledge the event without arming a timer.
          return;
      }
      schedule();
    },
    [schedule],
  );

  useEffect(() => {
    mountedRef.current = true;
    pendingRef.current = emptyPending();
    const channel = createSyncChannel();
    channelRef.current = channel;
    const unsubscribe = channel.subscribe(handleEvent);

    const requestList = () => {
      if (!mountedRef.current) return;
      pendingRef.current.list = true;
      schedule();
    };
    const onFocus = () => requestList();
    const onVisibilityChange = () => {
      // `visibilityState` is read at dispatch time: a tab that becomes hidden
      // and visible again must refetch even if focus never left.
      if (document.visibilityState === "visible") requestList();
    };

    window.addEventListener("focus", onFocus);
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      mountedRef.current = false;
      cancelTimer();
      pendingRef.current = emptyPending();
      window.removeEventListener("focus", onFocus);
      document.removeEventListener("visibilitychange", onVisibilityChange);
      unsubscribe();
      channel.close();
      channelRef.current = null;
    };
  }, [cancelTimer, handleEvent, schedule]);

  const notifyDesignsChanged = useCallback(
    (change?: { designId: string; revision: number }) => {
      channelRef.current?.post(
        change === undefined
          ? { type: "designs-changed", source: currentTabId() }
          : {
              type: "designs-changed",
              designId: change.designId,
              revision: change.revision,
              source: currentTabId(),
            },
      );
    },
    [],
  );

  const notifyAssetsChanged = useCallback((designId: string) => {
    channelRef.current?.post({ type: "assets-changed", designId, source: currentTabId() });
  }, []);

  const notifyTemplateChanged = useCallback(() => {
    channelRef.current?.post({ type: "template-changed", source: currentTabId() });
  }, []);

  return { notifyDesignsChanged, notifyAssetsChanged, notifyTemplateChanged };
}
