/**
 * RunCard - Displays a single agent run summary in the AttachRunModal.
 */

import type { AgentRunSummary } from "../../lib/api";

export const STATUS_COLORS: Record<string, string> = {
  pending: "bg-yellow-500/20 text-yellow-400",
  starting: "bg-yellow-500/20 text-yellow-400",
  running: "bg-blue-500/20 text-blue-400",
  needs_review: "bg-orange-500/20 text-orange-400",
  complete: "bg-green-500/20 text-green-400",
  failed: "bg-red-500/20 text-red-400",
  cancelled: "bg-slate-500/20 text-slate-400",
};

export const STATUS_OPTIONS = [
  { value: "", label: "All statuses" },
  { value: "running", label: "Running" },
  { value: "pending", label: "Pending" },
  { value: "complete", label: "Complete" },
  { value: "failed", label: "Failed" },
  { value: "needs_review", label: "Needs Review" },
  { value: "cancelled", label: "Cancelled" },
];

export function formatTime(dateStr: string): string {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
}

interface RunCardProps {
  run: AgentRunSummary;
  isSelected?: boolean;
  onClick?: () => void;
}

export function RunCard({
  run,
  isSelected = false,
  onClick,
}: RunCardProps) {
  const Comp = onClick ? "button" : "div";
  return (
    <Comp
      type={onClick ? "button" : undefined}
      onClick={onClick}
      className={`
        w-full text-left p-3 rounded-lg border transition-colors
        ${isSelected
          ? "border-blue-500/50 bg-blue-500/10"
          : "border-white/10 hover:border-white/20 hover:bg-white/5"
        }
        ${onClick ? "cursor-pointer" : ""}
      `}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium text-white truncate">
          {run.tag || run.run_id.slice(0, 12)}
        </span>
        <span className={`px-2 py-0.5 rounded-full text-xs font-medium shrink-0 ${STATUS_COLORS[run.status] || "bg-slate-500/20 text-slate-400"}`}>
          {run.status}
        </span>
      </div>
      <div className="flex items-center gap-3 mt-1 text-xs text-slate-500">
        <span>{formatTime(run.created_at)}</span>
        {run.status === "running" && run.progress_percent > 0 && (
          <span>{run.progress_percent}%</span>
        )}
        <span className="truncate">{run.run_id.slice(0, 8)}</span>
      </div>
    </Comp>
  );
}
