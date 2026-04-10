import type { GraphNode, GraphResponse } from "../../shared/services/api";

export type GraphLayoutMode = "radial" | "force" | "column";

export type CanvasNode = {
  id: string;
  label: string;
  score?: number;
  metadata: Record<string, unknown>;
  isCenter: boolean;
};

export type CanvasEdge = {
  id: string;
  source: string;
  target: string;
  weight: number;
  relationship: string;
};

export type CanvasGraph = {
  center: string;
  centerNodeID: string;
  nodes: CanvasNode[];
  edges: CanvasEdge[];
};

export type CanvasViewport = {
  scale: number;
  offsetX: number;
  offsetY: number;
};

export type CanvasRenderNode = CanvasNode & {
  x: number;
  y: number;
  radius: number;
  isSelected: boolean;
  isNeighbor: boolean;
  isFaded: boolean;
};

export type CanvasRenderEdge = CanvasEdge & {
  sourceX: number;
  sourceY: number;
  targetX: number;
  targetY: number;
  isHighlighted: boolean;
  isFaded: boolean;
};

export type CanvasRenderModel = {
  nodes: CanvasRenderNode[];
  edges: CanvasRenderEdge[];
  truncatedCount: number;
};

export const CANVAS_WORLD_WIDTH = 1200;
export const CANVAS_WORLD_HEIGHT = 720;

const DEFAULT_MAX_RENDER_NODES = 400;

const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(max, value));

const toFiniteWeight = (value: number | undefined, fallback: number) =>
  typeof value === "number" && Number.isFinite(value) ? value : fallback;

const normalizeCenterNodeID = (center: string) => `center:${center.trim().toLowerCase()}`;

const normalizeCenterNode = (node: GraphNode, center: string): CanvasNode => ({
  id: normalizeCenterNodeID(center),
  label: node.label || center,
  score: node.score,
  metadata: node.metadata ?? {},
  isCenter: true,
});

const normalizeRegularNode = (node: GraphNode): CanvasNode => ({
  id: node.id,
  label: node.label || node.id,
  score: node.score,
  metadata: node.metadata ?? {},
  isCenter: false,
});

const edgeKey = (source: string, target: string, relationship: string) =>
  `${source}->${target}:${relationship}`;

export function toCanvasGraph(response: GraphResponse): CanvasGraph {
  const centerNodeID = normalizeCenterNodeID(response.center);
  const nodeMap = new Map<string, CanvasNode>();

  for (const node of response.nodes) {
    if (node.id === "center") {
      nodeMap.set(centerNodeID, normalizeCenterNode(node, response.center));
      continue;
    }
    nodeMap.set(node.id, normalizeRegularNode(node));
  }

  if (!nodeMap.has(centerNodeID)) {
    nodeMap.set(centerNodeID, {
      id: centerNodeID,
      label: response.center,
      score: 1,
      metadata: { type: "center" },
      isCenter: true,
    });
  }

  const edgeMap = new Map<string, CanvasEdge>();
  for (const edge of response.edges) {
    const source = edge.source === "center" ? centerNodeID : edge.source;
    const target = edge.target === "center" ? centerNodeID : edge.target;
    if (!nodeMap.has(source) || !nodeMap.has(target) || source === target) {
      continue;
    }
    const relationship = edge.relationship || "semantic_similarity";
    const normalized: CanvasEdge = {
      id: edgeKey(source, target, relationship),
      source,
      target,
      weight: toFiniteWeight(edge.weight, 0),
      relationship,
    };
    edgeMap.set(normalized.id, normalized);
  }

  return {
    center: response.center,
    centerNodeID,
    nodes: [...nodeMap.values()],
    edges: [...edgeMap.values()],
  };
}

export function getNodeNeighbors(graph: CanvasGraph, nodeID: string): Set<string> {
  const neighbors = new Set<string>([nodeID]);
  for (const edge of graph.edges) {
    if (edge.source === nodeID) neighbors.add(edge.target);
    if (edge.target === nodeID) neighbors.add(edge.source);
  }
  return neighbors;
}

export function filterGraphByEdgeWeight(graph: CanvasGraph, minWeight: number): CanvasGraph {
  const threshold = clamp(minWeight, 0, 1);
  const edges = graph.edges.filter((edge) => edge.weight >= threshold);
  const visibleNodeIDs = new Set<string>([graph.centerNodeID]);
  for (const edge of edges) {
    visibleNodeIDs.add(edge.source);
    visibleNodeIDs.add(edge.target);
  }
  const nodes = graph.nodes.filter((node) => visibleNodeIDs.has(node.id));
  return {
    center: graph.center,
    centerNodeID: graph.centerNodeID,
    nodes,
    edges,
  };
}

const getNodeScore = (node: CanvasNode) => (typeof node.score === "number" ? node.score : 0);

export function capGraph(graph: CanvasGraph, maxNodes = DEFAULT_MAX_RENDER_NODES): {
  graph: CanvasGraph;
  truncatedCount: number;
} {
  if (graph.nodes.length <= maxNodes) {
    return { graph, truncatedCount: 0 };
  }

  const nodeMap = new Map(graph.nodes.map((node) => [node.id, node]));
  const edgesSorted = [...graph.edges].sort((a, b) => b.weight - a.weight);
  const keepNodeIDs = new Set<string>([graph.centerNodeID]);

  for (const edge of edgesSorted) {
    if (keepNodeIDs.size >= maxNodes) break;
    if (nodeMap.has(edge.source)) keepNodeIDs.add(edge.source);
    if (keepNodeIDs.size >= maxNodes) break;
    if (nodeMap.has(edge.target)) keepNodeIDs.add(edge.target);
  }

  if (keepNodeIDs.size < maxNodes) {
    const rankedNodes = [...graph.nodes]
      .filter((node) => !keepNodeIDs.has(node.id))
      .sort((a, b) => getNodeScore(b) - getNodeScore(a));
    for (const node of rankedNodes) {
      if (keepNodeIDs.size >= maxNodes) break;
      keepNodeIDs.add(node.id);
    }
  }

  const nodes = graph.nodes.filter((node) => keepNodeIDs.has(node.id));
  const edges = graph.edges.filter((edge) => keepNodeIDs.has(edge.source) && keepNodeIDs.has(edge.target));

  return {
    graph: {
      center: graph.center,
      centerNodeID: graph.centerNodeID,
      nodes,
      edges,
    },
    truncatedCount: graph.nodes.length - nodes.length,
  };
}

function layoutRadial(nodes: CanvasNode[]): Map<string, { x: number; y: number }> {
  const map = new Map<string, { x: number; y: number }>();
  if (!nodes.length) return map;

  const centerX = CANVAS_WORLD_WIDTH / 2;
  const centerY = CANVAS_WORLD_HEIGHT / 2;

  const fallbackCenter = nodes[0];
  if (!fallbackCenter) return map;
  const centerNode = nodes.find((node) => node.isCenter) ?? fallbackCenter;
  map.set(centerNode.id, { x: centerX, y: centerY });

  const others = nodes.filter((node) => node.id !== centerNode.id);
  const ringSize = 14;
  for (let index = 0; index < others.length; index += 1) {
    const node = others[index];
    if (!node) continue;
    const ring = Math.floor(index / ringSize) + 1;
    const ringIndex = index % ringSize;
    const countInRing = Math.min(ringSize, others.length - Math.floor(index / ringSize) * ringSize);
    const radius = 130 + ring * 92;
    const angle = (Math.PI * 2 * ringIndex) / Math.max(countInRing, 1);
    map.set(node.id, {
      x: centerX + Math.cos(angle) * radius,
      y: centerY + Math.sin(angle) * radius,
    });
  }

  return map;
}

function seededValue(seed: string) {
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash << 5) - hash + seed.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash % 10000) / 10000;
}

function layoutForce(nodes: CanvasNode[], edges: CanvasEdge[]): Map<string, { x: number; y: number }> {
  const positions = layoutRadial(nodes);
  for (const node of nodes) {
    const jitter = seededValue(node.id) - 0.5;
    const base = positions.get(node.id);
    if (base) {
      base.x += jitter * 40;
      base.y += jitter * 24;
      positions.set(node.id, base);
    }
  }

  const iterations = 80;
  const repulsion = 5800;
  const attraction = 0.004;
  const maxStep = 18;

  for (let step = 0; step < iterations; step += 1) {
    const delta = new Map<string, { x: number; y: number }>();

    for (let i = 0; i < nodes.length; i += 1) {
      const a = nodes[i];
      if (!a) continue;
      const pa = positions.get(a.id);
      if (!pa) continue;

      for (let j = i + 1; j < nodes.length; j += 1) {
        const b = nodes[j];
        if (!b) continue;
        const pb = positions.get(b.id);
        if (!pb) continue;
        const dx = pa.x - pb.x;
        const dy = pa.y - pb.y;
        const distSq = Math.max(dx * dx + dy * dy, 1);
        const force = repulsion / distSq;
        const dist = Math.sqrt(distSq);
        const ux = dx / dist;
        const uy = dy / dist;

        const da = delta.get(a.id) ?? { x: 0, y: 0 };
        const db = delta.get(b.id) ?? { x: 0, y: 0 };
        da.x += ux * force;
        da.y += uy * force;
        db.x -= ux * force;
        db.y -= uy * force;
        delta.set(a.id, da);
        delta.set(b.id, db);
      }
    }

    for (const edge of edges) {
      const source = positions.get(edge.source);
      const target = positions.get(edge.target);
      if (!source || !target) continue;
      const dx = target.x - source.x;
      const dy = target.y - source.y;
      const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
      const stretch = dist - 150;
      const strength = attraction * (1 + edge.weight * 2);
      const fx = (dx / dist) * stretch * strength;
      const fy = (dy / dist) * stretch * strength;

      const ds = delta.get(edge.source) ?? { x: 0, y: 0 };
      const dt = delta.get(edge.target) ?? { x: 0, y: 0 };
      ds.x += fx;
      ds.y += fy;
      dt.x -= fx;
      dt.y -= fy;
      delta.set(edge.source, ds);
      delta.set(edge.target, dt);
    }

    for (const node of nodes) {
      const point = positions.get(node.id);
      if (!point) continue;
      const d = delta.get(node.id) ?? { x: 0, y: 0 };
      point.x = clamp(point.x + clamp(d.x, -maxStep, maxStep), 60, CANVAS_WORLD_WIDTH - 60);
      point.y = clamp(point.y + clamp(d.y, -maxStep, maxStep), 60, CANVAS_WORLD_HEIGHT - 60);
      positions.set(node.id, point);
    }
  }

  return positions;
}

function layoutColumn(nodes: CanvasNode[]): Map<string, { x: number; y: number }> {
  const centerX = CANVAS_WORLD_WIDTH / 2;
  const leftX = CANVAS_WORLD_WIDTH * 0.26;
  const rightX = CANVAS_WORLD_WIDTH * 0.74;
  const centerY = CANVAS_WORLD_HEIGHT / 2;

  const map = new Map<string, { x: number; y: number }>();
  const fallbackCenter = nodes[0];
  if (!fallbackCenter) return map;
  const centerNode = nodes.find((node) => node.isCenter) ?? fallbackCenter;
  map.set(centerNode.id, { x: centerX, y: centerY });

  const others = nodes.filter((node) => node.id !== centerNode.id);
  const midpoint = Math.ceil(others.length / 2);
  const left = others.slice(0, midpoint);
  const right = others.slice(midpoint);

  const placeColumn = (column: CanvasNode[], x: number) => {
    if (column.length === 0) return;
    const step = Math.min(110, (CANVAS_WORLD_HEIGHT - 120) / column.length);
    for (let index = 0; index < column.length; index += 1) {
      const node = column[index];
      if (!node) continue;
      const y = 90 + index * step;
      map.set(node.id, { x, y });
    }
  };

  placeColumn(left, leftX);
  placeColumn(right, rightX);

  return map;
}

export function layoutNodes(
  nodes: CanvasNode[],
  edges: CanvasEdge[],
  mode: GraphLayoutMode,
): Map<string, { x: number; y: number }> {
  if (mode === "force") return layoutForce(nodes, edges);
  if (mode === "column") return layoutColumn(nodes);
  return layoutRadial(nodes);
}

export function buildCanvasRenderModel(params: {
  graph: CanvasGraph;
  layoutMode: GraphLayoutMode;
  minWeight: number;
  maxNodes?: number;
  selectedNodeID?: string;
  highlightNeighbors: boolean;
}): CanvasRenderModel {
  const weightedGraph = filterGraphByEdgeWeight(params.graph, params.minWeight);
  const { graph: cappedGraph, truncatedCount } = capGraph(weightedGraph, params.maxNodes);
  const positions = layoutNodes(cappedGraph.nodes, cappedGraph.edges, params.layoutMode);
  const selectedNodeID = params.selectedNodeID;
  const neighborSet =
    params.highlightNeighbors && selectedNodeID
      ? getNodeNeighbors(cappedGraph, selectedNodeID)
      : new Set<string>();

  const renderNodes: CanvasRenderNode[] = cappedGraph.nodes.map((node) => {
    const point = positions.get(node.id) ?? { x: CANVAS_WORLD_WIDTH / 2, y: CANVAS_WORLD_HEIGHT / 2 };
    const isSelected = selectedNodeID === node.id;
    const isNeighbor = neighborSet.has(node.id);
    const isFaded = Boolean(
      selectedNodeID && params.highlightNeighbors && !isSelected && !isNeighbor
    );
    return {
      ...node,
      x: point.x,
      y: point.y,
      radius: node.isCenter ? 26 : 11 + Math.round(clamp(getNodeScore(node), 0, 1) * 11),
      isSelected,
      isNeighbor,
      isFaded,
    };
  });

  const pointByID = new Map(renderNodes.map((node) => [node.id, node]));

  const renderEdges: CanvasRenderEdge[] = cappedGraph.edges
    .map((edge) => {
      const source = pointByID.get(edge.source);
      const target = pointByID.get(edge.target);
      if (!source || !target) return null;
      const isHighlighted =
        selectedNodeID !== undefined &&
        (edge.source === selectedNodeID || edge.target === selectedNodeID ||
          (params.highlightNeighbors && neighborSet.has(edge.source) && neighborSet.has(edge.target)));
      const isFaded = Boolean(selectedNodeID && params.highlightNeighbors && !isHighlighted);
      return {
        ...edge,
        sourceX: source.x,
        sourceY: source.y,
        targetX: target.x,
        targetY: target.y,
        isHighlighted,
        isFaded,
      } satisfies CanvasRenderEdge;
    })
    .filter((edge): edge is CanvasRenderEdge => edge !== null);

  return {
    nodes: renderNodes,
    edges: renderEdges,
    truncatedCount,
  };
}

export function fitViewportToRenderModel(model: CanvasRenderModel): CanvasViewport {
  if (model.nodes.length === 0) {
    return { scale: 1, offsetX: 0, offsetY: 0 };
  }

  let minX = Number.POSITIVE_INFINITY;
  let maxX = Number.NEGATIVE_INFINITY;
  let minY = Number.POSITIVE_INFINITY;
  let maxY = Number.NEGATIVE_INFINITY;

  for (const node of model.nodes) {
    minX = Math.min(minX, node.x - node.radius - 20);
    maxX = Math.max(maxX, node.x + node.radius + 20);
    minY = Math.min(minY, node.y - node.radius - 20);
    maxY = Math.max(maxY, node.y + node.radius + 20);
  }

  const width = Math.max(maxX - minX, 240);
  const height = Math.max(maxY - minY, 180);
  const padding = 0.92;
  const scaleX = (CANVAS_WORLD_WIDTH * padding) / width;
  const scaleY = (CANVAS_WORLD_HEIGHT * padding) / height;
  const scale = clamp(Math.min(scaleX, scaleY), 0.45, 2.4);

  const contentCenterX = minX + width / 2;
  const contentCenterY = minY + height / 2;

  return {
    scale,
    offsetX: CANVAS_WORLD_WIDTH / 2 - contentCenterX * scale,
    offsetY: CANVAS_WORLD_HEIGHT / 2 - contentCenterY * scale,
  };
}

export function mergeCanvasGraphs(params: {
  base: CanvasGraph;
  incoming: CanvasGraph;
  anchorNodeID?: string;
}): CanvasGraph {
  const { base, incoming, anchorNodeID } = params;
  const incomingCenterID = incoming.centerNodeID;

  const nodeMap = new Map(base.nodes.map((node) => [node.id, node]));

  const resolveNodeID = (node: CanvasNode) => {
    if (node.id === incomingCenterID && anchorNodeID) return anchorNodeID;
    return node.id;
  };

  for (const node of incoming.nodes) {
    const resolvedID = resolveNodeID(node);
    const existing = nodeMap.get(resolvedID);
    if (!existing) {
      nodeMap.set(resolvedID, {
        ...node,
        id: resolvedID,
        isCenter: resolvedID === base.centerNodeID,
      });
      continue;
    }

    nodeMap.set(resolvedID, {
      ...existing,
      score: existing.score ?? node.score,
      metadata: Object.keys(existing.metadata ?? {}).length > 0 ? existing.metadata : node.metadata,
      label: existing.label || node.label,
    });
  }

  const edgeMap = new Map(base.edges.map((edge) => [edge.id, edge]));
  for (const edge of incoming.edges) {
    const source = edge.source === incomingCenterID && anchorNodeID ? anchorNodeID : edge.source;
    const target = edge.target === incomingCenterID && anchorNodeID ? anchorNodeID : edge.target;
    if (source === target) continue;
    if (!nodeMap.has(source) || !nodeMap.has(target)) continue;

    const relationship = edge.relationship || "semantic_similarity";
    const id = edgeKey(source, target, relationship);
    const existing = edgeMap.get(id);
    if (!existing || edge.weight > existing.weight) {
      edgeMap.set(id, {
        id,
        source,
        target,
        relationship,
        weight: toFiniteWeight(edge.weight, existing?.weight ?? 0),
      });
    }
  }

  return {
    center: base.center,
    centerNodeID: base.centerNodeID,
    nodes: [...nodeMap.values()],
    edges: [...edgeMap.values()],
  };
}

export function createViewportTransform(viewport: CanvasViewport) {
  return `translate(${viewport.offsetX} ${viewport.offsetY}) scale(${viewport.scale})`;
}
