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
          <CartesianGrid stroke="var(--color-primary-muted)" strokeDasharray="4 4" />
          <XAxis
            dataKey="timestamp"
            stroke="var(--color-text-secondary)"
            tickFormatter={formatChartTime}
            minTickGap={20}
          />
          <YAxis
            stroke="var(--color-text-secondary)"
            domain={yDomain}
            tickFormatter={valueFormatter ?? defaultValueFormatter(unit)}
          />
          <Tooltip
            contentStyle={{
              background: 'var(--color-surface-raised)',
              border: '1px solid var(--color-border)',
              borderRadius: 'var(--radius-md)',
              color: 'var(--color-text)'
            }}
            labelStyle={{ color: 'var(--color-text-heading)', fontWeight: 600 }}
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
