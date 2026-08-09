import { useState } from "react";
import {
  type TokenAttributionGroupBy,
  type TokenAttributionView,
} from "../../api/statsClient";
import { useTokenAttribution } from "../../hooks/useTokenAttribution";
import { formatNumber, formatPercent, formatTokens } from "../../utils/formatters";

const groupOptions: Array<{ value: TokenAttributionGroupBy; label: string }> = [
  { value: "capability", label: "Capability" },
  { value: "executable", label: "Executable" },
  { value: "command_path", label: "Command path" },
  { value: "target_scenario_operation", label: "Scenario operation" },
];

const viewOptions: Array<{ value: TokenAttributionView; label: string }> = [
  { value: "footprint", label: "Footprint" },
  { value: "residency", label: "Residency" },
  { value: "incurred", label: "Incurred" },
];

const viewDescriptions: Record<TokenAttributionView, string> = {
  footprint: "Intrinsic payload added by each invocation; use it to find commands worth shrinking.",
  residency: "Payload multiplied by the turns it remains in context; compaction attenuation is an approximation.",
  incurred: "Provider-reported usage associated with the invocation; use it as the accounting view.",
};

export function TokenAttributionBreakdown() {
  const [groupBy, setGroupBy] = useState<TokenAttributionGroupBy>("capability");
  const [view, setView] = useState<TokenAttributionView>("footprint");
  const query = useTokenAttribution({ groupBy, view });
  const rows = query.data?.rows ?? [];

  return (
    <section className="rounded-lg border border-border bg-card/50 p-4 sm:p-6" data-testid="token-attribution-breakdown">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-muted-foreground">Token attribution</h3>
          <p className="mt-1 max-w-2xl text-xs text-muted-foreground">{viewDescriptions[view]}</p>
        </div>
        <div className="flex flex-wrap gap-2 text-xs">
          <label className="flex items-center gap-2 text-muted-foreground">
            Group by
            <select aria-label="Token attribution group by" className="rounded border border-border bg-background px-2 py-1 text-foreground" value={groupBy} onChange={(event) => setGroupBy(event.target.value as TokenAttributionGroupBy)}>
              {groupOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
          </label>
          <label className="flex items-center gap-2 text-muted-foreground">
            View
            <select aria-label="Token attribution view" className="rounded border border-border bg-background px-2 py-1 text-foreground" value={view} onChange={(event) => setView(event.target.value as TokenAttributionView)}>
              {viewOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
          </label>
        </div>
      </div>

      {query.isLoading ? <div className="mt-4 h-32 animate-pulse rounded bg-muted/20" aria-label="Loading token attribution" /> : null}
      {query.error ? <div className="mt-4 rounded border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500">Token attribution: {query.error.message}</div> : null}
      {!query.isLoading && !query.error && rows.length === 0 ? <div className="mt-4 flex h-32 items-center justify-center rounded border border-dashed border-border text-sm text-muted-foreground">No token attribution data available for the selected window.</div> : null}
      {!query.isLoading && !query.error && rows.length > 0 ? (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-xs">
            <thead className="border-b border-border text-muted-foreground">
              <tr>
                <th className="px-2 py-2 font-medium">{groupOptions.find((option) => option.value === groupBy)?.label}</th>
                <th className="px-2 py-2 text-right font-medium">Tokens</th>
                <th className="px-2 py-2 text-right font-medium">Estimated share</th>
                <th className="px-2 py-2 text-right font-medium">Calls</th>
                {view === "footprint" ? <><th className="px-2 py-2 text-right font-medium">P50 footprint</th><th className="px-2 py-2 text-right font-medium">P95 footprint</th><th className="px-2 py-2 text-right font-medium">Max footprint</th></> : null}
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {rows.map((row) => (
                <tr key={`${row.groupBy}:${row.value}`}>
                  <td className="px-2 py-2 font-medium text-foreground">{row.value || "unknown"}</td>
                  <td className="px-2 py-2 text-right text-foreground">{formatTokens(row.totalTokens)}</td>
                  <td className="px-2 py-2 text-right text-foreground">{formatPercent(row.estimatedTokenShare)}</td>
                  <td className="px-2 py-2 text-right text-muted-foreground">{formatNumber(row.callCount)}</td>
                  {view === "footprint" ? <><td className="px-2 py-2 text-right text-muted-foreground">{formatTokens(row.p50FootprintTokens)}</td><td className="px-2 py-2 text-right text-muted-foreground">{formatTokens(row.p95FootprintTokens)}</td><td className="px-2 py-2 text-right text-muted-foreground">{formatTokens(row.maxFootprintTokens)}</td></> : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
