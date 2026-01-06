// Run Status Trends - stacked area chart showing complete/failed/cancelled over time

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { useRunTrends } from "../../hooks/useRunTrends";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import { formatChartAxisByPreset, formatDateTime, formatNumber } from "../../utils/formatters";
import { CHART_COLORS, CHART_MARGINS, TOOLTIP_STYLE } from "../../utils/chartConfig";

export function RunStatusTrends() {
  const { data, isLoading, error } = useRunTrends();
  const { preset } = useTimeWindow();

  if (isLoading) {
    return (
      <div className="rounded-lg border border-border bg-card/50 p-6">
        <div className="mb-4 h-5 w-32 animate-pulse rounded bg-muted/30" />
        <div className="h-[300px] animate-pulse rounded bg-muted/20" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-6">
        <h3 className="text-sm font-semibold">Run Trends</h3>
        <p className="mt-2 text-sm text-red-500">Failed to load: {error.message}</p>
      </div>
    );
  }

  const buckets = data?.buckets ?? [];

  // Transform data for stacked area
  const chartData = buckets.map((bucket) => ({
    time: bucket.timestamp,
    completed: bucket.runsCompleted,
    failed: bucket.runsFailed,
    started: bucket.runsStarted,
  }));

  return (
    <div className="rounded-lg border border-border bg-card/50 p-6">
      <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        Run Trends
      </h3>
      {chartData.length === 0 ? (
        <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
          No data available for this time period
        </div>
      ) : (
        <div className="h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={CHART_MARGINS}>
              <defs>
                <linearGradient id="gradientCompleted" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={CHART_COLORS.complete} stopOpacity={0.8} />
                  <stop offset="95%" stopColor={CHART_COLORS.complete} stopOpacity={0.1} />
                </linearGradient>
                <linearGradient id="gradientFailed" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={CHART_COLORS.failed} stopOpacity={0.8} />
                  <stop offset="95%" stopColor={CHART_COLORS.failed} stopOpacity={0.1} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.grid} />
              <XAxis
                dataKey="time"
                tickFormatter={(value) => formatChartAxisByPreset(value, preset)}
                stroke={CHART_COLORS.axis}
                tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
                tickLine={{ stroke: CHART_COLORS.axis }}
              />
              <YAxis
                stroke={CHART_COLORS.axis}
                tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
                tickLine={{ stroke: CHART_COLORS.axis }}
                allowDecimals={false}
              />
              <Tooltip
                contentStyle={TOOLTIP_STYLE}
                labelFormatter={formatDateTime}
                formatter={(value: number, name: string) => [
                  formatNumber(value),
                  name.charAt(0).toUpperCase() + name.slice(1),
                ]}
              />
              <Legend
                wrapperStyle={{ fontSize: "12px", color: CHART_COLORS.text }}
              />
              <Area
                type="monotone"
                dataKey="completed"
                stackId="1"
                stroke={CHART_COLORS.complete}
                fill="url(#gradientCompleted)"
                name="Completed"
              />
              <Area
                type="monotone"
                dataKey="failed"
                stackId="1"
                stroke={CHART_COLORS.failed}
                fill="url(#gradientFailed)"
                name="Failed"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
