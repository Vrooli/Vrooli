// DOC: docs/reference/configuration.md
// Shared UI constants — single source of truth for tunable client-side values.

/** How often to poll the /health endpoint for general status (ms). */
export const HEALTH_POLL_INTERVAL_MS = 10_000;

/** How often to poll /health for the analytics dashboard (ms). */
export const METRICS_POLL_INTERVAL_MS = 5_000;

/** Max events retained in the live stream buffer. */
export const STREAM_MAX_EVENTS = 200;

/** Available query-limit options for the event log page. */
export const QUERY_LIMIT_OPTIONS = [25, 50, 100, 500] as const;

/** CSS class for styled form inputs (filter fields, selects). */
export const INPUT_CLASS =
  "rounded-lg border border-[var(--border-default)] bg-[var(--surface-inset)] px-3 py-2 text-sm text-[var(--text-secondary)] placeholder-[var(--text-faint)] outline-none focus:border-[var(--text-accent)]";

/** Health status to indicator color mapping. */
export const STATUS_COLORS: Record<string, string> = {
  healthy: "bg-[var(--status-healthy-bright)]",
  degraded: "bg-[var(--status-degraded)]",
  unhealthy: "bg-[var(--status-unhealthy-bright)]",
  unknown: "bg-[var(--status-unknown)]",
};
