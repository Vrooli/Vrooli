import * as d3 from "d3";
import type { DependencyGraphNode, DependencyGraphEdge, EdgeStatusFilter } from "../../types";

export type SimulationNode = DependencyGraphNode & d3.SimulationNodeDatum & {
  indexPosition: number;
};

export type SimulationEdge = DependencyGraphEdge & d3.SimulationLinkDatum<SimulationNode>;

export type LinkEndpoint = SimulationNode | { id: string } | string | number;

/**
 * Get fill color for a node based on its type
 */
export function getNodeFill(type: string): string {
  switch (type) {
    case "scenario":
      return "url(#nodeGlow)";
    case "resource":
      return "hsl(27, 87%, 62%)";
    default:
      return "hsl(276, 70%, 62%)";
  }
}

/**
 * Get stroke color for a node based on its type
 */
export function getNodeStroke(type: string): string {
  switch (type) {
    case "scenario":
      return "hsl(199, 90%, 72%)";
    case "resource":
      return "hsl(32, 90%, 70%)";
    default:
      return "hsl(280, 90%, 70%)";
  }
}

/**
 * Calculate node radius based on degree (number of connections)
 */
export function calculateNodeRadius(degree: number): number {
  return 16 + Math.min(degree * 1.4, 12);
}

/**
 * Build degree map (connection count) for all nodes in the graph
 */
export function buildDegreeMap(edges: DependencyGraphEdge[]): Map<string, number> {
  const map = new Map<string, number>();
  edges.forEach((edge) => {
    map.set(edge.source, (map.get(edge.source) ?? 0) + 1);
    map.set(edge.target, (map.get(edge.target) ?? 0) + 1);
  });
  return map;
}

/**
 * Check if an edge matches the current drift filter
 */
export function matchesDriftFilter(edge: DependencyGraphEdge, driftFilter: EdgeStatusFilter): boolean {
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
}

/**
 * Resolve a link endpoint to a SimulationNode
 * Handles various D3 edge endpoint formats
 */
export function resolveEdgeNode(
  value: LinkEndpoint,
  nodeMap: Map<string, SimulationNode>
): SimulationNode {
  if (typeof value === "string" || typeof value === "number") {
    const key = String(value);
    return (
      nodeMap.get(key) ??
      ({ id: key, label: key, group: "", type: "scenario" } as SimulationNode)
    );
  }
  if ("id" in value) {
    const key = value.id;
    return nodeMap.get(key) ?? ({
      ...(value as SimulationNode),
      id: key
    } as SimulationNode);
  }
  return value as SimulationNode;
}

/**
 * Check if a node matches the current search filter
 */
export function matchesSearchFilter(node: SimulationNode, filterText: string): boolean {
  if (!filterText) return true;
  const lower = filterText.toLowerCase();
  return node.label.toLowerCase().includes(lower) || node.id.toLowerCase().includes(lower);
}

/**
 * Get relative mouse position within a container
 */
export function getRelativePosition(
  event: MouseEvent | d3.D3DragEvent<any, any, any>,
  container: HTMLElement | null
): { x: number; y: number } {
  const bounds = container?.getBoundingClientRect();
  return {
    x: (event as MouseEvent).clientX - (bounds?.left ?? 0) + 12,
    y: (event as MouseEvent).clientY - (bounds?.top ?? 0) + 12
  };
}
