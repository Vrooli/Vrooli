import { useCallback, useMemo } from "react";
import { AlertCircle, Loader2 } from "lucide-react";

import { ControlPanel } from "./components/ControlPanel";
import { GraphCanvas } from "./components/graph/GraphCanvas";
import { SelectedNodePanel } from "./components/SelectedNodePanel";
import { StatsPanel } from "./components/StatsPanel";
import { ScenarioCatalogPanel } from "./components/ScenarioCatalogPanel";
import { ScenarioDetailPanel } from "./components/ScenarioDetailPanel";
import { Badge } from "./components/ui/badge";
import { Card, CardContent } from "./components/ui/card";
import { useGraphData } from "./hooks/useGraphData";
import { useScenarioCatalog } from "./hooks/useScenarioCatalog";
import type { DependencyGraphNode } from "./types";

export default function App() {
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
  } = useGraphData();
  const {
    summaries,
    loadingSummaries,
    selectedScenario,
    detail,
    detailLoading,
    scanLoading,
    selectScenario,
    refreshSummaries,
    scanScenario
  } = useScenarioCatalog();

  const handleScenarioScan = useCallback(
    (options?: { apply?: boolean }) => {
      if (!selectedScenario) return;
      void scanScenario(selectedScenario, options);
    },
    [scanScenario, selectedScenario]
  );

  const influentialNodes = useMemo(() => {
    if (!graph) return [] as Array<{ node: DependencyGraphNode; score: number }>;
    const map = new Map<string, { node: DependencyGraphNode; score: number }>();

    graph.nodes.forEach((node) => {
      map.set(node.id, { node, score: 0 });
    });

    graph.edges.forEach((edge) => {
      const baseWeight = edge.required ? 2 : 1;
      const weight = baseWeight + Math.max(edge.weight, 0);
      const currentSource = map.get(edge.source);
      const currentTarget = map.get(edge.target);
      if (currentSource) currentSource.score += weight;
      if (currentTarget) currentTarget.score += weight;
    });

    return Array.from(map.values())
      .sort((a, b) => b.score - a.score)
      .slice(0, 8);
  }, [graph]);

  const selectedConnections = useMemo(() => {
    if (!graph || !selectedNode) return [];
    return graph.edges
      .filter((edge) => edge.source === selectedNode.id || edge.target === selectedNode.id)
      .map((edge) => {
        const peerId = edge.source === selectedNode.id ? edge.target : edge.source;
        const peer = graph.nodes.find((node) => node.id === peerId);
        if (!peer) return null;
        return { node: peer, edge };
      })
      .filter(Boolean) as Array<{ node: DependencyGraphNode; edge: typeof graph.edges[number] }>;
  }, [graph, selectedNode]);

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
                Explore how scenarios, resources, and workflows interlock. Run holistic analysis, assess
                impact, and unlock optimized deployment strategies with a responsive, accessible, and beautiful interface.
              </p>
            </div>
            <div className="rounded-xl border border-border/40 bg-background/60 px-4 py-3 text-right text-xs text-muted-foreground">
              <p className="font-medium text-foreground">Live business signal</p>
              <p>Revenue leverage: $10K - $50K per shipped scenario</p>
              <p>Compound intelligence → Autonomous delivery</p>
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

        <section className="grid flex-1 gap-6 xl:grid-cols-[320px_minmax(0,1fr)_320px]">
          <aside className="space-y-6">
            <ControlPanel
              graphType={graphType}
              layout={layout}
              filter={filter}
              driftFilter={driftFilter}
              onGraphTypeChange={(value) => setGraphType(value)}
              onLayoutChange={(value) => setLayout(value)}
              onFilterChange={setFilter}
              onDriftFilterChange={setDriftFilter}
              onRefresh={() => fetchGraph(graphType)}
              onAnalyzeAll={analyzeAll}
              onExport={handleExport}
              loading={loading}
            />
            <Card className="glass border border-border/40">
              <CardContent className="space-y-3 py-5 text-sm text-muted-foreground">
                <p className="text-foreground">Why this matters</p>
                <p>
                  Every visualization crystallizes how Vrooli deploys compound intelligence across the local
                  ecosystem. Use these insights to prioritize engineering investment, derisk launches, and
                  unlock new business capabilities.
                </p>
              </CardContent>
            </Card>
          </aside>

          <div className="relative min-h-[540px] overflow-hidden rounded-3xl border border-border/50 bg-background/40 shadow-2xl shadow-black/40">
            {loading ? (
              <div className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-background/70 backdrop-blur">
                <Loader2 className="h-8 w-8 animate-spin text-primary" aria-hidden="true" />
                <p className="text-sm text-muted-foreground">Compiling live dependency intelligence…</p>
              </div>
            ) : null}
            <GraphCanvas
              graph={graph}
              layout={layout}
              filter={filter}
              driftFilter={driftFilter}
              selectedNodeId={selectedNode?.id}
              onSelectNode={(node) => setSelectedNode(node)}
              className="h-full"
            />
          </div>

          <aside className="space-y-6">
            <StatsPanel
              stats={stats}
              apiHealthy={apiHealthy}
              influentialNodes={influentialNodes}
            />
            {selectedNode ? (
              <SelectedNodePanel node={selectedNode} connections={selectedConnections} />
            ) : (
              <Card className="glass border border-border/40">
                <CardContent className="space-y-3 py-6 text-sm text-muted-foreground">
                  <p className="font-medium text-foreground">Select a node to inspect dependencies</p>
                  <p>
                    Hover or click any scenario or resource to surface its purpose, relationships, and
                    contextual metadata. Keyboard users can use <kbd className="rounded border border-border/60 bg-background/50 px-1">Tab</kbd> to focus nodes.
                  </p>
                </CardContent>
              </Card>
            )}
          </aside>
        </section>

        <section className="grid gap-6 lg:grid-cols-[320px_minmax(0,1fr)]">
          <ScenarioCatalogPanel
            scenarios={summaries}
            selected={selectedScenario}
            loading={loadingSummaries}
            onSelect={selectScenario}
            onRefresh={refreshSummaries}
          />
          <ScenarioDetailPanel
            detail={detail}
            loading={detailLoading}
            scanning={scanLoading}
            onScan={handleScenarioScan}
          />
        </section>
      </main>
    </div>
  );
}
