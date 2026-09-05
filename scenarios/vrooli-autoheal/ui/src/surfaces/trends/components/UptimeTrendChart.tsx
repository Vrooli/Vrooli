// Uptime trend area chart showing status distribution over time
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useQuery } from "@tanstack/react-query";
import { memo, useMemo } from "react";
import { fetchUptimeHistory, UptimeHistoryBucket } from "../../../lib/api";
import { themeColors } from "../../../shared/theme/colors";
import { ErrorDisplay } from "../../../shared/components";

interface ChartDataPoint {
  time: string;
  timestamp: Date;
  ok: number;
  warning: number;
  critical: number;
  total: number;
}

interface ChartPoint extends ChartDataPoint {
  x: number;
  okTop: number;
  okBottom: number;
  warningTop: number;
  warningBottom: number;
  criticalTop: number;
  criticalBottom: number;
}

function formatTimeLabel(date: Date): string {
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

interface UptimeTrendChartProps {
  windowHours?: number;
  bucketCount?: number;
}

const VIEWBOX_WIDTH = 640;
const VIEWBOX_HEIGHT = 192;
const PLOT = { top: 10, right: 12, bottom: 26, left: 36 };
const PLOT_WIDTH = VIEWBOX_WIDTH - PLOT.left - PLOT.right;
const PLOT_HEIGHT = VIEWBOX_HEIGHT - PLOT.top - PLOT.bottom;

function buildAreaPath(points: ChartPoint[], topKey: keyof ChartPoint, bottomKey: keyof ChartPoint): string {
  if (points.length === 0) return "";

  const top = points.map((point) => `${point.x},${point[topKey]}`);
  const bottom = [...points].reverse().map((point) => `${point.x},${point[bottomKey]}`);
  return `M ${top.join(" L ")} L ${bottom.join(" L ")} Z`;
}

function buildLinePath(points: ChartPoint[], key: keyof ChartPoint): string {
  if (points.length === 0) return "";
  return `M ${points.map((point) => `${point.x},${point[key]}`).join(" L ")}`;
}

function UptimeTrendChartImpl({ windowHours = 24, bucketCount = 24 }: UptimeTrendChartProps) {
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

  const chartModel = useMemo(() => {
    const maxTotal = Math.max(1, ...chartData.map((point) => point.total));
    const y = (value: number) => PLOT.top + PLOT_HEIGHT - (value / maxTotal) * PLOT_HEIGHT;

    const points: ChartPoint[] = chartData.map((point, index) => {
      const x = PLOT.left + (chartData.length === 1 ? PLOT_WIDTH / 2 : (index / (chartData.length - 1)) * PLOT_WIDTH);
      const critical = point.critical;
      const warning = point.critical + point.warning;
      const ok = warning + point.ok;

      return {
        ...point,
        x,
        criticalBottom: y(0),
        criticalTop: y(critical),
        warningBottom: y(critical),
        warningTop: y(warning),
        okBottom: y(warning),
        okTop: y(ok),
      };
    });

    const labelIndexes = points.length <= 3
      ? points.map((_, index) => index)
      : [0, Math.floor((points.length - 1) / 2), points.length - 1];

    const labels: ChartPoint[] = [];
    for (const index of labelIndexes) {
      const point = points[index];
      if (point) labels.push(point);
    }

    return {
      points,
      maxTotal,
      labels,
      okArea: buildAreaPath(points, "okTop", "okBottom"),
      warningArea: buildAreaPath(points, "warningTop", "warningBottom"),
      criticalArea: buildAreaPath(points, "criticalTop", "criticalBottom"),
      okLine: buildLinePath(points, "okTop"),
      warningLine: buildLinePath(points, "warningTop"),
      criticalLine: buildLinePath(points, "criticalTop"),
    };
  }, [chartData]);

  if (isLoading) {
    return (
      <div className="flex h-48 items-center justify-center text-text-muted">
        Loading trend data...
      </div>
    );
  }

  if (error) {
    return <ErrorDisplay error={error} onRetry={() => refetch()} compact />;
  }

  if (chartData.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-text-muted">
        <div className="text-center">
          <p>No historical data available yet</p>
          <p className="text-xs mt-1">Data will appear after health checks run</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-48" data-testid="autoheal-trends-chart">
      <svg
        className="h-full w-full"
        viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
        preserveAspectRatio="none"
        role="img"
        aria-label={`Uptime status trend over the last ${windowHours} hours`}
      >
        <defs>
          <linearGradient id="autohealChartOk" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={themeColors.chart.ok} stopOpacity={0.75} />
            <stop offset="95%" stopColor={themeColors.chart.ok} stopOpacity={0.12} />
          </linearGradient>
          <linearGradient id="autohealChartWarning" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={themeColors.chart.warning} stopOpacity={0.75} />
            <stop offset="95%" stopColor={themeColors.chart.warning} stopOpacity={0.12} />
          </linearGradient>
          <linearGradient id="autohealChartCritical" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={themeColors.chart.critical} stopOpacity={0.75} />
            <stop offset="95%" stopColor={themeColors.chart.critical} stopOpacity={0.12} />
          </linearGradient>
        </defs>

        {[0, 0.5, 1].map((ratio) => {
          const y = PLOT.top + PLOT_HEIGHT - ratio * PLOT_HEIGHT;
          return (
            <line
              key={ratio}
              x1={PLOT.left}
              x2={VIEWBOX_WIDTH - PLOT.right}
              y1={y}
              y2={y}
              stroke={themeColors.chart.grid}
              strokeDasharray="3 3"
              opacity={0.28}
            />
          );
        })}
        <line
          x1={PLOT.left}
          x2={PLOT.left}
          y1={PLOT.top}
          y2={VIEWBOX_HEIGHT - PLOT.bottom}
          stroke={themeColors.chart.axis}
          opacity={0.8}
        />
        <line
          x1={PLOT.left}
          x2={VIEWBOX_WIDTH - PLOT.right}
          y1={VIEWBOX_HEIGHT - PLOT.bottom}
          y2={VIEWBOX_HEIGHT - PLOT.bottom}
          stroke={themeColors.chart.axis}
          opacity={0.8}
        />

        <path d={chartModel.criticalArea} fill="url(#autohealChartCritical)" />
        <path d={chartModel.warningArea} fill="url(#autohealChartWarning)" />
        <path d={chartModel.okArea} fill="url(#autohealChartOk)" />
        <path d={chartModel.criticalLine} fill="none" stroke={themeColors.chart.critical} strokeWidth="2" vectorEffect="non-scaling-stroke" />
        <path d={chartModel.warningLine} fill="none" stroke={themeColors.chart.warning} strokeWidth="2" vectorEffect="non-scaling-stroke" />
        <path d={chartModel.okLine} fill="none" stroke={themeColors.chart.ok} strokeWidth="2" vectorEffect="non-scaling-stroke" />

        <text x={PLOT.left - 8} y={PLOT.top + 4} textAnchor="end" fill={themeColors.chart.tick} fontSize="10">
          {chartModel.maxTotal}
        </text>
        <text x={PLOT.left - 8} y={VIEWBOX_HEIGHT - PLOT.bottom + 4} textAnchor="end" fill={themeColors.chart.tick} fontSize="10">
          0
        </text>
        {chartModel.labels.map((point) => (
          <text
            key={`${point.timestamp.toISOString()}-${point.x}`}
            x={point.x}
            y={VIEWBOX_HEIGHT - 6}
            textAnchor="middle"
            fill={themeColors.chart.tick}
            fontSize="10"
          >
            {point.time}
          </text>
        ))}
      </svg>
    </div>
  );
}

export const UptimeTrendChart = memo(UptimeTrendChartImpl);
