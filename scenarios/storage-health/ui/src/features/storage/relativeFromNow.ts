import { formatRelativeTime } from "../../i18n/format";

/**
 * Render an RFC3339 timestamp as a coarse, locale-aware relative time
 * ("3 minutes ago"). Falls back to an empty string for an unparseable / empty
 * value so callers can branch on "never scanned". Kept in its own non-component
 * module so the feature's presentation components stay fast-refresh clean.
 */
export function relativeFromNow(rfc3339: string): string {
  if (!rfc3339) return "";
  const then = new Date(rfc3339).getTime();
  if (Number.isNaN(then)) return "";
  const deltaSec = Math.round((then - Date.now()) / 1000);
  const abs = Math.abs(deltaSec);
  if (abs < 60) return formatRelativeTime(deltaSec, "second");
  const deltaMin = Math.round(deltaSec / 60);
  if (Math.abs(deltaMin) < 60) return formatRelativeTime(deltaMin, "minute");
  const deltaHr = Math.round(deltaMin / 60);
  if (Math.abs(deltaHr) < 24) return formatRelativeTime(deltaHr, "hour");
  const deltaDay = Math.round(deltaHr / 24);
  return formatRelativeTime(deltaDay, "day");
}
