/**
 * Domain formatters for backup posture: byte sizes, durations, and the
 * relative "age" of a timestamp (e.g. "3 days ago", "never"). These build on
 * the locale-aware `i18n/format` primitives so output follows the user's
 * chosen locale. Keep all size/age/duration rendering here so the six surfaces
 * stay consistent and a single change restyles every metric.
 */
import { formatNumber, formatRelativeTime } from "../i18n/format";

const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB", "PB"] as const;

/**
 * Human byte size with a locale-aware mantissa (1.5 GB, 900 B). Accepts number
 * or bigint (proto int64 fields decode to bigint). Negative inputs clamp to 0.
 */
export function formatBytes(value: number | bigint): string {
  let bytes = typeof value === "bigint" ? Number(value) : value;
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return `0 ${BYTE_UNITS[0]}`;
  }
  let unit = 0;
  while (bytes >= 1024 && unit < BYTE_UNITS.length - 1) {
    bytes /= 1024;
    unit += 1;
  }
  const maximumFractionDigits = unit === 0 ? 0 : bytes < 10 ? 1 : 0;
  return `${formatNumber(bytes, { maximumFractionDigits })} ${BYTE_UNITS[unit]}`;
}

/**
 * Compact duration between two instants (e.g. "1m 12s", "340ms", "2h 5m").
 * Returns an em dash when either bound is missing (run still in flight).
 */
export function formatDuration(start?: Date, end?: Date): string {
  if (!start || !end) return "—";
  const ms = end.getTime() - start.getTime();
  if (!Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const totalSeconds = Math.round(ms / 1000);
  const h = Math.floor(totalSeconds / 3600);
  const m = Math.floor((totalSeconds % 3600) / 60);
  const s = totalSeconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

const AGE_THRESHOLDS: ReadonlyArray<[limitSeconds: number, divisor: number, unit: Intl.RelativeTimeFormatUnit]> = [
  [60, 1, "second"],
  [3600, 60, "minute"],
  [86_400, 3600, "hour"],
  [2_592_000, 86_400, "day"],
  [31_536_000, 2_592_000, "month"],
  [Number.POSITIVE_INFINITY, 31_536_000, "year"],
];

/**
 * Relative age of a past instant ("3 days ago"). `undefined` renders as the
 * provided `neverLabel` — used for "never backed up" / "never verified",
 * which is a first-class state in this product, not an empty value.
 */
export function formatAge(value: Date | undefined, neverLabel = "never", now: Date = new Date()): string {
  if (!value) return neverLabel;
  const deltaSeconds = (value.getTime() - now.getTime()) / 1000;
  const abs = Math.abs(deltaSeconds);
  for (const [limit, divisor, unit] of AGE_THRESHOLDS) {
    if (abs < limit) {
      return formatRelativeTime(Math.round(deltaSeconds / divisor), unit, { numeric: "auto" });
    }
  }
  return formatRelativeTime(Math.round(deltaSeconds / 31_536_000), "year", { numeric: "auto" });
}

/** True when `value` is older than `maxAgeMs` (used for the stale-verify chip). */
export function isOlderThan(value: Date | undefined, maxAgeMs: number, now: Date = new Date()): boolean {
  if (!value) return true;
  return now.getTime() - value.getTime() > maxAgeMs;
}
