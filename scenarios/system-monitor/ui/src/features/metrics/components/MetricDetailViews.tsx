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
import type { LegendPayload } from 'recharts';
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

export interface MetricLineChartProps {
  data: Array<{ timestamp: string } & Record<string, number | string>>;
  lines: MetricLineChartLineConfig[];
  unit?: string;
  height?: number;
  yDomain?: [number, number] | ['auto', 'auto'];
  valueFormatter?: (value: number) => string;
  className?: string;
  style?: CSSProperties;
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
      style={{
        background: 'var(--chart-tooltip-bg)',
        border: '1px solid var(--chart-tooltip-border)',
        borderRadius: 'var(--radius-md)',
        padding: '8px 12px',
        boxShadow: 'var(--chart-tooltip-shadow)',
        minWidth: 140
      }}
    >
      <div
        style={{
          color: 'var(--color-text-heading)',
          fontWeight: 600,
          fontSize: 'var(--text-xs)',
          marginBottom: 6,
          borderBottom: '1px solid var(--color-border)',
          paddingBottom: 4
        }}
      >
        {formattedLabel}
      </div>
      {visiblePayload.map(entry => (
        <div
          key={entry.dataKey}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            fontSize: 'var(--text-xs)',
            color: 'var(--color-text)',
            padding: '2px 0'
          }}
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
          <span style={{ color: 'var(--color-text-secondary)', flex: 1 }}>{entry.name}</span>
          <span style={{ fontWeight: 600, fontFamily: 'var(--font-mono)' }}>
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

function getLegendDataKey(entry: LegendPayload | undefined): string | undefined {
  if (typeof entry?.dataKey === 'string') {
    return entry.dataKey;
  }
  return undefined;
}

// ── MetricLineChart ─────────────────────────────────────────────────────────

export const MetricLineChart = ({
  data,
  lines,
  unit,
  height = 320,
  yDomain = ['auto', 'auto'],
  valueFormatter,
  className,
  style
}: MetricLineChartProps) => {
  const [hiddenSeries, setHiddenSeries] = useState<Set<string>>(new Set());

  const gradientIds = useMemo(
    () => Object.fromEntries(lines.map(line => [line.dataKey, `grad-${line.dataKey}-${Math.random().toString(36).slice(2, 6)}`])),
    [lines]
  );

  const handleLegendClick = useCallback((data: LegendPayload) => {
    const key = getLegendDataKey(data);
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
              wrapperStyle={{ color: 'var(--color-text)', cursor: 'pointer', fontSize: 'var(--text-xs)' }}
              onClick={handleLegendClick}
              formatter={(value, entry) => {
                const dataKey = getLegendDataKey(entry) ?? '';
                const isHidden = hiddenSeries.has(dataKey);
                return (
                  <span style={{ opacity: isHidden ? 0.35 : 1, textDecoration: isHidden ? 'line-through' : 'none' }}>
                    {value}
                  </span>
                );
              }}
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
      ) : (
        <div style={{
          textAlign: 'center',
          padding: 'var(--spacing-xl)'
        }} className="text-muted">
          Waiting for timeseries data...
        </div>
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
