import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { fetchDurableWorkloadBreakdown, statsQueryKeys } from "../../api/statsClient";
import { formatNumber, formatPercent, formatTokens } from "../../utils/formatters";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";
import { MeasureFrame } from "../measure/MeasureFrame";
import { runsLink } from "../../utils/navigation";

export function RecurringWorkloadPanel() {
  const { filter } = useTimeWindow();
  const definitions = useMeasureDefinitions();
  const query = useQuery({ queryKey: statsQueryKeys.workload(filter, 20), queryFn: () => fetchDurableWorkloadBreakdown(filter) });
  return (
    <MeasureFrame label="Recurring workloads" result={query.data} definition={definitions.data?.find((item) => item.id === "throughput.workload_breakdown")} loading={query.isLoading} error={query.error?.message}>
      <section className="rounded-lg border border-border bg-card/50 p-4 sm:p-6" data-testid="recurring-workload-panel">
        <div className="mb-3 flex flex-wrap items-baseline justify-between gap-2">
          <div><h3 className="text-sm font-semibold text-muted-foreground">Recurring workloads</h3><p className="text-xs text-muted-foreground">Consumption per successful completion; observational, not causal.</p></div>
        </div>
        {query.data?.rows.length === 0 ? <p className="text-sm text-muted-foreground">No workload keys recorded in this window.</p> : <div className="overflow-x-auto"><table className="w-full min-w-[560px] text-sm"><thead><tr className="border-b border-border text-left text-xs text-muted-foreground"><th className="pb-2 pr-3">Workload</th><th className="pb-2 pr-3">Runs</th><th className="pb-2 pr-3">Completion</th><th className="pb-2 pr-3">Tokens</th><th className="pb-2">Tokens / success</th></tr></thead><tbody>{query.data?.rows.filter((row) => row.key).map((row) => <tr key={row.key} className="border-b border-border last:border-0"><td className="py-2 pr-3"><Link className="font-medium hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" to={runsLink({ ...filter, workloadKey: row.key })}>{row.value || row.key}</Link></td><td className="py-2 pr-3 tabular-nums">{formatNumber(row.runCount)}</td><td className="py-2 pr-3 tabular-nums">{formatPercent(row.completionRate)}</td><td className="py-2 pr-3 tabular-nums">{formatTokens(row.totalTokens)}</td><td className="py-2 tabular-nums">{row.successCount > 0 ? formatTokens(row.consumptionPerSuccessfulCompletion) : "No successful completion"}</td></tr>)}</tbody></table></div>}
      </section>
    </MeasureFrame>
  );
}
