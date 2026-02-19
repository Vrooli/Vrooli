import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Radio } from "lucide-react";
import { fetchProbeHistory, type ProbeResult } from "../lib/api";
import { ProbeResults } from "../components/ProbeResults";
import { StatusBadge, statusToVariant } from "../components/ui/status-badge";
import { Pagination } from "../components/ui/pagination";
import { SortHeader, useSort, MOBILE_PAGE_SIZE, DESKTOP_PAGE_SIZE } from "../components/ui/sort-header";
import { RefreshButton } from "../components/ui/refresh-button";
import { QueryState } from "../components/ui/query-state";

type ProbeSortField = "subdomain" | "probe_type" | "status" | "latency_ms";

const compareProbes = (a: ProbeResult, b: ProbeResult, field: ProbeSortField): number => {
  switch (field) {
    case "subdomain": return a.subdomain.localeCompare(b.subdomain);
    case "probe_type": return a.probe_type.localeCompare(b.probe_type);
    case "status": return a.status.localeCompare(b.status);
    case "latency_ms": return a.latency_ms - b.latency_ms;
  }
};

function ProbeHistory() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["probe-history"],
    queryFn: fetchProbeHistory,
    refetchInterval: 30000,
  });

  const [mobilePage, setMobilePage] = useState(0);
  const [desktopPage, setDesktopPage] = useState(0);
  const { sorted, sortField, sortDir, toggleSort } = useSort(data, "subdomain" as ProbeSortField, compareProbes);

  const mobileSlice = sorted.slice(mobilePage * MOBILE_PAGE_SIZE, (mobilePage + 1) * MOBILE_PAGE_SIZE);
  const desktopSlice = sorted.slice(desktopPage * DESKTOP_PAGE_SIZE, (desktopPage + 1) * DESKTOP_PAGE_SIZE);

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="probes-history-table">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Radio className="h-5 w-5 text-slate-300" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Probe History</h2>
        </div>
        <RefreshButton onClick={() => refetch()} disabled={isFetching} aria-label="Refresh probe history" />
      </div>

      <QueryState
        isLoading={isLoading}
        error={error}
        refetch={refetch}
        loadingLabel="Loading probe history..."
        errorLabel="Failed to load probe history."
        skeleton={
          <div className="space-y-2">
            {[1, 2, 3].map((i) => <div key={i} className="h-10 rounded bg-white/5" />)}
          </div>
        }
      >
        {data && data.length === 0 && (
          <div className="mt-4 rounded-lg border border-dashed border-white/10 p-4 text-center">
            <p className="text-sm text-slate-300">No probe history recorded yet.</p>
            <p className="mt-1 text-xs text-slate-500">Run probes above to generate results. History accumulates over time.</p>
          </div>
        )}

        {/* Mobile: card layout */}
        {sorted.length > 0 && (
          <div className="mt-4 sm:hidden">
            <div className="flex flex-col gap-2" role="list" aria-label="Probe history cards">
              {mobileSlice.map((r: ProbeResult, i: number) => (
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
            <Pagination page={mobilePage} total={sorted.length} pageSize={MOBILE_PAGE_SIZE} onPageChange={setMobilePage} className="mt-3" />
          </div>
        )}

        {/* Desktop: table layout */}
        {sorted.length > 0 && (
          <div className="mt-4 hidden sm:block">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <caption className="sr-only">Probe history results</caption>
                <thead>
                  <tr className="border-b border-white/10 text-left text-slate-300">
                    <th scope="col" className="pb-2 pr-4"><SortHeader<ProbeSortField> field="subdomain" label="Subdomain" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2 pr-4"><SortHeader<ProbeSortField> field="probe_type" label="Type" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2 pr-4"><SortHeader<ProbeSortField> field="status" label="Status" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2 pr-4"><SortHeader<ProbeSortField> field="latency_ms" label="Latency" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2">Error</th>
                  </tr>
                </thead>
                <tbody>
                  {desktopSlice.map((r: ProbeResult, i: number) => (
                    <tr key={i} className="border-b border-white/5">
                      <td className="py-2 pr-4 font-medium text-slate-200">{r.subdomain}</td>
                      <td className="py-2 pr-4 text-slate-300">{r.probe_type}</td>
                      <td className="py-2 pr-4">
                        <StatusBadge variant={statusToVariant(r.status)} label={r.status} />
                      </td>
                      <td className="py-2 pr-4 font-mono text-slate-300">
                        {r.latency_ms > 0 ? `${r.latency_ms}ms` : "—"}
                      </td>
                      <td className="py-2 text-slate-300 break-words max-w-[300px]">
                        {r.error_msg || "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination page={desktopPage} total={sorted.length} pageSize={DESKTOP_PAGE_SIZE} onPageChange={setDesktopPage} className="mt-3" />
          </div>
        )}
      </QueryState>
    </div>
  );
}

export default function ProbesPage() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div>
        <h1 className="text-lg sm:text-xl font-semibold">Probe Dashboard</h1>
        <p className="text-sm text-slate-300">Check whether configured routes respond on their health endpoints.</p>
      </div>
      <ProbeResults />
      <ProbeHistory />
    </div>
  );
}
