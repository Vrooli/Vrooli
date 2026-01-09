// Uptime trend area chart showing status distribution over time
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import { fetchUptimeHistory, UptimeHistoryBucket } from "../lib/api";
import { ErrorDisplay } from "./ErrorDisplay";

interface ChartDataPoint {
  time: string;
  timestamp: Date;
  ok: number;
  warning: number;
  critical: number;
  total: number;
}

function formatTimeLabel(date: Date): string {
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatTooltipTime(date: Date): string {
  return date.toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

interface TooltipPayloadItem {
  name: string;
  value: number;
  color: string;
  payload?: ChartDataPoint;
}

interface CustomTooltipProps {
  active?: boolean;
  payload?: TooltipPayloadItem[];
  label?: string;
}

function CustomTooltip({ active, payload }: CustomTooltipProps) {
  if (!active || !payload || !payload.length) return null;

  const dataPoint = payload[0]?.payload;
  if (!dataPoint) return null;

  const total = dataPoint.total || (dataPoint.ok + dataPoint.warning + dataPoint.critical);
  const uptimePercent = total > 0 ? ((dataPoint.ok / total) * 100).toFixed(1) : "100.0";

  return (
    <div className="bg-slate-900 border border-white/20 rounded-lg p-3 shadow-xl">
      <p className="text-xs text-slate-400 mb-2">
        {formatTooltipTime(dataPoint.timestamp)}
      </p>
      <div className="space-y-1 text-sm">
        <div className="flex items-center justify-between gap-4">
          <span className="text-emerald-400">OK</span>
          <span className="font-medium">{dataPoint.ok}</span>
        </div>
        <div className="flex items-center justify-between gap-4">
          <span className="text-amber-400">Warning</span>
          <span className="font-medium">{dataPoint.warning}</span>
        </div>
        <div className="flex items-center justify-between gap-4">
          <span className="text-red-400">Critical</span>
          <span className="font-medium">{dataPoint.critical}</span>
        </div>
        <div className="border-t border-white/10 pt-1 mt-1">
          <div className="flex items-center justify-between gap-4">
            <span className="text-slate-400">Uptime</span>
            <span className="font-medium text-emerald-400">{uptimePercent}%</span>
          </div>
        </div>
      </div>
    </div>
  );
}

interface UptimeTrendChartProps {
  windowHours?: number;
  bucketCount?: number;
}

export function UptimeTrendChart({ windowHours = 24, bucketCount = 24 }: UptimeTrendChartProps) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["uptime-history", windowHours, bucketCount],
    queryFn: () => fetchUptimeHistory(windowHours, bucketCount),
    refetchInterval: 60000,
    staleTime: 30000,
  });

  const chartData = useMemo<ChartDataPoint[]>(() => {
    if (!data?.buckets) return [];

    return data.buckets.map((bucket: UptimeHistoryBucket) => ({
      time: formatTimeLabel(new Date(bucket.timestamp)),
      timestamp: new Date(bucket.timestamp),
      ok: bucket.ok,
      warning: bucket.warning,
      critical: bucket.critical,
      total: bucket.total,
    }));
  }, [data?.buckets]);

  if (isLoading) {
    return (
      <div className="h-48 flex items-center justify-center text-slate-500">
        Loading trend data...
      </div>
    );
  }

  if (error) {
    return <ErrorDisplay error={error} onRetry={() => refetch()} compact />;
  }

  if (chartData.length === 0) {
    return (
      <div className="h-48 flex items-center justify-center text-slate-500">
        <div className="text-center">
          <p>No historical data available yet</p>
          <p className="text-xs mt-1">Data will appear after health checks run</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-48" data-testid="autoheal-trends-chart">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="colorOk" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#10b981" stopOpacity={0.8} />
              <stop offset="95%" stopColor="#10b981" stopOpacity={0.1} />
            </linearGradient>
            <linearGradient id="colorWarning" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.8} />
              <stop offset="95%" stopColor="#f59e0b" stopOpacity={0.1} />
            </linearGradient>
            <linearGradient id="colorCritical" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#ef4444" stopOpacity={0.8} />
              <stop offset="95%" stopColor="#ef4444" stopOpacity={0.1} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#334155" opacity={0.3} />
          <XAxis
            dataKey="time"
            tick={{ fill: "#94a3b8", fontSize: 10 }}
            tickLine={{ stroke: "#475569" }}
            axisLine={{ stroke: "#475569" }}
            interval="preserveStartEnd"
            minTickGap={30}
          />
          <YAxis
            tick={{ fill: "#94a3b8", fontSize: 10 }}
            tickLine={{ stroke: "#475569" }}
            axisLine={{ stroke: "#475569" }}
            width={30}
          />
          <Tooltip content={<CustomTooltip />} />
          <Area
            type="monotone"
            dataKey="critical"
            stackId="1"
            stroke="#ef4444"
            fill="url(#colorCritical)"
            strokeWidth={2}
          />
          <Area
            type="monotone"
            dataKey="warning"
            stackId="1"
            stroke="#f59e0b"
            fill="url(#colorWarning)"
            strokeWidth={2}
          />
          <Area
            type="monotone"
            dataKey="ok"
            stackId="1"
            stroke="#10b981"
            fill="url(#colorOk)"
            strokeWidth={2}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
