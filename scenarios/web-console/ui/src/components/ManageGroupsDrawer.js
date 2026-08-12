import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useEffect, useRef, useState } from "react";
import { FolderMinus, FolderPlus, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { useGroupActions } from "../hooks/useGroupActions";
import { HEADER_COLORS } from "../consts/config";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import { ConfirmDialog } from "./ConfirmDialog";
import { DrawerShell } from "./DrawerShell";
const rowIconButtonClass = "shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary";
/**
 * ManageGroupsDrawer is the SSOT surface for tab-group management: per-group
 * session counts, rename, recolor, delete (with ungroup consequence), and
 * server-first creation. When opened with a session context (from a tab's
 * menu) each row also offers an assign/remove toggle for that session.
 * Open state lives in the workspace store (`manageGroupsTarget`) so TabBar,
 * SessionSidebar, and Workspace can all open it.
 */
export default function ManageGroupsDrawer() {
    const { t } = useTranslation();
    const target = useWorkspaceStore((s) => s.manageGroupsTarget);
    const setManageGroupsTarget = useWorkspaceStore((s) => s.setManageGroupsTarget);
    const groups = useWorkspaceStore((s) => s.groups);
    const panes = useWorkspaceStore((s) => s.panes);
    const updateGroup = useWorkspaceStore((s) => s.updateGroup);
    const { syncUpdateGroup } = useWorkspaceSync();
    const { assignPaneToGroup, removePaneFromGroup, deleteGroup, createGroup } = useGroupActions();
    const sessionId = target?.sessionId ?? null;
    const contextPane = sessionId ? panes.find((p) => p.sessionId === sessionId) : undefined;
    const [editingId, setEditingId] = useState(null);
    const [editName, setEditName] = useState("");
    const [paletteId, setPaletteId] = useState(null);
    const [deleteTarget, setDeleteTarget] = useState(null);
    const renameInputRef = useRef(null);
    const close = useCallback(() => setManageGroupsTarget(null), [setManageGroupsTarget]);
    // Reset transient editing state whenever the drawer opens fresh.
    useEffect(() => {
        if (target) {
            setEditingId(null);
            setEditName("");
            setPaletteId(null);
            setDeleteTarget(null);
        }
    }, [target]);
    useEffect(() => {
        if (editingId) {
            renameInputRef.current?.focus();
            renameInputRef.current?.select();
        }
    }, [editingId]);
    const startRename = useCallback((group) => {
        setEditingId(group.id);
        setEditName(group.name);
    }, []);
    const commitRename = useCallback(() => {
        if (editingId && editName.trim()) {
            const trimmed = editName.trim();
            updateGroup(editingId, { name: trimmed });
            syncUpdateGroup(editingId, { name: trimmed });
        }
        setEditingId(null);
        setEditName("");
    }, [editingId, editName, updateGroup, syncUpdateGroup]);
    const handleRecolor = useCallback((groupId, color) => {
        updateGroup(groupId, { color });
        syncUpdateGroup(groupId, { color });
        setPaletteId(null);
    }, [updateGroup, syncUpdateGroup]);
    const handleCreate = useCallback(async () => {
        try {
            const group = await createGroup();
            setEditingId(group.id);
            setEditName(group.name);
        }
        catch (err) {
            console.error("Failed to create group:", err);
        }
    }, [createGroup]);
    const deleteCount = deleteTarget
        ? panes.filter((p) => p.groupId === deleteTarget.id).length
        : 0;
    // While the delete confirm is layered on top, Escape/backdrop must resolve
    // the topmost surface (the confirm) and leave the drawer open.
    const requestClose = () => {
        if (deleteTarget) {
            setDeleteTarget(null);
            return;
        }
        close();
    };
    if (!target)
        return null;
    return (_jsxs(DrawerShell, { open: true, onClose: requestClose, size: "compact", closeAriaLabel: t(strings.manageGroups.closeAriaLabel), title: t(strings.manageGroups.title), panelTestId: "manage-groups-drawer", children: [_jsxs("div", { className: "flex h-full flex-col", children: [_jsxs("div", { className: "min-h-0 flex-1 space-y-2 overflow-y-auto p-4", children: [groups.length === 0 && (_jsx("p", { "data-testid": "manage-groups-empty", className: "py-6 text-center text-sm text-wc-text-muted", children: t(strings.manageGroups.empty) })), groups.map((group) => {
                                const count = panes.filter((p) => p.groupId === group.id).length;
                                const isEditing = editingId === group.id;
                                const isMember = contextPane?.groupId === group.id;
                                return (_jsxs("div", { "data-testid": `manage-groups-row-${group.id}`, className: "rounded-lg border border-wc-default bg-wc-surface-base/50", children: [_jsxs("div", { className: "flex items-center gap-2 px-3 py-2", children: [_jsx("button", { type: "button", "data-testid": `manage-groups-recolor-${group.id}`, "aria-label": t(strings.manageGroups.recolorAriaLabel, { name: group.name }), className: "h-4 w-4 shrink-0 rounded-full border border-wc-default transition hover:scale-110", style: { backgroundColor: group.color }, onClick: () => setPaletteId((prev) => (prev === group.id ? null : group.id)) }), isEditing ? (_jsx("input", { ref: renameInputRef, "data-testid": "manage-groups-rename-input", className: "min-w-0 flex-1 rounded bg-wc-surface-input px-2 py-1 text-sm font-medium text-wc-text-primary outline-none ring-1 ring-wc-accent", value: editName, onChange: (e) => setEditName(e.target.value), onKeyDown: (e) => {
                                                        if (e.key === "Enter")
                                                            commitRename();
                                                        if (e.key === "Escape") {
                                                            e.stopPropagation();
                                                            setEditingId(null);
                                                            setEditName("");
                                                        }
                                                    }, onBlur: commitRename })) : (_jsx("span", { className: "min-w-0 flex-1 truncate text-sm font-medium text-wc-text-primary", children: group.name })), _jsx("span", { "data-testid": `manage-groups-count-${group.id}`, className: "shrink-0 rounded bg-wc-surface-input px-1.5 py-0.5 text-[11px] text-wc-text-secondary", children: t(strings.manageGroups.sessionCount, { count }) }), contextPane && (_jsx("button", { type: "button", "data-testid": isMember ? `manage-groups-unassign-${group.id}` : `manage-groups-assign-${group.id}`, "aria-label": isMember ? t(strings.manageGroups.removeSession) : t(strings.manageGroups.assignSession), title: isMember ? t(strings.manageGroups.removeSession) : t(strings.manageGroups.assignSession), className: cn(rowIconButtonClass, isMember && "text-wc-accent"), onClick: () => isMember
                                                        ? removePaneFromGroup(contextPane.sessionId)
                                                        : assignPaneToGroup(contextPane.sessionId, group.id), children: isMember ? _jsx(FolderMinus, { className: "h-4 w-4" }) : _jsx(FolderPlus, { className: "h-4 w-4" }) })), _jsx("button", { type: "button", "data-testid": `manage-groups-rename-${group.id}`, "aria-label": t(strings.manageGroups.renameAriaLabel, { name: group.name }), className: rowIconButtonClass, onClick: () => startRename(group), children: _jsx(Pencil, { className: "h-4 w-4" }) }), _jsx("button", { type: "button", "data-testid": `manage-groups-delete-${group.id}`, "aria-label": t(strings.manageGroups.deleteAriaLabel, { name: group.name }), className: rowIconButtonClass, onClick: () => setDeleteTarget(group), children: _jsx(Trash2, { className: "h-4 w-4" }) })] }), paletteId === group.id && (_jsx("div", { "data-testid": "manage-groups-palette", className: "flex flex-wrap gap-1.5 border-t border-wc-default px-3 py-2", children: HEADER_COLORS.map((color) => (_jsx("button", { type: "button", "data-testid": `manage-groups-color-${color}`, className: cn("h-5 w-5 rounded-full border border-wc-default transition hover:scale-110", color === group.color && "ring-2 ring-wc-accent"), style: { backgroundColor: color }, title: color, onClick: () => handleRecolor(group.id, color) }, color))) }))] }, group.id));
                            })] }), _jsx("div", { className: "shrink-0 border-t border-wc-default p-3", children: _jsxs(Button, { "data-testid": "manage-groups-create", variant: "outline", className: "w-full", onClick: () => { void handleCreate(); }, children: [_jsx(Plus, { className: "h-4 w-4 me-2" }), t(strings.manageGroups.create)] }) })] }), _jsx(ConfirmDialog, { open: deleteTarget !== null, title: t(strings.manageGroups.deleteTitle, { name: deleteTarget?.name ?? "" }), body: deleteCount === 0
                    ? t(strings.manageGroups.deleteConsequenceNone)
                    : t(strings.manageGroups.deleteConsequence, { count: deleteCount }), cancelLabel: t(strings.manageGroups.deleteCancel), confirmLabel: t(strings.manageGroups.deleteConfirm), destructive: true, onCancel: () => setDeleteTarget(null), onConfirm: () => {
                    if (deleteTarget)
                        deleteGroup(deleteTarget.id);
                    setDeleteTarget(null);
                }, testIdPrefix: "manage-groups-delete-confirm" })] }));
}
