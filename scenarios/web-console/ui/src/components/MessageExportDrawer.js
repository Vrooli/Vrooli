import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Check, ClipboardCopy } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { DrawerShell } from "./DrawerShell";
import { buildMessageExport, DEFAULT_MESSAGE_EXPORT_FORMAT, MESSAGE_EXPORT_FORMATS, } from "../lib/messageExport";
/**
 * Render at most this many characters of the export in the live preview.
 * The copied text is always the full render; the cap only bounds what the
 * DOM has to lay out for very large selections.
 */
const PREVIEW_CHAR_LIMIT = 20000;
/**
 * MessageExportDrawer is the final step of the export flow: pick one of the
 * four approved formats, review the live preview and token estimate, and copy
 * the rendered text for a coding agent. It is deliberately not a second
 * selector — selection lives in the navigator; this drawer only formats and
 * copies what it is given.
 */
export default function MessageExportDrawer({ open, events, onClose }) {
    const { t } = useTranslation();
    const [format, setFormat] = useState(DEFAULT_MESSAGE_EXPORT_FORMAT);
    const [copyState, setCopyState] = useState("idle");
    const copiedTimerRef = useRef(null);
    // Transient feedback never outlives a reopen or a format switch.
    useEffect(() => {
        setCopyState("idle");
        return () => {
            if (copiedTimerRef.current)
                clearTimeout(copiedTimerRef.current);
        };
    }, [open, format]);
    const result = useMemo(() => buildMessageExport(events, format), [events, format]);
    const previewText = result.text.length > PREVIEW_CHAR_LIMIT ? result.text.slice(0, PREVIEW_CHAR_LIMIT) : result.text;
    const previewTruncated = result.text.length > PREVIEW_CHAR_LIMIT;
    const handleCopy = useCallback(async () => {
        try {
            await navigator.clipboard.writeText(result.text);
            setCopyState("copied");
            if (copiedTimerRef.current)
                clearTimeout(copiedTimerRef.current);
            copiedTimerRef.current = setTimeout(() => {
                setCopyState((prev) => (prev === "copied" ? "idle" : prev));
            }, 2000);
        }
        catch {
            setCopyState("failed");
        }
    }, [result.text]);
    return (_jsx(DrawerShell, { open: open, onClose: onClose, closeAriaLabel: t(strings.messageExport.closeAriaLabel), title: t(strings.messageExport.drawerTitle), panelTestId: "msg-export-drawer", headerExtra: _jsxs("div", { "data-testid": "msg-export-drawer-summary", className: "mt-1 text-[11px] text-wc-text-faint", children: [t(strings.messageExport.messageSummary, { count: result.messageCount }), " · ", t(strings.messageExport.approxTokens, { count: result.tokenEstimate })] }), children: _jsxs("div", { className: "flex h-full min-h-0 flex-col md:flex-row", children: [_jsxs("div", { className: "shrink-0 border-b border-wc-default/60 p-3 md:w-64 md:border-b-0 md:border-e md:overflow-y-auto", children: [_jsx("div", { className: "mb-2 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint", children: t(strings.messageExport.formatHeading) }), _jsx("div", { role: "radiogroup", "aria-label": t(strings.messageExport.formatHeading), className: "flex flex-row flex-wrap gap-1.5 md:flex-col", children: MESSAGE_EXPORT_FORMATS.map((descriptor) => {
                                const active = descriptor.id === format;
                                return (_jsxs("button", { type: "button", role: "radio", "aria-checked": active, "data-testid": `msg-export-format-${descriptor.id}`, "data-active": active, onClick: () => setFormat(descriptor.id), className: cn("min-h-[44px] rounded-lg px-3 py-2 text-start transition", active
                                        ? "bg-wc-accent/20 text-wc-text-primary"
                                        : "bg-wc-surface-input/40 text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary"), children: [_jsx("span", { className: "block text-xs font-medium", children: t(descriptor.labelKey) }), _jsx("span", { className: "mt-0.5 hidden text-[10px] text-wc-text-faint md:block", children: t(descriptor.descriptionKey) })] }, descriptor.id));
                            }) })] }), _jsxs("div", { className: "flex min-h-0 flex-1 flex-col p-3", children: [_jsx("div", { className: "mb-2 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint", children: t(strings.messageExport.previewHeading) }), _jsx("pre", { "data-testid": "msg-export-preview", "aria-label": t(strings.messageExport.previewAriaLabel), tabIndex: 0, className: "min-h-0 flex-1 select-text overflow-auto whitespace-pre-wrap break-words rounded-lg border border-wc-default/60 bg-wc-surface-base p-3 font-mono text-[11px] leading-relaxed text-wc-text-secondary", children: previewText }), previewTruncated && (_jsx("div", { "data-testid": "msg-export-preview-truncated", className: "mt-1 text-[10px] text-wc-text-faint", children: t(strings.messageExport.previewTruncated) })), copyState === "failed" && (_jsx("div", { "data-testid": "msg-export-copy-error", role: "alert", className: "mt-2 rounded-lg bg-red-500/10 px-3 py-2 text-[11px] text-red-400", children: t(strings.messageExport.copyFailed) })), _jsx("div", { className: "mt-3 flex shrink-0 items-center justify-end gap-2 pb-[max(0px,var(--wc-safe-bottom,0px))]", children: _jsx("button", { type: "button", "data-testid": "msg-export-copy", onClick: () => void handleCopy(), disabled: result.text.length === 0, className: cn("inline-flex min-h-[44px] items-center gap-1.5 rounded-lg px-4 py-2 text-xs font-semibold transition", copyState === "copied"
                                    ? "bg-emerald-500/20 text-emerald-300"
                                    : "bg-wc-accent/25 text-wc-text-primary hover:bg-wc-accent/35", "disabled:cursor-not-allowed disabled:opacity-40"), children: copyState === "copied" ? (_jsxs(_Fragment, { children: [_jsx(Check, { className: "h-3.5 w-3.5", "aria-hidden": "true" }), t(strings.messageExport.copiedFeedback)] })) : (_jsxs(_Fragment, { children: [_jsx(ClipboardCopy, { className: "h-3.5 w-3.5", "aria-hidden": "true" }), t(strings.messageExport.copyAction)] })) }) })] })] }) }));
}
