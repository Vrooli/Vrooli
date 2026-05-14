// DOC: docs/concepts/ARCHITECTURE.md#system-layers
import { lazy, Suspense, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { fetchHealth } from "./api/health";
import { HEALTH_RETRY_COUNT, HEALTH_RETRY_DELAY_MS } from "./consts/config";
import { strings } from "./consts/strings";
import { Button } from "./components/ui/button";
import ErrorBoundary from "./components/ErrorBoundary";
import { AlertTriangle, X } from "lucide-react";

const Workspace = lazy(() => import("./components/Workspace"));

const PageFallback = () => {
  const { t } = useTranslation();
  return (
    <div className="flex h-wc-app items-center justify-center bg-wc-surface-base text-wc-text-muted">
      {t(strings.app.loading)}
    </div>
  );
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
    refetchInterval: (query) => query.state.status === "error" ? 10_000 : false,
  });
  const { isLoading, error, isFetching } = healthQuery;

  // Reset dismissed state when connection recovers or drops again
  const showBanner = !!error && !dismissed;

  return (
    <ErrorBoundary region="app">
      {/* Connection banner — shown above workspace when API is unreachable */}
      {showBanner && (
        <div
          data-testid="connection-banner"
          className="flex items-center gap-2 bg-wc-error-surface border-b border-wc-error px-4 py-2 text-sm text-wc-error-text"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span className="flex-1">
            {t(strings.app.connectionBanner.message)}
          </span>
          <Button
            data-testid="health-retry-button"
            variant="outline"
            size="sm"
            className="shrink-0 text-xs h-7"
            onClick={() => {
              setDismissed(false);
              queryClient.invalidateQueries({ queryKey: ["health"] });
            }}
            disabled={isFetching}
          >
            {isFetching ? t(strings.app.connectionBanner.retrying) : t(strings.app.connectionBanner.retry)}
          </Button>
          <button
            data-testid="connection-banner-dismiss"
            onClick={() => setDismissed(true)}
            className="shrink-0 p-0.5 hover:text-red-100"
            aria-label={t(strings.app.connectionBanner.dismissAriaLabel)}
            type="button"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {/* Always render workspace — even during initial load or error */}
      <Suspense fallback={<PageFallback />}>
        {isLoading && !error ? (
          <PageFallback />
        ) : (
          <Workspace />
        )}
      </Suspense>
    </ErrorBoundary>
  );
}
