/**
 * InitiativeDependencyGraph
 *
 * Lightweight, read-only inline DAG showing dependency relationships
 * between an initiative's member backlog items. Uses ReactFlow + Dagre
 * for automatic layout.
 */

import { memo, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import {
  ReactFlow,
  ReactFlowProvider,
  Handle,
  Position,
  type Node,
  type Edge,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { applyDagreLayout } from "../../surfaces/graph/lib/layout-utils";
import { cn } from "../../lib/utils";
import { formatDisplayText } from "../../lib/format-utils";
import type { BacklogStatus } from "../../types";
import { getStatusColorClasses } from "../../surfaces/graph/lib/status-colors";
import { backlogDetailPath } from "../../app/routes/route-paths";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface DagItem {
  kind: string;
  name: string;
  title: string;
  status: BacklogStatus;
  dependsOn: string[];
  priority?: number;
  archivedAt?: string;
  missing?: boolean;
}

/**
 * Overlay describes the mutations a proposal would make against the current
 * graph. Used by ProposalReview to render a before/after preview without
 * actually mutating the underlying items.
 *
 * Each set identifies nodes/edges by canonical `"kind/name"` string so
 * `proposals` stay wire-shape-aligned with the rest of the app.
 */
export interface InitiativeGraphOverlay {
  /** Node IDs ("kind/name") that would be created. */
  addedNodeIds?: string[];
  /** Node IDs that would be archived. */
  archivedNodeIds?: string[];
  /** Node IDs that would move to another initiative (detaching them here). */
  movedOutNodeIds?: string[];
  /** Nodes whose status would change, keyed by node id → new status. */
  statusChanges?: Record<string, BacklogStatus>;
  /** Edges "from|to" that would be added. */
  addedEdges?: Array<{ from: string; to: string }>;
  /** Edges "from|to" that would be removed. */
  removedEdges?: Array<{ from: string; to: string }>;
  /** Items that would be spawned as additional placeholder nodes for the
   *  overlay (title + kind required so the preview renders something). */
  addedNodes?: Array<{
    id: string;
    kind: string;
    name: string;
    title: string;
    status?: BacklogStatus;
  }>;
}

interface InitiativeDependencyGraphProps {
  items: DagItem[];
  /**
   * Optional overlay used to render a before/after preview of a proposed
   * mutation list. When set, nodes gain a diff badge (added/archived/moved/
   * status changed) and edges render with dashed strokes for adds / faded
   * strokes for removals.
   */
  overlay?: InitiativeGraphOverlay;
}

// ---------------------------------------------------------------------------
// Mini node component
// ---------------------------------------------------------------------------

type NodeDiff = "added" | "archived" | "moved" | "status_change" | null;

interface MiniNodeData {
  title: string;
  status: BacklogStatus;
  kind: string;
  name: string;
  priority: number;
  archivedAt?: string;
  missing: boolean;
  depth: number;
  parentCount: number;
  childCount: number;
  /** When an overlay is active, describes what the mutation would do to this node. */
  diff: NodeDiff;
  /** If diff === "status_change", the target status after the mutation. */
  pendingStatus?: BacklogStatus;
  [key: string]: unknown;
}

const NODE_WIDTH = 272;
const NODE_HEIGHT = 142;

function computeDepthMap(items: DagItem[]): Map<string, number> {
  const itemKeys = new Set(items.map((item) => `${item.kind}/${item.name}`));
  const depsMap = new Map<string, string[]>();
  const memo = new Map<string, number>();

  for (const item of items) {
    const key = `${item.kind}/${item.name}`;
    depsMap.set(key, (item.dependsOn ?? []).filter((dep) => itemKeys.has(dep)));
  }

  const visit = (key: string, visiting = new Set<string>()): number => {
    if (memo.has(key)) return memo.get(key) ?? 0;
    if (visiting.has(key)) return 0;
    visiting.add(key);
    const deps = depsMap.get(key) ?? [];
    const depth = deps.length === 0 ? 0 : Math.max(...deps.map((dep) => visit(dep, visiting))) + 1;
    visiting.delete(key);
    memo.set(key, depth);
    return depth;
  };

  for (const key of depsMap.keys()) {
    visit(key);
  }
  return memo;
}

const DIFF_BADGE: Record<Exclude<NodeDiff, null>, { label: string; classes: string }> = {
  added: { label: "+ Added", classes: "border-emerald-500/40 bg-emerald-500/10 text-emerald-200" },
  archived: { label: "Archive", classes: "border-amber-500/40 bg-amber-500/10 text-amber-200" },
  moved: { label: "Move out", classes: "border-sky-500/40 bg-sky-500/10 text-sky-200" },
  status_change: { label: "Status", classes: "border-cyan-500/40 bg-cyan-500/10 text-cyan-200" },
};

const MiniDagNode = memo(function MiniDagNode({ data }: NodeProps<Node<MiniNodeData>>) {
  const navigate = useNavigate();
  const statusColors = getStatusColorClasses(data.status);
  const diffBadge = data.diff ? DIFF_BADGE[data.diff] : null;
  const borderDiff =
    data.diff === "added"
      ? "border-dashed border-emerald-400/70"
      : data.diff === "archived"
        ? "border-amber-500/60"
        : data.diff === "moved"
          ? "border-sky-500/60"
          : data.diff === "status_change"
            ? "border-cyan-500/60"
            : "border-slate-700/70";

  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-transparent !border-0 !w-0 !h-0" />
      <button
        type="button"
        onClick={() => navigate(backlogDetailPath(data.kind, data.name))}
        className={cn(
          "flex h-full w-full flex-col rounded-xl border bg-slate-950/95 p-3 text-left transition-colors hover:border-slate-500/80 hover:bg-slate-900",
          borderDiff,
        )}
        title={data.title}
      >
        <div className="flex items-start justify-between gap-3">
          <p className="line-clamp-3 flex-1 text-sm font-semibold leading-snug text-slate-100">
            {data.title}
          </p>
          <span className={cn("rounded-full border px-2 py-0.5 text-[10px] font-medium", statusColors.background, statusColors.border, statusColors.text)}>
            {formatDisplayText(data.status)}
          </span>
        </div>

        <div className="mt-2 flex flex-wrap items-center gap-2 text-[10px]">
          {diffBadge && (
            <span className={cn("rounded-full border px-2 py-0.5 font-medium", diffBadge.classes)}>
              {diffBadge.label}
              {data.diff === "status_change" && data.pendingStatus ? ` → ${formatDisplayText(data.pendingStatus)}` : ""}
            </span>
          )}
          {data.priority > 0 && (
            <span className="rounded-full bg-slate-800 px-2 py-0.5 font-medium text-slate-200">
              P{data.priority}
            </span>
          )}
          {data.depth > 0 && (
            <span className="rounded-full bg-slate-800 px-2 py-0.5 font-medium text-slate-400">
              Layer {data.depth}
            </span>
          )}
          {data.archivedAt && (
            <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 font-medium text-amber-300">
              Archived
            </span>
          )}
          {data.missing && (
            <span className="rounded-full border border-red-500/30 bg-red-500/10 px-2 py-0.5 font-medium text-red-300">
              Missing
            </span>
          )}
        </div>

        <div className="mt-2 flex flex-wrap gap-3 text-[11px] text-slate-500">
          <span>{data.kind}/{data.name}</span>
          {data.parentCount > 0 && <span className="text-amber-400/80">blocked by {data.parentCount}</span>}
          {data.childCount > 0 && <span>unblocks {data.childCount}</span>}
          {data.parentCount === 0 && data.childCount === 0 && <span>isolated in initiative</span>}
        </div>
      </button>
      <Handle type="source" position={Position.Bottom} className="!bg-transparent !border-0 !w-0 !h-0" />
    </>
  );
});

const nodeTypes = { miniDag: MiniDagNode };

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

function InitiativeDependencyGraphInner({ items, overlay }: InitiativeDependencyGraphProps) {
  const { nodes, edges, hasEdges } = useMemo(() => {
    // Build combined item list including overlay-added placeholder nodes so
    // depth / edge resolution account for them.
    const overlayAdded = overlay?.addedNodes ?? [];
    const effectiveItems: DagItem[] = [
      ...items,
      ...overlayAdded.map((n) => ({
        kind: n.kind,
        name: n.name,
        title: n.title,
        status: n.status ?? "backlog",
        dependsOn: [],
        priority: 0,
      })),
    ];

    const itemKeys = new Set(effectiveItems.map((i) => `${i.kind}/${i.name}`));
    const depthMap = computeDepthMap(effectiveItems);
    const parentCounts = new Map<string, number>();
    const childCounts = new Map<string, number>();

    for (const item of effectiveItems) {
      const key = `${item.kind}/${item.name}`;
      const scopedParents = (item.dependsOn ?? []).filter((dep) => itemKeys.has(dep));
      parentCounts.set(key, scopedParents.length);
      childCounts.set(key, 0);
    }
    for (const item of effectiveItems) {
      for (const dep of item.dependsOn ?? []) {
        if (!itemKeys.has(dep)) continue;
        childCounts.set(dep, (childCounts.get(dep) ?? 0) + 1);
      }
    }

    const addedNodeIds = new Set(overlay?.addedNodeIds ?? []);
    for (const n of overlayAdded) addedNodeIds.add(n.id);
    const archivedNodeIds = new Set(overlay?.archivedNodeIds ?? []);
    const movedOutNodeIds = new Set(overlay?.movedOutNodeIds ?? []);
    const statusChanges = overlay?.statusChanges ?? {};

    // Build nodes
    const rawNodes: Node<MiniNodeData>[] = effectiveItems.map((item) => {
      const id = `${item.kind}/${item.name}`;
      let diff: NodeDiff = null;
      if (addedNodeIds.has(id)) diff = "added";
      else if (archivedNodeIds.has(id)) diff = "archived";
      else if (movedOutNodeIds.has(id)) diff = "moved";
      else if (statusChanges[id]) diff = "status_change";
      return {
        id,
        type: "miniDag",
        position: { x: 0, y: 0 },
        width: NODE_WIDTH,
        height: NODE_HEIGHT,
        data: {
          title: item.title,
          status: item.status,
          kind: item.kind,
          name: item.name,
          priority: item.priority ?? 0,
          archivedAt: item.archivedAt,
          missing: item.missing ?? false,
          depth: depthMap.get(id) ?? 0,
          parentCount: parentCounts.get(id) ?? 0,
          childCount: childCounts.get(id) ?? 0,
          diff,
          pendingStatus: statusChanges[id],
        },
      };
    });

    const removedEdgeSet = new Set(
      (overlay?.removedEdges ?? []).map((e) => `${e.from}->${e.to}`),
    );

    // Build edges (only within initiative scope, plus overlay additions).
    const rawEdges: Edge[] = [];
    const pushEdge = (from: string, to: string, variant: "base" | "added" | "removed") => {
      const style =
        variant === "added"
          ? { stroke: "rgb(52 211 153 / 0.85)", strokeWidth: 2, strokeDasharray: "6 4" }
          : variant === "removed"
            ? { stroke: "rgb(248 113 113 / 0.55)", strokeWidth: 2, strokeDasharray: "2 3" }
            : { stroke: "rgb(100 116 139 / 0.6)", strokeWidth: 2 };
      const color =
        variant === "added"
          ? "rgb(52 211 153 / 0.9)"
          : variant === "removed"
            ? "rgb(248 113 113 / 0.6)"
            : "rgb(100 116 139 / 0.6)";
      rawEdges.push({
        id: `${from}->${to}:${variant}`,
        source: from,
        target: to,
        style,
        markerEnd: { type: "arrowclosed" as const, color },
        data: { variant },
      });
    };

    for (const item of effectiveItems) {
      const targetId = `${item.kind}/${item.name}`;
      for (const dep of item.dependsOn ?? []) {
        if (!itemKeys.has(dep)) continue;
        const key = `${dep}->${targetId}`;
        const variant = removedEdgeSet.has(key) ? "removed" : "base";
        pushEdge(dep, targetId, variant);
      }
    }
    for (const e of overlay?.addedEdges ?? []) {
      if (itemKeys.has(e.from) && itemKeys.has(e.to)) {
        pushEdge(e.from, e.to, "added");
      }
    }

    // Apply dagre layout
    const layoutedNodes = applyDagreLayout(rawNodes, rawEdges, "compact", "TB");

    return { nodes: layoutedNodes, edges: rawEdges, hasEdges: rawEdges.length > 0 };
  }, [items, overlay]);

  if (!hasEdges) {
    return (
      <p className="text-xs text-slate-500 italic">No dependencies between items</p>
    );
  }

  const graphHeight = Math.min(720, items.length * 150 + 100);

  return (
    <div style={{ height: graphHeight }} className="rounded-lg border border-slate-700/40 bg-slate-900/30">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        nodesDraggable={false}
        nodesConnectable={false}
        nodesFocusable={false}
        edgesFocusable={false}
        proOptions={{ hideAttribution: true }}
        className="[&_.react-flow__renderer]:!bg-transparent"
      />
    </div>
  );
}

export const InitiativeDependencyGraph = memo(function InitiativeDependencyGraph(
  props: InitiativeDependencyGraphProps,
) {
  return (
    <ReactFlowProvider>
      <InitiativeDependencyGraphInner {...props} />
    </ReactFlowProvider>
  );
});
