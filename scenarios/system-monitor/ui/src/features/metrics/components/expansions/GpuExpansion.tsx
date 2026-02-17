import type { GPUCardDetails } from '../../../../types';
import { formatMegabytes, formatPercentage } from '../../../../shared/utils/formatters';

interface GpuExpansionProps {
  details: GPUCardDetails;
}

export const GpuExpansion = ({ details }: GpuExpansionProps) => {
  const gpuMetrics = details?.metrics;
  if (!gpuMetrics) {
    return (
      <div style={{ marginTop: 'var(--spacing-md)', color: 'var(--color-text-dim)' }}>
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
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Devices:</span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>{summary.device_count}</span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Average Utilization:</span>
            <span className="detail-value" style={{ color: 'var(--color-accent)' }}>{summary.average_utilization_percent?.toFixed(1) ?? '\u2014'}%</span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Memory:</span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {summary.used_memory_mb?.toFixed(0) ?? '\u2014'} / {summary.total_memory_mb?.toFixed(0) ?? '\u2014'} MB
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Avg Temp:</span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {summary.device_count > 0 && summary.average_temperature_c > 0 ? `${summary.average_temperature_c.toFixed(1)}\u00B0C` : '\u2014'}
            </span>
          </div>
        </div>
      )}

      {gpuMetrics.driver_version && (
        <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
          Driver Version: <span style={{ color: 'var(--color-text-bright)' }}>{gpuMetrics.driver_version}</span>
        </div>
      )}

      {gpuMetrics.primary_model && (
        <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
          Primary Model: <span style={{ color: 'var(--color-text-bright)' }}>{gpuMetrics.primary_model}</span>
        </div>
      )}

      {gpuErrors && gpuErrors.length > 0 && (
        <div style={{
          border: '1px solid var(--color-warning)',
          padding: 'var(--spacing-sm) var(--spacing-md)',
          borderRadius: 'var(--border-radius-md)',
          color: 'var(--color-warning)',
          fontSize: 'var(--font-size-sm)'
        }}>
          {gpuErrors.join(' \u2022 ')}
        </div>
      )}

      {(devices?.length ?? 0) > 0 ? (
        <div className="device-list flex-col-gap-sm">
          {(devices ?? []).map(device => (
            <div key={device.uuid ?? device.index} style={{
              border: '1px solid var(--color-surface)',
              borderRadius: 'var(--border-radius-md)',
              padding: 'var(--spacing-md)',
              background: 'var(--surface-card-accent)'
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
                  {device.utilization_percent?.toFixed(1) ?? '\u2014'}%
                </span>
              </div>

              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
                gap: 'var(--spacing-sm)',
                fontSize: 'var(--font-size-xs)',
                color: 'var(--color-text-dim)'
              }}>
                <div>Memory: <span style={{ color: 'var(--color-text-bright)' }}>{formatMegabytes(device.memory_used_mb)} / {formatMegabytes(device.memory_total_mb)}</span></div>
                <div>Memory Util: <span style={{ color: 'var(--color-text-bright)' }}>{formatPercentage(device.memory_utilization_percent)}</span></div>
                <div>Temp: <span style={{ color: 'var(--color-text-bright)' }}>{device.temperature_c != null ? `${device.temperature_c.toFixed(1)}\u00B0C` : '\u2014'}</span></div>
                <div>Fan: <span style={{ color: 'var(--color-text-bright)' }}>{device.fan_speed_percent != null ? `${device.fan_speed_percent.toFixed(0)}%` : '\u2014'}</span></div>
                <div>Power: <span style={{ color: 'var(--color-text-bright)' }}>{device.power_draw_w != null ? `${device.power_draw_w.toFixed(1)} W` : '\u2014'}</span></div>
                <div>SM Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.sm_clock_mhz != null ? `${device.sm_clock_mhz.toFixed(0)} MHz` : '\u2014'}</span></div>
                <div>Mem Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.memory_clock_mhz != null ? `${device.memory_clock_mhz.toFixed(0)} MHz` : '\u2014'}</span></div>
              </div>

              {device.processes && device.processes.length > 0 && (
                <div style={{ marginTop: 'var(--spacing-sm)' }}>
                  <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', marginBottom: 'var(--spacing-xs)' }}>
                    Active Processes
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xxs)' }}>
                    {device.processes.slice(0, 5).map(process => (
                      <div key={`${device.uuid}-${process.pid}`} style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        fontSize: 'var(--font-size-xs)',
                        color: 'var(--color-text-bright)'
                      }}>
                        <span>{process.process_name} ({process.pid})</span>
                        <span>{formatMegabytes(process.memory_used_mb)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
          No GPU devices detected.
        </div>
      )}

      {details.lastUpdated && (
        <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
          Updated {new Date(details.lastUpdated).toLocaleTimeString()}
        </div>
      )}
    </div>
  );
};
