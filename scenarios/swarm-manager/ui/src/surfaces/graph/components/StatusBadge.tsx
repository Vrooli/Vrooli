/**
 * StatusBadge - Small execution status dot overlay for topology backlog nodes.
 */

import { cn } from "../../../lib/utils";

interface StatusBadgeProps {
  executionStatus: string | undefined;
}

const STATUS_STYLES: Record<string, string> = {
  running: "bg-cyan-400 animate-pulse",
  starting: "bg-cyan-400 animate-pulse",
  needs_review: "bg-amber-400",
  needs_fixup: "bg-amber-400",
  validating: "bg-amber-400",
  pending: "bg-slate-400",
  scheduled: "bg-slate-400",
  failed: "bg-red-400",
};

export function StatusBadge({ executionStatus }: StatusBadgeProps) {
  if (!executionStatus) return null;
  const style = STATUS_STYLES[executionStatus] ?? "bg-slate-400";

  return (
    <div
      className={cn("absolute -right-1 -top-1 h-2.5 w-2.5 rounded-full border border-slate-900", style)}
      title={`Execution: ${executionStatus.replace(/_/g, " ")}`}
      data-testid="status-badge"
    />
  );
}
