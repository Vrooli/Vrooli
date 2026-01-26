import { useMemo } from "react";
import { ErrorBoundary } from "./shared/components/ErrorBoundary";
import { AppHeader } from "./shared/components/AppHeader";
import { HeaderFallback } from "./shared/components/HeaderFallback";
import { getPageTitle } from "./shared/controllers/routeController";
import { useHealthStatus } from "./shared/hooks/knowledgeHooks";
import { useHashRoute } from "./shared/hooks/useHashRoute";
import { DashboardPage } from "./surfaces/dashboard/DashboardPage";
import { GraphPage } from "./surfaces/graph/GraphPage";
import { MetricsPage } from "./surfaces/metrics/MetricsPage";
import { SearchPage } from "./surfaces/search/SearchPage";

// AI_CHECK: REACT_STABILITY=6 | LAST: 2026-01-25

export default function App() {
  const { route, navigate } = useHashRoute();
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

      {route === "dashboard" && <DashboardPage health={healthState} onNavigate={navigate} />}
      {route === "search" && <SearchPage onNavigate={navigate} />}
      {route === "metrics" && <MetricsPage onNavigate={navigate} />}
      {route === "graph" && <GraphPage onNavigate={navigate} />}
    </div>
  );
}
