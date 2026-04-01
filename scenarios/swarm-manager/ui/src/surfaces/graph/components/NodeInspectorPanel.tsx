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

import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ExternalLink } from "lucide-react";
import { cn } from "../../../lib/utils";
import { StatusBadge } from "../../../components/detail/StatusBadge";
import { backlogService } from "../../../services";
import { useBacklogStore } from "../../../stores";
import type { BacklogKind, BacklogStatus } from "../../../types";
import { LOCKED_STATUSES } from "../../../lib/backlog-queue-utils";
import { FloatingPanel } from "../../../components/ui/floating-panel";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { useDetailSelectionStore } from "../../../stores/detail-selection-store";
import { useDetailNavigation } from "../../../hooks/useDetailNavigation";
import { hasDetailPage } from "../lib/detail-page-registry";
import { parseNodeId } from "../lib/node-id-parser";
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
  type ExecutionGraphNodeData,
  type CaptureGraphNodeData,
  type AgentActivityGraphNodeData,
  type RunGraphNodeData,
  type ScenarioGraphNodeData,
} from "../types";
import type { GraphLens } from "../stores/graph-data-store";

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
      const d = data as ExecutionGraphNodeData;
      return (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span className="rounded bg-slate-800 px-1.5 py-0.5 font-medium capitalize">{d.mode}</span>
          <span className="truncate">{d.backlogName}</span>
        </div>
      );
    }
    case "scenario": {
      const d = data as ScenarioGraphNodeData;
      return (
        <div className="text-xs text-slate-400">
          <span className="capitalize">{d.name}</span>
        </div>
      );
    }
    case "capture": {
      const d = data as CaptureGraphNodeData;
      return (
        <p className="line-clamp-2 text-xs text-slate-400">{d.text}</p>
      );
    }
    case "agent-activity": {
      const d = data as AgentActivityGraphNodeData;
      return (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span className="capitalize">{d.purpose}</span>
          <span className="truncate text-slate-500">{d.ownerName}</span>
        </div>
      );
    }
    case "agent-run": {
      const d = data as RunGraphNodeData;
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

export function NodeInspectorPanel() {
  const [, setSearchParams] = useSearchParams();
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const nodes = useGraphDataStore((s) => s.nodes);
  const currentLens = useGraphDataStore((s) => s.lens);

  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);
  const selectScenario = useDetailSelectionStore((s) => s.selectScenario);
  const selectExecution = useDetailSelectionStore((s) => s.selectExecution);
  const selectInitiative = useDetailSelectionStore((s) => s.selectInitiative);

  const { drillToLens } = useDetailNavigation();

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
    const parsed = parseNodeId(selectedNodeId);
    if (!parsed) return;

    switch (parsed.entityType) {
      case "backlog":
        if (parsed.kind && parsed.name) selectBacklog(parsed.kind, parsed.name);
        break;
      case "scenario":
        if (parsed.name) selectScenario(parsed.name);
        break;
      case "execution":
        selectExecution(parsed.identifier);
        break;
      case "initiative":
        if (parsed.name) selectInitiative(parsed.name);
        break;
    }
  };

  const handleDrillToLens = (lens: GraphLens) => {
    if (!selectedNodeId) return;
    drillToLens(selectedNodeId, lens);
  };

  if (!selectedNode || !nodeData) return null;

  const label = getGraphNodeLabel(selectedNode);
  const entityType = nodeData.entityType;
  const statusColors = getStatusColorClasses(nodeData.status);
  const showDetails = hasDetailPage(entityType);
  const lenses = getLensesForEntity(entityType).filter((l) => l.lens !== currentLens);

  return (
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

        {/* Action buttons */}
        {(showDetails || lenses.length > 0) && (
          <div className="flex flex-wrap items-center gap-1.5 border-t border-white/10 pt-3">
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
      </div>
    </FloatingPanel>
  );
}
