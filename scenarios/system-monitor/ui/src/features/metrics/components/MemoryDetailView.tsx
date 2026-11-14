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

  const kernel = memoryDetails?.kernelMemory;
  const kernelData = useMemo(() => combineFlowSeries([
    { key: 'slab', points: metricHistory?.kernelSlab },
    { key: 'reclaimable', points: metricHistory?.kernelReclaimable },
    { key: 'unreclaimable', points: metricHistory?.kernelUnreclaimable }
  ]), [metricHistory?.kernelSlab, metricHistory?.kernelReclaimable, metricHistory?.kernelUnreclaimable]);
  const hostTotal = kernel?.hostTotalBytes?.state;
  const unreclaimable = kernel?.unreclaimableBytes?.state;
  const unreclaimableShare = hostTotal?.case === 'measured' && hostTotal.value > 0 && unreclaimable?.case === 'measured'
    ? `${(Number(unreclaimable.value) / Number(hostTotal.value) * 100).toFixed(1)}% of physical RAM` : undefined;

  const swapUsage = memoryDetails?.swapUsage;
  const topProcesses = memoryDetails?.topProcesses;
  const topPagingProcesses = memoryDetails?.topPagingProcesses;
	const paging = memoryDetails?.paging;
	const fragmentation = memoryDetails?.fragmentation;
  const swapTrafficState = paging?.swapTrafficPagesPerSecond?.state;
  const swapTraffic = swapTrafficState?.case === 'measured' ? swapTrafficState.value : undefined;
  const pressure = memoryDetails?.memoryPressure;
  const pressureColor = pressure?.level === 'critical' ? 'var(--color-error)'
    : pressure?.level === 'elevated' ? 'var(--color-warning)' : undefined;

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
            Swap traffic: {swapTraffic === undefined ? (metricStateReason(swapTrafficState) ?? 'not yet sampled') : `${Number(swapTraffic).toFixed(1)} pages/sec`}
          </div>
          <div className="text-muted">
            Swap occupancy records pages held in swap. Traffic alone does not establish memory pressure or thrashing.
          </div>
        </div>

        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Memory Pressure</h3>
          <DetailRow label="Pressure level" value={pressure?.level || 'not measured'} valueColor={pressureColor} />
          <div className="text-muted" role="status">{pressure?.levelReason || 'Memory pressure has not been measured.'}</div>
          {pressure?.levelSource && <DetailRow label="Pressure source" value={pressure.levelSource} />}
          {pressure?.observedAt && <DetailRow label="Pressure observed" value={formatProtoTimestamp(pressure.observedAt)} />}
          <DetailRow label="Swap in" value={pressureSignalLabel(pressure?.signals?.swap_in_bytes_per_second, 'MiB/s')} />
          <DetailRow label="Swap out" value={pressureSignalLabel(pressure?.signals?.swap_out_bytes_per_second, 'MiB/s')} />
          <DetailRow label="Memory stalls, some (60s)" value={pressureSignalLabel(pressure?.signals?.psi_some_avg60, '%')} />
          <DetailRow label="Memory stalls, full (60s)" value={pressureSignalLabel(pressure?.signals?.psi_full_avg60, '%')} />
        </div>

      </div></DetailSection>

      <DetailSection id="kernel-memory" title="Kernel memory"><div className="card flex-col-gap-sm">
        <h3 className="section-heading">Kernel Slab Accounting</h3>
        <DetailRow label="Slab total" value={kernelBytesLabel(kernel?.slabBytes)} />
        <DetailRow label="Reclaimable slab" value={kernelBytesLabel(kernel?.reclaimableBytes)} />
        <DetailRow label="Unreclaimable slab" value={kernelBytesLabel(kernel?.unreclaimableBytes)} />
        {unreclaimableShare && <div className="card-subtitle">{unreclaimableShare}</div>}
        {kernel?.slabBytes?.observedAt && <DetailRow label="Kernel accounting observed" value={formatProtoTimestamp(kernel.slabBytes.observedAt)} />}
        <div className="text-muted">Kernel allocation is not a leak diagnosis. Compare growth over time with workload and memory pressure; unreclaimable slab cannot be freed by dropping file caches.</div>
        <MetricLineChart status={metricHistory === null ? 'loading' : 'ready'}
          seriesLabel="kernel slab bytes" data={kernelData}
          lines={[
            { dataKey: 'slab', name: 'Slab total', color: 'var(--color-info)' },
            { dataKey: 'reclaimable', name: 'Reclaimable', color: 'var(--color-success)' },
            { dataKey: 'unreclaimable', name: 'Unreclaimable', color: 'var(--color-warning)' }
          ]}
          unit="bytes" valueFormatter={value => formatBytes(value)}
        />
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

function kernelBytesLabel(signal: MetricValue | undefined): string {
  return signal?.state?.case === 'measured' ? formatBytes(Number(signal.state.value))
    : metricStateReason(signal?.state) ?? 'not yet measured';
}

function pressureSignalLabel(signal: MetricValue | undefined, unit: 'MiB/s' | '%'): string {
  if (signal?.state?.case !== 'measured') {
    return metricStateReason(signal?.state) ?? 'not yet measured';
  }
  const value = Number(signal.state.value);
  return unit === 'MiB/s' ? `${(value / (1024 * 1024)).toFixed(1)} MiB/s` : `${value.toFixed(2)}%`;
}

function metricStateReason(state: MetricValue['state'] | undefined): string | undefined {
	if (!state) return undefined;
	if (state.case === 'unsupportedReason' || state.case === 'failedError' || state.case === 'staleReason' || state.case === 'notYetSampledReason') {
		return String(state.value ?? 'Metric unavailable.');
  }
  return undefined;
}
