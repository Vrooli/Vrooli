/**
 * GraphCanvas - React Flow v12 canvas with Dagre layout.
 *
 * Renders nodes and edges from the graph data store,
 * applies layout from the UI store's current layout mode,
 * and persists viewport to localStorage.
 *
 * Topology lens features:
 * - Initiative-based clustering with ClusterNode parent nodes
 * - Edge type visual differentiation (color + dash pattern)
 * - Dynamic MiniMap (show >20, hide >120)
 * - Node capping at ~50 unclustered nodes
 * - Edge complexity management (straight >300, filter suggestion >500)
 */

import { useCallback, useEffect, useMemo, useRef } from "react";
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  MiniMap,
  useNodesState,
  useEdgesState,
  type OnNodesChange,
  type OnEdgesChange,
  type NodeMouseHandler,
  type Viewport,
  type NodeTypes,
  type DefaultEdgeOptions,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { applyDagreLayout } from "../lib/layout-utils";
import { bfsNeighborhood } from "../lib/bfs-selection";
import {
  buildClusterHierarchy,
  aggregateEdgesForCollapsed,
  applyNodeCap,
  UNASSIGNED_CLUSTER_ID,
} from "../lib/clustering-utils";
import { getEdgeStyle, STRAIGHT_EDGE_THRESHOLD, FILTER_SUGGESTION_THRESHOLD } from "../lib/edge-styles";
import { GraphNode } from "./GraphNode";
import { ClusterNode } from "./ClusterNode";
import { EdgeLegend } from "./EdgeLegend";

/**
 * Register custom node components.
 */
const nodeTypes: NodeTypes = {
  backlog: GraphNode,
  scenario: GraphNode,
  execution: GraphNode,
  capture: GraphNode,
  "agent-run": GraphNode,
  initiative: GraphNode,
  cluster: ClusterNode,
};

const baseEdgeOptions: DefaultEdgeOptions = {
  style: {
    stroke: "rgb(100 116 139 / 0.5)",
    strokeWidth: 1.5,
  },
  animated: false,
};

const NODE_CAP_LIMIT = 50;

export function GraphCanvas() {
  const storeNodes = useGraphDataStore((s) => s.nodes);
  const storeEdges = useGraphDataStore((s) => s.edges);
  const entityFilters = useGraphDataStore((s) => s.entityFilters);
  const lens = useGraphDataStore((s) => s.lens);
  const layoutMode = useGraphUIStore((s) => s.layoutMode);
  const highlightState = useGraphUIStore((s) => s.highlightState);
  const storedViewport = useGraphUIStore((s) => s.viewport);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const setViewport = useGraphUIStore((s) => s.setViewport);
  const collapsedClusters = useGraphUIStore((s) => s.collapsedClusters);
  const toggleClusterCollapse = useGraphUIStore((s) => s.toggleClusterCollapse);
  const setAllClustersCollapsed = useGraphUIStore((s) => s.setAllClustersCollapsed);

  const isTopology = lens === "topology";
  const initializedClusters = useRef(false);

  // Filter nodes by entity type visibility.
  const filteredNodes = useMemo(() => {
    return storeNodes.filter((node) => {
      const entityType = node.data?.entityType as string | undefined;
      if (!entityType) return true;
      return entityFilters[entityType as keyof typeof entityFilters] ?? true;
    });
  }, [storeNodes, entityFilters]);

  // Filter edges to only include those connecting visible nodes.
  const filteredEdges = useMemo(() => {
    const visibleIds = new Set(filteredNodes.map((n) => n.id));
    return storeEdges.filter((e) => visibleIds.has(e.source) && visibleIds.has(e.target));
  }, [storeEdges, filteredNodes]);

  // Topology: Build cluster hierarchy and transform nodes/edges.
  const { processedNodes, processedEdges, clusters } = useMemo(() => {
    if (!isTopology) {
      return { processedNodes: filteredNodes, processedEdges: filteredEdges, clusters: [] };
    }

    const { clusters: builtClusters, unclustered } = buildClusterHierarchy(filteredNodes, filteredEdges);

    // Build cluster parent nodes and child nodes with parentId
    const clusterNodes: typeof filteredNodes = [];
    const childNodes: typeof filteredNodes = [];
    const nodeMap = new Map(filteredNodes.map((n) => [n.id, n]));

    for (const cluster of builtClusters) {
      const isCollapsed = collapsedClusters.has(cluster.id);
      const isUnassigned = cluster.id === UNASSIGNED_CLUSTER_ID;

      // Cluster parent node
      clusterNodes.push({
        id: cluster.id,
        type: "cluster",
        position: { x: 0, y: 0 },
        data: {
          label: cluster.label,
          collapsed: isCollapsed,
          rollup: cluster.rollup,
          isUnassigned,
          entityType: "initiative",
        },
        style: isCollapsed ? undefined : { padding: 20 },
      });

      // Member nodes as children (only when expanded)
      if (!isCollapsed) {
        for (const memberId of cluster.members) {
          const memberNode = nodeMap.get(memberId);
          if (memberNode) {
            childNodes.push({
              ...memberNode,
              parentId: cluster.id,
              extent: "parent" as const,
              position: { x: 0, y: 40 }, // Relative to parent; Dagre will reposition
            });
          }
        }
      }
    }

    // Apply node cap to unclustered nodes
    const { visible: cappedUnclustered } = applyNodeCap(unclustered, NODE_CAP_LIMIT);

    const allNodes = [...clusterNodes, ...childNodes, ...cappedUnclustered];

    // Aggregate edges for collapsed clusters
    const aggregatedEdges = aggregateEdgesForCollapsed(filteredEdges, collapsedClusters, builtClusters);

    return { processedNodes: allNodes, processedEdges: aggregatedEdges, clusters: builtClusters };
  }, [filteredNodes, filteredEdges, isTopology, collapsedClusters]);

  // Initialize all clusters as collapsed on first topology data load.
  useEffect(() => {
    if (!isTopology || initializedClusters.current || clusters.length === 0) return;
    initializedClusters.current = true;
    setAllClustersCollapsed(clusters.map((c) => c.id));
  }, [isTopology, clusters, setAllClustersCollapsed]);

  // Reset cluster initialization when switching away from topology.
  useEffect(() => {
    if (!isTopology) {
      initializedClusters.current = false;
    }
  }, [isTopology]);

  // Apply edge styles for topology lens.
  const styledEdges = useMemo(() => {
    if (!isTopology) return processedEdges;

    const useStraight = processedEdges.length > STRAIGHT_EDGE_THRESHOLD;

    return processedEdges.map((edge) => ({
      ...edge,
      type: useStraight ? "straight" : undefined,
      style: getEdgeStyle(edge.type ?? undefined),
    }));
  }, [processedEdges, isTopology]);

  // Apply Dagre layout.
  const layoutedNodes = useMemo(() => {
    // For topology with clusters, only layout non-child nodes at the top level.
    // Child nodes are positioned relative to their parent.
    const topLevelNodes = processedNodes.filter((n) => !n.parentId);
    const childNodes = processedNodes.filter((n) => n.parentId);

    if (topLevelNodes.length === 0) return [];

    // Top-level edges (only between top-level nodes)
    const topLevelIds = new Set(topLevelNodes.map((n) => n.id));
    const topLevelEdges = styledEdges.filter(
      (e) => topLevelIds.has(e.source) && topLevelIds.has(e.target),
    );

    const positionedTopLevel = applyDagreLayout(topLevelNodes, topLevelEdges, layoutMode);

    // Position children within their parent cluster
    const parentGroups = new Map<string, typeof childNodes>();
    for (const child of childNodes) {
      const group = parentGroups.get(child.parentId!) ?? [];
      group.push(child);
      parentGroups.set(child.parentId!, group);
    }

    const positionedChildren: typeof childNodes = [];
    for (const [, children] of parentGroups) {
      const intraEdges = styledEdges.filter(
        (e) => children.some((c) => c.id === e.source) && children.some((c) => c.id === e.target),
      );
      const positioned = applyDagreLayout(children, intraEdges, layoutMode);
      positionedChildren.push(...positioned);
    }

    return [...positionedTopLevel, ...positionedChildren];
  }, [processedNodes, styledEdges, layoutMode]);

  // Apply highlight/dim styling.
  const styledNodes = useMemo(() => {
    if (highlightState.mode === "normal") return layoutedNodes;

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
  }, [layoutedNodes, highlightState]);

  const [nodes, setNodes, onNodesChange] = useNodesState(styledNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(styledEdges);

  // Sync store changes to local state.
  useEffect(() => {
    setNodes(styledNodes);
  }, [styledNodes, setNodes]);

  useEffect(() => {
    setEdges(styledEdges);
  }, [styledEdges, setEdges]);

  const handleNodeClick: NodeMouseHandler = useCallback(
    (_event, node) => {
      // If clicking a collapsed cluster node, expand it
      if (node.type === "cluster" && collapsedClusters.has(node.id)) {
        toggleClusterCollapse(node.id);
        return;
      }

      selectNode(node.id);

      // BFS neighborhood highlight.
      const neighborhood = bfsNeighborhood(node.id, processedNodes, styledEdges);
      setHighlightState({
        highlighted: neighborhood,
        mode: "dim",
      });
    },
    [selectNode, setHighlightState, processedNodes, styledEdges, collapsedClusters, toggleClusterCollapse],
  );

  const handleNodeDoubleClick: NodeMouseHandler = useCallback(
    (_event, node) => {
      // Double-click on cluster: select cluster as unit, BFS highlights all edges to/from any member
      if (node.type === "cluster") {
        selectNode(node.id);
        const cluster = clusters.find((c) => c.id === node.id);
        if (cluster) {
          const highlighted = new Set<string>([node.id, ...cluster.members]);
          // Add all nodes connected to any member
          for (const edge of styledEdges) {
            if (cluster.members.includes(edge.source) || edge.source === node.id) {
              highlighted.add(edge.target);
            }
            if (cluster.members.includes(edge.target) || edge.target === node.id) {
              highlighted.add(edge.source);
            }
          }
          setHighlightState({ highlighted, mode: "dim" });
        }
      }
    },
    [selectNode, setHighlightState, clusters, styledEdges],
  );

  const handlePaneClick = useCallback(() => {
    selectNode(null);
    setHighlightState({ highlighted: new Set(), mode: "normal" });
  }, [selectNode, setHighlightState]);

  const handleMoveEnd = useCallback(
    (_event: MouseEvent | TouchEvent | null, viewport: Viewport) => {
      setViewport(viewport);
    },
    [setViewport],
  );

  const defaultViewport: Viewport = storedViewport ?? { x: 0, y: 0, zoom: 1 };

  // MiniMap: show when >20 nodes, hide when >120.
  const visibleNodeCount = processedNodes.filter((n) => !n.parentId || !collapsedClusters.has(n.parentId)).length;
  const showMiniMap = visibleNodeCount > 20 && visibleNodeCount <= 120;

  // Edge complexity: filter suggestion banner.
  const showFilterSuggestion = isTopology && styledEdges.length > FILTER_SUGGESTION_THRESHOLD;

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
        onNodeDoubleClick={handleNodeDoubleClick}
        onPaneClick={handlePaneClick}
        onMoveEnd={handleMoveEnd}
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
              const et = (node.data as Record<string, unknown>)?.entityType as string | undefined;
              switch (et) {
                case "backlog": return "rgb(34 211 238 / 0.6)";
                case "scenario": return "rgb(167 139 250 / 0.6)";
                case "execution": return "rgb(251 191 36 / 0.6)";
                case "capture": return "rgb(52 211 153 / 0.6)";
                case "agent-run": return "rgb(251 113 133 / 0.6)";
                case "initiative": return "rgb(56 189 248 / 0.6)";
                default: return "rgb(148 163 184 / 0.4)";
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

      {/* Topology-specific overlays */}
      {isTopology && <EdgeLegend />}

      {showFilterSuggestion && (
        <div
          className="absolute left-1/2 top-3 z-20 -translate-x-1/2 rounded-lg border border-amber-500/40 bg-amber-950/90 px-4 py-2 text-xs text-amber-200 shadow-lg"
          data-testid="filter-suggestion"
        >
          High edge count ({styledEdges.length}). Consider filtering entity types to reduce visual complexity.
        </div>
      )}
    </div>
  );
}
