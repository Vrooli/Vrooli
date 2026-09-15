import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { analyzeAllScenarios, fetchAnalysisHealth, fetchDependencyGraph } from "../../api/client";
import { prettyNumber } from "../../lib/utils";
import type {
  DependencyGraphNode,
  GraphType,
  LayoutMode,
  EdgeStatusFilter
} from "../../types";

interface UseGraphDataProps {
  defaultType?: GraphType;
  defaultLayout?: LayoutMode;
}

export interface GraphStats {
  totalNodes: number;
  totalEdges: number;
  scenarioCount: number;
  resourceCount: number;
  complexityScore: string;
}

export function useGraphData({
  defaultType = "combined",
  defaultLayout = "force"
}: UseGraphDataProps = {}) {
  const [graphType, setGraphType] = useState<GraphType>(defaultType);
  const [layout, setLayout] = useState<LayoutMode>(defaultLayout);
  const [filter, setFilter] = useState("");
  const [driftFilter, setDriftFilter] = useState<EdgeStatusFilter>("all");
  const [selectedNode, setSelectedNode] = useState<DependencyGraphNode | null>(null);
  const queryClient = useQueryClient();

  const graphQuery = useQuery({
    queryKey: ["dependency-graph", graphType],
    queryFn: ({ signal }) => fetchDependencyGraph(graphType, signal),
    refetchOnWindowFocus: false
  });

  const healthQuery = useQuery({
    queryKey: ["analysis-health"],
    queryFn: fetchAnalysisHealth,
    refetchInterval: 30000,
    refetchOnWindowFocus: false
  });

  const fetchGraph = useCallback(
    async (type: GraphType = graphType) => {
      setSelectedNode(null);
      setGraphType(type);
      await queryClient.fetchQuery({
        queryKey: ["dependency-graph", type],
        queryFn: ({ signal }) => fetchDependencyGraph(type, signal)
      });
    },
    [graphType, queryClient]
  );

  const analyzeAll = useCallback(async () => {
    try {
      await analyzeAllScenarios();
      await fetchGraph(graphType);
      return true;
    } catch (err) {
      console.error("Analyze all failed", err);
      return false;
    }
  }, [fetchGraph, graphType]);

  useEffect(() => {
    setGraphType(defaultType);
  }, [defaultType]);

  const graph = graphQuery.data ?? null;
  const loading = graphQuery.isLoading || graphQuery.isFetching;
  const error = graphQuery.error instanceof Error ? graphQuery.error.message : null;
  const apiHealthy = healthQuery.data
    ? healthQuery.data.status === "ok" || healthQuery.data.status === "healthy"
    : healthQuery.isError
      ? false
      : null;

  const stats: GraphStats | null = useMemo(() => {
    if (!graph) return null;
    const scenarioCount = graph.nodes.filter((n) => n.type === "scenario").length;
    const resourceCount = graph.nodes.filter((n) => n.type === "resource").length;
    return {
      totalNodes: graph.nodes.length,
      totalEdges: graph.edges.length,
      scenarioCount,
      resourceCount,
      complexityScore: prettyNumber(
        typeof graph.metadata?.complexity_score === "number"
          ? graph.metadata.complexity_score
          : 0
      )
    };
  }, [graph]);

  return {
    apiHealthy,
    analyzeAll,
    error,
    fetchGraph,
    filter,
    graph,
    graphType,
    layout,
    loading,
    selectedNode,
    setFilter,
    driftFilter,
    setDriftFilter,
    setGraphType: fetchGraph,
    setLayout,
    setSelectedNode,
    stats
  };
}
