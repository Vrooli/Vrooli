import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ChevronDown, ChevronRight, FolderCog, Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";
/**
 * Quick ephemeral actions on a group header. Management operations (rename,
 * recolor, ungroup, delete) live in the Manage Groups drawer — this menu only
 * keeps the in-place toggles plus a deep link into that drawer.
 */
export default function GroupContextMenu({ position, group, onNewSession, onToggleCollapse, onManageGroups, onDismiss, }) {
    const { t } = useTranslation();
    const handleAction = (action) => {
        action();
        onDismiss();
    };
    return (_jsxs(ContextMenuBase, { position: position, onClose: onDismiss, "data-testid": "group-ctx-menu", children: [onNewSession && (_jsxs("button", { "data-testid": "group-ctx-new-session", className: contextMenuItemClass, onClick: () => handleAction(onNewSession), children: [_jsx(Plus, { className: "h-4 w-4 shrink-0" }), t(strings.groupContextMenu.newSession)] })), _jsxs("button", { "data-testid": "group-ctx-toggle-collapse", className: contextMenuItemClass, onClick: () => handleAction(onToggleCollapse), children: [group.isCollapsed ? (_jsx(ChevronDown, { className: "h-4 w-4 shrink-0" })) : (_jsx(ChevronRight, { className: "h-4 w-4 shrink-0" })), group.isCollapsed ? t(strings.groupContextMenu.expand) : t(strings.groupContextMenu.collapse)] }), _jsx("div", { className: "border-t border-wc-default my-1" }), _jsxs("button", { "data-testid": "group-ctx-manage-groups", className: contextMenuItemClass, onClick: () => handleAction(onManageGroups), children: [_jsx(FolderCog, { className: "h-4 w-4 shrink-0" }), t(strings.manageGroups.menuItem)] })] }));
}
