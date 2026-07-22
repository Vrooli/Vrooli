/* eslint-disable react-refresh/only-export-components */
/**
 * RollupProgressBar - Segmented progress bar for goal rollup data.
 *
 * Displays a horizontal bar with colored segments proportional to each status
 * count (completed, inProgress, failed, pending). Optionally shows a numeric
 * breakdown below the bar.
 *
 * Used in goal detail and sidebar cards.
 */

import { cn } from "../../lib/utils";


export interface GoalRollup {
  total: number;
  completed: number;
  inProgress: number;
  failed: number;
  pending: number;
  archived: number;
}

export interface RollupProgressBarProps {
  rollup: GoalRollup;
  /** Show the numeric breakdown labels below the bar. Default: false. */
  showLabels?: boolean;
  /** Height class for the bar. Default: "h-2.5". */
  barHeight?: string;
  /** Text size class for numeric labels. Default: "text-xs". */
  labelSize?: string;
  className?: string;
}

/**
 * Compute the total count from a rollup object.
 *
 * Archived items are intentionally excluded. They remain part of goal
 * scope, but the progress bar reflects only live work still participating in
 * the active lifecycle.
 */
export function rollupTotal(rollup: GoalRollup): number {
  return rollup.completed + rollup.inProgress + rollup.failed + rollup.pending;
}

/** Segment definition for the progress bar. */
interface Segment {
  key: string;
  count: number;
  barColor: string;
  labelColor: string;
  label: string;
  /** Only show this segment in the label row when count > 0. */
  hideWhenZero?: boolean;
}

function getSegments(rollup: GoalRollup): Segment[] {
  return [
    { key: "completed", count: rollup.completed, barColor: "bg-emerald-500", labelColor: "text-emerald-400", label: "completed" },
    { key: "inProgress", count: rollup.inProgress, barColor: "bg-purple-500", labelColor: "text-purple-400", label: "in progress" },
    { key: "failed", count: rollup.failed, barColor: "bg-red-500", labelColor: "text-red-400", label: "failed", hideWhenZero: true },
    { key: "pending", count: rollup.pending, barColor: "bg-slate-600", labelColor: "text-slate-400", label: "pending" },
  ];
}

export function RollupProgressBar({
  rollup,
  showLabels = false,
  barHeight = "h-2.5",
  labelSize = "text-xs",
  className,
}: RollupProgressBarProps) {
  const total = rollupTotal(rollup);
  if (total === 0) return null;

  const segments = getSegments(rollup);

  return (
    <div className={cn("space-y-1.5", className)} data-testid="rollup-progress-bar">
      {/* Segmented bar */}
      <div className={cn("flex w-full overflow-hidden rounded-full bg-slate-800", barHeight)}>
        {segments.map(
          (seg) =>
            seg.count > 0 && (
              <div
                key={seg.key}
                className={cn(seg.barColor, "transition-all")}
                style={{ width: `${(seg.count / total) * 100}%` }}
                title={`${seg.count} ${seg.label}`}
              />
            ),
        )}
      </div>

      {/* Optional numeric breakdown */}
      {showLabels && (
        <div className={cn("flex flex-wrap gap-x-5 gap-y-1", labelSize)}>
          {segments.map(
            (seg) =>
              (!seg.hideWhenZero || seg.count > 0) && (
                <span key={seg.key} className={seg.labelColor}>
                  {seg.count} {seg.label}
                </span>
              ),
          )}
          <span className="text-slate-500">{total} total</span>
        </div>
      )}
    </div>
  );
}
