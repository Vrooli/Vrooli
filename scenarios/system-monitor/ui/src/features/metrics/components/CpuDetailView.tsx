import { useMemo } from 'react';
import { Cpu } from 'lucide-react';

import { ProcessMonitor } from '../../monitoring/components/ProcessMonitor';
import { DetailRow } from '../../../shared/components/DetailRow';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  MetricHistory
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { formatInteger, formatProtoTimestamp } from '../../../shared/utils/formatters';
import { buildSingleSeriesData } from '../../../shared/utils/chartData';
import { renderProcessTable } from './MetricRenderHelpers';

export interface CpuDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  processMonitorData: ProcessMonitorData | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const CpuDetailView = ({ metrics, detailedMetrics, processMonitorData, metricHistory, onBack }: CpuDetailViewProps) => {
  const cpuUsage = detailedMetrics?.cpuDetails?.usage ?? (metrics?.cpu?.state?.case === 'measured' ? metrics.cpu.state.value : undefined);
  const cpuData = useMemo(() => buildSingleSeriesData(metricHistory?.cpu), [metricHistory?.cpu]);
  const loadAverage = detailedMetrics?.cpuDetails?.loadAverage ?? [];
  const contextSwitches = detailedMetrics?.cpuDetails?.contextSwitches;
  const goroutines = detailedMetrics?.cpuDetails?.totalGoroutines;
  const topProcesses = detailedMetrics?.cpuDetails?.topProcesses;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatProtoTimestamp(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="CPU PERFORMANCE"
      icon={<Cpu size={22} />}
      headline={cpuUsage === undefined ? 'Utilization not measured' : `${cpuUsage.toFixed(1)}% utilization`}
      subhead={subhead}
      onBack={onBack}
    >
      <MetricLineChart
        status={metricHistory === null ? 'loading' : 'ready'}
        seriesLabel="CPU"
        className="card"
        data-sm-style="sm-style-a796e75e8f"
        data={cpuData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'CPU Usage', color: 'var(--color-primary)' }]}
        unit="%"
        yDomain={[0, 100]}
        valueFormatter={value => `${value.toFixed(1)}%`}
      />

      <div className="detail-grid detail-grid-lg" data-sm-style="sm-style-f383142193">
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Load Profile</h3>
          <div className="card-subtitle">
            1m / 5m / 15m load average
          </div>
          <div data-sm-style="sm-style-09ba01c6ba">
            {loadAverage.slice(0, 3).map((value: number, index: number) => (
              <DetailRow
                key={`${value}-${index}`}
                label={index === 0 ? '1 min' : index === 1 ? '5 min' : '15 min'}
                value={value.toFixed(2)}
              />
            ))}
          </div>
        </div>

        <div className="card flex-col-gap-md">
          <div>
            <h3 className="section-heading">Runtime Signals</h3>
            <div className="card-subtitle">
              Scheduler and goroutine metrics
            </div>
          </div>
          <div className="detail-grid detail-grid-md">
            <DetailRow label="Context Switches" value={contextSwitches === undefined ? '—' : formatInteger(Number(contextSwitches))} valueColor="var(--color-primary)" />
            <DetailRow label="Goroutines" value={goroutines === undefined ? '—' : formatInteger(goroutines)} valueColor="var(--color-primary)" />
          </div>
        </div>
      </div>

      <div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Top CPU Consumers</h3>
          <div className="card-subtitle">
            Processes ranked by CPU utilization
          </div>
        </div>
        {renderProcessTable(topProcesses, 'CPU %', process => process.cpuPercent)}
      </div>

      <div className="card">
        <ProcessMonitor data={processMonitorData} collapsible={false} isExpanded={true} />
      </div>
    </MetricDetailLayout>
  );
};
