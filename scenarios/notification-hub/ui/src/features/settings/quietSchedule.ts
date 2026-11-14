/**
 * Quiet hours as the operator thinks of them, over the storage the Hub has.
 *
 * The Hub stores one window per weekday. People think in schedules: "10 PM to
 * 7 AM on weekdays". This module groups stored windows that share the same
 * hours, zone and critical setting into one schedule, and turns an edited
 * schedule back into the per-weekday writes the Hub needs.
 */

/** Weekdays as the Hub numbers them: 0 is Sunday. */
export type Weekday = 0 | 1 | 2 | 3 | 4 | 5 | 6;

/** A weekday as the string selectors take. */
export type DayKey = "0" | "1" | "2" | "3" | "4" | "5" | "6";
export const dayKey = (day: Weekday): DayKey => String(day) as DayKey;

/** The order a week is shown in. */
export const WEEK_ORDER: readonly Weekday[] = [1, 2, 3, 4, 5, 6, 0];
export const WEEKDAYS: readonly Weekday[] = [1, 2, 3, 4, 5];
export const WEEKEND: readonly Weekday[] = [6, 0];

/** A window as the Hub returns it. */
export interface StoredWindow {
  id: string;
  weekday: number;
  start: string;
  end: string;
  timezone: string;
  criticalOverride: boolean;
}

/** What every day of a schedule shares. */
export interface ScheduleSettings {
  start: string;
  end: string;
  timezone: string;
  criticalOverride: boolean;
}

export interface Schedule extends ScheduleSettings {
  /** In week order. */
  days: Weekday[];
}

export type ScheduleIssue = "noDays" | "badTime" | "sameTime" | "badZone";

const TIME = /^([01]\d|2[0-3]):[0-5]\d$/;

export function isWeekday(value: number): value is Weekday {
  return Number.isInteger(value) && value >= 0 && value <= 6;
}

export function sortDays(days: readonly Weekday[]): Weekday[] {
  return WEEK_ORDER.filter((day) => days.includes(day));
}

/** Group stored windows that share their hours, zone and critical setting. */
export function groupSchedules(windows: readonly StoredWindow[]): Schedule[] {
  const groups = new Map<string, Schedule>();
  for (const window of windows) {
    if (!isWeekday(window.weekday)) continue;
    const key = [window.start, window.end, window.timezone, window.criticalOverride].join("|");
    const group = groups.get(key);
    if (group) group.days.push(window.weekday);
    else groups.set(key, { days: [window.weekday], start: window.start, end: window.end, timezone: window.timezone, criticalOverride: window.criticalOverride });
  }
  return [...groups.values()]
    .map((schedule) => ({ ...schedule, days: sortDays(schedule.days) }))
    .sort((a, b) => WEEK_ORDER.indexOf(a.days[0] ?? 0) - WEEK_ORDER.indexOf(b.days[0] ?? 0));
}

function knownZone(timezone: string): boolean {
  try {
    new Intl.DateTimeFormat("en", { timeZone: timezone }).format();
    return true;
  } catch {
    return false;
  }
}

/**
 * Hours that end earlier than they start run overnight: from the start time on
 * each chosen day until the end time the next morning, as the Hub reads them.
 */
export function isOvernight(schedule: Pick<Schedule, "start" | "end">): boolean {
  return TIME.test(schedule.start) && TIME.test(schedule.end) && schedule.end < schedule.start;
}

/** Why a schedule cannot be saved, if it cannot. Equal times are the Hub's one refusal of the hours themselves. */
export function scheduleIssue(schedule: Schedule): ScheduleIssue | null {
  if (schedule.days.length === 0) return "noDays";
  if (!TIME.test(schedule.start) || !TIME.test(schedule.end)) return "badTime";
  if (schedule.start === schedule.end) return "sameTime";
  if (!knownZone(schedule.timezone)) return "badZone";
  return null;
}

export function sameSettings(a: ScheduleSettings, b: ScheduleSettings): boolean {
  return a.start === b.start && a.end === b.end && a.timezone === b.timezone && a.criticalOverride === b.criticalOverride;
}

export type WindowChange =
  | { kind: "set"; weekday: Weekday; settings: ScheduleSettings }
  | { kind: "delete"; weekday: Weekday; id: string };

/**
 * The writes that turn what the Hub holds into `target`. `previousDays` are
 * the days the schedule had before this edit; days dropped from it are deleted.
 * Planning against the current stored windows makes a retry after a partial
 * save redo only what is missing.
 */
export function planChanges(stored: readonly StoredWindow[], target: Schedule, previousDays: readonly Weekday[]): WindowChange[] {
  const byDay = new Map(stored.map((window) => [window.weekday, window]));
  const changes: WindowChange[] = [];
  for (const weekday of target.days) {
    const current = byDay.get(weekday);
    if (!current || !sameSettings(current, target)) {
      changes.push({ kind: "set", weekday, settings: { start: target.start, end: target.end, timezone: target.timezone, criticalOverride: target.criticalOverride } });
    }
  }
  for (const weekday of previousDays) {
    const current = byDay.get(weekday);
    if (!target.days.includes(weekday) && current) changes.push({ kind: "delete", weekday, id: current.id });
  }
  return changes;
}

/** Whether the Hub's latest windows show this change as applied. */
export function isApplied(change: WindowChange, latest: readonly StoredWindow[]): boolean {
  const current = latest.find((window) => window.weekday === change.weekday);
  if (change.kind === "delete") return current === undefined;
  return current !== undefined && sameSettings(current, change.settings);
}

export interface SaveOutcome {
  /** Days the latest read confirms. */
  applied: Weekday[];
  /** Days still to do: the first that did not apply, and everything after it. */
  pending: Weekday[];
  /** The follow-up read failed, so nothing beyond the writes' own answers is known. */
  readFailed: boolean;
}

/**
 * Apply changes one weekday at a time and trust only what the Hub reads back.
 * A write that fails stops the run, because later writes would likely fail the
 * same way; the read-back then says what really landed.
 */
export async function applyChanges(
  changes: readonly WindowChange[],
  io: { write: (change: WindowChange) => Promise<void>; readLatest: () => Promise<StoredWindow[]> },
): Promise<SaveOutcome> {
  let attempted = 0;
  for (const change of changes) {
    attempted += 1;
    try {
      await io.write(change);
    } catch {
      break;
    }
  }
  let latest: StoredWindow[];
  try {
    latest = await io.readLatest();
  } catch {
    return { applied: [], pending: changes.map((change) => change.weekday), readFailed: true };
  }
  const applied: Weekday[] = [];
  const pending: Weekday[] = [];
  changes.forEach((change, index) => {
    if (index < attempted && isApplied(change, latest)) applied.push(change.weekday);
    else pending.push(change.weekday);
  });
  return { applied, pending, readFailed: false };
}
