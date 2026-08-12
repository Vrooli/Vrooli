import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
/**
 * @vrooliComponentSource react-component-library:DrawerShell
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 66af1418-3596-413a-b978-2a70b7bc1511
 * @vrooliComponentAppliedAt 2026-07-14T03:49:23Z
 * @vrooliComponentSourceSha256 6bcf14cb7bed31be6c9cb045a02789ebeeb1f068a90c6287649ecda736d8be54
 * @vrooliComponentDriftHash 6bcf14cb7bed31be6c9cb045a02789ebeeb1f068a90c6287649ecda736d8be54
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useEffect, useId, useRef } from "react";
import { useEscapeKey } from "../hooks/useEscapeKey";
import { useFocusTrap } from "../hooks/useFocusTrap";
/**
 * DrawerShell is the shared modal/drawer surface for web-console: full-page
 * previews, the composer, settings, the launcher, and small compact panels.
 * It owns the backdrop, panel sizing, safe-area handling, dialog semantics
 * (role=dialog, aria-modal, labelled title), Escape-to-close, focus trapping,
 * and the header chrome. It is a pure UI contract and intentionally knows
 * nothing about its consumers' domains.
 */
export function DrawerShell({ open = true, onClose = () => { }, closeAriaLabel = "Close drawer", title = "Drawer", headerActions, headerExtra, panelTestId, size = "full", avoidKeyboard = false, children, }) {
    const closeButtonRef = useRef(null);
    const panelRef = useRef(null);
    const titleId = useId();
    useEscapeKey(open, onClose);
    useFocusTrap(open, panelRef);
    useEffect(() => {
        if (!open)
            return;
        const previousFocus = document.activeElement;
        closeButtonRef.current?.focus();
        return () => previousFocus?.focus();
    }, [open]);
    if (!open)
        return null;
    const desktopSizeClasses = size === "compact"
        ? "md:inset-x-auto md:bottom-auto md:left-1/2 md:top-1/2 md:w-full md:max-w-md md:-translate-x-1/2 md:-translate-y-1/2 md:max-h-[80vh] md:rounded-2xl md:border"
        : "md:inset-x-8 md:bottom-8 md:top-8 md:rounded-2xl md:border";
    return (_jsxs("div", { className: "fixed inset-0 z-wc-drawer", children: [_jsx("button", { type: "button", className: "absolute inset-0 bg-wc-backdrop", "aria-label": "Dismiss drawer backdrop", onClick: onClose }), _jsxs("div", { ref: panelRef, role: "dialog", "aria-modal": "true", "aria-labelledby": titleId, "data-testid": panelTestId, className: "wc-stable-theme absolute inset-x-0 top-[max(1rem,var(--wc-safe-top,0px))] flex flex-col overflow-hidden rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised shadow-2xl " +
                    desktopSizeClasses +
                    (avoidKeyboard ? " bottom-[var(--wc-kb-height,0px)]" : " bottom-0"), children: [_jsxs("div", { className: "shrink-0 border-b border-wc-default px-4 py-3", children: [_jsxs("div", { className: "flex items-center gap-3", children: [_jsx("h2", { id: titleId, className: "min-w-0 flex-1 truncate text-sm font-semibold text-wc-text-primary", children: title }), headerActions, _jsx("button", { ref: closeButtonRef, type: "button", onClick: onClose, className: "shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary", "aria-label": closeAriaLabel, children: _jsx("span", { "aria-hidden": true, children: "\u00D7" }) })] }), headerExtra] }), _jsx("div", { className: "min-h-0 flex-1 overflow-hidden", children: children })] })] }));
}
export default DrawerShell;
