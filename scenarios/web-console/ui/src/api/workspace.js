// Workspace domain home: Connect-RPC client, types, decoders, and
// wrappers for cross-device workspace layout sync (panes + tab groups).
import { createClient } from "@connectrpc/connect";
import { WorkspaceService } from "@vrooli/proto-types/web-console/v1/workspace/workspace_pb";
import { transport } from "./client";
export const workspaceClient = createClient(WorkspaceService, transport);
function decodePane(p) {
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
function decodeGroup(g) {
    return {
        id: g.id,
        name: g.name,
        color: g.color,
        sort_order: g.sortOrder,
        is_collapsed: g.isCollapsed,
    };
}
export async function getWorkspaceLayout() {
    const resp = await workspaceClient.getLayout({});
    return {
        active_pane: resp.activePane,
        panes: resp.panes.map(decodePane),
        groups: resp.groups.map(decodeGroup),
    };
}
export async function saveWorkspaceLayout(req) {
    await workspaceClient.saveLayout({
        activePane: req.active_pane ?? "",
        paneOrder: req.pane_order,
    });
}
export async function updateWorkspacePane(sessionId, update) {
    const req = { sessionId };
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
export async function deleteWorkspacePane(sessionId) {
    await workspaceClient.deletePane({ sessionId });
}
export async function createTabGroup(req) {
    const resp = await workspaceClient.createGroup(req);
    if (!resp.group) {
        throw new Error("workspace.createGroup: missing group in response");
    }
    return decodeGroup(resp.group);
}
export async function updateTabGroup(id, update) {
    const req = { id };
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
export async function deleteTabGroup(id) {
    await workspaceClient.deleteGroup({ id });
}
