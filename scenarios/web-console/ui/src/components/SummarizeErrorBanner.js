import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback } from "react";
import { AlertTriangle, Loader2, RotateCcw, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
/**
 * Persistent notice shown when TTS summarization fails (auto or on-demand).
 * Stays visible until the user dismisses or a retry succeeds. Mirrors the
 * visual language of VoiceRejectionBanner so the two failure modes read
 * consistently.
 */
export default function SummarizeErrorBanner({ state, onRetry, onDismiss, }) {
    const { t } = useTranslation();
    const handleRetry = useCallback(() => { onRetry(); }, [onRetry]);
    const handleDismiss = useCallback(() => { onDismiss(); }, [onDismiss]);
    const isRetrying = state.status === "retrying";
    const sourceLabel = state.source === "auto"
        ? t(strings.summarizeError.autoFailed)
        : t(strings.summarizeError.failed);
    return (_jsxs("div", { "data-testid": "summarize-error-banner", "data-source": state.source, "data-status": state.status, className: "wc-stable-theme flex items-start gap-2 border-b border-amber-500/30 bg-amber-500/10 py-2 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] text-xs text-amber-200", role: "status", children: [_jsx(AlertTriangle, { className: "mt-0.5 h-3.5 w-3.5 shrink-0", "aria-hidden": true }), _jsxs("div", { className: "flex-1 min-w-0", children: [_jsx("div", { className: "font-medium", children: sourceLabel }), _jsx("div", { className: "mt-0.5 break-words text-amber-200/80", children: state.message })] }), _jsxs("button", { type: "button", "data-testid": "summarize-error-retry", onClick: handleRetry, disabled: isRetrying, className: "shrink-0 inline-flex items-center gap-1 rounded border border-amber-400/40 bg-amber-500/20 px-2 py-1 font-medium text-amber-100 transition active:bg-amber-500/30 disabled:cursor-not-allowed disabled:opacity-60", title: t(strings.summarizeError.retryTitle), children: [isRetrying ? (_jsx(Loader2, { className: "h-3.5 w-3.5 animate-spin", "aria-hidden": true })) : (_jsx(RotateCcw, { className: "h-3.5 w-3.5", "aria-hidden": true })), _jsx("span", { children: isRetrying ? t(strings.summarizeError.retrying) : t(strings.summarizeError.retry) })] }), _jsx("button", { type: "button", "data-testid": "summarize-error-dismiss", onClick: handleDismiss, disabled: isRetrying, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input p-1 text-wc-text-secondary transition active:bg-wc-accent-active disabled:cursor-not-allowed disabled:opacity-60", title: t(strings.summarizeError.dismiss), "aria-label": t(strings.summarizeError.dismissAriaLabel), children: _jsx(X, { className: "h-3.5 w-3.5" }) })] }));
}
