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
import { formatInteger, formatTimeLabel } from '../../../shared/utils/formatters';
import {
  buildSingleSeriesData,
  renderProcessTable
} from './metricHelpers';

export interface CpuDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  processMonitorData: ProcessMonitorData | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const CpuDetailView = ({ metrics, detailedMetrics, processMonitorData, metricHistory, onBack }: CpuDetailViewProps) => {
  const cpuUsage = detailedMetrics?.cpu_details?.usage ?? metrics?.cpu_usage ?? 0;
  const cpuData = useMemo(() => buildSingleSeriesData(metricHistory?.cpu), [metricHistory?.cpu]);
  const loadAverage = detailedMetrics?.cpu_details?.load_average ?? [];
  const contextSwitches = detailedMetrics?.cpu_details?.context_switches ?? 0;
  const goroutines = detailedMetrics?.cpu_details?.total_goroutines ?? 0;
  const topProcesses = detailedMetrics?.cpu_details?.top_processes;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatTimeLabel(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="CPU PERFORMANCE"
      icon={<Cpu size={22} />}
      headline={`${cpuUsage.toFixed(1)}% utilization`}
      subhead={subhead}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        style={{ padding: 'var(--spacing-lg)' }}
        data={cpuData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'CPU Usage', color: 'var(--color-accent)' }]}
        unit="%"
        yDomain={[0, 100]}
        valueFormatter={value => `${value.toFixed(1)}%`}
      />

      <div className="detail-grid detail-grid-lg" style={{ gap: 'var(--spacing-lg)' }}>
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Load Profile</h3>
          <div className="card-subtitle">
            1m / 5m / 15m load average
          </div>
          <div style={{ display: 'flex', gap: 'var(--spacing-md)' }}>
            {loadAverage.slice(0, 3).map((value, index) => (
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
            <DetailRow label="Context Switches" value={formatInteger(contextSwitches)} valueColor="var(--color-accent)" />
            <DetailRow label="Goroutines" value={formatInteger(goroutines)} valueColor="var(--color-accent)" />
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
        {renderProcessTable(topProcesses, 'CPU %', process => process.cpu_percent)}
      </div>

      <div className="card">
        <ProcessMonitor data={processMonitorData} collapsible={false} isExpanded={true} />
      </div>
    </MetricDetailLayout>
  );
};
