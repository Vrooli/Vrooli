import { useCallback, useRef } from "react";
import {
  saveWorkspaceLayout,
  updateWorkspacePane,
  createTabGroup,
  updateTabGroup,
  deleteTabGroup,
  type WorkspacePaneDTO,
  type TabGroupDTO,
} from "../api/workspace";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";


/** Debounce delay (ms) for pane reorder saves. */
const REORDER_DEBOUNCE_MS = 300;

/**
 * Hook that syncs workspace mutations to the backend.
 * All operations are fire-and-forget: Zustand is updated immediately
 * for instant UI response, and the backend call follows asynchronously.
 */
export function useWorkspaceSync() {
  const reorderTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  /** Debounced save of pane order + active pane to backend. */
  const syncPaneOrder = useCallback((paneOrder: string[], activePane: string | null) => {
    if (reorderTimer.current) clearTimeout(reorderTimer.current);
    reorderTimer.current = setTimeout(() => {
      reorderTimer.current = null;
      saveWorkspaceLayout({ active_pane: activePane, pane_order: paneOrder }).catch((err) =>
        console.error("Failed to sync pane order:", err),
      );
    }, REORDER_DEBOUNCE_MS);
  }, []);

  /**
   * Persist the outcome of a pane move: the new order, plus the moved pane's
   * group and color, which a drop can change (see the store's
   * groupIdForDropPosition / withGroupAssigned).
   *
   * Every reorder surface — sidebar grip, tab strip, grid arrange-drag, the
   * touch context menu's Move Up/Down — funnels through here so none of them
   * can persist an order while silently dropping the membership change that
   * came with it. Reads the store directly because the caller has just
   * mutated it and any props it holds are a render behind.
   */
  const syncPaneMove = useCallback((sessionId: string) => {
    const { panes, activePane } = useWorkspaceStore.getState();
    syncPaneOrder(panes.map((pane) => pane.sessionId), activePane);
    const moved = panes.find((pane) => pane.sessionId === sessionId);
    if (!moved) return;
    updateWorkspacePane(sessionId, {
      group_id: moved.groupId,
      header_color: moved.headerColor,
    }).catch((err) => console.error("Failed to sync pane move:", err));
  }, [syncPaneOrder]);

  /** Immediate save of a single pane's metadata. */
  const syncPaneUpdate = useCallback(
    (sessionId: string, update: Partial<Omit<WorkspacePaneDTO, "session_id">>) => {
      updateWorkspacePane(sessionId, update).catch((err) =>
        console.error("Failed to sync pane update:", err),
      );
    },
    [],
  );

  /** Awaitable bulk save of the same metadata patch to many panes. Returns
   *  the session ids whose save failed so the caller can surface the partial
   *  failure (unlike the fire-and-forget single-pane path). */
  const syncPaneUpdates = useCallback(
    async (
      sessionIds: string[],
      update: Partial<Omit<WorkspacePaneDTO, "session_id">>,
    ): Promise<string[]> => {
      const results = await Promise.allSettled(
        sessionIds.map((id) => updateWorkspacePane(id, update)),
      );
      const failed: string[] = [];
      results.forEach((result, i) => {
        if (result.status === "rejected") {
          const id = sessionIds[i];
          if (id) failed.push(id);
          console.error("Failed to sync pane update:", result.reason);
        }
      });
      return failed;
    },
    [],
  );

  /** Immediate save of active pane change. */
  const syncActivePane = useCallback((paneOrder: string[], activePane: string | null) => {
    saveWorkspaceLayout({ active_pane: activePane, pane_order: paneOrder }).catch((err) =>
      console.error("Failed to sync active pane:", err),
    );
  }, []);

  /** Create a group on the backend and return the server-generated group. */
  const syncCreateGroup = useCallback(
    async (name: string, color: string): Promise<TabGroupDTO> => {
      return createTabGroup({ name, color });
    },
    [],
  );

  /** Update a group on the backend. */
  const syncUpdateGroup = useCallback(
    (id: string, update: Partial<Omit<TabGroupDTO, "id">>) => {
      updateTabGroup(id, update).catch((err) =>
        console.error("Failed to sync group update:", err),
      );
    },
    [],
  );

  /** Delete a group on the backend. */
  const syncDeleteGroup = useCallback((id: string) => {
    deleteTabGroup(id).catch((err) =>
      console.error("Failed to sync group delete:", err),
    );
  }, []);

  return {
    syncPaneOrder,
    syncPaneMove,
    syncPaneUpdate,
    syncPaneUpdates,
    syncActivePane,
    syncCreateGroup,
    syncUpdateGroup,
    syncDeleteGroup,
  };
}
