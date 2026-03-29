/**
 * ClusterNode - Initiative cluster node for topology lens.
 *
 * Collapsed: initiative title + rollup badge (total/completed/in_progress/failed/pending).
 * Expanded: initiative title header with children rendered inside parent boundary by React Flow.
 */

import { memo, useCallback } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Target, ChevronRight, ChevronDown } from "lucide-react";
import { cn } from "../../../lib/utils";
import type { RollupCounts } from "../lib/clustering-utils";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { ClusterGraphNodeData } from "../types";

function RollupBadge({ rollup }: { rollup: RollupCounts }) {
  return (
    <div className="flex items-center gap-1.5 text-[10px]" data-testid="rollup-badge">
      <span className="text-slate-400">{rollup.total} total</span>
      {rollup.completed > 0 && (
        <span className="rounded-full bg-green-500/20 px-1.5 text-green-300">{rollup.completed}</span>
      )}
      {rollup.in_progress > 0 && (
        <span className="rounded-full bg-cyan-500/20 px-1.5 text-cyan-300">{rollup.in_progress}</span>
      )}
      {rollup.failed > 0 && (
        <span className="rounded-full bg-red-500/20 px-1.5 text-red-300">{rollup.failed}</span>
      )}
      {rollup.pending > 0 && (
        <span className="rounded-full bg-slate-500/20 px-1.5 text-slate-400">{rollup.pending}</span>
      )}
    </div>
  );
}

function ClusterNodeComponent({ id, data, selected }: NodeProps) {
  const nodeData = data as ClusterGraphNodeData;
  const isCollapsed = nodeData.collapsed;
  const isUnassigned = nodeData.isUnassigned ?? false;
  const toggleTopologyCluster = useGraphUIStore((s) => s.toggleTopologyCluster);

  const handleChevronClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      toggleTopologyCluster(id);
    },
    [id, toggleTopologyCluster],
  );

  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-slate-500 !border-slate-400 !w-2 !h-2" />
      <div
        className={cn(
          "rounded-xl border px-3 py-2 shadow-lg backdrop-blur-sm",
          "transition-all duration-150",
          isUnassigned
            ? "bg-slate-800/80 border-slate-600/40"
            : "bg-slate-800/90 border-sky-500/40",
          selected && "ring-2 ring-cyan-400/60 border-cyan-400/70",
          !isCollapsed && "min-w-[240px] min-h-[100px]",
        )}
        data-testid="cluster-node"
      >
        {/* Header */}
        <div className="flex items-center gap-1.5 mb-1">
          <button
            type="button"
            onClick={handleChevronClick}
            className="flex items-center justify-center rounded p-0.5 hover:bg-slate-600/50 transition-colors cursor-pointer"
            aria-label={isCollapsed ? "Expand cluster" : "Collapse cluster"}
            data-testid="cluster-toggle"
          >
            {isCollapsed ? (
              <ChevronRight className="h-3 w-3 text-slate-400" />
            ) : (
              <ChevronDown className="h-3 w-3 text-slate-400" />
            )}
          </button>
          <Target className="h-3 w-3 shrink-0 text-sky-400" />
          <span className={cn(
            "rounded-full px-1.5 py-0 text-[10px] font-medium",
            isUnassigned ? "bg-slate-700/80 text-slate-400" : "bg-sky-500/20 text-sky-300",
          )}>
            {isUnassigned ? "unassigned" : "initiative"}
          </span>
        </div>

        {/* Label */}
        <p className="text-xs font-medium text-slate-100 leading-tight line-clamp-2 break-words">
          {nodeData.label}
        </p>

        {/* Rollup badge (collapsed only) */}
        {isCollapsed && nodeData.rollup && (
          <div className="mt-1.5">
            <RollupBadge rollup={nodeData.rollup} />
          </div>
        )}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-500 !border-slate-400 !w-2 !h-2" />
    </>
  );
}

export const ClusterNode = memo(ClusterNodeComponent);
