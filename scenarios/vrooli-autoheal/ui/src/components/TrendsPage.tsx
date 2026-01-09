// Trends page showing comprehensive historical data
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useQuery } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import { TrendingUp, Activity, AlertTriangle, Clock, Download } from "lucide-react";
import { fetchTimeline, fetchUptimeStats, fetchIncidents, fetchCheckTrends, TimelineEvent, Incident, CheckTrend } from "../lib/api";
import { ErrorDisplay } from "./ErrorDisplay";
import { StatusIcon } from "./StatusIcon";
import { UptimeTrendChart } from "./UptimeTrendChart";
import { CheckTrendGrid } from "./CheckTrendGrid";
import { CheckDetailModal } from "./CheckDetailModal";
import { exportTrendDataToCSV } from "../lib/export";
import { useCheckMetadata } from "../contexts/CheckMetadataContext";

// Helper to detect status transitions (incidents)
function detectIncidents(events: TimelineEvent[]): Array<{
  timestamp: string;
  checkId: string;
  fromStatus: string;
  toStatus: string;
  message: string;
}> {
  const incidents: Array<{
    timestamp: string;
    checkId: string;
    fromStatus: string;
    toStatus: string;
    message: string;
  }> = [];

  // Group events by checkId and sort by time (oldest first for transition detection)
  const byCheck = new Map<string, TimelineEvent[]>();
  for (const event of events) {
    const list = byCheck.get(event.checkId) || [];
    list.push(event);
    byCheck.set(event.checkId, list);
  }

  // Find transitions within each check
  byCheck.forEach((checkEvents) => {
    // Sort oldest to newest for transition detection
    const sorted = [...checkEvents].sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );

    for (let i = 1; i < sorted.length; i++) {
      const prev = sorted[i - 1];
      const curr = sorted[i];
      if (prev.status !== curr.status) {
        incidents.push({
          timestamp: curr.timestamp,
          checkId: curr.checkId,
          fromStatus: prev.status,
          toStatus: curr.status,
          message: curr.message,
        });
      }
    }
  });

  // Sort incidents by time (newest first)
  return incidents.sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  );
}

function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}h ago`;
  const diffDays = Math.floor(diffHour / 24);
  return `${diffDays}d ago`;
}

// Time window options
type TimeWindow = { label: string; hours: number; buckets: number };
const TIME_WINDOWS: TimeWindow[] = [
  { label: "6h", hours: 6, buckets: 12 },
  { label: "12h", hours: 12, buckets: 24 },
  { label: "24h", hours: 24, buckets: 24 },
  { label: "7d", hours: 168, buckets: 28 },
];

export function TrendsPage() {
  const [selectedWindow, setSelectedWindow] = useState<TimeWindow>(TIME_WINDOWS[2]); // Default 24h
  const [selectedCheckId, setSelectedCheckId] = useState<string | null>(null);
  const { getTitle } = useCheckMetadata();

  // Uptime stats for overall percentage
  const { data: uptimeData, isLoading: uptimeLoading, error: uptimeError, refetch: refetchUptime } = useQuery({
    queryKey: ["uptime"],
    queryFn: fetchUptimeStats,
    refetchInterval: 60000,
  });

  // Check trends from dedicated endpoint
  const { data: checkTrendsData, isLoading: trendsLoading, error: trendsError, refetch: refetchTrends } = useQuery({
    queryKey: ["check-trends", selectedWindow.hours],
    queryFn: () => fetchCheckTrends(selectedWindow.hours),
    refetchInterval: 60000,
  });

  // Incidents from dedicated endpoint
  const { data: incidentsData, isLoading: incidentsLoading, error: incidentsError, refetch: refetchIncidents } = useQuery({
    queryKey: ["incidents", selectedWindow.hours],
    queryFn: () => fetchIncidents(selectedWindow.hours, 50),
    refetchInterval: 60000,
  });

  // Fallback to timeline for backwards compatibility
  const { data: timelineData, isLoading: timelineLoading, error: timelineError, refetch: refetchTimeline } = useQuery({
    queryKey: ["timeline"],
    queryFn: fetchTimeline,
    refetchInterval: 30000,
  });

  // Use backend incidents if available, otherwise fallback to client-side detection
  const incidents = incidentsData?.incidents ?? (timelineData?.events ? detectIncidents(timelineData.events) : []);

  // Handle check drill-down
  const handleCheckClick = useCallback((checkId: string) => {
    setSelectedCheckId(checkId);
  }, []);

  // Handle CSV export
  const handleExport = useCallback(() => {
    const checkTrends = checkTrendsData?.trends ?? [];
    exportTrendDataToCSV({
      checkTrends,
      incidents,
      windowHours: selectedWindow.hours,
      uptimePercentage: uptimeData?.uptimePercentage ?? 0,
    });
  }, [checkTrendsData?.trends, incidents, selectedWindow.hours, uptimeData?.uptimePercentage]);

  return (
    <div className="space-y-6" data-testid="autoheal-trends-page">
      {/* Header with Time Window Selector */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <TrendingUp className="text-blue-400" size={28} />
          <div>
            <h2 className="text-xl font-semibold">Health Trends</h2>
            <p className="text-xs text-slate-400">Historical analysis and patterns</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Time Window Selector */}
          <div className="flex rounded-lg border border-white/10 overflow-hidden" data-testid="time-window-selector">
            {TIME_WINDOWS.map((window) => (
              <button
                key={window.label}
                onClick={() => setSelectedWindow(window)}
                className={`px-3 py-1.5 text-sm font-medium transition-colors ${
                  selectedWindow.label === window.label
                    ? "bg-blue-500 text-white"
                    : "bg-white/5 text-slate-400 hover:bg-white/10 hover:text-slate-200"
                }`}
                data-testid={`time-window-${window.label}`}
              >
                {window.label}
              </button>
            ))}
          </div>

          {/* Export Button */}
          <button
            onClick={handleExport}
            className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg border border-white/10 bg-white/5 text-slate-400 hover:bg-white/10 hover:text-slate-200 transition-colors"
            data-testid="export-csv-button"
          >
            <Download size={16} />
            Export CSV
          </button>
        </div>
      </div>

      {/* Uptime Overview */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-medium flex items-center gap-2">
            <Activity size={18} className="text-blue-400" />
            {selectedWindow.label} Health Trend
          </h3>
          {uptimeData && (
            <div className="text-right">
              <div className="text-2xl font-bold text-emerald-400">
                {uptimeData.uptimePercentage.toFixed(1)}%
              </div>
              <div className="text-xs text-slate-500">overall uptime</div>
            </div>
          )}
        </div>

        {uptimeLoading ? (
          <div className="h-48 flex items-center justify-center text-slate-500">
            Loading trend data...
          </div>
        ) : uptimeError ? (
          <ErrorDisplay error={uptimeError} onRetry={() => refetchUptime()} compact />
        ) : (
          <UptimeTrendChart windowHours={selectedWindow.hours} bucketCount={selectedWindow.buckets} />
        )}
      </div>

      {/* Per-Check Trends */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <h3 className="font-medium flex items-center gap-2 mb-4">
          <Activity size={18} className="text-blue-400" />
          Per-Check Health
          <span className="text-xs text-slate-500 font-normal">
            (click for details)
          </span>
        </h3>

        {trendsLoading ? (
          <div className="h-32 flex items-center justify-center text-slate-500">
            Loading check data...
          </div>
        ) : trendsError ? (
          <ErrorDisplay error={trendsError} onRetry={() => refetchTrends()} compact />
        ) : (
          <CheckTrendGrid
            trends={checkTrendsData?.trends ?? []}
            events={timelineData?.events ?? []}
            onCheckClick={handleCheckClick}
          />
        )}
      </div>

      {/* Incidents Timeline */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <h3 className="font-medium flex items-center gap-2 mb-4">
          <AlertTriangle size={18} className="text-amber-400" />
          Status Transitions
          <span className="text-xs text-slate-500 font-normal">
            ({incidents.length} in last {selectedWindow.label})
          </span>
        </h3>

        {incidentsLoading ? (
          <div className="h-32 flex items-center justify-center text-slate-500">
            Loading incidents...
          </div>
        ) : incidentsError && !incidents.length ? (
          <ErrorDisplay error={incidentsError} onRetry={() => refetchIncidents()} compact />
        ) : incidents.length === 0 ? (
          <div className="text-center py-8 text-slate-500">
            <Clock size={32} className="mx-auto mb-2 opacity-50" />
            <p>No status transitions detected</p>
            <p className="text-xs mt-1">All checks have maintained consistent status</p>
          </div>
        ) : (
          <div className="space-y-2 max-h-64 overflow-y-auto" data-testid="incidents-list">
            {incidents.slice(0, 50).map((incident, idx) => (
              <div
                key={`${incident.checkId}-${incident.timestamp}-${idx}`}
                className="flex items-start gap-3 p-3 rounded-lg bg-white/[0.02] hover:bg-white/[0.04] transition-colors cursor-pointer"
                onClick={() => handleCheckClick(incident.checkId)}
              >
                <div className="flex items-center gap-1 mt-0.5">
                  <StatusIcon status={incident.fromStatus as "ok" | "warning" | "critical"} size={12} />
                  <span className="text-slate-500">→</span>
                  <StatusIcon status={incident.toStatus as "ok" | "warning" | "critical"} size={12} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between gap-2">
                    <div className="min-w-0">
                      <span className="text-sm font-medium text-slate-200 block truncate" title={incident.checkId}>
                        {getTitle(incident.checkId)}
                      </span>
                      {getTitle(incident.checkId) !== incident.checkId && (
                        <span className="text-xs text-slate-600 font-mono">{incident.checkId}</span>
                      )}
                    </div>
                    <span className="text-xs text-slate-500 flex-shrink-0">
                      {formatRelativeTime(incident.timestamp)}
                    </span>
                  </div>
                  <p className="text-xs text-slate-400 truncate">{incident.message}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Check Detail Modal */}
      {selectedCheckId && (
        <CheckDetailModal
          checkId={selectedCheckId}
          onClose={() => setSelectedCheckId(null)}
        />
      )}
    </div>
  );
}
