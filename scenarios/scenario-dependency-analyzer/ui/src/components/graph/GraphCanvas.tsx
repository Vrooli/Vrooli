import { useEffect, useMemo, useRef, useState } from "react";
import * as d3 from "d3";

import type {
  DependencyGraph,
  DependencyGraphEdge,
  DependencyGraphNode,
  LayoutMode,
  EdgeStatusFilter
} from "../../types";
import { cn } from "../../lib/utils";
import { NodeTooltip, EdgeTooltip } from "./GraphTooltips";
import { useGraphSimulation } from "./useGraphSimulation";
import {
  type SimulationNode,
  type SimulationEdge,
  type LinkEndpoint,
  buildDegreeMap,
  matchesSearchFilter,
  resolveEdgeNode,
  getNodeStroke
} from "./graphUtils";

interface GraphCanvasProps {
  graph: DependencyGraph | null;
  layout: LayoutMode;
  filter: string;
  driftFilter: EdgeStatusFilter;
  selectedNodeId?: string | null;
  onSelectNode: (node: DependencyGraphNode | null) => void;
  className?: string;
}

interface HoverState {
  node: DependencyGraphNode;
  position: { x: number; y: number };
}

interface EdgeHoverState {
  edge: DependencyGraphEdge;
  position: { x: number; y: number };
}

/**
 * GraphCanvas component - renders an interactive D3 force-directed graph
 * for visualizing dependency relationships between scenarios and resources.
 *
 * Features:
 * - Three layout modes: force-directed, radial, and grid
 * - Interactive node selection and hover tooltips
 * - Real-time search filtering
 * - Drift detection filtering (missing/declared dependencies)
 * - Zoom and pan controls
 * - Keyboard navigation support
 */
export function GraphCanvas({
  graph,
  layout,
  filter,
  driftFilter,
  selectedNodeId,
  onSelectNode,
  className
}: GraphCanvasProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [hover, setHover] = useState<HoverState | null>(null);
  const [edgeHover, setEdgeHover] = useState<EdgeHoverState | null>(null);

  const filterText = filter.trim().toLowerCase();

  // Calculate degree (connection count) for each node
  const degreeMap = useMemo(() => {
    if (!graph) return new Map<string, number>();
    return buildDegreeMap(graph.edges);
  }, [graph]);

  // Setup D3 simulation with all graph rendering logic
  const nodeMapRef = useGraphSimulation({
    svgRef,
    containerRef,
    graph,
    layout,
    driftFilter,
    degreeMap,
    onSelectNode,
    onNodeHover: (node, position) => {
      if (node && position) {
        setHover({ node, position });
      } else {
        setHover(null);
      }
    },
    onEdgeHover: (edge, position) => {
      if (edge && position) {
        setEdgeHover({ edge, position });
      } else {
        setEdgeHover(null);
      }
    }
  });

  // Apply search filter highlighting
  useEffect(() => {
    if (!graph || !svgRef.current) return;
    const svg = d3.select(svgRef.current);

    const nodeSelection = svg.selectAll<SVGCircleElement, SimulationNode>("circle");
    const labelSelection = svg.selectAll<SVGTextElement, SimulationNode>("text");
    const linkSelection = svg.selectAll<SVGLineElement, SimulationEdge>("line");

    if (!filterText) {
      nodeSelection.style("opacity", 1).style("filter", null);
      labelSelection.style("opacity", 1);
      linkSelection.style("opacity", 0.68);
      return;
    }

    nodeSelection.style("opacity", (node) => (matchesSearchFilter(node, filterText) ? 1 : 0.22));
    labelSelection.style("opacity", (node) => (matchesSearchFilter(node, filterText) ? 1 : 0.18));
    linkSelection.style("opacity", (edge) => {
      const source = resolveEdgeNode(edge.source as LinkEndpoint, nodeMapRef.current);
      const target = resolveEdgeNode(edge.target as LinkEndpoint, nodeMapRef.current);
      const sourceMatch = matchesSearchFilter(source, filterText);
      const targetMatch = matchesSearchFilter(target, filterText);
      return sourceMatch || targetMatch ? 0.68 : 0.12;
    });
  }, [filterText, graph]);

  // Apply selection highlighting
  useEffect(() => {
    if (!svgRef.current) return;
    const svg = d3.select(svgRef.current);

    svg
      .selectAll<SVGCircleElement, SimulationNode>("circle")
      .classed("ring-4", (d) => d.id === selectedNodeId)
      .classed("ring-primary/60", (d) => d.id === selectedNodeId)
      .attr("stroke-width", (d) => (d.id === selectedNodeId ? 2.4 : 1.6))
      .attr("stroke", (d) =>
        d.id === selectedNodeId
          ? "hsl(162, 100%, 72%)"
          : getNodeStroke(d.type as string)
      );

    svg
      .selectAll<SVGLineElement, SimulationEdge>("line")
      .attr("stroke-opacity", (edge) => {
        if (!selectedNodeId) return 0.68;
        const source = resolveEdgeNode(edge.source as LinkEndpoint, nodeMapRef.current);
        const target = resolveEdgeNode(edge.target as LinkEndpoint, nodeMapRef.current);
        return source.id === selectedNodeId || target.id === selectedNodeId
          ? 0.95
          : 0.15;
      })
      .attr("stroke-width", (edge) => {
        if (!selectedNodeId) return edge.required ? 2.4 : 1.4;
        const source = resolveEdgeNode(edge.source as LinkEndpoint, nodeMapRef.current);
        const target = resolveEdgeNode(edge.target as LinkEndpoint, nodeMapRef.current);
        const isActive = source.id === selectedNodeId || target.id === selectedNodeId;
        return isActive ? 3.2 : edge.required ? 1.6 : 1.0;
      });
  }, [selectedNodeId]);

  return (
    <div className={cn("relative h-full w-full overflow-hidden", className)} ref={containerRef}>
      <svg ref={svgRef} className="h-full w-full" role="application" />
      {hover ? <NodeTooltip hover={hover} /> : null}
      {edgeHover ? <EdgeTooltip hover={edgeHover} /> : null}
    </div>
  );
}
