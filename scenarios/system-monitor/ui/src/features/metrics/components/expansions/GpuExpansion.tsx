import type { GPUCardDetails } from '../../../../types';
import { formatTime } from '../../../../shared/utils/formatters';
import { GpuDeviceCard } from '../GpuDeviceCard';

interface GpuExpansionProps {
  details: GPUCardDetails;
}

export const GpuExpansion = ({ details }: GpuExpansionProps) => {
  const gpuMetrics = details?.metrics;
  if (!gpuMetrics) {
    return (
      <div style={{ marginTop: 'var(--spacing-md)', color: 'var(--color-text-secondary)' }}>
        GPU metrics unavailable. Ensure compatible NVIDIA drivers are installed.
      </div>
    );
  }

  const { summary, devices, errors: gpuErrors } = gpuMetrics;

  return (
    <div className="metric-details flex-col-gap-md" style={{ marginTop: 'var(--spacing-md)' }}>
      {summary && (
        <div className="metric-grid-auto">
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>Devices:</span>
            <span className="detail-value" style={{ color: 'var(--color-text-heading)' }}>{summary.deviceCount}</span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>Average Utilization:</span>
            <span className="detail-value" style={{ color: 'var(--color-primary)' }}>{summary.averageUtilizationPercent?.toFixed(1) ?? '—'}%</span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>Memory:</span>
            <span className="detail-value" style={{ color: 'var(--color-text-heading)' }}>
              {summary.usedMemoryMb?.toFixed(0) ?? '—'} / {summary.totalMemoryMb?.toFixed(0) ?? '—'} MB
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>Avg Temp:</span>
            <span className="detail-value" style={{ color: 'var(--color-text-heading)' }}>
              {summary.deviceCount > 0 && summary.averageTemperatureC > 0 ? `${summary.averageTemperatureC.toFixed(1)}\u00B0C` : '—'}
            </span>
          </div>
        </div>
      )}

      {gpuMetrics.driverVersion && (
        <div style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
          Driver Version: <span style={{ color: 'var(--color-text-heading)' }}>{gpuMetrics.driverVersion}</span>
        </div>
      )}

      {gpuMetrics.primaryModel && (
        <div style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
          Primary Model: <span style={{ color: 'var(--color-text-heading)' }}>{gpuMetrics.primaryModel}</span>
        </div>
      )}

      {gpuErrors && gpuErrors.length > 0 && (
        <div style={{
          border: '1px solid var(--color-warning)',
          padding: 'var(--spacing-sm) var(--spacing-md)',
          borderRadius: 'var(--radius-md)',
          color: 'var(--color-warning)',
          fontSize: 'var(--text-sm)'
        }}>
          {gpuErrors.join(' \u2022 ')}
        </div>
      )}

      {(devices?.length ?? 0) > 0 ? (
        <div className="device-list flex-col-gap-sm">
          {(devices ?? []).map(device => (
            <GpuDeviceCard key={device.uuid ?? device.index} device={device} variant="expansion" />
          ))}
        </div>
      ) : (
        <div style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
          No GPU devices detected.
        </div>
      )}

      {details.lastUpdated && (
        <div style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-xs)' }}>
          Updated {formatTime(details.lastUpdated)}
        </div>
      )}
    </div>
  );
};
