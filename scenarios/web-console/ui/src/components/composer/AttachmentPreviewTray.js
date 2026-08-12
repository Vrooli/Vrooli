import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Loader2, X } from "lucide-react";
/**
 * Horizontal tray of reviewable image thumbnails shown inside the composer.
 * Attachments are NEVER submitted until send — this tray only stages them.
 * Modeled on swarm-manager's AttachmentPreviewTray, themed with wc-* tokens.
 */
export function AttachmentPreviewTray({ attachments, onRemove, removeAriaLabel }) {
    if (attachments.length === 0)
        return null;
    return (_jsx("div", { "data-testid": "composer-attachment-tray", className: "mb-2 flex gap-2 overflow-x-auto py-1", children: attachments.map((att) => (_jsxs("div", { className: "group relative shrink-0", children: [_jsx("div", { className: "h-20 w-20 overflow-hidden rounded-lg border border-wc-default bg-wc-surface-input", children: _jsx("img", { src: att.previewUrl, alt: att.file.name, className: "h-full w-full object-cover" }) }), att.status === "uploading" && (_jsx("div", { className: "absolute inset-0 flex items-center justify-center rounded-lg bg-wc-backdrop", children: _jsx(Loader2, { className: "h-4 w-4 animate-spin text-wc-text-primary" }) })), _jsx("button", { type: "button", "data-testid": `composer-attachment-remove-${att.id}`, onClick: () => onRemove(att.id), className: "absolute -end-2 -top-2 rounded-full border border-wc-default bg-wc-surface-raised p-1 text-wc-text-secondary transition hover:border-red-500/50 hover:text-red-400", "aria-label": removeAriaLabel, children: _jsx(X, { className: "h-3 w-3" }) })] }, att.id))) }));
}
