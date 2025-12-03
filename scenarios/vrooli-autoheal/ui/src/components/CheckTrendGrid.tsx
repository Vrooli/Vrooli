// Per-check trend grid showing individual check health over time
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useMemo } from "react";
import { TimelineEvent, HealthStatus, CheckTrend as APICheckTrend } from "../lib/api";
import { StatusIcon } from "./StatusIcon";

interface CheckTrendGridProps {
  /** Backend trends data (preferred) */
  trends?: APICheckTrend[];
  /** Timeline events for fallback client-side aggregation */
  events?: TimelineEvent[];
  /** Called when a check row is clicked */
  onCheckClick?: (checkId: string) => void;
}

interface LocalCheckTrend {
  checkId: string;
  total: number;
  ok: number;
  warning: number;
  critical: number;
  uptimePercent: number;
  currentStatus: HealthStatus;
  recentStatuses: HealthStatus[];
}

// Mini sparkline bar showing recent status history
function StatusSparkline({ statuses }: { statuses: HealthStatus[] }) {
  // Show last 12 statuses as small bars
  const displayStatuses = statuses.slice(0, 12);

  return (
    <div className="flex items-center gap-0.5">
      {displayStatuses.map((status, idx) => (
        <div
          key={idx}
          className={`w-1.5 h-4 rounded-sm transition-all ${
            status === "ok"
              ? "bg-emerald-500"
              : status === "warning"
              ? "bg-amber-500"
              : "bg-red-500"
          }`}
          style={{
            opacity: 0.4 + (idx / displayStatuses.length) * 0.6,
          }}
        />
      ))}
      {displayStatuses.length === 0 && (
        <span className="text-xs text-slate-500">No data</span>
      )}
    </div>
  );
}

export function CheckTrendGrid({ trends: backendTrends, events = [], onCheckClick }: CheckTrendGridProps) {
  // Use backend trends if available, otherwise fall back to client-side aggregation
  const trends = useMemo<LocalCheckTrend[]>(() => {
    // If we have backend trends, convert them to local format
    if (backendTrends && backendTrends.length > 0) {
      return backendTrends.map((t) => ({
        checkId: t.checkId,
        total: t.total,
        ok: t.ok,
        warning: t.warning,
        critical: t.critical,
        uptimePercent: t.uptimePercent,
        currentStatus: t.currentStatus as HealthStatus,
        recentStatuses: t.recentStatuses as HealthStatus[],
      }));
    }

    // Fallback: Group events by checkId
    const byCheck = new Map<string, TimelineEvent[]>();
    for (const event of events) {
      const list = byCheck.get(event.checkId) || [];
      list.push(event);
      byCheck.set(event.checkId, list);
    }

    // Calculate trends for each check
    const checkTrends: LocalCheckTrend[] = [];
    byCheck.forEach((checkEvents, checkId) => {
      // Sort by time (newest first)
      const sorted = [...checkEvents].sort(
        (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
      );

      const ok = sorted.filter((e) => e.status === "ok").length;
      const warning = sorted.filter((e) => e.status === "warning").length;
      const critical = sorted.filter((e) => e.status === "critical").length;
      const total = sorted.length;

      checkTrends.push({
        checkId,
        total,
        ok,
        warning,
        critical,
        uptimePercent: total > 0 ? (ok / total) * 100 : 100,
        currentStatus: (sorted[0]?.status as HealthStatus) || "ok",
        recentStatuses: sorted.slice(0, 12).map((e) => e.status as HealthStatus),
      });
    });

    // Sort by uptime (worst first), then by checkId
    return checkTrends.sort((a, b) => {
      if (a.uptimePercent !== b.uptimePercent) {
        return a.uptimePercent - b.uptimePercent;
      }
      return a.checkId.localeCompare(b.checkId);
    });
  }, [backendTrends, events]);

  if (trends.length === 0) {
    return (
      <div className="text-center py-8 text-slate-500">
        <p>No check data available yet</p>
        <p className="text-xs mt-1">Run a health check tick to see trends</p>
      </div>
    );
  }

  return (
    <div className="space-y-2" data-testid="autoheal-trends-check-grid">
      {trends.map((trend) => (
        <div
          key={trend.checkId}
          onClick={() => onCheckClick?.(trend.checkId)}
          className={`flex items-center gap-4 p-3 rounded-lg bg-white/[0.02] hover:bg-white/[0.04] transition-colors ${
            onCheckClick ? "cursor-pointer" : ""
          }`}
          role={onCheckClick ? "button" : undefined}
          tabIndex={onCheckClick ? 0 : undefined}
          onKeyDown={(e) => {
            if (onCheckClick && (e.key === "Enter" || e.key === " ")) {
              e.preventDefault();
              onCheckClick(trend.checkId);
            }
          }}
        >
          {/* Status icon */}
          <div className="flex-shrink-0">
            <StatusIcon status={trend.currentStatus} size={16} />
          </div>

          {/* Check name */}
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-slate-200 truncate">
              {trend.checkId}
            </div>
            <div className="text-xs text-slate-500">
              {trend.total} checks
            </div>
          </div>

          {/* Sparkline */}
          <div className="flex-shrink-0">
            <StatusSparkline statuses={trend.recentStatuses} />
          </div>

          {/* Uptime percentage */}
          <div className="flex-shrink-0 w-16 text-right">
            <div
              className={`text-sm font-medium ${
                trend.uptimePercent >= 99
                  ? "text-emerald-400"
                  : trend.uptimePercent >= 90
                  ? "text-amber-400"
                  : "text-red-400"
              }`}
            >
              {trend.uptimePercent.toFixed(0)}%
            </div>
          </div>

          {/* Progress bar */}
          <div className="flex-shrink-0 w-24 h-2 bg-slate-800 rounded-full overflow-hidden">
            <div className="h-full flex">
              <div
                className="h-full bg-emerald-500"
                style={{ width: `${(trend.ok / trend.total) * 100}%` }}
              />
              <div
                className="h-full bg-amber-500"
                style={{ width: `${(trend.warning / trend.total) * 100}%` }}
              />
              <div
                className="h-full bg-red-500"
                style={{ width: `${(trend.critical / trend.total) * 100}%` }}
              />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
