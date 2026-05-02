/**
 * PhaseNode
 *
 * Custom xyflow node for the operating-mode phase graph. Compact card with
 * the phase title, snake_case ID, and ring color reflecting role in the graph
 * (start, terminal, currently selected) and whether the phase writes the repo.
 */

import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { cn } from "../../../lib/utils";

export interface PhaseNodeData extends Record<string, unknown> {
  phase: string;
  title: string;
  isStart: boolean;
  isTerminal: boolean;
  writesRepo: boolean;
  selected: boolean;
}

export type PhaseNodeType = Node<PhaseNodeData, "phase">;

export function PhaseNode({ data }: NodeProps<PhaseNodeType>) {
  const ringClass = data.selected
    ? "ring-2 ring-cyan-400/80"
    : data.isStart
      ? "ring-2 ring-emerald-400/60"
      : data.isTerminal
        ? "ring-2 ring-violet-400/60"
        : "";

  return (
    <div
      className={cn(
        "min-w-[140px] rounded-lg border border-slate-700 bg-slate-900/95 px-3 py-2 text-left shadow-md backdrop-blur-sm",
        ringClass,
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-slate-500" />
      <div className="flex items-center gap-2">
        <p className="text-sm font-semibold text-slate-100">{data.title}</p>
        {data.writesRepo && (
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-emerald-400" aria-label="writes repo" />
        )}
      </div>
      <code className="mt-0.5 block text-[10px] font-mono text-slate-400">{data.phase}</code>
      <div className="mt-1 flex flex-wrap gap-1">
        {data.isStart && (
          <span className="rounded-full bg-emerald-500/15 px-1.5 py-0.5 text-[9px] font-medium text-emerald-300">
            start
          </span>
        )}
        {data.isTerminal && (
          <span className="rounded-full bg-violet-500/15 px-1.5 py-0.5 text-[9px] font-medium text-violet-300">
            terminal
          </span>
        )}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-500" />
    </div>
  );
}
