/**
 * Status Colors
 *
 * Unified status-to-color mapping. Color encodes STATUS, not entity type.
 * Both fill (background) and border are colored to make status instantly scannable.
 */

export interface StatusColorClasses {
  /** Tailwind background class, e.g. "bg-blue-500/20" */
  background: string;
  /** Tailwind border class, e.g. "border-blue-400/80" */
  border: string;
  /** Tailwind text class for labels on this background */
  text: string;
}

type StatusGroup = "neutral" | "active" | "waiting" | "done" | "error" | "terminal";

// PERF: Backgrounds use 90% opacity (solid enough to hide the dot grid
// behind the node) instead of 20% + backdrop-blur-sm. backdrop-blur forces
// per-node GPU compositor layers with Gaussian blur every frame during
// pan/zoom — the single most expensive CSS property in the graph.
// 90% opaque on the dark slate-950 background is visually nearly identical.
const STATUS_GROUP_COLORS: Record<StatusGroup, StatusColorClasses> = {
  neutral: {
    background: "bg-slate-800/90",
    border: "border-slate-400/70",
    text: "text-slate-200",
  },
  active: {
    background: "bg-cyan-950/90",
    border: "border-cyan-400/80",
    text: "text-cyan-100",
  },
  waiting: {
    background: "bg-amber-950/90",
    border: "border-amber-400/80",
    text: "text-amber-100",
  },
  done: {
    background: "bg-emerald-950/90",
    border: "border-emerald-400/80",
    text: "text-emerald-100",
  },
  error: {
    background: "bg-red-950/90",
    border: "border-red-400/80",
    text: "text-red-100",
  },
  terminal: {
    background: "bg-slate-900/90",
    border: "border-slate-500/50",
    text: "text-slate-400",
  },
};

const STATUS_TO_GROUP: Record<string, StatusGroup> = {
  // Neutral
  pending: "neutral",
  backlog: "neutral",
  unknown: "neutral",
  stopped: "neutral",
  unspecified: "neutral",
  // Active
  in_progress: "active",
  running: "active",
  starting: "active",
  researching: "active",
  classifying: "active",
  validating: "active",
  // Waiting
  queued: "waiting",
  needs_review: "waiting",
  needs_fixup: "waiting",
  needs_review_run: "waiting",
  ready: "waiting",
  // Done
  completed: "done",
  complete: "done",
  classified: "done",
  // Error
  failed: "error",
  error: "error",
  // Terminal
  archived: "terminal",
  canceled: "terminal",
  cancelled: "terminal",
};

function getStatusGroup(status: string | undefined): StatusGroup {
  if (!status) return "neutral";
  return STATUS_TO_GROUP[status] ?? "neutral";
}

/** Get Tailwind CSS classes for a node's status-based coloring. */
export function getStatusColorClasses(status: string | undefined): StatusColorClasses {
  return STATUS_GROUP_COLORS[getStatusGroup(status)];
}

/**
 * Backlog statuses considered "actionable" — the attention highlight and
 * sidebar chips treat these as ready-for-operator-input.
 */
export const ACTIONABLE_BACKLOG_STATUSES: ReadonlySet<string> = new Set([
  "backlog", "researching", "ready", "queued", "in_progress", "failed",
]);

/** Whether a backlog status is actionable (would appear in the Operations lens). */
export function isActionableBacklogStatus(status: string | undefined): boolean {
  return status !== undefined && ACTIONABLE_BACKLOG_STATUSES.has(status);
}

/** All status groups with their display info, for use in legends/help panels. */
export const STATUS_GROUP_INFO: { group: StatusGroup; label: string; exampleStatuses: string[]; classes: StatusColorClasses }[] = [
  { group: "neutral", label: "Pending", exampleStatuses: ["pending", "backlog", "stopped"], classes: STATUS_GROUP_COLORS.neutral },
  { group: "active", label: "Active", exampleStatuses: ["in_progress", "running", "starting"], classes: STATUS_GROUP_COLORS.active },
  { group: "waiting", label: "Waiting", exampleStatuses: ["queued", "needs_review", "ready"], classes: STATUS_GROUP_COLORS.waiting },
  { group: "done", label: "Done", exampleStatuses: ["completed", "classified"], classes: STATUS_GROUP_COLORS.done },
  { group: "error", label: "Error", exampleStatuses: ["failed", "error"], classes: STATUS_GROUP_COLORS.error },
  { group: "terminal", label: "Archived", exampleStatuses: ["archived", "cancelled"], classes: STATUS_GROUP_COLORS.terminal },
];
