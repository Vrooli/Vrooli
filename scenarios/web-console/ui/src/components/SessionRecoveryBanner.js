import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { memo } from "react";
import { History, Loader2, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { useSessionRecovery } from "../hooks/useSessionRecovery";
/**
 * Honest indicator for asynchronous startup session recovery. While the API
 * reattaches surviving tmux sessions in the background, the session list can be
 * incomplete — without this the user would see an apparently-empty workspace and
 * assume their durable sessions were lost. Renders nothing when no recovery is
 * (or was) observed this mount. On completion it offers a reload so the freshly
 * recovered sessions appear.
 */
function SessionRecoveryBannerInner() {
    const { t } = useTranslation();
    const { inProgress, total, recovered, adopted, justCompleted } = useSessionRecovery();
    if (!inProgress && !justCompleted)
        return null;
    return (_jsx("div", { "data-testid": "session-recovery-banner", role: "status", className: "wc-stable-theme flex items-center gap-2 border-b border-blue-500/30 bg-blue-500/10 py-2 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] text-xs text-blue-200", children: inProgress ? (_jsxs(_Fragment, { children: [_jsx(Loader2, { className: "h-3.5 w-3.5 shrink-0 animate-spin", "aria-hidden": true }), _jsx("span", { className: "min-w-0 flex-1 break-words", children: total > 0
                        ? t(strings.sessionRecovery.recovering, { recovered, total })
                        : t(strings.sessionRecovery.recoveringUnknown) })] })) : (_jsxs(_Fragment, { children: [_jsx(History, { className: "h-3.5 w-3.5 shrink-0", "aria-hidden": true }), _jsx("span", { className: "min-w-0 flex-1 break-words", children: t(strings.sessionRecovery.recovered, { count: recovered + adopted }) }), _jsxs("button", { type: "button", "data-testid": "session-recovery-view", onClick: () => window.location.reload(), className: "inline-flex shrink-0 items-center gap-1 rounded border border-blue-400/40 bg-blue-500/20 px-2 py-1 font-medium text-blue-100 transition active:bg-blue-500/30", children: [_jsx(RefreshCw, { className: "h-3.5 w-3.5", "aria-hidden": true }), _jsx("span", { children: t(strings.sessionRecovery.view) })] })] })) }));
}
const SessionRecoveryBanner = memo(SessionRecoveryBannerInner);
export default SessionRecoveryBanner;
