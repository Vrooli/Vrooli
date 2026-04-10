import { useMutation } from "@tanstack/react-query";
import { Zap, RefreshCw, Radio } from "lucide-react";
import { runProbes } from "../lib/api";
import { Button } from "./ui/button";
import { StatusBadge, statusToVariant } from "./ui/status-badge";

export function ProbeResults() {
  const { mutate, data, isPending } = useMutation({
    mutationFn: runProbes,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="liveness-probes-panel">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Zap className="h-5 w-5 text-slate-300" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Liveness Probes</h2>
        </div>
        <Button variant="outline" size="sm" onClick={() => mutate()} disabled={isPending} data-testid="probes-run-button">
          <RefreshCw className={`h-4 w-4 mr-2 ${isPending ? "animate-spin" : ""}`} />
          {isPending ? "Probing..." : "Run Probes"}
        </Button>
      </div>

      {!data && !isPending && (
        <div className="mt-4 rounded-lg border border-dashed border-white/10 p-6 text-center" data-testid="probes-empty-state">
          <Radio className="mx-auto h-8 w-8 text-slate-600" />
          <p className="mt-2 text-sm font-medium text-slate-300">No probe results yet</p>
          <p className="mt-1 text-xs text-slate-500">Run probes to check the liveness of all configured routes via their health endpoints.</p>
        </div>
      )}

      {isPending && (
        <div className="mt-4 space-y-2 animate-pulse" role="status">
          <span className="sr-only">Running probes...</span>
          {[1, 2].map((i) => <div key={i} className="h-10 rounded bg-white/5" />)}
        </div>
      )}

      {data && (
        <>
          <div className="mt-4 flex gap-4 text-sm">
            <span className="text-green-400" data-testid="probes-summary-up">{data.summary.up} up</span>
            <span className="text-red-400" data-testid="probes-summary-down">{data.summary.down} down</span>
            <span className="text-slate-300">{data.summary.total} total</span>
          </div>

          {/* Mobile: card layout */}
          <div className="mt-3 flex flex-col gap-2 sm:hidden" role="list" aria-label="Probe result cards">
            {data.results.map((r, i) => (
              <div key={i} className="rounded-lg bg-black/20 p-3" role="listitem">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-slate-200 text-sm">{r.subdomain}</span>
                  <StatusBadge variant={statusToVariant(r.status)} label={r.status} />
                </div>
                <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-300">
                  <span>{r.probe_type}</span>
                  {r.latency_ms > 0 && <span className="font-mono">{r.latency_ms}ms</span>}
                  {r.error_msg && <span className="text-red-400 break-all">{r.error_msg}</span>}
                </div>
              </div>
            ))}
          </div>

          {/* Desktop: table layout */}
          <div className="mt-3 hidden sm:block overflow-x-auto" data-testid="probes-results-table">
            <table className="w-full text-sm">
              <caption className="sr-only">Liveness probe results</caption>
              <thead>
                <tr className="border-b border-white/10 text-left text-slate-300">
                  <th scope="col" className="pb-2 pr-4">Subdomain</th>
                  <th scope="col" className="pb-2 pr-4">Type</th>
                  <th scope="col" className="pb-2 pr-4">Status</th>
                  <th scope="col" className="pb-2 pr-4">Latency</th>
                  <th scope="col" className="pb-2">Error</th>
                </tr>
              </thead>
              <tbody>
                {data.results.map((r, i) => (
                  <tr key={i} className="border-b border-white/5">
                    <td className="py-2 pr-4 font-medium text-slate-200">{r.subdomain}</td>
                    <td className="py-2 pr-4 text-slate-300">{r.probe_type}</td>
                    <td className="py-2 pr-4">
                      <StatusBadge variant={statusToVariant(r.status)} label={r.status} />
                    </td>
                    <td className="py-2 pr-4 font-mono text-slate-300">
                      {r.latency_ms > 0 ? `${r.latency_ms}ms` : "—"}
                    </td>
                    <td className="py-2 text-slate-300 truncate max-w-[200px]" title={r.error_msg || undefined}>
                      {r.error_msg || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
