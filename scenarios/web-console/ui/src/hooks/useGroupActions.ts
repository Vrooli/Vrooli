import { useCallback } from "react";
import { useWorkspaceStore, type RoleMeta, type TabGroupMeta } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "./useWorkspaceSync";
import { nextGroupColor } from "../lib/paneColor";
import { captureGroupSnapshot, isGroupClosable } from "../lib/groupLifecycle";
import { createRole } from "../api/workspaceRoles";

// [REQ:P0-014c] Group Assignment And Administration Split
// [REQ:P0-014f] Group Auto-Close With Undo

/** How long a closed group stays undoable. */
export const UNDO_WINDOW_MS = 10_000;

/**
 * Shared tab-group operations, each pairing the local store mutation with its
 * backend sync. One implementation serves the Manage Groups drawer, TabBar,
 * SessionSidebar, and the launcher, so the ungroup-then-delete and
 * server-first-create flows are never duplicated per surface.
 */
export function useGroupActions() {
  const addGroup = useWorkspaceStore((s) => s.addGroup);
  const removeGroup = useWorkspaceStore((s) => s.removeGroup);
  const setPaneGroup = useWorkspaceStore((s) => s.setPaneGroup);
  const addRole = useWorkspaceStore((s) => s.addRole);
  const setClosedGroupUndo = useWorkspaceStore((s) => s.setClosedGroupUndo);
  const { syncCreateGroup, syncDeleteGroup, syncPaneUpdate, syncPaneOrder, syncPaneMove } = useWorkspaceSync();

  /**
   * Assign a pane to a group. The store repositions it into the group's block
   * and may seed its color from the group, so this syncs the order and the
   * pane patch together via syncPaneMove.
   */
  const assignPaneToGroup = useCallback((sessionId: string, groupId: string) => {
    setPaneGroup(sessionId, groupId);
    syncPaneMove(sessionId);
  }, [setPaneGroup, syncPaneMove]);

  /**
   * Clear one pane's group membership. Also a move: the pane leaves the
   * group's block, so the surviving members must stay contiguous.
   */
  const removePaneFromGroup = useCallback((sessionId: string) => {
    setPaneGroup(sessionId, null);
    syncPaneMove(sessionId);
  }, [setPaneGroup, syncPaneMove]);

  /** Clear group membership for every pane in a group (the group itself stays). */
  const ungroupAllMembers = useCallback((groupId: string) => {
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

  /**
   * Server-first group creation with a caller-chosen name.
   *
   * The name is a parameter because a group named for its work is the whole
   * point: creating "New Group" and making the operator rename it is two
   * steps where one would do.
   */
  const createNamedGroup = useCallback(async (name: string): Promise<TabGroupMeta> => {
    const existing = useWorkspaceStore.getState().groups;
    const serverGroup = await syncCreateGroup(
      name.trim() || "New Group",
      nextGroupColor(existing.map((g) => g.color)),
    );
    const group: TabGroupMeta = {
      id: serverGroup.id,
      name: serverGroup.name,
      color: serverGroup.color,
      isCollapsed: false,
    };
    addGroup(group);
    return group;
  }, [syncCreateGroup, addGroup]);

  /** The unnamed case the drawer's "New group" button still wants. */
  const createGroup = useCallback(() => createNamedGroup("New Group"), [createNamedGroup]);

  /**
   * Close a group, keeping everything needed to put it back.
   *
   * The snapshot is captured BEFORE anything is deleted. Deleting first and
   * reconstructing afterwards would lose the roles, which cascade in the
   * database and cannot be read back once the group is gone.
   */
  const closeGroup = useCallback((groupId: string) => {
    const { groups, panes, roles } = useWorkspaceStore.getState();
    const snapshot = captureGroupSnapshot(groupId, groups, panes, roles);

    ungroupAllMembers(groupId);
    removeGroup(groupId);
    syncDeleteGroup(groupId);

    if (snapshot) setClosedGroupUndo(snapshot);
  }, [removeGroup, setClosedGroupUndo, syncDeleteGroup, ungroupAllMembers]);

  /** Hard-delete a group. Kept as the non-undoable name for existing callers. */
  const deleteGroup = useCallback((groupId: string) => {
    ungroupAllMembers(groupId);
    removeGroup(groupId);
    syncDeleteGroup(groupId);
  }, [ungroupAllMembers, removeGroup, syncDeleteGroup]);

  /**
   * Put back the most recently closed group.
   *
   * Replays create-group, then create-role per role, then re-assign each
   * surviving pane. The snapshot is cleared ONLY after every step resolves:
   * a half-replayed restore that dropped its source would leave a group with
   * some of its roles and no way to recover the rest.
   */
  const restoreClosedGroup = useCallback(async (): Promise<boolean> => {
    const snapshot = useWorkspaceStore.getState().closedGroupUndo;
    if (!snapshot) return false;

    try {
      const serverGroup = await syncCreateGroup(snapshot.group.name, snapshot.group.color);
      const restored: TabGroupMeta = {
        id: serverGroup.id,
        name: serverGroup.name,
        color: serverGroup.color,
        isCollapsed: snapshot.group.isCollapsed,
      };
      addGroup(restored);

      // Roles are created sequentially: their order is content, and a
      // concurrent burst would let the server assign sort orders in whatever
      // sequence the responses happen to land.
      for (const role of snapshot.roles) {
        const created = await createRole({
          group_id: restored.id,
          label: role.label,
          command: role.command,
          working_dir: role.workingDir,
          incoming_prompt: role.incomingPrompt,
          backend: role.backend,
          target_id: role.targetId,
          // A role whose session is gone comes back waiting. Pointing it at a
          // dead session id would be worse than an honest placeholder.
          session_id: livePaneFor(role) ? role.sessionId : null,
          sort_order: role.sortOrder,
        });
        const meta: RoleMeta = {
          id: created.id,
          groupId: created.group_id,
          label: created.label,
          command: created.command,
          workingDir: created.working_dir,
          incomingPrompt: created.incoming_prompt,
          backend: created.backend,
          targetId: created.target_id,
          sessionId: created.session_id,
          sortOrder: created.sort_order,
        };
        addRole(meta);
      }

      for (const sessionId of snapshot.memberSessionIds) {
        // Only panes that still exist: a session closed during the undo
        // window is genuinely gone and must not be resurrected as a member.
        if (!useWorkspaceStore.getState().panes.some((p) => p.sessionId === sessionId)) continue;
        setPaneGroup(sessionId, restored.id);
        syncPaneUpdate(sessionId, { group_id: restored.id });
      }
      const { panes, activePane } = useWorkspaceStore.getState();
      syncPaneOrder(panes.map((p) => p.sessionId), activePane);

      setClosedGroupUndo(null);
      return true;
    } catch (error) {
      console.error("Failed to restore closed group:", error);
      // Deliberately keep the snapshot: the operator can try again, and a
      // silent drop here would lose the only copy of the group.
      return false;
    }
  }, [addGroup, addRole, setClosedGroupUndo, setPaneGroup, syncCreateGroup, syncPaneOrder, syncPaneUpdate]);

  const dismissClosedGroupUndo = useCallback(() => {
    setClosedGroupUndo(null);
  }, [setClosedGroupUndo]);

  /**
   * Close a group if, and only if, it is finished.
   *
   * Scoped to ONE group id on purpose: this runs whenever a pane goes away,
   * and a scan across every group would cost on every session change.
   */
  const closeGroupIfFinished = useCallback((groupId: string | null) => {
    if (!groupId) return;
    const { autoCloseEmptyGroups, groups, panes, roles } = useWorkspaceStore.getState();
    if (!autoCloseEmptyGroups) return;
    if (!groups.some((g) => g.id === groupId)) return;
    if (!isGroupClosable(groupId, panes, roles)) return;
    closeGroup(groupId);
  }, [closeGroup]);

  return {
    assignPaneToGroup,
    removePaneFromGroup,
    ungroupAllMembers,
    deleteGroup,
    closeGroup,
    closeGroupIfFinished,
    restoreClosedGroup,
    dismissClosedGroupUndo,
    createGroup,
    createNamedGroup,
  };
}

/** True when the role's session still has a pane, so it can come back running. */
function livePaneFor(role: RoleMeta): boolean {
  if (!role.sessionId) return false;
  return useWorkspaceStore.getState().panes.some((p) => p.sessionId === role.sessionId);
}
