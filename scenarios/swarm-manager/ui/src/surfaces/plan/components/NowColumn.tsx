/**
 * NowColumn — the in-flight column, at full Operations-Center parity:
 * live activity cards from the proven operations polling path, lane
 * utilization bars, queue chip, group-by (initiative / phase), select
 * mode, and Spawn / Refresh actions. Bulk-stop confirms and outcomes are
 * handled by OpsBulkActions mounted at the board level.
 */

import { useMemo, useState } from "react";
import { Bot, ChevronDown, ChevronUp, HelpCircle, ListChecks, Plus, RefreshCw } from "lucide-react";
import { ActivityRow } from "../../../components/operations/ActivityRow";
import { LaneBar } from "../../../components/operations/LaneBar";
import { groupByInitiative, laneLabel, orderLanes } from "../../../components/operations/utils";
import { Button } from "../../../components/ui/button";
import { Tooltip } from "../../../components/ui/tooltip";
import { cn } from "../../../lib/utils";
import { useOperationsStore } from "../../../stores/operations-store";
import { OPERATIONS_LANES, type ActivityRow as ActivityRowType } from "../../../types/operations";
import { useSpawnSwarmAgent } from "../hooks/useSpawnSwarmAgent";
import { ColumnHeader } from "./ColumnHeader";

function bucketByLane(activities: ActivityRowType[]): Array<{ lane: string; rows: ActivityRowType[] }> {
  const buckets = new Map<string, ActivityRowType[]>();
  for (const lane of OPERATIONS_LANES) {
    buckets.set(lane, []);
  }
  for (const row of activities) {
    if (row.lane && buckets.has(row.lane)) {
      buckets.get(row.lane)?.push(row);
    }
  }
  return Array.from(buckets.entries()).map(([lane, rows]) => ({ lane, rows }));
}

export function NowColumn() {
  const [lanesOpen, setLanesOpen] = useState(true);
  const { spawn, isSpawning, error: spawnError } = useSpawnSwarmAgent();
  const view = useOperationsStore((s) => s.view);
  const isRefreshing = useOperationsStore((s) => s.isRefreshing);
  const viewMode = useOperationsStore((s) => s.viewMode);
  const selectionMode = useOperationsStore((s) => s.selectionMode);
  const toggleSelectionMode = useOperationsStore((s) => s.toggleSelectionMode);
  const refresh = useOperationsStore((s) => s.refresh);

  const activities = useMemo(() => view?.activities ?? [], [view?.activities]);
  const lanes = useMemo(() => orderLanes(view?.lanes ?? []), [view?.lanes]);
  const queueDepth = view?.queue.depth ?? 0;

  const initiativeGroups = useMemo(
    () => (viewMode === "by-initiative" ? groupByInitiative(activities) : []),
    [viewMode, activities],
  );
  const laneBuckets = useMemo(
    () => (viewMode === "by-phase" ? bucketByLane(activities) : []),
    [viewMode, activities],
  );

  const isIdle = activities.length === 0 && queueDepth === 0;

  return (
    <section
      className="flex h-full w-72 shrink-0 flex-col bg-slate-950/60 md:w-80"
      data-testid="plan-column-now"
    >
      <ColumnHeader
        title="Now"
        count={activities.length}
        subtitle={queueDepth > 0 ? `queue: ${queueDepth}` : "running"}
        action={
          <>
            <button
              type="button"
              onClick={() => toggleSelectionMode()}
              className={cn(
                "rounded p-1 transition-colors",
                selectionMode ? "bg-slate-700/80 text-cyan-400" : "text-slate-500 hover:text-slate-300",
              )}
              title={selectionMode ? "Exit select mode" : "Select agents to stop"}
              data-testid="plan-now-select-toggle"
            >
              <ListChecks className="h-4 w-4" aria-hidden />
            </button>
            <button
              type="button"
              onClick={() => void refresh({ force: true })}
              className="rounded p-1 text-slate-500 transition-colors hover:text-slate-300"
              title="Refresh"
              data-testid="plan-now-refresh"
            >
              <RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} aria-hidden />
            </button>
            <button
              type="button"
              onClick={() => void spawn()}
              disabled={isSpawning}
              className="rounded p-1 text-slate-500 transition-colors hover:text-slate-300 disabled:opacity-50"
              title="Spawn swarm-operations agent"
              data-testid="plan-now-spawn"
            >
              <Plus className="h-4 w-4" aria-hidden />
            </button>
          </>
        }
        testId="plan-column-now-header"
      />
      <div className="border-b border-slate-800/60">
        <div className="flex items-center justify-between px-3 pt-2">
          <button
            type="button"
            onClick={() => setLanesOpen((prev) => !prev)}
            className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-slate-400 transition-colors hover:text-slate-200"
            aria-expanded={lanesOpen}
            aria-label={`${lanesOpen ? "Collapse" : "Expand"} lane utilization`}
            data-testid="plan-now-lanes-toggle"
          >
            <span>Lanes</span>
            {lanesOpen ? (
              <ChevronUp className="h-3 w-3" aria-hidden />
            ) : (
              <ChevronDown className="h-3 w-3" aria-hidden />
            )}
          </button>
          <Tooltip content="Each lane runs agents up to its own concurrency limit. Bars show active / capacity for the investigate, execute, review, and reconcile lanes, plus any queued overflow.">
            <button
              type="button"
              className="rounded p-0.5 text-slate-500 transition-colors hover:text-slate-300"
              aria-label="About lanes"
              data-testid="plan-now-lanes-help"
            >
              <HelpCircle className="h-3.5 w-3.5" aria-hidden />
            </button>
          </Tooltip>
        </div>
        {lanesOpen && (
          <div className="space-y-1.5 px-3 pb-3 pt-2">
            {lanes.map((lane) => (
              <LaneBar key={lane.lane} status={lane} />
            ))}
          </div>
        )}
      </div>
      <div className="flex-1 space-y-3 overflow-y-auto p-2">
        {isIdle ? (
          <div
            className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-slate-800 px-4 py-8 text-center"
            data-testid="plan-now-empty"
          >
            <Bot className="h-8 w-8 text-slate-600" aria-hidden />
            <p className="text-sm text-slate-500">No agents running</p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void spawn()}
              disabled={isSpawning}
              data-testid="plan-now-spawn-cta"
            >
              {isSpawning ? "Spawning…" : "Spawn agent"}
            </Button>
            {spawnError && <p className="text-xs text-rose-400">{spawnError}</p>}
          </div>
        ) : viewMode === "by-initiative" ? (
          initiativeGroups.map((group) => (
            <div key={group.key || "__standalone__"} data-testid={`plan-now-group-${group.key || "standalone"}`}>
              <p className="px-1 pb-1 text-xs font-medium text-slate-400">
                {group.standalone ? "standalone" : group.label}
                <span className="ml-1 text-slate-600">{group.rows.length}</span>
              </p>
              <div className="space-y-1.5">
                {group.rows.map((row) => (
                  <ActivityRow
                    key={row.runId || row.activityId}
                    row={row}
                    selectable={selectionMode}
                  />
                ))}
              </div>
            </div>
          ))
        ) : (
          laneBuckets.map(({ lane, rows }) => (
            <div key={lane} data-testid={`plan-now-lane-group-${lane}`}>
              <p className="px-1 pb-1 text-xs font-medium text-slate-400">
                {laneLabel(lane)}
                <span className="ml-1 text-slate-600">{rows.length}</span>
              </p>
              {rows.length === 0 ? (
                <p className="px-1 text-xs text-slate-600">idle</p>
              ) : (
                <div className="space-y-1.5">
                  {rows.map((row) => (
                    <ActivityRow
                      key={row.runId || row.activityId}
                      row={row}
                      showLane={false}
                      selectable={selectionMode}
                    />
                  ))}
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </section>
  );
}
