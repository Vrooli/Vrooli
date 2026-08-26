/**
 * @libraryId react-component-library:Calendar
 * @displayName Calendar
 * @description A localized, keyboard-operable calendar for single, multiple, range, week, and month selection.
 * @version 1.0.3
 * @tags ["forms","calendar","date","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource forms.calendar */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { useMemo, useState, type CSSProperties, type RefObject } from "react";
import { useDirection } from "../../../../hooks/useDirection/versions/1.0.0/useDirection";
import { useLocale } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";
import { useRovingFocus } from "../../../../hooks/useRovingFocus/versions/1.0.0/useRovingFocus";

export type CalendarMode = "single" | "multiple" | "range" | "week" | "month";
export type CalendarRange = { start: Date; end?: Date };
export type CalendarValue = Date | Date[] | CalendarRange | null;

export interface CalendarProps {
  month?: Date;
  value?: CalendarValue;
  defaultValue?: CalendarValue;
  mode?: CalendarMode;
  minDate?: Date;
  maxDate?: Date;
  disabledDate?: (date: Date) => boolean;
  firstDayOfWeek?: 0 | 1 | 2 | 3 | 4 | 5 | 6;
  label?: string;
  onChange?: (value: CalendarValue) => void;
  onMonthChange?: (month: Date) => void;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-calendar] { display: grid; gap: var(--space-md, 1rem); inline-size: 100%; min-inline-size: 0; box-sizing: border-box; padding: var(--space-md, 1rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 1rem); background: var(--color-surface-raised, #fff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 8px 24px rgb(15 23 42 / .08)); }
[data-rcl-calendar-header] { display: flex; align-items: center; justify-content: space-between; gap: var(--space-sm, .75rem); }
[data-rcl-calendar-heading] { display: grid; gap: var(--space-3xs, .25rem); min-inline-size: 0; }
[data-rcl-calendar-month] { font: var(--text-subtitle, 650 1rem/1.35 system-ui, sans-serif); }
[data-rcl-calendar-mode] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1rem system-ui, sans-serif); }
[data-rcl-calendar-nav] { display: flex; gap: var(--space-2xs, .5rem); }
[data-rcl-calendar-nav] button { display: grid; place-items: center; inline-size: var(--tap-target-min, 44px); block-size: var(--tap-target-min, 44px); border: var(--border-hairline, 1px) solid var(--color-border-strong, #94a3b8); border-radius: var(--radius-control, .625rem); background: var(--color-surface, #fff); color: var(--color-foreground, #0f172a); font: var(--text-title, 700 1rem/1 system-ui, sans-serif); cursor: pointer; }
[data-rcl-calendar-nav] button:hover { border-color: var(--color-primary, #2563eb); color: var(--color-primary, #2563eb); }
[data-rcl-calendar-nav] button:focus-visible, [data-rcl-calendar-day]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus, #2563eb) 34%, transparent); outline-offset: 2px; }
[data-rcl-calendar-weekdays], [data-rcl-calendar-grid] { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: var(--space-3xs, .25rem); }
[data-rcl-calendar-weekday] { padding-block: var(--space-2xs, .5rem); color: var(--color-muted-foreground, #64748b); text-align: center; font: var(--text-overline, 700 .6875rem/1rem system-ui, sans-serif); letter-spacing: .06em; text-transform: uppercase; }
[data-rcl-calendar-day] { position: relative; display: grid; place-items: center; min-inline-size: 0; min-block-size: var(--tap-target-min, 44px); border: var(--border-hairline, 1px) solid transparent; border-radius: var(--radius-control, .625rem); background: transparent; color: var(--color-foreground, #0f172a); font: var(--text-label, 650 .8125rem/1.25rem system-ui, sans-serif); cursor: pointer; }
[data-rcl-calendar-day][data-outside="true"] { color: var(--color-muted-foreground, #64748b); opacity: .5; }
[data-rcl-calendar-day][data-today="true"] { border-color: color-mix(in srgb, var(--color-primary, #2563eb) 46%, var(--color-border, #cbd5e1)); }
[data-rcl-calendar-day][data-selected="true"] { border-color: var(--color-primary, #2563eb); background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #fff); }
[data-rcl-calendar-day][data-in-range="true"] { border-radius: 0; background: color-mix(in srgb, var(--color-primary, #2563eb) 13%, var(--color-surface, #fff)); color: var(--color-foreground, #0f172a); }
[data-rcl-calendar-day][data-range-edge="true"] { border-radius: var(--radius-control, .625rem); background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #fff); }
[data-rcl-calendar-day]:disabled { cursor: not-allowed; opacity: .38; text-decoration: line-through; }
[data-rcl-calendar-footer] { display: flex; align-items: flex-start; justify-content: space-between; flex-wrap: wrap; gap: var(--space-xs, .625rem); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1rem system-ui, sans-serif); }
[data-rcl-calendar-help] { max-inline-size: 38ch; }
@media (max-width: 34rem) { [data-rcl-calendar] { padding: var(--space-sm, .75rem); } [data-rcl-calendar-weekdays], [data-rcl-calendar-grid] { gap: var(--space-3xs, .2rem); } [data-rcl-calendar-day] { min-block-size: 40px; } }
@media (forced-colors: active) { [data-rcl-calendar], [data-rcl-calendar-nav] button, [data-rcl-calendar-day] { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } [data-rcl-calendar-day][data-selected="true"], [data-rcl-calendar-day][data-range-edge="true"] { background: Highlight; color: HighlightText; } }
`;

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}
function dateKey(date: Date) {
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
}
function sameDay(a: Date | undefined, b: Date | undefined) {
  return Boolean(a && b && dateKey(a) === dateKey(b));
}
function addDays(date: Date, amount: number) {
  const next = new Date(date);
  next.setDate(next.getDate() + amount);
  return next;
}
function monthStart(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}
function monthEnd(date: Date) {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}
function asDate(value: CalendarValue): Date | undefined {
  return value instanceof Date ? value : Array.isArray(value) ? value[0] : value?.start;
}
function valueDates(value: CalendarValue): Date[] {
  return value instanceof Date
    ? [value]
    : Array.isArray(value)
      ? value
      : value
        ? [value.start, ...(value.end ? [value.end] : [])]
        : [];
}
function isBetween(date: Date, start?: Date, end?: Date) {
  return Boolean(start && end && date >= startOfDay(start) && date <= startOfDay(end));
}

export function Calendar({
  month: controlledMonth,
  value,
  defaultValue = null,
  mode = "single",
  minDate,
  maxDate,
  disabledDate,
  firstDayOfWeek = 0,
  label = translate("forms.calendar.label.1", "Calendar"),
  onChange,
  onMonthChange,
  className,
  style,
}: CalendarProps) {
  const locale = useLocale();
  const direction = useDirection();
  const initialMonth = asDate(value ?? defaultValue) ?? new Date();
  const [internalMonth, setInternalMonth] = useState(monthStart(initialMonth));
  const [internalValue, setInternalValue] = useState<CalendarValue>(defaultValue);
  const [activeIndex, setActiveIndex] = useState(0);
  const currentMonth = monthStart(controlledMonth ?? internalMonth);
  const selectedValue = value === undefined ? internalValue : value;
  const formatter = useMemo(
    () => new Intl.DateTimeFormat(locale, { month: "long", year: "numeric" }),
    [locale],
  );
  const dayFormatter = useMemo(
    () => new Intl.DateTimeFormat(locale, { weekday: "short" }),
    [locale],
  );
  const fullFormatter = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: "full" }),
    [locale],
  );
  const days = useMemo(() => {
    const first = monthStart(currentMonth);
    const offset = (first.getDay() - firstDayOfWeek + 7) % 7;
    return Array.from({ length: 42 }, (_, index) => addDays(first, index - offset));
  }, [currentMonth, firstDayOfWeek]);
  const refs = useMemo<RefObject<HTMLButtonElement>[]>(
    () => days.map(() => ({ current: null })),
    [days],
  );
  const onDayKeyDown = useRovingFocus(refs, activeIndex, setActiveIndex, {
    orientation: "both",
    loop: true,
  });
  const selectedDates = valueDates(selectedValue);
  const selectedDate = asDate(selectedValue);
  const range =
    !Array.isArray(selectedValue) && selectedValue && !(selectedValue instanceof Date)
      ? selectedValue
      : undefined;
  const disabled = (date: Date) =>
    Boolean(
      (minDate && date < startOfDay(minDate)) ||
        (maxDate && date > startOfDay(maxDate)) ||
        disabledDate?.(date),
    );
  const updateMonth = (offset: number) => {
    const next = new Date(currentMonth.getFullYear(), currentMonth.getMonth() + offset, 1);
    if (!controlledMonth) setInternalMonth(next);
    onMonthChange?.(next);
  };
  const selectDay = (day: Date) => {
    if (disabled(day)) return;
    const clean = startOfDay(day);
    let next: CalendarValue = clean;
    if (mode === "multiple")
      next = selectedDates.some((item) => sameDay(item, clean))
        ? selectedDates.filter((item) => !sameDay(item, clean))
        : [...selectedDates, clean];
    if (mode === "range")
      next =
        range?.start && !range.end
          ? clean < range.start
            ? { start: clean, end: range.start }
            : { start: range.start, end: clean }
          : { start: clean };
    if (mode === "week") {
      const start = addDays(clean, -((clean.getDay() - firstDayOfWeek + 7) % 7));
      next = { start, end: addDays(start, 6) };
    }
    if (mode === "month") next = { start: monthStart(clean), end: monthEnd(clean) };
    if (mode === "single") next = clean;
    if (value === undefined) setInternalValue(next);
    onChange?.(next);
  };
  const weekdayLabels = Array.from({ length: 7 }, (_, index) =>
    dayFormatter.format(new Date(2026, 7, 1 + firstDayOfWeek + index)),
  );
  return (
    <section
      data-rcl-calendar
      className={className}
      style={style}
      aria-label={label}
      dir={direction}
    >
      <style data-rcl-calendar-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <header data-rcl-calendar-header>
        <div data-rcl-calendar-heading>
          <span data-rcl-calendar-month>{formatter.format(currentMonth)}</span>
          <span data-rcl-calendar-mode>
            {mode === "single" ? "Choose a date" : `Choose ${mode}`}
          </span>
        </div>
        <nav
          data-rcl-calendar-nav
          aria-label={translate("forms.calendar.aria-label.2", "Calendar navigation")}
        >
          <button
            data-testid="forms.calendar"
            type="button"
            aria-label={translate("forms.calendar.aria-label.3", "Previous month")}
            onClick={() => updateMonth(-1)}
          >
            ‹
          </button>
          <button
            data-testid="forms.calendar"
            type="button"
            aria-label={translate("forms.calendar.aria-label.4", "Next month")}
            onClick={() => updateMonth(1)}
          >
            ›
          </button>
        </nav>
      </header>
      <div data-rcl-calendar-weekdays role="row">
        {weekdayLabels.map((day, index) => (
          <span key={`${day}-${index}`} data-rcl-calendar-weekday role="columnheader">
            {day}
          </span>
        ))}
      </div>
      <div data-rcl-calendar-grid role="grid" aria-label={formatter.format(currentMonth)}>
        {days.map((day, index) => {
          const outside = day.getMonth() !== currentMonth.getMonth();
          const isSelected =
            mode === "multiple"
              ? selectedDates.some((item) => sameDay(item, day))
              : mode === "month"
                ? day.getMonth() === asDate(selectedValue)?.getMonth() &&
                  day.getFullYear() === asDate(selectedValue)?.getFullYear()
                : sameDay(asDate(selectedValue), day);
          const inRange =
            isBetween(day, range?.start, range?.end) ||
            (mode === "week" && isBetween(day, range?.start, range?.end));
          const edge = Boolean(
            (range?.start && sameDay(range.start, day)) || (range?.end && sameDay(range.end, day)),
          );
          return (
            <button
              data-testid="forms.calendar"
              key={dateKey(day)}
              ref={refs[index]}
              type="button"
              role="gridcell"
              aria-label={fullFormatter.format(day)}
              aria-selected={isSelected || inRange}
              data-rcl-calendar-day
              data-outside={outside || undefined}
              data-today={sameDay(day, new Date()) || undefined}
              data-selected={isSelected || undefined}
              data-in-range={inRange || undefined}
              data-range-edge={edge || undefined}
              disabled={disabled(day)}
              tabIndex={index === activeIndex ? 0 : -1}
              onFocus={() => setActiveIndex(index)}
              onKeyDown={onDayKeyDown}
              onClick={() => selectDay(day)}
            >
              {day.getDate()}
            </button>
          );
        })}
      </div>
      <footer data-rcl-calendar-footer>
        <span data-rcl-calendar-help>
          {selectedDates.length
            ? `${selectedDates.length} date${selectedDates.length === 1 ? "" : "s"} selected`
            : "No date selected"}
        </span>
        <span aria-live="polite">{selectedDate ? fullFormatter.format(selectedDate) : ""}</span>
      </footer>
    </section>
  );
}
