import { useMemo } from 'react';
import { MemoryStick } from 'lucide-react';

import { DetailRow } from '../../../shared/components/DetailRow';
import { formatBytes, formatProtoTimestamp } from '../../../shared/utils/formatters';
import type {
  MetricsResponse,
  DetailedMetrics,
  MetricHistory
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { buildSingleSeriesData } from '../../../shared/utils/chartData';
import { renderProcessTable, renderGrowthPatterns } from './MetricRenderHelpers';

export interface MemoryDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const MemoryDetailView = ({ metrics, detailedMetrics, metricHistory, onBack }: MemoryDetailViewProps) => {
  const memoryUsage = detailedMetrics?.memoryDetails?.usage ?? (metrics?.memory?.state?.case === 'measured' ? metrics.memory.state.value : undefined);
  const memoryData = useMemo(() => buildSingleSeriesData(metricHistory?.memory), [metricHistory?.memory]);
  const memoryDetails = detailedMetrics?.memoryDetails;

  const swapUsage = memoryDetails?.swapUsage;
  const growthPatterns = memoryDetails?.growthPatterns;
  const topProcesses = memoryDetails?.topProcesses;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatProtoTimestamp(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="MEMORY UTILIZATION"
      icon={<MemoryStick size={22} />}
      headline={memoryUsage === undefined ? 'Utilization not measured' : `${memoryUsage.toFixed(1)}% used`}
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

      <div className="detail-grid detail-grid-lg" data-sm-style="sm-style-f383142193">
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Swap Activity</h3>
          {swapUsage ? (
            <div className="detail-grid detail-grid-md">
              <DetailRow label="Swap Used" value={formatBytes(Number(swapUsage.used))} />
              <DetailRow label="Swap Total" value={formatBytes(Number(swapUsage.total))} />
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
        {renderProcessTable(topProcesses, 'Memory (MB)', process => process.memoryMb)}
      </div>
    </MetricDetailLayout>
  );
};
