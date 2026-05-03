/**
 * ByPhaseView — four-column board grouping activities by lane.
 *
 * Each column maps 1:1 to one of the four canonical operations lanes
 * (Investigate / Execute / Review / Reconcile) and renders the subset
 * of `ActivityRow`s whose `lane` matches that column. The lane chip on
 * the row is suppressed inside this view (`showLane=false`) because the
 * column header already conveys the lane.
 *
 * Activities without a `lane` value are dropped silently. The
 * `/api/v1/operations` aggregator always sets `lane` when derivable;
 * the only rows that surface without a lane are queue rows whose
 * phase-kind isn't yet decided. Surfacing those in a "No lane" gutter
 * would be confusing — they are visible in the by-initiative view if
 * the operator needs them.
 *
 * Layout:
 *   Single horizontal flex row at every breakpoint. Each column has a
 *   `min-w-[260px]` floor and `flex-1` so columns grow to fill the row
 *   when there is room and the container scrolls horizontally when the
 *   floors exceed the available width. This avoids the prior failure
 *   mode where four equal grid columns squished below useful width on
 *   narrow desktops / sidebar-open layouts. Column headers are sticky
 *   relative to the column scroll body so scrolling within a long lane
 *   keeps the lane label visible.
 */

import { Card } from "../../ui/card";
import { selectors } from "../../../consts/selectors";
import { cn } from "../../../lib/utils";
import { OPERATIONS_LANES } from "../../../types/operations";
import type {
  ActivityRow as ActivityRowType,
  OperationsLane,
} from "../../../types/operations";
import { ActivityRow } from "../ActivityRow";
import { laneLabel, lanePalette } from "../utils";

export interface ByPhaseViewProps {
  activities: ActivityRowType[];
  /**
   * When true, rows render with leading checkboxes; selection state is
   * read from `useOperationsStore` inside ActivityRow itself.
   */
  selectable?: boolean;
}

interface LaneBucket {
  lane: OperationsLane;
  rows: ActivityRowType[];
}

/**
 * Bucket activities into the four canonical lanes in declared order.
 * Activities whose `lane` is missing or non-canonical are dropped — the
 * row will still surface in the by-initiative view.
 */
function bucketByLane(activities: ActivityRowType[]): LaneBucket[] {
  const buckets = new Map<OperationsLane, ActivityRowType[]>();
  for (const lane of OPERATIONS_LANES) buckets.set(lane, []);
  for (const row of activities) {
    if (!row.lane) continue;
    const bucket = buckets.get(row.lane as OperationsLane);
    if (!bucket) continue; // non-canonical lane → drop
    bucket.push(row);
  }
  return OPERATIONS_LANES.map((lane) => ({
    lane,
    rows: buckets.get(lane) ?? [],
  }));
}

export function ByPhaseView({ activities, selectable = false }: ByPhaseViewProps) {
  const buckets = bucketByLane(activities);

  return (
    <div
      className="flex gap-3 overflow-x-auto pb-1"
      data-testid={selectors.operationsCenter.byPhaseBoard}
      role="list"
    >
      {buckets.map((bucket) => (
        <LaneColumn
          key={bucket.lane}
          lane={bucket.lane}
          rows={bucket.rows}
          selectable={selectable}
        />
      ))}
    </div>
  );
}

interface LaneColumnProps {
  lane: OperationsLane;
  rows: ActivityRowType[];
  selectable?: boolean;
}

function LaneColumn({ lane, rows, selectable = false }: LaneColumnProps) {
  const palette = lanePalette(lane);
  return (
    <Card
      padding="sm"
      className="flex min-w-[260px] flex-1 basis-0 flex-col"
      data-testid={selectors.operationsCenter.byPhaseColumn}
      data-lane={lane}
      role="listitem"
    >
      <header
        className="sticky top-0 z-10 -mx-3 -mt-3 mb-2 flex items-center justify-between gap-2 rounded-t-lg bg-slate-900/85 px-3 py-2 backdrop-blur-sm"
        data-testid={selectors.operationsCenter.byPhaseColumnHeader}
        data-lane={lane}
      >
        <h2
          className={cn(
            "truncate text-sm font-medium",
            palette.text,
          )}
        >
          {laneLabel(lane)}
        </h2>
        <span className="shrink-0 rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300">
          {rows.length}
        </span>
      </header>
      {rows.length === 0 ? (
        <p
          className="rounded-md border border-dashed border-white/5 px-2 py-3 text-center text-[11px] text-slate-500"
          data-testid={selectors.operationsCenter.byPhaseColumnEmpty}
          data-lane={lane}
        >
          No active activity
        </p>
      ) : (
        <div className="flex flex-col gap-2">
          {rows.map((row) => (
            <ActivityRow
              key={row.runId ?? row.activityId}
              row={row}
              showLane={false}
              selectable={selectable}
            />
          ))}
        </div>
      )}
    </Card>
  );
}
