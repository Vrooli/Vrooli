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
      <div className="flex-col-gap-lg">
        <MetricLineChart
          data={gpuHistory}
          lines={[{ dataKey: 'value', name: 'Utilization', color: 'var(--color-info)', strokeWidth: 2 }]}
          unit="%"
          yDomain={[0, 100]}
          valueFormatter={formatPercentage}
        />

        {gpuMetrics ? (
          <div className="flex-col-gap-lg">
            {gpuMetrics.summary && (
            <div className="metric-grid-auto">
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
                <div key={stat.label} className="pool-card">
                  <div className="detail-row-label mb-sm">
                    {stat.label}
                  </div>
                  <div className="text-bright text-lg font-semibold">
                    {stat.value}
                  </div>
                </div>
              ))}
            </div>
            )}

            {gpuMetrics.errors && gpuMetrics.errors.length > 0 && (
              <div className="warning-box">
                {gpuMetrics.errors.join(' \u2022 ')}
              </div>
            )}

            <div className="flex-col-gap-md">
              {gpuMetrics.devices.length > 0 ? (
                gpuMetrics.devices.map(device => (
                  <div key={device.uuid || device.index} className="gpu-device-card">
                    <div className="flex-row-between" style={{ marginBottom: 'var(--spacing-sm)' }}>
                      <span className="text-bright font-semibold">
                        {device.name} (GPU {device.index})
                      </span>
                      <span className="text-accent text-sm">
                        {device.utilization_percent.toFixed(1)}%
                      </span>
                    </div>

                    <div className="gpu-stats-grid">
                      <div>Memory: <span className="text-bright">{formatMegabytes(device.memory_used_mb)} / {formatMegabytes(device.memory_total_mb)}</span></div>
                      <div>Memory Util: <span className="text-bright">{formatPercentage(device.memory_utilization_percent)}</span></div>
                      <div>Temperature: <span className="text-bright">{device.temperature_c != null ? `${device.temperature_c.toFixed(1)}\u00B0C` : '\u2014'}</span></div>
                      <div>Fan: <span className="text-bright">{device.fan_speed_percent != null ? `${device.fan_speed_percent.toFixed(0)}%` : '\u2014'}</span></div>
                      <div>Power: <span className="text-bright">{device.power_draw_w != null ? `${device.power_draw_w.toFixed(1)} W` : '\u2014'}</span></div>
                      <div>SM Clock: <span className="text-bright">{device.sm_clock_mhz != null ? `${device.sm_clock_mhz.toFixed(0)} MHz` : '\u2014'}</span></div>
                      <div>Mem Clock: <span className="text-bright">{device.memory_clock_mhz != null ? `${device.memory_clock_mhz.toFixed(0)} MHz` : '\u2014'}</span></div>
                    </div>

                    {device.processes && device.processes.length > 0 && (
                      <div style={{ marginTop: 'var(--spacing-sm)' }}>
                        <div className="text-dim-xs" style={{ marginBottom: '2px' }}>
                          Processes
                        </div>
                        <div className="flex-col-gap-sm">
                          {device.processes.map(process => (
                            <div key={`${device.uuid || device.index}-${process.pid}`} className="flex-row-between text-bright text-xs">
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
                <div className="text-dim-sm">
                  No GPU devices detected.
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="text-dim-sm">
            GPU metrics unavailable on this host.
          </div>
        )}
      </div>
    </MetricDetailLayout>
  );
};
