import { useMemo } from 'react';
import { CircuitBoard } from 'lucide-react';

import { formatMegabytes, formatPercentage } from '../../../shared/utils/formatters';
import type {
  DetailedMetrics,
  MetricHistory,
  GPUMetrics
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { buildSingleSeriesData } from './metricHelpers';

export interface GpuDetailViewProps {
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const GpuDetailView = ({ detailedMetrics, metricHistory, onBack }: GpuDetailViewProps) => {
  const gpuMetrics: GPUMetrics | null = detailedMetrics?.gpu_details ?? null;
  const gpuHistory = useMemo(() => buildSingleSeriesData(metricHistory?.gpu), [metricHistory?.gpu]);

  const headline = gpuMetrics?.summary
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
      subhead={subheadParts.length > 0 ? subheadParts.join(' \u2022 ') : undefined}
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
            {gpuMetrics.summary && (
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
                  ? `${gpuMetrics.summary.average_temperature_c.toFixed(1)}\u00B0C`
                  : '\u2014'
              }].map(stat => (
                <div key={stat.label} style={{
                  border: '1px solid var(--alpha-accent-20)',
                  borderRadius: 'var(--border-radius-md)',
                  padding: 'var(--spacing-md)',
                  background: 'rgba(0, 40, 0, 0.2)'
                }}>
                  <div className="detail-row-label" style={{ marginBottom: 'var(--spacing-xs)' }}>
                    {stat.label}
                  </div>
                  <div style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)', fontWeight: 600 }}>
                    {stat.value}
                  </div>
                </div>
              ))}
            </div>
            )}

            {gpuMetrics.errors && gpuMetrics.errors.length > 0 && (
              <div style={{
                border: '1px solid var(--color-warning)',
                borderRadius: 'var(--border-radius-md)',
                padding: 'var(--spacing-sm) var(--spacing-md)',
                color: 'var(--color-warning)',
                fontSize: 'var(--font-size-sm)'
              }}>
                {gpuMetrics.errors.join(' \u2022 ')}
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
                      <div>Temperature: <span style={{ color: 'var(--color-text-bright)' }}>{device.temperature_c != null ? `${device.temperature_c.toFixed(1)}\u00B0C` : '\u2014'}</span></div>
                      <div>Fan: <span style={{ color: 'var(--color-text-bright)' }}>{device.fan_speed_percent != null ? `${device.fan_speed_percent.toFixed(0)}%` : '\u2014'}</span></div>
                      <div>Power: <span style={{ color: 'var(--color-text-bright)' }}>{device.power_draw_w != null ? `${device.power_draw_w.toFixed(1)} W` : '\u2014'}</span></div>
                      <div>SM Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.sm_clock_mhz != null ? `${device.sm_clock_mhz.toFixed(0)} MHz` : '\u2014'}</span></div>
                      <div>Mem Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.memory_clock_mhz != null ? `${device.memory_clock_mhz.toFixed(0)} MHz` : '\u2014'}</span></div>
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
