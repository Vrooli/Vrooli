import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useRef, useCallback } from "react";
import { Check, X, Mic } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
/** Auto-dismiss timeout for unacted command suggestions. */
const AUTO_DISMISS_MS = 5000;
export default function VoiceCommandSuggestion({ suggestion, onConfirm, onDismiss, }) {
    const { t } = useTranslation();
    const timerRef = useRef(null);
    // Auto-dismiss after timeout
    useEffect(() => {
        timerRef.current = setTimeout(() => {
            onDismiss(suggestion);
        }, AUTO_DISMISS_MS);
        return () => {
            if (timerRef.current)
                clearTimeout(timerRef.current);
        };
    }, [suggestion, onDismiss]);
    const handleConfirm = useCallback(() => {
        if (timerRef.current)
            clearTimeout(timerRef.current);
        onConfirm(suggestion);
    }, [suggestion, onConfirm]);
    const handleDismiss = useCallback(() => {
        if (timerRef.current)
            clearTimeout(timerRef.current);
        onDismiss(suggestion);
    }, [suggestion, onDismiss]);
    return (_jsxs("div", { "data-testid": "voice-command-suggestion", "data-audio-state": "command-suggestion", className: "flex items-center gap-2 border-t border-wc-default bg-wc-surface-raised px-2 py-1.5 animate-in slide-in-from-bottom-2 duration-200 touch-manipulation select-none", onMouseDown: (e) => e.preventDefault(), children: [_jsx(Mic, { className: "h-3.5 w-3.5 shrink-0 text-cyan-400" }), _jsx("span", { className: "flex-1 text-xs text-wc-text-primary truncate", children: suggestion.description }), _jsx("button", { "data-testid": "voice-command-confirm", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: handleConfirm, className: "shrink-0 rounded border border-green-500/40 bg-green-500/10 p-1.5 text-green-400 transition active:bg-green-500/25 touch-manipulation", title: t(strings.voiceCommandSuggestion.executeTitle), children: _jsx(Check, { className: "h-3.5 w-3.5" }) }), _jsx("button", { "data-testid": "voice-command-dismiss", tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: handleDismiss, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation", title: t(strings.voiceCommandSuggestion.dismissTitle), children: _jsx(X, { className: "h-3.5 w-3.5" }) })] }));
}
