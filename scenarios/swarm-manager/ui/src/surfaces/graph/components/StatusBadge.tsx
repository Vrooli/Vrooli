/**
 * StatusBadge - Execution status dot overlay for topology backlog nodes (right tip).
 * ActionableBadge - Actionable indicator for topology backlog nodes (left tip).
 *
 * Both badges are absolutely positioned within the GraphNode outer wrapper,
 * rendered AFTER the shape div so they paint above the backdrop-blur layer.
 *
 * For diamond nodes (100x100 layout box, visually rotated 45deg):
 *   - The diamond surface touches the layout box only at edge midpoints (the tips).
 *   - Badges use top-1/2 -translate-y-1/2 to sit at the left/right tips.
 *
 * Both use solid backgrounds + dark border-slate-900 for a clean glow effect.
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
      className={cn(
        "absolute -right-2 top-1/2 -translate-y-1/2 h-4 w-4 rounded-full border-2 border-slate-900",
        style,
      )}
      title={`Execution: ${executionStatus.replace(/_/g, " ")}`}
      data-testid="status-badge"
    />
  );
}

interface ActionableBadgeProps {
  status: string;
}

/** Solid background colors with dark border for glow effect matching StatusBadge. */
const ACTIONABLE_STYLES: Record<string, string> = {
  backlog: "bg-slate-400",
  researching: "bg-cyan-400",
  in_progress: "bg-cyan-400",
  ready: "bg-amber-400",
  queued: "bg-amber-400",
  failed: "bg-red-400",
};

const ACTIONABLE_FALLBACK = "bg-slate-400";

export function ActionableBadge({ status }: ActionableBadgeProps) {
  const style = ACTIONABLE_STYLES[status] ?? ACTIONABLE_FALLBACK;
  return (
    <div
      className={cn(
        "absolute -left-2 top-1/2 -translate-y-1/2 h-4 w-4 rounded-full border-2 border-slate-900",
        style,
      )}
      title={`Actionable: ${status.replace(/_/g, " ")}`}
      data-testid="actionable-badge"
    />
  );
}
