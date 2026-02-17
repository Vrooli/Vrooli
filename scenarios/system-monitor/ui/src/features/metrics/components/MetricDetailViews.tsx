import type { ReactNode, CSSProperties } from 'react';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend
} from 'recharts';
import { ArrowLeft } from 'lucide-react';

import { formatTimeLabel } from './metricHelpers';

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
  <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-lg)' }}>
    <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-lg)', flexWrap: 'wrap' }}>
      <button
        type="button"
        className="btn btn-action"
        onClick={onBack}
        style={{ textTransform: 'uppercase', letterSpacing: '0.12em', fontSize: 'var(--font-size-xs)' }}
      >
        <ArrowLeft size={16} />
        Back To Dashboard
      </button>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-md)' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--spacing-sm)',
            color: 'var(--color-text-bright)',
            textTransform: 'uppercase',
            letterSpacing: '0.12em'
          }}
        >
          {icon}
          <span>{title}</span>
        </div>
        <div style={{ fontSize: 'var(--font-size-xl)', color: 'var(--color-accent)' }}>{headline}</div>
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

const formatAxisTime = (timestamp: string) => {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
};

const defaultValueFormatter = (unit?: string) => (value: number) => {
  if (!Number.isFinite(value)) {
    return '\u2014';
  }
  if (unit) {
    return `${value.toFixed(2)}${unit}`;
  }
  return value.toFixed(2);
};

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
}: MetricLineChartProps) => (
  <div className={className} style={{ width: '100%', ...(style ?? {}) }}>
    {data.length > 0 ? (
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
          <CartesianGrid stroke="var(--alpha-accent-15)" strokeDasharray="4 4" />
          <XAxis
            dataKey="timestamp"
            stroke="var(--color-text-dim)"
            tickFormatter={formatAxisTime}
            minTickGap={20}
          />
          <YAxis
            stroke="var(--color-text-dim)"
            domain={yDomain}
            tickFormatter={valueFormatter ?? defaultValueFormatter(unit)}
          />
          <Tooltip
            contentStyle={{
              background: 'rgba(7, 25, 16, 0.88)',
              border: '1px solid var(--color-surface-border)',
              borderRadius: 'var(--border-radius-md)',
              color: 'var(--color-text)'
            }}
            labelStyle={{ color: 'var(--color-text-bright)', fontWeight: 600 }}
            labelFormatter={label => formatTimeLabel(label as string)}
            formatter={(value, key) => {
              const numericValue = typeof value === 'number' ? value : Number(value);
              const formatted = Number.isFinite(numericValue)
                ? (valueFormatter ?? defaultValueFormatter(unit))(numericValue)
                : String(value);
              const lineConfig = lines.find(line => line.dataKey === key);
              return [formatted, lineConfig?.name ?? (key as string)];
            }}
          />
          <Legend wrapperStyle={{ color: 'var(--color-text)' }} />
          {lines.map(line => (
            <Line
              key={line.dataKey}
              type={line.type ?? 'monotone'}
              dataKey={line.dataKey}
              name={line.name}
              stroke={line.color}
              strokeWidth={line.strokeWidth ?? 2}
              dot={false}
              isAnimationActive={false}
            />
          ))}
        </LineChart>
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
