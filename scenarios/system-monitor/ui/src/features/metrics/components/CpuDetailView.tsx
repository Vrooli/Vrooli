import { useMemo } from 'react';
import { Cpu } from 'lucide-react';

import { ProcessMonitor } from '../../monitoring/components/ProcessMonitor';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  MetricHistory
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import {
  formatTimeLabel,
  buildSingleSeriesData,
  renderProcessTable,
  formatInteger
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
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Load Profile</h3>
          <div className="card-subtitle">
            1m / 5m / 15m load average
          </div>
          <div style={{ display: 'flex', gap: 'var(--spacing-md)' }}>
            {loadAverage.slice(0, 3).map((value, index) => (
              <div key={`${value}-${index}`} className="detail-row">
                <span className="detail-row-label">
                  {index === 0 ? '1 min' : index === 1 ? '5 min' : '15 min'}
                </span>
                <span className="detail-row-value">{value.toFixed(2)}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Runtime Signals</h3>
            <div className="card-subtitle">
              Scheduler and goroutine metrics
            </div>
          </div>
          <div className="detail-grid detail-grid-md">
            <div className="detail-row">
              <span className="detail-row-label">Context Switches</span>
              <span style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-lg)' }}>{formatInteger(contextSwitches)}</span>
            </div>
            <div className="detail-row">
              <span className="detail-row-label">Goroutines</span>
              <span style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-lg)' }}>{formatInteger(goroutines)}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
        <div>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Top CPU Consumers</h3>
          <div className="card-subtitle">
            Processes ranked by CPU utilization
          </div>
        </div>
        {renderProcessTable(topProcesses, 'CPU %', process => process.cpu_percent)}
      </div>

      <div className="card" style={{ padding: 'var(--spacing-lg)' }}>
        <ProcessMonitor data={processMonitorData} collapsible={false} isExpanded={true} />
      </div>
    </MetricDetailLayout>
  );
};
