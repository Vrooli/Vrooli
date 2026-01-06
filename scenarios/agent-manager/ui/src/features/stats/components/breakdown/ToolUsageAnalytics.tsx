// Tool Usage Analytics - horizontal bar chart showing tool call frequency

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
import { useToolUsage } from "../../hooks/useToolUsage";
import { formatNumber, formatPercent } from "../../utils/formatters";
import { CHART_COLORS, TOOLTIP_STYLE, getSeriesColor } from "../../utils/chartConfig";

export function ToolUsageAnalytics() {
  const { data, isLoading, error } = useToolUsage({ limit: 10 });

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
        <h3 className="text-sm font-semibold">Tool Usage</h3>
        <p className="mt-2 text-sm text-red-500">Failed to load: {error.message}</p>
      </div>
    );
  }

  const tools = data?.tools ?? [];

  // Prepare data for chart - already sorted by API
  const chartData = tools.map((tool) => ({
    name: tool.toolName || "Unknown",
    calls: tool.callCount,
    successRate: tool.callCount > 0 ? tool.successCount / tool.callCount : 0,
    failedCount: tool.failedCount,
  }));

  const totalCalls = chartData.reduce((sum, d) => sum + d.calls, 0);

  return (
    <div className="rounded-lg border border-border bg-card/50 p-6">
      <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        Tool Usage
      </h3>
      {chartData.length === 0 ? (
        <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
          No tool usage data available
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
                content={({ active, payload }) => {
                  if (!active || !payload?.[0]) return null;
                  const item = payload[0].payload;
                  return (
                    <div className="rounded border border-border bg-card p-3 text-xs shadow-lg">
                      <div className="mb-2 font-medium">{item.name}</div>
                      <div className="space-y-1 text-muted-foreground">
                        <div className="flex justify-between gap-4">
                          <span>Calls:</span>
                          <span className="font-medium text-foreground">
                            {formatNumber(item.calls)}
                            {totalCalls > 0 && (
                              <span className="ml-1 text-muted-foreground">
                                ({((item.calls / totalCalls) * 100).toFixed(1)}%)
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
                          <span>Failed:</span>
                          <span className="font-medium text-red-500">
                            {formatNumber(item.failedCount)}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                }}
              />
              <Bar dataKey="calls" radius={[0, 4, 4, 0]}>
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
