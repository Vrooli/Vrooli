/**
 * GraphNode - Custom node renderer for all entity types.
 *
 * Shape varies by entity type (diamond, rectangle, hexagon, circle, pentagon, octagon, pill).
 * Color encodes STATUS via both fill and border for instant scannability.
 *
 * @see lib/entity-shapes.ts for shape mapping
 * @see lib/status-colors.ts for color mapping
 */

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Lightbulb, Package, Zap, MessageSquare, Activity, Target } from "lucide-react";
import { cn } from "../../../lib/utils";
import { BACKLOG_KIND_ICONS } from "../../../types";
import type { BacklogKind } from "../../../types";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { GraphEntityType, GraphNodeData } from "../types";
import { ActionableBadge, StatusBadge } from "./StatusBadge";
import { getShapeClasses, getShapeDimensions, needsContentCounterRotation, usesClipPath } from "../lib/entity-shapes";
import { getStatusColorClasses, isActionableBacklogStatus } from "../lib/status-colors";

const ENTITY_ICONS: Record<GraphEntityType, React.ElementType> = {
  backlog: Lightbulb,
  scenario: Package,
  execution: Zap,
  "agent-activity": Activity,
  capture: MessageSquare,
  "agent-run": Activity,
  initiative: Target,
};

/** Shapes that need fixed, equal-sided sizing. */
const FIXED_SIZE_SHAPES = new Set<GraphEntityType>(["backlog", "initiative", "execution", "capture", "agent-run", "agent-activity"]);

const DEFAULT_ENTITY: GraphEntityType = "backlog";

function GraphNodeComponent({ id, data }: NodeProps) {
  const nodeData = data as GraphNodeData;
  const lens = useGraphDataStore((s) => s.lens);
  const isSelected = useGraphUIStore((s) => s.selectedNodeId === id);
  const entityType = nodeData.entityType ?? DEFAULT_ENTITY;
  const backlogKindIcon = entityType === "backlog" && nodeData.kind
    ? BACKLOG_KIND_ICONS[nodeData.kind as BacklogKind]
    : undefined;
  const Icon = backlogKindIcon ?? ENTITY_ICONS[entityType] ?? ENTITY_ICONS[DEFAULT_ENTITY];
  const shapeClass = getShapeClasses(entityType);
  const statusColors = getStatusColorClasses(nodeData.status);
  const counterRotate = needsContentCounterRotation(entityType);
  const isClipped = usesClipPath(entityType);
  const dims = getShapeDimensions(entityType);
  const isFixedSize = FIXED_SIZE_SHAPES.has(entityType);

  return (
    <>
      {/* PERF: Minimal handles — only needed for React Flow edge routing.
          Invisible (opacity-0) since graph is read-only (nodesConnectable=false). */}
      <Handle type="target" position={Position.Top} className="!opacity-0 !w-1 !h-1 !min-w-0 !min-h-0 !border-0 !p-0" />
      {/* Outer wrapper: handles drop-shadow for clipped shapes */}
      <div
        className={cn(
          "relative",
          isSelected && isClipped && "drop-shadow-[0_0_8px_rgba(34,211,238,0.7)]",
          Boolean(nodeData.pulsing) && "graph-node-pulse",
        )}
        onAnimationEnd={(e) => {
          if (e.animationName === "graph-node-pulse") {
            e.currentTarget.classList.remove("graph-node-pulse");
          }
        }}
      >
        <div
          className={cn(
            "flex items-center justify-center",
            shapeClass,
            statusColors.background,
            // Border: thicker cyan when selected, normal status border otherwise
            isSelected
              ? "border-[3px] border-cyan-400"
              : cn("border-2", statusColors.border),
            // Selection ring — only for non-clipped shapes (clip-path hides box-shadow/ring)
            isSelected && !isClipped && "ring-2 ring-cyan-400/50 shadow-lg shadow-cyan-500/30",
            // Sizing
            isFixedSize
              ? ""
              : "min-w-[120px] max-w-[180px] px-3 py-2",
          )}
          style={isFixedSize ? { width: dims.width, height: dims.height } : undefined}
        >
          <div className={cn(counterRotate && "-rotate-45", "flex flex-col items-center gap-0.5 w-full px-1")}>
            {/* Entity type badge with icon */}
            <div className="flex items-center gap-1">
              <Icon className={cn("h-3 w-3 shrink-0", statusColors.text)} />
              <span className={cn("text-[9px] font-medium uppercase tracking-wide", statusColors.text)}>
                {entityType === "agent-activity" ? "activity" : entityType === "agent-run" ? "run" : entityType}
              </span>
            </div>

            {/* Label */}
            <p className={cn(
              "text-[10px] font-medium leading-tight text-center break-words",
              statusColors.text,
              isFixedSize ? "line-clamp-2 max-w-[110px]" : "line-clamp-2",
            )}>
              {nodeData.label}
            </p>

            {/* Status text */}
            {nodeData.status && (
              <span className="text-[8px] text-slate-400 truncate max-w-full">
                {nodeData.status.replace(/_/g, " ")}
              </span>
            )}
          </div>
        </div>
        {/* Badges render AFTER the shape div so they paint above the backdrop-blur */}
        {lens === "topology" && entityType === "backlog" && "activeExecutionStatus" in nodeData && (
          <StatusBadge executionStatus={(nodeData as { activeExecutionStatus?: string }).activeExecutionStatus} />
        )}
        {lens === "topology" && entityType === "backlog" && nodeData.status && isActionableBacklogStatus(nodeData.status) && (
          <ActionableBadge status={nodeData.status} />
        )}
      </div>
      <Handle type="source" position={Position.Bottom} className="!opacity-0 !w-1 !h-1 !min-w-0 !min-h-0 !border-0 !p-0" />
    </>
  );
}

export const GraphNode = memo(GraphNodeComponent);
