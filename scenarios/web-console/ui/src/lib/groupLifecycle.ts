// When a group is finished, and what it costs to be wrong.
//
// [REQ:P0-014f] Group Auto-Close With Undo

import type { PaneMetadata, RoleMeta, TabGroupMeta } from "../stores/useWorkspaceStore";

/**
 * A group is closable when it holds no panes AND no waiting roles.
 *
 * The waiting-role exemption is the entire safety argument for closing a group
 * without asking. An empty group is usually finished work, but a group created
 * from a template whose roles have not started yet is also empty — and that
 * one is the operator's plan for the next hour, not litter. Roles are what let
 * the console tell those two apart, which is why auto-close could not ship
 * before roles existed.
 *
 * A RUNNING role does not need its own clause: a running role has a session,
 * and that session has a pane, so the pane check already covers it.
 */
export function isGroupClosable(
  groupId: string,
  panes: readonly PaneMetadata[],
  roles: readonly RoleMeta[],
): boolean {
  if (panes.some((pane) => pane.groupId === groupId)) return false;
  if (roles.some((role) => role.groupId === groupId && role.sessionId === null)) return false;
  return true;
}

/** Groups that hold no panes, for the drawer's Empty section. */
export function emptyGroups(
  groups: readonly TabGroupMeta[],
  panes: readonly PaneMetadata[],
): TabGroupMeta[] {
  return groups.filter((group) => !panes.some((pane) => pane.groupId === group.id));
}

/**
 * Everything needed to put a closed group back.
 *
 * The snapshot is captured BEFORE anything is deleted, and it is kept until
 * the replay confirms. A restore that half-succeeded and then dropped its
 * source would leave a group with some of its roles and no way to recover the
 * rest, which is worse than not offering undo at all.
 */
export interface ClosedGroupSnapshot {
  group: TabGroupMeta;
  /** Every role the group held, including running ones. */
  roles: RoleMeta[];
  /** Session ids that were in the group, so membership can be restored. */
  memberSessionIds: string[];
  /** Where the group sat, so a restore does not send it to the end. */
  sortIndex: number;
}

/** Capture the state a close is about to destroy. */
export function captureGroupSnapshot(
  groupId: string,
  groups: readonly TabGroupMeta[],
  panes: readonly PaneMetadata[],
  roles: readonly RoleMeta[],
): ClosedGroupSnapshot | null {
  const sortIndex = groups.findIndex((g) => g.id === groupId);
  const group = groups[sortIndex];
  if (!group) return null;
  return {
    group: { ...group },
    roles: roles.filter((r) => r.groupId === groupId).map((r) => ({ ...r })),
    memberSessionIds: panes.filter((p) => p.groupId === groupId).map((p) => p.sessionId),
    sortIndex: Math.max(0, sortIndex),
  };
}
