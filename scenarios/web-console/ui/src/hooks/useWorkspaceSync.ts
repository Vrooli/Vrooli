import { useCallback, useRef } from "react";
import {
  saveWorkspaceLayout,
  updateWorkspacePane,
  createTabGroup,
  updateTabGroup,
  deleteTabGroup,
  type WorkspacePaneDTO,
  type TabGroupDTO,
} from "../lib/api";


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

  /** Immediate save of a single pane's metadata. */
  const syncPaneUpdate = useCallback(
    (sessionId: string, update: Partial<Omit<WorkspacePaneDTO, "session_id">>) => {
      updateWorkspacePane(sessionId, update).catch((err) =>
        console.error("Failed to sync pane update:", err),
      );
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
    syncPaneUpdate,
    syncActivePane,
    syncCreateGroup,
    syncUpdateGroup,
    syncDeleteGroup,
  };
}
