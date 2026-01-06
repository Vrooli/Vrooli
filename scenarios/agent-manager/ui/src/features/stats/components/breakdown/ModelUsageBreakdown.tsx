// Model Usage Breakdown - horizontal bar chart showing model usage

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import { useModelBreakdown } from "../../hooks/useModelBreakdown";
import { formatNumber, formatCurrency, formatPercent, formatTokens } from "../../utils/formatters";
import { CHART_COLORS, TOOLTIP_STYLE, getSeriesColor } from "../../utils/chartConfig";

export function ModelUsageBreakdown() {
  const { data, isLoading, error } = useModelBreakdown();

  if (isLoading) {
    return (
      <div className="rounded-lg border border-border bg-card/50 p-6">
        <div className="mb-4 h-5 w-32 animate-pulse rounded bg-muted/30" />
        <div className="h-[250px] animate-pulse rounded bg-muted/20" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-6">
        <h3 className="text-sm font-semibold">Model Usage</h3>
        <p className="mt-2 text-sm text-red-500">Failed to load: {error.message}</p>
      </div>
    );
  }

  const models = data?.models ?? [];

  // Prepare data for chart - sort by run count descending
  const chartData = [...models]
    .sort((a, b) => b.runCount - a.runCount)
    .map((model) => ({
      name: model.model || "Unknown",
      runs: model.runCount,
      successRate: model.runCount > 0 ? model.successCount / model.runCount : 0,
      cost: model.totalCostUsd,
      tokens: model.totalTokens,
    }));

  const totalRuns = chartData.reduce((sum, d) => sum + d.runs, 0);

  return (
    <div className="rounded-lg border border-border bg-card/50 p-6">
      <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        Model Usage
      </h3>
      {chartData.length === 0 ? (
        <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
          No model data available
        </div>
      ) : (
        <div className="h-[250px]">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart
              data={chartData}
              layout="vertical"
              margin={{ top: 5, right: 30, left: 80, bottom: 5 }}
            >
              <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.grid} />
              <XAxis
                type="number"
                stroke={CHART_COLORS.axis}
                tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
                tickLine={{ stroke: CHART_COLORS.axis }}
              />
              <YAxis
                type="category"
                dataKey="name"
                stroke={CHART_COLORS.axis}
                tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
                tickLine={{ stroke: CHART_COLORS.axis }}
                width={75}
                tickFormatter={(value) =>
                  value.length > 12 ? `${value.slice(0, 12)}...` : value
                }
              />
              <Tooltip
                contentStyle={TOOLTIP_STYLE}
                formatter={(value: number, name: string, props) => {
                  const item = props.payload;
                  if (name === "runs") {
                    const pct = totalRuns > 0 ? (value / totalRuns) * 100 : 0;
                    return [`${formatNumber(value)} (${pct.toFixed(1)}%)`, "Runs"];
                  }
                  return [value, name];
                }}
                labelFormatter={(label) => <span className="font-medium">{label}</span>}
                content={({ active, payload }) => {
                  if (!active || !payload?.[0]) return null;
                  const item = payload[0].payload;
                  return (
                    <div className="rounded border border-border bg-card p-3 text-xs shadow-lg">
                      <div className="mb-2 font-medium">{item.name}</div>
                      <div className="space-y-1 text-muted-foreground">
                        <div className="flex justify-between gap-4">
                          <span>Runs:</span>
                          <span className="font-medium text-foreground">
                            {formatNumber(item.runs)}
                            {totalRuns > 0 && (
                              <span className="ml-1 text-muted-foreground">
                                ({((item.runs / totalRuns) * 100).toFixed(1)}%)
                              </span>
                            )}
                          </span>
                        </div>
                        <div className="flex justify-between gap-4">
                          <span>Success:</span>
                          <span
                            className={
                              item.successRate >= 0.9
                                ? "font-medium text-emerald-500"
                                : item.successRate >= 0.7
                                ? "font-medium text-amber-500"
                                : "font-medium text-red-500"
                            }
                          >
                            {formatPercent(item.successRate)}
                          </span>
                        </div>
                        <div className="flex justify-between gap-4">
                          <span>Cost:</span>
                          <span className="font-medium text-foreground">
                            {formatCurrency(item.cost)}
                          </span>
                        </div>
                        <div className="flex justify-between gap-4">
                          <span>Tokens:</span>
                          <span className="font-medium text-foreground">
                            {formatTokens(item.tokens)}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                }}
              />
              <Bar dataKey="runs" radius={[0, 4, 4, 0]}>
                {chartData.map((entry, index) => (
                  <Cell key={entry.name} fill={getSeriesColor(index)} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
