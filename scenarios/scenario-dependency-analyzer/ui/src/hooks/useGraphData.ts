import { buildApiUrl } from "@vrooli/api-base";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getApiBaseUrl, prettyNumber } from "../lib/utils";
import type {
  DependencyGraph,
  DependencyGraphNode,
  GraphType,
  LayoutMode,
  EdgeStatusFilter
} from "../types";

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
  const [graph, setGraph] = useState<DependencyGraph | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<DependencyGraphNode | null>(null);
  const [apiHealthy, setApiHealthy] = useState<boolean | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const apiBase = useMemo(() => getApiBaseUrl(), []);

  const fetchGraph = useCallback(
    async (type: GraphType = graphType) => {
      if (abortRef.current) {
        abortRef.current.abort();
      }
      const controller = new AbortController();
      abortRef.current = controller;
      setLoading(true);
      setError(null);
      try {
        const url = buildApiUrl(`/graph/${type}`, { baseUrl: apiBase });
        const response = await fetch(url, {
          signal: controller.signal
        });
        if (!response.ok) {
          throw new Error(`Graph request failed (${response.status})`);
        }
        const data = (await response.json()) as DependencyGraph;
        setGraph(data);
        setSelectedNode(null);
        setGraphType(type);
      } catch (err) {
        if ((err as Error).name === "AbortError") {
          return;
        }
        setError((err as Error).message);
      } finally {
        setLoading(false);
      }
    },
    [apiBase, graphType]
  );

  const checkHealth = useCallback(async () => {
    try {
      const response = await fetch(buildApiUrl("/health/analysis", { baseUrl: apiBase }));
      if (!response.ok) {
        throw new Error("Health check failed");
      }
      const payload = (await response.json()) as { status: string };
      setApiHealthy(payload.status === "ok" || payload.status === "healthy");
    } catch (err) {
      console.error("API health check failed", err);
      setApiHealthy(false);
    }
  }, [apiBase]);

  const analyzeAll = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetch(buildApiUrl("/analyze/all", { baseUrl: apiBase }));
      if (!response.ok) {
        throw new Error(`Analyze all failed (${response.status})`);
      }
      await fetchGraph(graphType);
      return true;
    } catch (err) {
      setError((err as Error).message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [apiBase, fetchGraph, graphType]);

  useEffect(() => {
    fetchGraph(defaultType).catch((err) => {
      console.error(err);
    });
    checkHealth();
    const interval = setInterval(checkHealth, 30000);
    return () => {
      clearInterval(interval);
      abortRef.current?.abort();
    };
  }, [checkHealth, defaultType, fetchGraph]);

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
    setGraphType: (type: GraphType) => fetchGraph(type),
    setLayout,
    setSelectedNode,
    stats
  };
}
