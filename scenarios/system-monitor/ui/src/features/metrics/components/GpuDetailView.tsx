import { useMemo } from 'react';
import { CircuitBoard } from 'lucide-react';

import { formatPercentage } from '../../../shared/utils/formatters';
import type {
  DetailedMetrics,
  MetricHistory,
  GPUMetrics
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { buildSingleSeriesData } from '../../../shared/utils/chartData';
import { GpuDeviceCard } from './GpuDeviceCard';

export interface GpuDetailViewProps {
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const GpuDetailView = ({ detailedMetrics, metricHistory, onBack }: GpuDetailViewProps) => {
  const gpuMetrics: GPUMetrics | null = detailedMetrics?.gpuDetails ?? null;
  const gpuHistory = useMemo(() => buildSingleSeriesData(metricHistory?.gpu), [metricHistory?.gpu]);

  const headline = gpuMetrics?.summary
    ? `${gpuMetrics.summary.averageUtilizationPercent.toFixed(1)}% Avg`
    : 'Awaiting telemetry';

  const subheadParts: string[] = [];
  if (gpuMetrics?.driverVersion) {
    subheadParts.push(`Driver ${gpuMetrics.driverVersion}`);
  }
  if (gpuMetrics?.primaryModel) {
    subheadParts.push(gpuMetrics.primaryModel);
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
        status={metricHistory === null ? 'loading' : 'ready'}
        seriesLabel="GPU"
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
                value: String(gpuMetrics.summary.deviceCount)
              }, {
                label: 'Average Utilization',
                value: `${gpuMetrics.summary.averageUtilizationPercent.toFixed(1)}%`
              }, {
                label: 'Memory Used',
                value: `${gpuMetrics.summary.usedMemoryMb.toFixed(0)} / ${gpuMetrics.summary.totalMemoryMb.toFixed(0)} MB`
              }, {
                label: 'Average Temperature',
                value: gpuMetrics.summary.deviceCount > 0 && gpuMetrics.summary.averageTemperatureC > 0
                  ? `${gpuMetrics.summary.averageTemperatureC.toFixed(1)}\u00B0C`
                  : '—'
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
                  <GpuDeviceCard key={device.uuid || device.index} device={device} variant="detail" />
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
