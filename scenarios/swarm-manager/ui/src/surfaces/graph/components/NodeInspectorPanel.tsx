/**
 * NodeInspectorPanel - Compact floating panel for the selected graph node.
 *
 * Appears when a node is selected on the graph. Shows entity info,
 * an "Open Details" button (for entities with detail pages), and
 * cross-lens navigation buttons (filtered to exclude the current lens).
 *
 * Keeps the user in the graph view for exploration instead of
 * immediately opening a full-screen detail overlay.
 */

import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertCircle, ExternalLink, Play, Target } from "lucide-react";
import { FocusActionsSection } from "./FocusActionsSection";
import { SetAsGoalDialog } from "../../../components/goals/SetAsGoalDialog";
import { cn } from "../../../lib/utils";
import { useNodeGoalBadges } from "../hooks/useGoalMembership";
import { StatusBadge } from "../../../components/detail/StatusBadge";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { defaultApiClient } from "../../../lib/api-client";
import { backlogService } from "../../../services";
import { useBacklogStore } from "../../../stores";
import type { BacklogKind, BacklogStatus } from "../../../types";
import { LOCKED_STATUSES } from "../../../lib/backlog-queue-utils";
import { FloatingPanel } from "../../../components/ui/floating-panel";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { computeNodeAttention, formatAttentionSummary } from "../lib/attention";
import { hasDetailPage } from "../lib/detail-page-registry";
import { getStatusColorClasses } from "../lib/status-colors";
import {
  BACKLOG_LENSES,
  SCENARIO_LENSES,
  EXECUTION_LENSES,
  INITIATIVE_LENSES,
} from "../../../components/detail/lens-options";
import type { LensOption } from "../../../components/detail/lens-options";
import {
  getGraphNodeData,
  getGraphNodeLabel,
  type GraphEntityType,
  type GraphNodeData,
  type BacklogGraphNodeData,
  type InitiativeGraphNodeData,
} from "../types";
import { detailPathFromNodeId, graphPath } from "../../../app/routes/route-paths";

function getLensesForEntity(entityType: GraphEntityType): LensOption[] {
  switch (entityType) {
    case "backlog":
      return BACKLOG_LENSES;
    case "scenario":
      return SCENARIO_LENSES;
    case "execution":
      return EXECUTION_LENSES;
    case "initiative":
      return INITIATIVE_LENSES;
    default:
      return [];
  }
}

function EntityMeta({ data }: { data: GraphNodeData }) {
  switch (data.entityType) {
    case "backlog": {
      const d = data as BacklogGraphNodeData;
      return (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span className="rounded bg-slate-800 px-1.5 py-0.5 font-medium capitalize">{d.kind}</span>
          {d.priority > 0 && <span>P{d.priority}</span>}
          {d.activeExecutionStatus && (
            <span className="text-amber-400">Running</span>
          )}
        </div>
      );
    }
    case "initiative": {
      const d = data as InitiativeGraphNodeData;
      const { rollup } = d;
      if (!rollup || rollup.total === 0) return null;
      const pct = Math.round((rollup.completed / rollup.total) * 100);
      return (
        <div className="flex flex-col gap-1">
          <div className="flex h-1.5 w-full overflow-hidden rounded-full bg-slate-800">
            {rollup.completed > 0 && (
              <div className="bg-emerald-500" style={{ width: `${(rollup.completed / rollup.total) * 100}%` }} />
            )}
            {rollup.in_progress > 0 && (
              <div className="bg-cyan-500" style={{ width: `${(rollup.in_progress / rollup.total) * 100}%` }} />
            )}
            {rollup.failed > 0 && (
              <div className="bg-red-500" style={{ width: `${(rollup.failed / rollup.total) * 100}%` }} />
            )}
          </div>
          <span className="text-xs text-slate-400">{pct}% complete ({rollup.completed}/{rollup.total})</span>
        </div>
      );
    }
    case "execution": {
      const d = data;
      return (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span className="rounded bg-slate-800 px-1.5 py-0.5 font-medium capitalize">{d.mode}</span>
          <span className="truncate">{d.backlogName}</span>
        </div>
      );
    }
    case "scenario": {
      const d = data;
      return (
        <div className="text-xs text-slate-400">
          <span className="capitalize">{d.name}</span>
        </div>
      );
    }
    case "capture": {
      const d = data;
      return (
        <p className="line-clamp-2 text-xs text-slate-400">{d.text}</p>
      );
    }
    case "agent-activity": {
      const d = data;
      return (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span className="capitalize">{d.purpose}</span>
          <span className="truncate text-slate-500">{d.ownerName}</span>
        </div>
      );
    }
    case "agent-run": {
      const d = data;
      return (
        <div className="text-xs text-slate-400 font-mono">
          {d.runId}
        </div>
      );
    }
    default:
      return null;
  }
}

const INSPECTOR_POSITION = { x: window.innerWidth - 380, y: window.innerHeight - 300 };

/**
 * goalTargetForNode maps a backlog or initiative node to a goal target ref
 * ("<kind>/<name>" for items, "initiative/<name>" for initiatives). Other node
 * types cannot be goal targets and return null.
 */
function goalTargetForNode(data: GraphNodeData): { ref: string; title: string } | null {
  if (data.entityType === "backlog" && data.rawType === "BacklogItem") {
    const d = data;
    return { ref: `${d.kind}/${d.name}`, title: d.title || d.name };
  }
  if (data.entityType === "initiative" && data.rawType === "Initiative") {
    const d = data;
    return { ref: `initiative/${d.name}`, title: d.title || d.name };
  }
  return null;
}

export function NodeInspectorPanel() {
  const navigate = useNavigate();
  const [, setSearchParams] = useSearchParams();
  const [goalDialogOpen, setGoalDialogOpen] = useState(false);
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const nodes = useGraphDataStore((s) => s.nodes);
  const currentLens = useGraphDataStore((s) => s.lens);

  const queryClient = useQueryClient();
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);

  const statusMutation = useMutation({
    mutationFn: ({ kind, name: itemName, newStatus }: { kind: BacklogKind; name: string; newStatus: BacklogStatus }) =>
      backlogService.update(kind, itemName, { status: newStatus }),
    onSuccess: () => {
      void fetchBacklog({ force: true });
      void queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
    },
  });

  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null;
    return nodes.find((n) => n.id === selectedNodeId) ?? null;
  }, [selectedNodeId, nodes]);

  const nodeData = useMemo<GraphNodeData | null>(() => {
    if (!selectedNode) return null;
    return getGraphNodeData(selectedNode);
  }, [selectedNode]);
  const goalBadges = useNodeGoalBadges(selectedNodeId ?? "");

  const handleClose = () => {
    selectNode(null);
    setHighlightState({ highlighted: new Set<string>(), mode: "normal" });
    // Clear URL synchronously to prevent the focus-restoration effect
    // in GraphCanvas from re-selecting via the stale ?select= param.
    setSearchParams((prev) => {
      if (!prev.has("select")) return prev;
      const next = new URLSearchParams(prev);
      next.delete("select");
      return next;
    });
  };

  const handleOpenDetails = () => {
    if (!selectedNodeId || !nodeData) return;
    const path = detailPathFromNodeId(selectedNodeId);
    if (path) navigate(path);
  };

  const handleDrillToLens = (lens: import("../../../app/routes/route-paths").AppGraphLens) => {
    if (!selectedNodeId) return;
    navigate(graphPath({ lens, focus: selectedNodeId, select: selectedNodeId }));
  };

  const queueMutation = useMutation({
    mutationFn: ({ kind, name: itemName }: { kind: BacklogKind; name: string }) =>
      defaultApiClient.post(API_ENDPOINTS.backlogQueue(kind, itemName), {}),
    onSuccess: () => {
      void fetchBacklog({ force: true });
      void queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
    },
  });

  const attention = useMemo(() => {
    if (!nodeData) return null;
    return computeNodeAttention(nodeData);
  }, [nodeData]);

  if (!selectedNode || !nodeData) return null;

  const label = getGraphNodeLabel(selectedNode);
  const entityType = nodeData.entityType;
  const statusColors = getStatusColorClasses(nodeData.status);
  const showDetails = hasDetailPage(entityType);
  const lenses = getLensesForEntity(entityType).filter((l) => l.lens !== currentLens);
  const isReadyBacklog = entityType === "backlog" && nodeData.status === "ready";
  const isFocusLens = currentLens === "focus";
  const goalTarget = goalTargetForNode(nodeData);

  return (
    <>
    <FloatingPanel
      isOpen
      onClose={handleClose}
      title={label}
      initialPosition={INSPECTOR_POSITION}
      className="!max-w-sm"
      testId="node-inspector"
    >
      <div className="flex flex-col gap-3">
        {/* Entity type + status */}
        <div className="flex items-center gap-2">
          <span className="rounded-full bg-slate-700/60 px-2 py-0.5 text-xs font-medium uppercase tracking-wider text-slate-400">
            {entityType === "agent-activity" ? "activity" : entityType === "agent-run" ? "run" : entityType}
          </span>
          {nodeData.status && (
            entityType === "backlog" && !LOCKED_STATUSES.has(nodeData.status as BacklogStatus) ? (
              <StatusBadge
                status={nodeData.status}
                onStatusChange={(newStatus) => {
                  const d = nodeData as BacklogGraphNodeData;
                  statusMutation.mutate({ kind: d.kind, name: d.name, newStatus });
                }}
                statusChangePending={statusMutation.isPending}
              />
            ) : (
              <span className={cn("rounded-full px-2 py-0.5 text-xs font-medium", statusColors.background, statusColors.text)}>
                {nodeData.status.replace(/_/g, " ")}
              </span>
            )
          )}
        </div>

        {/* Entity-specific metadata */}
        <EntityMeta data={nodeData} />

        {goalBadges.length > 0 && (
          <div
            className="rounded-lg border border-fuchsia-400/30 bg-fuchsia-500/10 px-3 py-2"
            data-testid="inspector-goal-membership"
          >
            <p className="mb-1 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-fuchsia-200">
              <Target className="h-3 w-3" aria-hidden />
              In goal{goalBadges.length > 1 ? "s" : ""}
            </p>
            <div className="flex flex-wrap gap-1.5">
              {goalBadges.map((goal) => (
                <span
                  key={goal.name}
                  className="rounded-full border border-fuchsia-400/30 bg-fuchsia-500/15 px-2 py-0.5 text-xs text-fuchsia-100"
                  title={`Priority ${goal.priority}`}
                >
                  {goal.title}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Attention chip + quick actions (suppressed in focus mode — FocusActionsSection handles it) */}
        {!isFocusLens && (attention?.needsAttention || isReadyBacklog) && (
          <div className="flex flex-wrap items-center gap-1.5">
            {attention?.needsAttention && (
              <button
                type="button"
                onClick={handleOpenDetails}
                className="flex items-center gap-1.5 rounded-full bg-amber-500/15 border border-amber-500/30 px-2.5 py-1 text-xs font-medium text-amber-200 transition-colors hover:bg-amber-500/25"
                data-testid="inspector-attention-chip"
              >
                <AlertCircle className="h-3 w-3 text-amber-400" />
                {formatAttentionSummary(attention.reasons)}
              </button>
            )}
            {isReadyBacklog && (
              <button
                type="button"
                onClick={() => {
                  const d = nodeData;
                  queueMutation.mutate({ kind: d.kind, name: d.name });
                }}
                disabled={queueMutation.isPending}
                className="flex items-center gap-1.5 rounded-lg bg-emerald-600/80 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-600 disabled:opacity-50"
                data-testid="inspector-run-button"
              >
                <Play className="h-3 w-3" />
                {queueMutation.isPending ? "Queuing…" : "Run"}
              </button>
            )}
          </div>
        )}

        {/* Focus-mode inline actions */}
        {isFocusLens && selectedNodeId && (
          <FocusActionsSection nodeData={nodeData} nodeId={selectedNodeId} />
        )}

        {/* Action buttons */}
        {(showDetails || lenses.length > 0 || goalTarget) && (
          <div className="flex flex-wrap items-center gap-1.5 border-t border-white/10 pt-3">
            {goalTarget && (
              <button
                type="button"
                onClick={() => setGoalDialogOpen(true)}
                className="flex items-center gap-1.5 rounded-lg bg-slate-700/50 px-3 py-1.5 text-xs font-medium text-slate-100 transition-colors hover:bg-slate-700/70"
                data-testid="inspector-set-goal"
              >
                <Target className="h-3 w-3 text-cyan-400" />
                Set as goal
              </button>
            )}
            {showDetails && (
              <button
                type="button"
                onClick={handleOpenDetails}
                className="flex items-center gap-1.5 rounded-lg bg-cyan-600/80 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-600"
                data-testid="inspector-open-details"
              >
                <ExternalLink className="h-3 w-3" />
                Open Details
              </button>
            )}
            {lenses.map(({ lens, label: lensLabel, icon: Icon, iconColorClass }) => (
              <button
                key={lens}
                type="button"
                onClick={() => handleDrillToLens(lens)}
                className="flex items-center gap-1.5 rounded-lg bg-slate-700/50 px-3 py-1.5 text-xs font-medium text-slate-100 transition-colors hover:bg-slate-700/70"
                data-testid={`inspector-lens-${lens}`}
              >
                <Icon className={cn("h-3 w-3", iconColorClass)} />
                {lensLabel}
              </button>
            ))}
          </div>
        )}
        {!goalTarget && (
          <p className="border-t border-white/10 pt-3 text-xs text-slate-500" data-testid="inspector-goal-unsupported">
            Goal targets are available for backlog items and initiatives.
          </p>
        )}
      </div>
    </FloatingPanel>
    {goalTarget && (
      <SetAsGoalDialog
        isOpen={goalDialogOpen}
        onClose={() => setGoalDialogOpen(false)}
        targetRef={goalTarget.ref}
        targetTitle={goalTarget.title}
      />
    )}
    </>
  );
}
