/**
 * GraphCanvas - React Flow canvas for the API-backed swarm graph.
 *
 * Applies persisted graph settings, deterministic topology grouping, and
 * Dagre layout before rendering.
 */

import { useCallback, useEffect, useMemo, useRef } from "react";
import {
  Background,
  BackgroundVariant,
  MiniMap,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type DefaultEdgeOptions,
  type NodeMouseHandler,
  type NodeTypes,
  type OnEdgesChange,
  type OnNodesChange,
  type ReactFlowInstance,
  type Viewport,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { applyDagreLayout } from "../lib/layout-utils";
import { buildGraphPresentation } from "../lib/graph-presentation";
import {
  FILTER_SUGGESTION_THRESHOLD,
  getEdgeStyle,
  STRAIGHT_EDGE_THRESHOLD,
} from "../lib/edge-styles";
import { bfsNeighborhood } from "../lib/bfs-selection";
import {
  getGraphNodeData,
  type GraphEdge,
  type GraphNode,
} from "../types";
import { ClusterNode } from "./ClusterNode";
import { EdgeLegend } from "./EdgeLegend";
import { GraphNode as GraphNodeComponent } from "./GraphNode";

const nodeTypes: NodeTypes = {
  backlog: GraphNodeComponent,
  scenario: GraphNodeComponent,
  execution: GraphNodeComponent,
  capture: GraphNodeComponent,
  "agent-run": GraphNodeComponent,
  initiative: GraphNodeComponent,
  cluster: ClusterNode,
};

const baseEdgeOptions: DefaultEdgeOptions = {
  style: {
    stroke: "rgb(100 116 139 / 0.5)",
    strokeWidth: 1.5,
  },
  animated: false,
};

export function GraphCanvas() {
  const storeNodes = useGraphDataStore((s) => s.nodes);
  const storeEdges = useGraphDataStore((s) => s.edges);
  const lens = useGraphDataStore((s) => s.lens);
  const meta = useGraphDataStore((s) => s.meta);
  const loading = useGraphDataStore((s) => s.loading);
  const error = useGraphDataStore((s) => s.error);
  const settings = useGraphDataStore((s) => s.settingsByLens[s.lens]);
  const groupingMode = settings.groupingMode;
  const autoFitOnChange = settings.autoFitOnChange;

  const layoutMode = useGraphUIStore((s) => s.layoutMode);
  const layoutDirection = useGraphUIStore((s) => s.layoutDirection);
  const highlightState = useGraphUIStore((s) => s.highlightState);
  const storedViewport = useGraphUIStore((s) => s.viewportByLens[lens]);
  const fitViewNonce = useGraphUIStore((s) => s.fitViewNonce);
  const expandedTopologyClusters = useGraphUIStore((s) => s.expandedTopologyClusters);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const setViewportForLens = useGraphUIStore((s) => s.setViewportForLens);
  const toggleTopologyCluster = useGraphUIStore((s) => s.toggleTopologyCluster);

  const flowRef = useRef<ReactFlowInstance<GraphNode, GraphEdge> | null>(null);

  const { processedNodes, processedEdges, visibleEdgeTypes, visibleNodeCount } = useMemo(() => {
    return buildGraphPresentation({
      lens,
      nodes: storeNodes,
      edges: storeEdges,
      settings,
      expandedTopologyClusters,
    });
  }, [expandedTopologyClusters, lens, settings, storeEdges, storeNodes]);

  const styledEdges = useMemo<GraphEdge[]>(() => {
    const useStraightEdges = processedEdges.length > STRAIGHT_EDGE_THRESHOLD;
    return processedEdges.map((edge) => ({
      ...edge,
      data: {
        ...(edge.data ?? {}),
        relationshipType: edge.type,
      },
      type: useStraightEdges ? "straight" : undefined,
      style: getEdgeStyle(edge.type ?? undefined),
    }));
  }, [processedEdges]);

  const layoutedNodes = useMemo(() => {
    const topLevelNodes = processedNodes.filter((node) => !node.parentId);
    const childNodes = processedNodes.filter((node) => node.parentId);

    if (topLevelNodes.length === 0) {
      return [];
    }

    const topLevelIds = new Set(topLevelNodes.map((node) => node.id));
    const topLevelEdges = styledEdges.filter(
      (edge) => topLevelIds.has(edge.source) && topLevelIds.has(edge.target),
    );

    const positionedTopLevel = applyDagreLayout(
      topLevelNodes,
      topLevelEdges,
      layoutMode,
      layoutDirection,
    );

    const childrenByParent = new Map<string, typeof childNodes>();
    for (const child of childNodes) {
      const parentId = child.parentId;
      if (!parentId) {
        continue;
      }
      const existing = childrenByParent.get(parentId) ?? [];
      existing.push(child);
      childrenByParent.set(parentId, existing);
    }

    const positionedChildren: typeof childNodes = [];
    for (const children of childrenByParent.values()) {
      const childIds = new Set(children.map((child) => child.id));
      const intraEdges = styledEdges.filter(
        (edge) => childIds.has(edge.source) && childIds.has(edge.target),
      );
      positionedChildren.push(
        ...applyDagreLayout(children, intraEdges, layoutMode, layoutDirection),
      );
    }

    return [...positionedTopLevel, ...positionedChildren];
  }, [layoutDirection, layoutMode, processedNodes, styledEdges]);

  const styledNodes = useMemo(() => {
    if (highlightState.mode === "normal") {
      return layoutedNodes;
    }

    return layoutedNodes.map((node) => {
      const isHighlighted = highlightState.highlighted.has(node.id);
      if (highlightState.mode === "hide" && !isHighlighted) {
        return { ...node, hidden: true };
      }
      if (highlightState.mode === "dim" && !isHighlighted) {
        return { ...node, style: { ...node.style, opacity: 0.25 } };
      }
      return node;
    });
  }, [highlightState, layoutedNodes]);

  const [nodes, setNodes, onNodesChange] = useNodesState(styledNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(styledEdges);

  useEffect(() => {
    setNodes(styledNodes);
  }, [setNodes, styledNodes]);

  useEffect(() => {
    setEdges(styledEdges);
  }, [setEdges, styledEdges]);

  const autoFitFingerprint = useMemo(() => {
    return JSON.stringify({
      lens,
      layoutMode,
      layoutDirection,
      groupingMode,
      showSecondaryEdges: settings.showSecondaryEdges,
      nodeIds: styledNodes.map((node) => node.id),
      edgeIds: styledEdges.map((edge) => edge.id),
    });
  }, [groupingMode, lens, layoutDirection, layoutMode, settings.showSecondaryEdges, styledEdges, styledNodes]);

  useEffect(() => {
    if (!autoFitOnChange || !flowRef.current || styledNodes.length === 0) {
      return;
    }

    const raf = window.requestAnimationFrame(() => {
      flowRef.current?.fitView({ padding: 0.2, maxZoom: 1.2 });
    });

    return () => window.cancelAnimationFrame(raf);
  }, [autoFitFingerprint, autoFitOnChange, styledNodes.length]);

  useEffect(() => {
    if (!flowRef.current || fitViewNonce === 0 || styledNodes.length === 0) {
      return;
    }

    const raf = window.requestAnimationFrame(() => {
      flowRef.current?.fitView({ padding: 0.2, maxZoom: 1.2 });
    });

    return () => window.cancelAnimationFrame(raf);
  }, [fitViewNonce, styledNodes.length]);

  const handleNodeClick: NodeMouseHandler = useCallback(
    (_event, node) => {
      const graphNode = node as GraphNode;
      if (node.type === "cluster") {
        toggleTopologyCluster(node.id);
        return;
      }

      selectNode(node.id);
      setHighlightState({
        highlighted: bfsNeighborhood(graphNode.id, processedNodes, styledEdges),
        mode: "dim",
      });
    },
    [processedNodes, selectNode, setHighlightState, styledEdges, toggleTopologyCluster],
  );

  const handlePaneClick = useCallback(() => {
    selectNode(null);
    setHighlightState({ highlighted: new Set(), mode: "normal" });
  }, [selectNode, setHighlightState]);

  const handleMoveEnd = useCallback(
    (_event: MouseEvent | TouchEvent | null, viewport: Viewport) => {
      setViewportForLens(lens, viewport);
    },
    [lens, setViewportForLens],
  );

  const defaultViewport: Viewport = storedViewport ?? { x: 0, y: 0, zoom: 1 };
  const showMiniMap = visibleNodeCount > 20 && visibleNodeCount <= 120;
  const showFilterSuggestion = processedEdges.length > FILTER_SUGGESTION_THRESHOLD;

  return (
    <div className="h-full w-full" data-testid="graph-canvas">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        defaultEdgeOptions={baseEdgeOptions}
        onNodesChange={onNodesChange as OnNodesChange}
        onEdgesChange={onEdgesChange as OnEdgesChange}
        onNodeClick={handleNodeClick}
        onPaneClick={handlePaneClick}
        onMoveEnd={handleMoveEnd}
        onInit={(instance) => {
          flowRef.current = instance as unknown as ReactFlowInstance<GraphNode, GraphEdge>;
        }}
        defaultViewport={defaultViewport}
        fitView={!storedViewport}
        fitViewOptions={{ padding: 0.2, maxZoom: 1.2 }}
        proOptions={{ hideAttribution: true }}
        minZoom={0.1}
        maxZoom={2}
      >
        <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="rgb(51 65 85 / 0.4)" />
        {showMiniMap && (
          <MiniMap
            nodeStrokeWidth={3}
            nodeColor={(node) => {
              const entityType = getGraphNodeData(node).entityType;
              switch (entityType) {
                case "backlog":
                  return "rgb(34 211 238 / 0.6)";
                case "scenario":
                  return "rgb(167 139 250 / 0.6)";
                case "execution":
                  return "rgb(251 191 36 / 0.6)";
                case "capture":
                  return "rgb(52 211 153 / 0.6)";
                case "agent-run":
                  return "rgb(251 113 133 / 0.6)";
                case "initiative":
                  return "rgb(56 189 248 / 0.6)";
                default:
                  return "rgb(148 163 184 / 0.4)";
              }
            }}
            maskColor="rgb(2 6 23 / 0.7)"
            style={{
              backgroundColor: "rgb(15 23 42 / 0.8)",
              borderRadius: 8,
              border: "1px solid rgb(51 65 85 / 0.5)",
            }}
            className="!bottom-3 !right-3"
          />
        )}
      </ReactFlow>

      {visibleEdgeTypes.length > 0 && <EdgeLegend edgeTypes={visibleEdgeTypes} />}

      {loading && (
        <div
          className="pointer-events-none absolute inset-x-0 top-3 z-20 mx-auto w-fit rounded-lg border border-slate-700/80 bg-slate-950/90 px-4 py-2 text-xs text-slate-300 shadow-lg"
          data-testid="graph-loading"
        >
          Refreshing graph…
        </div>
      )}

      {error && (
        <div
          className="absolute left-1/2 top-3 z-20 -translate-x-1/2 rounded-lg border border-red-500/30 bg-red-950/90 px-4 py-2 text-xs text-red-200 shadow-lg"
          data-testid="graph-error"
        >
          {error}
        </div>
      )}

      {!loading && !error && styledNodes.length === 0 && (
        <div
          className="absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-1/2 rounded-xl border border-slate-700/70 bg-slate-950/90 px-5 py-4 text-center shadow-lg"
          data-testid="graph-empty"
        >
          <p className="text-sm font-medium text-slate-100">No nodes match the current graph controls.</p>
          <p className="mt-1 text-xs text-slate-500">Try restoring entity or status visibility.</p>
        </div>
      )}

      {lens === "operations" && meta?.agentManagerAvailable === false && (
        <div
          className="absolute bottom-3 left-3 z-20 rounded-lg border border-amber-500/30 bg-amber-950/90 px-4 py-2 text-xs text-amber-200 shadow-lg"
          data-testid="operations-agent-manager-warning"
        >
          Agent manager is unavailable, so run nodes may be missing from this view.
        </div>
      )}

      {showFilterSuggestion && (
        <div
          className="absolute left-1/2 top-14 z-20 -translate-x-1/2 rounded-lg border border-amber-500/40 bg-amber-950/90 px-4 py-2 text-xs text-amber-200 shadow-lg"
          data-testid="filter-suggestion"
        >
          High edge count ({processedEdges.length}). Use graph controls to filter entity types, statuses, or secondary edges.
        </div>
      )}
    </div>
  );
}
