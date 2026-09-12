// Cost Distribution - per-run token and cost percentiles plus a token histogram

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
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
import { fetchDurableRunCost, statsQueryKeys, type CostTokenBucket, type DurableRunCost } from "../../api/statsClient";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";
import { formatNumber, formatTokens } from "../../utils/formatters";
import { formatUsdFixed } from "../../../../lib/currency";
import { CHART_COLORS, CHART_MARGINS, TOOLTIP_STYLE } from "../../utils/chartConfig";
import { MeasureFrame } from "../measure/MeasureFrame";

const PERCENTILE_ROWS = [
  { key: "p50", label: "P50" },
  { key: "p90", label: "P90" },
  { key: "p95", label: "P95" },
  { key: "p99", label: "P99" },
  { key: "max", label: "Max" },
] as const;

export function CostDistributionCard() {
  const { filter } = useTimeWindow();
  const definitions = useMeasureDefinitions();
  const { data, isLoading, error } = useQuery({
    queryKey: [...statsQueryKeys.cost(filter), "distribution"],
    queryFn: () => fetchDurableRunCost(filter),
  });

  const buckets: CostTokenBucket[] = useMemo(() => data?.tokenBuckets ?? [], [data?.tokenBuckets]);
  const sampleSize = data?.distributionSampleSize ?? 0;
  const unobserved = data?.distributionUnobservedRuns ?? 0;

  return (
    <MeasureFrame
      label="Cost distribution"
      result={data}
      definition={definitions.data?.find((item) => item.id === "throughput.run_cost")}
      loading={isLoading}
      error={error?.message}
    >
      <div className="rounded-lg border border-border bg-card/50 p-4 sm:p-6">
        <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
          <h3 className="text-sm font-semibold text-muted-foreground">
            Per-run token &amp; cost distribution
          </h3>
          {data ? (
            <p className="text-xs text-muted-foreground">
              {formatNumber(sampleSize)} runs with observed usage
              {unobserved > 0 ? ` · ${formatNumber(unobserved)} uncounted` : ""}
            </p>
          ) : null}
        </div>

        {!data || sampleSize === 0 ? (
          <div className="flex h-[200px] items-center justify-center text-sm text-muted-foreground">
            No runs with observed token usage in the selected window
          </div>
        ) : (
          <div className="space-y-6">
            <div className="grid gap-4 sm:grid-cols-2">
              <PercentileTable
                title="Tokens per run"
                rows={PERCENTILE_ROWS.map((row) => ({
                  label: row.label,
                  value: formatTokens(tokenFor(data, row.key)),
                }))}
              />
              <PercentileTable
                title="Cost per run"
                rows={PERCENTILE_ROWS.map((row) => ({
                  label: row.label,
                  value: formatUsdFixed(costFor(data, row.key), 4),
                }))}
              />
            </div>

            <div>
              <p className="mb-2 text-xs font-medium text-muted-foreground">
                Run count by token volume
              </p>
              <div className="h-[200px] w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={buckets} margin={CHART_MARGINS}>
                    <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.grid} vertical={false} />
                    <XAxis dataKey="label" stroke={CHART_COLORS.axis} fontSize={11} tickLine={false} />
                    <YAxis stroke={CHART_COLORS.axis} fontSize={11} tickLine={false} allowDecimals={false} />
                    <Tooltip
                      contentStyle={TOOLTIP_STYLE}
                      formatter={(value) => [formatNumber(Number(value) || 0), "Runs"]}
                    />
                    <Bar dataKey="runCount" radius={[4, 4, 0, 0]}>
                      {buckets.map((bucket, index) => (
                        <Cell key={bucket.label} fill={CHART_COLORS.series[index % CHART_COLORS.series.length]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>
        )}
      </div>
    </MeasureFrame>
  );
}

function PercentileTable({
  title,
  rows,
}: {
  title: string;
  rows: Array<{ label: string; value: string }>;
}) {
  return (
    <div className="rounded-md border border-border/60 bg-muted/20 p-3">
      <p className="mb-2 text-xs font-medium text-muted-foreground">{title}</p>
      <dl className="space-y-1">
        {rows.map((row) => (
          <div key={row.label} className="flex items-center justify-between text-sm">
            <dt className="text-muted-foreground">{row.label}</dt>
            <dd className="font-medium tabular-nums text-foreground">{row.value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

type CostResponse = DurableRunCost;

function tokenFor(data: CostResponse, key: (typeof PERCENTILE_ROWS)[number]["key"]): number {
  switch (key) {
    case "p50":
      return data.p50Tokens;
    case "p90":
      return data.p90Tokens;
    case "p95":
      return data.p95Tokens;
    case "p99":
      return data.p99Tokens;
    default:
      return data.maxTokens;
  }
}

function costFor(data: CostResponse, key: (typeof PERCENTILE_ROWS)[number]["key"]): number {
  switch (key) {
    case "p50":
      return data.p50CostUsd;
    case "p90":
      return data.p90CostUsd;
    case "p95":
      return data.p95CostUsd;
    case "p99":
      return data.p99CostUsd;
    default:
      return data.maxCostUsd;
  }
}
