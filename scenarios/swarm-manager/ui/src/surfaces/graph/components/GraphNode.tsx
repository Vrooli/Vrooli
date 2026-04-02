/**
 * GraphNode - Custom node renderer for all entity types.
 *
 * Shape varies by entity type (stretched-hexagon, stadium, parallelogram, etc.).
 * Color encodes STATUS via both fill and border for instant scannability.
 *
 * All visual identity metadata lives in the ENTITY_REGISTRY (lib/entity-shapes.ts).
 * This component simply reads from it — no entity-type-keyed maps here.
 *
 * @see lib/entity-shapes.ts for shape mapping
 * @see lib/status-colors.ts for color mapping
 */

import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { cn } from "../../../lib/utils";
import { BACKLOG_KIND_ICONS } from "../../../types";
import type { BacklogKind } from "../../../types";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { GraphEntityType, GraphNodeData } from "../types";
import { ActionableBadge, StatusBadge } from "./StatusBadge";
import {
  getClipPathStyle,
  getEntityBadgeLabel,
  getEntityIcon,
  getShapeClasses,
  getShapeDimensions,
  usesClipPath,
} from "../lib/entity-shapes";
import { getStatusColorClasses, isActionableBacklogStatus } from "../lib/status-colors";

const DEFAULT_ENTITY: GraphEntityType = "backlog";

function GraphNodeComponent({ id, data }: NodeProps) {
  const nodeData = data as GraphNodeData;
  const lens = useGraphDataStore((s) => s.lens);
  const isSelected = useGraphUIStore((s) => s.selectedNodeId === id);
  const entityType = nodeData.entityType ?? DEFAULT_ENTITY;
  const backlogKindIcon = entityType === "backlog" && nodeData.kind
    ? BACKLOG_KIND_ICONS[nodeData.kind as BacklogKind]
    : undefined;
  const Icon = backlogKindIcon ?? getEntityIcon(entityType);
  const shapeClass = getShapeClasses(entityType);
  const statusColors = getStatusColorClasses(nodeData.status);
  const isClipped = usesClipPath(entityType);
  const dims = getShapeDimensions(entityType);
  const clipStyle = getClipPathStyle(entityType);

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
          Boolean(nodeData.pulsing) && (nodeData.pulseMode === "persistent"
            ? "graph-node-attention-pulse"
            : "graph-node-pulse"),
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
          )}
          style={{ width: dims.width, height: dims.height, ...clipStyle }}
        >
          <div className="flex flex-col items-center gap-0.5 w-full px-3">
            {/* Entity type badge with icon */}
            <div className="flex items-center gap-1">
              <Icon className={cn("h-3 w-3 shrink-0", statusColors.text)} />
              <span className={cn("text-[9px] font-medium uppercase tracking-wide", statusColors.text)}>
                {getEntityBadgeLabel(entityType)}
              </span>
            </div>

            {/* Label */}
            <p className={cn(
              "text-[10px] font-medium leading-tight text-center break-words line-clamp-2 max-w-[110px]",
              statusColors.text,
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
