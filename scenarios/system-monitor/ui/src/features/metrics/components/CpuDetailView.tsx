import { useMemo } from 'react';
import { Cpu } from 'lucide-react';

import { ProcessMonitor } from '../../monitoring/components/ProcessMonitor';
import { DetailRow } from '../../../shared/components/DetailRow';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  MetricHistory,
  MetricValue
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { formatProtoTimestamp } from '../../../shared/utils/formatters';
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
  const topProcesses = detailedMetrics?.cpuDetails?.topProcesses;
  const topCPUSecondsProcesses = detailedMetrics?.cpuDetails?.topCpuSecondsProcesses;
  const cpuDetails = detailedMetrics?.cpuDetails;
  const modeHistory = useMemo(() => mergeSeries(metricHistory?.cpuModeIowait, metricHistory?.cpuModeSteal), [metricHistory?.cpuModeIowait, metricHistory?.cpuModeSteal]);
  const stallHistory = useMemo(() => mergeSeries(metricHistory?.cpuStallSome, metricHistory?.cpuStallFull), [metricHistory?.cpuStallSome, metricHistory?.cpuStallFull]);

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatProtoTimestamp(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="CPU PERFORMANCE"
      icon={<Cpu size={22} />}
      headline={cpuVerdict(cpuDetails, cpuUsage)}
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

      <div className="detail-grid detail-grid-lg">
        <div className="card" data-testid="cpu-mode-chart">
          <h3 className="section-heading">Mode Breakdown History</h3>
          <MetricLineChart
            status={metricHistory === null ? 'loading' : 'ready'}
            seriesLabel="CPU modes"
            data={modeHistory}
            lines={[
              { dataKey: 'iowait', name: 'I/O wait', color: 'var(--color-warning)' },
              { dataKey: 'steal', name: 'Steal', color: 'var(--color-error)' }
            ]}
            unit="%"
            yDomain={[0, 100]}
          />
        </div>
        <div className="card" data-testid="cpu-stall-chart">
          <h3 className="section-heading">Stall History</h3>
          <MetricLineChart
            status={metricHistory === null ? 'loading' : 'ready'}
            seriesLabel="CPU stall"
            data={stallHistory}
            lines={[
              { dataKey: 'some', name: 'Some', color: 'var(--color-warning)' },
              { dataKey: 'full', name: 'Full', color: 'var(--color-error)' }
            ]}
            unit="%"
            yDomain={[0, 100]}
          />
        </div>
      </div>

      <div className="card flex-col-gap-md" data-testid="cpu-mode-breakdown">
        <h3 className="section-heading">Mode Breakdown</h3>
        <div className="detail-grid detail-grid-md">
          {Object.entries(cpuDetails?.modeBreakdown ?? {}).map(([mode, value]) => (
            <DetailRow key={mode} label={mode} value={metricLabel(value)} />
          ))}
          {Object.keys(cpuDetails?.modeBreakdown ?? {}).length === 0 && <DetailRow label="Mode accounting" value="not yet sampled" />}
        </div>
      </div>

      <div className="detail-grid detail-grid-lg">
        <div className="card flex-col-gap-sm" data-testid="cpu-stall-panel">
          <h3 className="section-heading">Stall Evidence</h3>
          <DetailRow label="CPU pressure some" value={metricLabel(cpuDetails?.cpuPsiSomeAvg10)} />
          <DetailRow label="CPU pressure full" value={metricLabel(cpuDetails?.cpuPsiFullAvg10)} />
        </div>
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Saturation</h3>
          <DetailRow label="Run queue" value={metricLabel(cpuDetails?.runQueueDepth)} />
          <DetailRow label="Normalized load 1m" value={metricLabel(cpuDetails?.normalizedLoad1)} />
          <DetailRow label="Normalized load 5m" value={metricLabel(cpuDetails?.normalizedLoad5)} />
        </div>
      </div>

      <div className="detail-grid detail-grid-lg">
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Per-Core Distribution</h3>
          {Object.entries(cpuDetails?.perCoreUtilization ?? {}).map(([core, value]) => <DetailRow key={core} label={core} value={metricLabel(value)} />)}
          {Object.keys(cpuDetails?.perCoreUtilization ?? {}).length === 0 && <DetailRow label="Per-core utilization" value="not yet sampled" />}
          <DetailRow label="Imbalance index" value={metricLabel(cpuDetails?.coreImbalanceIndex)} />
        </div>
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Throttling & Derate</h3>
          <DetailRow label="Quota throttling" value={metricLabel(cpuDetails?.quotaThrottling)} />
          <DetailRow label="Frequency derate" value={metricLabel(cpuDetails?.frequencyDerateRatio)} />
          <DetailRow label="Thermal evidence" value={metricLabel(cpuDetails?.thermalThrottleEvidence)} />
          <DetailRow label="Nearest thermal trip" value={metricLabel(cpuDetails?.thermalTripPointCelsius)} />
        </div>
      </div>

      <div className="card flex-col-gap-md" data-testid="cpu-attribution-panel">
        <div>
          <h3 className="section-heading">Top CPU Consumers</h3>
          <div className="card-subtitle">
            Processes ranked by CPU utilization
          </div>
        </div>
        {renderProcessTable(topProcesses, 'CPU %', process => process.cpuPercent)}
        <h3 className="section-heading">Cumulative CPU-seconds</h3>
        <div className="card-subtitle">Process CPU time accumulated during the sampling interval</div>
        {renderProcessTable(topCPUSecondsProcesses, 'CPU seconds', process => process.cpuSeconds)}
        <div className="card-subtitle">Historical owner attribution is available through <code>metrics process-timeline --rank cpu_seconds</code>.</div>
      </div>

      <div className="card">
        <ProcessMonitor data={processMonitorData} collapsible={false} isExpanded={true} />
      </div>
    </MetricDetailLayout>
  );
};

function mergeSeries(first?: Array<{ timestamp: string; value: number }>, second?: Array<{ timestamp: string; value: number }>): Array<{ timestamp: string; iowait?: number; steal?: number; some?: number; full?: number }> {
  const merged = new Map<string, { timestamp: string; iowait?: number; steal?: number; some?: number; full?: number }>();
  for (const point of first ?? []) merged.set(point.timestamp, { timestamp: point.timestamp, iowait: point.value, some: point.value });
  for (const point of second ?? []) {
    const row = merged.get(point.timestamp) ?? { timestamp: point.timestamp };
    row.steal = point.value;
    row.full = point.value;
    merged.set(point.timestamp, row);
  }
  return [...merged.values()].sort((a, b) => a.timestamp.localeCompare(b.timestamp));
}

const metricLabel = (metric?: MetricValue, suffix = ''): string => {
	if (!metric?.state?.case) return 'not yet sampled';
	if (metric.state.case === 'measured') return typeof metric.state.value === 'number' ? `${metric.state.value.toFixed(1)}${suffix}` : 'measured';
	if (metric.state.case === 'notYetSampledReason') return `waiting for next sample: ${metric.state.value}`;
	return `${metric.state.case}: ${String(metric.state.value || 'no value')}${metric.provenance ? ` (${metric.provenance})` : ''}`;
};

const cpuVerdict = (details: DetailedMetrics['cpuDetails'] | undefined, usage: number | undefined): string => {
  const value = details?.usageState?.state?.case === 'measured' ? details.usageState.state.value : usage;
  if (value === undefined) return 'Utilization not measured';
  const steal = details?.modeBreakdown?.steal;
  if (steal?.state?.case === 'measured' && (steal.state.value ?? 0) > 5) return `${value.toFixed(1)}% utilization — hypervisor steal detected`;
  if (details?.cpuPsiSomeAvg10?.state?.case === 'measured' && (details.cpuPsiSomeAvg10.state.value ?? 0) > 10) return `${value.toFixed(1)}% utilization — CPU work is stalled`;
  if (details?.quotaThrottling?.state?.case === 'measured') return `${value.toFixed(1)}% utilization — quota throttling observed`;
  if (details?.coreImbalanceIndex?.state?.case === 'measured' && (details.coreImbalanceIndex.state.value ?? 0) > 25) return `${value.toFixed(1)}% utilization — workload is imbalanced`;
  return `${value.toFixed(1)}% utilization`;
};
