import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
// DOC: docs/internal/ERROR_SEMANTICS.md#client-side-failure-handling
// DOC: docs/internal/SEAMS.md#axis-3-error-codes--recovery-api--ui
import { AlertTriangle, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
// [REQ:P0-001b] Independent Pane Session Lifecycle — error feedback
export default function ErrorBanner({ error, onDismiss, onRetry, className = "", }) {
    const { t } = useTranslation();
    return (_jsxs("div", { "data-testid": "create-error-banner", className: cn("wc-stable-theme rounded-md border border-wc-error bg-wc-error-surface py-2 ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] text-sm text-wc-error-text", className), children: [_jsxs("div", { className: "flex items-center gap-2", children: [_jsx(AlertTriangle, { className: "h-4 w-4 shrink-0" }), _jsx("span", { className: "flex-1", children: error.message }), error.retry && onRetry && (_jsx("button", { "data-testid": "error-retry-button", onClick: onRetry, className: "shrink-0 text-xs underline hover:text-red-100", children: t(strings.errorBanner.retry) })), _jsx("button", { onClick: onDismiss, "aria-label": t(strings.errorBanner.dismiss), className: "shrink-0 p-0.5 hover:text-red-100", children: _jsx(X, { className: "h-3 w-3" }) })] }), error.recovery && (_jsx("p", { "data-testid": "error-recovery-hint", className: "mt-1 text-xs text-wc-error-detail/70 ps-6", children: error.recovery }))] }));
}
