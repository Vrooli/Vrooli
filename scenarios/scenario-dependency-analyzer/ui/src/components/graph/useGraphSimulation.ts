import { useEffect, useRef } from "react";
import * as d3 from "d3";
import type { DependencyGraph, DependencyGraphNode, LayoutMode, EdgeStatusFilter } from "../../types";
import {
  type SimulationNode,
  type SimulationEdge,
  calculateNodeRadius,
  getNodeFill,
  getNodeStroke,
  matchesDriftFilter,
  getRelativePosition
} from "./graphUtils";

interface UseGraphSimulationProps {
  svgRef: React.RefObject<SVGSVGElement>;
  containerRef: React.RefObject<HTMLDivElement>;
  graph: DependencyGraph | null;
  layout: LayoutMode;
  driftFilter: EdgeStatusFilter;
  degreeMap: Map<string, number>;
  onSelectNode: (node: DependencyGraphNode | null) => void;
  onNodeHover: (node: DependencyGraphNode | null, position?: { x: number; y: number }) => void;
  onEdgeHover: (edge: DependencyGraph["edges"][number] | null, position?: { x: number; y: number }) => void;
}

/**
 * Custom hook that manages the D3 force simulation for the dependency graph
 * Handles SVG setup, node/edge rendering, force configuration, and interactions
 */
export function useGraphSimulation({
  svgRef,
  containerRef,
  graph,
  layout,
  driftFilter,
  degreeMap,
  onSelectNode,
  onNodeHover,
  onEdgeHover
}: UseGraphSimulationProps) {
  const nodeMapRef = useRef(new Map<string, SimulationNode>());

  useEffect(() => {
    if (!graph || !containerRef.current || !svgRef.current) {
      return;
    }

    const container = containerRef.current;
    const svg = d3.select(svgRef.current);
    const width = container.clientWidth || 960;
    const height = container.clientHeight || 640;

    // Clear and setup SVG
    svg.selectAll("*").remove();
    svg.attr("viewBox", `0 0 ${width} ${height}`).attr("aria-label", "Dependency graph");

    const root = svg.append("g");

    // Setup zoom behavior
    const zoom = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.2, 5])
      .on("zoom", (event: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
        root.attr("transform", event.transform.toString());
      });

    svg.call(zoom as any);

    // Setup gradient for scenario nodes
    const defs = svg.append("defs");
    const gradient = defs.append("radialGradient").attr("id", "nodeGlow");
    gradient
      .append("stop")
      .attr("offset", "0%")
      .attr("stop-color", "hsl(162, 100%, 52%)")
      .attr("stop-opacity", 0.95);
    gradient
      .append("stop")
      .attr("offset", "100%")
      .attr("stop-color", "hsl(162, 100%, 32%)")
      .attr("stop-opacity", 0.15);

    // Prepare nodes with index positions for grid layout
    const nodes: SimulationNode[] = graph.nodes.map((node, index) => ({
      ...node,
      indexPosition: index
    }));

    nodeMapRef.current = new Map(nodes.map((node) => [node.id, node]));

    const radiusForNode = (d: SimulationNode) => calculateNodeRadius(degreeMap.get(d.id) ?? 0);

    // Filter edges by drift filter
    const edges: SimulationEdge[] = graph.edges
      .filter((edge) => matchesDriftFilter(edge, driftFilter))
      .map((edge) => ({ ...edge }));

    // Render edges
    const link = root
      .append("g")
      .attr("stroke-linecap", "round")
      .attr("stroke-opacity", 0.68)
      .selectAll<SVGLineElement, SimulationEdge>("line")
      .data(edges)
      .join("line")
      .attr("stroke-width", (edge) => (edge.required ? 2.4 : 1.4))
      .attr("stroke", (edge) =>
        edge.required
          ? "hsl(0, 78%, 62%)"
          : driftFilter === "all"
            ? "hsl(162, 80%, 48%)"
            : edge.type === "resource"
              ? "hsl(48, 96%, 62%)"
              : "hsl(162, 80%, 48%)"
      )
      .attr("stroke-dasharray", (edge) => (edge.required ? null : "5 4"))
      .on("mouseenter", (event: MouseEvent, edge) => {
        onEdgeHover(edge, getRelativePosition(event, containerRef.current));
      })
      .on("mouseleave", () => onEdgeHover(null));

    // Render nodes
    const nodeGroup = root
      .append("g")
      .selectAll<SVGCircleElement, SimulationNode>("circle")
      .data(nodes)
      .join("circle")
      .attr("r", (d) => radiusForNode(d))
      .attr("fill", (d) => getNodeFill(d.type as string))
      .attr("stroke", (d) => getNodeStroke(d.type as string))
      .attr("stroke-width", 1.6)
      .attr("role", "button")
      .attr("tabindex", 0)
      .attr("aria-label", (d) => `${d.label} node`)
      .style("cursor", "pointer")
      .on("mouseenter", (event: MouseEvent, node) => {
        onEdgeHover(null);
        onNodeHover(node, getRelativePosition(event, containerRef.current));
      })
      .on("mouseleave", () => onNodeHover(null))
      .on("focus", (_event: FocusEvent, node) => {
        onNodeHover(null);
        onEdgeHover(null);
        onSelectNode(node);
      })
      .on("click", (_event: MouseEvent, node) => {
        onNodeHover(null);
        onEdgeHover(null);
        onSelectNode(node);
      })
      .on("keydown", (event: KeyboardEvent, node) => {
        if (event.key === " " || event.key === "Enter") {
          event.preventDefault();
          onSelectNode(node);
        }
      });

    // Render labels
    const labels = root
      .append("g")
      .selectAll<SVGTextElement, SimulationNode>("text")
      .data(nodes)
      .join("text")
      .attr("text-anchor", "middle")
      .attr("font-size", 12)
      .attr("font-weight", 520)
      .attr("fill", "hsla(210,20%,96%,0.92)")
      .attr("pointer-events", "none")
      .text((d) => d.label);

    // Create simulation
    const simulation = d3.forceSimulation(nodes);

    const linkForce = d3
      .forceLink<SimulationNode, SimulationEdge>(edges)
      .id((node) => node.id)
      .distance(layout === "force" ? 140 : layout === "radial" ? 160 : 150)
      .strength(layout === "grid" ? 0.15 : 0.8);

    simulation.force("link", linkForce);

    // Configure forces based on layout mode
    if (layout === "force") {
      simulation
        .force("charge", d3.forceManyBody().strength(-320))
        .force(
          "collision",
          d3.forceCollide<SimulationNode>().radius((d) => radiusForNode(d) + 8)
        )
        .force("center", d3.forceCenter(width / 2, height / 2));
    } else if (layout === "radial") {
      simulation
        .force("charge", d3.forceManyBody().strength(-260))
        .force(
          "radial",
          d3
            .forceRadial<SimulationNode>(
              (d) => (d.type === "resource" ? 140 : d.type === "scenario" ? 320 : 240),
              width / 2,
              height / 2
            )
            .strength(0.8)
        )
        .force(
          "collision",
          d3.forceCollide<SimulationNode>().radius((d) => radiusForNode(d) + 6)
        )
        .alpha(1.1);
    } else {
      // Grid layout
      const columns = Math.max(3, Math.ceil(Math.sqrt(nodes.length)));
      const rows = Math.ceil(nodes.length / columns);
      const cellWidth = width / (columns + 1);
      const cellHeight = height / (rows + 1);
      simulation
        .force("charge", d3.forceManyBody().strength(-120))
        .force(
          "x",
          d3.forceX<SimulationNode>((d) =>
            ((d.indexPosition % columns) + 1) * cellWidth
          ).strength(0.6)
        )
        .force(
          "y",
          d3.forceY<SimulationNode>((d) =>
            (Math.floor(d.indexPosition / columns) + 1) * cellHeight
          ).strength(0.8)
        )
        .force(
          "collision",
          d3.forceCollide<SimulationNode>().radius((d) => radiusForNode(d) + 4)
        )
        .alpha(1.05);
    }

    // Update positions on each simulation tick
    simulation.on("tick", () => {
      link
        .attr("x1", (edge) => (edge.source as SimulationNode).x ?? 0)
        .attr("y1", (edge) => (edge.source as SimulationNode).y ?? 0)
        .attr("x2", (edge) => (edge.target as SimulationNode).x ?? 0)
        .attr("y2", (edge) => (edge.target as SimulationNode).y ?? 0);

      nodeGroup
        .attr("cx", (d) => d.x ?? 0)
        .attr("cy", (d) => d.y ?? 0);

      labels
        .attr("x", (d) => d.x ?? 0)
        .attr("y", (d) => (d.y ?? 0) + 28);
    });

    return () => {
      simulation.stop();
    };
  }, [graph, layout, degreeMap, driftFilter, onSelectNode, onNodeHover, onEdgeHover, containerRef, svgRef]);

  return nodeMapRef;
}
