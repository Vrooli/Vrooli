import type { DiskInfo } from '../../../types';
import { DetailRow } from '../../../shared/components/DetailRow';
import { formatBytes } from '../../../shared/utils/formatters';

// ── Render Helpers ─────────────────────────────────────────────────────────

export const buildDiskUsageCard = (
  diskUsage?: DiskInfo,
  options?: { title?: string; subtitle?: string }
) => {
  if (!diskUsage) {
    return (
      <div className="card" data-sm-style="sm-style-a796e75e8f">
        <h3 className="section-heading">{options?.title ?? 'Disk Utilization'}</h3>
        <div className="text-muted">
          Disk usage metrics are unavailable.
        </div>
      </div>
    );
  }

  const freeBytes = Number(diskUsage.total) - Number(diskUsage.used);

  return (
    <div className="card flex-col-gap-md" data-sm-style="sm-style-a796e75e8f">
      <div>
        <h3 className="section-heading">{options?.title ?? 'Disk Utilization'}</h3>
        <div className="card-subtitle">
          {options?.subtitle ?? 'Current usage across monitored volumes'}
        </div>
      </div>
      <div className="progress-bar progress-bar-lg" data-sm-style="sm-style-dce622b2b5">
        <div
          className="progress-fill"
          style={{
            width: `${Math.min(Math.max(diskUsage.percent, 0), 100)}%`,
            background: 'linear-gradient(90deg, var(--color-warning), var(--color-error))',
            borderRadius: 'var(--radius-sm)',
            boxShadow: '0 0 12px var(--color-error-muted)'
          }}
        />
      </div>
      <div className="detail-grid detail-grid-md">
        <DetailRow label="Used" value={formatBytes(Number(diskUsage.used))} />
        <DetailRow label="Free" value={formatBytes(freeBytes)} />
        <DetailRow label="Capacity" value={formatBytes(Number(diskUsage.total))} />
        <DetailRow label="Utilization" value={`${diskUsage.percent.toFixed(1)}%`} valueColor="var(--color-warning)" />
      </div>
    </div>
  );
};

export const renderProcessTable = (
  processes: Array<{ name: string; pid: number; cpuPercent?: number; memoryMb?: number }> | undefined,
  valueLabel: string,
  valueAccessor: (process: { cpuPercent?: number; memoryMb?: number }) => number | undefined
) => {
  if (!processes || processes.length === 0) {
    return (
      <div className="text-muted">
        No process data available yet.
      </div>
    );
  }

  return (
    <div data-sm-style="sm-style-d383f0755e">
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
                <td data-sm-style="sm-style-dbed1e5364">{process.name}</td>
                <td data-sm-style="sm-style-bb03b2fa99">{process.pid}</td>
                <td data-sm-style="sm-style-392c7463c7">
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

export const renderGrowthPatterns = (
  patterns: Array<{ process: string; growthMbPerHour: number; riskLevel: string }> | undefined
) => {
  if (!patterns || patterns.length === 0) {
    return (
      <div className="text-muted">
        No anomalous growth patterns detected.
      </div>
    );
  }

  return (
    <div data-sm-style="sm-style-f42c3fdadb">
      {patterns.slice(0, 8).map(pattern => (
        <div
          key={`${pattern.process}-${pattern.growthMbPerHour}`}
          data-sm-style="sm-style-c900eb7d7c"
        >
          <span data-sm-style="sm-style-bb03b2fa99">{pattern.process}</span>
          <span className={`risk-level-${pattern.riskLevel.toLowerCase()}`}>
            {pattern.growthMbPerHour.toFixed(1)} MB/hr ({pattern.riskLevel})
          </span>
        </div>
      ))}
    </div>
  );
};
