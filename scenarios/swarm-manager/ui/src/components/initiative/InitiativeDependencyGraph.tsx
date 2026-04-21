/**
 * InitiativeDependencyGraph
 *
 * Lightweight, read-only inline DAG showing dependency relationships
 * between an initiative's member backlog items. Uses ReactFlow + Dagre
 * for automatic layout.
 */

import { memo, useMemo } from "react";
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
import { useDetailSelectionStore } from "../../stores/detail-selection-store";
import { getStatusColorClasses } from "../../surfaces/graph/lib/status-colors";

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

interface InitiativeDependencyGraphProps {
  items: DagItem[];
}

// ---------------------------------------------------------------------------
// Mini node component
// ---------------------------------------------------------------------------

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

const MiniDagNode = memo(function MiniDagNode({ data }: NodeProps<Node<MiniNodeData>>) {
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);
  const statusColors = getStatusColorClasses(data.status);

  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-transparent !border-0 !w-0 !h-0" />
      <button
        type="button"
        onClick={() => selectBacklog(data.kind, data.name)}
        className="flex h-full w-full flex-col rounded-xl border border-slate-700/70 bg-slate-950/95 p-3 text-left transition-colors hover:border-slate-500/80 hover:bg-slate-900"
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

function InitiativeDependencyGraphInner({ items }: InitiativeDependencyGraphProps) {
  const { nodes, edges, hasEdges } = useMemo(() => {
    const itemKeys = new Set(items.map((i) => `${i.kind}/${i.name}`));
    const depthMap = computeDepthMap(items);
    const parentCounts = new Map<string, number>();
    const childCounts = new Map<string, number>();

    for (const item of items) {
      const key = `${item.kind}/${item.name}`;
      const scopedParents = (item.dependsOn ?? []).filter((dep) => itemKeys.has(dep));
      parentCounts.set(key, scopedParents.length);
      childCounts.set(key, 0);
    }
    for (const item of items) {
      for (const dep of item.dependsOn ?? []) {
        if (!itemKeys.has(dep)) continue;
        childCounts.set(dep, (childCounts.get(dep) ?? 0) + 1);
      }
    }

    // Build nodes
    const rawNodes: Node<MiniNodeData>[] = items.map((item) => ({
      id: `${item.kind}/${item.name}`,
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
        depth: depthMap.get(`${item.kind}/${item.name}`) ?? 0,
        parentCount: parentCounts.get(`${item.kind}/${item.name}`) ?? 0,
        childCount: childCounts.get(`${item.kind}/${item.name}`) ?? 0,
      },
    }));

    // Build edges (only within initiative scope)
    const rawEdges: Edge[] = [];
    for (const item of items) {
      const targetId = `${item.kind}/${item.name}`;
      for (const dep of item.dependsOn) {
        if (itemKeys.has(dep)) {
          rawEdges.push({
            id: `${dep}->${targetId}`,
            source: dep,
            target: targetId,
            style: { stroke: "rgb(100 116 139 / 0.6)", strokeWidth: 2 },
            markerEnd: { type: "arrowclosed" as const, color: "rgb(100 116 139 / 0.6)" },
          });
        }
      }
    }

    // Apply dagre layout
    const layoutedNodes = applyDagreLayout(rawNodes, rawEdges, "compact", "TB");

    return { nodes: layoutedNodes, edges: rawEdges, hasEdges: rawEdges.length > 0 };
  }, [items]);

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
