import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
// DOC: docs/concepts/ARCHITECTURE.md#system-layers
import { lazy, Suspense, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { fetchHealth } from "./api/health";
import { HEALTH_RETRY_COUNT, HEALTH_RETRY_DELAY_MS } from "./consts/config";
import { strings } from "./consts/strings";
import { Button } from "./components/ui/button";
import ErrorBoundary from "./components/ErrorBoundary";
import TopSafeArea from "./components/TopSafeArea";
import { AlertTriangle, X } from "lucide-react";
const Workspace = lazy(() => import("./components/Workspace"));
const PageFallback = () => {
    const { t } = useTranslation();
    return (_jsx("div", { className: "flex h-wc-app items-center justify-center bg-wc-surface-base text-wc-text-muted", children: t(strings.app.loading) }));
};
export default function App() {
    const { t } = useTranslation();
    const queryClient = useQueryClient();
    const [dismissed, setDismissed] = useState(false);
    const healthQuery = useQuery({
        queryKey: ["health"],
        queryFn: fetchHealth,
        retry: HEALTH_RETRY_COUNT,
        retryDelay: HEALTH_RETRY_DELAY_MS,
        // Keep polling so banner auto-clears when connection recovers
        refetchInterval: (query) => query.state.status === "error" ? 10000 : false,
    });
    const { isLoading, error, isFetching } = healthQuery;
    // Reset dismissed state when connection recovers or drops again
    const showBanner = !!error && !dismissed;
    return (_jsxs(ErrorBoundary, { region: "app", children: [_jsx("div", { className: "wc-ios-tint-edge wc-ios-tint-edge-bottom", "aria-hidden": "true" }), showBanner && (_jsx(TopSafeArea, { testId: "connection-top-edge", fillClassName: "wc-stable-theme bg-wc-error-surface", children: _jsxs("div", { "data-testid": "connection-banner", className: "wc-stable-theme flex items-center gap-2 bg-wc-error-surface border-b border-wc-error py-2 ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] text-sm text-wc-error-text", children: [_jsx(AlertTriangle, { className: "h-4 w-4 shrink-0" }), _jsx("span", { className: "flex-1", children: t(strings.app.connectionBanner.message) }), _jsx(Button, { "data-testid": "health-retry-button", variant: "outline", size: "sm", className: "shrink-0 text-xs h-7", onClick: () => {
                                setDismissed(false);
                                queryClient.invalidateQueries({ queryKey: ["health"] });
                            }, disabled: isFetching, children: isFetching ? t(strings.app.connectionBanner.retrying) : t(strings.app.connectionBanner.retry) }), _jsx("button", { "data-testid": "connection-banner-dismiss", onClick: () => setDismissed(true), className: "shrink-0 p-0.5 hover:text-red-100", "aria-label": t(strings.app.connectionBanner.dismissAriaLabel), type: "button", children: _jsx(X, { className: "h-3.5 w-3.5" }) })] }) })), _jsx(Suspense, { fallback: _jsx(PageFallback, {}), children: isLoading && !error ? (_jsx(PageFallback, {})) : (_jsx(Workspace, { topSafeAreaReserved: showBanner })) })] }));
}
