import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useCallback, useRef, useEffect, useMemo } from "react";
import { createPortal } from "react-dom";
import { Search, SquareSlash } from "lucide-react";
import { useTranslation } from "react-i18next";
import { KEY_COMBOS, CATEGORY_ORDER, filterCombos } from "../consts/key-combos";
import { sendComboSequence } from "../lib/comboSequence";
import { strings } from "../consts/strings";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
export default function KeyComboPicker({ onInput, onFocusTerminal, triggerClassName }) {
    const { t } = useTranslation();
    const [open, setOpen] = useState(false);
    const [searchQuery, setSearchQuery] = useState("");
    const searchRef = useRef(null);
    const recentCombos = useWorkspaceStore((s) => s.recentCombos);
    const addRecentCombo = useWorkspaceStore((s) => s.addRecentCombo);
    // When the picker opens, dismiss the virtual keyboard first, then focus
    // the search input.  The trigger button uses onPointerDown preventDefault
    // (to avoid stealing focus from the terminal on regular key presses), which
    // means the previously-focused element (xterm's hidden textarea, or the
    // MobileToolbar textarea) is never blurred.  If we don't blur it explicitly,
    // the virtual keyboard stays open behind the picker sheet, which on mobile
    // covers the combo list and is confusing.
    //
    // The 50ms delay before focusing search gives the browser enough time to
    // actually retract the keyboard animation; focusing a new input immediately
    // after blur can cause the keyboard to snap back open on some mobile browsers.
    useEffect(() => {
        if (open) {
            // Blur whatever currently has focus (dismisses virtual keyboard)
            if (document.activeElement instanceof HTMLElement) {
                document.activeElement.blur();
            }
            const id = setTimeout(() => searchRef.current?.focus(), 50);
            return () => clearTimeout(id);
        }
        setSearchQuery("");
    }, [open]);
    const filtered = useMemo(() => filterCombos(KEY_COMBOS, searchQuery), [searchQuery]);
    const recentItems = useMemo(() => {
        if (searchQuery)
            return [];
        return recentCombos
            .map((id) => KEY_COMBOS.find((c) => c.id === id))
            .filter((c) => c !== undefined);
    }, [recentCombos, searchQuery]);
    const handleSelect = useCallback((combo) => {
        setOpen(false);
        void sendComboSequence(combo.sequence, onInput);
        addRecentCombo(combo.id);
        onFocusTerminal?.();
    }, [onInput, onFocusTerminal, addRecentCombo]);
    const comboButton = (combo, testIdPrefix) => (_jsxs("button", { "data-testid": `${testIdPrefix}-${combo.id}`, tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: () => handleSelect(combo), className: "flex w-full items-center gap-2 rounded px-2 py-1.5 text-start text-sm transition active:bg-wc-accent-active hover:bg-wc-surface-input", children: [_jsx("span", { className: "shrink-0 rounded border border-wc-default bg-wc-surface-input px-1.5 py-0.5 font-mono text-xs text-wc-text-primary", children: combo.keys }), _jsx("span", { className: "min-w-0 truncate text-wc-text-secondary", children: combo.label })] }, combo.id));
    return (_jsxs(_Fragment, { children: [_jsx("button", { "data-testid": "combo-picker-trigger", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: () => setOpen(true), className: triggerClassName ?? "shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation", title: t(strings.keyComboPicker.triggerTitle), children: _jsx(SquareSlash, { className: "h-3.5 w-3.5" }) }), open &&
                createPortal(_jsxs("div", { className: "fixed inset-0 z-wc-popover-backdrop", onMouseDown: (e) => e.preventDefault(), children: [_jsx("div", { "data-testid": "combo-picker-backdrop", className: "absolute inset-0 bg-wc-backdrop", onClick: () => setOpen(false) }), _jsxs("div", { "data-testid": "combo-picker-panel", className: "wc-stable-theme absolute bottom-0 left-0 right-0 z-wc-popover flex max-h-[60dvh] flex-col rounded-t-xl border-t border-wc-default bg-wc-surface-raised pb-[var(--wc-safe-bottom)] ps-[var(--wc-safe-left,0px)] pe-[var(--wc-safe-right,0px)] shadow-2xl", children: [_jsx("div", { className: "flex justify-center py-2", children: _jsx("div", { className: "h-1 w-8 rounded-full bg-wc-text-muted/40" }) }), _jsxs("div", { className: "flex items-center gap-2 border-b border-wc-default px-3 pb-2", children: [_jsx(Search, { className: "h-4 w-4 shrink-0 text-wc-text-muted" }), _jsx("input", { ref: searchRef, "data-testid": "combo-picker-search", type: "text", value: searchQuery, onChange: (e) => setSearchQuery(e.target.value), placeholder: t(strings.keyComboPicker.searchPlaceholder), className: "min-w-0 flex-1 bg-transparent text-sm text-wc-text-primary placeholder:text-wc-text-muted outline-none" })] }), _jsxs("div", { className: "flex-1 overflow-y-auto px-2 py-2", children: [recentItems.length > 0 && (_jsxs("div", { className: "mb-2", children: [_jsx("h3", { className: "px-2 pb-1 text-xs font-semibold uppercase tracking-wider text-wc-text-muted", children: t(strings.keyComboPicker.recent) }), recentItems.map((c) => comboButton(c, "combo-recent"))] })), CATEGORY_ORDER.map((cat) => {
                                            const items = filtered.filter((c) => c.category === cat);
                                            if (items.length === 0)
                                                return null;
                                            return (_jsxs("div", { className: "mb-2", children: [_jsx("h3", { className: "px-2 pb-1 text-xs font-semibold uppercase tracking-wider text-wc-text-muted", children: cat }), items.map((c) => comboButton(c, "combo-item"))] }, cat));
                                        }), filtered.length === 0 && (_jsx("p", { className: "px-2 py-4 text-center text-sm text-wc-text-muted", children: t(strings.keyComboPicker.noResults, { query: searchQuery }) }))] })] })] }), document.body)] }));
}
