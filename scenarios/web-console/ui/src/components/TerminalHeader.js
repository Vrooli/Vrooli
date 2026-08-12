import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useRef, useCallback } from "react";
import { GripVertical, Loader2, MessageSquareText, TerminalSquare, Palette, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { paneAccentStyle } from "../lib/paneColor";
import { Button } from "./ui/button";
export default function TerminalHeader({ sessionId, name, headerColor, isActive, viewMode = "terminal", unreadCount = 0, onClose, onFocus, onToggleView, isViewSwitchPending = false, onDragStart, }) {
    const { t } = useTranslation();
    const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
    const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
    const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
    const panes = useWorkspaceStore((s) => s.panes);
    // Group color, for panes grouped before joining a group started seeding the
    // pane's own color. Selecting the resolved color (a primitive) rather than
    // the group object keeps this subscription from re-rendering the header on
    // unrelated group edits.
    const groupColor = useWorkspaceStore((s) => {
        const groupId = s.panes.find((p) => p.sessionId === sessionId)?.groupId;
        return groupId ? s.groups.find((g) => g.id === groupId)?.color ?? null : null;
    });
    const [editing, setEditing] = useState(false);
    const [editValue, setEditValue] = useState(name);
    const inputRef = useRef(null);
    const commitRename = useCallback(() => {
        const trimmed = editValue.trim();
        if (trimmed && trimmed !== name) {
            renamePaneById(sessionId, trimmed);
        }
        else {
            setEditValue(name);
        }
        setEditing(false);
    }, [editValue, name, renamePaneById, sessionId]);
    const startEditing = useCallback(() => {
        setEditValue(name);
        setEditing(true);
        requestAnimationFrame(() => inputRef.current?.select());
    }, [name]);
    const bgStyle = paneAccentStyle(headerColor, groupColor, "header");
    return (_jsxs("div", { "data-testid": `terminal-header-${sessionId}`, className: cn("flex h-7 items-center gap-1 px-1.5 text-xs select-none", isActive ? "border-b-2 border-wc-accent" : "border-b border-wc-default"), style: bgStyle ?? { backgroundColor: "rgb(var(--wc-surface-header))" }, onClick: onFocus, children: [_jsx("button", { type: "button", "data-testid": `terminal-drag-handle-${sessionId}`, className: "flex h-5 w-5 items-center justify-center shrink-0 text-wc-text-faint hover:text-wc-text-secondary cursor-grab active:cursor-grabbing touch-none", onPointerDown: (e) => onDragStart?.(sessionId, e), onKeyDown: (e) => {
                    const idx = panes.findIndex((p) => p.sessionId === sessionId);
                    if (idx === -1)
                        return;
                    if (e.key === "ArrowUp" && idx > 0) {
                        e.preventDefault();
                        movePaneToIndex(sessionId, idx - 1);
                    }
                    else if (e.key === "ArrowDown" && idx < panes.length - 1) {
                        e.preventDefault();
                        movePaneToIndex(sessionId, idx + 1);
                    }
                }, "aria-label": t(strings.terminalHeader.reorderAria, { name }), "aria-roledescription": "drag handle", children: _jsx(GripVertical, { className: "h-3 w-3" }) }), editing ? (_jsx("input", { ref: inputRef, "data-testid": `terminal-header-name-input-${sessionId}`, className: "min-w-0 flex-1 bg-transparent text-xs text-wc-text-primary outline-none", value: editValue, onChange: (e) => setEditValue(e.target.value), onBlur: commitRename, onKeyDown: (e) => {
                    if (e.key === "Enter")
                        commitRename();
                    if (e.key === "Escape") {
                        setEditValue(name);
                        setEditing(false);
                    }
                } })) : (_jsx("span", { "data-testid": `terminal-header-name-${sessionId}`, className: "min-w-0 flex-1 truncate text-wc-text-secondary cursor-pointer", onClick: (e) => {
                    e.stopPropagation();
                    startEditing();
                }, title: t(strings.terminalHeader.renameTitle), children: name })), unreadCount > 0 && (_jsx("span", { className: "rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-wc-accent-fg", children: unreadCount })), onToggleView && (_jsx("button", { "data-testid": `terminal-header-toggle-view-${sessionId}`, type: "button", className: "flex h-8 w-8 items-center justify-center rounded-full border border-wc-default bg-wc-surface-raised/80 shrink-0 text-wc-text-secondary transition-colors hover:bg-wc-surface-input hover:text-wc-text-primary backdrop-blur-sm", onClick: (e) => {
                    e.stopPropagation();
                    onToggleView();
                }, title: viewMode === "terminal" ? t(strings.terminalHeader.showMessages) : t(strings.terminalHeader.showTerminal), children: isViewSwitchPending
                    ? _jsx(Loader2, { "data-testid": `terminal-header-toggle-view-pending-${sessionId}`, className: "h-3.5 w-3.5 animate-spin" })
                    : viewMode === "terminal" ? _jsx(MessageSquareText, { className: "h-3.5 w-3.5" }) : _jsx(TerminalSquare, { className: "h-3.5 w-3.5" }) })), _jsx("button", { "data-testid": `terminal-header-appearance-${sessionId}`, type: "button", className: "flex h-5 w-5 items-center justify-center rounded shrink-0 text-wc-text-faint hover:text-wc-text-secondary", onClick: (e) => {
                    e.stopPropagation();
                    setAppearanceModalPane(sessionId);
                }, title: t(strings.terminalHeader.appearanceTitle), children: _jsx(Palette, { className: "h-3 w-3" }) }), _jsx(Button, { "data-testid": `terminal-close-${sessionId}`, variant: "ghost", size: "icon", className: "h-5 w-5 shrink-0 text-wc-text-faint", onClick: (e) => {
                    e.stopPropagation();
                    onClose();
                }, title: t(strings.terminalHeader.closeTitle), children: _jsx(X, { className: "h-3 w-3" }) })] }));
}
