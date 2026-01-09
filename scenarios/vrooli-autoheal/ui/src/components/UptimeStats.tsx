// Uptime statistics display component with sparkline
// [REQ:PERSIST-HISTORY-001] [REQ:UI-EVENTS-001] [REQ:FAIL-SAFE-001]
import { useQuery } from "@tanstack/react-query";
import { TrendingUp, Clock, ChevronRight } from "lucide-react";
import { fetchUptimeStats, fetchTimeline, HealthStatus } from "../lib/api";
import { ErrorDisplay } from "./ErrorDisplay";
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

// Sparkline component showing recent status history
function StatusSparkline({ statuses }: { statuses: HealthStatus[] }) {
  // Show last 20 statuses as small bars
  const displayStatuses = statuses.slice(0, 20);

  if (displayStatuses.length === 0) {
    return <span className="text-xs text-slate-500">No data</span>;
  }

  return (
    <div className="flex items-end gap-px h-6">
      {displayStatuses.map((status, idx) => (
        <div
          key={idx}
          className={`w-1 rounded-t-sm transition-all ${
            status === "ok"
              ? "bg-emerald-500 h-full"
              : status === "warning"
              ? "bg-amber-500 h-4"
              : "bg-red-500 h-5"
          }`}
          style={{
            opacity: 0.5 + (idx / displayStatuses.length) * 0.5,
          }}
        />
      ))}
    </div>
  );
}

interface UptimeStatsProps {
  onShowTrends?: () => void;
}

export function UptimeStats({ onShowTrends }: UptimeStatsProps) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["uptime"],
    queryFn: fetchUptimeStats,
    refetchInterval: 60000, // Refresh every minute
    staleTime: 30000, // Consider stale after 30s
    retry: 2, // Retry failed requests twice
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
  });

  // Fetch timeline for sparkline
  const { data: timelineData } = useQuery({
    queryKey: ["timeline"],
    queryFn: fetchTimeline,
    refetchInterval: 30000,
  });

  // Extract recent statuses for sparkline (newest first)
  const recentStatuses: HealthStatus[] = (timelineData?.events || [])
    .slice(0, 20)
    .map((e) => e.status as HealthStatus);

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
        <ErrorDisplay
          error={error}
          onRetry={() => refetch()}
          compact
        />
      </div>
    );
  }

  const uptimeColor = getUptimeColor(data.uptimePercentage);
  const uptimeLabel = getUptimeLabel(data.uptimePercentage);

  const isClickable = !!onShowTrends;

  return (
    <div
      className={`rounded-xl border border-white/10 bg-white/5 p-4 transition-colors ${
        isClickable ? "cursor-pointer hover:bg-white/[0.08] hover:border-white/20" : ""
      }`}
      data-testid={selectors.uptimeStats}
      onClick={onShowTrends}
      role={isClickable ? "button" : undefined}
      tabIndex={isClickable ? 0 : undefined}
      onKeyDown={isClickable ? (e) => { if (e.key === "Enter" || e.key === " ") onShowTrends?.(); } : undefined}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <TrendingUp size={18} className="text-blue-400" />
          <h3 className="font-medium text-sm">Uptime ({data.windowHours}h)</h3>
        </div>
        {isClickable && (
          <ChevronRight size={16} className="text-slate-500" />
        )}
      </div>

      {/* Main percentage and sparkline row */}
      <div className="flex items-center justify-between gap-4 py-2">
        <div>
          <div className={`text-3xl font-bold ${uptimeColor}`}>
            {data.uptimePercentage.toFixed(1)}%
          </div>
          <div className={`text-xs ${uptimeColor} opacity-80 mt-1`}>{uptimeLabel}</div>
        </div>
        <div className="flex-1">
          <StatusSparkline statuses={recentStatuses} />
        </div>
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

      {/* Total events and hint */}
      <div className="mt-3 pt-3 border-t border-white/10 flex items-center justify-between text-xs text-slate-500">
        <div className="flex items-center gap-1.5">
          <Clock size={12} />
          <span>{data.totalEvents} checks in {data.windowHours}h</span>
        </div>
        {isClickable && (
          <span className="text-blue-400">View trends →</span>
        )}
      </div>
    </div>
  );
}
