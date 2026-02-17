import { useMemo } from 'react';
import { MemoryStick } from 'lucide-react';

import { formatBytes } from '../../../shared/utils/formatters';
import type {
  MetricsResponse,
  DetailedMetrics,
  MetricHistory
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import {
  formatTimeLabel,
  buildSingleSeriesData,
  renderProcessTable,
  renderGrowthPatterns
} from './metricHelpers';

export interface MemoryDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const MemoryDetailView = ({ metrics, detailedMetrics, metricHistory, onBack }: MemoryDetailViewProps) => {
  const memoryUsage = detailedMetrics?.memory_details?.usage ?? metrics?.memory_usage ?? 0;
  const memoryData = useMemo(() => buildSingleSeriesData(metricHistory?.memory), [metricHistory?.memory]);
  const memoryDetails = detailedMetrics?.memory_details;

  const swapUsage = memoryDetails?.swap_usage;
  const growthPatterns = memoryDetails?.growth_patterns;
  const topProcesses = memoryDetails?.top_processes;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatTimeLabel(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="MEMORY UTILIZATION"
      icon={<MemoryStick size={22} />}
      headline={`${memoryUsage.toFixed(1)}% used`}
      subhead={subhead}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        style={{ padding: 'var(--spacing-lg)' }}
        data={memoryData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'Memory Usage', color: 'var(--color-warning)' }]}
        unit="%"
        yDomain={[0, 100]}
        valueFormatter={value => `${value.toFixed(1)}%`}
      />

      <div className="detail-grid detail-grid-lg" style={{ gap: 'var(--spacing-lg)' }}>
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Swap Activity</h3>
          {swapUsage ? (
            <div className="detail-grid detail-grid-md">
              <div className="detail-row">
                <span className="detail-row-label">Swap Used</span>
                <span className="detail-row-value">{formatBytes(swapUsage.used)}</span>
              </div>
              <div className="detail-row">
                <span className="detail-row-label">Swap Total</span>
                <span className="detail-row-value">{formatBytes(swapUsage.total)}</span>
              </div>
              <div className="detail-row">
                <span className="detail-row-label">Utilization</span>
                <span style={{ color: 'var(--color-warning)', fontSize: 'var(--font-size-lg)' }}>{swapUsage.percent.toFixed(1)}%</span>
              </div>
            </div>
          ) : (
            <div className="text-muted">
              Swap metrics unavailable.
            </div>
          )}
        </div>

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Growth Patterns</h3>
          <div className="card-subtitle">
            Heaviest allocators during the observation window
          </div>
          {renderGrowthPatterns(growthPatterns)}
        </div>
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
        <div>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Top Memory Consumers</h3>
          <div className="card-subtitle">
            Processes ranked by resident set size
          </div>
        </div>
        {renderProcessTable(topProcesses, 'Memory (MB)', process => process.memory_mb)}
      </div>
    </MetricDetailLayout>
  );
};
