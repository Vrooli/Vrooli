import { formatDate, formatRelativeTime } from "../i18n/format";

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/**
 * How long ago `value` was, in the reader's language ("12 minutes ago",
 * "yesterday"). Past a week the calendar date reads better than a count, so the
 * short date is returned instead. Returns null when `value` is not a time.
 */
export function formatAgo(value: string | undefined, now: number = Date.now()): string | null {
  if (!value) return null;
  const at = Date.parse(value);
  if (Number.isNaN(at)) return null;
  const elapsed = now - at;
  // A clock a little ahead of the server must not read as "in 2 minutes".
  if (elapsed < MINUTE) return formatRelativeTime(0, "second", { numeric: "auto" });
  if (elapsed < HOUR) return formatRelativeTime(-Math.floor(elapsed / MINUTE), "minute", { numeric: "auto" });
  if (elapsed < DAY) return formatRelativeTime(-Math.floor(elapsed / HOUR), "hour", { numeric: "auto" });
  if (elapsed < 7 * DAY) return formatRelativeTime(-Math.floor(elapsed / DAY), "day", { numeric: "auto" });
  const sameYear = new Date(at).getFullYear() === new Date(now).getFullYear();
  return formatDate(at, sameYear ? { month: "short", day: "numeric" } : { year: "numeric", month: "short", day: "numeric" });
}

/** The full local date, time and zone for a timestamp, for a tooltip or a record. */
export function formatExact(value: string | undefined): string | null {
  if (!value) return null;
  const at = Date.parse(value);
  if (Number.isNaN(at)) return null;
  return formatDate(at, { year: "numeric", month: "short", day: "numeric", hour: "numeric", minute: "2-digit", timeZoneName: "short" });
}

/** "in 2 hours" or "5 minutes ago" for a deadline; null when `value` is not a time. */
export function formatDueIn(value: string | undefined, now: number = Date.now()): string | null {
  if (!value) return null;
  const at = Date.parse(value);
  if (Number.isNaN(at)) return null;
  const diff = at - now;
  const magnitude = Math.abs(diff);
  const sign = diff < 0 ? -1 : 1;
  if (magnitude < HOUR) return formatRelativeTime(sign * Math.max(1, Math.round(magnitude / MINUTE)), "minute", { numeric: "auto" });
  if (magnitude < 2 * DAY) return formatRelativeTime(sign * Math.round(magnitude / HOUR), "hour", { numeric: "auto" });
  return formatRelativeTime(sign * Math.round(magnitude / DAY), "day", { numeric: "auto" });
}
