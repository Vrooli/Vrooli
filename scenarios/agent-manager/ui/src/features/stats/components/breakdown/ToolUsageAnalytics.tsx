// Tool Usage Analytics - horizontal bar chart showing tool call frequency

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
import { formatStatusLabel, formatUnknownLabel, statusBadgeVariant } from "../../../../lib/display";
import { formatStandardRelativeTime } from "../../../../lib/dateTime";
import { useToolCommandBreakdown, useToolUsage, useToolUsageModels, useToolUsageRuns } from "../../hooks/useToolUsage";
import { formatNumber, formatPercent } from "../../utils/formatters";
import { CHART_COLORS, TOOLTIP_STYLE, getSeriesColor } from "../../utils/chartConfig";
import { MeasureFrame } from "../measure/MeasureFrame";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import { runsLink } from "../../utils/navigation";

interface ToolChartDatum {
  name: string;
  calls: number;
  successRate: number;
  failedCount: number;
}

export function ToolUsageAnalytics() {
  const { data, isLoading, error } = useToolUsage({ limit: 10 });
  const [selectedTool, setSelectedTool] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"runs" | "models" | "commands">("runs");
  const definitions = useMeasureDefinitions();
  const { filter } = useTimeWindow();

  const tools = data?.tools;
  const { chartData, totalCalls } = useMemo(() => {
    const nextChartData: ToolChartDatum[] = (tools ?? []).map((tool) => ({
      name: tool.toolName || "unknown",
      calls: tool.callCount,
      successRate: tool.callCount > 0 ? tool.successCount / tool.callCount : 0,
      failedCount: tool.failedCount,
    }));

    return {
      chartData: nextChartData,
      totalCalls: nextChartData.reduce((sum, d) => sum + d.calls, 0),
    };
  }, [tools]);
  const selectedStats = useMemo(
    () => chartData.find((entry) => entry.name === selectedTool) ?? null,
    [chartData, selectedTool]
  );
  const {
    data: toolRuns,
    isLoading: runsLoading,
    error: runsError,
  } = useToolUsageRuns({
    toolName: selectedTool ?? undefined,
    enabled: !!selectedTool && activeTab === "runs",
    limit: 25,
  });
  const {
    data: toolModels,
    isLoading: modelsLoading,
    error: modelsError,
  } = useToolUsageModels({
    toolName: selectedTool ?? undefined,
    enabled: !!selectedTool && activeTab === "models",
    limit: 25,
  });
  const { data: commandData, isLoading: commandsLoading, error: commandsError } = useToolCommandBreakdown({ toolName: selectedTool ?? undefined, enabled: !!selectedTool && activeTab === "commands", limit: 25 });

  return (
    <MeasureFrame label="Tool usage" result={data?.measure} definition={definitions.data?.find((item) => item.id === "friction.tool_usage")} loading={isLoading} error={error?.message}>
    <div className="rounded-lg border border-border bg-card/50 p-4 sm:p-6">
      {selectedTool && selectedStats ? (
        <>
          <div className="mb-2 sm:mb-4 flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setSelectedTool(null)}
                className="h-8 w-8"
                aria-label="Back to tool usage chart"
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <div>
                <h3 className="text-sm font-semibold text-muted-foreground">
                  Tool Usage
                </h3>
                <p className="text-sm font-medium text-foreground">
                  {formatUnknownLabel(selectedStats.name)}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1">
                Calls <span className="ml-1 font-semibold text-foreground">{formatNumber(selectedStats.calls)}</span>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1">
                Success <span className="ml-1 font-semibold text-foreground">{formatPercent(selectedStats.successRate)}</span>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1">
                Failed <span className="ml-1 font-semibold text-foreground">{formatNumber(selectedStats.failedCount)}</span>
              </div>
            </div>
          </div>
          <div className="mb-3 flex items-center gap-2">
            <button
              type="button"
              onClick={() => setActiveTab("runs")}
              className={`rounded-full border px-3 py-1 text-xs font-semibold transition-colors ${
                activeTab === "runs"
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-border text-muted-foreground hover:text-foreground"
              }`}
            >
              Runs
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("models")}
              className={`rounded-full border px-3 py-1 text-xs font-semibold transition-colors ${
                activeTab === "models"
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-border text-muted-foreground hover:text-foreground"
              }`}
            >
              Models
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("commands")}
              className={`rounded-full border px-3 py-1 text-xs font-semibold transition-colors ${activeTab === "commands" ? "border-primary bg-primary/10 text-primary" : "border-border text-muted-foreground hover:text-foreground"}`}
            >
              Commands
            </button>
          </div>
          {activeTab === "runs" ? (
            runsLoading ? (
              <div className="h-[200px] sm:h-[250px] animate-pulse rounded bg-muted/20" />
            ) : runsError ? (
              <div className="rounded border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500">
                Failed to load runs: {runsError.message}
              </div>
            ) : (toolRuns?.runs?.length ?? 0) === 0 ? (
              <div className="flex h-[200px] sm:h-[250px] items-center justify-center text-sm text-muted-foreground">
                No runs found for this tool in the selected window
              </div>
            ) : (
              <div className="max-h-[200px] sm:max-h-[260px] overflow-y-auto pr-2 divide-y divide-border/60">
                {toolRuns?.runs.map((run) => (
                  <div key={run.runId} className="flex flex-wrap items-center justify-between gap-4 py-3">
                    <div>
                      <Link
                        to={`/runs/${run.runId}`}
                        className="text-sm font-medium text-foreground hover:underline"
                      >
                        {run.taskTitle || `Run ${run.runId.slice(0, 8)}`}
                      </Link>
                      <div className="text-xs text-muted-foreground">
                        {run.profileName || "Profile unavailable"} • {run.createdAt ? formatStandardRelativeTime(run.createdAt) : "Time unavailable"} • {run.runId.slice(0, 8)}
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      {run.status ? <Badge variant={statusBadgeVariant(run.status)}>{formatStatusLabel(run.status)}</Badge> : <span className="text-xs text-muted-foreground">Status unavailable</span>}
                      <div className="text-right text-xs text-muted-foreground">
                        <div>{run.callCount === undefined ? "Call count unavailable" : `${formatNumber(run.callCount)} calls`}</div>
                        <div className="text-muted-foreground">
                          {run.model || "Model unavailable"}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )
          ) : activeTab === "commands" ? commandsLoading ? (
            <div className="h-[200px] sm:h-[250px] animate-pulse rounded bg-muted/20" />
          ) : commandsError ? (
            <div className="rounded border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500">Failed to load commands: {commandsError.message}</div>
          ) : (commandData?.rows.length ?? 0) === 0 ? (
            <div className="flex h-[200px] sm:h-[250px] items-center justify-center text-sm text-muted-foreground">Command detail is not recorded for this window.</div>
          ) : (
            <div className="max-h-[200px] sm:max-h-[260px] overflow-y-auto pr-2 divide-y divide-border/60">
              {commandData?.rows.map((command) => (
                <Link key={`${command.executable}:${command.commandPath}`} to={runsLink({ ...filter, toolName: selectedTool ?? undefined })} className="flex flex-wrap items-center justify-between gap-4 py-3 hover:bg-muted/20">
                  <div className="min-w-0"><div className="text-sm font-medium">{command.executable || "Executable unavailable"}</div><div className="truncate text-xs text-muted-foreground" title={command.commandPath}>{command.commandPath || "Command path unavailable"}{command.truncated ? "…" : ""}</div></div>
                  <div className="text-right text-xs text-muted-foreground"><div>{formatNumber(command.callCount)} calls · {formatNumber(command.runCount)} runs</div><div>{formatPercent(command.callCount > 0 ? command.successCount / command.callCount : 0)} success</div></div>
                </Link>
              ))}
            </div>
          ) : modelsLoading ? (
            <div className="h-[200px] sm:h-[250px] animate-pulse rounded bg-muted/20" />
          ) : modelsError ? (
            <div className="rounded border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500">
              Failed to load models: {modelsError.message}
            </div>
          ) : (toolModels?.models?.length ?? 0) === 0 ? (
            <div className="flex h-[200px] sm:h-[250px] items-center justify-center text-sm text-muted-foreground">
              No model usage found for this tool in the selected window
            </div>
          ) : (
            <div className="max-h-[200px] sm:max-h-[260px] overflow-y-auto pr-2 divide-y divide-border/60">
              {toolModels?.models.map((model) => (
                <div key={model.model} className="flex flex-wrap items-center justify-between gap-4 py-3">
                  <div>
                    <div className="text-sm font-medium text-foreground">
                      {formatUnknownLabel(model.model)}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {formatNumber(model.runCount)} runs • {model.callCount === undefined ? "Call count unavailable" : `${formatNumber(model.callCount)} calls`}
                    </div>
                  </div>
                  <div className="text-right text-xs text-muted-foreground">
                    <div>{model.callCount === undefined ? "Success rate unavailable" : `${formatPercent(model.callCount > 0 ? model.successCount / model.callCount : 0)} success`}</div>
                    <div>{formatNumber(model.failedCount)} failed</div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      ) : (
        <>
          <div className="mb-2 sm:mb-4 flex items-baseline justify-between">
            <h3 className="text-sm font-semibold text-muted-foreground">
              Tool Usage
            </h3>
            <span className="text-xs text-muted-foreground">Click a bar to view runs</span>
          </div>
          {chartData.length === 0 ? (
            <div className="flex h-[200px] sm:h-[250px] items-center justify-center text-sm text-muted-foreground">
              No tool usage data available
            </div>
          ) : (
            <div className="h-[200px] sm:h-[250px]">
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
                    tickFormatter={(value: string) => {
                      const label = formatUnknownLabel(value);
                      return label.length > 12 ? `${label.slice(0, 12)}...` : label;
                    }}
                  />
                  <Tooltip
                    contentStyle={TOOLTIP_STYLE}
                    content={({ active, payload }) => {
                      if (!active || !payload?.[0]) return null;
                      const item = payload[0].payload as ToolChartDatum;
                      return (
                        <div className="rounded border border-border bg-card p-3 text-xs shadow-lg">
                          <div className="mb-2 font-medium">{formatUnknownLabel(item.name)}</div>
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
                  <Bar
                    dataKey="calls"
                    radius={[0, 4, 4, 0]}
                    onClick={(_, index) => {
                      if (typeof index === "number") {
                        setActiveTab("runs");
                        setSelectedTool(chartData[index]?.name ?? null);
                      }
                    }}
                    cursor="pointer"
                  >
                    {chartData.map((entry, index) => (
                      <Cell
                        key={entry.name}
                        fill={getSeriesColor(index)}
                        opacity={selectedTool && entry.name !== selectedTool ? 0.35 : 1}
                      />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          )}
          {chartData.length > 0 && <div className="mt-3 flex flex-wrap gap-2" aria-label="Tool run links">{chartData.map((item) => <Link key={item.name} to={runsLink({ ...filter, toolName: item.name })} className="rounded border border-border px-2 py-1 text-xs text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">View {formatUnknownLabel(item.name)} runs</Link>)}</div>}
        </>
      )}
    </div>
    </MeasureFrame>
  );
}
