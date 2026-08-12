import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useId, useRef } from "react";
import { useEscapeKey } from "../hooks/useEscapeKey";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { cn } from "../lib/classnames";
/**
 * ConfirmDialog is the single confirm primitive for destructive yes/no
 * decisions (close session, discard attachments). It renders a centered card
 * on the confirm z tier — above the drawer tier — so it works standalone and
 * layered over an open DrawerShell. Owns role=alertdialog semantics, Escape
 * (= cancel), focus trapping, and auto-focusing the safe Cancel button.
 */
export function ConfirmDialog({ open, title, body, cancelLabel, confirmLabel, destructive = false, onCancel, onConfirm, testIdPrefix, }) {
    useEscapeKey(open, onCancel);
    const panelRef = useRef(null);
    useFocusTrap(open, panelRef);
    // Auto-focus Cancel for safety: Enter never confirms destruction by default.
    const cancelRef = useRef(null);
    useEffect(() => {
        if (open)
            cancelRef.current?.focus();
    }, [open]);
    const titleId = useId();
    const bodyId = useId();
    if (!open)
        return null;
    return (_jsx("div", { "data-testid": `${testIdPrefix}-dialog`, className: "fixed inset-0 z-wc-confirm flex items-center justify-center bg-wc-backdrop p-4", onClick: onCancel, children: _jsxs("div", { ref: panelRef, role: "alertdialog", "aria-modal": "true", "aria-labelledby": titleId, "aria-describedby": bodyId, className: "wc-stable-theme w-full max-w-sm rounded-lg border border-wc-default bg-wc-surface-raised p-5 shadow-xl", onClick: (e) => e.stopPropagation(), children: [_jsx("h2", { id: titleId, className: "mb-2 text-sm font-semibold text-wc-text-primary", children: title }), _jsx("p", { id: bodyId, className: "mb-4 text-xs text-wc-text-secondary", children: body }), _jsxs("div", { className: "flex justify-end gap-2", children: [_jsx("button", { ref: cancelRef, type: "button", "data-testid": `${testIdPrefix}-cancel`, className: "rounded-full px-4 py-1.5 text-sm font-medium text-wc-text-primary transition hover:bg-white/10", onClick: onCancel, children: cancelLabel }), _jsx("button", { type: "button", "data-testid": `${testIdPrefix}-confirm`, className: cn("rounded-full px-4 py-1.5 text-sm font-medium transition", destructive
                                ? "bg-red-600 text-white hover:bg-red-700"
                                : "bg-wc-accent text-wc-accent-fg hover:opacity-90"), onClick: onConfirm, children: confirmLabel })] })] }) }));
}
