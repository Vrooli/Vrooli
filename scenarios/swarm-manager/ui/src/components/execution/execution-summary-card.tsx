/**
 * ExecutionSummaryCard — the compact execution summary shared by the sidebar
 * ExecutionsTab and the SessionContextPicker. This is the terse list row, NOT
 * the rich `ExecutionCard` used by ExecutionListView.
 *
 * It is row content only: the CollectionList row owns the chrome, selection,
 * and the open interaction. Wrap it in CollectionRow.
 */
import { memo } from "react";
import { cn } from "../../lib/utils";
import { formatRelativeTime } from "../../lib/format-utils";
import type { ExecutionRecord } from "../../types";

const STATUS_COLORS: Record<string, string> = {
  pending: "bg-slate-700/60 text-slate-300",
  starting: "bg-blue-500/20 text-blue-300",
  running: "bg-cyan-500/20 text-cyan-300",
  needs_review: "bg-amber-500/20 text-amber-300",
  validating: "bg-blue-500/20 text-blue-300",
  needs_fixup: "bg-amber-500/20 text-amber-300",
  completed: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
  cancelling: "bg-amber-500/20 text-amber-300",
  canceled: "bg-slate-700/40 text-slate-500",
};

const MODE_LABELS: Record<string, string> = {
  manual: "Manual",
  yolo: "YOLO",
};

export interface ExecutionSummaryCardProps {
  item: ExecutionRecord;
}

function ExecutionSummaryCardImpl({ item }: ExecutionSummaryCardProps) {
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

export const ExecutionSummaryCard = memo(ExecutionSummaryCardImpl);
