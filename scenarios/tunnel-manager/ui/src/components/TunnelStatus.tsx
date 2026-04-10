import { useQuery } from "@tanstack/react-query";
import { Activity, RefreshCw, RotateCcw } from "lucide-react";
import { fetchTunnelHealth, type TunnelStatus as TunnelStatusType } from "../lib/api";
import { timeAgo } from "../lib/utils";
import { Button } from "./ui/button";
import { Tooltip } from "./ui/tooltip";
import { StatusBadge, statusToVariant } from "./ui/status-badge";

export function TunnelStatusPanel() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["tunnel-health"],
    queryFn: fetchTunnelHealth,
    refetchInterval: 30000,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="tunnel-health-panel">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="h-5 w-5 text-slate-300" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Tunnel Health</h2>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching} aria-label="Refresh tunnel health" data-testid="tunnel-health-refresh">
          <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} aria-hidden="true" />
        </Button>
      </div>

      {isLoading && (
        <div className="mt-4 space-y-3 animate-pulse" role="status">
          <span className="sr-only">Checking tunnel health...</span>
          <div className="h-8 w-48 rounded bg-white/5" />
          <div className="grid grid-cols-2 gap-3">
            <div className="h-16 rounded-lg bg-white/5" />
            <div className="h-16 rounded-lg bg-white/5" />
          </div>
        </div>
      )}
      {error && (
        <div className="mt-4 rounded-lg border border-red-500/20 bg-red-500/5 p-4" role="alert">
          <p className="text-sm text-red-400">Unable to reach API. Is the scenario running?</p>
          <p className="mt-1 text-xs text-slate-500">Check that the tunnel-manager API is started and accessible.</p>
          <Button variant="outline" size="sm" onClick={() => refetch()} className="mt-3 text-red-400 border-red-400/30 hover:bg-red-500/10" data-testid="tunnel-health-retry">
            <RotateCcw className="h-3 w-3 mr-2" aria-hidden="true" />
            Retry
          </Button>
        </div>
      )}
      {data && <TunnelStatusDetails status={data} />}
    </div>
  );
}

function TunnelStatusDetails({ status }: { status: TunnelStatusType }) {
  return (
    <div className="mt-4 space-y-3">
      <div className="flex items-center gap-3">
        <StatusBadge
          variant={statusToVariant(status.status)}
          label={status.status.toUpperCase()}
          className="text-base font-bold px-3 py-1"
          data-testid="tunnel-status-indicator"
        />
        <Tooltip content={status.score >= 80 ? "Tunnel is performing well" : status.score >= 50 ? "Tunnel has degraded performance" : "Tunnel needs attention"}>
          <span className={`ml-auto text-2xl font-bold cursor-help ${status.score >= 80 ? "text-green-400" : status.score >= 50 ? "text-yellow-400" : "text-red-400"}`} data-testid="tunnel-score-value">{status.score}/100</span>
        </Tooltip>
      </div>

      <div className="grid grid-cols-2 gap-3 text-sm">
        <div className="rounded-lg bg-black/20 p-3">
          <Tooltip content="Status of the cloudflared systemd service on this host">
            <p className="text-slate-300 cursor-help border-b border-dotted border-slate-600">Systemd</p>
          </Tooltip>
          <p className="font-medium text-slate-200">{status.systemd}</p>
        </div>
        <div className="rounded-lg bg-black/20 p-3">
          <Tooltip content="Whether the tunnel's /ready endpoint responds successfully">
            <p className="text-slate-300 cursor-help border-b border-dotted border-slate-600">Ready Endpoint</p>
          </Tooltip>
          <p className="font-medium text-slate-200">
            {status.ready}
            {status.ready_latency_ms > 0 && ` (${status.ready_latency_ms}ms)`}
          </p>
        </div>
      </div>

      {status.message && (
        <p className="text-sm text-slate-300">{status.message}</p>
      )}
      <Tooltip content={new Date(status.checked_at).toLocaleString()}>
        <p className="text-xs text-slate-500 cursor-help">Last checked: {timeAgo(status.checked_at)}</p>
      </Tooltip>
    </div>
  );
}
