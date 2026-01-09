// Cost and Duration Trends - dual-axis line chart

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { useRunTrends } from "../../hooks/useRunTrends";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import {
  formatChartAxisByPreset,
  formatCurrency,
  formatDuration,
  formatDateTime,
} from "../../utils/formatters";
import { CHART_COLORS, CHART_MARGINS, TOOLTIP_STYLE } from "../../utils/chartConfig";

export function CostDurationTrends() {
  const { data, isLoading, error } = useRunTrends();
  const { preset } = useTimeWindow();

  if (isLoading) {
    return (
      <div className="rounded-lg border border-border bg-card/50 p-6">
        <div className="mb-4 h-5 w-40 animate-pulse rounded bg-muted/30" />
        <div className="h-[300px] animate-pulse rounded bg-muted/20" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-6">
        <h3 className="text-sm font-semibold">Cost & Duration</h3>
        <p className="mt-2 text-sm text-red-500">Failed to load: {error.message}</p>
      </div>
    );
  }

  const buckets = data?.buckets ?? [];

  const chartData = buckets.map((bucket) => ({
    time: bucket.timestamp,
    cost: bucket.totalCostUsd,
    duration: bucket.avgDurationMs,
  }));

  // Find max values for dual axes
  const maxCost = Math.max(...chartData.map((d) => d.cost), 0);
  const maxDuration = Math.max(...chartData.map((d) => d.duration), 0);

  return (
    <div className="rounded-lg border border-border bg-card/50 p-6">
      <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        Cost & Duration Trends
      </h3>
      {chartData.length === 0 ? (
        <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
          No data available for this time period
        </div>
      ) : (
        <div className="h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={CHART_MARGINS}>
              <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.grid} />
              <XAxis
                dataKey="time"
                tickFormatter={(value) => formatChartAxisByPreset(value, preset)}
                stroke={CHART_COLORS.axis}
                tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
                tickLine={{ stroke: CHART_COLORS.axis }}
              />
              <YAxis
                yAxisId="cost"
                orientation="left"
                stroke={CHART_COLORS.info}
                tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
                tickLine={{ stroke: CHART_COLORS.info }}
                tickFormatter={(v) => `$${v.toFixed(2)}`}
                domain={[0, maxCost * 1.1 || 1]}
              />
              <YAxis
                yAxisId="duration"
                orientation="right"
                stroke={CHART_COLORS.warning}
                tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
                tickLine={{ stroke: CHART_COLORS.warning }}
                tickFormatter={(v) => formatDuration(v)}
                domain={[0, maxDuration * 1.1 || 1000]}
              />
              <Tooltip
                contentStyle={TOOLTIP_STYLE}
                labelFormatter={formatDateTime}
                formatter={(value: number, name: string) => {
                  if (name === "Cost") return [formatCurrency(value), name];
                  return [formatDuration(value), name];
                }}
              />
              <Legend
                wrapperStyle={{ fontSize: "12px", color: CHART_COLORS.text }}
              />
              <Line
                yAxisId="cost"
                type="monotone"
                dataKey="cost"
                stroke={CHART_COLORS.info}
                strokeWidth={2}
                dot={false}
                name="Cost"
              />
              <Line
                yAxisId="duration"
                type="monotone"
                dataKey="duration"
                stroke={CHART_COLORS.warning}
                strokeWidth={2}
                dot={false}
                name="Avg Duration"
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
