/**
 * OpsHeader — stats strip for the Operations Center page.
 *
 * Composition: page title + four lane bars + queue chip + last-window
 * "done · failed" chip + manual refresh button. Layout is a flex row on
 * desktop and stacked on mobile so each piece stays legible in narrow
 * viewports.
 */

import { Bot, RefreshCw } from "lucide-react";
import { Button } from "../ui/button";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import type { OperationsView } from "../../types/operations";
import { LaneBar } from "./LaneBar";
import { orderLanes } from "./utils";

export interface OpsHeaderProps {
  view: OperationsView | null;
  isRefreshing: boolean;
  onRefresh(): void;
  windowSeconds: number;
  className?: string;
}

function countByStatus(
  view: OperationsView | null,
  status: "complete" | "failed" | "cancelled",
): number {
  if (!view) return 0;
  return view.recentlyFinished.filter((row) => row.status === status).length;
}

function formatWindow(seconds: number): string {
  if (seconds <= 0) return "—";
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  const hours = Math.round(seconds / 360) / 10; // one decimal
  return `${hours}h`;
}

export function OpsHeader({
  view,
  isRefreshing,
  onRefresh,
  windowSeconds,
  className,
}: OpsHeaderProps) {
  const lanes = view ? orderLanes(view.lanes) : [];
  const queueDepth = view?.queue.depth ?? 0;
  const completeCount = countByStatus(view, "complete");
  const failedCount = countByStatus(view, "failed");
  const cancelledCount = countByStatus(view, "cancelled");
  const windowLabel = formatWindow(windowSeconds);

  return (
    <header
      className={cn(
        "flex flex-col gap-4 rounded-xl border border-white/10 bg-slate-900/60 p-4",
        className,
      )}
      data-testid={selectors.operationsCenter.header}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Bot className="h-5 w-5 text-cyan-400" aria-hidden />
          <h1 className="text-base font-semibold text-slate-100">
            Operations Center
          </h1>
          <span className="text-xs text-slate-400">last {windowLabel}</span>
        </div>
        <div className="flex items-center gap-3">
          <span
            className="inline-flex items-center gap-1.5 rounded-full bg-slate-800/80 px-3 py-1 text-[11px] font-medium text-slate-300"
            data-testid={selectors.operationsCenter.queueChip}
            aria-label={`${queueDepth} queued`}
          >
            <span className="text-slate-500">Queue</span>
            <span className="tabular-nums text-slate-200">{queueDepth}</span>
          </span>
          <span
            className="inline-flex items-center gap-2 rounded-full bg-slate-800/80 px-3 py-1 text-[11px] font-medium"
            data-testid={selectors.operationsCenter.finishedChip}
            aria-label={`${completeCount} complete, ${failedCount} failed, ${cancelledCount} cancelled in window`}
          >
            <span className="text-emerald-300">{completeCount} ✓</span>
            <span className="text-rose-300">{failedCount} ✗</span>
            {cancelledCount > 0 && (
              <span className="text-slate-400">{cancelledCount} ⊘</span>
            )}
          </span>
          <Button
            variant="ghost"
            size="sm"
            onClick={onRefresh}
            disabled={isRefreshing}
            data-testid={selectors.operationsCenter.refreshButton}
          >
            <RefreshCw
              className={cn("mr-1.5 h-3.5 w-3.5", isRefreshing && "animate-spin")}
              aria-hidden
            />
            Refresh
          </Button>
        </div>
      </div>
      <div className="flex flex-wrap gap-x-6 gap-y-3 sm:flex-nowrap sm:overflow-x-auto">
        {lanes.map((lane) => (
          <LaneBar key={lane.lane} status={lane} />
        ))}
      </div>
    </header>
  );
}
