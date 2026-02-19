import { useQuery } from "@tanstack/react-query";
import { Activity, RefreshCw } from "lucide-react";
import { fetchTunnelHealth, type TunnelStatus as TunnelStatusType } from "../lib/api";
import { Button } from "./ui/button";

function statusColor(status: string): string {
  switch (status) {
    case "healthy": return "text-green-400";
    case "degraded": return "text-yellow-400";
    case "unhealthy": return "text-red-400";
    default: return "text-slate-400";
  }
}

function statusIcon(status: string): string {
  switch (status) {
    case "healthy": return "bg-green-500";
    case "degraded": return "bg-yellow-500";
    case "unhealthy": return "bg-red-500";
    default: return "bg-slate-500";
  }
}

export function TunnelStatusPanel() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["tunnel-health"],
    queryFn: fetchTunnelHealth,
    refetchInterval: 30000,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="h-5 w-5 text-slate-400" />
          <h2 className="text-lg font-semibold">Tunnel Health</h2>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
          <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
        </Button>
      </div>

      {isLoading && <p className="mt-4 text-slate-400">Checking tunnel health...</p>}
      {error && (
        <p className="mt-4 text-red-400">Unable to reach API. Is the scenario running?</p>
      )}
      {data && <TunnelStatusDetails status={data} />}
    </div>
  );
}

function TunnelStatusDetails({ status }: { status: TunnelStatusType }) {
  return (
    <div className="mt-4 space-y-3">
      <div className="flex items-center gap-3">
        <span className={`h-3 w-3 rounded-full ${statusIcon(status.status)}`} />
        <span className={`text-xl font-bold ${statusColor(status.status)}`}>
          {status.status.toUpperCase()}
        </span>
        <span className="ml-auto text-2xl font-bold text-slate-200">{status.score}/100</span>
      </div>

      <div className="grid grid-cols-2 gap-3 text-sm">
        <div className="rounded-lg bg-black/20 p-3">
          <p className="text-slate-400">Systemd</p>
          <p className="font-medium text-slate-200">{status.systemd}</p>
        </div>
        <div className="rounded-lg bg-black/20 p-3">
          <p className="text-slate-400">Ready Endpoint</p>
          <p className="font-medium text-slate-200">
            {status.ready}
            {status.ready_latency_ms > 0 && ` (${status.ready_latency_ms}ms)`}
          </p>
        </div>
      </div>

      {status.message && (
        <p className="text-sm text-slate-400">{status.message}</p>
      )}
      <p className="text-xs text-slate-500">Last checked: {new Date(status.checked_at).toLocaleString()}</p>
    </div>
  );
}
