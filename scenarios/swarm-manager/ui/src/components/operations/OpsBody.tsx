/**
 * OpsBody — view-mode switch and recently-finished tail.
 *
 * Renders the active-view body (by-initiative card list or by-phase
 * lane board) plus a "Recently finished" tail so operators can confirm
 * a round they just kicked off completed without leaving the page.
 *
 * The view toggle is gated by `enableByPhaseView`. P7a ships the
 * by-phase board and flips the gate to `true`; the prop survives so
 * a future plan can hide the by-phase view behind a feature flag
 * without churning the toggle layout.
 */

import { useState } from "react";
import { CheckSquare, ChevronDown, ChevronUp, Layers, ListTree } from "lucide-react";
import { Card } from "../ui/card";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import type {
  ActivityRow as ActivityRowType,
  OperationsViewMode,
} from "../../types/operations";
import { ActivityRow } from "./ActivityRow";
import { ByInitiativeView } from "./views/ByInitiativeView";
import { ByPhaseView } from "./views/ByPhaseView";

export interface OpsBodyProps {
  view: OperationsViewMode;
  onViewChange(next: OperationsViewMode): void;
  activities: ActivityRowType[];
  recentlyFinished: ActivityRowType[];
  enableByPhaseView?: boolean;
  /**
   * When true, active rows render with a leading checkbox and read
   * selection / stopping state from the operations-store. Recently
   * finished rows are never selectable — bulk-stop targets active runs
   * only.
   */
  selectable?: boolean;
  /**
   * When provided, renders a "Select" toggle next to the view tabs that
   * flips multi-select on/off. The toggle is omitted when no handler is
   * supplied so non-page hosts (e.g. tests, embeds) can render a static
   * `selectable={true}` body without an extra control.
   */
  selectionMode?: boolean;
  onSelectionModeToggle?(): void;
}

const FINISHED_PREVIEW_COUNT = 8;

export function OpsBody({
  view,
  onViewChange,
  activities,
  recentlyFinished,
  enableByPhaseView = false,
  selectable = false,
  selectionMode,
  onSelectionModeToggle,
}: OpsBodyProps) {
  const showSelectionToggle = typeof onSelectionModeToggle === "function";
  const selectionToggleActive = Boolean(selectionMode);
  const [finishedExpanded, setFinishedExpanded] = useState(false);

  return (
    <section
      className="flex flex-col gap-3"
      data-testid={selectors.operationsCenter.body}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div role="tablist" className="inline-flex rounded-full bg-slate-900/60 p-1">
          <button
            type="button"
            role="tab"
            aria-selected={view === "by-initiative"}
            onClick={() => onViewChange("by-initiative")}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors",
              view === "by-initiative"
                ? "bg-slate-700 text-slate-100"
                : "text-slate-400 hover:text-slate-200",
            )}
            data-testid={selectors.operationsCenter.viewToggleByInitiative}
          >
            <Layers className="h-3.5 w-3.5" aria-hidden />
            By initiative
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={view === "by-phase" && enableByPhaseView}
            onClick={() => enableByPhaseView && onViewChange("by-phase")}
            disabled={!enableByPhaseView}
            title={enableByPhaseView ? undefined : "By-phase view is disabled"}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors",
              view === "by-phase" && enableByPhaseView
                ? "bg-slate-700 text-slate-100"
                : "text-slate-400 hover:text-slate-200",
              !enableByPhaseView && "cursor-not-allowed opacity-60",
            )}
            data-testid={selectors.operationsCenter.viewToggleByPhase}
          >
            <ListTree className="h-3.5 w-3.5" aria-hidden />
            By phase
          </button>
        </div>
        {showSelectionToggle && (
          <button
            type="button"
            onClick={onSelectionModeToggle}
            aria-pressed={selectionToggleActive}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors",
              selectionToggleActive
                ? "border-cyan-400/40 bg-cyan-500/15 text-cyan-100 hover:bg-cyan-500/25"
                : "border-slate-700/60 bg-slate-900/60 text-slate-400 hover:bg-slate-800/80 hover:text-slate-200",
            )}
            data-testid={selectors.operationsCenter.selectionModeToggle}
          >
            <CheckSquare className="h-3.5 w-3.5" aria-hidden />
            {selectionToggleActive ? "Selecting" : "Select"}
          </button>
        )}
      </div>

      {view === "by-phase" && enableByPhaseView ? (
        <ByPhaseView activities={activities} selectable={selectable} />
      ) : (
        <ByInitiativeView activities={activities} selectable={selectable} />
      )}

      {recentlyFinished.length > 0 && (
        <Card padding="sm">
          <header className="flex items-center justify-between gap-2">
            <h2 className="text-sm font-medium text-slate-200">
              Recently finished
              <span className="ml-2 text-[11px] font-normal text-slate-500">
                {recentlyFinished.length}
              </span>
            </h2>
            {recentlyFinished.length > FINISHED_PREVIEW_COUNT && (
              <button
                type="button"
                className="inline-flex items-center gap-1 text-[11px] font-medium text-slate-400 hover:text-slate-200"
                onClick={() => setFinishedExpanded((v) => !v)}
                aria-expanded={finishedExpanded}
              >
                {finishedExpanded ? (
                  <>
                    <ChevronUp className="h-3.5 w-3.5" aria-hidden />
                    Collapse
                  </>
                ) : (
                  <>
                    <ChevronDown className="h-3.5 w-3.5" aria-hidden />
                    Show all
                  </>
                )}
              </button>
            )}
          </header>
          <div className="mt-2 flex flex-col gap-2">
            {(finishedExpanded
              ? recentlyFinished
              : recentlyFinished.slice(0, FINISHED_PREVIEW_COUNT)
            ).map((row) => (
              <ActivityRow
                key={row.runId ?? row.activityId}
                row={row}
                showLane
              />
            ))}
          </div>
        </Card>
      )}
    </section>
  );
}
