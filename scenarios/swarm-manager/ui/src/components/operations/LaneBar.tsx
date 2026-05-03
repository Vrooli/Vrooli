/**
 * LaneBar — single-row utilization indicator for one phase-kind lane.
 *
 * Renders a labeled bar with `active / capacity` text and a subtle queue
 * count when non-zero. When utilization meets or exceeds the warning
 * threshold (≥80%), the bar fill switches to amber so saturation is
 * visible at a glance.
 *
 * Lays out tightly so four bars fit comfortably on desktop in one row;
 * the parent `OpsHeader` collapses these into a horizontally scrolling
 * strip on narrow viewports.
 */

import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import type { LaneStatus } from "../../types/operations";
import {
  LANE_WARNING_THRESHOLD,
  laneLabel,
  lanePalette,
} from "./utils";

export interface LaneBarProps {
  status: LaneStatus;
  className?: string;
}

export function LaneBar({ status, className }: LaneBarProps) {
  const palette = lanePalette(status.lane);
  const utilization =
    status.capacity > 0
      ? Math.min(1, status.active / status.capacity)
      : status.active > 0
        ? 1
        : 0;
  const warning = utilization >= LANE_WARNING_THRESHOLD && status.active > 0;
  const fill = warning ? palette.barWarning : palette.bar;
  const widthPercent = Math.round(utilization * 100);

  return (
    <div
      className={cn("min-w-[140px] flex-1", className)}
      data-testid={selectors.operationsCenter.laneBar}
      data-lane={status.lane}
      data-warning={warning ? "true" : "false"}
    >
      <div className="flex items-baseline justify-between gap-2">
        <span className={cn("text-xs font-medium", palette.text)}>
          {laneLabel(status.lane)}
        </span>
        <span className="text-[11px] tabular-nums text-slate-400">
          {status.active}
          <span className="text-slate-500"> / {status.capacity}</span>
          {status.queue > 0 && (
            <span className="ml-1.5 text-amber-300">+{status.queue} queued</span>
          )}
        </span>
      </div>
      <div
        className={cn(
          "mt-1 h-1.5 w-full overflow-hidden rounded-full",
          palette.track,
        )}
        role="progressbar"
        aria-label={`${laneLabel(status.lane)} lane utilization`}
        aria-valuenow={status.active}
        aria-valuemin={0}
        aria-valuemax={status.capacity || status.active || 1}
      >
        <div
          className={cn("h-full rounded-full transition-all duration-300", fill)}
          style={{ width: `${widthPercent}%` }}
        />
      </div>
    </div>
  );
}
