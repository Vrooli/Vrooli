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
      <div data-sm-style="sm-style-8b5a895ea0">
        GPU metrics unavailable. Ensure compatible NVIDIA drivers are installed.
      </div>
    );
  }

  const { summary, devices, errors: gpuErrors } = gpuMetrics;

  return (
    <div className="metric-details flex-col-gap-md" data-sm-style="sm-style-323fdcc1e0">
      {summary && (
        <div className="metric-grid-auto">
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">Devices:</span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">{summary.deviceCount}</span>
          </div>
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">Average Utilization:</span>
            <span className="detail-value" data-sm-style="sm-style-392c7463c7">{summary.averageUtilizationPercent?.toFixed(1) ?? '—'}%</span>
          </div>
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">Memory:</span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {summary.usedMemoryMb?.toFixed(0) ?? '—'} / {summary.totalMemoryMb?.toFixed(0) ?? '—'} MB
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">Avg Temp:</span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {summary.deviceCount > 0 && summary.averageTemperatureC > 0 ? `${summary.averageTemperatureC.toFixed(1)}\u00B0C` : '—'}
            </span>
          </div>
        </div>
      )}

      {gpuMetrics.driverVersion && (
        <div data-sm-style="sm-style-3078a903f6">
          Driver Version: <span data-sm-style="sm-style-dbed1e5364">{gpuMetrics.driverVersion}</span>
        </div>
      )}

      {gpuMetrics.primaryModel && (
        <div data-sm-style="sm-style-3078a903f6">
          Primary Model: <span data-sm-style="sm-style-dbed1e5364">{gpuMetrics.primaryModel}</span>
        </div>
      )}

      {gpuErrors && gpuErrors.length > 0 && (
        <div data-sm-style="sm-style-345204f43b">
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
        <div data-sm-style="sm-style-3078a903f6">
          No GPU devices detected.
        </div>
      )}

      {details.lastUpdated && (
        <div data-sm-style="sm-style-39e2159503">
          Updated {formatTime(details.lastUpdated)}
        </div>
      )}
    </div>
  );
};
