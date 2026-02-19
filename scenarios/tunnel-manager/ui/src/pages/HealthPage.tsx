import { useQuery } from "@tanstack/react-query";
import { Heart } from "lucide-react";
import { fetchDetailedHealth, type DetailedHealth } from "../lib/api";
import { timeAgo } from "../lib/utils";
import { Tooltip } from "../components/ui/tooltip";
import { StatusBadge, statusToVariant } from "../components/ui/status-badge";
import { RefreshButton } from "../components/ui/refresh-button";
import { QueryState } from "../components/ui/query-state";

function boolVariant(val?: boolean) {
  if (val === undefined) return "neutral" as const;
  return val ? "success" as const : "error" as const;
}

function boolLabel(val?: boolean) {
  if (val === undefined) return "—";
  return val ? "up" : "down";
}

function TunnelOverview({ health }: { health: DetailedHealth }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="health-overview-panel">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-2 text-lg font-semibold">
          <Heart className="h-5 w-5 text-blue-400" aria-hidden="true" />
          Tunnel Overview
        </h2>
        <StatusBadge variant={statusToVariant(health.status)} label={health.status} className="px-3 py-1 text-sm" data-testid="health-status-badge" />
      </div>

      <div className="mt-4 grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-lg border border-white/10 bg-white/5 p-3 sm:p-4">
          <Tooltip content="Whether the tunnel's /ready endpoint responds successfully">
            <p className="text-xs sm:text-sm text-slate-300 cursor-help border-b border-dotted border-slate-600">Ready</p>
          </Tooltip>
          <p className="mt-1 text-lg font-bold">{health.tunnel.ready}</p>
        </div>
        <div className="rounded-lg border border-white/10 bg-white/5 p-3 sm:p-4">
          <Tooltip content="Status of the cloudflared systemd service">
            <p className="text-xs sm:text-sm text-slate-300 cursor-help border-b border-dotted border-slate-600">Systemd</p>
          </Tooltip>
          <p className="mt-1 text-lg font-bold">{health.tunnel.systemd || "—"}</p>
        </div>
        <div className="rounded-lg border border-white/10 bg-white/5 p-3 sm:p-4">
          <Tooltip content="Composite health score (0-100) based on readiness, systemd state, and route health">
            <p className="text-xs sm:text-sm text-slate-300 cursor-help border-b border-dotted border-slate-600">Score</p>
          </Tooltip>
          <p className="mt-1 text-lg font-bold">{health.tunnel.score}/100</p>
        </div>
        {health.tunnel.ready_latency_ms !== undefined && (
          <div className="rounded-lg border border-white/10 bg-white/5 p-3 sm:p-4">
            <Tooltip content="Latency of the tunnel's /ready endpoint response">
              <p className="text-xs sm:text-sm text-slate-300 cursor-help border-b border-dotted border-slate-600">Ready Latency</p>
            </Tooltip>
            <p className="mt-1 text-lg font-bold">{health.tunnel.ready_latency_ms}ms</p>
          </div>
        )}
      </div>
    </div>
  );
}

function RouteHealthTable({ routes }: { routes: DetailedHealth["routes"] }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="health-route-table">
      <h2 className="mb-4 text-lg font-semibold">Route Health</h2>

      {routes.length === 0 ? (
        <div className="rounded-lg border border-dashed border-white/10 p-4 text-center">
          <p className="text-sm text-slate-300">No routes configured.</p>
          <p className="mt-1 text-xs text-slate-500">Add routes to monitor their internal and external health status.</p>
        </div>
      ) : (
        <>
          {/* Mobile: card layout */}
          <div className="flex flex-col gap-2 sm:hidden" role="list" aria-label="Route health cards">
            {routes.map((r) => (
              <div key={r.subdomain} className="rounded-lg bg-black/20 p-3" role="listitem">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-slate-200 text-sm">{r.subdomain}</span>
                  <StatusBadge variant={r.enabled ? "success" : "neutral"} label={r.enabled ? "enabled" : "disabled"} />
                </div>
                <p className="text-xs text-slate-300 mt-1">{r.scenario_name}</p>
                <div className="mt-2 flex gap-3">
                  <div className="flex items-center gap-1">
                    <Tooltip content="Whether the route responds on its local port">
                      <span className="text-xs text-slate-500 cursor-help">Internal:</span>
                    </Tooltip>
                    <StatusBadge variant={boolVariant(r.internal_up)} label={boolLabel(r.internal_up)} />
                  </div>
                  <div className="flex items-center gap-1">
                    <Tooltip content="Whether the route is accessible via the Cloudflare tunnel">
                      <span className="text-xs text-slate-500 cursor-help">External:</span>
                    </Tooltip>
                    <StatusBadge variant={boolVariant(r.external_up)} label={boolLabel(r.external_up)} />
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Desktop: table layout */}
          <div className="hidden sm:block overflow-x-auto">
            <table className="w-full text-sm">
              <caption className="sr-only">Route health status</caption>
              <thead>
                <tr className="border-b border-white/10 text-left text-slate-300">
                  <th scope="col" className="pb-2 pr-4">Subdomain</th>
                  <th scope="col" className="pb-2 pr-4">Scenario</th>
                  <th scope="col" className="pb-2 pr-4">Enabled</th>
                  <th scope="col" className="pb-2 pr-4">
                    <Tooltip content="Whether the route responds on its local port">
                      <span className="cursor-help border-b border-dotted border-slate-600">Internal</span>
                    </Tooltip>
                  </th>
                  <th scope="col" className="pb-2">
                    <Tooltip content="Whether the route is accessible via the Cloudflare tunnel">
                      <span className="cursor-help border-b border-dotted border-slate-600">External</span>
                    </Tooltip>
                  </th>
                </tr>
              </thead>
              <tbody>
                {routes.map((r) => (
                  <tr key={r.subdomain} className="border-b border-white/5">
                    <td className="py-2 pr-4 font-medium">{r.subdomain}</td>
                    <td className="py-2 pr-4 text-slate-300">{r.scenario_name}</td>
                    <td className="py-2 pr-4">
                      <StatusBadge variant={r.enabled ? "success" : "neutral"} label={r.enabled ? "yes" : "no"} />
                    </td>
                    <td className="py-2 pr-4">
                      <StatusBadge variant={boolVariant(r.internal_up)} label={boolLabel(r.internal_up)} />
                    </td>
                    <td className="py-2">
                      <StatusBadge variant={boolVariant(r.external_up)} label={boolLabel(r.external_up)} />
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

export default function HealthPage() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["detailed-health"],
    queryFn: fetchDetailedHealth,
    refetchInterval: 15000,
  });

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-lg sm:text-xl font-semibold">Detailed Health</h1>
        <RefreshButton onClick={() => refetch()} disabled={isFetching} aria-label="Refresh health data" data-testid="health-refresh" />
      </div>

      <QueryState
        isLoading={isLoading}
        error={error}
        refetch={refetch}
        loadingLabel="Loading health data..."
        errorLabel="Failed to load detailed health."
        skeleton={
          <div className="space-y-4">
            <div className="h-40 rounded-xl bg-white/5" />
            <div className="h-32 rounded-xl bg-white/5" />
          </div>
        }
      >
        {data && (
          <>
            <TunnelOverview health={data} />
            <RouteHealthTable routes={data.routes} />
            <Tooltip content={new Date(data.timestamp).toLocaleString()}>
              <p className="text-xs text-slate-500 cursor-help">
                Last checked: {timeAgo(data.timestamp)}
              </p>
            </Tooltip>
          </>
        )}
      </QueryState>
    </div>
  );
}
