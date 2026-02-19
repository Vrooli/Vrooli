import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BarChart3, Activity, TrendingUp } from "lucide-react";
import { fetchMetricsHistory, fetchMetricsLatest, type MetricsRecord } from "../lib/api";
import { timeAgo } from "../lib/utils";
import { Tooltip } from "../components/ui/tooltip";
import { Pagination } from "../components/ui/pagination";
import { SortHeader, useSort, MOBILE_PAGE_SIZE, DESKTOP_PAGE_SIZE } from "../components/ui/sort-header";
import { RefreshButton } from "../components/ui/refresh-button";
import { QueryState } from "../components/ui/query-state";
import { EmptyState } from "../components/ui/empty-state";

type MetricsSortField = "scraped_at" | "ha_connections" | "active_streams" | "smoothed_rtt_ms" | "request_errors";

function isMetricsRecord(v: MetricsRecord | { status: string }): v is MetricsRecord {
  return "id" in v;
}

function LatestMetrics() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["metrics-latest"],
    queryFn: fetchMetricsLatest,
    refetchInterval: 15000,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="metrics-current-panel">
      <QueryState
        isLoading={isLoading}
        error={error}
        loadingLabel="Loading current metrics..."
        errorLabel="Failed to load latest metrics."
        skeleton={
          <>
            <div className="h-6 w-40 rounded bg-white/5 mb-4" />
            <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
              {[1, 2, 3, 4].map((i) => <div key={i} className="h-20 rounded-lg bg-white/5" />)}
            </div>
          </>
        }
      >
        <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold">
          <Activity className="h-5 w-5 text-blue-400" />
          Current Metrics
        </h2>

        {(!data || !isMetricsRecord(data)) ? (
          <EmptyState
            icon={TrendingUp}
            title="No metrics data yet"
            description="Metrics are scraped from the cloudflared metrics endpoint. Data will appear once the tunnel is active."
          />
        ) : (
          <>
            <div className="grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
              {[
                { label: "HA Connections", value: data.ha_connections, tooltip: "Number of high-availability connections to Cloudflare edge servers" },
                { label: "Active Streams", value: data.active_streams, tooltip: "Currently open HTTP/2 streams through the tunnel" },
                { label: "RTT", value: `${data.smoothed_rtt_ms.toFixed(1)}ms`, tooltip: "Smoothed round-trip time to Cloudflare edge" },
                { label: "Request Errors", value: data.request_errors, tooltip: "Cumulative count of failed requests through the tunnel" },
              ].map((s) => (
                <div key={s.label} className="rounded-lg border border-white/10 bg-white/5 p-3 sm:p-4">
                  <Tooltip content={s.tooltip}>
                    <p className="text-xs sm:text-sm text-slate-300 cursor-help border-b border-dotted border-slate-600">{s.label}</p>
                  </Tooltip>
                  <p className="mt-1 text-xl sm:text-2xl font-bold">{s.value}</p>
                </div>
              ))}
            </div>
            <Tooltip content={new Date(data.scraped_at).toLocaleString()}>
              <p className="mt-3 text-xs text-slate-500 cursor-help">
                Last scraped: {timeAgo(data.scraped_at)}
              </p>
            </Tooltip>
          </>
        )}
      </QueryState>
    </div>
  );
}

const compareMetrics = (a: MetricsRecord, b: MetricsRecord, field: MetricsSortField): number => {
  if (field === "scraped_at") return new Date(a.scraped_at).getTime() - new Date(b.scraped_at).getTime();
  return (a[field] as number) - (b[field] as number);
};

function MetricsHistory() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["metrics-history"],
    queryFn: () => fetchMetricsHistory(24),
    refetchInterval: 60000,
  });

  const [mobilePage, setMobilePage] = useState(0);
  const [desktopPage, setDesktopPage] = useState(0);
  const { sorted, sortField, sortDir, toggleSort } = useSort(data, "scraped_at" as MetricsSortField, compareMetrics, "desc");

  const mobileSlice = sorted.slice(mobilePage * MOBILE_PAGE_SIZE, (mobilePage + 1) * MOBILE_PAGE_SIZE);
  const desktopSlice = sorted.slice(desktopPage * DESKTOP_PAGE_SIZE, (desktopPage + 1) * DESKTOP_PAGE_SIZE);

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="metrics-history-panel">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-2 text-lg font-semibold">
          <BarChart3 className="h-5 w-5 text-slate-300" aria-hidden="true" />
          Metrics History (24h)
        </h2>
        <RefreshButton onClick={() => refetch()} disabled={isFetching} aria-label="Refresh metrics history" data-testid="metrics-refresh" />
      </div>

      <QueryState
        isLoading={isLoading}
        error={error}
        refetch={refetch}
        loadingLabel="Loading metrics history..."
        errorLabel="Failed to load metrics history."
        skeleton={
          <div className="space-y-2">
            {[1, 2, 3].map((i) => <div key={i} className="h-10 rounded bg-white/5" />)}
          </div>
        }
      >
        {data && data.length === 0 && (
          <div className="mt-4 rounded-lg border border-dashed border-white/10 p-4 text-center">
            <p className="text-sm text-slate-300">No metrics recorded in the last 24 hours.</p>
          </div>
        )}

        {/* Mobile: card layout */}
        {sorted.length > 0 && (
          <div className="mt-4 sm:hidden">
            <div className="flex flex-col gap-2" role="list" aria-label="Metrics history cards">
              {mobileSlice.map((r: MetricsRecord) => (
                <div key={r.id} className="rounded-lg bg-black/20 p-3 text-sm" role="listitem">
                  <div className="flex items-center justify-between">
                    <span className="text-slate-300">{new Date(r.scraped_at).toLocaleTimeString()}</span>
                    {r.request_errors > 0 && <span className="text-red-400 text-xs">{r.request_errors} errors</span>}
                  </div>
                  <div className="mt-1 grid grid-cols-3 gap-2 text-xs">
                    <div><span className="text-slate-500">HA</span> <span className="font-mono">{r.ha_connections}</span></div>
                    <div><span className="text-slate-500">Streams</span> <span className="font-mono">{r.active_streams}</span></div>
                    <div><span className="text-slate-500">RTT</span> <span className="font-mono">{r.smoothed_rtt_ms.toFixed(1)}ms</span></div>
                  </div>
                </div>
              ))}
            </div>
            <Pagination page={mobilePage} total={sorted.length} pageSize={MOBILE_PAGE_SIZE} onPageChange={setMobilePage} className="mt-3" />
          </div>
        )}

        {/* Desktop: table layout */}
        {sorted.length > 0 && (
          <div className="mt-4 hidden sm:block" data-testid="metrics-history-table">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <caption className="sr-only">Tunnel metrics history (last 24 hours)</caption>
                <thead>
                  <tr className="border-b border-white/10 text-left text-slate-300">
                    <th scope="col" className="pb-2 pr-4"><SortHeader<MetricsSortField> field="scraped_at" label="Time" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2 pr-4"><SortHeader<MetricsSortField> field="ha_connections" label="HA Conns" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2 pr-4"><SortHeader<MetricsSortField> field="active_streams" label="Streams" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2 pr-4"><SortHeader<MetricsSortField> field="smoothed_rtt_ms" label="RTT (ms)" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                    <th scope="col" className="pb-2"><SortHeader<MetricsSortField> field="request_errors" label="Errors" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                  </tr>
                </thead>
                <tbody>
                  {desktopSlice.map((r: MetricsRecord) => (
                    <tr key={r.id} className="border-b border-white/5">
                      <td className="py-2 pr-4 text-slate-300">
                        {new Date(r.scraped_at).toLocaleTimeString()}
                      </td>
                      <td className="py-2 pr-4 font-mono">{r.ha_connections}</td>
                      <td className="py-2 pr-4 font-mono">{r.active_streams}</td>
                      <td className="py-2 pr-4 font-mono">{r.smoothed_rtt_ms.toFixed(1)}</td>
                      <td className="py-2 font-mono">
                        <span className={r.request_errors > 0 ? "text-red-400" : "text-slate-300"}>
                          {r.request_errors}
                        </span>
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

export default function MetricsPage() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div>
        <h1 className="text-lg sm:text-xl font-semibold">Tunnel Metrics</h1>
        <p className="text-sm text-slate-300">Performance data scraped from the cloudflared metrics endpoint.</p>
      </div>
      <LatestMetrics />
      <MetricsHistory />
    </div>
  );
}
