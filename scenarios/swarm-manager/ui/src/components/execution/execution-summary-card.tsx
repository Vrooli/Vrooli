/**
 * ExecutionSummaryCard — the compact execution row shared by the sidebar
 * ExecutionsTab and the SessionContextPicker. This is the terse list row, NOT
 * the rich `ExecutionCard` used by ExecutionListView.
 *
 * - Sidebar mode (no `selection`): a button that opens the execution, with an
 *   optional bulk-selection checkbox. Behavior is identical to the previous
 *   inlined markup.
 * - Pick mode (`selection.selectionMode`): renders inside PickModeRow.
 */
import { memo } from "react";
import { cn } from "../../lib/utils";
import { formatRelativeTime } from "../../lib/format-utils";
import type { ExecutionRecord } from "../../types";
import { PickModeRow } from "../session/context/selectable-card";
import type { CardSelection } from "../session/context/selectable";

const STATUS_COLORS: Record<string, string> = {
  pending: "bg-slate-700/60 text-slate-300",
  starting: "bg-blue-500/20 text-blue-300",
  running: "bg-cyan-500/20 text-cyan-300",
  needs_review: "bg-amber-500/20 text-amber-300",
  validating: "bg-blue-500/20 text-blue-300",
  needs_fixup: "bg-amber-500/20 text-amber-300",
  completed: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
  canceled: "bg-slate-700/40 text-slate-500",
};

const MODE_LABELS: Record<string, string> = {
  manual: "Manual",
  yolo: "YOLO",
};

export interface ExecutionSummaryCardProps {
  item: ExecutionRecord;
  // Sidebar mode
  onOpen?: () => void;
  batchMode?: boolean;
  batchSelected?: boolean;
  onBatchToggle?: () => void;
  // Picker pick mode
  selection?: CardSelection;
}

function ExecutionCardBody({ item }: { item: ExecutionRecord }) {
  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
          {item.backlogName}
        </p>
        <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[item.status] ?? "bg-slate-700/60 text-slate-300")}>
          {item.status.replace(/_/g, " ")}
        </span>
      </div>
      <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-500">
        <span>{MODE_LABELS[item.mode] ?? item.mode}</span>
        <span>{formatRelativeTime(item.createdAt)}</span>
      </div>
    </>
  );
}

function ExecutionSummaryCardImpl({
  item,
  onOpen,
  batchMode = false,
  batchSelected = false,
  onBatchToggle,
  selection,
}: ExecutionSummaryCardProps) {
  if (selection?.selectionMode) {
    return (
      <PickModeRow selection={selection}>
        <ExecutionCardBody item={item} />
      </PickModeRow>
    );
  }

  return (
    <button
      type="button"
      onClick={() => onOpen?.()}
      className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
      data-testid="sidebar-execution-item"
    >
      <div className="flex items-start gap-2">
        {batchMode && (
          <input
            type="checkbox"
            aria-label={`${batchSelected ? "Deselect" : "Select"} execution ${item.backlogName}`}
            checked={batchSelected}
            onClick={(event) => event.stopPropagation()}
            onChange={(event) => {
              event.stopPropagation();
              onBatchToggle?.();
            }}
            className="mt-0.5"
          />
        )}
        <div className="min-w-0 flex-1">
          <ExecutionCardBody item={item} />
        </div>
      </div>
    </button>
  );
}

export const ExecutionSummaryCard = memo(ExecutionSummaryCardImpl);
