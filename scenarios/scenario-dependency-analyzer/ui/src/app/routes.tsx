import { lazy, Suspense, useCallback, useEffect, useMemo } from "react";
import { AlertCircle } from "lucide-react";
import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  useLocation,
  useNavigate,
  type RouteObject
} from "react-router-dom";

import { Card, CardContent } from "../components/ui/card";
import { AppShell } from "../layout/AppShell";
import {
  type AppRoute,
  isRouteKey,
  routeDefinitions,
  routePath
} from "./routeDefinitions";
import { useScenarioCatalog } from "../features/catalog/useScenarioCatalog";
import { useGraphData } from "../features/graph/useGraphData";
import { statusTone } from "../theme/status";
import type { GraphType, LayoutMode } from "../types";

export const routes = routeDefinitions;

const CatalogPage = lazy(() =>
  import("../features/catalog/CatalogPage").then((module) => ({ default: module.CatalogPage }))
);
const DeploymentPage = lazy(() =>
  import("../features/deployment/DeploymentPage").then((module) => ({ default: module.DeploymentPage }))
);
const GraphPage = lazy(() =>
  import("../features/graph/GraphPage").then((module) => ({ default: module.GraphPage }))
);
const GovernancePage = lazy(() =>
  import("../features/governance/GovernancePage").then((module) => ({ default: module.GovernancePage }))
);
const OrientationPage = lazy(() =>
  import("../pages/OrientationPage").then((module) => ({ default: module.OrientationPage }))
);

const routeFromLocation = (pathname: string, search: string): AppRoute => {
  const searchParams = new URLSearchParams(search);
  const legacyView = searchParams.get("view");
  if (pathname.endsWith("/graph")) return "graph";
  if (pathname.endsWith("/deployment")) return "deployment";
  if (pathname.endsWith("/catalog")) return "catalog";
  if (pathname.endsWith("/governance")) return "governance";
  return isRouteKey(legacyView) ? legacyView : "overview";
};

const routerRoutes: RouteObject[] = [
  {
    path: "*",
    element: <ScenarioDependencyAnalyzerRoutes />
  }
];

export function AppRouter() {
  const router = createBrowserRouter(routerRoutes);
  return <RouterProvider router={router} />;
}

export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routerRoutes, { initialEntries });
  return <RouterProvider router={router} />;
}

function ScenarioDependencyAnalyzerRoutes() {
  const location = useLocation();
  const routerNavigate = useNavigate();
  const activeRoute = routeFromLocation(location.pathname, location.search);

  const onNavigate = useCallback(
    (routeKey: AppRoute) => {
      const params = new URLSearchParams(location.search);
      params.delete("view");
      const search = params.toString();
      routerNavigate({
        pathname: routePath(routeKey),
        search: search ? `?${search}` : ""
      });
    },
    [location.search, routerNavigate]
  );

  const updateSearchParams = useCallback((updates: Record<string, string | null | undefined>) => {
    const params = new URLSearchParams(location.search);
    Object.entries(updates).forEach(([key, value]) => {
      if (value === null || value === undefined || value === "") {
        params.delete(key);
      } else {
        params.set(key, value);
      }
    });
    const search = params.toString();
    routerNavigate(
      {
        pathname: location.pathname,
        search: search ? `?${search}` : ""
      },
      { replace: true }
    );
  }, [location.pathname, location.search, routerNavigate]);

  const searchParams = useMemo(() => {
    return new URLSearchParams(location.search);
  }, [location.search]);

  const initialGraphType = useMemo(() => {
    const value = searchParams?.get("graph_type");
    if (value === "scenario" || value === "scenarios") return "scenario";
    if (value === "resource" || value === "resources") return "resource";
    return "combined";
  }, [searchParams]);

  const initialLayout = useMemo(() => {
    const value = searchParams?.get("layout");
    return value === "grid" || value === "radial" ? value : "force";
  }, [searchParams]);

  const {
    analyzeAll,
    apiHealthy,
    error,
    fetchGraph,
    filter,
    driftFilter,
    graph,
    graphType,
    layout,
    loading,
    selectedNode,
    setFilter,
    setDriftFilter,
    setGraphType,
    setLayout,
    setSelectedNode,
    stats
  } = useGraphData({ defaultType: initialGraphType as GraphType, defaultLayout: initialLayout as LayoutMode });

  const {
    summaries,
    loadingSummaries,
    selectedScenario,
    detail,
    detailLoading,
    scanLoading,
    optimizeLoading,
    selectScenario,
    refreshSummaries,
    scanScenario,
    optimizeScenario
  } = useScenarioCatalog();

  const handleScenarioScan = useCallback(
    (scenarioName: string, apply?: boolean) => {
      return scanScenario(scenarioName, { apply });
    },
    [scanScenario]
  );

  const handleScenarioScanForDetail = useCallback(
    (options?: { apply?: boolean }) => {
      if (!selectedScenario) return;
      void scanScenario(selectedScenario, options);
    },
    [scanScenario, selectedScenario]
  );

  const handleOptimize = useCallback(
    (options?: { apply?: boolean }) => {
      if (!selectedScenario) return;
      void optimizeScenario(selectedScenario, options);
    },
    [optimizeScenario, selectedScenario]
  );

  const handleSelectScenario = useCallback(
    (scenarioName: string) => {
      selectScenario(scenarioName);
      updateSearchParams({ scenario: scenarioName });
    },
    [selectScenario, updateSearchParams]
  );

  useEffect(() => {
    const scenario = searchParams?.get("scenario");
    if (scenario) {
      setFilter(scenario);
      setSelectedNode(null);
      selectScenario(scenario);
      if (!searchParams?.get("view") && activeRoute === "overview") {
        onNavigate("catalog");
      }
    }
  }, [activeRoute, onNavigate, searchParams, selectScenario, setFilter, setSelectedNode]);

  const handleGraphTypeChange = useCallback(
    (value: GraphType) => {
      setGraphType(value);
      updateSearchParams({ graph_type: value });
    },
    [setGraphType, updateSearchParams]
  );

  const handleLayoutChange = useCallback(
    (value: LayoutMode) => {
      setLayout(value);
      updateSearchParams({ layout: value });
    },
    [setLayout, updateSearchParams]
  );

  const handleFilterChange = useCallback(
    (value: string) => {
      setFilter(value);
      updateSearchParams({ scenario: value || null });
    },
    [setFilter, updateSearchParams]
  );

  const handleExport = () => {
    if (!graph) return;
    const data = JSON.stringify(graph, null, 2);
    const blob = new Blob([data], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `dependency-graph-${graph.graph_type}-${new Date().toISOString()}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  const handleSelectScenarioForDeployment = useCallback(
    (scenarioName: string, options?: { openCatalog?: boolean }) => {
      selectScenario(scenarioName);
      updateSearchParams({ scenario: scenarioName });
      if (options?.openCatalog) {
        onNavigate("catalog");
      }
    },
    [onNavigate, selectScenario, updateSearchParams]
  );

  return (
    <AppShell activeRoute={activeRoute} onNavigate={onNavigate}>
      {error ? (
        <Card className={`border text-sm ${statusTone("danger").panel}`}>
          <CardContent className="flex items-center gap-3 py-4">
            <AlertCircle className="h-4 w-4" aria-hidden="true" />
            <div>
              <p className="font-semibold">We hit turbulence while loading data.</p>
              <p className="text-xs opacity-80">{error}</p>
            </div>
          </CardContent>
        </Card>
      ) : null}

      <Suspense fallback={<RouteLoadingState />}>
        {activeRoute === "overview" ? (
          <OrientationPage
            onAnalyzeAll={() => void analyzeAll()}
            onGoGraph={() => onNavigate("graph")}
            onGoDeployment={() => onNavigate("deployment")}
            onGoCatalog={() => onNavigate("catalog")}
            hasGraphData={Boolean(graph)}
            hasScenarioSummaries={summaries.length > 0}
            apiHealthy={apiHealthy}
          />
        ) : null}

        {activeRoute === "graph" ? (
          <GraphPage
            graph={graph}
            graphType={graphType}
            layout={layout}
            filter={filter}
            driftFilter={driftFilter}
            loading={loading}
            selectedNode={selectedNode}
            apiHealthy={apiHealthy}
            stats={stats}
            onGraphTypeChange={handleGraphTypeChange}
            onLayoutChange={handleLayoutChange}
            onFilterChange={handleFilterChange}
            onDriftFilterChange={setDriftFilter}
            onSelectNode={setSelectedNode}
            onRefresh={() => fetchGraph(graphType)}
            onAnalyzeAll={analyzeAll}
            onExport={handleExport}
          />
        ) : null}

        {activeRoute === "deployment" ? (
          <DeploymentPage
            scenarios={summaries}
            loading={loadingSummaries}
            onRefresh={refreshSummaries}
            onScanScenario={handleScenarioScan}
            onSelectScenario={handleSelectScenarioForDeployment}
          />
        ) : null}

        {activeRoute === "catalog" ? (
          <CatalogPage
            scenarios={summaries}
            selectedScenario={selectedScenario}
            detail={detail}
            loadingSummaries={loadingSummaries}
            detailLoading={detailLoading}
            scanLoading={scanLoading}
            optimizeLoading={optimizeLoading}
            onSelectScenario={handleSelectScenario}
            onRefresh={refreshSummaries}
            onScan={handleScenarioScanForDetail}
            onOptimize={handleOptimize}
          />
        ) : null}

        {activeRoute === "governance" ? <GovernancePage /> : null}
      </Suspense>
    </AppShell>
  );
}

function RouteLoadingState() {
  return (
    <Card className={`border text-sm ${statusTone("info").panel}`} aria-live="polite">
      <CardContent className="py-4">
        <p className="font-semibold">Loading workspace surface...</p>
      </CardContent>
    </Card>
  );
}
