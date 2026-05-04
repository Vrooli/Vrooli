// Per-check trend grid showing individual check health over time
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { memo, useMemo, useState, useCallback } from "react";
import type { KeyboardEvent } from "react";
import { TimelineEvent, HealthStatus, CheckTrend as APICheckTrend, normalizeHealthStatus } from "../../../lib/api";
import { StatusIcon, StatusSparkline } from "../../../shared/components";
import { useCheckMetadata } from "../../../shared/contexts/CheckMetadataContext";
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

// Sortable column header component
const EMPTY_STATUSES: HealthStatus[] = [];
const TREND_VISIBLE_STEP = 100;

const SortableHeader = memo(function SortableHeader({
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
        isActive ? "text-accent-primary" : "text-text-muted hover:text-text-primary"
      }`}
    >
      {label}
      <span className="w-3">
        {isActive && (currentDirection === "asc" ? <ChevronUp size={12} /> : <ChevronDown size={12} />)}
      </span>
    </button>
  );
});

interface CheckTrendRowProps {
  trend: LocalCheckTrend;
  getTitle: (checkId: string) => string;
  onCheckClick?: (checkId: string) => void;
}

const CheckTrendRow = memo(function CheckTrendRow({
  trend,
  getTitle,
  onCheckClick,
}: CheckTrendRowProps) {
  const title = getTitle(trend.checkId);
  const showCheckId = title !== trend.checkId;
  const clickable = Boolean(onCheckClick);

  const handleClick = useCallback(() => {
    onCheckClick?.(trend.checkId);
  }, [onCheckClick, trend.checkId]);

  const handleKeyDown = useCallback((e: KeyboardEvent<HTMLTableRowElement>) => {
    if (onCheckClick && (e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      onCheckClick(trend.checkId);
    }
  }, [onCheckClick, trend.checkId]);

  return (
    <tr
      onClick={clickable ? handleClick : undefined}
      className={`border-b border-border-default/30 transition-colors hover:bg-surface-overlay/30 ${
        clickable ? "cursor-pointer" : ""
      }`}
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      onKeyDown={clickable ? handleKeyDown : undefined}
    >
      <td className="py-2 pr-4">
        <div className="flex items-center gap-2">
          <StatusIcon status={trend.currentStatus} size={14} />
          <div className="min-w-0">
            <div className="truncate text-sm font-medium text-text-primary" title={trend.checkId}>
              {title}
            </div>
            {showCheckId && (
              <div className="truncate font-mono text-xs text-text-muted/80">{trend.checkId}</div>
            )}
          </div>
        </div>
      </td>

      <td className="py-2 px-2">
        <StatusSparkline statuses={trend.recentStatuses} />
      </td>

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

      <td className="py-2 px-2 text-right">
        <span className="text-sm text-emerald-400">{trend.ok}</span>
      </td>

      <td className="py-2 px-2 text-right">
        <span className={`text-sm ${trend.warning > 0 ? "text-amber-400" : "text-text-muted/60"}`}>
          {trend.warning}
        </span>
      </td>

      <td className="py-2 px-2 text-right">
        <span className={`text-sm ${trend.critical > 0 ? "text-red-400" : "text-text-muted/60"}`}>
          {trend.critical}
        </span>
      </td>

      <td className="py-2 pl-2 text-right">
        <span className="text-sm text-text-muted">{trend.total}</span>
      </td>
    </tr>
  );
});

export function CheckTrendGrid({ trends: backendTrends, events = [], onCheckClick }: CheckTrendGridProps) {
  const { getTitle } = useCheckMetadata();
  const [sortKey, setSortKey] = useState<SortKey>("uptimePercent");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [visibleCount, setVisibleCount] = useState(TREND_VISIBLE_STEP);

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
        currentStatus: normalizeHealthStatus(t.currentStatus, "ok"),
        recentStatuses: Array.isArray(t.recentStatuses)
          ? t.recentStatuses.map((status) => normalizeHealthStatus(status, "ok"))
          : EMPTY_STATUSES,
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

      let ok = 0;
      let warning = 0;
      let critical = 0;
      for (const event of sorted) {
        if (event.status === "ok") ok += 1;
        else if (event.status === "warning") warning += 1;
        else if (event.status === "critical") critical += 1;
      }
      const total = sorted.length;

      checkTrends.push({
        checkId,
        total,
        ok,
        warning,
        critical,
        uptimePercent: total > 0 ? (ok / total) * 100 : 100,
        currentStatus: normalizeHealthStatus(sorted[0]?.status, "ok"),
        recentStatuses: sorted.length > 0
          ? sorted.slice(0, 12).map((e) => normalizeHealthStatus(e.status, "ok"))
          : EMPTY_STATUSES,
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

  const visibleTrends = useMemo(() => trends.slice(0, visibleCount), [trends, visibleCount]);
  const hiddenTrendCount = Math.max(0, trends.length - visibleTrends.length);

  const showMore = useCallback(() => {
    setVisibleCount((count) => Math.min(count + TREND_VISIBLE_STEP, trends.length));
  }, [trends.length]);

  const sortAndResetVisibleCount = useCallback((key: SortKey) => {
    setVisibleCount(TREND_VISIBLE_STEP);
    handleSort(key);
  }, [handleSort]);

  if (trends.length === 0) {
    return (
      <div className="py-8 text-center text-text-muted">
        <p>No check data available yet</p>
        <p className="text-xs mt-1">Run a health check tick to see trends</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto" data-testid="autoheal-trends-check-grid">
      <table className="w-full">
        <thead>
          <tr className="border-b border-border-default/60">
            <th className="pb-2 pr-4 text-left">
              <SortableHeader
                label="Check"
                sortKey="checkId"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={sortAndResetVisibleCount}
              />
            </th>
            <th className="pb-2 px-2 text-center w-16">
              <span className="text-xs font-medium uppercase tracking-wider text-text-muted">Trend</span>
            </th>
            <th className="pb-2 px-2 text-right w-16">
              <SortableHeader
                label="Uptime"
                sortKey="uptimePercent"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={sortAndResetVisibleCount}
                align="right"
              />
            </th>
            <th className="pb-2 px-2 text-right w-14">
              <SortableHeader
                label="OK"
                sortKey="ok"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={sortAndResetVisibleCount}
                align="right"
              />
            </th>
            <th className="pb-2 px-2 text-right w-14">
              <SortableHeader
                label="Warn"
                sortKey="warning"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={sortAndResetVisibleCount}
                align="right"
              />
            </th>
            <th className="pb-2 px-2 text-right w-14">
              <SortableHeader
                label="Crit"
                sortKey="critical"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={sortAndResetVisibleCount}
                align="right"
              />
            </th>
            <th className="pb-2 pl-2 text-right w-14">
              <SortableHeader
                label="Total"
                sortKey="total"
                currentSort={sortKey}
                currentDirection={sortDirection}
                onSort={sortAndResetVisibleCount}
                align="right"
              />
            </th>
          </tr>
        </thead>
        <tbody>
          {visibleTrends.map((trend) => (
            <CheckTrendRow
              key={trend.checkId}
              trend={trend}
              getTitle={getTitle}
              onCheckClick={onCheckClick}
            />
          ))}
        </tbody>
      </table>
      {hiddenTrendCount > 0 && (
        <div className="border-t border-border-default/40 pt-3 text-center">
          <button
            type="button"
            onClick={showMore}
            className="rounded border border-border-default/70 px-3 py-1.5 text-sm text-text-secondary transition-colors hover:border-accent-primary/70 hover:text-text-primary"
          >
            Show more ({hiddenTrendCount} remaining)
          </button>
        </div>
      )}
    </div>
  );
}
