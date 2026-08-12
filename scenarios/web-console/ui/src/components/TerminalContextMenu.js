import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useState } from "react";
import { ClipboardPaste, Copy, Image, TextSelect, Trash2, Volume2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";
/**
 * PASTE_SUCCESS_FLASH_MS is how long the "Pasted" confirmation stays
 * visible before the menu auto-closes. Short enough to feel snappy.
 */
const PASTE_SUCCESS_FLASH_MS = 600;
/**
 * PASTE_FAILURE_HOLD_MS is how long the failure reason stays visible
 * before the menu auto-closes. Longer than the success flash so the
 * user can actually read the reason.
 */
const PASTE_FAILURE_HOLD_MS = 3000;
export default function TerminalContextMenu({ position, hasSelection, onCopy, onPaste, onSelectAll, onClear, onUploadImage, onSpeak, onClose, }) {
    const { t } = useTranslation();
    const [pasteState, setPasteState] = useState({ kind: "idle" });
    const handlePaste = useCallback(async () => {
        let text;
        try {
            text = await navigator.clipboard.readText();
        }
        catch {
            setPasteState({ kind: "clipboard_unavailable" });
            setTimeout(() => setPasteState({ kind: "idle" }), 2000);
            return;
        }
        if (!text) {
            onClose();
            return;
        }
        setPasteState({ kind: "pending" });
        try {
            const res = await onPaste(text);
            if (res.status === "ok") {
                setPasteState({ kind: "succeeded" });
                setTimeout(onClose, PASTE_SUCCESS_FLASH_MS);
            }
            else {
                setPasteState({ kind: "failed", reason: res.reason });
                setTimeout(onClose, PASTE_FAILURE_HOLD_MS);
            }
        }
        catch (err) {
            const reason = err instanceof Error ? err.message : String(err);
            setPasteState({ kind: "failed", reason });
            setTimeout(onClose, PASTE_FAILURE_HOLD_MS);
        }
    }, [onPaste, onClose]);
    const pasteLabel = (() => {
        switch (pasteState.kind) {
            case "pending":
                return t(strings.terminalContextMenu.pasting);
            case "succeeded":
                return t(strings.terminalContextMenu.pasted);
            case "failed":
                return t(strings.terminalContextMenu.pasteFailed, { reason: pasteState.reason });
            case "clipboard_unavailable":
                return t(strings.terminalContextMenu.useCtrlVHint);
            case "idle":
            default:
                return t(strings.terminalContextMenu.paste);
        }
    })();
    const pasteDisabled = pasteState.kind === "pending";
    return (_jsxs(ContextMenuBase, { position: position, onClose: onClose, "data-testid": "terminal-context-menu", children: [hasSelection && (_jsxs("button", { "data-testid": "ctx-copy", className: contextMenuItemClass, onClick: onCopy, children: [_jsx(Copy, { className: "h-4 w-4 shrink-0" }), t(strings.terminalContextMenu.copy)] })), hasSelection && onSpeak && (_jsxs("button", { "data-testid": "ctx-speak", className: contextMenuItemClass, onClick: onSpeak, children: [_jsx(Volume2, { className: "h-4 w-4 shrink-0" }), t(strings.terminalContextMenu.speak)] })), _jsxs("button", { "data-testid": "ctx-paste", "data-paste-state": pasteState.kind, className: contextMenuItemClass, onClick: handlePaste, disabled: pasteDisabled, children: [_jsx(ClipboardPaste, { className: "h-4 w-4 shrink-0" }), pasteLabel] }), onUploadImage && (_jsxs("button", { "data-testid": "ctx-upload-image", className: contextMenuItemClass, onClick: onUploadImage, children: [_jsx(Image, { className: "h-4 w-4 shrink-0" }), t(strings.terminalContextMenu.uploadImage)] })), _jsxs("button", { "data-testid": "ctx-select-all", className: contextMenuItemClass, onClick: onSelectAll, children: [_jsx(TextSelect, { className: "h-4 w-4 shrink-0" }), t(strings.terminalContextMenu.selectAll)] }), _jsxs("button", { "data-testid": "ctx-clear", className: contextMenuItemClass, onClick: onClear, children: [_jsx(Trash2, { className: "h-4 w-4 shrink-0" }), t(strings.terminalContextMenu.clearTerminal)] })] }));
}
