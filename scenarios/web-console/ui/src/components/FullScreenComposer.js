import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useEffect, useRef, useState } from "react";
import { ImagePlus, Loader2, SendHorizontal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ConfirmDialog } from "./ConfirmDialog";
import { DrawerShell } from "./DrawerShell";
import { AttachmentPreviewTray } from "./composer/AttachmentPreviewTray";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { composeComposerPayload } from "../lib/composerPayload";
/**
 * FullScreenComposer — a portaled DrawerShell overlay for authoring long,
 * mixed text+image messages. It is an OVERLAY, not a pane replacement: the
 * xterm terminal stays mounted underneath and never reflows.
 *
 * The textarea is uncontrolled and bound to the shared `draft`, so the
 * collapsed toolbar input and this composer read/write one value that cannot
 * diverge. Terminal keys/modifiers are intentionally absent — this is a message
 * composer, not a terminal-key surface.
 */
export default function FullScreenComposer({ open, onClose, draft, onInput, subscribeInputSettled, onFocusTerminal, mic, attachments = [], onAttachFiles, onRemoveAttachment, resolveAttachmentPaths, onClearAttachments, }) {
    const { t } = useTranslation();
    const textareaRef = useRef(null);
    const fileInputRef = useRef(null);
    const [status, setStatus] = useState("idle");
    const [errorMsg, setErrorMsg] = useState(null);
    const [showDiscardPrompt, setShowDiscardPrompt] = useState(false);
    const settlementUnsubRef = useRef(null);
    // Bind the uncontrolled textarea to the shared draft: reseed on peer changes
    // (voice, session reload, collapsed-input edits) without clobbering our caret.
    useEffect(() => {
        if (!open)
            return;
        return draft.subscribe((change) => {
            const el = textareaRef.current;
            const isOwnTyping = change.reason === "input" && el != null && document.activeElement === el;
            if (el && !isOwnTyping && el.value !== change.value) {
                el.value = change.value;
                if (change.caret != null) {
                    try {
                        el.setSelectionRange(change.caret, change.caret);
                    }
                    catch {
                        /* detached during teardown */
                    }
                }
            }
        });
    }, [draft, open]);
    // Auto-focus the textarea on open and restore focus to the opener on close.
    const openerRef = useRef(null);
    useEffect(() => {
        if (!open)
            return;
        openerRef.current = document.activeElement ?? null;
        setStatus("idle");
        setErrorMsg(null);
        setShowDiscardPrompt(false);
        const raf = requestAnimationFrame(() => {
            const el = textareaRef.current;
            if (el) {
                el.value = draft.getValue();
                el.focus();
                const end = el.value.length;
                try {
                    el.setSelectionRange(end, end);
                }
                catch {
                    /* ignore */
                }
            }
        });
        return () => {
            cancelAnimationFrame(raf);
            const opener = openerRef.current;
            if (opener && typeof opener.focus === "function")
                opener.focus();
        };
    }, [open, draft]);
    // Cancel any dangling settlement subscription on unmount.
    useEffect(() => {
        return () => {
            settlementUnsubRef.current?.();
            settlementUnsubRef.current = null;
        };
    }, []);
    const handleChange = useCallback((e) => {
        draft.handleChange(e.currentTarget);
    }, [draft]);
    const awaitNextSettlement = useCallback((onSettle) => {
        if (!subscribeInputSettled)
            return;
        settlementUnsubRef.current?.();
        const unsub = subscribeInputSettled((_seq, ok) => {
            settlementUnsubRef.current?.();
            settlementUnsubRef.current = null;
            onSettle(ok);
        });
        settlementUnsubRef.current = unsub;
    }, [subscribeInputSettled]);
    const hasAttachments = attachments.length > 0;
    // Minimizing with staged (never-sent) images would silently strand them, so
    // intercept close and prompt to discard first. Send/clear paths call onClose
    // directly (after clearing), so they never hit this prompt.
    const requestClose = useCallback(() => {
        if (hasAttachments) {
            setShowDiscardPrompt(true);
            return;
        }
        onClose();
    }, [hasAttachments, onClose]);
    const confirmDiscard = useCallback(() => {
        onClearAttachments?.();
        setShowDiscardPrompt(false);
        onClose();
    }, [onClearAttachments, onClose]);
    const handleFilesPicked = useCallback((e) => {
        const files = Array.from(e.target.files ?? []);
        e.target.value = "";
        if (files.length > 0)
            onAttachFiles?.(files);
    }, [onAttachFiles]);
    const handleSend = useCallback(async () => {
        if (status === "sending" || status === "uploading")
            return;
        const text = draft.getValue();
        if (text.length === 0 && !hasAttachments)
            return;
        setErrorMsg(null);
        // Upload staged files first so a failure never clears text/attachments.
        let paths = [];
        if (hasAttachments && resolveAttachmentPaths) {
            setStatus("uploading");
            try {
                paths = await resolveAttachmentPaths();
            }
            catch {
                setStatus("failed");
                setErrorMsg(t(strings.composer.uploadFailed));
                return;
            }
        }
        const payload = composeComposerPayload(text, paths);
        const result = onInput(payload, "toolbar-submit");
        if (result.status === "rejected") {
            // "disposed" — pane torn down. Keep the draft/attachments; go idle.
            setStatus("idle");
            return;
        }
        if (result.status === "queued") {
            // Not sent immediately (connection lost / gate paused). Per the send
            // contract we only clear+minimize on an ok settlement, so keep the draft
            // and attachments and surface the queued state.
            setStatus("queued");
            return;
        }
        setStatus("sending");
        const finalizeSuccess = () => {
            draft.reset();
            onClearAttachments?.();
            setStatus("idle");
            onClose();
            onFocusTerminal?.();
        };
        const finalizeFailure = () => {
            setStatus("failed");
            setErrorMsg(t(strings.composer.sendFailed));
        };
        if (subscribeInputSettled) {
            awaitNextSettlement((ok) => (ok ? finalizeSuccess() : finalizeFailure()));
        }
        else {
            finalizeSuccess();
        }
    }, [
        status,
        draft,
        hasAttachments,
        resolveAttachmentPaths,
        onInput,
        subscribeInputSettled,
        awaitNextSettlement,
        onClearAttachments,
        onClose,
        onFocusTerminal,
        t,
    ]);
    const isBusy = status === "sending" || status === "uploading";
    const canSend = true; // empty+no-attachments is guarded inside handleSend
    return (_jsx(DrawerShell, { open: open, onClose: requestClose, closeAriaLabel: t(strings.composer.closeAriaLabel), title: t(strings.composer.title), panelTestId: "full-screen-composer", avoidKeyboard: true, children: _jsxs("div", { className: "relative flex h-full flex-col", children: [_jsx("textarea", { ref: textareaRef, "data-testid": "composer-input", defaultValue: draft.getValue(), onChange: handleChange, onSelect: (e) => draft.trackSelection(e.currentTarget), onBlur: (e) => draft.trackSelection(e.currentTarget), autoComplete: "off", autoCorrect: "on", spellCheck: true, placeholder: t(strings.composer.placeholder), className: "min-h-0 flex-1 resize-none bg-transparent px-4 py-3 text-base text-wc-text-primary placeholder:text-wc-text-muted outline-none" }), (status === "queued" || status === "failed") && errorMsg !== null && (_jsx("div", { "data-testid": "composer-error", className: cn("px-4 py-1 text-xs", status === "failed" ? "text-red-400" : "text-yellow-400"), children: errorMsg })), status === "queued" && errorMsg === null && (_jsx("div", { "data-testid": "composer-status-queued", className: "px-4 py-1 text-xs text-yellow-400", children: t(strings.mobileToolbar.statusQueued) })), onRemoveAttachment && (_jsx("div", { className: "px-4", children: _jsx(AttachmentPreviewTray, { attachments: attachments, onRemove: onRemoveAttachment, removeAriaLabel: t(strings.composer.removeAttachmentAriaLabel) }) })), _jsxs("div", { className: "flex items-stretch gap-2 border-t border-wc-default px-3 pt-2 pb-[max(0.5rem,var(--wc-safe-bottom,0px))]", children: [onAttachFiles && (_jsxs(_Fragment, { children: [_jsx("button", { type: "button", "data-testid": "composer-attach", onClick: () => fileInputRef.current?.click(), disabled: isBusy, className: "flex shrink-0 items-center justify-center rounded border border-wc-default bg-wc-surface-input p-2 text-wc-text-secondary transition hover:text-wc-text-primary disabled:opacity-50", title: t(strings.composer.attachImageTitle), "aria-label": t(strings.composer.attachImageTitle), children: _jsx(ImagePlus, { className: "h-4 w-4" }) }), _jsx("input", { ref: fileInputRef, type: "file", accept: "image/jpeg,image/png,image/gif,image/webp", multiple: true, hidden: true, "data-testid": "composer-file-input", onChange: handleFilesPicked })] })), mic, _jsx("div", { className: "min-w-0 flex-1" }), _jsxs("button", { type: "button", "data-testid": "composer-send", onClick: handleSend, disabled: !canSend || isBusy, className: "inline-flex shrink-0 items-center gap-1.5 rounded border border-wc-accent bg-wc-accent/20 px-4 py-2 text-sm font-medium text-wc-text-primary transition active:bg-wc-accent-active disabled:opacity-60", title: isBusy ? t(strings.composer.sendingTitle) : t(strings.composer.sendTitle), children: [isBusy ? (_jsx(Loader2, { "data-testid": "composer-sending", className: "h-4 w-4 animate-spin" })) : (_jsx(SendHorizontal, { className: "h-4 w-4" })), _jsx("span", { children: t(strings.composer.sendTitle) })] })] }), _jsx(ConfirmDialog, { open: showDiscardPrompt, title: t(strings.composer.discardTitle), body: t(strings.composer.discardMessage, { count: attachments.length }), cancelLabel: t(strings.composer.discardCancel), confirmLabel: t(strings.composer.discardConfirm), destructive: true, onCancel: () => setShowDiscardPrompt(false), onConfirm: confirmDiscard, testIdPrefix: "composer-discard" })] }) }));
}
