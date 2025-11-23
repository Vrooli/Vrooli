import { useCallback, useMemo, useState, useEffect } from "react";
import { AlertCircle } from "lucide-react";

import { Badge } from "./components/ui/badge";
import { Card, CardContent } from "./components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./components/ui/tabs";
import { OrientationPage } from "./pages/OrientationPage";
import { useGraphData } from "./hooks/useGraphData";
import { useScenarioCatalog } from "./hooks/useScenarioCatalog";
import { GraphPage } from "./pages/GraphPage";
import { DeploymentPage } from "./pages/DeploymentPage";
import { CatalogPage } from "./pages/CatalogPage";
import type { GraphType, LayoutMode } from "./types";

export default function App() {
  const searchParams = useMemo(() => {
    if (typeof window === "undefined") return null;
    return new URLSearchParams(window.location.search);
  }, []);

  const initialGraphType = useMemo(() => {
    const value = searchParams?.get("graph_type");
    return value === "scenarios" || value === "resources" || value === "combined" ? value : "combined";
  }, [searchParams]);

  const initialLayout = useMemo(() => {
    const value = searchParams?.get("layout");
    return value === "grid" || value === "hierarchical" ? value : "force";
  }, [searchParams]);

  const [activeTab, setActiveTab] = useState(() => searchParams?.get("view") ?? "overview");

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
      void scanScenario(scenarioName, { apply });
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

  useEffect(() => {
    const scenario = searchParams?.get("scenario");
    if (scenario) {
      setFilter(scenario);
      setSelectedNode(null);
    }

    const view = searchParams?.get("view");
    if (view === "overview" || view === "graph" || view === "deployment" || view === "catalog") {
      setActiveTab(view);
    }
  }, [searchParams, setFilter, setSelectedNode]);

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
    (scenarioName: string) => {
      selectScenario(scenarioName);
      setActiveTab("catalog");
    },
    [selectScenario]
  );

  return (
    <div className="min-h-screen bg-transparent text-foreground">
      <div className="relative overflow-hidden">
        <div className="pointer-events-none absolute inset-0 opacity-40">
          <div className="absolute -left-32 top-24 h-72 w-72 rounded-full bg-primary/30 blur-3xl" />
          <div className="absolute -top-20 right-0 h-96 w-96 rounded-full bg-purple-500/20 blur-3xl" />
        </div>
        <header className="z-10 px-6 pb-4 pt-10 sm:px-10">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="space-y-3">
              <Badge className="uppercase tracking-wide" variant="secondary">
                Scenario Dependency Analyzer
              </Badge>
              <h1 className="text-3xl font-semibold text-foreground sm:text-4xl">
                Systems-level intelligence for the Vrooli ecosystem
              </h1>
              <p className="max-w-2xl text-sm text-muted-foreground">
                Explore dependency graphs, assess deployment readiness, and manage scenario metadata with a responsive,
                accessible interface designed for compound intelligence. Start in Orientation to get a quick walkthrough.
              </p>
            </div>
          </div>
        </header>
      </div>

      <main className="flex flex-1 flex-col gap-6 px-6 pb-10 sm:px-10">
        {error ? (
          <Card className="border border-red-500/40 bg-red-500/10 text-sm text-red-100">
            <CardContent className="flex items-center gap-3 py-4">
              <AlertCircle className="h-4 w-4" aria-hidden="true" />
              <div>
                <p className="font-semibold">We hit turbulence while loading data.</p>
                <p className="text-xs text-red-100/80">{error}</p>
              </div>
            </CardContent>
          </Card>
        ) : null}

        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="mb-6 grid w-full max-w-[800px] grid-cols-4">
            <TabsTrigger value="overview">Orientation</TabsTrigger>
            <TabsTrigger value="graph">Dependency Graph</TabsTrigger>
            <TabsTrigger value="deployment">Deployment Readiness</TabsTrigger>
            <TabsTrigger value="catalog">Scenario Catalog</TabsTrigger>
          </TabsList>

          {/* Tab 0: Orientation */}
          <TabsContent value="overview" className="space-y-6">
            <OrientationPage
              onAnalyzeAll={() => void analyzeAll()}
              onGoGraph={() => setActiveTab("graph")}
              onGoDeployment={() => setActiveTab("deployment")}
              onGoCatalog={() => setActiveTab("catalog")}
              hasGraphData={Boolean(graph)}
              hasScenarioSummaries={summaries.length > 0}
              apiHealthy={apiHealthy}
            />
          </TabsContent>

          {/* Tab 1: Dependency Graph */}
          <TabsContent value="graph" className="space-y-6">
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
              onGraphTypeChange={setGraphType}
              onLayoutChange={setLayout}
              onFilterChange={setFilter}
              onDriftFilterChange={setDriftFilter}
              onSelectNode={setSelectedNode}
              onRefresh={() => fetchGraph(graphType)}
              onAnalyzeAll={analyzeAll}
              onExport={handleExport}
            />
          </TabsContent>

          {/* Tab 2: Deployment Readiness */}
          <TabsContent value="deployment" className="space-y-6">
            <DeploymentPage
              scenarios={summaries}
              loading={loadingSummaries}
              onRefresh={refreshSummaries}
              onScanScenario={handleScenarioScan}
              onSelectScenario={handleSelectScenarioForDeployment}
            />
          </TabsContent>

          {/* Tab 3: Scenario Catalog */}
          <TabsContent value="catalog" className="space-y-6">
            <CatalogPage
              scenarios={summaries}
              selectedScenario={selectedScenario}
              detail={detail}
              loadingSummaries={loadingSummaries}
              detailLoading={detailLoading}
              scanLoading={scanLoading}
              optimizeLoading={optimizeLoading}
              onSelectScenario={selectScenario}
              onRefresh={refreshSummaries}
              onScan={handleScenarioScanForDetail}
              onOptimize={handleOptimize}
            />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  );
}
