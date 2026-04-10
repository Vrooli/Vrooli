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
import { BACKLOG_STATUS_CHIP_COLORS } from "../../types";
import type { BacklogStatus } from "../../types";
import { useDetailSelectionStore } from "../../stores/detail-selection-store";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface DagItem {
  kind: string;
  name: string;
  title: string;
  status: BacklogStatus;
  dependsOn: string[];
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
  [key: string]: unknown;
}

const NODE_WIDTH = 220;
const NODE_HEIGHT = 70;

const MiniDagNode = memo(function MiniDagNode({ data }: NodeProps<Node<MiniNodeData>>) {
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);
  const chipColors = BACKLOG_STATUS_CHIP_COLORS[data.status] ?? "bg-slate-600/20 text-slate-300";

  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-transparent !border-0 !w-0 !h-0" />
      <button
        type="button"
        onClick={() => selectBacklog(data.kind, data.name)}
        className={`rounded-lg px-3 py-2 text-xs font-medium text-center leading-snug w-[${NODE_WIDTH}px] hover:brightness-125 transition-colors cursor-pointer line-clamp-3 ${chipColors}`}
        title={data.title}
      >
        {data.title}
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

  const graphHeight = Math.min(400, items.length * 70 + 80);

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
