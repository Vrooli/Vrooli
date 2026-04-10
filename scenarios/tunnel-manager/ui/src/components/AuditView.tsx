import { useQuery } from "@tanstack/react-query";
import { Shield, CheckCircle2 } from "lucide-react";
import { fetchAudit } from "../lib/api";
import { StatusBadge, statusToVariant } from "./ui/status-badge";
import { RefreshButton } from "./ui/refresh-button";
import { QueryState } from "./ui/query-state";

export function AuditView() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["audit"],
    queryFn: fetchAudit,
    refetchInterval: 60000,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="audit-panel">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-slate-300" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Port Compliance Audit</h2>
        </div>
        <RefreshButton onClick={() => refetch()} disabled={isFetching} aria-label="Refresh audit" data-testid="audit-refresh" />
      </div>

      <QueryState
        isLoading={isLoading}
        error={error}
        refetch={refetch}
        loadingLabel="Running audit..."
        errorLabel="Failed to run audit"
        skeleton={
          <div className="space-y-2">
            <div className="h-5 w-40 rounded bg-white/5" />
            <div className="h-24 rounded bg-white/5" />
          </div>
        }
      >
        {data && (
          <>
            <div className="mt-4 flex gap-4 text-sm">
              <span className="text-green-400" data-testid="audit-summary-compliant">{data.compliant} compliant</span>
              {data.violations > 0 && (
                <span className="text-red-400" data-testid="audit-summary-violations">{data.violations} violation(s)</span>
              )}
              <span className="text-slate-300">{data.total} total</span>
            </div>

            {data.results.length === 0 && data.total === 0 && (
              <div className="mt-3 rounded-lg border border-dashed border-white/10 p-4 text-center">
                <CheckCircle2 className="mx-auto h-6 w-6 text-slate-600" />
                <p className="mt-1 text-xs text-slate-500">No routes to audit. Add routes to see compliance results.</p>
              </div>
            )}

            {/* Mobile: card layout */}
            {data.results.length > 0 && (
              <div className="mt-3 flex flex-col gap-2 sm:hidden" role="list" aria-label="Audit result cards">
                {data.results.map((r, i) => (
                  <div key={i} className="rounded-lg bg-black/20 p-3" role="listitem">
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-slate-200 text-sm">{r.subdomain}</span>
                      <StatusBadge variant={statusToVariant(r.status)} label={r.status} />
                    </div>
                    <div className="mt-1 text-xs text-slate-300">
                      <span>{r.scenario_name}</span>
                      <span className="mx-2">|</span>
                      <span className="font-mono">expected: {r.expected_port}</span>
                      {r.actual_port && <span className="font-mono"> / actual: {r.actual_port}</span>}
                    </div>
                    {r.detail && <p className="mt-1 text-xs text-slate-500 break-words">{r.detail}</p>}
                  </div>
                ))}
              </div>
            )}

            {/* Desktop: table layout */}
            {data.results.length > 0 && (
              <div className="mt-3 hidden sm:block overflow-x-auto" data-testid="audit-results-table">
                <table className="w-full text-sm">
                  <caption className="sr-only">Port compliance audit results</caption>
                  <thead>
                    <tr className="border-b border-white/10 text-left text-slate-300">
                      <th scope="col" className="pb-2 pr-4">Subdomain</th>
                      <th scope="col" className="pb-2 pr-4">Scenario</th>
                      <th scope="col" className="pb-2 pr-4">Expected</th>
                      <th scope="col" className="pb-2 pr-4">Actual</th>
                      <th scope="col" className="pb-2 pr-4">Status</th>
                      <th scope="col" className="pb-2">Detail</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.results.map((r, i) => (
                      <tr key={i} className="border-b border-white/5">
                        <td className="py-2 pr-4 font-medium text-slate-200">{r.subdomain}</td>
                        <td className="py-2 pr-4 text-slate-300">{r.scenario_name}</td>
                        <td className="py-2 pr-4 font-mono text-slate-300">{r.expected_port}</td>
                        <td className="py-2 pr-4 font-mono text-slate-300">{r.actual_port || "—"}</td>
                        <td className="py-2 pr-4">
                          <StatusBadge variant={statusToVariant(r.status)} label={r.status} />
                        </td>
                        <td className="py-2 text-slate-300 truncate max-w-[200px]" title={r.detail || undefined}>{r.detail || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )}
      </QueryState>
    </div>
  );
}
