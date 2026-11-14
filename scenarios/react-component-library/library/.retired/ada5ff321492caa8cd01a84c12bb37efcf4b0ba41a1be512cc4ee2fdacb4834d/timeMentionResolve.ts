import { AMBIGUOUS_ZONES, type TimeMentionData, type TimeMentionRole } from "./timeMentionGrammar";

/**
 * Resolution puts a detected mention into the reader's frame: their zone, the
 * current time, and — when the host knows it — when the text was written.
 *
 * The anchor is optional. Without it, everything that depends on when the text
 * was written (a missing date, `yesterday`, `in 20 minutes`) silently falls
 * away: the mention resolves to less, or to nothing, and renders as plain text.
 * Everything that only needs "now" keeps working.
 */

export type TimeMentionDisplay = "annotate" | "local" | "relative";
export const TIME_MENTION_DISPLAYS: readonly TimeMentionDisplay[] = [
  "annotate",
  "local",
  "relative",
];

export interface TimeFrame {
  /** When the text was written, in epoch milliseconds, or null when unknown. */
  anchor: number | null;
  now: number;
  timeZone: string;
  locale: string;
}

/** A translatable sentence: a string key, its English fallback, and its values. */
export interface TimeMentionText {
  key: string;
  fallback: string;
  values?: Readonly<Record<string, string>>;
}

export interface TimeMentionNote extends TimeMentionText {
  tone: "info" | "warning";
}

export interface ResolvedTimeMention {
  kind: TimeMentionData["kind"];
  /** The exact moment, when the text pins one down. */
  instant: number | null;
  /** False when only the clock time is known (a zoned time with no anchor). */
  dateKnown: boolean;
  showSeconds: boolean;
  /** The writer's UTC offset in minutes, when they stated one. */
  writtenOffset: number | null;
  /** A calendar day, for dates and relative day words (UTC noon of that day). */
  day: number | null;
  durationMs: number | null;
  role: TimeMentionRole | null;
  /** A short correction shown beside the text, e.g. "a Thursday". */
  lint: TimeMentionText | null;
  notes: TimeMentionNote[];
}

const HOUR = 3_600_000;
const DAY = 86_400_000;

interface ZonedParts {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  second: number;
}

const partsFormatters = new Map<string, Intl.DateTimeFormat>();

function zonedParts(ms: number, timeZone: string): ZonedParts {
  let formatter = partsFormatters.get(timeZone);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat("en-US", {
      timeZone,
      hourCycle: "h23",
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "numeric",
      minute: "numeric",
      second: "numeric",
    });
    partsFormatters.set(timeZone, formatter);
  }
  const parts: Record<string, number> = {};
  for (const part of formatter.formatToParts(ms)) parts[part.type] = Number(part.value);
  return {
    year: parts.year ?? 1970,
    month: parts.month ?? 1,
    day: parts.day ?? 1,
    hour: (parts.hour ?? 0) % 24,
    minute: parts.minute ?? 0,
    second: parts.second ?? 0,
  };
}

/** The reader's UTC offset in minutes at `ms`. */
export function zoneOffsetAt(ms: number, timeZone: string): number {
  const p = zonedParts(ms, timeZone);
  const asUtc = Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute, p.second);
  return Math.round((asUtc - Math.floor(ms / 1000) * 1000) / 60_000);
}

/** UTC noon of the reader's calendar day containing `ms`; days compare by equality. */
export function calendarDay(ms: number, timeZone: string): number {
  const p = zonedParts(ms, timeZone);
  return Date.UTC(p.year, p.month - 1, p.day, 12);
}

export function isNight(ms: number, timeZone: string): boolean {
  const { hour } = zonedParts(ms, timeZone);
  return hour >= 22 || hour < 6;
}

/** The occurrence of a zoned clock time nearest `base`, plus the next-nearest one. */
function nearestOccurrence(
  hour: number,
  minute: number,
  second: number,
  offset: number,
  base: number,
) {
  const shifted = new Date(base + offset * 60_000);
  const candidates = [-1, 0, 1]
    .map(
      (k) =>
        Date.UTC(
          shifted.getUTCFullYear(),
          shifted.getUTCMonth(),
          shifted.getUTCDate() + k,
          hour,
          minute,
          second,
        ) -
        offset * 60_000,
    )
    .sort((a, b) => Math.abs(a - base) - Math.abs(b - base));
  return { at: candidates[0] ?? base, alternative: candidates[1] ?? base };
}

function formatter(frame: TimeFrame, options: Intl.DateTimeFormatOptions) {
  return new Intl.DateTimeFormat(frame.locale, { timeZone: frame.timeZone, ...options });
}

export function formatClock(ms: number, frame: TimeFrame, seconds = false): string {
  return formatter(frame, {
    hour: "numeric",
    minute: "2-digit",
    ...(seconds ? { second: "2-digit" } : {}),
  }).format(ms);
}

export function formatDate(
  ms: number,
  frame: TimeFrame,
  weekday: "short" | "long" = "short",
): string {
  return formatter(frame, { weekday, month: "short", day: "numeric" }).format(ms);
}

/** A UTC-noon calendar day as a full date, independent of the reader's zone. */
export function formatCalendarDay(day: number, locale: string, withWeekday = true): string {
  return new Intl.DateTimeFormat(locale, {
    timeZone: "UTC",
    ...(withWeekday ? { weekday: "long" } : {}),
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(day);
}

export function zoneName(ms: number, frame: TimeFrame): string {
  const part = formatter(frame, { timeZoneName: "short" })
    .formatToParts(ms)
    .find((p) => p.type === "timeZoneName");
  return part?.value ?? frame.timeZone;
}

/** `today`, `tomorrow`, `in 3 days` — localized by Intl; empty when the day is today. */
export function formatDayDistance(day: number, frame: TimeFrame, omitToday = true): string {
  const days = Math.round((day - calendarDay(frame.now, frame.timeZone)) / DAY);
  if (days === 0 && omitToday) return "";
  return new Intl.RelativeTimeFormat(frame.locale, { numeric: "auto" }).format(days, "day");
}

/** The day word for an instant relative to now: "", "tomorrow", "yesterday", or a short date. */
export function formatDayLabel(ms: number, frame: TimeFrame): string {
  const days = Math.round(
    (calendarDay(ms, frame.timeZone) - calendarDay(frame.now, frame.timeZone)) / DAY,
  );
  if (days === 0) return "";
  if (Math.abs(days) === 1)
    return new Intl.RelativeTimeFormat(frame.locale, { numeric: "auto" }).format(days, "day");
  return formatter(frame, { month: "short", day: "numeric" }).format(ms);
}

/** A compact duration: `45s`, `42m`, `7h 26m`, `3d 4h`. */
export function formatDuration(ms: number): string {
  const seconds = Math.round(Math.abs(ms) / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 48) return minutes % 60 ? `${hours}h ${minutes % 60}m` : `${hours}h`;
  const days = Math.floor(hours / 24);
  return hours % 24 ? `${days}d ${hours % 24}h` : `${days}d`;
}

/** `in 7h 26m`, `2h ago`, or `now`, as a translatable sentence. */
export function relativeText(ms: number, now: number): TimeMentionText {
  const delta = ms - now;
  if (Math.abs(delta) < 45_000) return { key: "data-display.time-mention.now", fallback: "now" };
  const duration = formatDuration(delta);
  return delta > 0
    ? { key: "data-display.time-mention.in", fallback: "in {duration}", values: { duration } }
    : { key: "data-display.time-mention.ago", fallback: "{duration} ago", values: { duration } };
}

/** The deadline badge: `in 42m`, `passed 2h ago`, or `was due 3d ago`. */
export function deadlineText(instant: number, role: TimeMentionRole, now: number): TimeMentionText {
  const delta = instant - now;
  const duration = formatDuration(delta);
  if (delta >= 0)
    return { key: "data-display.time-mention.in", fallback: "in {duration}", values: { duration } };
  return role === "promise"
    ? {
        key: "data-display.time-mention.was-due",
        fallback: "was due {duration} ago",
        values: { duration },
      }
    : {
        key: "data-display.time-mention.passed",
        fallback: "passed {duration} ago",
        values: { duration },
      };
}

export type DeadlineState = "upcoming" | "soon" | "past";
export function deadlineState(instant: number, now: number): DeadlineState {
  const delta = instant - now;
  return delta < 0 ? "past" : delta < HOUR ? "soon" : "upcoming";
}

const WEEKDAY_NAMES = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];

const base = (
  kind: TimeMentionData["kind"],
  role: TimeMentionRole | null,
): ResolvedTimeMention => ({
  kind,
  instant: null,
  dateKnown: true,
  showSeconds: false,
  writtenOffset: null,
  day: null,
  durationMs: null,
  role,
  lint: null,
  notes: [],
});

/**
 * Put a mention into the reader's frame. Returns null when the mention means
 * nothing without information the host did not provide, or when annotating it
 * would add nothing; the caller then renders the text as written.
 */
export function resolveTimeMention(
  data: TimeMentionData,
  role: TimeMentionRole | null,
  frame: TimeFrame,
): ResolvedTimeMention | null {
  const { anchor } = frame;
  switch (data.kind) {
    case "instant": {
      const resolved = {
        ...base("instant", role),
        instant: data.at,
        showSeconds: data.seconds,
        writtenOffset: data.offset,
      };
      if (data.source === "epoch")
        resolved.notes.push({
          tone: "info",
          key: "data-display.time-mention.note.epoch",
          fallback: "Read as a Unix timestamp because the words before it name a time.",
        });
      return resolved;
    }
    case "clock": {
      const second = data.second ?? 0;
      const resolved = {
        ...base("clock", role),
        showSeconds: data.second !== null,
        writtenOffset: data.offset,
      };
      const ambiguity = AMBIGUOUS_ZONES[data.zone];
      if (anchor === null) {
        // The clock converts on any day; which day it is cannot be known.
        resolved.instant = nearestOccurrence(
          data.hour,
          data.minute,
          second,
          data.offset,
          frame.now,
        ).at;
        resolved.dateKnown = false;
        resolved.role = null;
        resolved.notes.push({
          tone: "info",
          key: "data-display.time-mention.note.no-date",
          fallback: "No date was written, so only the clock time is converted.",
        });
      } else {
        const { at, alternative } = nearestOccurrence(
          data.hour,
          data.minute,
          second,
          data.offset,
          anchor,
        );
        resolved.instant = at;
        resolved.notes.push({
          tone: "info",
          key: "data-display.time-mention.note.date-inferred",
          fallback:
            "No date was written. This is the occurrence nearest to when it was sent, at {sent}.",
          values: { sent: `${formatDate(anchor, frame)}, ${formatClock(anchor, frame)}` },
        });
        // The other reading is 24h minus this one's distance away; only warn
        // when the two are close enough that either could be meant.
        if (Math.abs(at - anchor) > 9 * HOUR)
          resolved.notes.push({
            tone: "warning",
            key: "data-display.time-mention.note.date-uncertain",
            fallback: "It could also mean {alternative}.",
            values: {
              alternative: `${formatDate(alternative, frame)}, ${formatClock(alternative, frame)}`,
            },
          });
      }
      if (ambiguity)
        resolved.notes.push({
          tone: "warning",
          key: "data-display.time-mention.note.ambiguous-zone",
          fallback: "{zone} is read as {reading}. It can also mean {others}.",
          values: {
            zone: data.zone,
            reading: ambiguity[0] ?? data.zone,
            others: ambiguity.slice(1).join(" or "),
          },
        });
      return resolved;
    }
    case "date": {
      const reference = anchor ?? frame.now;
      const referenceYear = new Date(calendarDay(reference, frame.timeZone)).getUTCFullYear();
      const year = data.year ?? referenceYear;
      const day = Date.UTC(year, data.month - 1, data.day, 12);
      const resolved = { ...base("date", null), day };
      const actual = new Date(day).getUTCDay();
      if (data.weekday !== null && data.weekday !== actual) {
        const weekday = WEEKDAY_NAMES[actual] ?? "";
        resolved.lint = {
          key: "data-display.time-mention.lint.weekday",
          fallback: "a {weekday}",
          values: { weekday },
        };
        resolved.notes.push({
          tone: "warning",
          key: "data-display.time-mention.note.weekday",
          fallback: "{date} is a {weekday}, not the weekday written.",
          values: { date: formatCalendarDay(day, frame.locale, false), weekday },
        });
      }
      if (data.year !== null && Math.abs(data.year - referenceYear) > 1) {
        resolved.lint ??= {
          key: "data-display.time-mention.lint.year",
          fallback: "{year}?",
          values: { year: String(data.year) },
        };
        resolved.notes.push({
          tone: "warning",
          key:
            anchor === null
              ? "data-display.time-mention.note.year-now"
              : "data-display.time-mention.note.year-sent",
          fallback:
            anchor === null
              ? "The year is far from the current year."
              : "The year is far from when this was sent.",
        });
      }
      return resolved;
    }
    case "relative-day": {
      if (anchor === null) return null;
      const day = calendarDay(anchor + data.shift * DAY, frame.timeZone);
      // While the reader is still on the writer's day, the word is correct as written.
      if (calendarDay(frame.now, frame.timeZone) === calendarDay(anchor, frame.timeZone))
        return null;
      const resolved = { ...base("relative-day", null), day };
      resolved.notes.push({
        tone: "info",
        key: "data-display.time-mention.note.relative-day",
        fallback: "Relative to when it was sent, on {sent}.",
        values: { sent: formatDate(anchor, frame, "long") },
      });
      return resolved;
    }
    case "relative-offset": {
      if (anchor === null) return null;
      const resolved = { ...base("relative-offset", role), instant: anchor + data.ms };
      resolved.notes.push({
        tone: "info",
        key: "data-display.time-mention.note.relative-offset",
        fallback: "Measured from when it was sent, at {sent}.",
        values: { sent: formatClock(anchor, frame) },
      });
      return resolved;
    }
    case "duration":
      return { ...base("duration", null), durationMs: data.ms };
    case "unzoned": {
      const resolved = { ...base("unzoned", null), dateKnown: data.date !== null };
      const reference = anchor ?? frame.now;
      const asUtc = data.date
        ? Date.UTC(
            data.date.year,
            data.date.month - 1,
            data.date.day,
            data.hour,
            data.minute,
            data.second ?? 0,
          )
        : nearestOccurrence(data.hour, data.minute, data.second ?? 0, 0, reference).at;
      if (data.date)
        resolved.day = Date.UTC(data.date.year, data.date.month - 1, data.date.day, 12);
      resolved.notes.push(
        {
          tone: "info",
          key: "data-display.time-mention.note.unzoned",
          fallback: "No time zone was written, so it isn't converted.",
        },
        {
          tone: "info",
          key: "data-display.time-mention.note.if-utc",
          fallback: "If it means UTC, that's {time} for you.",
          values: {
            time:
              formatClock(asUtc, frame) +
              (data.date ? ` ${formatDayLabel(asUtc, frame)}`.trimEnd() : ""),
          },
        },
      );
      return resolved;
    }
  }
}
