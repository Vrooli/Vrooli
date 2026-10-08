import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";

import { formatDate } from "../i18n/format";

/**
 * Format a proto Timestamp as a locale-aware date+time string, or an em dash
 * when absent. Central so every detail view renders times identically.
 */
export function formatTimestamp(ts?: Timestamp): string {
  if (!ts) return "—";
  return formatDate(timestampDate(ts), { dateStyle: "medium", timeStyle: "short" });
}

/**
 * Human-readable elapsed time between two proto Timestamps (a validation run's
 * start and finish). Returns an em dash when either endpoint is missing.
 */
export function formatDuration(start?: Timestamp, finish?: Timestamp): string {
  if (!start || !finish) return "—";
  const ms = timestampDate(finish).getTime() - timestampDate(start).getTime();
  if (!Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  const totalSeconds = Math.round(ms / 1000);
  if (totalSeconds < 60) return `${totalSeconds}s`;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes < 60) return seconds === 0 ? `${minutes}m` : `${minutes}m ${seconds}s`;
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  return remMinutes === 0 ? `${hours}h` : `${hours}h ${remMinutes}m`;
}
