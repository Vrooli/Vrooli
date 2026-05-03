/**
 * ActivityRow — compact row component shared between by-initiative and
 * by-phase views.
 *
 * Renders a status dot, the activity's display name, a short subtitle
 * (`mode · phase · round` for operating-mode runs, `purpose` for
 * everything else), the lane chip, and runtime. Click navigates to the
 * existing activity / execution detail surfaces — operating-mode runs
 * link to the initiative details page (where the round opens), backlog
 * spawns link to the existing execution detail page if one exists,
 * captures and sessions link to their respective detail pages.
 *
 * P6 ships this as read-only. P7b layers in row-level checkboxes for
 * bulk selection on top of the same component.
 */

import { useNavigate } from "react-router-dom";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import { useOperationsStore } from "../../stores/operations-store";
import { StatusChip, type StatusChipColors } from "../ui/status-chip";
import type { ActivityRow as ActivityRowType } from "../../types/operations";
import {
  activityDisplayName,
  activitySubtitle,
  formatRuntime,
  laneLabel,
  lanePalette,
} from "./utils";

export interface ActivityRowProps {
  row: ActivityRowType;
  /** When false, the row renders without lane chip (e.g. inside a lane column). */
  showLane?: boolean;
  className?: string;
  /**
   * When true, render a leading checkbox the operator can use to add the
   * row to bulk-action selection. Selection state, the in-flight
   * `isStopping` flag, and the toggle action are read from
   * `useOperationsStore` so the view tree (ByInitiativeView, ByPhaseView)
   * does not need to thread props down. Pages that render ActivityRow
   * outside the Operations Center leave this off and the row stays
   * read-only.
   *
   * Rows without a `runId` are not selectable (the bulk endpoint targets
   * run IDs); the checkbox is disabled in that case so the visual layout
   * stays uniform but the operator gets a hint about why a queue row
   * isn't selectable.
   */
  selectable?: boolean;
}

const PENDING_STATUS_COLORS: StatusChipColors = {
  background: "bg-slate-700/50",
  text: "text-slate-300",
  dot: "bg-slate-500",
};

const STATUS_COLORS: Record<string, StatusChipColors> = {
  pending: PENDING_STATUS_COLORS,
  starting: {
    background: "bg-cyan-500/15",
    text: "text-cyan-300",
    dot: "bg-cyan-500",
  },
  running: {
    background: "bg-emerald-500/15",
    text: "text-emerald-300",
    dot: "bg-emerald-500",
  },
  needs_review: {
    background: "bg-amber-500/15",
    text: "text-amber-300",
    dot: "bg-amber-500",
  },
  complete: {
    background: "bg-slate-700/40",
    text: "text-emerald-300",
    dot: "bg-emerald-500",
  },
  failed: {
    background: "bg-rose-500/15",
    text: "text-rose-300",
    dot: "bg-rose-500",
  },
  cancelled: {
    background: "bg-slate-700/40",
    text: "text-slate-400",
    dot: "bg-slate-500",
  },
};

const STATUS_LABELS: Record<string, string> = {
  pending: "Pending",
  starting: "Starting",
  running: "Running",
  needs_review: "Needs review",
  complete: "Complete",
  failed: "Failed",
  cancelled: "Cancelled",
};

function statusColors(status: string): StatusChipColors {
  return STATUS_COLORS[status] ?? PENDING_STATUS_COLORS;
}

function statusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status;
}

function detailHref(row: ActivityRowType): string | null {
  if (row.ownerType === "backlog" && row.ownerKind && row.ownerName) {
    return `/backlog/${row.ownerKind}/${row.ownerName}`;
  }
  if (row.ownerType === "initiative" && row.ownerName) {
    return `/initiatives/${row.ownerName}`;
  }
  if (row.ownerType === "scenario" && row.ownerName) {
    return `/scenarios/${row.ownerName}`;
  }
  if (row.ownerType === "capture" && row.ownerName) {
    return `/captures/${row.ownerName}`;
  }
  if (row.ownerType === "session" && row.ownerName) {
    return `/sessions/${row.ownerName}`;
  }
  return null;
}

export function ActivityRow({
  row,
  showLane = true,
  className,
  selectable = false,
}: ActivityRowProps) {
  const navigate = useNavigate();
  // Subscribe narrowly so unrelated store changes do not re-render every
  // row. The `runId` lookup short-circuits when selection is disabled.
  const runId = row.runId ?? null;
  const selected = useOperationsStore((s) =>
    selectable && runId ? s.selection.has(runId) : false,
  );
  const isStopping = useOperationsStore((s) =>
    selectable && runId ? s.stoppingRunIds.has(runId) : false,
  );
  const toggleSelection = useOperationsStore((s) => s.toggleSelection);
  const href = detailHref(row);
  const colors = statusColors(row.status);
  const label = statusLabel(row.status);
  const active =
    row.status === "running" ||
    row.status === "starting" ||
    row.status === "pending";
  const palette = row.lane ? lanePalette(row.lane) : null;

  // A row is checkbox-selectable only when the bulk endpoint can target it.
  // Queue rows without a runId are excluded; the disabled checkbox still
  // renders so the visual layout stays uniform.
  const canSelect = selectable && !!row.runId && !isStopping;

  const handleClick = () => {
    if (isStopping) return;
    if (href) navigate(href);
  };

  return (
    <div
      className={cn(
        "group flex items-center justify-between gap-3 rounded-lg border border-white/5 bg-slate-900/50 px-3 py-2 transition-colors",
        href && !isStopping && "cursor-pointer hover:border-cyan-500/40 hover:bg-slate-800/60",
        selected && "border-cyan-500/60 bg-slate-800/40",
        isStopping && "opacity-60",
        className,
      )}
      onClick={handleClick}
      onKeyDown={(e) => {
        if (!href || isStopping) return;
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          navigate(href);
        }
      }}
      role={href && !isStopping ? "button" : undefined}
      tabIndex={href && !isStopping ? 0 : -1}
      data-testid={selectors.operationsCenter.activityRow}
      data-run-id={row.runId ?? row.activityId}
      data-status={row.status}
      data-stopping={isStopping ? "true" : undefined}
      data-selected={selected ? "true" : undefined}
    >
      {selectable && (
        <input
          type="checkbox"
          aria-label={`Select activity ${activityDisplayName(row)}`}
          className="shrink-0 cursor-pointer accent-cyan-500 disabled:cursor-not-allowed disabled:opacity-50"
          data-testid={selectors.operationsCenter.activityRowCheckbox}
          data-run-id={row.runId ?? row.activityId}
          checked={selected}
          disabled={!canSelect}
          onClick={(event) => event.stopPropagation()}
          onChange={(event) => {
            event.stopPropagation();
            if (!row.runId) return;
            toggleSelection(row.runId);
          }}
        />
      )}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <StatusChip
            label={label}
            colors={colors}
            leadingDot
            pulse={active && !isStopping}
          />
          <span className="truncate text-sm font-medium text-slate-100">
            {activityDisplayName(row)}
          </span>
          {isStopping && (
            <span className="rounded-full bg-amber-500/15 px-2 py-0.5 text-[10px] font-medium text-amber-300">
              Stopping…
            </span>
          )}
        </div>
        <p className="mt-0.5 truncate text-[11px] text-slate-400">
          {activitySubtitle(row)}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-3">
        {showLane && row.lane && palette && (
          <span
            className={cn(
              "rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium",
              palette.text,
            )}
          >
            {laneLabel(row.lane)}
          </span>
        )}
        <span className="text-[11px] tabular-nums text-slate-400">
          {formatRuntime(row.runtimeSeconds)}
        </span>
      </div>
    </div>
  );
}
