/**
 * ByInitiativeView — groups activities by initiative.
 *
 * Each card shows an initiative name + a count of active rows + a list
 * of activity rows. A "Standalone items" bucket at the bottom catches
 * everything that does not belong to an initiative (item-level work,
 * scenario / capture / session spawns).
 *
 * P6 ships this as the only body view. P7a adds `ByPhaseView`; the
 * parent `OpsBody` switches between the two based on store state.
 */

import { Layers } from "lucide-react";
import { Card } from "../../ui/card";
import { selectors } from "../../../consts/selectors";
import type { ActivityRow as ActivityRowType } from "../../../types/operations";
import { ActivityRow } from "../ActivityRow";
import { groupByInitiative } from "../utils";

export interface ByInitiativeViewProps {
  activities: ActivityRowType[];
}

function modeBadgeText(rows: ActivityRowType[]): string | null {
  for (const row of rows) {
    if (row.mode) return row.mode;
  }
  return null;
}

export function ByInitiativeView({ activities }: ByInitiativeViewProps) {
  const groups = groupByInitiative(activities);

  if (groups.length === 0) return null;

  return (
    <div className="flex flex-col gap-3">
      {groups.map((group) =>
        group.standalone ? (
          <Card
            key={group.key || "standalone"}
            padding="sm"
            data-testid={selectors.operationsCenter.standaloneBucket}
          >
            <header className="mb-2 flex items-center justify-between gap-2">
              <h2 className="text-sm font-medium text-slate-200">
                Standalone items
              </h2>
              <span className="text-[11px] text-slate-500">
                {group.rows.length} active
              </span>
            </header>
            <div className="flex flex-col gap-2">
              {group.rows.map((row) => (
                <ActivityRow
                  key={row.runId ?? row.activityId}
                  row={row}
                  showLane
                />
              ))}
            </div>
          </Card>
        ) : (
          <Card
            key={group.key}
            padding="sm"
            data-testid={selectors.operationsCenter.initiativeCard}
            data-initiative={group.label}
          >
            <header className="mb-2 flex items-center justify-between gap-2">
              <div className="flex min-w-0 items-center gap-2">
                <Layers className="h-4 w-4 shrink-0 text-cyan-400" aria-hidden />
                <h2 className="truncate text-sm font-medium text-slate-100">
                  {group.label}
                </h2>
                {modeBadgeText(group.rows) && (
                  <span className="shrink-0 rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300">
                    {modeBadgeText(group.rows)}
                  </span>
                )}
              </div>
              <span className="shrink-0 text-[11px] text-slate-500">
                {group.rows.length} active
              </span>
            </header>
            <div className="flex flex-col gap-2">
              {group.rows.map((row) => (
                <ActivityRow
                  key={row.runId ?? row.activityId}
                  row={row}
                  showLane
                />
              ))}
            </div>
          </Card>
        ),
      )}
    </div>
  );
}
