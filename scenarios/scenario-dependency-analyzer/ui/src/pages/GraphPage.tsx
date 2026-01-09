import { useMemo } from "react";
import { Loader2 } from "lucide-react";
import { ControlPanel } from "../components/ControlPanel";
import { GraphCanvas } from "../components/graph/GraphCanvas";
import { SelectedNodePanel } from "../components/SelectedNodePanel";
import { StatsPanel } from "../components/StatsPanel";
import { Card, CardContent } from "../components/ui/card";
import type { DependencyGraphNode, DependencyGraph, GraphType, LayoutMode, EdgeStatusFilter } from "../types";
import type { GraphStats } from "../hooks/useGraphData";

interface GraphPageProps {
  graph: DependencyGraph | null;
  graphType: GraphType;
  layout: LayoutMode;
  filter: string;
  driftFilter: EdgeStatusFilter;
  loading: boolean;
  selectedNode: DependencyGraphNode | null;
  apiHealthy: boolean | null;
  stats: GraphStats | null;
  onGraphTypeChange: (type: GraphType) => void;
  onLayoutChange: (layout: LayoutMode) => void;
  onFilterChange: (filter: string) => void;
  onDriftFilterChange: (filter: EdgeStatusFilter) => void;
  onSelectNode: (node: DependencyGraphNode | null) => void;
  onRefresh: () => void;
  onAnalyzeAll: () => void;
  onExport: () => void;
}

export function GraphPage({
  graph,
  graphType,
  layout,
  filter,
  driftFilter,
  loading,
  selectedNode,
  apiHealthy,
  stats,
  onGraphTypeChange,
  onLayoutChange,
  onFilterChange,
  onDriftFilterChange,
  onSelectNode,
  onRefresh,
  onAnalyzeAll,
  onExport,
}: GraphPageProps) {
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

  return (
    <section className="grid flex-1 gap-6 xl:grid-cols-[320px_minmax(0,1fr)_320px]">
      <aside className="space-y-6">
        <ControlPanel
          graphType={graphType}
          layout={layout}
          filter={filter}
          driftFilter={driftFilter}
          onGraphTypeChange={onGraphTypeChange}
          onLayoutChange={onLayoutChange}
          onFilterChange={onFilterChange}
          onDriftFilterChange={onDriftFilterChange}
          onRefresh={onRefresh}
          onAnalyzeAll={onAnalyzeAll}
          onExport={onExport}
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
        {!loading && !graph ? (
          <div className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-4 px-6 text-center">
            <p className="text-base font-semibold text-foreground">No graph loaded yet</p>
            <p className="max-w-xl text-sm text-muted-foreground">
              Kick off a scan to populate the dependency graph. We’ll pull every scenario and resource
              link and then you can filter or export the results.
            </p>
            <div className="flex flex-wrap gap-3">
              <button
                className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90"
                onClick={() => void onAnalyzeAll()}
              >
                Analyze all scenarios
              </button>
              <button
                className="rounded-lg border border-border/60 px-3 py-2 text-xs font-semibold text-foreground hover:bg-background/70"
                onClick={onRefresh}
              >
                Retry loading graph
              </button>
            </div>
          </div>
        ) : null}
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
          onSelectNode={onSelectNode}
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
  );
}
