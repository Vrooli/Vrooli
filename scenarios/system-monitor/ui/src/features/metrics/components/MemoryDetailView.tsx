import { useMemo } from 'react';
import { MemoryStick } from 'lucide-react';

import { DetailRow } from '../../../shared/components/DetailRow';
import { formatBytes, formatProtoTimestamp } from '../../../shared/utils/formatters';
import type {
  MetricsResponse,
  DetailedMetrics,
  MetricHistory,
  MetricValue
} from '../../../types';
import { DetailSection, MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { buildSingleSeriesData, combineFlowSeries, combineMemorySeries } from '../../../shared/utils/chartData';
import { renderProcessTable } from './MetricRenderHelpers';

export interface MemoryDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const MemoryDetailView = ({ metrics, detailedMetrics, metricHistory, onBack }: MemoryDetailViewProps) => {
  const memoryUsage = detailedMetrics?.memoryDetails?.usage ?? (metrics?.memory?.state?.case === 'measured' ? metrics.memory.state.value : undefined);
  const memoryData = useMemo(
    () => combineMemorySeries(metricHistory?.memory, metricHistory?.swap),
    [metricHistory?.memory, metricHistory?.swap]
  );
  const swapFlowData = useMemo(() => combineFlowSeries([
    { key: 'swapLevel', points: metricHistory?.swap },
    { key: 'swapTraffic', points: metricHistory?.swapTraffic }
  ]), [metricHistory?.swap, metricHistory?.swapTraffic]);
  const memoryDetails = detailedMetrics?.memoryDetails;
  const majorFaultData = useMemo(
    () => buildSingleSeriesData(metricHistory?.majorFaults),
    [metricHistory?.majorFaults]
  );
  const fragmentationData = useMemo(() => combineFlowSeries([
    { key: 'fragmentation', points: metricHistory?.fragmentation }
  ]), [metricHistory?.fragmentation]);

  const swapUsage = memoryDetails?.swapUsage;
  const topProcesses = memoryDetails?.topProcesses;
  const topPagingProcesses = memoryDetails?.topPagingProcesses;
	const paging = memoryDetails?.paging;
	const fragmentation = memoryDetails?.fragmentation;
  const swapTrafficState = paging?.swapTrafficPagesPerSecond?.state;
  const swapTraffic = swapTrafficState?.case === 'measured' ? swapTrafficState.value : undefined;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatProtoTimestamp(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      layoutId="memory"
      title="MEMORY UTILIZATION"
      icon={<MemoryStick size={22} />}
      headline={memoryUsage === undefined ? 'Utilization not measured' : `${memoryUsage.toFixed(1)}% used`}
      subhead={subhead}
      onBack={onBack}
    >
      <DetailSection id="memory-history" title="Memory and swap history"><MetricLineChart
        status={metricHistory === null ? 'loading' : 'ready'}
        seriesLabel="memory"
        className="card"
        data={memoryData}
        lines={[
          { dataKey: 'memory', name: 'Memory Usage', color: 'var(--color-warning)' },
          { dataKey: 'swap', name: 'Swap Usage', color: 'var(--color-info)' }
        ]}
        unit="%"
        yDomain={[0, 100]}
        valueFormatter={value => `${value.toFixed(1)}%`}
      /></DetailSection>

      <DetailSection id="swap-flow" title="Swap flow"><MetricLineChart
        status={metricHistory === null ? 'loading' : 'ready'}
        seriesLabel="swap level and traffic"
        className="card"
        data={swapFlowData}
        lines={[
          { dataKey: 'swapLevel', name: 'Swap level', color: 'var(--color-info)' },
          { dataKey: 'swapTraffic', name: 'Swap traffic', color: 'var(--color-warning)', yAxisId: 'right' }
        ]}
        unit="%"
        yDomain={[0, 100]}
        rightYAxisUnit="/sec"
        valueFormatter={value => `${value.toFixed(1)}%`}
      /></DetailSection>

      <DetailSection id="major-faults" title="Major faults"><MetricLineChart
        status={metricHistory === null ? 'loading' : 'ready'}
        seriesLabel="major faults"
        className="card"
        data={majorFaultData}
        lines={[
          { dataKey: 'value', name: 'Major faults', color: 'var(--color-error)' }
        ]}
        unit="/sec"
        yDomain={[1, 'auto']}
        yAxisScale="log"
        valueFormatter={value => `${value.toFixed(1)}/sec`}
      /></DetailSection>

      <DetailSection id="swap-activity" title="Swap activity"><div className="detail-grid detail-grid-lg" data-sm-style="sm-style-f383142193">
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
          <div className="card-subtitle">
            Swap traffic: {swapTraffic === undefined ? 'not yet sampled' : `${Number(swapTraffic).toFixed(1)} pages/sec`}
          </div>
          <div className="text-muted">
            {swapTraffic === undefined ? 'Paging flow is unavailable.' : swapTraffic > 128 ? 'Active paging indicates thrashing.' : 'Swap residency is quiet.'}
          </div>
        </div>

      </div></DetailSection>

      {fragmentation?.maxFreeOrder?.state?.case === 'measured' ? (
      <DetailSection id="fragmentation" title="Memory fragmentation"><div className="card flex-col-gap-sm">
        <h3 className="section-heading">Memory Fragmentation</h3>
        {
          <>
            <MetricLineChart
              status={metricHistory === null ? 'loading' : 'ready'}
              seriesLabel="fragmentation"
              data={fragmentationData}
              lines={[{ dataKey: 'fragmentation', name: 'Max free order', color: 'var(--color-warning)' }]}
              unit="order"
              valueFormatter={value => value.toFixed(0)}
            />
            <DetailRow label="Highest free order" value={String(fragmentation.maxFreeOrder.state.value)} />
            {fragmentation.compactionFailureRatio?.state?.case === 'measured' && (
              <DetailRow label="Compaction failure ratio" value={`${(Number(fragmentation.compactionFailureRatio.state.value) * 100).toFixed(1)}%`} />
            )}
            {fragmentation.buddyinfo && <pre className="text-muted">{JSON.stringify(fragmentation.buddyinfo, null, 2)}</pre>}
          </>
        }
      </div></DetailSection>
      ) : (
        <DetailSection id="fragmentation" title="Memory fragmentation"><div className="text-muted" role="status">{metricStateReason(fragmentation?.maxFreeOrder?.state) ?? 'Fragmentation is not yet measured.'}</div></DetailSection>
      )}

      <DetailSection id="memory-consumers" title="Top memory consumers"><div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Top Memory Consumers</h3>
          <div className="card-subtitle">
            Processes ranked by resident set size
          </div>
        </div>
        {renderProcessTable(topProcesses, 'Memory (MB)', process => process.memoryMb)}
      </div></DetailSection>

      <DetailSection id="paging-consumers" title="Top paging consumers"><div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Top Paging Consumers</h3>
          <div className="card-subtitle">Processes ranked by major faults per second</div>
        </div>
				{renderProcessTable(topPagingProcesses, 'Major faults/sec', process => process.majorFaultsPerSecond)}
      </div></DetailSection>
    </MetricDetailLayout>
  );
};

function metricStateReason(state: MetricValue['state'] | undefined): string | undefined {
	if (!state) return undefined;
	if (state.case === 'unsupportedReason' || state.case === 'failedError' || state.case === 'staleReason' || state.case === 'notYetSampledReason') {
		return String(state.value ?? 'Metric unavailable.');
  }
  return undefined;
}
