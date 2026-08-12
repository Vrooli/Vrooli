import { useCallback } from "react";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
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
    const { syncCreateGroup, syncDeleteGroup, syncPaneUpdate, syncPaneOrder, syncPaneMove } = useWorkspaceSync();
    /**
     * Assign a pane to a group. The store repositions it into the group's block
     * and may seed its color from the group, so this syncs the order and the
     * pane patch together via syncPaneMove.
     */
    const assignPaneToGroup = useCallback((sessionId, groupId) => {
        setPaneGroup(sessionId, groupId);
        syncPaneMove(sessionId);
    }, [setPaneGroup, syncPaneMove]);
    /**
     * Clear one pane's group membership. Also a move: the pane leaves the
     * group's block, so the surviving members must stay contiguous.
     */
    const removePaneFromGroup = useCallback((sessionId) => {
        setPaneGroup(sessionId, null);
        syncPaneMove(sessionId);
    }, [setPaneGroup, syncPaneMove]);
    /** Clear group membership for every pane in a group (the group itself stays). */
    const ungroupAllMembers = useCallback((groupId) => {
        const members = useWorkspaceStore.getState().panes.filter((p) => p.groupId === groupId);
        for (const p of members) {
            setPaneGroup(p.sessionId, null);
            syncPaneUpdate(p.sessionId, { group_id: null });
        }
        // One order save for the whole batch — the members' positions are settled
        // only after the last of them has left.
        if (members.length > 0) {
            const { panes, activePane } = useWorkspaceStore.getState();
            syncPaneOrder(panes.map((p) => p.sessionId), activePane);
        }
    }, [setPaneGroup, syncPaneUpdate, syncPaneOrder]);
    /** Hard-delete a group: ungroup its members, then drop it locally + remotely. */
    const deleteGroup = useCallback((groupId) => {
        ungroupAllMembers(groupId);
        removeGroup(groupId);
        syncDeleteGroup(groupId);
    }, [ungroupAllMembers, removeGroup, syncDeleteGroup]);
    /**
     * Server-first group creation: the backend generates the id, then the local
     * store adopts it. Never fabricate group ids client-side.
     */
    const createGroup = useCallback(async () => {
        const existing = useWorkspaceStore.getState().groups;
        const serverGroup = await syncCreateGroup("New Group", nextGroupColor(existing.map((g) => g.color)));
        const group = {
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
