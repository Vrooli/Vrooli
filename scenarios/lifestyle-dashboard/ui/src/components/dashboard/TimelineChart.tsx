/**
 * TimelineChart displays a bar chart visualization of events over time.
 * Groups events by day and shows relative activity levels.
 *
 * [REQ:LD-QUERY-AGGREGATE] - Visual aggregation of event data
 * [REQ:LD-DASHBOARD-TIMELINE] - Timeline view across domains
 */
import type { TimelineEntry } from "../../lib/api";

interface TimelineChartProps {
  data: TimelineEntry[];
}

export function TimelineChart({ data }: TimelineChartProps) {
  // Group by day
  const byDay = data.reduce((acc, entry) => {
    if (!acc[entry.day]) acc[entry.day] = 0;
    acc[entry.day] += entry.count;
    return acc;
  }, {} as Record<string, number>);

  const days = Object.keys(byDay).sort();
  const maxCount = Math.max(...Object.values(byDay), 1);

  if (days.length === 0) {
    return (
      <div className="flex items-center justify-center h-32 text-slate-500 text-sm">
        No data to display yet
      </div>
    );
  }

  return (
    <div className="flex items-end gap-1 h-32">
      {days.map((day) => {
        const count = byDay[day];
        const height = (count / maxCount) * 100;
        return (
          <div key={day} className="flex-1 flex flex-col items-center gap-1">
            <div
              className="w-full bg-gradient-to-t from-violet-600 to-violet-400 rounded-t"
              style={{ height: `${Math.max(height, 4)}%` }}
              title={`${day}: ${count} events`}
            />
            <span className="text-[10px] text-slate-500 -rotate-45 origin-top-left whitespace-nowrap">
              {new Date(day).toLocaleDateString("en-US", { weekday: "short" })}
            </span>
          </div>
        );
      })}
    </div>
  );
}
