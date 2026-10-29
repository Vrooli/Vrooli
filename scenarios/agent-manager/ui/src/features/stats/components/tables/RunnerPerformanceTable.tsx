// Runner Performance Table - sortable table with runner metrics

import { useRunnerPerformance } from "../../hooks/useRunnerPerformance";
import {
  formatPercent,
  formatDuration,
  formatNumber,
  formatTokens,
} from "../../utils/formatters";
import { formatUsdFixed } from "../../../../lib/currency";
import { MeasureFrame } from "../measure/MeasureFrame";
import { compareSortValues, SortHeaderCell, useTableSort } from "./SortableHeader";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";

type SortField = "runnerType" | "runCount" | "successRate" | "totalCostUsd" | "totalTokens" | "avgDurationMs";

export function RunnerPerformanceTable() {
  const { data, isLoading, error } = useRunnerPerformance();
  const definitions = useMeasureDefinitions();
  const sort = useTableSort<SortField>("runCount");

  const runners = data?.runners ?? [];

  // Sort data
  const sortedRunners = [...runners].sort((a, b) => {
    let aVal: number | string;
    let bVal: number | string;

    switch (sort.field) {
      case "runnerType":
        aVal = a.runnerType;
        bVal = b.runnerType;
        break;
      case "successRate":
        aVal = a.runCount > 0 ? a.successCount / a.runCount : 0;
        bVal = b.runCount > 0 ? b.successCount / b.runCount : 0;
        break;
      default:
        aVal = a[sort.field] ?? 0;
        bVal = b[sort.field] ?? 0;
    }

    return compareSortValues(aVal, bVal, sort.direction);
  });

  return (
    <MeasureFrame label="Runner performance" result={data?.measure} definition={definitions.data?.find((item) => item.id === "throughput.runner_breakdown")} loading={isLoading} error={error?.message}>
    <div className="rounded-lg border border-border bg-card/50 p-4 sm:p-6 overflow-hidden">
      <h3 className="mb-2 sm:mb-4 text-sm font-semibold text-muted-foreground">
        Runner Performance
      </h3>
      {runners.length === 0 ? (
        <p className="text-sm text-muted-foreground">No runner data available</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[400px] text-sm">
            <thead>
              <tr className="border-b border-border text-left">
                <SortHeaderCell sort={sort} field="runnerType" label="Runner" />
                <SortHeaderCell sort={sort} field="runCount" label="Runs" />
                <SortHeaderCell sort={sort} field="successRate" label="Success" />
                <SortHeaderCell sort={sort} field="totalCostUsd" label="Cost" />
                <SortHeaderCell sort={sort} field="totalTokens" label="Tokens" />
                <SortHeaderCell sort={sort} field="avgDurationMs" label="Avg Time" last />
              </tr>
            </thead>
            <tbody>
              {sortedRunners.map((runner) => {
                const successRate =
                  runner.runCount > 0 ? runner.successCount / runner.runCount : 0;
                return (
                  <tr
                    key={runner.runnerType}
                    className="border-b border-border last:border-0"
                  >
                    <td className="py-2 pr-4 font-medium">{runner.runnerType}</td>
                    <td className="py-2 pr-4 tabular-nums">
                      {formatNumber(runner.runCount)}
                    </td>
                    <td className="py-2 pr-4 tabular-nums">
                      <span
                        className={
                          successRate >= 0.9
                            ? "text-emerald-500"
                            : successRate >= 0.7
                            ? "text-amber-500"
                            : "text-red-500"
                        }
                      >
                        {formatPercent(successRate)}
                      </span>
                    </td>
                    <td className="py-2 pr-4 tabular-nums">
                      {formatUsdFixed(runner.totalCostUsd, 2)}
                    </td>
                    <td className="py-2 pr-4 tabular-nums">{formatTokens(runner.totalTokens ?? 0)}</td>
                    <td className="py-2 tabular-nums">
                      {formatDuration(runner.avgDurationMs)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
    </MeasureFrame>
  );
}
