// Uptime statistics display component
// [REQ:PERSIST-HISTORY-001] [REQ:UI-EVENTS-001]
import { useQuery } from "@tanstack/react-query";
import { TrendingUp, Clock } from "lucide-react";
import { fetchUptimeStats } from "../lib/api";
import { selectors } from "../consts/selectors";

function getUptimeColor(percentage: number): string {
  if (percentage >= 99) return "text-emerald-400";
  if (percentage >= 95) return "text-emerald-300";
  if (percentage >= 90) return "text-amber-400";
  if (percentage >= 80) return "text-amber-500";
  return "text-red-400";
}

function getUptimeLabel(percentage: number): string {
  if (percentage >= 99) return "Excellent";
  if (percentage >= 95) return "Good";
  if (percentage >= 90) return "Fair";
  if (percentage >= 80) return "Poor";
  return "Critical";
}

export function UptimeStats() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["uptime"],
    queryFn: fetchUptimeStats,
    refetchInterval: 60000, // Refresh every minute
    staleTime: 30000, // Consider stale after 30s
  });

  if (isLoading) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 p-4">
        <div className="flex items-center gap-2 mb-3">
          <TrendingUp size={18} className="text-blue-400" />
          <h3 className="font-medium text-sm">Uptime (24h)</h3>
        </div>
        <p className="text-sm text-slate-500">Loading...</p>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 p-4">
        <div className="flex items-center gap-2 mb-3">
          <TrendingUp size={18} className="text-blue-400" />
          <h3 className="font-medium text-sm">Uptime (24h)</h3>
        </div>
        <p className="text-sm text-red-400">Failed to load</p>
      </div>
    );
  }

  const uptimeColor = getUptimeColor(data.uptimePercentage);
  const uptimeLabel = getUptimeLabel(data.uptimePercentage);

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4" data-testid={selectors.uptimeStats}>
      {/* Header */}
      <div className="flex items-center gap-2 mb-3">
        <TrendingUp size={18} className="text-blue-400" />
        <h3 className="font-medium text-sm">Uptime ({data.windowHours}h)</h3>
      </div>

      {/* Main percentage */}
      <div className="text-center py-2">
        <div className={`text-3xl font-bold ${uptimeColor}`}>
          {data.uptimePercentage.toFixed(1)}%
        </div>
        <div className={`text-xs ${uptimeColor} opacity-80 mt-1`}>{uptimeLabel}</div>
      </div>

      {/* Progress bar */}
      <div className="mt-3 h-2 bg-slate-800 rounded-full overflow-hidden">
        <div
          className={`h-full transition-all duration-500 ${
            data.uptimePercentage >= 95 ? "bg-emerald-500" :
            data.uptimePercentage >= 80 ? "bg-amber-500" : "bg-red-500"
          }`}
          style={{ width: `${Math.min(100, data.uptimePercentage)}%` }}
        />
      </div>

      {/* Breakdown */}
      <div className="mt-3 grid grid-cols-3 gap-2 text-center text-xs">
        <div>
          <div className="text-emerald-400 font-medium">{data.okEvents}</div>
          <div className="text-slate-500">OK</div>
        </div>
        <div>
          <div className="text-amber-400 font-medium">{data.warningEvents}</div>
          <div className="text-slate-500">Warn</div>
        </div>
        <div>
          <div className="text-red-400 font-medium">{data.criticalEvents}</div>
          <div className="text-slate-500">Crit</div>
        </div>
      </div>

      {/* Total events */}
      <div className="mt-3 pt-3 border-t border-white/10 flex items-center justify-center gap-1.5 text-xs text-slate-500">
        <Clock size={12} />
        <span>{data.totalEvents} checks in {data.windowHours}h</span>
      </div>
    </div>
  );
}
