import type { DiskCardDetails } from '../../../../types';
import { formatBytes, formatTime } from '../../../../shared/utils/formatters';

interface DiskExpansionProps {
  details: DiskCardDetails;
}

export const DiskExpansion = ({ details }: DiskExpansionProps) => {
  const diskUsage = details.diskUsage;
  const total = Number(diskUsage?.total ?? 0);
  const used = Number(diskUsage?.used ?? 0);
  const freeBytes = Number.isFinite(total - used) ? total - used : undefined;

  return (
    <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
      <div className="metric-grid-auto" style={{ marginBottom: 'var(--spacing-md)' }}>
        <div className="detail-item">
          <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
            Total Capacity:
          </span>
          <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
            {formatBytes(total)}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
            Used:
          </span>
          <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
            {formatBytes(used)}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
            Free:
          </span>
          <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
            {formatBytes(freeBytes)}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
            Utilization:
          </span>
          <span className="detail-value" style={{ color: 'var(--color-warning)' }}>
            {diskUsage?.percent?.toFixed(1) ?? '—'}%
          </span>
        </div>
      </div>

      {details.storageIO && (
        <div className="metric-grid-auto" style={{ marginBottom: 'var(--spacing-sm)' }}>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
              Disk Queue Depth:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {details.storageIO.diskQueueDepth?.toFixed(2) ?? '—'}
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
              I/O Wait:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {details.storageIO.ioWaitPercent?.toFixed(1) ?? '—'}%
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
              Read Throughput:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {details.storageIO.readMbPerSec?.toFixed(2) ?? '—'} MB/s
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
              Write Throughput:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {details.storageIO.writeMbPerSec?.toFixed(2) ?? '—'} MB/s
            </span>
          </div>
        </div>
      )}

      {details.lastUpdated && (
        <div style={{
          marginTop: 'var(--spacing-sm)',
          color: 'var(--color-text-dim)',
          fontSize: 'var(--font-size-xs)'
        }}>
          Updated {formatTime(details.lastUpdated)}
        </div>
      )}
    </div>
  );
};
