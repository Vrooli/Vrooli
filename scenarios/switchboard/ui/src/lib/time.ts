import { formatDate, formatRelativeTime } from "../i18n/format";

const UNITS: Array<[Intl.RelativeTimeFormatUnit, number]> = [
  ["year", 1000 * 60 * 60 * 24 * 365],
  ["month", 1000 * 60 * 60 * 24 * 30],
  ["week", 1000 * 60 * 60 * 24 * 7],
  ["day", 1000 * 60 * 60 * 24],
  ["hour", 1000 * 60 * 60],
  ["minute", 1000 * 60],
];

/**
 * Human relative time ("3 minutes ago", "in 2 hours"). Anything under a minute
 * collapses to "now"-style output from Intl so the console never shows a
 * ticking seconds counter.
 */
export function relativeTime(value: string | number | Date, now: number = Date.now()): string {
  const time = typeof value === "string" || value instanceof Date ? new Date(value).getTime() : value;
  if (Number.isNaN(time)) return "";
  const delta = time - now;
  for (const [unit, ms] of UNITS) {
    if (Math.abs(delta) >= ms) {
      return formatRelativeTime(Math.round(delta / ms), unit, { numeric: "auto" });
    }
  }
  return formatRelativeTime(0, "minute", { numeric: "auto" });
}

export function clockTime(value: string | number | Date): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return formatDate(date, { hour: "numeric", minute: "2-digit" });
}

export function dayLabel(value: string | number | Date): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return formatDate(date, { weekday: "long", month: "short", day: "numeric" });
}

export function sameDay(a: string | number | Date, b: string | number | Date): boolean {
  const da = new Date(a);
  const db = new Date(b);
  return da.getFullYear() === db.getFullYear() && da.getMonth() === db.getMonth() && da.getDate() === db.getDate();
}

/** Milliseconds until `value`; negative when already past. */
export function msUntil(value: string | number | Date, now: number = Date.now()): number {
  const time = new Date(value).getTime();
  return Number.isNaN(time) ? 0 : time - now;
}
