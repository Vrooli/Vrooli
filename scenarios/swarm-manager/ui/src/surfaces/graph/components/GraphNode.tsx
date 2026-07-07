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
import { Target } from "lucide-react";
import { cn } from "../../../lib/utils";
import { BACKLOG_KIND_ICONS } from "../../../types";
import type { BacklogKind } from "../../../types";
import { selectors } from "../../../consts/selectors";
import { useGovernanceStore, isCircuitBroken } from "../../../stores/governance-store";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { GraphEntityType, GraphNodeData } from "../types";
import { phaseLabel } from "../../../components/initiative/operating-mode/utils";
import { ActionableBadge, CircuitBrokenNodeBadge, StatusBadge } from "./StatusBadge";
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
  const goalBadges = nodeData.goalBadges ?? [];
  const inGoal = goalBadges.length > 0;
  const circuitBroken = useGovernanceStore((s) =>
    entityType === "backlog" && "name" in nodeData && "kind" in nodeData
      ? isCircuitBroken(s, nodeData.kind as string, (nodeData as { name: string }).name)
      : false,
  );
  const backlogKindIcon = entityType === "backlog" && nodeData.kind
    ? BACKLOG_KIND_ICONS[nodeData.kind as BacklogKind]
    : undefined;
  const Icon = backlogKindIcon ?? getEntityIcon(entityType);
  const shapeClass = getShapeClasses(entityType);
  const statusColors = getStatusColorClasses(nodeData.status);
  const isClipped = usesClipPath(entityType);
  const dims = getShapeDimensions(entityType);
  const clipStyle = getClipPathStyle(entityType);
  const activeRound =
    entityType === "initiative" && "activeRound" in nodeData
      ? (nodeData as { activeRound?: { mode: string; phase: string; round: number; status: string } }).activeRound
      : undefined;
  const isAgentRunning = activeRound?.status === "agent_running";

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
          inGoal && !isSelected && isClipped && "drop-shadow-[0_0_6px_rgba(232,121,249,0.6)]",
          Boolean(nodeData.pulsing) && (nodeData.pulseMode === "persistent"
            ? "graph-node-attention-pulse"
            : "graph-node-pulse"),
          // Active operating-mode round in agent_running state pulses
          // persistently so operators can spot mid-phase initiatives at a
          // glance from the workspace graph.
          isAgentRunning && "graph-node-attention-pulse",
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
            // Goal-membership tint — a fuchsia ring distinct from cyan selection.
            inGoal && !isSelected && !isClipped && "ring-2 ring-fuchsia-400/40",
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
        {lens === "topology" && entityType === "backlog" && circuitBroken && (
          <CircuitBrokenNodeBadge />
        )}
        {/* Goal-membership badge: a fuchsia target chip listing the node's goals. */}
        {inGoal && (
          <div
            className="absolute -left-1.5 -top-1.5 flex items-center gap-0.5 rounded-full border border-fuchsia-400/50 bg-fuchsia-500/20 px-1 py-0.5 text-[8px] font-medium text-fuchsia-200 backdrop-blur"
            title={`In goal${goalBadges.length > 1 ? "s" : ""}: ${goalBadges.map((g) => g.title).join(", ")}`}
            data-testid="graph-node-goal-badge"
          >
            <Target className="h-2.5 w-2.5" aria-hidden />
            {goalBadges.length > 1 && <span>{goalBadges.length}</span>}
          </div>
        )}
        {entityType === "initiative" && activeRound ? (
          <div
            data-testid={selectors.initiativeDetails.graphNodeActiveRoundChip}
            className={cn(
              "absolute -bottom-2 left-1/2 -translate-x-1/2 whitespace-nowrap rounded-full border px-2 py-0.5 text-[9px] font-medium",
              isAgentRunning
                ? "border-cyan-400/60 bg-cyan-500/15 text-cyan-200"
                : "border-amber-400/60 bg-amber-500/10 text-amber-200",
            )}
            title={`${activeRound.mode} · round ${activeRound.round} · ${activeRound.status.replace(/_/g, " ")}`}
          >
            {phaseLabel(activeRound.phase)}
          </div>
        ) : null}
      </div>
      <Handle type="source" position={Position.Bottom} className="!opacity-0 !w-1 !h-1 !min-w-0 !min-h-0 !border-0 !p-0" />
    </>
  );
}

export const GraphNode = memo(GraphNodeComponent);
