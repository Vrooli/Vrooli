/**
 * GraphNode - Custom node renderer for all entity types.
 *
 * Shows: label, entity type badge, status indicator, kind badge.
 * Color-coded by entity type for visual distinction.
 */

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Lightbulb, Package, Zap, MessageSquare, Activity, Target } from "lucide-react";
import { cn } from "../../../lib/utils";

type EntityType = "backlog" | "scenario" | "execution" | "capture" | "agent-run" | "initiative";

interface GraphNodeData {
  label: string;
  status?: string;
  kind?: string;
  entityType?: EntityType;
  [key: string]: unknown;
}

const ENTITY_COLORS: Record<EntityType, { bg: string; border: string; badge: string; icon: React.ElementType }> = {
  backlog: {
    bg: "bg-slate-800/90",
    border: "border-cyan-500/40",
    badge: "bg-cyan-500/20 text-cyan-300",
    icon: Lightbulb,
  },
  scenario: {
    bg: "bg-slate-800/90",
    border: "border-violet-500/40",
    badge: "bg-violet-500/20 text-violet-300",
    icon: Package,
  },
  execution: {
    bg: "bg-slate-800/90",
    border: "border-amber-500/40",
    badge: "bg-amber-500/20 text-amber-300",
    icon: Zap,
  },
  capture: {
    bg: "bg-slate-800/90",
    border: "border-emerald-500/40",
    badge: "bg-emerald-500/20 text-emerald-300",
    icon: MessageSquare,
  },
  "agent-run": {
    bg: "bg-slate-800/90",
    border: "border-rose-500/40",
    badge: "bg-rose-500/20 text-rose-300",
    icon: Activity,
  },
  initiative: {
    bg: "bg-slate-800/90",
    border: "border-sky-500/40",
    badge: "bg-sky-500/20 text-sky-300",
    icon: Target,
  },
};

const STATUS_DOT_COLORS: Record<string, string> = {
  // Backlog statuses
  backlog: "bg-slate-400",
  researching: "bg-blue-400",
  ready: "bg-emerald-400",
  queued: "bg-amber-400",
  in_progress: "bg-cyan-400",
  completed: "bg-green-400",
  failed: "bg-red-400",
  archived: "bg-slate-500",
  // Scenario statuses
  running: "bg-green-400",
  stopped: "bg-slate-400",
  error: "bg-red-400",
  unknown: "bg-slate-500",
  // Execution statuses
  pending: "bg-slate-400",
  scheduled: "bg-blue-400",
  starting: "bg-cyan-400",
  needs_review: "bg-amber-400",
  validating: "bg-blue-400",
  needs_fixup: "bg-orange-400",
  canceled: "bg-slate-500",
  // Capture statuses
  classifying: "bg-blue-400",
  classified: "bg-emerald-400",
  // Agent run statuses
  needs_review_run: "bg-amber-400",
  complete: "bg-green-400",
  cancelled: "bg-slate-500",
};

const DEFAULT_ENTITY: EntityType = "backlog";

function GraphNodeComponent({ data, selected }: NodeProps) {
  const nodeData = data as GraphNodeData;
  const entityType = (nodeData.entityType ?? DEFAULT_ENTITY) as EntityType;
  const colors = ENTITY_COLORS[entityType] ?? ENTITY_COLORS[DEFAULT_ENTITY];
  const Icon = colors.icon;
  const statusDot = STATUS_DOT_COLORS[nodeData.status ?? ""] ?? "bg-slate-500";

  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-slate-500 !border-slate-400 !w-2 !h-2" />
      <div
        className={cn(
          "rounded-lg border px-3 py-2 shadow-lg backdrop-blur-sm min-w-[140px] max-w-[220px]",
          "transition-all duration-150",
          colors.bg,
          colors.border,
          selected && "ring-2 ring-cyan-400/60 border-cyan-400/70 shadow-cyan-500/20",
          Boolean(nodeData.pulsing) && "graph-node-pulse",
        )}
        onAnimationEnd={(e) => {
          if (e.animationName === "graph-node-pulse") {
            // Remove pulse class after animation completes — no React re-render needed,
            // the class will be re-applied on next WS update.
            e.currentTarget.classList.remove("graph-node-pulse");
          }
        }}
      >
        {/* Header: icon + entity type */}
        <div className="flex items-center gap-1.5 mb-1">
          <Icon className="h-3 w-3 shrink-0 text-slate-400" />
          <span className={cn("rounded-full px-1.5 py-0 text-[10px] font-medium", colors.badge)}>
            {entityType}
          </span>
          {nodeData.kind && (
            <span className="rounded-full bg-slate-700/80 px-1.5 py-0 text-[10px] text-slate-400">
              {nodeData.kind}
            </span>
          )}
        </div>

        {/* Label */}
        <p className="text-xs font-medium text-slate-100 leading-tight line-clamp-2 break-words">
          {nodeData.label}
        </p>

        {/* Status */}
        {nodeData.status && (
          <div className="flex items-center gap-1.5 mt-1.5">
            <span className={cn("h-1.5 w-1.5 rounded-full shrink-0", statusDot)} />
            <span className="text-[10px] text-slate-400 truncate">
              {nodeData.status.replace(/_/g, " ")}
            </span>
          </div>
        )}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-500 !border-slate-400 !w-2 !h-2" />
    </>
  );
}

export const GraphNode = memo(GraphNodeComponent);
