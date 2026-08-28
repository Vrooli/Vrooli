// Workspace domain home: Connect-RPC client, types, decoders, and
// wrappers for cross-device workspace layout sync (panes + tab groups).

import { createClient } from "@connectrpc/connect";
import { WorkspaceService } from "@vrooli/proto-types/web-console/v1/workspace/workspace_pb";

import { transport } from "./client";
import { decodeRole, type RoleDTO } from "./workspaceRoles";

export const workspaceClient = createClient(WorkspaceService, transport);

// Workspace layout (cross-device sync). The snake_case shape is preserved
// to match the persisted server-side representation; the camelCase proto
// fields are translated in decodePane / decodeGroup.
export interface WorkspacePaneDTO {
  session_id: string;
  name: string;
  header_color: string;
  theme_id: string;
  font_size: number;
  sort_order: number;
  group_id: string | null;
  supports_messages_view: boolean;
  /** User-set "come back to this" flag; independent of the message read cursor. */
  manually_unread: boolean;
}

export interface TabGroupDTO {
  id: string;
  name: string;
  color: string;
  sort_order: number;
  is_collapsed: boolean;
}

export interface WorkspaceLayoutDTO {
  active_pane: string;
  panes: WorkspacePaneDTO[];
  groups: TabGroupDTO[];
  /** Named positions inside groups. Empty for a workspace that uses none. */
  roles: RoleDTO[];
}

function decodePane(p: {
  sessionId: string;
  name: string;
  headerColor: string;
  themeId: string;
  fontSize: number;
  sortOrder: number;
  groupId: string;
  supportsMessagesView: boolean;
  manuallyUnread: boolean;
}): WorkspacePaneDTO {
  return {
    session_id: p.sessionId,
    name: p.name,
    header_color: p.headerColor,
    theme_id: p.themeId,
    font_size: p.fontSize,
    sort_order: p.sortOrder,
    group_id: p.groupId === "" ? null : p.groupId,
    supports_messages_view: p.supportsMessagesView,
    manually_unread: p.manuallyUnread,
  };
}

function decodeGroup(g: {
  id: string;
  name: string;
  color: string;
  sortOrder: number;
  isCollapsed: boolean;
}): TabGroupDTO {
  return {
    id: g.id,
    name: g.name,
    color: g.color,
    sort_order: g.sortOrder,
    is_collapsed: g.isCollapsed,
  };
}

export async function getWorkspaceLayout(): Promise<WorkspaceLayoutDTO> {
  const resp = await workspaceClient.getLayout({});
  return {
    active_pane: resp.activePane,
    panes: resp.panes.map(decodePane),
    groups: resp.groups.map(decodeGroup),
    // Roles ride along in the same response: hydration stays one round trip.
    roles: resp.roles.map(decodeRole),
  };
}

export async function saveWorkspaceLayout(req: {
  active_pane: string | null;
  pane_order: string[];
}): Promise<void> {
  await workspaceClient.saveLayout({
    activePane: req.active_pane ?? "",
    paneOrder: req.pane_order,
  });
}

export async function updateWorkspacePane(
  sessionId: string,
  update: Partial<Omit<WorkspacePaneDTO, "session_id">>,
): Promise<WorkspacePaneDTO> {
  const req: Parameters<typeof workspaceClient.updatePane>[0] = { sessionId };
  if (update.name !== undefined) {
    req.name = update.name;
    req.hasName = true;
  }
  if (update.header_color !== undefined) {
    req.headerColor = update.header_color;
    req.hasHeaderColor = true;
  }
  if (update.theme_id !== undefined) {
    req.themeId = update.theme_id;
    req.hasThemeId = true;
  }
  if (update.font_size !== undefined) {
    req.fontSize = update.font_size;
    req.hasFontSize = true;
  }
  if (update.sort_order !== undefined) {
    req.sortOrder = update.sort_order;
    req.hasSortOrder = true;
  }
  if (update.group_id !== undefined) {
    req.groupId = update.group_id ?? "";
    req.hasGroupId = true;
  }
  if (update.supports_messages_view !== undefined) {
    req.supportsMessagesView = update.supports_messages_view;
    req.hasSupportsMessagesView = true;
  }
  if (update.manually_unread !== undefined) {
    req.manuallyUnread = update.manually_unread;
    req.hasManuallyUnread = true;
  }
  const resp = await workspaceClient.updatePane(req);
  if (!resp.pane) {
    throw new Error("workspace.updatePane: missing pane in response");
  }
  return decodePane(resp.pane);
}

export async function deleteWorkspacePane(sessionId: string): Promise<void> {
  await workspaceClient.deletePane({ sessionId });
}

export async function createTabGroup(req: {
  name: string;
  color: string;
}): Promise<TabGroupDTO> {
  const resp = await workspaceClient.createGroup(req);
  if (!resp.group) {
    throw new Error("workspace.createGroup: missing group in response");
  }
  return decodeGroup(resp.group);
}

export async function updateTabGroup(
  id: string,
  update: Partial<Omit<TabGroupDTO, "id">>,
): Promise<TabGroupDTO> {
  const req: Parameters<typeof workspaceClient.updateGroup>[0] = { id };
  if (update.name !== undefined) {
    req.name = update.name;
    req.hasName = true;
  }
  if (update.color !== undefined) {
    req.color = update.color;
    req.hasColor = true;
  }
  if (update.is_collapsed !== undefined) {
    req.isCollapsed = update.is_collapsed;
    req.hasIsCollapsed = true;
  }
  const resp = await workspaceClient.updateGroup(req);
  if (!resp.group) {
    throw new Error("workspace.updateGroup: missing group in response");
  }
  return decodeGroup(resp.group);
}

export async function deleteTabGroup(id: string): Promise<void> {
  await workspaceClient.deleteGroup({ id });
}
