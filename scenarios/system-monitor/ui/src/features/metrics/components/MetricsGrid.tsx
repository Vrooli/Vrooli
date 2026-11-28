import { useMemo } from 'react';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type {
  MetricsResponse,
  DetailedMetrics,
  CardType,
  MetricHistory,
  StorageIOInfo,
  DiskCardDetails,
  GPUCardDetails,
  ChartDataPoint
} from '../../../types';
import { MetricCard } from './MetricCard';
import type { ChartInspection } from './MetricSparkline';
import type { Alert } from '../../../types';

/**
 * Which server alert rules belong to which card. Card badges count active
 * server alerts; the dashboard never derives an alert from a metric value.
 */
export const CARD_ALERT_RULES: Record<CardType, readonly string[]> = {
  cpu: ['cpu_usage'],
  memory: ['memory_usage', 'memory_pressure'],
  gpu: ['gpu_usage'],
  disk: [],
  network: ['tcp_connections'],
};

export const cardAlertCount = (card: CardType, alerts: readonly Alert[]): number =>
  alerts.filter(alert => !alert.resolved && CARD_ALERT_RULES[card].includes(alert.category)).length;

/** A total is meaningful only when both rates belong to the same observation. */
export const combineDiskRates = (read: ChartDataPoint[], write: ChartDataPoint[]): ChartDataPoint[] => {
  const writes = new Map(write.filter(point => point.timestamp && Number.isFinite(point.value)).map(point => [point.timestamp, point]));
  return read.flatMap(point => {
    const writePoint = writes.get(point.timestamp);
    return point.timestamp && Number.isFinite(point.value) && writePoint !== undefined
      ? [{ timestamp: point.timestamp, value: point.value + writePoint.value, ...(point.gapBefore || writePoint.gapBefore ? { gapBefore: true } : {}) }]
      : [];
  });
};

interface MetricsGridProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  expandedCards: Set<CardType>;
  onToggleCard: (cardType: CardType) => void;
  metricHistory: MetricHistory | null;
  storageIO?: StorageIOInfo | null;
  onOpenDetail: (cardType: CardType) => void;
  diskLastUpdated?: string;
  /** Active server alerts for the local host (empty for a remote subject). */
  serverAlerts?: readonly Alert[];
  remoteSubject?: boolean;
  inspectionStore?: Map<string, ChartInspection>;
  infrastructureUnavailable?: boolean;
}

export const MetricsGrid = ({
  metrics,
  detailedMetrics,
  expandedCards,
  onToggleCard,
  metricHistory,
  storageIO,
  onOpenDetail,
  diskLastUpdated,
  serverAlerts = [],
  remoteSubject = false,
  inspectionStore,
  infrastructureUnavailable = false
}: MetricsGridProps) => {
  const diskIOHistory = useMemo(() => {
    const readSeries = metricHistory?.diskRead ?? [];
    const writeSeries = metricHistory?.diskWrite ?? [];
    if (readSeries.length === 0 && writeSeries.length === 0) {
      return undefined;
    }

    return combineDiskRates(readSeries, writeSeries);
  }, [metricHistory?.diskRead, metricHistory?.diskWrite]);

  const readRate = storageIO?.readRate?.state;
  const writeRate = storageIO?.writeRate?.state;
  const readAt = storageIO?.readRate?.observedAt;
  const writeAt = storageIO?.writeRate?.observedAt;
  const sameObservation = readAt && writeAt && readAt.seconds === writeAt.seconds && readAt.nanos === writeAt.nanos;
  const diskRate = !infrastructureUnavailable && sameObservation && readRate?.case === 'measured' && writeRate?.case === 'measured' && Number.isFinite(readRate.value) && Number.isFinite(writeRate.value) ? readRate.value + writeRate.value : undefined;
  const rateReason = [readRate, writeRate].find(state => state && state.case !== 'measured' && typeof state.value === 'string');
  const diskUnavailable = infrastructureUnavailable ? 'Disk throughput refresh failed' : diskRate === undefined ? (rateReason && typeof rateReason.value === 'string' ? rateReason.value : 'Disk throughput is not currently measured') : undefined;

  const diskDetails = useMemo<DiskCardDetails | undefined>(() => {
    if (!detailedMetrics?.memoryDetails?.diskUsage) {
      return undefined;
    }
    return {
      diskUsage: detailedMetrics.memoryDetails.diskUsage,
      // The expanded occupancy detail does not consume the throughput series.
      storageIO: undefined,
      lastUpdated: diskLastUpdated ?? (detailedMetrics.timestamp ? timestampDate(detailedMetrics.timestamp).toISOString() : undefined)
    };
  }, [detailedMetrics, diskLastUpdated]);

  const gpuDetails = useMemo<GPUCardDetails | undefined>(() => {
    if (!detailedMetrics?.gpuDetails) {
      return undefined;
    }
    return {
      metrics: detailedMetrics.gpuDetails,
      lastUpdated: detailedMetrics.timestamp ? timestampDate(detailedMetrics.timestamp).toISOString() : undefined
    };
  }, [detailedMetrics]);
  
  return (
    <div className="metrics-grid">
      
      {/* CPU Usage Card */}
      <MetricCard
        type="cpu"
        inspectionStore={inspectionStore}
        label="CPU USAGE"
        unit="%"
        metric={metrics?.cpu}
        isExpanded={expandedCards.has('cpu')}
        onToggle={() => { onToggleCard('cpu'); }}
        details={detailedMetrics?.cpuDetails}
        alertCount={cardAlertCount('cpu', serverAlerts)}
        history={metricHistory?.cpu}
        emptyHistoryMessage={remoteSubject ? 'History unavailable for this remote machine' : undefined}
        historyWindowSeconds={metricHistory?.windowSeconds}
        windowStart={metricHistory?.windowStart}
        windowEnd={metricHistory?.windowEnd}
        sampleIntervalSeconds={metricHistory?.sampleIntervalSeconds}
        valueDomain={[0, 100]}
        onOpenDetails={() => { onOpenDetail('cpu'); }}
        detailButtonLabel="Open CPU detail"
      />

      {/* Memory Card */}
      <MetricCard
        type="memory"
        inspectionStore={inspectionStore}
        label="MEMORY"
        unit="%"
        metric={metrics?.memory}
        isExpanded={expandedCards.has('memory')}
        onToggle={() => { onToggleCard('memory'); }}
        details={detailedMetrics?.memoryDetails}
        alertCount={cardAlertCount('memory', serverAlerts)}
        history={metricHistory?.memory}
        emptyHistoryMessage={remoteSubject ? 'History unavailable for this remote machine' : undefined}
        historyWindowSeconds={metricHistory?.windowSeconds}
        windowStart={metricHistory?.windowStart}
        windowEnd={metricHistory?.windowEnd}
        sampleIntervalSeconds={metricHistory?.sampleIntervalSeconds}
        valueDomain={[0, 100]}
        onOpenDetails={() => { onOpenDetail('memory'); }}
        detailButtonLabel="Open memory detail"
      />

      {/* GPU Card */}
      <MetricCard
        type="gpu"
        inspectionStore={inspectionStore}
        label="GPU"
        unit="%"
        metric={metrics?.gpu}
        isExpanded={expandedCards.has('gpu')}
        onToggle={() => { onToggleCard('gpu'); }}
        details={gpuDetails}
        alertCount={cardAlertCount('gpu', serverAlerts)}
        history={metricHistory?.gpu}
        emptyHistoryMessage={remoteSubject ? 'History unavailable for this remote machine' : undefined}
        historyWindowSeconds={metricHistory?.windowSeconds}
        windowStart={metricHistory?.windowStart}
        windowEnd={metricHistory?.windowEnd}
        valueDomain={[0, 100]}
        historyUnit=" %"
        onOpenDetails={() => { onOpenDetail('gpu'); }}
        detailButtonLabel="Open GPU detail"
      />

      {/* Disk Card */}
      <MetricCard
        type="disk"
        inspectionStore={inspectionStore}
        label="DISK I/O"
        unit="MB/s"
        value={diskRate}
        currentUnavailableReason={diskUnavailable}
        valueObservedAt={diskRate === undefined ? undefined : diskLastUpdated}
        secondaryText={`${metrics?.disk?.state.case === 'measured' ? `${metrics.disk.state.value.toFixed(1)}% space used · ` : ''}I/O history begins in this session`}
        isExpanded={expandedCards.has('disk')}
        onToggle={() => { onToggleCard('disk'); }}
        details={diskDetails}
        alertCount={cardAlertCount('disk', serverAlerts)}
        history={diskIOHistory}
        sampleIntervalSeconds={5}
        emptyHistoryMessage={remoteSubject ? 'History unavailable for this remote machine' : 'Disk throughput is unavailable from the current source'}
        historyWindowSeconds={metricHistory?.windowSeconds}
        windowStart={metricHistory?.windowStart}
        windowEnd={metricHistory?.windowEnd}
        historyUnit=" MB/s"
        onOpenDetails={() => { onOpenDetail('disk'); }}
        detailButtonLabel="Open disk detail"
      />

      {/* Network & Connections Card */}
      <MetricCard
        type="network"
        inspectionStore={inspectionStore}
        label="NETWORK & CONNECTIONS"
        unit="#"
        metric={metrics?.connections}
        isExpanded={expandedCards.has('network')}
        onToggle={() => { onToggleCard('network'); }}
        details={detailedMetrics?.networkDetails}
        alertCount={cardAlertCount('network', serverAlerts)}
        history={metricHistory?.network}
        emptyHistoryMessage={remoteSubject ? 'History unavailable for this remote machine' : undefined}
        historyWindowSeconds={metricHistory?.windowSeconds}
        windowStart={metricHistory?.windowStart}
        windowEnd={metricHistory?.windowEnd}
        sampleIntervalSeconds={metricHistory?.sampleIntervalSeconds}
        onOpenDetails={() => { onOpenDetail('network'); }}
        detailButtonLabel="Open network detail"
      />

    </div>
  );
};
