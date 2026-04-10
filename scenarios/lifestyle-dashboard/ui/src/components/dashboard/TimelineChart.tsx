/**
 * TimelineChart displays a bar chart visualization of events over time.
 * Groups events by day and shows relative activity levels.
 * Supports selectable periods (7d/30d/90d) per OT-P0-004.
 *
 * [REQ:LD-UI-TRENDS] - Trend charts with 7d/30d/90d selectable periods
 * [REQ:LD-QUERY-AGGREGATE] - Visual aggregation of event data
 * [REQ:LD-DASHBOARD-TIMELINE] - Timeline view across domains
 */
import type { TimelineEntry } from "../../lib/api";
import { DATA_SELECTORS } from "../../consts/selectors";

export type TrendPeriod = 7 | 30 | 90;

interface TimelineChartProps {
  data: TimelineEntry[];
  /** Currently selected period in days */
  period?: TrendPeriod;
  /** Callback when period changes */
  onPeriodChange?: (period: TrendPeriod) => void;
  /** Whether to show period selector */
  showPeriodSelector?: boolean;
}

const PERIOD_OPTIONS: { value: TrendPeriod; label: string }[] = [
  { value: 7, label: "7d" },
  { value: 30, label: "30d" },
  { value: 90, label: "90d" },
];

export function TimelineChart({
  data,
  period = 7,
  onPeriodChange,
  showPeriodSelector = false
}: TimelineChartProps) {
  // Group by day
  const byDay = data.reduce<Record<string, number>>((acc, entry) => {
    acc[entry.day] = (acc[entry.day] ?? 0) + entry.count;
    return acc;
  }, {});

  const days = Object.keys(byDay).sort();
  const maxCount = Math.max(...Object.values(byDay), 1);

  // Calculate total for summary
  const totalEvents = Object.values(byDay).reduce((sum, count) => sum + count, 0);

  if (days.length === 0) {
    return (
      <div className="flex flex-col gap-2">
        {showPeriodSelector && onPeriodChange && (
          <PeriodSelector period={period} onPeriodChange={onPeriodChange} />
        )}
        <div
          className="flex items-center justify-center h-32 text-slate-500 text-sm"
          data-testid={DATA_SELECTORS.TIMELINE_EMPTY}
        >
          No data to display yet
        </div>
      </div>
    );
  }

  // For longer periods, show fewer labels to avoid crowding
  const labelInterval = period > 30 ? 7 : period > 7 ? 3 : 1;

  return (
    <div className="flex flex-col gap-2" data-testid={DATA_SELECTORS.TIMELINE_CHART}>
      {/* Period selector and summary */}
      {showPeriodSelector && onPeriodChange && (
        <div className="flex items-center justify-between">
          <PeriodSelector period={period} onPeriodChange={onPeriodChange} />
          <span className="text-xs text-slate-400">
            {totalEvents} events in {period}d
          </span>
        </div>
      )}

      {/* Chart */}
      <div className="flex items-end gap-px h-32" data-testid={DATA_SELECTORS.TIMELINE_BARS}>
        {days.map((day, index) => {
          const count = byDay[day] ?? 0;
          const height = (count / maxCount) * 100;
          const showLabel = index % labelInterval === 0 || index === days.length - 1;

          return (
            <div key={day} className="flex-1 flex flex-col items-center gap-1 min-w-0">
              <div
                className="w-full bg-gradient-to-t from-violet-600 to-violet-400 rounded-t transition-all duration-300 hover:from-violet-500 hover:to-violet-300"
                style={{ height: `${Math.max(height, 4)}%` }}
                title={`${day}: ${count} events`}
                data-testid={`timeline-bar-${day}`}
              />
              {showLabel && (
                <span className="text-[10px] text-slate-500 truncate max-w-full">
                  {formatDayLabel(day, period)}
                </span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

/**
 * Period selector component for switching between 7d/30d/90d views
 * [REQ:LD-UI-TRENDS] - Trend period selection
 */
function PeriodSelector({
  period,
  onPeriodChange
}: {
  period: TrendPeriod;
  onPeriodChange: (period: TrendPeriod) => void;
}) {
  return (
    <div
      className="inline-flex rounded-lg bg-slate-800 p-0.5"
      data-testid={DATA_SELECTORS.TIMELINE_PERIOD_SELECTOR}
    >
      {PERIOD_OPTIONS.map((option) => (
        <button
          key={option.value}
          onClick={() => onPeriodChange(option.value)}
          className={`
            px-3 py-1 text-xs font-medium rounded-md transition-all
            ${period === option.value
              ? "bg-violet-600 text-white"
              : "text-slate-400 hover:text-white"
            }
          `}
          data-testid={`period-${option.value}d`}
          aria-pressed={period === option.value}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

/**
 * Format day label based on period length
 * - 7d: Show weekday (Mon, Tue, etc.)
 * - 30d: Show day number (1, 15, etc.)
 * - 90d: Show month abbreviation (Jan, Feb, etc.)
 */
function formatDayLabel(day: string, period: TrendPeriod): string {
  const date = new Date(day);
  if (period <= 7) {
    return date.toLocaleDateString("en-US", { weekday: "short" });
  } else if (period <= 30) {
    return date.getDate().toString();
  } else {
    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  }
}
