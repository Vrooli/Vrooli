/**
 * graphAdapter — pure converter from the proto `CodeGraph` (the shared
 * common.v1 envelope typescript-code-graph emits) to the render models the
 * module graph canvas, accessible list, and file/symbol drill-down consume.
 *
 * This is the ONLY typescript-code-graph-specific seam in the explorer
 * feature; the canvas / list / legend / filter components are shaped to be
 * shared with go-code-graph unchanged. Keep the proto-shape knowledge here.
 *
 * Wire-shape recap (see api/handlers/graph/adapter.go + the ts-morph sidecar):
 *   - File nodes:    kind = FILE (1), id "file:<path>", attributes.language.
 *   - Module nodes:  attributes.kind = "TS_NODE_KIND_MODULE", id
 *                    "ts_module:<path>". One module per .ts/.tsx file's
 *                    namespace — these are the dependency-graph nodes (the
 *                    TS analogue of go-code-graph's packages). The numeric
 *                    `kind` carries the TsNodeKind extension value (200),
 *                    which is outside the common NodeKind enum, so we key on
 *                    attributes.kind rather than the numeric field.
 *   - Symbol nodes:  attributes.kind = "TS_NODE_KIND_<COMPONENT|HOOK|CLASS|
 *                    INTERFACE|TYPE|FUNCTION|VAR|CONST|RE_EXPORT>", id
 *                    "ts_<kind>:<path>:<name>". Linked back to a File node by
 *                    matching `path` (TS has no file_id attribute).
 *                    leading_comments carries JSDoc / leading comments.
 *   - Dependency edges: kind = IMPORT (1) or RE_EXPORT (3), from/to are
 *                    module ids.
 *
 * Layout strategy: deterministic layered placement over the module dependency
 * graph (Kahn-style BFS), nodes sorted by path inside each layer so the same
 * graph always renders the same picture. Pure numbers; the canvas scales to
 * pixels at render time.
 */
import {
  NodeKind,
  EdgeKind,
} from "@vrooli/proto-types/common/v1/code_graph_pb";
import type {
  CodeGraph,
  CodeGraphNode,
} from "@vrooli/proto-types/common/v1/code_graph_pb";

const LAYER_SPACING = 220;
const ROW_SPACING = 72;

/** attributes.kind value the sidecar attaches to module (file-namespace) nodes. */
export const TS_MODULE_KIND = "TS_NODE_KIND_MODULE";

export interface PackageLayoutNode {
  readonly id: string;
  readonly label: string;
  readonly path: string;
  /** 0-indexed layer (column in horizontal layout). */
  readonly layer: number;
  /** Stable position within the layer (row). */
  readonly index: number;
  readonly x: number;
  readonly y: number;
  /** True when this module participates in an import cycle. */
  readonly inCycle: boolean;
}

export interface PackageLayoutEdge {
  readonly from: string;
  readonly to: string;
  readonly inCycle: boolean;
}

export interface GraphLayout {
  readonly nodes: readonly PackageLayoutNode[];
  readonly edges: readonly PackageLayoutEdge[];
  /** Every module path observed (pre-filter), sorted — feeds the filter bar. */
  readonly packages: readonly string[];
  /** Count of modules participating in at least one import cycle. */
  readonly cycleCount: number;
}

export interface SymbolEntry {
  readonly id: string;
  readonly name: string;
  /** TS symbol kind from attributes.kind: TS_NODE_KIND_* or "unknown". */
  readonly kind: string;
  readonly exported: boolean;
  /** Leading comments / JSDoc block attached to the declaration, verbatim. */
  readonly leadingComments: readonly string[];
}

export interface FileEntry {
  readonly id: string;
  readonly path: string;
  readonly name: string;
  readonly symbols: readonly SymbolEntry[];
}

/**
 * attributes.kind values for the symbol-level nodes that appear in the
 * file→symbol drill-down. Excludes TS_NODE_KIND_MODULE (the graph node) and
 * TS_NODE_KIND_UNSPECIFIED.
 */
export const TS_SYMBOL_KINDS = [
  "TS_NODE_KIND_COMPONENT",
  "TS_NODE_KIND_HOOK",
  "TS_NODE_KIND_CLASS",
  "TS_NODE_KIND_INTERFACE",
  "TS_NODE_KIND_TYPE",
  "TS_NODE_KIND_FUNCTION",
  "TS_NODE_KIND_VAR",
  "TS_NODE_KIND_CONST",
  "TS_NODE_KIND_RE_EXPORT",
] as const;

export function isFileNode(node: CodeGraphNode): boolean {
  return node.kind === NodeKind.FILE;
}

export function isSymbolNode(node: CodeGraphNode): boolean {
  const kind = node.attributes.kind;
  return kind !== undefined && (TS_SYMBOL_KINDS as readonly string[]).includes(kind);
}

/** A module (file-namespace) node — the TS analogue of a Go package. */
export function isModuleNode(node: CodeGraphNode): boolean {
  return node.attributes.kind === TS_MODULE_KIND;
}

/** True for the IMPORT / RE_EXPORT edges that form the module dependency graph. */
function isDependencyEdge(kind: EdgeKind): boolean {
  return kind === EdgeKind.IMPORT || kind === EdgeKind.RE_EXPORT;
}

export interface GraphSummary {
  readonly files: number;
  /** Module (file-namespace) node count — surfaced as "modules" in the UI. */
  readonly packages: number;
  readonly symbols: number;
  readonly imports: number;
}

/**
 * Count the headline node/edge populations for the stats header. Pure.
 */
export function summarizeGraph(graph: CodeGraph | undefined): GraphSummary {
  if (graph === undefined) {
    return { files: 0, packages: 0, symbols: 0, imports: 0 };
  }
  let files = 0;
  let packages = 0;
  let symbols = 0;
  for (const node of graph.nodes) {
    if (isFileNode(node)) files += 1;
    else if (isModuleNode(node)) packages += 1;
    else if (isSymbolNode(node)) symbols += 1;
  }
  let imports = 0;
  for (const edge of graph.edges) {
    if (isDependencyEdge(edge.kind)) imports += 1;
  }
  return { files, packages, symbols, imports };
}

/**
 * Detect every node id that participates in an import cycle. Uses an
 * iterative Tarjan SCC: any node in an SCC of size > 1, or with a self-edge,
 * is in a cycle. Pure and deterministic.
 */
export function detectCycleNodes(
  nodeIds: readonly string[],
  adjacency: ReadonlyMap<string, readonly string[]>,
): Set<string> {
  const index = new Map<string, number>();
  const lowlink = new Map<string, number>();
  const onStack = new Set<string>();
  const stack: string[] = [];
  const inCycle = new Set<string>();
  let counter = 0;

  // Iterative DFS to avoid blowing the call stack on large graphs.
  for (const root of nodeIds) {
    if (index.has(root)) continue;
    const work: Array<{ node: string; childIdx: number }> = [{ node: root, childIdx: 0 }];
    while (work.length > 0) {
      const frame = work[work.length - 1];
      if (frame === undefined) break;
      const { node } = frame;
      if (frame.childIdx === 0) {
        index.set(node, counter);
        lowlink.set(node, counter);
        counter += 1;
        stack.push(node);
        onStack.add(node);
      }
      const neighbors = adjacency.get(node) ?? [];
      if (frame.childIdx < neighbors.length) {
        const next = neighbors[frame.childIdx];
        frame.childIdx += 1;
        if (next === undefined) continue;
        if (!index.has(next)) {
          work.push({ node: next, childIdx: 0 });
        } else if (onStack.has(next)) {
          lowlink.set(node, Math.min(lowlink.get(node) ?? 0, index.get(next) ?? 0));
        }
        continue;
      }
      // Done with this node's children; propagate lowlink to parent.
      if (work.length > 1) {
        const parent = work[work.length - 2];
        if (parent !== undefined) {
          lowlink.set(parent.node, Math.min(lowlink.get(parent.node) ?? 0, lowlink.get(node) ?? 0));
        }
      }
      if ((lowlink.get(node) ?? 0) === (index.get(node) ?? 0)) {
        // Root of an SCC; pop it.
        const scc: string[] = [];
        for (;;) {
          const popped = stack.pop();
          if (popped === undefined) break;
          onStack.delete(popped);
          scc.push(popped);
          if (popped === node) break;
        }
        if (scc.length > 1) {
          for (const member of scc) inCycle.add(member);
        } else {
          // Single-member SCC is a cycle only with a self-edge.
          const self = adjacency.get(node) ?? [];
          if (self.includes(node)) inCycle.add(node);
        }
      }
      work.pop();
    }
  }
  return inCycle;
}

/**
 * Build the layered module-dependency layout. Pure: same inputs → same
 * outputs.
 *
 * @param graph the proto CodeGraph (may be undefined / empty).
 * @param packageFilter when non-empty, only modules whose path is in the set
 *                      are emitted; edges touching filtered-out modules drop.
 */
export function buildGraphLayout(
  graph: CodeGraph | undefined,
  packageFilter: ReadonlySet<string> = new Set(),
): GraphLayout {
  if (graph === undefined) {
    return { nodes: [], edges: [], packages: [], cycleCount: 0 };
  }

  const moduleNodes = graph.nodes.filter(isModuleNode);
  const moduleIds = new Set(moduleNodes.map((n) => n.id));

  // Module → module adjacency from IMPORT / RE_EXPORT edges.
  const outgoing = new Map<string, string[]>();
  const incoming = new Map<string, string[]>();
  for (const edge of graph.edges) {
    if (!isDependencyEdge(edge.kind)) continue;
    if (!moduleIds.has(edge.fromNodeId) || !moduleIds.has(edge.toNodeId)) continue;
    const outs = outgoing.get(edge.fromNodeId) ?? [];
    outs.push(edge.toNodeId);
    outgoing.set(edge.fromNodeId, outs);
    const ins = incoming.get(edge.toNodeId) ?? [];
    ins.push(edge.fromNodeId);
    incoming.set(edge.toNodeId, ins);
  }

  const cycleNodes = detectCycleNodes(
    moduleNodes.map((n) => n.id),
    outgoing,
  );

  // Layer assignment via Kahn-style BFS from modules with no incoming edges.
  const layerOf = new Map<string, number>();
  const queue: string[] = [];
  for (const node of moduleNodes) {
    const ins = incoming.get(node.id);
    if (ins === undefined || ins.length === 0) {
      layerOf.set(node.id, 0);
      queue.push(node.id);
    }
  }
  if (queue.length === 0) {
    // Pure cycle: seed everything at layer 0.
    for (const node of moduleNodes) {
      layerOf.set(node.id, 0);
      queue.push(node.id);
    }
  }
  while (queue.length > 0) {
    const head = queue.shift();
    if (head === undefined) break;
    const baseLayer = layerOf.get(head) ?? 0;
    for (const next of outgoing.get(head) ?? []) {
      const candidate = baseLayer + 1;
      const existing = layerOf.get(next);
      // Guard against runaway growth in cyclic graphs by capping the climb.
      if ((existing === undefined || candidate > existing) && candidate <= moduleNodes.length) {
        layerOf.set(next, candidate);
        queue.push(next);
      }
    }
  }

  const byLayer = new Map<number, CodeGraphNode[]>();
  for (const node of moduleNodes) {
    const layer = layerOf.get(node.id) ?? 0;
    const list = byLayer.get(layer) ?? [];
    list.push(node);
    byLayer.set(layer, list);
  }
  const sortedLayers = Array.from(byLayer.keys()).sort((a, b) => a - b);

  const filterActive = packageFilter.size > 0;
  const nodes: PackageLayoutNode[] = [];
  const includedIds = new Set<string>();
  for (const layer of sortedLayers) {
    const inLayer = (byLayer.get(layer) ?? [])
      .slice()
      .sort((a, b) => a.path.localeCompare(b.path));
    let index = 0;
    for (const node of inLayer) {
      if (filterActive && !packageFilter.has(node.path)) continue;
      includedIds.add(node.id);
      nodes.push({
        id: node.id,
        label: node.name.length > 0 ? node.name : node.path,
        path: node.path,
        layer,
        index,
        x: layer * LAYER_SPACING,
        y: index * ROW_SPACING,
        inCycle: cycleNodes.has(node.id),
      });
      index += 1;
    }
  }

  const edges: PackageLayoutEdge[] = [];
  for (const [from, outs] of outgoing.entries()) {
    if (!includedIds.has(from)) continue;
    for (const to of outs) {
      if (!includedIds.has(to)) continue;
      edges.push({ from, to, inCycle: cycleNodes.has(from) && cycleNodes.has(to) });
    }
  }

  const packages = moduleNodes
    .map((n) => n.path)
    .slice()
    .sort((a, b) => a.localeCompare(b));

  return { nodes, edges, packages, cycleCount: cycleNodes.size };
}

/**
 * Build the file → symbols index for the drill-down panel. Symbols are linked
 * to their file by matching `path` (TS has no file_id attribute). Files are
 * sorted by path; symbols within a file are sorted by name. Pure.
 */
export function buildFileIndex(graph: CodeGraph | undefined): FileEntry[] {
  if (graph === undefined) return [];

  const symbolsByPath = new Map<string, SymbolEntry[]>();
  for (const node of graph.nodes) {
    if (!isSymbolNode(node)) continue;
    if (node.path.length === 0) continue;
    const list = symbolsByPath.get(node.path) ?? [];
    list.push({
      id: node.id,
      name: node.name,
      kind: node.attributes.kind ?? "unknown",
      exported: node.attributes.exported === "true",
      leadingComments: node.leadingComments,
    });
    symbolsByPath.set(node.path, list);
  }

  const files: FileEntry[] = [];
  for (const node of graph.nodes) {
    if (!isFileNode(node)) continue;
    const symbols = (symbolsByPath.get(node.path) ?? [])
      .slice()
      .sort((a, b) => a.name.localeCompare(b.name));
    files.push({
      id: node.id,
      path: node.path,
      name: node.name.length > 0 ? node.name : node.path,
      symbols,
    });
  }
  files.sort((a, b) => a.path.localeCompare(b.path));
  return files;
}
