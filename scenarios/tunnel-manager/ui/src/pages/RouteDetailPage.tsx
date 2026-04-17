import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ExternalLink } from "lucide-react";
import { fetchRoute, fetchProbeHistory, type Route, type ProbeResult } from "../lib/api";
import { statusToVariant, timeAgo } from "../lib/utils";
import { Tooltip } from "../components/ui/tooltip";
import { StatusBadge } from "../components/ui/status-badge";
import { RefreshButton } from "../components/ui/refresh-button";
import { QueryState } from "../components/ui/query-state";

function RouteInfo({ route }: { route: Route }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold truncate">{route.subdomain}</h2>
        <StatusBadge variant={route.enabled ? "success" : "neutral"} label={route.enabled ? "enabled" : "disabled"} />
      </div>

      <dl className="mt-4 grid gap-4 grid-cols-1 sm:grid-cols-2">
        <div>
          <dt className="text-sm text-slate-300">Scenario</dt>
          <dd className="mt-1 font-medium">{route.scenario_name}</dd>
        </div>
        <div>
          <dt className="text-sm text-slate-300">Local Port</dt>
          <dd className="mt-1 font-mono">{route.local_port}</dd>
        </div>
        <div>
          <dt className="text-sm text-slate-300">Health Path</dt>
          <dd className="mt-1 font-mono">{route.health_path || "—"}</dd>
        </div>
        <div>
          <dt className="text-sm text-slate-300">Public URL</dt>
          <dd className="mt-1 break-all">
            {route.public_url ? (
              <a href={route.public_url} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 text-blue-400 hover:text-blue-300">
                {route.public_url}
                <ExternalLink className="h-3 w-3 shrink-0" aria-hidden="true" />
                <span className="sr-only">(opens in new tab)</span>
              </a>
            ) : "—"}
          </dd>
        </div>
        <div>
          <dt className="text-sm text-slate-300">Created</dt>
          <dd className="mt-1 text-slate-300">
            <Tooltip content={new Date(route.created_at).toLocaleString()}>
              <span className="cursor-help">{timeAgo(route.created_at)}</span>
            </Tooltip>
          </dd>
        </div>
        <div>
          <dt className="text-sm text-slate-300">Updated</dt>
          <dd className="mt-1 text-slate-300">
            <Tooltip content={new Date(route.updated_at).toLocaleString()}>
              <span className="cursor-help">{timeAgo(route.updated_at)}</span>
            </Tooltip>
          </dd>
        </div>
      </dl>
    </div>
  );
}

function RouteProbeHistory({ routeId, subdomain }: { routeId: number; subdomain: string }) {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["probe-history"],
    queryFn: fetchProbeHistory,
    refetchInterval: 30000,
  });

  const filtered = data?.filter((p: ProbeResult) => p.route_id === routeId || p.subdomain === subdomain) ?? [];

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Probe History</h2>
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
            {[1, 2].map((i) => <div key={i} className="h-10 rounded bg-white/5" />)}
          </div>
        }
      >
        {!isLoading && !error && filtered.length === 0 && (
          <div className="mt-4 rounded-lg border border-dashed border-white/10 p-4 text-center">
            <p className="text-sm text-slate-300">No probe results for this route.</p>
            <p className="mt-1 text-xs text-slate-500">Run probes from the Probes page to generate results.</p>
          </div>
        )}

        {/* Mobile: card layout */}
        {filtered.length > 0 && (
          <div className="mt-4 flex flex-col gap-2 sm:hidden" aria-label="Probe history cards" role="list">
            {filtered.map((r: ProbeResult, i: number) => (
              <div key={i} className="rounded-lg bg-black/20 p-3" role="listitem">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-300">{r.probe_type}</span>
                  <StatusBadge variant={statusToVariant(r.status)} label={r.status} />
                </div>
                <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-300">
                  {r.latency_ms > 0 && <span className="font-mono">{r.latency_ms}ms</span>}
                  {r.error_msg && <span className="text-red-400 break-all">{r.error_msg}</span>}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Desktop: table layout */}
        {filtered.length > 0 && (
          <div className="mt-4 hidden sm:block overflow-x-auto">
            <table className="w-full text-sm">
              <caption className="sr-only">Probe history for route {subdomain}</caption>
              <thead>
                <tr className="border-b border-white/10 text-left text-slate-300">
                  <th scope="col" className="pb-2 pr-4">Type</th>
                  <th scope="col" className="pb-2 pr-4">Status</th>
                  <th scope="col" className="pb-2 pr-4">Latency</th>
                  <th scope="col" className="pb-2">Error</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((r: ProbeResult, i: number) => (
                  <tr key={i} className="border-b border-white/5">
                    <td className="py-2 pr-4 text-slate-300">{r.probe_type}</td>
                    <td className="py-2 pr-4">
                      <StatusBadge variant={statusToVariant(r.status)} label={r.status} />
                    </td>
                    <td className="py-2 pr-4 font-mono text-slate-300">
                      {r.latency_ms > 0 ? `${r.latency_ms}ms` : "—"}
                    </td>
                    <td className="py-2 text-slate-300 break-words max-w-[200px]" title={r.error_msg || undefined}>
                      {r.error_msg || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </QueryState>
    </div>
  );
}

export default function RouteDetailPage() {
  const { id } = useParams<{ id: string }>();
  const routeId = Number(id);

  const { data: route, isLoading, error } = useQuery({
    queryKey: ["route", routeId],
    queryFn: () => fetchRoute(routeId),
    enabled: !isNaN(routeId),
  });

  if (isNaN(routeId)) {
    return <p className="text-red-400" role="alert">Invalid route ID.</p>;
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      <Link to="/routes" className="inline-flex items-center gap-1 text-sm text-slate-300 hover:text-slate-200 transition-colors focus-visible:ring-1 focus-visible:ring-blue-500/50 focus-visible:outline-none rounded py-2">
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to routes
      </Link>

      <QueryState
        isLoading={isLoading}
        error={error}
        loadingLabel="Loading route details..."
        errorLabel="Failed to load route."
        skeleton={
          <div className="space-y-4">
            <div className="h-48 rounded-xl bg-white/5" />
            <div className="h-32 rounded-xl bg-white/5" />
          </div>
        }
      >
        {error && (
          <Link to="/routes" className="mt-2 inline-block text-sm text-blue-400 hover:text-blue-300">
            Return to routes
          </Link>
        )}

        {route && (
          <>
            <RouteInfo route={route} />
            <RouteProbeHistory routeId={routeId} subdomain={route.subdomain} />
          </>
        )}
      </QueryState>
    </div>
  );
}
