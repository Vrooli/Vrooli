import { useCallback } from "react";
import { useWorkspaceStore, type TabGroupMeta } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "./useWorkspaceSync";
import { nextGroupColor } from "../lib/paneColor";

/**
 * Shared tab-group operations, each pairing the local store mutation with its
 * backend sync. One implementation serves the Manage Groups drawer, TabBar,
 * and SessionSidebar so the ungroup-then-delete and server-first-create flows
 * are never duplicated per surface.
 */
export function useGroupActions() {
  const addGroup = useWorkspaceStore((s) => s.addGroup);
  const removeGroup = useWorkspaceStore((s) => s.removeGroup);
  const setPaneGroup = useWorkspaceStore((s) => s.setPaneGroup);
  const addPaneToGroup = useWorkspaceStore((s) => s.addPaneToGroup);
  const { syncCreateGroup, syncDeleteGroup, syncPaneUpdate, syncPaneOrder } = useWorkspaceSync();

  /** Assign a pane to a group, keeping the group contiguous + syncing both. */
  const assignPaneToGroup = useCallback((sessionId: string, groupId: string) => {
    addPaneToGroup(sessionId, groupId);
    syncPaneUpdate(sessionId, { group_id: groupId });
    const { panes: updated, activePane: active } = useWorkspaceStore.getState();
    syncPaneOrder(updated.map((p) => p.sessionId), active);
  }, [addPaneToGroup, syncPaneUpdate, syncPaneOrder]);

  /** Clear one pane's group membership. */
  const removePaneFromGroup = useCallback((sessionId: string) => {
    setPaneGroup(sessionId, null);
    syncPaneUpdate(sessionId, { group_id: null });
  }, [setPaneGroup, syncPaneUpdate]);

  /** Clear group membership for every pane in a group (the group itself stays). */
  const ungroupAllMembers = useCallback((groupId: string) => {
    for (const p of useWorkspaceStore.getState().panes) {
      if (p.groupId === groupId) {
        setPaneGroup(p.sessionId, null);
        syncPaneUpdate(p.sessionId, { group_id: null });
      }
    }
  }, [setPaneGroup, syncPaneUpdate]);

  /** Hard-delete a group: ungroup its members, then drop it locally + remotely. */
  const deleteGroup = useCallback((groupId: string) => {
    ungroupAllMembers(groupId);
    removeGroup(groupId);
    syncDeleteGroup(groupId);
  }, [ungroupAllMembers, removeGroup, syncDeleteGroup]);

  /**
   * Server-first group creation: the backend generates the id, then the local
   * store adopts it. Never fabricate group ids client-side.
   */
  const createGroup = useCallback(async (): Promise<TabGroupMeta> => {
    const existing = useWorkspaceStore.getState().groups;
    const serverGroup = await syncCreateGroup("New Group", nextGroupColor(existing.map((g) => g.color)));
    const group: TabGroupMeta = {
      id: serverGroup.id,
      name: serverGroup.name,
      color: serverGroup.color,
      isCollapsed: false,
    };
    addGroup(group);
    return group;
  }, [syncCreateGroup, addGroup]);

  return { assignPaneToGroup, removePaneFromGroup, ungroupAllMembers, deleteGroup, createGroup };
}
