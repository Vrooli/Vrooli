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

/** RGB color string for use in SVG/Canvas contexts (e.g. MiniMap). */
export interface StatusRgb {
  rgb: string;
}

type StatusGroup = "neutral" | "active" | "waiting" | "done" | "error" | "terminal";

const STATUS_GROUP_COLORS: Record<StatusGroup, StatusColorClasses> = {
  neutral: {
    background: "bg-slate-500/20",
    border: "border-slate-400/70",
    text: "text-slate-200",
  },
  active: {
    background: "bg-cyan-500/20",
    border: "border-cyan-400/80",
    text: "text-cyan-100",
  },
  waiting: {
    background: "bg-amber-500/20",
    border: "border-amber-400/80",
    text: "text-amber-100",
  },
  done: {
    background: "bg-emerald-500/20",
    border: "border-emerald-400/80",
    text: "text-emerald-100",
  },
  error: {
    background: "bg-red-500/20",
    border: "border-red-400/80",
    text: "text-red-100",
  },
  terminal: {
    background: "bg-slate-600/20",
    border: "border-slate-500/50",
    text: "text-slate-400",
  },
};

const STATUS_GROUP_RGB: Record<StatusGroup, string> = {
  neutral: "rgb(148 163 184 / 0.6)",
  active: "rgb(34 211 238 / 0.6)",
  waiting: "rgb(251 191 36 / 0.6)",
  done: "rgb(52 211 153 / 0.6)",
  error: "rgb(248 113 113 / 0.6)",
  terminal: "rgb(100 116 139 / 0.4)",
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
  scheduled: "waiting",
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

/** Get an RGB color string for SVG/Canvas contexts (e.g. MiniMap nodeColor). */
export function getStatusRgb(status: string | undefined): string {
  return STATUS_GROUP_RGB[getStatusGroup(status)];
}

/**
 * Backlog statuses considered "actionable" — these items appear in the Operations lens.
 * Mirrors the Go-side `actionableBacklogStatuses` in projection.go.
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
