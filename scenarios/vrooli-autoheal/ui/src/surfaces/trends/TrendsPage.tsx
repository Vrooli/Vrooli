// Trends page showing comprehensive historical data
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useQuery } from "@tanstack/react-query";
import { memo, Profiler, useState, useCallback, useMemo } from "react";
import { TrendingUp, Activity, AlertTriangle, Clock, Download } from "lucide-react";
import { fetchTimeline, fetchUptimeStats, fetchIncidents, fetchCheckTrends, TimelineEvent, type CheckTrend, type Incident } from "../../lib/api";
import { CheckDetailModal, ErrorDisplay, StatusIcon } from "../../shared/components";
import { Button, Card } from "../../shared/ui/primitives";
import { UptimeTrendChart, CheckTrendGrid } from "./components";
import { exportTrendDataToCSV } from "../../lib/export";
import { useCheckMetadata } from "../../shared/contexts/CheckMetadataContext";
import { formatRelativeTime } from "../../lib/utils";
import { onProfilerRender } from "../../lib/profiler";

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
      if (!prev || !curr) {
        continue;
      }
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

// Time window options
type TimeWindow = { label: string; hours: number; buckets: number };
const TIME_WINDOWS: TimeWindow[] = [
  { label: "6h", hours: 6, buckets: 12 },
  { label: "12h", hours: 12, buckets: 24 },
  { label: "24h", hours: 24, buckets: 24 },
  { label: "7d", hours: 168, buckets: 28 },
];
const DEFAULT_TIME_WINDOW: TimeWindow = TIME_WINDOWS[2] ?? { label: "24h", hours: 24, buckets: 24 };
const EMPTY_CHECK_TRENDS: CheckTrend[] = [];
const EMPTY_TIMELINE_EVENTS: TimelineEvent[] = [];
const INCIDENT_RENDER_LIMIT = 50;

interface IncidentRowProps {
  incident: Incident;
  title: string;
  onCheckClick: (checkId: string) => void;
}

const IncidentRow = memo(function IncidentRow({
  incident,
  title,
  onCheckClick,
}: IncidentRowProps) {
  const showCheckId = title !== incident.checkId;
  const handleClick = useCallback(() => {
    onCheckClick(incident.checkId);
  }, [incident.checkId, onCheckClick]);

  return (
    <div
      className="flex cursor-pointer items-start gap-3 rounded-lg bg-surface-overlay/20 p-3 transition-colors hover:bg-surface-overlay/40"
      onClick={handleClick}
    >
      <div className="flex items-center gap-1 mt-0.5">
        <StatusIcon status={incident.fromStatus as "ok" | "warning" | "critical"} size={12} />
        <span className="text-text-muted">→</span>
        <StatusIcon status={incident.toStatus as "ok" | "warning" | "critical"} size={12} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0">
            <span className="block truncate text-sm font-medium text-text-primary" title={incident.checkId}>
              {title}
            </span>
            {showCheckId && (
              <span className="font-mono text-xs text-text-muted/80 break-all">{incident.checkId}</span>
            )}
          </div>
          <span className="shrink-0 text-xs text-text-muted">
            {formatRelativeTime(incident.timestamp)}
          </span>
        </div>
        <p className="break-words text-xs text-text-muted">{incident.message}</p>
      </div>
    </div>
  );
});

interface IncidentsListProps {
  incidents: Incident[];
  getTitle: (checkId: string) => string;
  onCheckClick: (checkId: string) => void;
}

const IncidentsList = memo(function IncidentsList({
  incidents,
  getTitle,
  onCheckClick,
}: IncidentsListProps) {
  const visibleIncidents = useMemo(
    () => incidents.slice(0, INCIDENT_RENDER_LIMIT).map((incident) => ({
      incident,
      title: getTitle(incident.checkId),
    })),
    [getTitle, incidents],
  );

  return (
    <div className="max-h-64 space-y-2 overflow-y-auto" data-testid="incidents-list">
      {visibleIncidents.map(({ incident, title }, idx) => (
        <IncidentRow
          key={`${incident.checkId}-${incident.timestamp}-${idx}`}
          incident={incident}
          title={title}
          onCheckClick={onCheckClick}
        />
      ))}
    </div>
  );
});

function TrendsPageImpl() {
  const [selectedWindow, setSelectedWindow] = useState<TimeWindow>(DEFAULT_TIME_WINDOW); // Default 24h
  const [selectedCheckId, setSelectedCheckId] = useState<string | null>(null);
  const { getTitle } = useCheckMetadata();

  // Uptime stats for overall percentage
  const { data: uptimeData, isLoading: uptimeLoading, error: uptimeError, refetch: refetchUptime } = useQuery({
    queryKey: ["uptime"],
    queryFn: fetchUptimeStats,
    refetchInterval: 60000,
    refetchIntervalInBackground: false,
  });

  // Check trends from dedicated endpoint
  const { data: checkTrendsData, isLoading: trendsLoading, error: trendsError, refetch: refetchTrends } = useQuery({
    queryKey: ["check-trends", selectedWindow.hours],
    queryFn: () => fetchCheckTrends(selectedWindow.hours),
    refetchInterval: 60000,
    refetchIntervalInBackground: false,
  });

  // Incidents from dedicated endpoint
  const { data: incidentsData, isLoading: incidentsLoading, error: incidentsError, refetch: refetchIncidents } = useQuery({
    queryKey: ["incidents", selectedWindow.hours],
    queryFn: () => fetchIncidents(selectedWindow.hours, 50),
    refetchInterval: 60000,
    refetchIntervalInBackground: false,
  });

  // Fallback to timeline for backwards compatibility
  const { data: timelineData } = useQuery({
    queryKey: ["timeline"],
    queryFn: fetchTimeline,
    refetchInterval: 30000,
    refetchIntervalInBackground: false,
    staleTime: 30000,
  });

  // Use backend incidents if available, otherwise fallback to client-side detection
  const incidents = useMemo(
    () => incidentsData?.incidents ?? (timelineData?.events ? detectIncidents(timelineData.events) : []),
    [incidentsData?.incidents, timelineData?.events]
  );
  const checkTrends = checkTrendsData?.trends ?? EMPTY_CHECK_TRENDS;
  const timelineEvents = timelineData?.events ?? EMPTY_TIMELINE_EVENTS;

  // Handle check drill-down
  const handleCheckClick = useCallback((checkId: string) => {
    setSelectedCheckId(checkId);
  }, []);

  // Handle CSV export
  const handleExport = useCallback(() => {
    exportTrendDataToCSV({
      checkTrends,
      incidents,
      windowHours: selectedWindow.hours,
      uptimePercentage: uptimeData?.uptimePercentage ?? 0,
    });
  }, [checkTrends, incidents, selectedWindow.hours, uptimeData?.uptimePercentage]);

  return (
    <div className="space-y-6" data-testid="autoheal-trends-page">
      {/* Header with Time Window Selector */}
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-3">
          <TrendingUp className="text-accent-primary" size={24} />
          <div>
            <h2 className="text-xl font-semibold">Health Trends</h2>
            <p className="text-xs text-text-muted">Historical analysis and patterns</p>
          </div>
        </div>

        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-end">
          {/* Time Window Selector */}
          <div className="flex overflow-x-auto rounded-lg border border-border-default/70" data-testid="time-window-selector">
            {TIME_WINDOWS.map((window) => (
              <button
                key={window.label}
                onClick={() => setSelectedWindow(window)}
                className={`shrink-0 px-3 py-1.5 text-sm font-medium transition-colors ${
                  selectedWindow.label === window.label
                    ? "bg-accent-primary text-text-primary"
                    : "bg-surface-overlay/40 text-text-muted hover:bg-surface-overlay/70 hover:text-text-primary"
                }`}
                data-testid={`time-window-${window.label}`}
              >
                {window.label}
              </button>
            ))}
          </div>

          {/* Export Button */}
          <Button
            onClick={handleExport}
            variant="outline"
            size="sm"
            className="justify-center"
            data-testid="export-csv-button"
          >
            <Download size={16} />
            Export CSV
          </Button>
        </div>
      </div>

      {/* Uptime Overview */}
      <Card className="p-4 sm:p-6">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h3 className="font-medium flex items-center gap-2">
            <Activity size={18} className="text-accent-primary" />
            {selectedWindow.label} Health Trend
          </h3>
          {uptimeData && (
            <div className="text-right">
              <div className="text-2xl font-bold text-accent-success">
                {uptimeData.uptimePercentage.toFixed(1)}%
              </div>
              <div className="text-xs text-text-muted">overall uptime</div>
            </div>
          )}
        </div>

        {uptimeLoading ? (
          <div className="flex h-48 items-center justify-center text-text-muted">
            Loading trend data...
          </div>
        ) : uptimeError ? (
          <ErrorDisplay error={uptimeError} onRetry={() => refetchUptime()} compact />
        ) : (
          <Profiler id="UptimeTrendChart" onRender={onProfilerRender}>
            <UptimeTrendChart windowHours={selectedWindow.hours} bucketCount={selectedWindow.buckets} />
          </Profiler>
        )}
      </Card>

      {/* Per-Check Trends */}
      <Card className="p-4 sm:p-6">
        <h3 className="font-medium flex items-center gap-2 mb-4">
          <Activity size={18} className="text-accent-primary" />
          Per-Check Health
          <span className="text-xs font-normal text-text-muted">
            (click for details)
          </span>
        </h3>

        {trendsLoading ? (
          <div className="flex h-32 items-center justify-center text-text-muted">
            Loading check data...
          </div>
        ) : trendsError ? (
          <ErrorDisplay error={trendsError} onRetry={() => refetchTrends()} compact />
        ) : (
          <Profiler id="CheckTrendGrid" onRender={onProfilerRender}>
            <CheckTrendGrid
              trends={checkTrends}
              events={timelineEvents}
              onCheckClick={handleCheckClick}
            />
          </Profiler>
        )}
      </Card>

      {/* Incidents Timeline */}
      <Card className="p-4 sm:p-6">
        <h3 className="font-medium flex items-center gap-2 mb-4">
          <AlertTriangle size={18} className="text-accent-warning" />
          Status Transitions
          <span className="text-xs font-normal text-text-muted">
            ({incidents.length} in last {selectedWindow.label})
          </span>
        </h3>

        {incidentsLoading ? (
          <div className="flex h-32 items-center justify-center text-text-muted">
            Loading incidents...
          </div>
        ) : incidentsError && !incidents.length ? (
          <ErrorDisplay error={incidentsError} onRetry={() => refetchIncidents()} compact />
        ) : incidents.length === 0 ? (
          <div className="py-8 text-center text-text-muted">
            <Clock size={32} className="mx-auto mb-2 opacity-50" />
            <p>No status transitions detected</p>
            <p className="text-xs mt-1">All checks have maintained consistent status</p>
          </div>
        ) : (
          <IncidentsList
            incidents={incidents}
            getTitle={getTitle}
            onCheckClick={handleCheckClick}
          />
        )}
      </Card>

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

export function TrendsPage() {
  return (
    <Profiler id="TrendsPage" onRender={onProfilerRender}>
      <TrendsPageImpl />
    </Profiler>
  );
}
