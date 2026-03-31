/**
 * GraphCanvas - React Flow canvas for the API-backed swarm graph.
 *
 * Applies persisted graph settings, deterministic topology grouping, and
 * Dagre layout before rendering.
 */

import { memo, useCallback, useEffect, useMemo, useRef } from "react";
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
import { useDetailSelectionStore } from "../../../stores/detail-selection-store";
import { parseNodeId } from "../lib/node-id-parser";
import { applyDagreLayout } from "../lib/layout-utils";
import { buildGraphPresentation } from "../lib/graph-presentation";
import {
  FILTER_SUGGESTION_THRESHOLD,
  getEdgeMarker,
  getEdgeStyle,
  STRAIGHT_EDGE_THRESHOLD,
} from "../lib/edge-styles";
import { getStatusRgb } from "../lib/status-colors";
import { hasDetailPage } from "../lib/detail-page-registry";
import { computeVisualFocus, clearVisualFocus } from "../lib/visual-focus";
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
    strokeWidth: 2.5,
  },
  animated: false,
};

// PERF: Memoized because GraphCanvas takes no props — it reads all state
// from Zustand stores. Without memo, every GraphWorkspace re-render (e.g.,
// from the 5-second activity polling) would cascade into GraphCanvas,
// re-evaluating all its useMemo/useCallback hooks unnecessarily.
export const GraphCanvas = memo(function GraphCanvas() {
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
  const fitViewNonce = useGraphUIStore((s) => s.fitViewNonce);
  const expandedTopologyClusters = useGraphUIStore((s) => s.expandedTopologyClusters);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const setViewportForLens = useGraphUIStore((s) => s.setViewportForLens);
  const toggleTopologyCluster = useGraphUIStore((s) => s.toggleTopologyCluster);

  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const focusNodeId = useGraphDataStore((s) => s.focusNodeId);

  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);
  const selectScenario = useDetailSelectionStore((s) => s.selectScenario);
  const selectExecution = useDetailSelectionStore((s) => s.selectExecution);
  const selectInitiative = useDetailSelectionStore((s) => s.selectInitiative);

  const flowRef = useRef<ReactFlowInstance<GraphNode, GraphEdge> | null>(null);

  // Track whether the initial data load has completed for the current lens.
  // Used to suppress autoFitOnChange during the first data arrival when a
  // stored viewport exists (otherwise fitView overrides the restored viewport).
  const initialLoadCompleteRef = useRef(false);
  // Track which lens the initialLoadComplete flag is for, so lens switches reset it.
  const initialLoadLensRef = useRef(lens);

  const { processedNodes, processedEdges, visibleEdgeTypes } = useMemo(() => {
    return buildGraphPresentation({
      lens,
      nodes: storeNodes,
      edges: storeEdges,
      settings,
      expandedTopologyClusters,
    });
  }, [expandedTopologyClusters, lens, settings, storeEdges, storeNodes]);

  // PERF: Split edge styling into two layers:
  // 1. Base styling (type-based colors, markers) — only changes when edges change.
  // 2. Highlight overlay (opacity) — only changes when highlight state changes.
  // This prevents recomputing base styles when only the highlight changes.
  const baseStyledEdges = useMemo<GraphEdge[]>(() => {
    const useStraightEdges = processedEdges.length > STRAIGHT_EDGE_THRESHOLD;
    return processedEdges.map((edge) => ({
      ...edge,
      data: {
        ...(edge.data ?? {}),
        relationshipType: edge.type,
      },
      type: useStraightEdges ? "straight" : undefined,
      style: getEdgeStyle(edge.type ?? undefined),
      markerEnd: getEdgeMarker(edge.type ?? undefined),
    }));
  }, [processedEdges]);

  // In normal mode (common during pan/zoom), skip the highlight pass entirely.
  const styledEdges = useMemo<GraphEdge[]>(() => {
    if (highlightState.mode === "normal") {
      return baseStyledEdges;
    }

    return baseStyledEdges.map((edge) => {
      const srcHighlighted = highlightState.highlighted.has(edge.source);
      const tgtHighlighted = highlightState.highlighted.has(edge.target);
      const bothHighlighted = srcHighlighted && tgtHighlighted;
      if (highlightState.mode === "dim" && !bothHighlighted) {
        return { ...edge, style: { ...edge.style, opacity: 0.15, transition: "opacity 0.2s ease" } };
      }
      if (highlightState.mode === "hide" && !bothHighlighted) {
        return { ...edge, style: { ...edge.style, opacity: 0, transition: "opacity 0.2s ease" } };
      }
      return { ...edge, style: { ...edge.style, transition: "opacity 0.2s ease" } };
    });
  }, [baseStyledEdges, highlightState]);

  // PERF: Layout depends on processedEdges (connectivity only), NOT styledEdges.
  // Dagre only needs source/target to compute positions. Using styledEdges here
  // would cause a full Dagre re-layout on every highlight change (styledEdges
  // get new object refs when highlight state changes), which is very expensive.
  const layoutedNodes = useMemo(() => {
    const topLevelNodes = processedNodes.filter((node) => !node.parentId);
    const childNodes = processedNodes.filter((node) => node.parentId);

    if (topLevelNodes.length === 0) {
      return [];
    }

    const topLevelIds = new Set(topLevelNodes.map((node) => node.id));
    const topLevelEdges = processedEdges.filter(
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
      const intraEdges = processedEdges.filter(
        (edge) => childIds.has(edge.source) && childIds.has(edge.target),
      );
      positionedChildren.push(
        ...applyDagreLayout(children, intraEdges, layoutMode, layoutDirection),
      );
    }

    return [...positionedTopLevel, ...positionedChildren];
  }, [layoutDirection, layoutMode, processedEdges, processedNodes]);

  // PERF: In "normal" mode (no highlight active — the common case during
  // pan/zoom), return layoutedNodes directly without creating new objects.
  // This preserves referential identity so React Flow's internal diffing
  // can skip re-rendering nodes that haven't changed.
  const styledNodes = useMemo(() => {
    if (highlightState.mode === "normal") {
      return layoutedNodes;
    }

    const transition = "opacity 0.2s ease";
    return layoutedNodes.map((node) => {
      const isHighlighted = highlightState.highlighted.has(node.id);
      if (highlightState.mode === "hide" && !isHighlighted) {
        return { ...node, hidden: true };
      }
      if (highlightState.mode === "dim" && !isHighlighted) {
        return { ...node, style: { ...node.style, opacity: 0.5, transition } };
      }
      return { ...node, style: { ...node.style, opacity: 1, transition } };
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

  // Reset initial-load tracking when the lens changes.
  if (lens !== initialLoadLensRef.current) {
    initialLoadCompleteRef.current = false;
    initialLoadLensRef.current = lens;
  }

  // PERF: Use a cheap fingerprint instead of JSON.stringify of all IDs.
  // The fingerprint detects structural changes (nodes/edges added/removed,
  // layout/lens changes) without serializing every ID on every render.
  // We hash the sorted IDs into a single string using join, which is much
  // cheaper than JSON.stringify of the full arrays.
  const nodeFingerprint = useMemo(
    () => processedNodes.map((n) => n.id).join("\0"),
    [processedNodes],
  );
  const edgeFingerprint = useMemo(
    () => processedEdges.map((e) => e.id).join("\0"),
    [processedEdges],
  );
  const autoFitFingerprint = `${lens}|${layoutMode}|${layoutDirection}|${groupingMode}|${settings.showSecondaryEdges}|${nodeFingerprint}|${edgeFingerprint}`;

  useEffect(() => {
    if (!autoFitOnChange || !flowRef.current || styledNodes.length === 0) {
      return;
    }

    // On the first data arrival after mount or lens switch, skip fitView if
    // a stored viewport exists. The stored viewport was already applied via
    // ReactFlow's defaultViewport prop, and fitView would override it.
    //
    // IMPORTANT: We read the viewport from the store snapshot (getState)
    // rather than using the reactive `storedViewport` selector, because
    // every pan/zoom updates the stored viewport. If it were a dependency,
    // this effect would re-fire on every interaction and call fitView,
    // snapping the camera back.
    if (!initialLoadCompleteRef.current) {
      initialLoadCompleteRef.current = true;
      const currentLens = useGraphDataStore.getState().lens;
      const savedViewport = useGraphUIStore.getState().viewportByLens[currentLens];
      if (savedViewport) {
        return;
      }
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

  // Restore visual focus when graph data arrives and a selection or focus node
  // exists. This handles three scenarios:
  // 1. Page refresh: selectedNodeId is restored from URL, but highlight state
  //    is in-memory only and lost. Recompute BFS + dim when nodes arrive.
  // 2. Lens drill: focusNodeId is set by drillToLens, but no visual focus
  //    was applied. When the new lens's data loads, apply focus to that node.
  // 3. Normal operation: if highlight is already applied (mode !== "normal"),
  //    skip — the user already has the correct visual state.
  const focusRestoredRef = useRef(false);
  useEffect(() => {
    // Only act when nodes are available and highlight isn't already applied.
    if (processedNodes.length === 0 || highlightState.mode !== "normal") {
      return;
    }

    // Prefer focusNodeId (from lens drill) over selectedNodeId (from URL restore).
    const targetNodeId = focusNodeId ?? selectedNodeId;
    if (!targetNodeId) {
      focusRestoredRef.current = false;
      return;
    }

    // Prevent re-triggering after we've already restored focus for this node.
    if (focusRestoredRef.current) return;
    focusRestoredRef.current = true;

    const focus = computeVisualFocus(targetNodeId, processedNodes, styledEdges);
    if (focus) {
      selectNode(focus.selectedNodeId);
      setHighlightState(focus.highlightState);
    }
  }, [focusNodeId, highlightState.mode, processedNodes, selectedNodeId, selectNode, setHighlightState, styledEdges]);

  // Reset the focus-restored flag when the target node changes, so the
  // effect can fire again for a new selection/focus.
  useEffect(() => {
    focusRestoredRef.current = false;
  }, [focusNodeId, selectedNodeId]);

  const handleNodeClick: NodeMouseHandler = useCallback(
    (_event, node) => {
      // Cluster nodes toggle on single-click instead of opening inspector.
      if (node.type === "cluster") {
        toggleTopologyCluster(node.id);

        // Cluster nodes backed by an initiative open its detail page.
        const data = getGraphNodeData(node as GraphNode);
        if ("isUnassigned" in data && !data.isUnassigned) {
          const name = node.id.replace(/^initiative\//, "");
          if (name) selectInitiative(name);
        }
        return;
      }

      const parsed = parseNodeId(node.id);
      const opensDetailPage = parsed !== null && hasDetailPage(parsed.entityType);

      if (parsed && opensDetailPage) {
        // Entity has a detail page: the overlay covers the entire graph,
        // so dim/highlight would be invisible and leave stale state on close.
        // Clear any existing visual focus before opening the detail page.
        const cleared = clearVisualFocus();
        selectNode(cleared.selectedNodeId);
        setHighlightState(cleared.highlightState);

        // Open the detail page.
        switch (parsed.entityType) {
          case "backlog":
            if (parsed.kind && parsed.name) selectBacklog(parsed.kind, parsed.name);
            break;
          case "scenario":
            if (parsed.name) selectScenario(parsed.name);
            break;
          case "execution":
            selectExecution(parsed.identifier);
            break;
          case "initiative":
            if (parsed.name) selectInitiative(parsed.name);
            break;
        }
      } else {
        // Entity has no detail page (capture, agent-activity, agent-run)
        // or unrecognized node ID: apply visual focus so the user sees
        // the node highlighted in the graph.
        const focus = computeVisualFocus(node.id, processedNodes, styledEdges);
        if (focus) {
          selectNode(focus.selectedNodeId);
          setHighlightState(focus.highlightState);
        }
      }
    },
    [processedNodes, selectBacklog, selectExecution, selectInitiative, selectNode, selectScenario, setHighlightState, styledEdges, toggleTopologyCluster],
  );

  const handlePaneClick = useCallback(() => {
    const cleared = clearVisualFocus();
    selectNode(cleared.selectedNodeId);
    setHighlightState(cleared.highlightState);
  }, [selectNode, setHighlightState]);

  const handleMoveEnd = useCallback(
    (_event: MouseEvent | TouchEvent | null, viewport: Viewport) => {
      setViewportForLens(lens, viewport);
    },
    [lens, setViewportForLens],
  );

  const handleInit = useCallback(
    (instance: ReactFlowInstance<GraphNode, GraphEdge>) => {
      flowRef.current = instance;
    },
    [],
  );

  // PERF: Read the initial viewport once at mount time, not reactively.
  // The storedViewport changes on every pan/zoom (via setViewportForLens),
  // but defaultViewport is only used by React Flow on initial mount.
  // Using a reactive selector would cause the entire component to re-render
  // on every viewport change even though React Flow ignores the prop after mount.
  const initialViewportRef = useRef(useGraphUIStore.getState().viewportByLens[lens]);
  const defaultViewport: Viewport = initialViewportRef.current ?? { x: 0, y: 0, zoom: 1 };
  const hasStoredViewport = initialViewportRef.current !== null;
  const showMiniMap = settings.showMiniMap;
  const showFilterSuggestion = processedEdges.length > FILTER_SUGGESTION_THRESHOLD;

  // PERF: Stable callback reference so MiniMap doesn't re-render on every
  // GraphCanvas render. Without this, the inline arrow function creates a
  // new reference each render, causing MiniMap to redraw all node colors.
  const miniMapNodeColor = useCallback(
    (node: { data?: unknown }) => getStatusRgb(getGraphNodeData(node).status),
    [],
  );

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
        onInit={handleInit}
        defaultViewport={defaultViewport}
        fitView={!hasStoredViewport}
        fitViewOptions={{ padding: 0.2, maxZoom: 1.2 }}
        proOptions={{ hideAttribution: true }}
        minZoom={0.1}
        maxZoom={2}
        /* Performance: only render nodes/edges in viewport */
        onlyRenderVisibleElements
        /* Touch: prevent nodes from capturing pinch-to-zoom gestures.
           nodeDragThreshold requires a 5px move before drag starts,
           letting the browser recognize pinch/zoom first. */
        nodesDraggable={false}
        nodeDragThreshold={5}
        /* Disable unnecessary features for read-only graph */
        nodesConnectable={false}
        elementsSelectable={false}
      >
        <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="rgb(51 65 85 / 0.4)" />
        {showMiniMap && (
          <MiniMap
            style={{
              width: 140,
              height: 100,
              backgroundColor: "rgb(15 23 42 / 0.8)",
              borderRadius: 8,
              border: "1px solid rgb(51 65 85 / 0.5)",
            }}
            nodeStrokeWidth={2}
            nodeColor={miniMapNodeColor}
            maskColor="rgb(2 6 23 / 0.7)"
            className="!bottom-3 !right-3"
          />
        )}
      </ReactFlow>

      {visibleEdgeTypes.length > 0 && <EdgeLegend edgeTypes={visibleEdgeTypes} />}

      {loading && (
        <div
          className="pointer-events-none absolute inset-x-0 top-14 z-20 mx-auto w-fit rounded-lg border border-slate-700/80 bg-slate-950/90 px-4 py-2 text-xs text-slate-300 shadow-lg"
          data-testid="graph-loading"
        >
          Refreshing graph…
        </div>
      )}

      {error && (
        <div
          className="absolute left-1/2 top-14 z-20 -translate-x-1/2 rounded-lg border border-red-500/30 bg-red-950/90 px-4 py-2 text-xs text-red-200 shadow-lg"
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
          className="absolute left-1/2 top-24 z-20 -translate-x-1/2 rounded-lg border border-amber-500/40 bg-amber-950/90 px-4 py-2 text-xs text-amber-200 shadow-lg"
          data-testid="filter-suggestion"
        >
          High edge count ({processedEdges.length}). Use graph controls to filter entity types, statuses, or secondary edges.
        </div>
      )}
    </div>
  );
});
