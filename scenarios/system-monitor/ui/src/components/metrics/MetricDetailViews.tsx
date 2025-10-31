import { useMemo, useState, useCallback, useEffect, useRef } from 'react';
import type { ReactNode, CSSProperties, ChangeEvent } from 'react';
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
import { ArrowLeft, Cpu, MemoryStick, Network, HardDrive, CircuitBoard } from 'lucide-react';

import { ProcessMonitor } from '../monitoring/ProcessMonitor';
import { buildApiUrl } from '../../utils/apiBase';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  MetricHistory,
  ChartDataPoint,
  StorageIOInfo,
  DiskInfo,
  DiskDetailResponse,
  DiskPartitionInfo,
  DiskUsageEntry,
  GPUMetrics
} from '../../types';

interface MetricDetailLayoutProps {
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

interface MetricLineChartProps {
  data: Array<{ timestamp: string } & Record<string, number | string>>;
  lines: MetricLineChartLineConfig[];
  unit?: string;
  height?: number;
  yDomain?: [number, number] | ['auto', 'auto'];
  valueFormatter?: (value: number) => string;
  className?: string;
  style?: CSSProperties;
}

interface CpuDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  processMonitorData: ProcessMonitorData | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

interface MemoryDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

interface NetworkDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

interface DiskDetailViewProps {
  detailedMetrics: DetailedMetrics | null;
  storageIO?: StorageIOInfo | null;
  metricHistory: MetricHistory | null;
  diskLastUpdated?: string;
  onBack: () => void;
}

interface GpuDetailViewProps {
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

const MetricDetailLayout = ({ title, icon, headline, subhead, onBack, children }: MetricDetailLayoutProps) => (
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
      <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
        {subhead}
      </div>
    )}
    {children}
  </div>
);

const formatTimeLabel = (timestamp: string) => {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
};

const formatAxisTime = (timestamp: string) => {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
};

const defaultValueFormatter = (unit?: string) => (value: number) => {
  if (!Number.isFinite(value)) {
    return '—';
  }
  if (unit) {
    return `${value.toFixed(2)}${unit}`;
  }
  return value.toFixed(2);
};

const formatPercentage = (value: number) => `${value.toFixed(1)}%`;
const formatInteger = (value: number) => Math.round(value).toLocaleString();
const formatMbPerSecond = (value: number) => `${value.toFixed(2)} MB/s`;

const formatMegabytes = (value?: number) => {
  if (!Number.isFinite(value ?? NaN)) {
    return '—';
  }
  return `${Number(value).toFixed(0)} MB`;
};

const formatBytes = (value?: number) => {
  if (!Number.isFinite(value ?? NaN) || (value ?? 0) <= 0) {
    return '—';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const absolute = Math.max(0, value ?? 0);
  if (absolute === 0) {
    return '0 B';
  }
  const exponent = Math.min(Math.floor(Math.log(absolute) / Math.log(1024)), units.length - 1);
  const scaled = absolute / Math.pow(1024, exponent);
  const precision = scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2;
  return `${scaled.toFixed(precision)} ${units[exponent]}`;
};

const buildSingleSeriesData = (series?: ChartDataPoint[]) => {
  if (!series || series.length === 0) {
    return [] as Array<{ timestamp: string; value: number }>;
  }
  return [...series]
    .map(point => ({ timestamp: point.timestamp, value: Number(point.value) }))
    .filter(point => !Number.isNaN(point.value))
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
};

const combineDiskSeries = (readSeries?: ChartDataPoint[], writeSeries?: ChartDataPoint[]) => {
  const combined = new Map<string, { timestamp: string; read: number; write: number }>();
  (readSeries ?? []).forEach(point => {
    const existing = combined.get(point.timestamp) ?? { timestamp: point.timestamp, read: 0, write: 0 };
    existing.read = Number(point.value) || 0;
    combined.set(point.timestamp, existing);
  });
  (writeSeries ?? []).forEach(point => {
    const existing = combined.get(point.timestamp) ?? { timestamp: point.timestamp, read: 0, write: 0 };
    existing.write = Number(point.value) || 0;
    combined.set(point.timestamp, existing);
  });
  return Array.from(combined.values()).sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
};

const MetricLineChart = ({
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
        color: 'var(--color-text-dim)',
        padding: 'var(--spacing-xl)',
        letterSpacing: '0.08em'
      }}>
        Waiting for timeseries data…
      </div>
    )}
  </div>
);

const buildDiskUsageCard = (
  diskUsage?: DiskInfo,
  options?: { title?: string; subtitle?: string }
) => {
  if (!diskUsage) {
    return (
      <div className="card" style={{ padding: 'var(--spacing-lg)' }}>
        <h3 style={{ marginTop: 0, color: 'var(--color-text-bright)' }}>{options?.title ?? 'Disk Utilization'}</h3>
        <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
          Disk usage metrics are unavailable.
        </div>
      </div>
    );
  }

  const freeBytes = diskUsage.total - diskUsage.used;

  return (
    <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
      <div>
        <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>{options?.title ?? 'Disk Utilization'}</h3>
        <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
          {options?.subtitle ?? 'Current usage across monitored volumes'}
        </div>
      </div>
      <div style={{
        width: '100%',
        height: '10px',
        background: 'var(--alpha-accent-10)',
        borderRadius: 'var(--border-radius-sm)'
      }}>
        <div
          style={{
            width: `${Math.min(Math.max(diskUsage.percent, 0), 100)}%`,
            height: '100%',
            background: 'linear-gradient(90deg, var(--color-warning), var(--color-error))',
            borderRadius: 'var(--border-radius-sm)',
            boxShadow: '0 0 12px rgba(255,0,64,0.35)'
          }}
        />
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 'var(--spacing-md)' }}>
        <div>
          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Used</div>
          <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{formatBytes(diskUsage.used)}</div>
        </div>
        <div>
          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Free</div>
          <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{formatBytes(freeBytes)}</div>
        </div>
        <div>
          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Capacity</div>
          <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{formatBytes(diskUsage.total)}</div>
        </div>
        <div>
          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Utilization</div>
          <div style={{ color: 'var(--color-warning)', fontSize: 'var(--font-size-lg)' }}>{diskUsage.percent.toFixed(1)}%</div>
        </div>
      </div>
    </div>
  );
};

const renderProcessTable = (
  processes: Array<{ name: string; pid: number; cpu_percent?: number; memory_mb?: number }> | undefined,
  valueLabel: string,
  valueAccessor: (process: { cpu_percent?: number; memory_mb?: number }) => number | undefined
) => {
  if (!processes || processes.length === 0) {
    return (
      <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
        No process data available yet.
      </div>
    );
  }

  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 'var(--font-size-sm)' }}>
        <thead>
          <tr style={{ color: 'var(--color-text-dim)', textAlign: 'left' }}>
            <th style={{ padding: 'var(--spacing-xs)' }}>Process</th>
            <th style={{ padding: 'var(--spacing-xs)' }}>PID</th>
            <th style={{ padding: 'var(--spacing-xs)' }}>{valueLabel}</th>
          </tr>
        </thead>
        <tbody>
          {processes.slice(0, 10).map(process => {
            const value = valueAccessor(process);
            return (
              <tr key={`${process.name}-${process.pid}`} style={{ borderTop: '1px solid var(--alpha-accent-10)' }}>
                <td style={{ padding: 'var(--spacing-xs)', color: 'var(--color-text-bright)' }}>{process.name}</td>
                <td style={{ padding: 'var(--spacing-xs)', color: 'var(--color-text)' }}>{process.pid}</td>
                <td style={{ padding: 'var(--spacing-xs)', color: 'var(--color-accent)' }}>
                  {value !== undefined ? value.toFixed(1) : '—'}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

const renderGrowthPatterns = (
  patterns: Array<{ process: string; growth_mb_per_hour: number; risk_level: string }> | undefined
) => {
  if (!patterns || patterns.length === 0) {
    return (
      <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
        No anomalous growth patterns detected.
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
      {patterns.slice(0, 8).map(pattern => (
        <div
          key={`${pattern.process}-${pattern.growth_mb_per_hour}`}
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            fontSize: 'var(--font-size-sm)'
          }}
        >
          <span style={{ color: 'var(--color-text)' }}>{pattern.process}</span>
          <span
            style={{
              color:
                pattern.risk_level === 'high'
                  ? 'var(--color-error)'
                  : pattern.risk_level === 'medium'
                    ? 'var(--color-warning)'
                    : 'var(--color-success)'
            }}
          >
            {pattern.growth_mb_per_hour.toFixed(1)} MB/hr ({pattern.risk_level})
          </span>
        </div>
      ))}
    </div>
  );
};

export const CpuDetailView = ({ metrics, detailedMetrics, processMonitorData, metricHistory, onBack }: CpuDetailViewProps) => {
  const cpuUsage = detailedMetrics?.cpu_details?.usage ?? metrics?.cpu_usage ?? 0;
  const cpuData = useMemo(() => buildSingleSeriesData(metricHistory?.cpu), [metricHistory?.cpu]);
  const loadAverage = detailedMetrics?.cpu_details?.load_average ?? [];
  const contextSwitches = detailedMetrics?.cpu_details?.context_switches ?? 0;
  const goroutines = detailedMetrics?.cpu_details?.total_goroutines ?? 0;
  const topProcesses = detailedMetrics?.cpu_details?.top_processes;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatTimeLabel(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="CPU PERFORMANCE"
      icon={<Cpu size={22} />}
      headline={`${cpuUsage.toFixed(1)}% utilization`}
      subhead={subhead}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        style={{ padding: 'var(--spacing-lg)' }}
        data={cpuData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'CPU Usage', color: 'var(--color-accent)' }]}
        unit="%"
        yDomain={[0, 100]}
        valueFormatter={value => `${value.toFixed(1)}%`}
      />

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: 'var(--spacing-lg)' }}>
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Load Profile</h3>
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
            1m / 5m / 15m load average
          </div>
          <div style={{ display: 'flex', gap: 'var(--spacing-md)' }}>
            {loadAverage.slice(0, 3).map((value, index) => (
              <div key={`${value}-${index}`} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>
                  {index === 0 ? '1 min' : index === 1 ? '5 min' : '15 min'}
                </span>
                <span style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{value.toFixed(2)}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Runtime Signals</h3>
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
              Scheduler and goroutine metrics
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 'var(--spacing-md)' }}>
            <div>
              <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Context Switches</div>
              <div style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-lg)' }}>{formatInteger(contextSwitches)}</div>
            </div>
            <div>
              <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Goroutines</div>
              <div style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-lg)' }}>{formatInteger(goroutines)}</div>
            </div>
          </div>
        </div>
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
        <div>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Top CPU Consumers</h3>
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
            Processes ranked by CPU utilization
          </div>
        </div>
        {renderProcessTable(topProcesses, 'CPU %', process => process.cpu_percent)}
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)' }}>
        <ProcessMonitor data={processMonitorData} collapsible={false} isExpanded={true} />
      </div>
    </MetricDetailLayout>
  );
};

export const MemoryDetailView = ({ metrics, detailedMetrics, metricHistory, onBack }: MemoryDetailViewProps) => {
  const memoryUsage = detailedMetrics?.memory_details?.usage ?? metrics?.memory_usage ?? 0;
  const memoryData = useMemo(() => buildSingleSeriesData(metricHistory?.memory), [metricHistory?.memory]);
  const memoryDetails = detailedMetrics?.memory_details;

  const swapUsage = memoryDetails?.swap_usage;
  const growthPatterns = memoryDetails?.growth_patterns;
  const topProcesses = memoryDetails?.top_processes;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatTimeLabel(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="MEMORY UTILIZATION"
      icon={<MemoryStick size={22} />}
      headline={`${memoryUsage.toFixed(1)}% used`}
      subhead={subhead}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        style={{ padding: 'var(--spacing-lg)' }}
        data={memoryData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'Memory Usage', color: 'var(--color-warning)' }]}
        unit="%"
        yDomain={[0, 100]}
        valueFormatter={value => `${value.toFixed(1)}%`}
      />

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: 'var(--spacing-lg)' }}>
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Swap Activity</h3>
          {swapUsage ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 'var(--spacing-md)' }}>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Swap Used</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{formatBytes(swapUsage.used)}</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Swap Total</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{formatBytes(swapUsage.total)}</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Utilization</div>
                <div style={{ color: 'var(--color-warning)', fontSize: 'var(--font-size-lg)' }}>{swapUsage.percent.toFixed(1)}%</div>
              </div>
            </div>
          ) : (
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
              Swap metrics unavailable.
            </div>
          )}
        </div>

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Growth Patterns</h3>
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
            Heaviest allocators during the observation window
          </div>
          {renderGrowthPatterns(growthPatterns)}
        </div>
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
        <div>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Top Memory Consumers</h3>
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
            Processes ranked by resident set size
          </div>
        </div>
        {renderProcessTable(topProcesses, 'Memory (MB)', process => process.memory_mb)}
      </div>
    </MetricDetailLayout>
  );
};

export const NetworkDetailView = ({ metrics, detailedMetrics, metricHistory, onBack }: NetworkDetailViewProps) => {
  const networkData = useMemo(() => buildSingleSeriesData(metricHistory?.network), [metricHistory?.network]);
  const networkDetails = detailedMetrics?.network_details;
  const totalConnections = metrics?.tcp_connections ?? networkDetails?.tcp_states?.total ?? 0;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatTimeLabel(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="NETWORK ACTIVITY"
      icon={<Network size={22} />}
      headline={`${totalConnections.toLocaleString()} active connections`}
      subhead={subhead}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        style={{ padding: 'var(--spacing-lg)' }}
        data={networkData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'TCP Connections', color: 'var(--color-accent)' }]}
        unit=""
        yDomain={['auto', 'auto']}
        valueFormatter={value => `${Math.round(value).toLocaleString()} connections`}
      />

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 'var(--spacing-lg)' }}>
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>TCP States</h3>
          {networkDetails ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))', gap: 'var(--spacing-sm)' }}>
              {Object.entries(networkDetails.tcp_states).filter(([key]) => key !== 'total').map(([state, value]) => (
                <div key={state} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>{state.toUpperCase()}</span>
                  <span style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-md)' }}>{Number(value).toLocaleString()}</span>
                </div>
              ))}
            </div>
          ) : (
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
              Connection state metrics unavailable.
            </div>
          )}
        </div>

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Network Health</h3>
          {networkDetails ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 'var(--spacing-md)' }}>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Ingress Bandwidth</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.network_stats.bandwidth_in_mbps.toFixed(2)} Mbps</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Egress Bandwidth</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.network_stats.bandwidth_out_mbps.toFixed(2)} Mbps</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Packet Loss</div>
                <div style={{ color: 'var(--color-warning)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.network_stats.packet_loss.toFixed(2)}%</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>DNS Success</div>
                <div style={{ color: 'var(--color-success)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.network_stats.dns_success_rate.toFixed(1)}%</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>DNS Latency</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.network_stats.dns_latency_ms.toFixed(0)} ms</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Port Usage</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.port_usage.used} / {networkDetails.port_usage.total}</div>
              </div>
            </div>
          ) : (
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
              Network statistics unavailable.
            </div>
          )}
        </div>
      </div>

      {networkDetails?.connection_pools && networkDetails.connection_pools.length > 0 && (
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Connection Pools</h3>
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
              Resource utilization across HTTP/database pools
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 'var(--spacing-md)' }}>
            {networkDetails.connection_pools.map(pool => (
              <div key={pool.name} style={{
                border: '1px solid var(--alpha-accent-15)',
                borderRadius: 'var(--border-radius-md)',
                padding: 'var(--spacing-md)',
                background: 'rgba(0, 0, 0, 0.4)'
              }}>
                <div style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-sm)' }}>{pool.name}</div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>
                  Active: <span style={{ color: 'var(--color-text)' }}>{pool.active}</span> · Idle: <span style={{ color: 'var(--color-text)' }}>{pool.idle}</span>
                </div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>
                  Waiting: <span style={{ color: 'var(--color-text)' }}>{pool.waiting}</span> / Max {pool.max_size}
                </div>
                <div style={{
                  marginTop: 'var(--spacing-xs)',
                  color: pool.leak_risk === 'high'
                    ? 'var(--color-error)'
                    : pool.leak_risk === 'medium'
                      ? 'var(--color-warning)'
                      : 'var(--color-success)'
                }}>
                  Leak risk: {pool.leak_risk}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </MetricDetailLayout>
  );
};

export const DiskDetailView = ({ detailedMetrics, storageIO, metricHistory, diskLastUpdated, onBack }: DiskDetailViewProps) => {
  const DEFAULT_DEPTH = 2;
  const [diskDetails, setDiskDetails] = useState<DiskDetailResponse | null>(null);
  const [selectedMount, setSelectedMount] = useState<string>('/');
  const [depth, setDepth] = useState<number>(DEFAULT_DEPTH);
  const [includeFiles, setIncludeFiles] = useState<boolean>(false);
  const [detailsLoading, setDetailsLoading] = useState<boolean>(false);
  const [detailsError, setDetailsError] = useState<string | null>(null);
  const activeRequestRef = useRef<AbortController | null>(null);

  const diskUsage = detailedMetrics?.memory_details?.disk_usage;
  const diskIoHistory = useMemo(
    () => combineDiskSeries(metricHistory?.diskRead, metricHistory?.diskWrite),
    [metricHistory?.diskRead, metricHistory?.diskWrite]
  );
  const diskUsageHistory = useMemo(() => buildSingleSeriesData(metricHistory?.diskUsage), [metricHistory?.diskUsage]);
  const fileDescriptors = detailedMetrics?.system_details?.file_descriptors;
  const inotifyWatchers = detailedMetrics?.system_details?.inotify_watchers;
  const watchersSupported = inotifyWatchers?.supported ?? true;
  const watcherPercent = inotifyWatchers && Number.isFinite(inotifyWatchers.watches_percent)
    ? inotifyWatchers.watches_percent
    : undefined;
  const watcherInstancePercent = inotifyWatchers && Number.isFinite(inotifyWatchers.instances_percent)
    ? inotifyWatchers.instances_percent
    : undefined;
  const getUtilizationColor = (percent: number) => (
    percent >= 85 ? 'var(--color-error)' : percent >= 70 ? 'var(--color-warning)' : 'var(--color-success)'
  );

  const fetchDiskDetails = useCallback(
    async (mount: string, nextDepth: number, includeFilesValue: boolean) => {
      if (activeRequestRef.current) {
        activeRequestRef.current.abort();
      }
      setDetailsLoading(true);
      setDetailsError(null);
      const controller = new AbortController();
      activeRequestRef.current = controller;
      try {
        const params = new URLSearchParams();
        if (mount) {
          params.set('mount', mount);
        }
        params.set('depth', String(nextDepth));
        params.set('limit', '8');
        if (includeFilesValue) {
          params.set('include_files', 'true');
        }

        const response = await fetch(buildApiUrl(`/api/metrics/disk/details?${params.toString()}`), {
          signal: controller.signal
        });
        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || `request failed with status ${response.status}`);
        }

        const data: DiskDetailResponse = await response.json();
        setDiskDetails(data);
        setSelectedMount(data.active_mount || mount);
        setDepth(data.depth);
        setIncludeFiles(includeFilesValue);
        setDetailsError(null);
        activeRequestRef.current = null;
      } catch (error) {
        activeRequestRef.current = null;
        if ((error as { name?: string })?.name === 'AbortError') {
          setDetailsError('Scan cancelled');
        } else {
          setDetailsError(error instanceof Error ? error.message : String(error));
        }
      } finally {
        setDetailsLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    fetchDiskDetails('/', DEFAULT_DEPTH, false);
    return () => {
      if (activeRequestRef.current) {
        activeRequestRef.current.abort();
      }
    };
  }, [fetchDiskDetails]);

  const partitions = diskDetails?.partitions ?? [];
  const activePartition = useMemo<DiskPartitionInfo | null>(() => {
    if (partitions.length === 0) {
      return null;
    }
    const exact = partitions.find(partition => partition.mount_point === selectedMount);
    return exact ?? partitions[0];
  }, [partitions, selectedMount]);

  const summaryDiskInfo: DiskInfo | undefined = activePartition
    ? {
        used: activePartition.used_bytes,
        total: activePartition.size_bytes,
        percent: activePartition.use_percent
      }
    : diskUsage;

  const selectedMountLabel = activePartition?.mount_point ?? selectedMount;
  const deviceLabel = activePartition?.device ? `Device ${activePartition.device}` : undefined;

  const lastUpdated = diskDetails?.timestamp
    ? diskDetails.timestamp
    : diskLastUpdated ?? detailedMetrics?.timestamp;

  const subheadParts: string[] = [];
  if (deviceLabel) {
    subheadParts.push(deviceLabel);
  }
  if (lastUpdated) {
    subheadParts.push(`Last scan ${formatTimeLabel(lastUpdated)}`);
  }
  if (detailsLoading) {
    subheadParts.push('Analyzing…');
  }

  const topDirectories = diskDetails?.top_directories ?? [];
  const largestFiles = diskDetails?.largest_files ?? [];

  const handleMountSelect = (mountPoint: string) => {
    setSelectedMount(mountPoint);
    fetchDiskDetails(mountPoint, depth, includeFiles);
  };

  const handleDepthChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextDepth = Number(event.target.value);
    setDepth(nextDepth);
    fetchDiskDetails(selectedMount, nextDepth, includeFiles);
  };

  const handleRefresh = () => {
    fetchDiskDetails(selectedMount, depth, includeFiles);
  };

  const handleScanLargestFiles = () => {
    setIncludeFiles(true);
    fetchDiskDetails(selectedMount, depth, true);
  };

  const handleStopScan = () => {
    if (activeRequestRef.current) {
      activeRequestRef.current.abort();
      activeRequestRef.current = null;
      setDetailsLoading(false);
    }
  };

  return (
    <MetricDetailLayout
      title="DISK PERFORMANCE"
      icon={<HardDrive size={22} />}
      headline={summaryDiskInfo ? `${summaryDiskInfo.percent.toFixed(1)}% utilized on ${selectedMountLabel}` : 'Awaiting disk telemetry'}
      subhead={subheadParts.length > 0 ? subheadParts.join(' • ') : undefined}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        style={{ padding: 'var(--spacing-lg)' }}
        data={diskIoHistory.map(point => ({ timestamp: point.timestamp, read: point.read, write: point.write }))}
        lines={[
          { dataKey: 'read', name: 'Read Throughput', color: 'var(--color-accent)' },
          { dataKey: 'write', name: 'Write Throughput', color: 'var(--color-warning)' }
        ]}
        unit=" MB/s"
        valueFormatter={formatMbPerSecond}
        yDomain={['auto', 'auto']}
      />

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 'var(--spacing-lg)' }}>
        {buildDiskUsageCard(summaryDiskInfo, {
          title: `Usage for ${selectedMountLabel}`,
          subtitle: deviceLabel ?? 'Current usage across monitored volumes'
        })}

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Storage I/O Snapshot</h3>
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
              Real-time disk queue and wait metrics
            </div>
          </div>
          {storageIO ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 'var(--spacing-md)' }}>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Disk Queue Depth</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{storageIO.disk_queue_depth.toFixed(2)}</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>I/O Wait</div>
                <div style={{ color: 'var(--color-warning)', fontSize: 'var(--font-size-lg)' }}>{storageIO.io_wait_percent.toFixed(1)}%</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Read Throughput</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{storageIO.read_mb_per_sec.toFixed(2)} MB/s</div>
              </div>
              <div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.08em' }}>Write Throughput</div>
                <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>{storageIO.write_mb_per_sec.toFixed(2)} MB/s</div>
              </div>
            </div>
          ) : (
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
              Storage I/O metrics unavailable.
            </div>
          )}
        </div>

        {fileDescriptors && (
          <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
            <div>
              <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>File Descriptor Utilization</h3>
              <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
                Tracks open file handles across all services
              </div>
            </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
            <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-xl)', fontWeight: 600 }}>
              {fileDescriptors.used.toLocaleString()} / {fileDescriptors.max.toLocaleString()}
            </div>
            <div style={{
              color: getUtilizationColor(fileDescriptors.percent),
              fontSize: 'var(--font-size-lg)',
              fontWeight: 600
            }}>
              {fileDescriptors.percent.toFixed(1)}%
            </div>
          </div>
            <div style={{
              width: '100%',
              height: '6px',
              background: 'var(--alpha-accent-20)',
              borderRadius: '3px',
              overflow: 'hidden'
            }}>
              <div
                style={{
                  width: `${Math.min(Math.max(fileDescriptors.percent, 0), 100)}%`,
                  height: '100%',
                  background: fileDescriptors.percent >= 70
                    ? getUtilizationColor(fileDescriptors.percent)
                    : 'linear-gradient(90deg, var(--color-accent), var(--color-text-bright))',
                  transition: 'width var(--transition-normal)'
                }}
              />
            </div>
            <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.06em' }}>
              Sustained values above 80% risk "too many open files" errors during heavy disk activity.
            </div>
          </div>
        )}

        {inotifyWatchers && (
          <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
            <div>
              <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Inotify Watcher Utilization</h3>
              <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
                Kernel file watcher instances and watch descriptors in use
              </div>
            </div>
            {watchersSupported ? (
              <>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
                  <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)', fontWeight: 600 }}>
                    {inotifyWatchers.watches_used.toLocaleString()} / {inotifyWatchers.watches_max.toLocaleString()} watches
                  </div>
                  <div style={{
                    color: watcherPercent !== undefined ? getUtilizationColor(watcherPercent) : 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-md)',
                    fontWeight: 600
                  }}>
                    {watcherPercent !== undefined ? `${watcherPercent.toFixed(1)}%` : '—'}
                  </div>
                </div>
                <div style={{
                  width: '100%',
                  height: '6px',
                  background: 'var(--alpha-accent-20)',
                  borderRadius: '3px',
                  overflow: 'hidden'
                }}>
                  <div
                    style={{
                      width: `${Math.min(Math.max(watcherPercent ?? 0, 0), 100)}%`,
                      height: '100%',
                      background: watcherPercent !== undefined && watcherPercent >= 70
                        ? getUtilizationColor(watcherPercent)
                        : 'linear-gradient(90deg, var(--color-accent), var(--color-text-bright))',
                      transition: 'width var(--transition-normal)'
                    }}
                  />
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.06em' }}>
                  <span>
                    Instances: {inotifyWatchers.instances_used.toLocaleString()} / {inotifyWatchers.instances_max.toLocaleString()}
                  </span>
                  <span style={{ color: watcherInstancePercent !== undefined ? getUtilizationColor(watcherInstancePercent) : 'var(--color-text-dim)' }}>
                    {watcherInstancePercent !== undefined ? `${watcherInstancePercent.toFixed(1)}%` : '—'}
                  </span>
                </div>
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.06em' }}>
                  Keep watcher usage below 80% to avoid hitting Linux inotify limits that break file watchers and dev servers.
                </div>
              </>
            ) : (
              <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
                Inotify watcher metrics are not available on this platform.
              </div>
            )}
          </div>
        )}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 'var(--spacing-lg)' }}>
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Mounted Volumes</h3>
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
              Select a mount to drill into its usage profile
            </div>
          </div>
          {partitions.length > 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
              {partitions.map(partition => {
                const isActive = partition.mount_point === selectedMount;
                const percent = Math.min(Math.max(partition.use_percent, 0), 100);
                return (
                  <button
                    key={`${partition.device}-${partition.mount_point}`}
                    type="button"
                    onClick={() => handleMountSelect(partition.mount_point)}
                    disabled={detailsLoading && isActive}
                    style={{
                      textAlign: 'left',
                      border: `1px solid ${isActive ? 'var(--color-text-bright)' : 'var(--alpha-accent-20)'}`,
                      background: 'rgba(0,0,0,0.4)',
                      color: 'var(--color-text)',
                      padding: 'var(--spacing-sm) var(--spacing-md)',
                      borderRadius: 'var(--border-radius-md)',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 'var(--spacing-xs)',
                      cursor: detailsLoading && isActive ? 'not-allowed' : 'pointer',
                      opacity: detailsLoading && isActive ? 0.6 : 1,
                      letterSpacing: '0.04em'
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
                      <span style={{ color: 'var(--color-text-bright)', fontWeight: 600 }}>{partition.mount_point}</span>
                      <span style={{ color: 'var(--color-warning)', fontSize: 'var(--font-size-sm)' }}>{percent.toFixed(1)}%</span>
                    </div>
                    <div style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-text-dim)' }}>{partition.device}</div>
                    <div style={{
                      width: '100%',
                      height: '6px',
                      background: 'var(--alpha-accent-10)',
                      borderRadius: 'var(--border-radius-sm)'
                    }}>
                      <div
                        style={{
                          width: `${percent}%`,
                          height: '100%',
                          background: 'linear-gradient(90deg, var(--color-warning), var(--color-error))',
                          borderRadius: 'var(--border-radius-sm)'
                        }}
                      />
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 'var(--font-size-xs)', color: 'var(--color-text-dim)' }}>
                      <span>Used {partition.used_human}</span>
                      <span>Free {partition.available_human}</span>
                    </div>
                  </button>
                );
              })}
            </div>
          ) : (
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
              Partition information is unavailable on this platform.
            </div>
          )}
        </div>

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Analysis Controls</h3>
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
              Customize the depth and scope of disk analysis
            </div>
          </div>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)', color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
            Depth
            <select
              value={depth}
              onChange={handleDepthChange}
              disabled={detailsLoading}
              style={{
                background: 'rgba(0,0,0,0.6)',
                color: 'var(--color-text)',
                border: '1px solid var(--color-accent)',
                borderRadius: 'var(--border-radius-md)',
                padding: 'var(--spacing-xs) var(--spacing-sm)'
              }}
            >
              <option value={1}>Top-level directories</option>
              <option value={2}>Include first sub-level</option>
              <option value={3}>Include two sub-levels</option>
              <option value={4}>Deep scan (slower)</option>
            </select>
          </label>
          <div style={{ display: 'flex', gap: 'var(--spacing-sm)', flexWrap: 'wrap' }}>
            <button
              type="button"
              className="btn btn-action"
              onClick={handleRefresh}
              disabled={detailsLoading}
            >
              {detailsLoading ? 'Scanning…' : 'Refresh Scan'}
            </button>
            <button
              type="button"
              className="btn btn-action"
              onClick={handleScanLargestFiles}
              disabled={detailsLoading}
            >
              {includeFiles ? 'Rescan Largest Files' : 'Find Largest Files (>50MB)'}
            </button>
            <button
              type="button"
              className="btn btn-action"
              onClick={handleStopScan}
              disabled={!detailsLoading}
            >
              Stop Scan
            </button>
          </div>
          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', letterSpacing: '0.06em' }}>
            Deeper scans and file discovery may take longer on large volumes.
          </div>
        </div>
      </div>

      {detailsError && (
        <div className="card" style={{ padding: 'var(--spacing-lg)', color: 'var(--color-error)', letterSpacing: '0.08em' }}>
          Failed to analyze disk usage: {detailsError}
        </div>
      )}

      {diskDetails?.notes && diskDetails.notes.length > 0 && (
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)', color: 'var(--color-warning)' }}>
          {diskDetails.notes.map((note, index) => (
            <div key={`${note}-${index}`} style={{ letterSpacing: '0.08em' }}>
              • {note}
            </div>
          ))}
        </div>
      )}

      <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', flexWrap: 'wrap', gap: 'var(--spacing-sm)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Top Directories</h3>
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
              Heaviest paths within {selectedMountLabel} (depth {depth})
            </div>
          </div>
        </div>
        {detailsLoading && topDirectories.length === 0 ? (
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>Analyzing directory usage…</div>
        ) : topDirectories.length > 0 ? (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 'var(--font-size-sm)' }}>
              <thead>
                <tr style={{ color: 'var(--color-text-dim)', textAlign: 'left' }}>
                  <th style={{ padding: 'var(--spacing-xs)' }}>Path</th>
                  <th style={{ padding: 'var(--spacing-xs)' }}>Size</th>
                </tr>
              </thead>
              <tbody>
                {topDirectories.map((entry: DiskUsageEntry) => (
                  <tr key={entry.path} style={{ borderTop: '1px solid var(--alpha-accent-15)' }}>
                    <td style={{ padding: 'var(--spacing-xs)', color: 'var(--color-text-bright)' }}>{entry.path}</td>
                    <td style={{ padding: 'var(--spacing-xs)', color: 'var(--color-accent)', whiteSpace: 'nowrap' }}>{entry.size_human}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
            No directories exceeded the scan threshold at this depth.
          </div>
        )}
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
        <div>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Largest Files</h3>
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
            Files larger than 50 MB within {selectedMountLabel}
          </div>
        </div>
        {includeFiles ? (
          largestFiles.length > 0 ? (
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 'var(--font-size-sm)' }}>
                <thead>
                  <tr style={{ color: 'var(--color-text-dim)', textAlign: 'left' }}>
                    <th style={{ padding: 'var(--spacing-xs)' }}>File</th>
                    <th style={{ padding: 'var(--spacing-xs)' }}>Size</th>
                  </tr>
                </thead>
                <tbody>
                  {largestFiles.map(entry => (
                    <tr key={entry.path} style={{ borderTop: '1px solid var(--alpha-accent-15)' }}>
                      <td style={{ padding: 'var(--spacing-xs)', color: 'var(--color-text-bright)' }}>{entry.path}</td>
                      <td style={{ padding: 'var(--spacing-xs)', color: 'var(--color-accent)', whiteSpace: 'nowrap' }}>{entry.size_human}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : detailsLoading ? (
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>Scanning for large files…</div>
          ) : (
            <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
              No files above 50 MB were detected in this mount.
            </div>
          )
        ) : (
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
            Run the "Find Largest Files" scan to surface oversized artifacts.
          </div>
        )}
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
        <div>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Disk Usage History</h3>
          <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
            Utilization trend across the observation window
          </div>
        </div>
        <MetricLineChart
          data={diskUsageHistory.map(point => ({ timestamp: point.timestamp, value: point.value }))}
          lines={[{ dataKey: 'value', name: 'Disk Usage', color: 'var(--color-info)' }]}
          unit="%"
          valueFormatter={formatPercentage}
          yDomain={[0, 100]}
          height={260}
        />
      </div>
    </MetricDetailLayout>
  );
};

export const GpuDetailView = ({ detailedMetrics, metricHistory, onBack }: GpuDetailViewProps) => {
  const gpuMetrics: GPUMetrics | null = detailedMetrics?.gpu_details ?? null;
  const gpuHistory = useMemo(() => buildSingleSeriesData(metricHistory?.gpu), [metricHistory?.gpu]);

  const headline = gpuMetrics
    ? `${gpuMetrics.summary.average_utilization_percent.toFixed(1)}% Avg`
    : 'Awaiting telemetry';

  const subheadParts: string[] = [];
  if (gpuMetrics?.driver_version) {
    subheadParts.push(`Driver ${gpuMetrics.driver_version}`);
  }
  if (gpuMetrics?.primary_model) {
    subheadParts.push(gpuMetrics.primary_model);
  }

  return (
    <MetricDetailLayout
      title="GPU UTILIZATION"
      icon={<CircuitBoard size={18} />}
      headline={headline}
      subhead={subheadParts.length > 0 ? subheadParts.join(' • ') : undefined}
      onBack={onBack}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-lg)' }}>
        <MetricLineChart
          data={gpuHistory}
          lines={[{ dataKey: 'value', name: 'Utilization', color: 'var(--color-info)', strokeWidth: 2 }]}
          unit="%"
          yDomain={[0, 100]}
          valueFormatter={formatPercentage}
        />

        {gpuMetrics ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-lg)' }}>
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
              gap: 'var(--spacing-md)'
            }}>
              {[{
                label: 'Devices',
                value: String(gpuMetrics.summary.device_count)
              }, {
                label: 'Average Utilization',
                value: `${gpuMetrics.summary.average_utilization_percent.toFixed(1)}%`
              }, {
                label: 'Memory Used',
                value: `${gpuMetrics.summary.used_memory_mb.toFixed(0)} / ${gpuMetrics.summary.total_memory_mb.toFixed(0)} MB`
              }, {
                label: 'Average Temperature',
                value: gpuMetrics.summary.device_count > 0 && gpuMetrics.summary.average_temperature_c > 0
                  ? `${gpuMetrics.summary.average_temperature_c.toFixed(1)}°C`
                  : '—'
              }].map(stat => (
                <div key={stat.label} style={{
                  border: '1px solid var(--alpha-accent-20)',
                  borderRadius: 'var(--border-radius-md)',
                  padding: 'var(--spacing-md)',
                  background: 'rgba(0, 40, 0, 0.2)'
                }}>
                  <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-xs)', marginBottom: 'var(--spacing-xs)' }}>
                    {stat.label}
                  </div>
                  <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)', fontWeight: 600 }}>
                    {stat.value}
                  </div>
                </div>
              ))}
            </div>

            {gpuMetrics.errors && gpuMetrics.errors.length > 0 && (
              <div style={{
                border: '1px solid var(--color-warning)',
                borderRadius: 'var(--border-radius-md)',
                padding: 'var(--spacing-sm) var(--spacing-md)',
                color: 'var(--color-warning)',
                fontSize: 'var(--font-size-sm)'
              }}>
                {gpuMetrics.errors.join(' • ')}
              </div>
            )}

            <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
              {gpuMetrics.devices.length > 0 ? (
                gpuMetrics.devices.map(device => (
                  <div key={device.uuid || device.index} style={{
                    border: '1px solid var(--alpha-accent-20)',
                    borderRadius: 'var(--border-radius-md)',
                    padding: 'var(--spacing-md)',
                    background: 'rgba(0,0,0,0.35)'
                  }}>
                    <div style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      marginBottom: 'var(--spacing-sm)'
                    }}>
                      <span style={{ color: 'var(--color-text-bright)', fontWeight: 600 }}>
                        {device.name} (GPU {device.index})
                      </span>
                      <span style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-sm)' }}>
                        {device.utilization_percent.toFixed(1)}%
                      </span>
                    </div>

                    <div style={{
                      display: 'grid',
                      gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
                      gap: 'var(--spacing-sm)',
                      color: 'var(--color-text-dim)',
                      fontSize: 'var(--font-size-xs)'
                    }}>
                      <div>Memory: <span style={{ color: 'var(--color-text-bright)' }}>{formatMegabytes(device.memory_used_mb)} / {formatMegabytes(device.memory_total_mb)}</span></div>
                      <div>Memory Util: <span style={{ color: 'var(--color-text-bright)' }}>{formatPercentage(device.memory_utilization_percent)}</span></div>
                      <div>Temperature: <span style={{ color: 'var(--color-text-bright)' }}>{device.temperature_c != null ? `${device.temperature_c.toFixed(1)}°C` : '—'}</span></div>
                      <div>Fan: <span style={{ color: 'var(--color-text-bright)' }}>{device.fan_speed_percent != null ? `${device.fan_speed_percent.toFixed(0)}%` : '—'}</span></div>
                      <div>Power: <span style={{ color: 'var(--color-text-bright)' }}>{device.power_draw_w != null ? `${device.power_draw_w.toFixed(1)} W` : '—'}</span></div>
                      <div>SM Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.sm_clock_mhz != null ? `${device.sm_clock_mhz.toFixed(0)} MHz` : '—'}</span></div>
                      <div>Mem Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.memory_clock_mhz != null ? `${device.memory_clock_mhz.toFixed(0)} MHz` : '—'}</span></div>
                    </div>

                    {device.processes && device.processes.length > 0 && (
                      <div style={{ marginTop: 'var(--spacing-sm)' }}>
                        <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', marginBottom: 'var(--spacing-xxs)' }}>
                          Processes
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xxs)' }}>
                          {device.processes.map(process => (
                            <div key={`${device.uuid || device.index}-${process.pid}`} style={{
                              display: 'flex',
                              justifyContent: 'space-between',
                              color: 'var(--color-text-bright)',
                              fontSize: 'var(--font-size-xs)'
                            }}>
                              <span>{process.process_name} ({process.pid})</span>
                              <span>{formatMegabytes(process.memory_used_mb)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                ))
              ) : (
                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                  No GPU devices detected.
                </div>
              )}
            </div>
          </div>
        ) : (
          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
            GPU metrics unavailable on this host.
          </div>
        )}
      </div>
    </MetricDetailLayout>
  );
};
