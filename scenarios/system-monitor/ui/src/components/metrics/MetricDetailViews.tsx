import { useMemo } from 'react';
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
import { ArrowLeft, Cpu, MemoryStick, Network, HardDrive } from 'lucide-react';

import { ProcessMonitor } from '../monitoring/ProcessMonitor';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  MetricHistory,
  ChartDataPoint,
  StorageIOInfo,
  DiskInfo
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
          <CartesianGrid stroke="rgba(0, 255, 0, 0.15)" strokeDasharray="4 4" />
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
              background: 'rgba(0, 0, 0, 0.85)',
              border: '1px solid var(--color-accent)',
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

const buildDiskUsageCard = (diskUsage?: DiskInfo) => {
  if (!diskUsage) {
    return (
      <div className="card" style={{ padding: 'var(--spacing-lg)' }}>
        <h3 style={{ marginTop: 0, color: 'var(--color-text-bright)' }}>Disk Utilization</h3>
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
        <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Disk Utilization</h3>
        <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em', fontSize: 'var(--font-size-sm)' }}>
          Current usage across monitored volumes
        </div>
      </div>
      <div style={{
        width: '100%',
        height: '10px',
        background: 'rgba(0, 255, 0, 0.1)',
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
              <tr key={`${process.name}-${process.pid}`} style={{ borderTop: '1px solid rgba(0, 255, 0, 0.1)' }}>
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
                border: '1px solid rgba(0, 255, 0, 0.15)',
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
  const diskUsage = detailedMetrics?.memory_details?.disk_usage;
  const diskIoHistory = useMemo(
    () => combineDiskSeries(metricHistory?.diskRead, metricHistory?.diskWrite),
    [metricHistory?.diskRead, metricHistory?.diskWrite]
  );
  const diskUsageHistory = useMemo(() => buildSingleSeriesData(metricHistory?.diskUsage), [metricHistory?.diskUsage]);

  const subhead = diskLastUpdated
    ? `Storage I/O sampled ${formatTimeLabel(diskLastUpdated)}`
    : detailedMetrics?.timestamp
      ? `Updated ${formatTimeLabel(detailedMetrics.timestamp)}`
      : undefined;

  return (
    <MetricDetailLayout
      title="DISK PERFORMANCE"
      icon={<HardDrive size={22} />}
      headline={diskUsage ? `${diskUsage.percent.toFixed(1)}% utilized` : 'Awaiting disk telemetry'}
      subhead={subhead}
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
        {buildDiskUsageCard(diskUsage)}

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
