/**
 * graphAdapter — pure converter from a proto `GraphSnapshot` (+ optional
 * conflict overlay) to the layout model the canvas and accessible list
 * consume.
 *
 * The adapter is deliberately framework-agnostic and side-effect-free so it
 * unit-tests trivially against the proto fixtures and so renderer choices
 * (SVG today, viz-js tomorrow) can swap without churning the data path.
 *
 * Layout strategy: deterministic layered placement.
 *   - Layer 0 contains every file with no internal imports declared inside
 *     the snapshot; subsequent layers grow by BFS expansion through the
 *     `imports` edges.
 *   - Inside a layer, nodes are sorted lexicographically by path so the
 *     same snapshot always renders the same picture.
 *   - Coordinates are pure numbers; the canvas component decides the unit
 *     scale at render time.
 *
 * No imports from React, no global state, no DOM. The only dependency is
 * the proto-generated `GraphSnapshot` / `Conflict` shapes.
 */
import type { GraphSnapshot } from "@vrooli/proto-types/architecture-cartographer/v1/graph/graph_pb";
import type { Conflict } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

import { severityToLevel } from "../../conflicts/severity";
import type { SeverityLevel } from "../../../components/SeverityBadge";

export interface GraphLayoutNode {
  readonly id: string;
  readonly label: string;
  readonly path: string;
  readonly domain: string;
  /** 0-indexed layer (column in horizontal layout). */
  readonly layer: number;
  /** Stable position within the layer (row). */
  readonly index: number;
  /** Pure layout coordinate; canvas scales to pixels at render time. */
  readonly x: number;
  readonly y: number;
  /** Highest-severity active conflict touching this node, if any. */
  readonly conflictSeverity?: SeverityLevel;
}

export interface GraphLayoutEdge {
  readonly from: string;
  readonly to: string;
}

export interface GraphLayout {
  readonly nodes: readonly GraphLayoutNode[];
  readonly edges: readonly GraphLayoutEdge[];
  /** Distinct domain keys observed in the snapshot, sorted. */
  readonly domains: readonly string[];
}

const LAYER_SPACING = 220;
const ROW_SPACING = 72;

/**
 * Build the layered layout. Pure: same inputs → same outputs.
 *
 * @param snapshot the proto GraphSnapshot (may have empty fields).
 * @param conflicts the active conflicts whose `locations` should overlay onto
 *                  matching files.
 * @param domainFilter when non-empty, only nodes whose domain is in the
 *                     set are emitted; edges to filtered-out nodes drop.
 */
export function buildGraphLayout(
  snapshot: GraphSnapshot | undefined,
  conflicts: readonly Conflict[],
  domainFilter: ReadonlySet<string> = new Set(),
): GraphLayout {
  if (snapshot === undefined) {
    return { nodes: [], edges: [], domains: [] };
  }

  const fileById = new Map<string, GraphSnapshot["files"][number]>();
  for (const file of snapshot.files) {
    fileById.set(file.id, file);
  }

  // Map file path → domain via the package directory. Cartographer's proto
  // doesn't (yet) attach a domain to the file directly, so we derive it
  // from the package's top directory segment when available.
  const packageById = new Map<string, GraphSnapshot["packages"][number]>();
  for (const pkg of snapshot.packages) {
    packageById.set(pkg.id, pkg);
  }
  const domainForFile = (file: GraphSnapshot["files"][number]): string => {
    const pkg = packageById.get(file.packageId);
    if (pkg === undefined || pkg.directory.length === 0) return "";
    // First path segment is the conventional domain root in cartographer's
    // own layout (e.g., "api/internal/<domain>/…").
    const segments = pkg.directory.split("/").filter((s) => s.length > 0);
    return segments.length > 0 ? (segments[0] ?? "") : "";
  };

  // Build a map of path → highest-severity conflict level.
  const conflictByPath = new Map<string, SeverityLevel>();
  const severityRank: Record<SeverityLevel, number> = {
    info: 0,
    low: 1,
    medium: 2,
    high: 3,
    critical: 4,
  };
  for (const conflict of conflicts) {
    const level = severityToLevel(conflict.severity);
    for (const loc of conflict.locations) {
      const prior = conflictByPath.get(loc);
      if (prior === undefined || severityRank[level] > severityRank[prior]) {
        conflictByPath.set(loc, level);
      }
    }
  }

  // Adjacency: file id → set of file ids it imports (resolved through
  // package edges). We treat the file as importing every file in the
  // imported package — sufficient for layered ordering at v1.
  const filesByPackage = new Map<string, string[]>();
  for (const file of snapshot.files) {
    const list = filesByPackage.get(file.packageId) ?? [];
    list.push(file.id);
    filesByPackage.set(file.packageId, list);
  }
  const outgoing = new Map<string, string[]>();
  const incoming = new Map<string, string[]>();
  for (const edge of snapshot.imports) {
    if (!fileById.has(edge.from)) continue;
    const targets = filesByPackage.get(edge.toPackageId) ?? [];
    for (const targetId of targets) {
      if (targetId === edge.from) continue;
      const outs = outgoing.get(edge.from) ?? [];
      outs.push(targetId);
      outgoing.set(edge.from, outs);
      const ins = incoming.get(targetId) ?? [];
      ins.push(edge.from);
      incoming.set(targetId, ins);
    }
  }

  // Assign layers via Kahn-style BFS from nodes with no internal incoming
  // edges. Cycles fall through and pick up the lowest layer they hit.
  const layerOf = new Map<string, number>();
  const queue: string[] = [];
  for (const file of snapshot.files) {
    const ins = incoming.get(file.id);
    if (ins === undefined || ins.length === 0) {
      layerOf.set(file.id, 0);
      queue.push(file.id);
    }
  }
  // Defensive: if every node had incoming edges (pure cycle), seed with all.
  if (queue.length === 0) {
    for (const file of snapshot.files) {
      layerOf.set(file.id, 0);
      queue.push(file.id);
    }
  }
  while (queue.length > 0) {
    const head = queue.shift();
    if (head === undefined) break;
    const baseLayer = layerOf.get(head) ?? 0;
    const outs = outgoing.get(head) ?? [];
    for (const next of outs) {
      const candidate = baseLayer + 1;
      const existing = layerOf.get(next);
      if (existing === undefined || candidate > existing) {
        layerOf.set(next, candidate);
        queue.push(next);
      }
    }
  }

  // Group by layer, sort each layer by path for determinism, and compute
  // coordinates.
  const byLayer = new Map<number, GraphSnapshot["files"][number][]>();
  for (const file of snapshot.files) {
    const layer = layerOf.get(file.id) ?? 0;
    const list = byLayer.get(layer) ?? [];
    list.push(file);
    byLayer.set(layer, list);
  }
  const sortedLayers = Array.from(byLayer.keys()).sort((a, b) => a - b);

  const domains = new Set<string>();
  const filterActive = domainFilter.size > 0;
  const nodes: GraphLayoutNode[] = [];
  const includedIds = new Set<string>();
  for (const layer of sortedLayers) {
    const filesInLayer = (byLayer.get(layer) ?? []).slice().sort((a, b) =>
      a.path.localeCompare(b.path),
    );
    let index = 0;
    for (const file of filesInLayer) {
      const domain = domainForFile(file);
      domains.add(domain);
      if (filterActive && !domainFilter.has(domain)) continue;
      includedIds.add(file.id);
      nodes.push({
        id: file.id,
        label: file.path.split("/").pop() ?? file.path,
        path: file.path,
        domain,
        layer,
        index,
        x: layer * LAYER_SPACING,
        y: index * ROW_SPACING,
        conflictSeverity: conflictByPath.get(file.path),
      });
      index += 1;
    }
  }

  const edges: GraphLayoutEdge[] = [];
  for (const [from, outs] of outgoing.entries()) {
    if (!includedIds.has(from)) continue;
    for (const to of outs) {
      if (!includedIds.has(to)) continue;
      edges.push({ from, to });
    }
  }

  return {
    nodes,
    edges,
    domains: Array.from(domains).sort(),
  };
}
