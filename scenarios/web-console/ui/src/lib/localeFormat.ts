/**
 * Dates, durations and token counts in the app's language.
 *
 * Every launcher surface formats through here so a reset, a check time and a
 * "last seen" can never appear in three different shapes. Callers pass
 * `i18n.resolvedLanguage`.
 */
type Locale = string | undefined;

const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

function parse(value: string): Date | null {
  const at = Date.parse(value);
  return Number.isFinite(at) ? new Date(at) : null;
}

/** An absolute moment with its day, such as "Wed, Nov 4, 10:00 AM". Empty when unreadable. */
export function formatDateTime(value: string, locale: Locale): string {
  const date = parse(value);
  if (!date) return "";
  return new Intl.DateTimeFormat(locale, { weekday: "short", month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(date);
}

/** A calendar date with its year, for facts that are days old. Empty when unreadable. */
export function formatDate(value: string, locale: Locale): string {
  const date = parse(value);
  return date ? new Intl.DateTimeFormat(locale, { dateStyle: "medium" }).format(date) : "";
}

/** A time today, or its weekday and time, because "08:47 PM" alone reads as tonight. */
export function formatClockTime(value: string, locale: Locale, now = Date.now()): string {
  const date = parse(value);
  if (!date) return "";
  const sameDay = date.toDateString() === new Date(now).toDateString();
  return new Intl.DateTimeFormat(locale, sameDay ? { hour: "numeric", minute: "2-digit" } : { weekday: "short", hour: "numeric", minute: "2-digit" }).format(date);
}

type DurationPart = { value: number; unit: "minute" | "hour" | "day" };

function durationPart(ms: number): DurationPart {
  if (ms >= DAY_MS) return { value: Math.round(ms / DAY_MS), unit: "day" };
  if (ms >= HOUR_MS) return { value: Math.round(ms / HOUR_MS), unit: "hour" };
  return { value: Math.max(1, Math.round(ms / MINUTE_MS)), unit: "minute" };
}

function formatUnit({ value, unit }: DurationPart, locale: Locale, unitDisplay: "narrow" | "long"): string {
  return new Intl.NumberFormat(locale, { style: "unit", unit, unitDisplay }).format(value);
}

/** The smallest readable span: "4d", "5h", "35m". */
export function formatShortDuration(ms: number, locale: Locale): string {
  return formatUnit(durationPart(ms), locale, "narrow");
}

/** The same span for a sentence or a screen reader: "4 days". */
export function formatLongDuration(ms: number, locale: Locale): string {
  return formatUnit(durationPart(ms), locale, "long");
}

/** "3 minutes ago", or "" when the time cannot be read. */
export function formatAge(value: string, locale: Locale, now = Date.now()): string {
  const date = parse(value);
  if (!date) return "";
  const { value: amount, unit } = durationPart(Math.max(0, now - date.getTime()));
  return new Intl.RelativeTimeFormat(locale, { numeric: "always" }).format(-amount, unit);
}

/** "in 4 days", or "" when the time cannot be read. */
export function formatTimeUntil(value: string, locale: Locale, now = Date.now()): string {
  const date = parse(value);
  if (!date) return "";
  const { value: amount, unit } = durationPart(Math.max(0, date.getTime() - now));
  return new Intl.RelativeTimeFormat(locale, { numeric: "always" }).format(amount, unit);
}

/** Token counts stay short: "300K", "1M". */
export function formatTokens(tokens: number, locale: Locale): string {
  return new Intl.NumberFormat(locale, { notation: "compact", maximumFractionDigits: 1 }).format(tokens);
}

export function formatPercent(value: number, locale: Locale): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(value);
}
