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
  processes: Array<{ name: string; pid: number; cpuPercent?: number; memoryMb?: number; majorFaultsPerSecond?: number; cpuSeconds?: number }> | undefined,
  valueLabel: string,
  valueAccessor: (process: { cpuPercent?: number; memoryMb?: number; majorFaultsPerSecond?: number; cpuSeconds?: number }) => number | undefined
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
