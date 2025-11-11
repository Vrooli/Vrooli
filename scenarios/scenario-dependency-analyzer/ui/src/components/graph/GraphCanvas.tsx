import { useEffect, useMemo, useRef, useState } from "react";
import * as d3 from "d3";

import type {
  DependencyGraph,
  DependencyGraphEdge,
  DependencyGraphNode,
  GraphType,
  LayoutMode,
  EdgeStatusFilter
} from "../../types";
import { cn } from "../../lib/utils";

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

type SimulationNode = DependencyGraphNode & d3.SimulationNodeDatum & {
  indexPosition: number;
};

type SimulationEdge = DependencyGraphEdge & d3.SimulationLinkDatum<SimulationNode>;

type LinkEndpoint = SimulationNode | { id: string } | string | number;

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
  const nodeMapRef = useRef(new Map<string, SimulationNode>());

  const filterText = filter.trim().toLowerCase();

  const degreeMap = useMemo(() => {
    const map = new Map<string, number>();
    if (!graph) return map;
    graph.edges.forEach((edge) => {
      map.set(edge.source, (map.get(edge.source) ?? 0) + 1);
      map.set(edge.target, (map.get(edge.target) ?? 0) + 1);
    });
    return map;
  }, [graph]);

  useEffect(() => {
    if (!graph || !containerRef.current || !svgRef.current) {
      return;
    }

    const container = containerRef.current;
    const svg = d3.select(svgRef.current);
    const width = container.clientWidth || 960;
    const height = container.clientHeight || 640;

    svg.selectAll("*").remove();
    svg.attr("viewBox", `0 0 ${width} ${height}`).attr("aria-label", "Dependency graph");

    const root = svg.append("g");

    const zoom = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.2, 5])
      .on("zoom", (event: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
        root.attr("transform", event.transform.toString());
      });

    svg.call(zoom as any);

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

    const nodes: SimulationNode[] = graph.nodes.map((node, index) => ({
      ...node,
      indexPosition: index
    }));

    nodeMapRef.current = new Map(nodes.map((node) => [node.id, node]));

    const radiusForNode = (d: SimulationNode) =>
      16 + Math.min((degreeMap.get(d.id) ?? 0) * 1.4, 12);

    const matchDrift = (edge: DependencyGraphEdge) => {
      const metadata = (edge.metadata as Record<string, unknown> | undefined) ?? {};
      const configuration = (metadata.configuration as Record<string, unknown> | undefined) ?? {};
      const hasFile = typeof configuration.found_in_file === "string";

      if (driftFilter === "all") return true;
      if (driftFilter === "missing") {
        return edge.type === "resource" && hasFile;
      }
      if (driftFilter === "declared-only") {
        return edge.type === "resource" && metadata.access_method === "declared" && !hasFile;
      }
      return true;
    };

    const edges: SimulationEdge[] = graph.edges
      .filter((edge) => matchDrift(edge))
      .map((edge) => ({ ...edge }));

    const getRelativePosition = (event: MouseEvent | d3.D3DragEvent<any, any, any>) => {
      const bounds = containerRef.current?.getBoundingClientRect();
      return {
        x: (event as MouseEvent).clientX - (bounds?.left ?? 0) + 12,
        y: (event as MouseEvent).clientY - (bounds?.top ?? 0) + 12
      };
    };

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
        setEdgeHover({ edge, position: getRelativePosition(event) });
      })
      .on("mouseleave", () => setEdgeHover(null));

    const nodeGroup = root
      .append("g")
      .selectAll<SVGCircleElement, SimulationNode>("circle")
      .data(nodes)
      .join("circle")
      .attr("r", (d) => radiusForNode(d))
      .attr("fill", (d) => getNodeFill(d.type as GraphType | string))
      .attr("stroke", (d) => getNodeStroke(d.type as GraphType | string))
      .attr("stroke-width", 1.6)
      .attr("role", "button")
      .attr("tabindex", 0)
      .attr("aria-label", (d) => `${d.label} node`)
      .style("cursor", "pointer")
      .on("mouseenter", (event: MouseEvent, node) => {
        setEdgeHover(null);
        setHover({ node, position: getRelativePosition(event) });
      })
      .on("mouseleave", () => setHover(null))
      .on("focus", (_event: FocusEvent, node) => {
        setHover(null);
        setEdgeHover(null);
        onSelectNode(node);
      })
      .on("click", (_event: MouseEvent, node) => {
        setHover(null);
        setEdgeHover(null);
        onSelectNode(node);
      })
      .on("keydown", (event: KeyboardEvent, node) => {
        if (event.key === " " || event.key === "Enter") {
          event.preventDefault();
          onSelectNode(node);
        }
      });

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

    const simulation = d3.forceSimulation(nodes);

    const linkForce = d3
      .forceLink<SimulationNode, SimulationEdge>(edges)
      .id((node) => node.id)
      .distance(layout === "force" ? 140 : layout === "radial" ? 160 : 150)
      .strength(layout === "grid" ? 0.15 : 0.8);

    simulation.force("link", linkForce);

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
  }, [graph, layout, degreeMap, onSelectNode]);

  useEffect(() => {
    if (!graph || !svgRef.current) return;
    const svg = d3.select(svgRef.current);
    const filterLower = filterText;

    const nodeSelection = svg.selectAll<SVGCircleElement, SimulationNode>("circle");
    const labelSelection = svg.selectAll<SVGTextElement, SimulationNode>("text");
    const linkSelection = svg.selectAll<SVGLineElement, SimulationEdge>("line");

    if (!filterLower) {
      nodeSelection.style("opacity", 1).style("filter", null);
      labelSelection.style("opacity", 1);
      linkSelection.style("opacity", 0.68);
      return;
    }

    const resolveEdgeNode = (value: LinkEndpoint): SimulationNode => {
      if (typeof value === "string" || typeof value === "number") {
        const key = String(value);
        return (
          nodeMapRef.current.get(key) ??
          ({ id: key, label: key, group: "", type: "scenario" } as SimulationNode)
        );
      }
      if ("id" in value) {
        const key = value.id;
        return nodeMapRef.current.get(key) ?? ({
          ...(value as SimulationNode),
          id: key
        } as SimulationNode);
      }
      return value as SimulationNode;
    };

    const matchNode = (node: SimulationNode) =>
      node.label.toLowerCase().includes(filterLower) ||
      node.id.toLowerCase().includes(filterLower);

    nodeSelection.style("opacity", (node) => (matchNode(node) ? 1 : 0.22));
    labelSelection.style("opacity", (node) => (matchNode(node) ? 1 : 0.18));
    linkSelection.style("opacity", (edge) => {
      const source = edge.source as LinkEndpoint;
      const target = edge.target as LinkEndpoint;
      const sourceMatch = matchNode(resolveEdgeNode(source));
      const targetMatch = matchNode(resolveEdgeNode(target));
      return sourceMatch || targetMatch ? 0.68 : 0.12;
    });
  }, [filterText, graph]);

  useEffect(() => {
    if (!svgRef.current) return;
    const svg = d3.select(svgRef.current);
    const resolveEdgeNode = (value: LinkEndpoint): SimulationNode => {
      if (typeof value === "string" || typeof value === "number") {
        const key = String(value);
        return (
          nodeMapRef.current.get(key) ??
          ({ id: key, label: key, group: "", type: "scenario" } as SimulationNode)
        );
      }
      if ("id" in value) {
        const key = value.id;
        return nodeMapRef.current.get(key) ?? ({
          ...(value as SimulationNode),
          id: key
        } as SimulationNode);
      }
      return value as SimulationNode;
    };
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
        const source = resolveEdgeNode(edge.source as LinkEndpoint);
        const target = resolveEdgeNode(edge.target as LinkEndpoint);
        return source.id === selectedNodeId || target.id === selectedNodeId
          ? 0.95
          : 0.15;
      })
      .attr("stroke-width", (edge) => {
        if (!selectedNodeId) return edge.required ? 2.4 : 1.4;
        const source = resolveEdgeNode(edge.source as LinkEndpoint);
        const target = resolveEdgeNode(edge.target as LinkEndpoint);
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

function NodeTooltip({ hover }: { hover: HoverState }) {
  const { node, position } = hover;
  return (
    <div
      className="pointer-events-none absolute z-20 min-w-[220px] rounded-lg border border-border/40 bg-background/80 p-4 text-sm shadow-xl shadow-black/20 backdrop-blur"
      style={{ left: position.x, top: position.y }}
    >
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {node.type}
      </p>
      <p className="mt-1 font-semibold text-foreground">{node.label}</p>
      {node.metadata && node.metadata["purpose"] ? (
        <p className="mt-2 text-xs text-muted-foreground/90">
          {(node.metadata["purpose"] as string) ?? ""}
        </p>
      ) : null}
      {node.group ? (
        <p className="mt-3 text-[11px] uppercase tracking-widest text-primary/80">
          {node.group}
        </p>
      ) : null}
    </div>
  );
}

function EdgeTooltip({ hover }: { hover: EdgeHoverState }) {
  const { edge, position } = hover;
  const metadata = (edge.metadata as Record<string, unknown> | undefined) ?? {};
  const configuration = (metadata.configuration as Record<string, unknown> | undefined) ?? {};

  const formatTimestamp = (value?: unknown) => {
    if (typeof value !== "string") return null;
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return null;
    return date.toLocaleString();
  };

  return (
    <div
      className="pointer-events-none absolute z-20 min-w-[240px] rounded-lg border border-border/40 bg-background/90 p-4 text-xs shadow-2xl shadow-black/40 backdrop-blur"
      style={{ left: position.x, top: position.y }}
    >
      <p className="font-semibold text-foreground">{edge.label || edge.type}</p>
      <p className="mt-1 text-[11px] uppercase tracking-widest text-muted-foreground/80">
        {edge.required ? "Required" : "Optional"} · Weight {edge.weight.toFixed(2)}
      </p>
      {metadata.access_method ? (
        <p className="mt-2 text-muted-foreground">Access: {String(metadata.access_method)}</p>
      ) : null}
      {metadata.purpose ? (
        <p className="mt-1 text-muted-foreground">Purpose: {String(metadata.purpose)}</p>
      ) : null}
      {typeof configuration.found_in_file === "string" ? (
        <p className="mt-1 text-muted-foreground">File: {configuration.found_in_file as string}</p>
      ) : null}
      {typeof configuration.pattern_type === "string" ? (
        <p className="text-muted-foreground">Pattern: {configuration.pattern_type as string}</p>
      ) : null}
      {metadata.discovered_at ? (
        <p className="mt-1 text-muted-foreground">Detected: {formatTimestamp(metadata.discovered_at)}</p>
      ) : null}
      {metadata.last_verified ? (
        <p className="text-muted-foreground">Verified: {formatTimestamp(metadata.last_verified)}</p>
      ) : null}
    </div>
  );
}

function getNodeFill(type: string) {
  switch (type) {
    case "scenario":
      return "url(#nodeGlow)";
    case "resource":
      return "hsl(27, 87%, 62%)";
    default:
      return "hsl(276, 70%, 62%)";
  }
}

function getNodeStroke(type: string) {
  switch (type) {
    case "scenario":
      return "hsl(199, 90%, 72%)";
    case "resource":
      return "hsl(32, 90%, 70%)";
    default:
      return "hsl(280, 90%, 70%)";
  }
}
