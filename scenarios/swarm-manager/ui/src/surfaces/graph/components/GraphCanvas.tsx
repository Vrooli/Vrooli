/**
 * GraphCanvas - React Flow canvas for the API-backed swarm graph.
 *
 * Applies persisted graph settings, deterministic topology grouping, and
 * Dagre layout before rendering.
 */

import { Profiler, memo, useCallback, useEffect, useMemo, useRef } from "react";
import { onProfilerRender } from "../../../lib/profiler";
import { useGraphAutoFit } from "../hooks/useGraphAutoFit";
import {
  Background,
  BackgroundVariant,
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
import { useGraphSettingsStore } from "../stores/graph-settings-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { applyDagreLayout } from "../lib/layout-utils";
import { buildGraphPresentation } from "../lib/graph-presentation";
import { getEdgeMarker, getEdgeStyle, STRAIGHT_EDGE_THRESHOLD } from "../lib/edge-styles";
import { GRAPH_ENTITY_TYPES } from "../lib/entity-shapes";
import { computeVisualFocus, clearVisualFocus } from "../lib/visual-focus";
import { getGraphNodeData, type GraphEdge, type GraphNode } from "../types";
import { nodeIdToGoalRef, useGoalMembershipIndex } from "../hooks/useGoalMembership";
import { EdgeLegend } from "./EdgeLegend";
import { FocusEmptyState } from "./FocusEmptyState";
import { GraphNode as GraphNodeComponent } from "./GraphNode";

/** Derived from ENTITY_REGISTRY — adding a new entity type automatically registers it here. */
const nodeTypes: NodeTypes = {
  ...Object.fromEntries(GRAPH_ENTITY_TYPES.map((t) => [t, GraphNodeComponent])),
} as NodeTypes;

const baseEdgeOptions: DefaultEdgeOptions = {
  style: {
    stroke: "rgb(100 116 139 / 0.5)",
    strokeWidth: 2.5,
  },
  animated: false,
};

const VIEWPORT_RENDERING_THRESHOLD = STRAIGHT_EDGE_THRESHOLD;

// PERF: Memoized because GraphCanvas takes no props — it reads all state
// from Zustand stores. Without memo, every GraphWorkspace re-render (e.g.,
// from the 5-second activity polling) would cascade into GraphCanvas,
// re-evaluating all its useMemo/useCallback hooks unnecessarily.
const GraphCanvasImpl = memo(function GraphCanvasImpl() {
  const storeNodes = useGraphDataStore((s) => s.nodes);
  const storeEdges = useGraphDataStore((s) => s.edges);
  const lens = useGraphDataStore((s) => s.lens);
  const loading = useGraphDataStore((s) => s.loading);
  const error = useGraphDataStore((s) => s.error);
  const settings = useGraphSettingsStore((s) => s.settingsByLens[s.activeLens]);
  const autoFitOnChange = settings.autoFitOnChange;

  const layoutMode = useGraphUIStore((s) => s.layoutMode);
  const layoutDirection = useGraphUIStore((s) => s.layoutDirection);
  const highlightState = useGraphUIStore((s) => s.highlightState);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const setViewportIntentForLens = useGraphUIStore((s) => s.setViewportIntentForLens);

  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const focusNodeId = useGraphDataStore((s) => s.focusNodeId);
  const goalMembershipIndex = useGoalMembershipIndex();

  const flowRef = useRef<ReactFlowInstance<GraphNode> | null>(null);

  const { processedNodes, processedEdges, visibleEdgeTypes } = useMemo(() => {
    return buildGraphPresentation({
      lens,
      nodes: storeNodes,
      edges: storeEdges,
      settings,
    });
  }, [lens, settings, storeEdges, storeNodes]);

  // PERF: Split edge styling into two layers:
  // 1. Base styling (type-based colors, markers) — only changes when edges change.
  // 2. Highlight overlay (opacity) — only changes when highlight state changes.
  // This prevents recomputing base styles when only the highlight changes.
  const baseStyledEdges = useMemo<GraphEdge[]>(() => {
    const useStraightEdges = processedEdges.length >= STRAIGHT_EDGE_THRESHOLD;
    return processedEdges.map((edge) => ({
      ...edge,
      type: useStraightEdges ? "straight" : edge.type,
      data: {
        ...(edge.data ?? {}),
        relationshipType: edge.type,
      },
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

  const nodesWithGoalBadges = useMemo<GraphNode[]>(() => {
    if (goalMembershipIndex.size === 0) {
      return layoutedNodes;
    }

    let changed = false;
    const next = layoutedNodes.map((node) => {
      const ref = nodeIdToGoalRef(node.id);
      const goalBadges = ref ? goalMembershipIndex.get(ref) : undefined;
      if (!goalBadges || goalBadges.length === 0) {
        return node;
      }

      const data = getGraphNodeData(node);
      if (data.goalBadges === goalBadges) {
        return node;
      }

      changed = true;
      return {
        ...node,
        data: {
          ...data,
          goalBadges,
        },
      };
    });

    return changed ? next : layoutedNodes;
  }, [goalMembershipIndex, layoutedNodes]);

  // PERF: In "normal" mode (no highlight active — the common case during
  // pan/zoom), return the graph-level badge projection directly without
  // creating highlight wrappers. This preserves referential identity so React
  // Flow's internal diffing can skip nodes that have not changed.
  const styledNodes = useMemo(() => {
    if (highlightState.mode === "normal") {
      return nodesWithGoalBadges;
    }

    const transition = "opacity 0.2s ease";
    return nodesWithGoalBadges.map((node) => {
      const isHighlighted = highlightState.highlighted.has(node.id);
      if (highlightState.mode === "hide" && !isHighlighted) {
        return { ...node, hidden: true };
      }
      if (highlightState.mode === "dim" && !isHighlighted) {
        return { ...node, style: { ...node.style, opacity: 0.5, transition } };
      }
      return { ...node, style: { ...node.style, opacity: 1, transition } };
    });
  }, [highlightState, nodesWithGoalBadges]);

  const [nodes, setNodes, onNodesChange] = useNodesState(styledNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(styledEdges);
  const useViewportRendering =
    styledNodes.length >= VIEWPORT_RENDERING_THRESHOLD ||
    styledEdges.length >= VIEWPORT_RENDERING_THRESHOLD;

  useEffect(() => {
    setNodes(styledNodes);
  }, [setNodes, styledNodes]);

  useEffect(() => {
    setEdges(styledEdges);
  }, [setEdges, styledEdges]);

  useGraphAutoFit({
    flowRef,
    lens,
    layoutMode,
    layoutDirection,
    showSecondaryEdges: settings.showSecondaryEdges,
    autoFitOnChange,
    processedNodes,
    processedEdges,
    styledNodesLength: styledNodes.length,
  });

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

    // Land the camera on the target too. Deep links (?select=/?focus=) and
    // lens drills should show the node, not wherever the persisted viewport
    // intent or fitView happens to leave the camera. Scheduled a frame later
    // than useGraphAutoFit's restore so this wins the race.
    const target = processedNodes.find((n) => n.id === targetNodeId);
    if (target) {
      const raf = window.requestAnimationFrame(() => {
        flowRef.current?.setCenter(target.position.x, target.position.y, {
          zoom: 1,
          duration: 300,
        });
      });
      return () => window.cancelAnimationFrame(raf);
    }
  }, [flowRef, focusNodeId, highlightState.mode, processedNodes, selectedNodeId, selectNode, setHighlightState, styledEdges]);

  // Reset the focus-restored flag on navigation context changes (lens switch
  // or focusNodeId change), so restoration fires for the new context.
  // Do NOT reset on selectedNodeId changes — those are user interactions
  // within the current context (node click, deselect) and should not
  // re-trigger restoration.
  useEffect(() => {
    focusRestoredRef.current = false;
  }, [focusNodeId, lens]);

  const handleNodeClick: NodeMouseHandler = useCallback(
    (_event, node) => {
      // All node clicks apply visual focus (BFS highlight/dim).
      // The NodeInspectorPanel reads selectedNodeId from the store
      // and shows entity info + navigation buttons.
      const focus = computeVisualFocus(node.id, processedNodes, styledEdges);
      if (focus) {
        selectNode(focus.selectedNodeId);
        setHighlightState(focus.highlightState);
      }
    },
    [processedNodes, selectNode, setHighlightState, styledEdges],
  );

  const handlePaneClick = useCallback(() => {
    const cleared = clearVisualFocus();
    selectNode(cleared.selectedNodeId);
    setHighlightState(cleared.highlightState);
  }, [selectNode, setHighlightState]);

  // Intent = "the user was looking at node X at zoom Z." Persisted per lens so
  // that a refresh restores what the user was looking at, not the raw pixel
  // coords (which are only valid for the exact container size, layout, and
  // node set that produced them).
  const handleMoveEnd = useCallback(
    (_event: MouseEvent | TouchEvent | null, viewport: Viewport) => {
      setViewportIntentForLens(lens, { nodeId: selectedNodeId, zoom: viewport.zoom });
    },
    [lens, selectedNodeId, setViewportIntentForLens],
  );

  // Track selection changes separately — the user can change selection without
  // panning/zooming (e.g., clicking a node in the sidebar). Read the current
  // zoom from the React Flow instance so we don't lose it.
  useEffect(() => {
    const instance = flowRef.current;
    if (!instance) return;
    const currentZoom = instance.getViewport().zoom;
    setViewportIntentForLens(lens, { nodeId: selectedNodeId, zoom: currentZoom });
  }, [lens, selectedNodeId, setViewportIntentForLens]);

  const setFlowInstance = useGraphUIStore((s) => s.setFlowInstance);

  const handleInit = useCallback(
    (instance: ReactFlowInstance<GraphNode>) => {
      flowRef.current = instance;
      setFlowInstance(instance);
    },
    [setFlowInstance],
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
        fitView
        fitViewOptions={{ padding: 0.2, maxZoom: 1.2 }}
        proOptions={{ hideAttribution: true }}
        minZoom={0.1}
        maxZoom={2}
        onlyRenderVisibleElements={useViewportRendering}
        /* Touch: prevent nodes from capturing pinch-to-zoom gestures.
           nodeDragThreshold requires a 5px move before drag starts,
           letting the browser recognize pinch/zoom first. */
        nodesDraggable={false}
        nodeDragThreshold={5}
        /* Disable unnecessary features for read-only graph */
        nodesConnectable={false}
        elementsSelectable={false}
      >
        {!useViewportRendering && (
          <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="rgb(51 65 85 / 0.4)" />
        )}
      </ReactFlow>

      {visibleEdgeTypes.length > 0 && <EdgeLegend edgeTypes={visibleEdgeTypes} />}

      {useViewportRendering && (
        <div
          className="pointer-events-none absolute bottom-4 right-4 z-10 rounded-md border border-slate-700/80 bg-slate-950/90 px-2.5 py-1.5 text-[11px] font-medium text-slate-300 shadow-lg"
          data-testid="graph-topology-summary"
          aria-label={`${styledNodes.length} graph nodes and ${styledEdges.length} graph edges`}
        >
          {styledNodes.length} nodes / {styledEdges.length} edges
        </div>
      )}

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
        lens === "focus" ? (
          <FocusEmptyState />
        ) : (
          <div
            className="absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-1/2 rounded-xl border border-slate-700/70 bg-slate-950/90 px-5 py-4 text-center shadow-lg"
            data-testid="graph-empty"
          >
            <p className="text-sm font-medium text-slate-100">No nodes match the current graph controls.</p>
            <p className="mt-1 text-xs text-slate-500">Try restoring entity or status visibility.</p>
          </div>
        )
      )}


    </div>
  );
});

export function GraphCanvas() {
  return (
    <Profiler id="GraphCanvas" onRender={onProfilerRender}>
      <GraphCanvasImpl />
    </Profiler>
  );
}
