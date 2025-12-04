// Per-check trend grid showing individual check health over time
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useMemo, useState, useCallback } from "react";
import { TimelineEvent, HealthStatus, CheckTrend as APICheckTrend } from "../lib/api";
import { StatusIcon } from "./StatusIcon";
import { useCheckMetadata } from "../contexts/CheckMetadataContext";
import { ChevronUp, ChevronDown } from "lucide-react";

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

// Sortable column keys
type SortKey = "checkId" | "total" | "ok" | "warning" | "critical" | "uptimePercent";
type SortDirection = "asc" | "desc";

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

// Sortable column header component
function SortableHeader({
  label,
  sortKey,
  currentSort,
  currentDirection,
  onSort,
  align = "left",
}: {
  label: string;
  sortKey: SortKey;
  currentSort: SortKey;
  currentDirection: SortDirection;
  onSort: (key: SortKey) => void;
  align?: "left" | "right" | "center";
}) {
  const isActive = currentSort === sortKey;
  const alignClass = align === "right" ? "justify-end" : align === "center" ? "justify-center" : "justify-start";

  return (
    <button
      onClick={() => onSort(sortKey)}
      className={`flex items-center gap-1 text-xs font-medium uppercase tracking-wider transition-colors ${alignClass} ${
        isActive ? "text-blue-400" : "text-slate-500 hover:text-slate-300"
      }`}
    >
      {label}
      <span className="w-3">
        {isActive && (currentDirection === "asc" ? <ChevronUp size={12} /> : <ChevronDown size={12} />)}
      </span>
    </button>
  );
}

export function CheckTrendGrid({ trends: backendTrends, events = [], onCheckClick }: CheckTrendGridProps) {
  const { getTitle } = useCheckMetadata();
  const [sortKey, setSortKey] = useState<SortKey>("uptimePercent");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  // Handle sort column click
  const handleSort = useCallback((key: SortKey) => {
    if (key === sortKey) {
      // Toggle direction if same column
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      // New column - default direction based on column type
      setSortKey(key);
      // For uptime and ok, ascending means worst first (lower is worse)
      // For warning/critical, descending means worst first (higher is worse)
      setSortDirection(key === "warning" || key === "critical" ? "desc" : "asc");
    }
  }, [sortKey]);

  // Use backend trends if available, otherwise fall back to client-side aggregation
  const baseTrends = useMemo<LocalCheckTrend[]>(() => {
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

    return checkTrends;
  }, [backendTrends, events]);

  // Sort trends based on current sort settings
  const trends = useMemo(() => {
    return [...baseTrends].sort((a, b) => {
      let comparison = 0;

      switch (sortKey) {
        case "checkId":
          comparison = a.checkId.localeCompare(b.checkId);
          break;
        case "total":
          comparison = a.total - b.total;
          break;
        case "ok":
          comparison = a.ok - b.ok;
          break;
        case "warning":
          comparison = a.warning - b.warning;
          break;
        case "critical":
          comparison = a.critical - b.critical;
          break;
        case "uptimePercent":
          comparison = a.uptimePercent - b.uptimePercent;
          break;
      }

      return sortDirection === "asc" ? comparison : -comparison;
    });
  }, [baseTrends, sortKey, sortDirection]);

  if (trends.length === 0) {
    return (
      <div className="text-center py-8 text-slate-500">
        <p>No check data available yet</p>
        <p className="text-xs mt-1">Run a health check tick to see trends</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto" data-testid="autoheal-trends-check-grid">
      <table className="w-full">
        <thead>
          <tr className="border-b border-white/10">
            <th className="pb-2 pr-4 text-left">
              <SortableHeader
                label="Check"
                sortKey="checkId"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={handleSort}
              />
            </th>
            <th className="pb-2 px-2 text-center w-16">
              <span className="text-xs font-medium uppercase tracking-wider text-slate-500">Trend</span>
            </th>
            <th className="pb-2 px-2 text-right w-16">
              <SortableHeader
                label="Uptime"
                sortKey="uptimePercent"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={handleSort}
                align="right"
              />
            </th>
            <th className="pb-2 px-2 text-right w-14">
              <SortableHeader
                label="OK"
                sortKey="ok"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={handleSort}
                align="right"
              />
            </th>
            <th className="pb-2 px-2 text-right w-14">
              <SortableHeader
                label="Warn"
                sortKey="warning"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={handleSort}
                align="right"
              />
            </th>
            <th className="pb-2 px-2 text-right w-14">
              <SortableHeader
                label="Crit"
                sortKey="critical"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={handleSort}
                align="right"
              />
            </th>
            <th className="pb-2 pl-2 text-right w-14">
              <SortableHeader
                label="Total"
                sortKey="total"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={handleSort}
                align="right"
              />
            </th>
          </tr>
        </thead>
        <tbody>
          {trends.map((trend) => (
            <tr
              key={trend.checkId}
              onClick={() => onCheckClick?.(trend.checkId)}
              className={`border-b border-white/5 hover:bg-white/[0.02] transition-colors ${
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
              {/* Check name with status icon */}
              <td className="py-2 pr-4">
                <div className="flex items-center gap-2">
                  <StatusIcon status={trend.currentStatus} size={14} />
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-slate-200 truncate" title={trend.checkId}>
                      {getTitle(trend.checkId)}
                    </div>
                    {getTitle(trend.checkId) !== trend.checkId && (
                      <div className="text-xs text-slate-600 font-mono truncate">{trend.checkId}</div>
                    )}
                  </div>
                </div>
              </td>

              {/* Sparkline */}
              <td className="py-2 px-2">
                <StatusSparkline statuses={trend.recentStatuses} />
              </td>

              {/* Uptime percentage */}
              <td className="py-2 px-2 text-right">
                <span
                  className={`text-sm font-medium ${
                    trend.uptimePercent >= 99
                      ? "text-emerald-400"
                      : trend.uptimePercent >= 90
                      ? "text-amber-400"
                      : "text-red-400"
                  }`}
                >
                  {trend.uptimePercent.toFixed(0)}%
                </span>
              </td>

              {/* OK count */}
              <td className="py-2 px-2 text-right">
                <span className="text-sm text-emerald-400">{trend.ok}</span>
              </td>

              {/* Warning count */}
              <td className="py-2 px-2 text-right">
                <span className={`text-sm ${trend.warning > 0 ? "text-amber-400" : "text-slate-600"}`}>
                  {trend.warning}
                </span>
              </td>

              {/* Critical count */}
              <td className="py-2 px-2 text-right">
                <span className={`text-sm ${trend.critical > 0 ? "text-red-400" : "text-slate-600"}`}>
                  {trend.critical}
                </span>
              </td>

              {/* Total count */}
              <td className="py-2 pl-2 text-right">
                <span className="text-sm text-slate-400">{trend.total}</span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
