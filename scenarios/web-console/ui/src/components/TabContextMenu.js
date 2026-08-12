import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ArrowDown, ArrowUp, ClipboardCopy, Dot, FolderCog, FolderMinus, FolderPlus, MailOpen, Palette, Pencil, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";
export default function TabContextMenu({ position, sessionId, currentGroupId, isManuallyUnread, onToggleManuallyUnread, onRename, onCustomize, onRemoveFromGroup, onManageGroups, onMoveUp, onMoveDown, onClose, onDismiss, }) {
    const { t } = useTranslation();
    const handleAction = (action) => {
        action();
        onDismiss();
    };
    return (_jsxs(ContextMenuBase, { position: position, onClose: onDismiss, children: [_jsxs("button", { "data-testid": "tab-ctx-rename", className: contextMenuItemClass, onClick: () => handleAction(onRename), children: [_jsx(Pencil, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.rename)] }), _jsxs("button", { "data-testid": "tab-ctx-toggle-unread", className: contextMenuItemClass, onClick: () => handleAction(onToggleManuallyUnread), children: [isManuallyUnread
                        ? _jsx(MailOpen, { className: "h-4 w-4 shrink-0" })
                        : _jsx(Dot, { className: "h-4 w-4 shrink-0" }), t(isManuallyUnread
                        ? strings.tabContextMenu.markAsRead
                        : strings.tabContextMenu.markAsUnread)] }), _jsxs("button", { "data-testid": "tab-ctx-customize", className: contextMenuItemClass, onClick: () => handleAction(onCustomize), children: [_jsx(Palette, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.customizeAppearance)] }), _jsx("div", { className: "border-t border-wc-default my-1" }), currentGroupId ? (_jsxs(_Fragment, { children: [_jsxs("button", { "data-testid": "tab-ctx-remove-from-group", className: contextMenuItemClass, onClick: () => handleAction(onRemoveFromGroup), children: [_jsx(FolderMinus, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.removeFromGroup)] }), _jsxs("button", { "data-testid": "tab-ctx-manage-groups", className: contextMenuItemClass, onClick: () => handleAction(onManageGroups), children: [_jsx(FolderCog, { className: "h-4 w-4 shrink-0" }), t(strings.manageGroups.menuItem)] })] })) : (_jsxs("button", { "data-testid": "tab-ctx-add-to-group", className: contextMenuItemClass, onClick: () => handleAction(onManageGroups), children: [_jsx(FolderPlus, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.addToGroup)] })), (onMoveUp || onMoveDown) && (_jsxs(_Fragment, { children: [_jsx("div", { className: "border-t border-wc-default my-1" }), onMoveUp && (_jsxs("button", { "data-testid": "tab-ctx-move-up", className: contextMenuItemClass, onClick: () => handleAction(onMoveUp), children: [_jsx(ArrowUp, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.moveUp)] })), onMoveDown && (_jsxs("button", { "data-testid": "tab-ctx-move-down", className: contextMenuItemClass, onClick: () => handleAction(onMoveDown), children: [_jsx(ArrowDown, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.moveDown)] }))] })), _jsx("div", { className: "border-t border-wc-default my-1" }), _jsxs("button", { "data-testid": "tab-ctx-close", className: contextMenuItemClass, onClick: () => handleAction(() => onClose(sessionId)), children: [_jsx(X, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.closeTab)] }), _jsx("div", { className: "border-t border-wc-default my-1" }), _jsxs("button", { "data-testid": "tab-ctx-copy-debug-log", className: contextMenuItemClass, onClick: () => handleAction(() => {
                    const probe = window.__wc_terminal_output;
                    const data = probe?.[sessionId] ?? "";
                    const payload = data || "(empty probe)";
                    if (navigator.clipboard?.writeText) {
                        navigator.clipboard
                            .writeText(payload)
                            .then(() => alert(`Copied ${data.length} chars`))
                            .catch(() => alert("Clipboard denied"));
                    }
                    else {
                        alert("Clipboard unavailable");
                    }
                }), children: [_jsx(ClipboardCopy, { className: "h-4 w-4 shrink-0" }), t(strings.tabContextMenu.copyDebugLog)] })] }));
}
