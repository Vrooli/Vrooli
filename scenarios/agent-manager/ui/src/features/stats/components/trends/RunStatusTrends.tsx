// Run Status Trends - stacked area chart showing complete/failed/cancelled over time

import { useMemo } from "react";
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
import { formatNumber } from "../../utils/formatters";
import { formatChartAxisByPreset, formatStatsDateTime } from "../../../../lib/dateTime";
import { CHART_COLORS, CHART_MARGINS, TOOLTIP_STYLE } from "../../utils/chartConfig";
import { MeasureFrame } from "../measure/MeasureFrame";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";

export function RunStatusTrends() {
  const { data, isLoading, error } = useRunTrends();
  const { preset } = useTimeWindow();
  const definitions = useMeasureDefinitions();
  const buckets = data?.buckets;
  const chartData = useMemo(
    () =>
      (buckets ?? []).map((bucket) => ({
        time: bucket.timestamp,
        completed: bucket.runsCompleted,
        failed: bucket.runsFailed,
        started: bucket.runsStarted,
      })),
    [buckets]
  );

  return (
    <MeasureFrame label="Run trends" result={data?.measure} definition={definitions.data?.find((item) => item.id === "throughput.terminal_run_trend")} loading={isLoading} error={error?.message}>
    <div className="rounded-lg border border-border bg-card/50 p-4 sm:p-6 min-w-0">
      <h3 className="mb-2 sm:mb-4 text-sm font-semibold text-muted-foreground">
        Run Trends
      </h3>
      {chartData.length === 0 ? (
        <div className="flex h-[200px] sm:h-[300px] items-center justify-center text-sm text-muted-foreground">
          No data available for this time period
        </div>
      ) : (
        <div className="h-[200px] sm:h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ ...CHART_MARGINS, bottom: 5, left: 5 }}>
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
                tickFormatter={(value: string) => formatChartAxisByPreset(value, preset)}
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
                labelFormatter={(label) => formatStatsDateTime(String(label ?? ""))}
                formatter={(value, name) => {
                  const numericValue = typeof value === "number" ? value : Number(value ?? 0);
                  const label = String(name ?? "");
                  return [formatNumber(numericValue), label.charAt(0).toUpperCase() + label.slice(1)];
                }}
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
    </MeasureFrame>
  );
}
