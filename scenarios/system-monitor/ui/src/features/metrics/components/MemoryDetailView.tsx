import { useMemo } from 'react';
import { MemoryStick } from 'lucide-react';

import { DetailRow } from '../../../shared/components/DetailRow';
import { formatBytes } from '../../../shared/utils/formatters';
import type {
  MetricsResponse,
  DetailedMetrics,
  MetricHistory
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { formatTimeLabel } from '../../../shared/utils/formatters';
import {
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
        data={memoryData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'Memory Usage', color: 'var(--color-warning)' }]}
        unit="%"
        yDomain={[0, 100]}
        valueFormatter={value => `${value.toFixed(1)}%`}
      />

      <div className="detail-grid detail-grid-lg" style={{ gap: 'var(--spacing-lg)' }}>
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Swap Activity</h3>
          {swapUsage ? (
            <div className="detail-grid detail-grid-md">
              <DetailRow label="Swap Used" value={formatBytes(swapUsage.used)} />
              <DetailRow label="Swap Total" value={formatBytes(swapUsage.total)} />
              <DetailRow label="Utilization" value={`${swapUsage.percent.toFixed(1)}%`} valueColor="var(--color-warning)" />
            </div>
          ) : (
            <div className="text-muted">
              Swap metrics unavailable.
            </div>
          )}
        </div>

        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Growth Patterns</h3>
          <div className="card-subtitle">
            Heaviest allocators during the observation window
          </div>
          {renderGrowthPatterns(growthPatterns)}
        </div>
      </div>

      <div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Top Memory Consumers</h3>
          <div className="card-subtitle">
            Processes ranked by resident set size
          </div>
        </div>
        {renderProcessTable(topProcesses, 'Memory (MB)', process => process.memory_mb)}
      </div>
    </MetricDetailLayout>
  );
};
