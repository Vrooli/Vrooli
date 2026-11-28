/** A Monday-first week, with the writes the desk can actually see on each day. */
export interface WeekDay {
  key: string;
  letter: string;
  date: number;
  at: Date;
  count: number;
  today: boolean;
  off: boolean;
}

export function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

export function weekDays(writes: Date[], now: Date = new Date()): WeekDay[] {
  const monday = new Date(now);
  const weekday = (now.getDay() + 6) % 7;
  monday.setHours(0, 0, 0, 0);
  monday.setDate(now.getDate() - weekday);
  const letters = ["M", "T", "W", "T", "F", "S", "S"];
  return letters.map((letter, index) => {
    const at = new Date(monday);
    at.setDate(monday.getDate() + index);
    return {
      key: at.toISOString(),
      letter,
      date: at.getDate(),
      at,
      count: writes.filter((date) => sameDay(date, at)).length,
      today: sameDay(at, now),
      off: index >= 5,
    };
  });
}

/** Seven bar heights for a card's mini row, one per day. */
export function miniWeek(writes: Date[], now: Date = new Date()): Array<{ count: number; today: boolean }> {
  return weekDays(writes, now).map((day) => ({ count: day.count, today: day.today }));
}
