import type { ChartDataPoint, DiskInfo } from '../../../types';
import { DetailRow } from '../../../shared/components/DetailRow';
import { formatBytes } from '../../../shared/utils/formatters';

// ── Data Builders ──────────────────────────────────────────────────────────

export const buildSingleSeriesData = (series?: ChartDataPoint[]) => {
  if (!series || series.length === 0) {
    return [] as Array<{ timestamp: string; value: number }>;
  }
  return [...series]
    .map(point => ({ timestamp: point.timestamp, value: Number(point.value) }))
    .filter(point => !Number.isNaN(point.value))
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
};

export const combineDiskSeries = (readSeries?: ChartDataPoint[], writeSeries?: ChartDataPoint[]) => {
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

// ── Render Helpers ─────────────────────────────────────────────────────────

export const buildDiskUsageCard = (
  diskUsage?: DiskInfo,
  options?: { title?: string; subtitle?: string }
) => {
  if (!diskUsage) {
    return (
      <div className="card" style={{ padding: 'var(--spacing-lg)' }}>
        <h3 className="section-heading">{options?.title ?? 'Disk Utilization'}</h3>
        <div className="text-muted">
          Disk usage metrics are unavailable.
        </div>
      </div>
    );
  }

  const freeBytes = diskUsage.total - diskUsage.used;

  return (
    <div className="card flex-col-gap-md" style={{ padding: 'var(--spacing-lg)' }}>
      <div>
        <h3 className="section-heading">{options?.title ?? 'Disk Utilization'}</h3>
        <div className="card-subtitle">
          {options?.subtitle ?? 'Current usage across monitored volumes'}
        </div>
      </div>
      <div className="progress-bar progress-bar-lg" style={{ borderRadius: 'var(--border-radius-sm)' }}>
        <div
          className="progress-fill"
          style={{
            width: `${Math.min(Math.max(diskUsage.percent, 0), 100)}%`,
            background: 'linear-gradient(90deg, var(--color-warning), var(--color-error))',
            borderRadius: 'var(--border-radius-sm)',
            boxShadow: '0 0 12px var(--color-error-glow)'
          }}
        />
      </div>
      <div className="detail-grid detail-grid-md">
        <DetailRow label="Used" value={formatBytes(diskUsage.used)} />
        <DetailRow label="Free" value={formatBytes(freeBytes)} />
        <DetailRow label="Capacity" value={formatBytes(diskUsage.total)} />
        <DetailRow label="Utilization" value={`${diskUsage.percent.toFixed(1)}%`} valueColor="var(--color-warning)" />
      </div>
    </div>
  );
};

export const renderProcessTable = (
  processes: Array<{ name: string; pid: number; cpu_percent?: number; memory_mb?: number }> | undefined,
  valueLabel: string,
  valueAccessor: (process: { cpu_percent?: number; memory_mb?: number }) => number | undefined
) => {
  if (!processes || processes.length === 0) {
    return (
      <div className="text-muted">
        No process data available yet.
      </div>
    );
  }

  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="data-table">
        <thead>
          <tr>
            <th>Process</th>
            <th>PID</th>
            <th>{valueLabel}</th>
          </tr>
        </thead>
        <tbody>
          {processes.slice(0, 10).map(process => {
            const value = valueAccessor(process);
            return (
              <tr key={`${process.name}-${process.pid}`}>
                <td style={{ color: 'var(--color-text-bright)' }}>{process.name}</td>
                <td style={{ color: 'var(--color-text)' }}>{process.pid}</td>
                <td style={{ color: 'var(--color-accent)' }}>
                  {value !== undefined ? value.toFixed(1) : '\u2014'}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

export const renderGrowthPatterns = (
  patterns: Array<{ process: string; growth_mb_per_hour: number; risk_level: string }> | undefined
) => {
  if (!patterns || patterns.length === 0) {
    return (
      <div className="text-muted">
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
