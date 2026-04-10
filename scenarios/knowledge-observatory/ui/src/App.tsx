import { Suspense, lazy, useEffect, useMemo } from "react";
import { ErrorBoundary } from "./shared/components/ErrorBoundary";
import { AppHeader } from "./shared/components/AppHeader";
import { HeaderFallback } from "./shared/components/HeaderFallback";
import { PageShell } from "./shared/components/PageShell";
import { getPageTitle } from "./shared/controllers/routeController";
import { useHealthStatus } from "./shared/hooks/knowledgeHooks";
import { useHashRoute } from "./shared/hooks/useHashRoute";
import { DashboardPage } from "./surfaces/dashboard/DashboardPage";

const ExplorerPage = lazy(() =>
  import("./surfaces/explorer/ExplorerPage").then((module) => ({ default: module.ExplorerPage }))
);
const GraphPage = lazy(() =>
  import("./surfaces/graph/GraphPage").then((module) => ({ default: module.GraphPage }))
);
const MetricsPage = lazy(() =>
  import("./surfaces/metrics/MetricsPage").then((module) => ({ default: module.MetricsPage }))
);
const CollectionDetailsPage = lazy(() =>
  import("./surfaces/metrics/CollectionDetailsPage").then((module) => ({ default: module.CollectionDetailsPage }))
);
const SearchPage = lazy(() =>
  import("./surfaces/search/SearchPage").then((module) => ({ default: module.SearchPage }))
);

// Viewer is now integrated into Explorer - redirect old viewer URLs
function ViewerRedirect({ onNavigate }: { onNavigate: (route: "explorer") => void }) {
  useEffect(() => {
    // Redirect viewer route to explorer
    onNavigate("explorer");
  }, [onNavigate]);
  return (
    <PageShell>
      <div className="ko-card p-6">
        <p className="ko-text-sm ko-muted">Redirecting to Explorer...</p>
      </div>
    </PageShell>
  );
}

// AI_CHECK: REACT_STABILITY=6 | LAST: 2026-01-25

export default function App() {
  const { route, navigate, collectionName, navigateToCollection } = useHashRoute();
  const healthState = useHealthStatus();
  const pageTitle = useMemo(() => getPageTitle(route), [route]);

  return (
    <div className="ko-app-shell">
      <ErrorBoundary
        fallback={({ error, reset }) => (
          <HeaderFallback errorMessage={error.message} onRetry={reset} onNavigate={navigate} />
        )}
      >
        <AppHeader
          route={route}
          pageTitle={pageTitle}
          statusPulse={healthState.viewModel.statusPulse}
          statusLabel={healthState.viewModel.statusLabel}
        />
      </ErrorBoundary>

      <Suspense
        fallback={
          <PageShell>
            <div className="ko-card p-6">
              <p className="ko-text-sm ko-muted">Loading workspace…</p>
            </div>
          </PageShell>
        }
      >
        {route === "dashboard" && <DashboardPage health={healthState} onNavigate={navigate} />}
        {route === "search" && <SearchPage onNavigate={navigate} />}
        {route === "explorer" && <ExplorerPage onNavigate={navigate} />}
        {route === "viewer" && <ViewerRedirect onNavigate={navigate} />}
        {route === "metrics" && <MetricsPage onNavigate={navigate} onOpenCollection={navigateToCollection} />}
        {route === "collection" && (
          <CollectionDetailsPage
            collectionName={collectionName}
            onNavigate={navigate}
            onOpenCollection={navigateToCollection}
          />
        )}
        {route === "graph" && <GraphPage onNavigate={navigate} />}
      </Suspense>
    </div>
  );
}
