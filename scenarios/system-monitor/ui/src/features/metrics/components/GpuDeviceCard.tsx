import { formatMegabytes, formatPercentage } from '../../../shared/utils/formatters';
import type { GPUDeviceMetrics } from '../../../types';

interface GpuDeviceCardProps {
  device: GPUDeviceMetrics;
  variant?: 'detail' | 'expansion';
}

export const GpuDeviceCard = ({ device, variant = 'detail' }: GpuDeviceCardProps) => {
  const isExpansion = variant === 'expansion';

  const statsContent = (
    <>
      <div>Memory: <span style={isExpansion ? { color: 'var(--color-text-heading)' } : undefined} className={isExpansion ? undefined : 'text-bright'}>{formatMegabytes(device.memoryUsedMb)} / {formatMegabytes(device.memoryTotalMb)}</span></div>
      <div>Memory Util: <span style={isExpansion ? { color: 'var(--color-text-heading)' } : undefined} className={isExpansion ? undefined : 'text-bright'}>{formatPercentage(device.memoryUtilizationPercent)}</span></div>
      <div>Temp: <span style={isExpansion ? { color: 'var(--color-text-heading)' } : undefined} className={isExpansion ? undefined : 'text-bright'}>{device.temperatureC != null ? `${device.temperatureC.toFixed(1)}\u00B0C` : '—'}</span></div>
      <div>Fan: <span style={isExpansion ? { color: 'var(--color-text-heading)' } : undefined} className={isExpansion ? undefined : 'text-bright'}>{device.fanSpeedPercent != null ? `${device.fanSpeedPercent.toFixed(0)}%` : '—'}</span></div>
      <div>Power: <span style={isExpansion ? { color: 'var(--color-text-heading)' } : undefined} className={isExpansion ? undefined : 'text-bright'}>{device.powerDrawW != null ? `${device.powerDrawW.toFixed(1)} W` : '—'}</span></div>
      <div>SM Clock: <span style={isExpansion ? { color: 'var(--color-text-heading)' } : undefined} className={isExpansion ? undefined : 'text-bright'}>{device.smClockMhz != null ? `${device.smClockMhz.toFixed(0)} MHz` : '—'}</span></div>
      <div>Mem Clock: <span style={isExpansion ? { color: 'var(--color-text-heading)' } : undefined} className={isExpansion ? undefined : 'text-bright'}>{device.memoryClockMhz != null ? `${device.memoryClockMhz.toFixed(0)} MHz` : '—'}</span></div>
    </>
  );

  const processSection = device.processes && device.processes.length > 0 && (
    <div data-sm-style="sm-style-9ed1054ccd">
      <div
        className={isExpansion ? undefined : 'text-dim-xs'}
        style={isExpansion ? { color: 'var(--color-text-secondary)', fontSize: 'var(--text-xs)', marginBottom: 'var(--spacing-xs)' } : { marginBottom: '2px' }}
      >
        {isExpansion ? 'Active Processes' : 'Processes'}
      </div>
      <div className={isExpansion ? undefined : 'flex-col-gap-sm'} style={isExpansion ? { display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xxs)' } : undefined}>
        {device.processes.slice(0, isExpansion ? 5 : undefined).map(process => (
          <div
            key={`${device.uuid || device.index}-${process.pid}`}
            className={isExpansion ? undefined : 'flex-row-between text-bright text-xs'}
            style={isExpansion ? {
              display: 'flex',
              justifyContent: 'space-between',
              fontSize: 'var(--text-xs)',
              color: 'var(--color-text-heading)'
            } : undefined}
          >
            <span>{process.processName} ({process.pid})</span>
            <span>{formatMegabytes(process.memoryUsedMb)}</span>
          </div>
        ))}
      </div>
    </div>
  );

  if (isExpansion) {
    return (
      <div data-sm-style="sm-style-8dd0eabdeb">
        <div data-sm-style="sm-style-512805d4a2">
          <span data-sm-style="sm-style-6e97bc6917">
            {device.name} (GPU {device.index})
          </span>
          <span data-sm-style="sm-style-0419277d6a">
            {device.utilizationPercent?.toFixed(1) ?? '—'}%
          </span>
        </div>

        <div data-sm-style="sm-style-7fdc0d68d5">
          {statsContent}
        </div>

        {processSection}
      </div>
    );
  }

  return (
    <div className="gpu-device-card">
      <div className="flex-row-between" data-sm-style="sm-style-c08663b577">
        <span className="text-bright font-semibold">
          {device.name} (GPU {device.index})
        </span>
        <span className="text-accent text-sm">
          {device.utilizationPercent.toFixed(1)}%
        </span>
      </div>

      <div className="gpu-stats-grid">
        {statsContent}
      </div>

      {processSection}
    </div>
  );
};
