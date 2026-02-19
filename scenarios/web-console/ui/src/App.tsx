// DOC: docs/concepts/ARCHITECTURE.md#system-layers
import { lazy, Suspense } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchHealth } from "./lib/api";
import { HEALTH_RETRY_COUNT, HEALTH_RETRY_DELAY_MS } from "./consts/config";
import { Button } from "./components/ui/button";
import ErrorBoundary from "./components/ErrorBoundary";
import { useHashRoute } from "./hooks/useHashRoute";

const Workspace = lazy(() => import("./components/Workspace"));
const SettingsPage = lazy(() => import("./pages/SettingsPage"));
const SessionsPage = lazy(() => import("./pages/SessionsPage"));

const PageFallback = () => (
  <div className="flex h-screen items-center justify-center bg-wc-surface-base text-wc-text-muted">
    Loading...
  </div>
);

export default function App() {
  const queryClient = useQueryClient();
  const { route, navigate } = useHashRoute();
  const { isLoading, error, isFetching } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    retry: HEALTH_RETRY_COUNT,
    retryDelay: HEALTH_RETRY_DELAY_MS,
  });

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-wc-surface-base text-wc-text-muted">
        Connecting to API...
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-screen items-center justify-center bg-wc-surface-base text-wc-error-detail">
        <div className="text-center">
          <p className="text-lg font-medium">Unable to reach the API</p>
          <p className="mt-2 text-sm text-wc-text-faint">
            Make sure the scenario is running via{" "}
            <code className="rounded bg-white/10 px-1">vrooli scenario start web-console</code>
          </p>
          <Button
            data-testid="health-retry-button"
            variant="outline"
            size="sm"
            className="mt-4"
            onClick={() => queryClient.invalidateQueries({ queryKey: ["health"] })}
            disabled={isFetching}
          >
            {isFetching ? "Retrying..." : "Retry Connection"}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <ErrorBoundary region="app">
      <Suspense fallback={<PageFallback />}>
        {route === "settings" && (
          <SettingsPage onBack={() => navigate("workspace")} />
        )}
        {route === "sessions" && (
          <SessionsPage onBack={() => navigate("workspace")} />
        )}
        {route === "workspace" && (
          <Workspace onNavigate={navigate} />
        )}
      </Suspense>
    </ErrorBoundary>
  );
}
