/**
 * The time-mention grammar: finds the times, dates and durations a writer put
 * in prose, and records only what the text itself says.
 *
 * Nothing here knows the reader's zone, the current time, or when the text was
 * written; `resolveTimeMention` adds that frame. Keeping detection frame-free
 * lets a markdown pass run once per document and stay memoizable.
 *
 * The grammar is deliberately strict. A missed mention costs nothing — the
 * text renders as written — while a false one puts a wrong time next to a
 * version number or an identifier. Every pattern therefore needs a structural
 * signal (an explicit zone, an ISO shape, a context keyword, a preposition)
 * before it matches.
 */

export type TimeMentionRole = "deadline" | "promise";

export interface CalendarDate {
  year: number;
  month: number;
  day: number;
}

export type TimeMentionData =
  /** A complete moment: ISO 8601 with a zone, or a Unix timestamp. */
  | {
      kind: "instant";
      source: "iso" | "epoch";
      at: number;
      seconds: boolean;
      offset: number | null;
    }
  /** A clock time with an explicit zone but no date, e.g. `08:53:33 UTC`. */
  | {
      kind: "clock";
      hour: number;
      minute: number;
      second: number | null;
      offset: number;
      zone: string;
    }
  /** A calendar date, optionally with the weekday the writer claimed for it. */
  | { kind: "date"; year: number | null; month: number; day: number; weekday: number | null }
  /** `today`, `yesterday`, `tomorrow`: a day relative to when the text was written. */
  | { kind: "relative-day"; shift: number }
  /** `in 20 minutes`, `3 hours ago`: an offset from when the text was written. */
  | { kind: "relative-offset"; ms: number }
  /** A duration whose raw form is hard to read: `7380s`, `PT90M`. */
  | { kind: "duration"; ms: number }
  /** A clock time with no zone, e.g. `at 9:30`, optionally with an ISO date. */
  | {
      kind: "unzoned";
      hour: number;
      minute: number;
      second: number | null;
      date: CalendarDate | null;
    };

export interface TimeMentionMatch {
  start: number;
  end: number;
  text: string;
  data: TimeMentionData;
  role: TimeMentionRole | null;
}

/** Zone abbreviations and their UTC offsets in minutes. */
export const ZONE_OFFSETS: Readonly<Record<string, number>> = {
  UTC: 0,
  GMT: 0,
  Z: 0,
  EST: -300,
  EDT: -240,
  CST: -360,
  CDT: -300,
  MST: -420,
  MDT: -360,
  PST: -480,
  PDT: -420,
  AKST: -540,
  AKDT: -480,
  HST: -600,
  BST: 60,
  IST: 330,
  CET: 60,
  CEST: 120,
  EET: 120,
  EEST: 180,
  JST: 540,
  KST: 540,
  AEST: 600,
  AEDT: 660,
};

/** Abbreviations that name more than one zone; the first entry is the reading used. */
export const AMBIGUOUS_ZONES: Readonly<Record<string, readonly string[]>> = {
  CST: ["US Central (UTC−6)", "China Standard (UTC+8)"],
  IST: ["India (UTC+5:30)", "Irish (UTC+1)", "Israel (UTC+2)"],
  BST: ["British Summer (UTC+1)", "Bangladesh (UTC+6)"],
};

const ZONE_PATTERN =
  "UTC|GMT|Z|EST|EDT|CST|CDT|MST|MDT|PST|PDT|AKST|AKDT|HST|BST|IST|CET|CEST|EET|EEST|JST|KST|AEST|AEDT|[+-](?:0\\d|1[0-4]):?[0-5]\\d";

const MONTHS = ["jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"];
const WEEKDAYS = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];
const UNIT_MS: Readonly<Record<string, number>> = {
  m: 60_000,
  h: 3_600_000,
  d: 86_400_000,
  w: 604_800_000,
};

/** The offset in minutes for an abbreviation or `±HH:MM`, or null when unknown. */
export function zoneOffsetMinutes(zone: string): number | null {
  const normalized = zone.trim().toUpperCase();
  if (normalized in ZONE_OFFSETS) return ZONE_OFFSETS[normalized] ?? null;
  const match = /^([+-])(\d{2}):?(\d{2})$/.exec(normalized);
  if (!match) return null;
  const minutes = Number(match[2]) * 60 + Number(match[3]);
  return match[1] === "-" ? -minutes : minutes;
}

/** A numeric capture group; an unmatched group reads as 0. */
function group(match: RegExpExecArray, index: number): number {
  return Number(match[index] ?? 0);
}

function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

function to24Hour(hour: number, meridiem: string | undefined): number | null {
  if (!meridiem) return hour;
  if (hour < 1 || hour > 12) return null;
  const pm = meridiem.toLowerCase().startsWith("p");
  return (hour % 12) + (pm ? 12 : 0);
}

function deadlineRole(text: string, start: number, end: number): TimeMentionRole | null {
  const before = text.slice(Math.max(0, start - 40), start);
  const after = text.slice(end, end + 24);
  return /\b(?:cutoff|deadline|until|expires?|expiry|due|closes?|ends?|by)\b[^.;\n]{0,16}$/i.test(
    before,
  ) || /^[^.;\n]{0,20}\b(?:cutoff|deadline)\b/i.test(after)
    ? "deadline"
    : null;
}

interface Pattern {
  regex: RegExp;
  /** Characters to skip at the start of the match, e.g. a leading preposition. */
  lead?: (match: RegExpExecArray) => number;
  build: (match: RegExpExecArray, text: string) => TimeMentionData | null;
}

const PATTERNS: readonly Pattern[] = [
  // ISO 8601 date-time. With a zone it is a complete moment; without one the
  // writer's zone is unknown, so it is reported as unzoned rather than guessed.
  {
    regex: new RegExp(
      `(?<![\\w/.-])(\\d{4})-(\\d{2})-(\\d{2})[T ](\\d{2}):(\\d{2})(?::(\\d{2})(?:\\.\\d+)?)?(?:\\s?(Z|UTC|[+-]\\d{2}:?\\d{2}))?(?![\\w:])`,
      "g",
    ),
    build: (m) => {
      const [year, month, day, hour, minute] = [
        group(m, 1),
        group(m, 2),
        group(m, 3),
        group(m, 4),
        group(m, 5),
      ];
      const second = m[6] === undefined ? null : Number(m[6]);
      if (
        month < 1 ||
        month > 12 ||
        day < 1 ||
        day > daysInMonth(year, month) ||
        hour > 23 ||
        minute > 59
      )
        return null;
      if (second !== null && second > 59) return null;
      if (!m[7]) return { kind: "unzoned", hour, minute, second, date: { year, month, day } };
      const offset = zoneOffsetMinutes(m[7]);
      if (offset === null) return null;
      const at = Date.UTC(year, month - 1, day, hour, minute, second ?? 0) - offset * 60_000;
      return { kind: "instant", source: "iso", at, seconds: second !== null, offset };
    },
  },
  // Unix timestamps between 2017 and 2033, only beside a word that says the
  // number is a time. A bare ten-digit number is far more often an id or size.
  {
    regex: /\b(1[5-9]\d{8})(\d{3})?\b/g,
    build: (m, text) => {
      const context = text.slice(Math.max(0, m.index - 24), m.index);
      if (
        !/(?:\b(?:ts|time|timestamp|epoch|unix|since|until|at|expires?|expiry|created|updated|deadline)|_(?:ts|at|time))\s*[:=]?\s*["']?$/i.test(
          context,
        )
      )
        return null;
      const at = m[2] ? Number(`${m[1] ?? ""}${m[2]}`) : group(m, 1) * 1000;
      return { kind: "instant", source: "epoch", at, seconds: true, offset: null };
    },
  },
  // A clock time with an explicit zone: `08:53:33 UTC`, `2:00 PM EDT`, `14:00+02:00`.
  {
    regex: new RegExp(
      `(?<![\\w:.])([01]?\\d|2[0-3]):([0-5]\\d)(?::([0-5]\\d))?(?:\\s?([AaPp]\\.?[Mm]\\.?))?\\s?(${ZONE_PATTERN})(?![\\w:])`,
      "g",
    ),
    build: (m) => {
      const hour = to24Hour(Number(m[1]), m[4]);
      const zone = (m[5] ?? "").toUpperCase();
      const offset = zoneOffsetMinutes(zone);
      if (hour === null || offset === null) return null;
      return {
        kind: "clock",
        hour,
        minute: Number(m[2]),
        second: m[3] === undefined ? null : Number(m[3]),
        offset,
        zone,
      };
    },
  },
  // ISO 8601 durations: `PT90M`, `P1DT2H`.
  {
    regex: /\bP(?:(\d+)D)?(?:T(?=\d)(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?\b/g,
    build: (m) => {
      if (!m[1] && !m[2] && !m[3] && !m[4]) return null;
      const ms =
        (Number(m[1] ?? 0) * 86_400 +
          Number(m[2] ?? 0) * 3_600 +
          Number(m[3] ?? 0) * 60 +
          Number(m[4] ?? 0)) *
        1000;
      return ms > 0 ? { kind: "duration", ms } : null;
    },
  },
  // Written dates, optionally with a weekday and year: `Wednesday, Oct 29`, `March 3, 2026`.
  // Months must be capitalized, so "may 3 tests" stays prose.
  {
    regex:
      /\b(?:(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday|Mon|Tues?|Wed|Thu(?:rs?)?|Fri|Sat|Sun)\.?,?\s+)?(Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|June?|July?|Aug(?:ust)?|Sep(?:t(?:ember)?)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\.?\s+(\d{1,2})(?:st|nd|rd|th)?\b(?:,?\s+((?:19|20)\d{2})\b)?/g,
    build: (m) => {
      const month = MONTHS.indexOf((m[2] ?? "").slice(0, 3).toLowerCase()) + 1;
      const day = Number(m[3]);
      const year = m[4] ? Number(m[4]) : null;
      if (day < 1 || day > daysInMonth(year ?? 2024, month)) return null;
      const weekday = m[1] ? WEEKDAYS.indexOf(m[1].slice(0, 3).toLowerCase()) : null;
      return { kind: "date", year, month, day, weekday };
    },
  },
  // ISO calendar dates, but not inside file names or paths like `notes/2026-10-29.md`.
  {
    regex: /(?<![\w/.-])((?:19|20)\d{2})-(\d{2})-(\d{2})(?![\w/-]|\.\w)/g,
    build: (m) => {
      const [year, month, day] = [group(m, 1), group(m, 2), group(m, 3)];
      if (month < 1 || month > 12 || day < 1 || day > daysInMonth(year, month)) return null;
      return { kind: "date", year, month, day, weekday: null };
    },
  },
  // Words that are relative to when the text was written.
  {
    regex:
      /\b(?:(today|tonight|this (?:morning|afternoon|evening))|(yesterday|last night)|(tomorrow)|in (an?|\d+) (minutes?|mins?|hours?|hrs?|days?|weeks?)|(an?|\d+) (minutes?|mins?|hours?|hrs?|days?|weeks?) ago)\b/gi,
    build: (m) => {
      if (m[1]) return { kind: "relative-day", shift: 0 };
      if (m[2]) return { kind: "relative-day", shift: -1 };
      if (m[3]) return { kind: "relative-day", shift: 1 };
      const future = m[4] !== undefined;
      const count = (future ? m[4] : m[6]) ?? "";
      const unit = ((future ? m[5] : m[7]) ?? "m").toLowerCase();
      const amount = /^an?$/i.test(count) ? 1 : Number(count);
      const ms = amount * (UNIT_MS[unit.startsWith("mi") ? "m" : (unit[0] ?? "m")] ?? 60_000);
      return { kind: "relative-offset", ms: future ? ms : -ms };
    },
  },
  // A clock time with no zone, only after a preposition that makes it a time.
  {
    regex:
      /\b(?:at|by|until|before|after|around|from)\s+([01]?\d|2[0-3]):([0-5]\d)(?::([0-5]\d))?(?:\s?([AaPp]\.?[Mm]\.?))?(?![\w:]|\.\d)/g,
    lead: (m) => m[0].search(/\d/),
    build: (m) => {
      const hour = to24Hour(Number(m[1]), m[4]);
      if (hour === null) return null;
      return {
        kind: "unzoned",
        hour,
        minute: Number(m[2]),
        second: m[3] === undefined ? null : Number(m[3]),
        date: null,
      };
    },
  },
  // Raw seconds and milliseconds, only when a readable form is shorter.
  // `1990s` is a decade, not 33 minutes.
  {
    regex: /\b(\d+(?:\.\d+)?)\s?(ms|s|secs?|seconds)\b/g,
    build: (m) => {
      const value = Number(m[1]);
      const unit = m[2] ?? "s";
      if (unit === "s" && /^(?:19|20)\d0$/.test(m[1] ?? "")) return null;
      const ms = unit === "ms" ? value : value * 1000;
      if (unit === "ms" ? ms < 2_000 : ms < 120_000) return null;
      return { kind: "duration", ms };
    },
  },
];

/**
 * Text around `text` that is not part of it — for markdown, the rest of the
 * paragraph outside a bold or emphasized run. Roles read it, so "the
 * **08:53 UTC** cutoff" is still a deadline.
 */
export interface TimeMentionContext {
  before?: string;
  after?: string;
}

/** Every time mention in `text`, in reading order, with no two overlapping. */
export function detectTimeMentions(
  text: string,
  context: TimeMentionContext = {},
): TimeMentionMatch[] {
  const before = context.before ?? "";
  const surrounded = before + text + (context.after ?? "");
  const found: TimeMentionMatch[] = [];
  for (const pattern of PATTERNS) {
    pattern.regex.lastIndex = 0;
    let match: RegExpExecArray | null;
    while ((match = pattern.regex.exec(text))) {
      const lead = Math.max(0, pattern.lead?.(match) ?? 0);
      const start = match.index + lead;
      const end = match.index + match[0].length;
      if (found.some((other) => start < other.end && end > other.start)) continue;
      const data = pattern.build(match, text);
      if (!data) continue;
      const timed =
        data.kind === "instant" || data.kind === "clock" || data.kind === "relative-offset";
      const role: TimeMentionRole | null =
        data.kind === "relative-offset" && data.ms > 0
          ? "promise"
          : timed && !(data.kind === "instant" && data.source === "epoch")
            ? deadlineRole(surrounded, before.length + start, before.length + end)
            : null;
      found.push({ start, end, text: text.slice(start, end), data, role });
    }
  }
  return found.sort((a, b) => a.start - b.start);
}
