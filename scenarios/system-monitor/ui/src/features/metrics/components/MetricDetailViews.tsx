import { useState, useMemo, useCallback } from 'react';
import type { ReactNode, CSSProperties } from 'react';
import {
  ResponsiveContainer,
  ComposedChart,
  Line,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend
} from 'recharts';
import { ArrowLeft } from 'lucide-react';

import { formatTimeLabel, formatChartTime } from '../../../shared/utils/formatters';

// ── Interfaces ──────────────────────────────────────────────────────────────

export interface MetricDetailLayoutProps {
  title: string;
  icon: ReactNode;
  headline: string;
  subhead?: string;
  onBack: () => void;
  children: ReactNode;
}

interface MetricLineChartLineConfig {
  dataKey: string;
  name: string;
  color: string;
  strokeWidth?: number;
  type?: 'linear' | 'monotone' | 'natural' | 'stepAfter' | 'stepBefore';
}

/**
 * Why a chart has three empty states, not one.
 *
 * `loading` (history not fetched yet), `empty` (fetched, nothing in the
 * window) and `error` (the series could not be loaded) look identical to a
 * user if they render the same sentence — which is how a hard backend failure
 * can sit unnoticed behind what reads as a slow spinner. They are kept
 * distinct here deliberately.
 */
export type MetricChartStatus = 'loading' | 'ready' | 'error';

export interface MetricLineChartProps {
  data: Array<{ timestamp: string } & Record<string, number | string>>;
  lines: MetricLineChartLineConfig[];
  unit?: string;
  height?: number;
  yDomain?: [number, number] | ['auto', 'auto'];
  valueFormatter?: (value: number) => string;
  className?: string;
  style?: CSSProperties;
  /** Defaults to 'ready' so existing call sites keep their behaviour. */
  status?: MetricChartStatus;
  /** Shown under the error title when status is 'error'. */
  errorMessage?: string;
  /** Label used in the empty state, e.g. "memory". */
  seriesLabel?: string;
}

// ── Shared Components ───────────────────────────────────────────────────────

export const MetricDetailLayout = ({ title, icon, headline, subhead, onBack, children }: MetricDetailLayoutProps) => (
  <div className="flex-col-gap-lg">
    <div className="metric-detail-toolbar">
      <button
        type="button"
        className="btn btn-action metric-detail-back-btn"
        onClick={onBack}
      >
        <ArrowLeft size={16} />
        Back To Dashboard
      </button>
      <div className="icon-text gap-md">
        <div className="metric-detail-title">
          {icon}
          <span>{title}</span>
        </div>
        <div className="metric-detail-headline">{headline}</div>
      </div>
    </div>
    {subhead && (
      <div className="card-subtitle">
        {subhead}
      </div>
    )}
    {children}
  </div>
);

// ── Private Helpers ─────────────────────────────────────────────────────────

const defaultValueFormatter = (unit?: string) => (value: number) => {
  if (!Number.isFinite(value)) {
    return '—';
  }
  if (unit) {
    return `${value.toFixed(2)}${unit}`;
  }
  return value.toFixed(2);
};

// ── Custom Tooltip ──────────────────────────────────────────────────────────

interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{
    value: number;
    name: string;
    color: string;
    dataKey: string;
  }>;
  label?: string;
  formatter: (value: number) => string;
  hiddenSeries: Set<string>;
}

const ChartTooltip = ({ active, payload, label, formatter, hiddenSeries }: CustomTooltipProps) => {
  if (!active || !payload || payload.length === 0) return null;

  const visiblePayload = payload.filter(entry => !hiddenSeries.has(entry.dataKey));
  if (visiblePayload.length === 0) return null;
  const formattedLabel = typeof label === 'string' ? formatTimeLabel(label) : '—';

  return (
    <div
      data-sm-style="sm-style-674a98adf0"
    >
      <div
        data-sm-style="sm-style-f2d36b52d9"
      >
        {formattedLabel}
      </div>
      {visiblePayload.map(entry => (
        <div
          key={entry.dataKey}
          data-sm-style="sm-style-3a3c6a968c"
        >
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: entry.color,
              flexShrink: 0
            }}
          />
          <span data-sm-style="sm-style-2f52d20bb2">{entry.name}</span>
          <span data-sm-style="sm-style-dee1e2e177">
            {Number.isFinite(entry.value) ? formatter(entry.value) : '—'}
          </span>
        </div>
      ))}
    </div>
  );
};

// ── Custom Cursor ───────────────────────────────────────────────────────────

const ChartCursor = (props: { points?: Array<{ x: number }>; height?: number }) => {
  const { points, height } = props;
  if (!points || points.length === 0) return null;
  const firstPoint = points[0];
  if (!firstPoint) return null;
  const x = firstPoint.x;
  return (
    <line
      x1={x}
      x2={x}
      y1={0}
      y2={height ?? 0}
      stroke="var(--chart-cursor-color)"
      strokeWidth={1}
      strokeDasharray="4 3"
    />
  );
};

// ── Chart Placeholders ──────────────────────────────────────────────────────

const ChartSkeleton = ({ height }: { height: number }) => (
  <div
    className="chart-placeholder"
    style={{ height }}
    role="status"
    aria-live="polite"
    aria-label="Loading chart data"
  >
    <div className="chart-skeleton__frame" aria-hidden="true">
      <div className="chart-skeleton__axis-y">
        {[0, 1, 2, 3, 4].map(tick => (
          <span key={tick} className="loading-skeleton__block chart-skeleton__tick" />
        ))}
      </div>
      <div className="chart-skeleton__plot">
        <div className="chart-skeleton__wave" />
      </div>
      <div className="chart-skeleton__axis-x">
        {[0, 1, 2, 3, 4, 5].map(tick => (
          <span key={tick} className="loading-skeleton__block chart-skeleton__tick" />
        ))}
      </div>
    </div>
    <div className="chart-placeholder__caption">Loading timeseries…</div>
  </div>
);

const ChartEmptyState = ({ height, seriesLabel }: { height: number; seriesLabel?: string }) => (
  <div className="chart-placeholder chart-placeholder--empty" style={{ height }} role="status">
    <div className="chart-placeholder__title">No data in this window</div>
    <div className="chart-placeholder__caption">
      {seriesLabel
        ? `No ${seriesLabel} samples were recorded for the selected time range.`
        : 'No samples were recorded for the selected time range.'}
    </div>
  </div>
);

const ChartErrorState = ({ height, errorMessage }: { height: number; errorMessage?: string }) => (
  <div className="chart-placeholder chart-placeholder--error" style={{ height }} role="alert">
    <div className="chart-placeholder__title">Timeseries unavailable</div>
    <div className="chart-placeholder__caption">
      {errorMessage ?? 'The metrics history could not be loaded.'}
    </div>
  </div>
);

// ── MetricLineChart ─────────────────────────────────────────────────────────

export const MetricLineChart = ({
  data,
  lines,
  unit,
  height = 320,
  yDomain = ['auto', 'auto'],
  valueFormatter,
  className,
  style,
  status = 'ready',
  errorMessage,
  seriesLabel
}: MetricLineChartProps) => {
  const [hiddenSeries, setHiddenSeries] = useState<Set<string>>(new Set());

  const gradientIds = useMemo(
    () => Object.fromEntries(lines.map(line => [line.dataKey, `grad-${line.dataKey}-${Math.random().toString(36).slice(2, 6)}`])),
    [lines]
  );

  const toggleSeries = useCallback((key: string) => {
    if (!key) return;
    setHiddenSeries(prev => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        if (next.size < lines.length - 1) {
          next.add(key);
        }
      }
      return next;
    });
  }, [lines.length]);

  const fmtr = valueFormatter ?? defaultValueFormatter(unit);

  return (
    <div className={className} style={{ width: '100%', ...(style ?? {}) }}>
      {data.length > 0 ? (
        <ResponsiveContainer width="100%" height={height}>
          <ComposedChart data={data} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
            <defs>
              {lines.map(line => (
                <linearGradient key={line.dataKey} id={gradientIds[line.dataKey]} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={line.color} stopOpacity={0.25} />
                  <stop offset="100%" stopColor={line.color} stopOpacity={0} />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid stroke="var(--chart-grid)" strokeDasharray="4 4" />
            <XAxis
              dataKey="timestamp"
              stroke="var(--color-text-secondary)"
              tickFormatter={formatChartTime}
              minTickGap={20}
              fontSize={11}
            />
            <YAxis
              stroke="var(--color-text-secondary)"
              domain={yDomain}
              tickFormatter={fmtr}
              fontSize={11}
            />
            <Tooltip
              content={<ChartTooltip formatter={fmtr} hiddenSeries={hiddenSeries} />}
              cursor={<ChartCursor />}
            />
            <Legend
              wrapperStyle={{ color: 'var(--color-text)', fontSize: 'var(--text-xs)' }}
              // Rendered from `lines`, not from the chart's graphical items.
              // Each series contributes a Line *and* a decorative Area, so an
              // inferred legend lists every series twice.
              content={() => (
                <ul className="metric-chart-legend">
                  {lines.map(line => {
                    const isHidden = hiddenSeries.has(line.dataKey);
                    return (
                      <li key={line.dataKey}>
                        <button
                          type="button"
                          className="metric-chart-legend__item"
                          aria-pressed={!isHidden}
                          onClick={() => toggleSeries(line.dataKey)}
                          style={{ opacity: isHidden ? 0.35 : 1 }}
                        >
                          <span
                            className="metric-chart-legend__swatch"
                            style={{ background: line.color }}
                            aria-hidden="true"
                          />
                          <span style={{ textDecoration: isHidden ? 'line-through' : 'none' }}>
                            {line.name}
                          </span>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
            />
            {lines.map(line => {
              const isHidden = hiddenSeries.has(line.dataKey);
              return (
                <Area
                  key={`area-${line.dataKey}`}
                  type={line.type ?? 'monotone'}
                  dataKey={line.dataKey}
                  fill={`url(#${gradientIds[line.dataKey]})`}
                  stroke="none"
                  isAnimationActive={false}
                  hide={isHidden}
                  // The gradient fill is decoration for the line that shares
                  // its dataKey. Without this it registers a second, unnamed
                  // legend entry per series and doubles the legend.
                  legendType="none"
                />
              );
            })}
            {lines.map(line => {
              const isHidden = hiddenSeries.has(line.dataKey);
              return (
                <Line
                  key={line.dataKey}
                  type={line.type ?? 'monotone'}
                  dataKey={line.dataKey}
                  name={line.name}
                  stroke={line.color}
                  strokeWidth={line.strokeWidth ?? 2}
                  dot={false}
                  activeDot={{
                    r: 4,
                    strokeWidth: 2,
                    stroke: 'var(--chart-dot-stroke)',
                    fill: line.color
                  }}
                  isAnimationActive={false}
                  hide={isHidden}
                />
              );
            })}
          </ComposedChart>
        </ResponsiveContainer>
      ) : status === 'error' ? (
        <ChartErrorState height={height} errorMessage={errorMessage} />
      ) : status === 'loading' ? (
        <ChartSkeleton height={height} />
      ) : (
        <ChartEmptyState height={height} seriesLabel={seriesLabel} />
      )}
    </div>
  );
};

// ── Barrel Re-exports ───────────────────────────────────────────────────────

export { CpuDetailView } from './CpuDetailView';
export type { CpuDetailViewProps } from './CpuDetailView';
export { MemoryDetailView } from './MemoryDetailView';
export type { MemoryDetailViewProps } from './MemoryDetailView';
export { NetworkDetailView } from './NetworkDetailView';
export type { NetworkDetailViewProps } from './NetworkDetailView';
export { DiskDetailView } from './DiskDetailView';
export type { DiskDetailViewProps } from './DiskDetailView';
export { GpuDetailView } from './GpuDetailView';
export type { GpuDetailViewProps } from './GpuDetailView';
