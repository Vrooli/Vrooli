import type { TFunction } from "i18next";
import { strings } from "../../consts/strings";
import { formatClock, formatList, weekdayName } from "./format";
import { WEEK_ORDER, WEEKDAYS, WEEKEND, sortDays, type Schedule, type Weekday } from "./quietSchedule";

function sameDays(a: readonly Weekday[], b: readonly Weekday[]): boolean {
  return a.length === b.length && sortDays(a).every((day, index) => day === sortDays(b)[index]);
}

/** "Weekdays", "Every day", "Mon–Wed and Fri": the days as a person would say them. */
export function describeDays(days: readonly Weekday[], lang: string, t: TFunction): string {
  if (days.length === 7) return t(strings.settings.days.every);
  if (sameDays(days, WEEKDAYS)) return t(strings.settings.days.weekdays);
  if (sameDays(days, WEEKEND)) return t(strings.settings.days.weekends);
  const ordered = sortDays(days);
  const parts: string[] = [];
  for (let index = 0; index < ordered.length;) {
    let end = index;
    while (end + 1 < ordered.length && WEEK_ORDER.indexOf(ordered[end + 1] ?? 0) === WEEK_ORDER.indexOf(ordered[end] ?? 0) + 1) end += 1;
    const first = ordered[index] ?? 0;
    const last = ordered[end] ?? 0;
    if (end - index >= 2) parts.push(`${weekdayName(first, lang, "short")}–${weekdayName(last, lang, "short")}`);
    else for (let day = index; day <= end; day += 1) parts.push(weekdayName(ordered[day] ?? 0, lang, "short"));
    index = end + 1;
  }
  return formatList(parts, lang);
}

/** "10:00 PM–7:00 AM" */
export function describeHours(schedule: Pick<Schedule, "start" | "end">, lang: string): string {
  return `${formatClock(schedule.start, lang)}–${formatClock(schedule.end, lang)}`;
}
