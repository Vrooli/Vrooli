// Model Usage Breakdown - horizontal bar chart showing model usage

import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
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
import { ChevronLeft } from "lucide-react";
import { Badge } from "../../../../components/ui/badge";
import { Button } from "../../../../components/ui/button";
import { formatRelativeTime } from "../../../../lib/utils";
import { useModelBreakdown, useModelUsageRuns } from "../../hooks/useModelBreakdown";
import { formatNumber, formatCurrency, formatPercent, formatTokens } from "../../utils/formatters";
import { CHART_COLORS, TOOLTIP_STYLE, getSeriesColor } from "../../utils/chartConfig";

export function ModelUsageBreakdown() {
  const { data, isLoading, error } = useModelBreakdown();
  const [selectedModel, setSelectedModel] = useState<string | null>(null);

  const models = data?.models ?? [];

  // Prepare data for chart - sort by run count descending
  const chartData = [...models]
    .sort((a, b) => b.runCount - a.runCount)
    .map((model) => ({
      name: model.model || "unknown",
      runs: model.runCount,
      successRate: model.runCount > 0 ? model.successCount / model.runCount : 0,
      cost: model.totalCostUsd,
      tokens: model.totalTokens,
    }));

  const totalRuns = chartData.reduce((sum, d) => sum + d.runs, 0);
  const selectedStats = useMemo(
    () => chartData.find((entry) => entry.name === selectedModel) ?? null,
    [chartData, selectedModel]
  );
  const {
    data: modelRuns,
    isLoading: runsLoading,
    error: runsError,
  } = useModelUsageRuns({
    model: selectedModel ?? undefined,
    enabled: !!selectedModel,
    limit: 25,
  });

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

  return (
    <div className="rounded-lg border border-border bg-card/50 p-6">
      {selectedModel && selectedStats ? (
        <>
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setSelectedModel(null)}
                className="h-8 w-8"
                aria-label="Back to model usage chart"
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <div>
                <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
                  Model Usage
                </h3>
                <p className="text-sm font-medium text-foreground">
                  {formatModelName(selectedStats.name)}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1">
                Runs <span className="ml-1 font-semibold text-foreground">{formatNumber(selectedStats.runs)}</span>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1">
                Success <span className="ml-1 font-semibold text-foreground">{formatPercent(selectedStats.successRate)}</span>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1">
                Cost <span className="ml-1 font-semibold text-foreground">{formatCurrency(selectedStats.cost)}</span>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1">
                Tokens <span className="ml-1 font-semibold text-foreground">{formatTokens(selectedStats.tokens)}</span>
              </div>
            </div>
          </div>
          {runsLoading ? (
            <div className="h-[250px] animate-pulse rounded bg-muted/20" />
          ) : runsError ? (
            <div className="rounded border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500">
              Failed to load runs: {runsError.message}
            </div>
          ) : (modelRuns?.runs?.length ?? 0) === 0 ? (
            <div className="flex h-[250px] items-center justify-center text-sm text-muted-foreground">
              No runs found for this model in the selected window
            </div>
          ) : (
            <div className="max-h-[260px] overflow-y-auto pr-2 divide-y divide-border/60">
              {modelRuns?.runs.map((run) => (
                <div key={run.runId} className="flex flex-wrap items-center justify-between gap-4 py-3">
                  <div>
                    <Link
                      to={`/runs/${run.runId}`}
                      className="text-sm font-medium text-foreground hover:underline"
                    >
                      {run.taskTitle || "Untitled Task"}
                    </Link>
                    <div className="text-xs text-muted-foreground">
                      {run.profileName} • {formatRelativeTime(run.createdAt)} • {run.runId.slice(0, 8)}
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <Badge variant={statusVariant(run.status)}>{statusLabel(run.status)}</Badge>
                    <div className="text-right text-xs text-muted-foreground">
                      <div>{formatCurrency(run.totalCostUsd)}</div>
                      <div>{formatTokens(run.totalTokens)}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      ) : (
        <>
          <div className="mb-4 flex items-baseline justify-between">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              Model Usage
            </h3>
            <span className="text-xs text-muted-foreground">Click a bar to view runs</span>
          </div>
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
                    tickFormatter={(value) => {
                      const label = formatModelName(value);
                      return label.length > 12 ? `${label.slice(0, 12)}...` : label;
                    }}
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
                    labelFormatter={(label) => (
                      <span className="font-medium">{formatModelName(label)}</span>
                    )}
                    content={({ active, payload }) => {
                      if (!active || !payload?.[0]) return null;
                      const item = payload[0].payload;
                      return (
                        <div className="rounded border border-border bg-card p-3 text-xs shadow-lg">
                          <div className="mb-2 font-medium">{formatModelName(item.name)}</div>
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
                  <Bar
                    dataKey="runs"
                    radius={[0, 4, 4, 0]}
                    onClick={(_, index) => {
                      if (typeof index === "number") {
                        setSelectedModel(chartData[index]?.name ?? null);
                      }
                    }}
                    cursor="pointer"
                  >
                    {chartData.map((entry, index) => (
                      <Cell
                        key={entry.name}
                        fill={getSeriesColor(index)}
                        opacity={selectedModel && entry.name !== selectedModel ? 0.35 : 1}
                      />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function statusVariant(status: string) {
  switch (status) {
    case "pending":
    case "starting":
    case "running":
    case "needs_review":
    case "complete":
    case "failed":
    case "cancelled":
      return status;
    default:
      return "secondary";
  }
}

function statusLabel(status: string) {
  return status
    .split("_")
    .map((word) => (word ? word[0].toUpperCase() + word.slice(1) : word))
    .join(" ");
}

function formatModelName(value: string) {
  if (!value || value === "unknown") return "Unknown";
  return value;
}
