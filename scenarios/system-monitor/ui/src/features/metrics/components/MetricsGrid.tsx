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

interface MetricsGridProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  expandedCards: Set<CardType>;
  onToggleCard: (cardType: CardType) => void;
  metricHistory: MetricHistory | null;
  storageIO?: StorageIOInfo | null;
  onOpenDetail: (cardType: CardType) => void;
  diskLastUpdated?: string;
}

export const MetricsGrid = ({
  metrics,
  detailedMetrics,
  expandedCards,
  onToggleCard,
  metricHistory,
  storageIO,
  onOpenDetail,
  diskLastUpdated
}: MetricsGridProps) => {
  const diskIOHistory = useMemo(() => {
    const readSeries = metricHistory?.diskRead ?? [];
    const writeSeries = metricHistory?.diskWrite ?? [];
    if (readSeries.length === 0 && writeSeries.length === 0) {
      return undefined;
    }

    const length = Math.max(readSeries.length, writeSeries.length);
    const combined: ChartDataPoint[] = [];
    for (let index = 0; index < length; index++) {
      const readPoint = readSeries[index];
      const writePoint = writeSeries[index];
      const timestamp = readPoint?.timestamp ?? writePoint?.timestamp;
      if (!timestamp) {
        continue;
      }
      const value = (readPoint?.value ?? 0) + (writePoint?.value ?? 0);
      combined.push({ timestamp, value });
    }
    return combined;
  }, [metricHistory?.diskRead, metricHistory?.diskWrite]);

  const diskDetails = useMemo<DiskCardDetails | undefined>(() => {
    if (!detailedMetrics?.memoryDetails?.diskUsage) {
      return undefined;
    }
    return {
      diskUsage: detailedMetrics.memoryDetails.diskUsage,
      storageIO: storageIO ?? undefined,
      lastUpdated: diskLastUpdated ?? (detailedMetrics.timestamp ? timestampDate(detailedMetrics.timestamp).toISOString() : undefined)
    };
  }, [detailedMetrics, storageIO, diskLastUpdated]);

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
        label="CPU USAGE"
        unit="%"
        metric={metrics?.cpu}
        isExpanded={expandedCards.has('cpu')}
        onToggle={() => { onToggleCard('cpu'); }}
        details={detailedMetrics?.cpuDetails}
        alertCount={0} // TODO: Calculate based on thresholds
        history={metricHistory?.cpu}
        historyWindowSeconds={metricHistory?.windowSeconds}
        valueDomain={[0, 100]}
        onOpenDetails={() => { onOpenDetail('cpu'); }}
        detailButtonLabel="Open CPU detail"
      />

      {/* Memory Card */}
      <MetricCard
        type="memory"
        label="MEMORY"
        unit="%"
        metric={metrics?.memory}
        isExpanded={expandedCards.has('memory')}
        onToggle={() => { onToggleCard('memory'); }}
        details={detailedMetrics?.memoryDetails}
        alertCount={0} // TODO: Calculate based on thresholds
        history={metricHistory?.memory}
        historyWindowSeconds={metricHistory?.windowSeconds}
        valueDomain={[0, 100]}
        onOpenDetails={() => { onOpenDetail('memory'); }}
        detailButtonLabel="Open memory detail"
      />

      {/* GPU Card */}
      <MetricCard
        type="gpu"
        label="GPU"
        unit="%"
        metric={metrics?.gpu}
        isExpanded={expandedCards.has('gpu')}
        onToggle={() => { onToggleCard('gpu'); }}
        details={gpuDetails}
        alertCount={0}
        history={metricHistory?.gpu}
        historyWindowSeconds={metricHistory?.windowSeconds}
        valueDomain={[0, 100]}
        historyUnit=" %"
        onOpenDetails={() => { onOpenDetail('gpu'); }}
        detailButtonLabel="Open GPU detail"
      />

      {/* Disk Card */}
      <MetricCard
        type="disk"
        label="DISK"
        unit="%"
        metric={metrics?.disk}
        isExpanded={expandedCards.has('disk')}
        onToggle={() => { onToggleCard('disk'); }}
        details={diskDetails}
        alertCount={0}
        history={diskIOHistory}
        historyWindowSeconds={metricHistory?.windowSeconds}
        historyUnit=" MB/s"
        onOpenDetails={() => { onOpenDetail('disk'); }}
        detailButtonLabel="Open disk detail"
      />

      {/* Network & Connections Card */}
      <MetricCard
        type="network"
        label="NETWORK & CONNECTIONS"
        unit="#"
        metric={metrics?.connections}
        isExpanded={expandedCards.has('network')}
        onToggle={() => { onToggleCard('network'); }}
        details={detailedMetrics?.networkDetails}
        alertCount={0} // TODO: Calculate based on thresholds
        history={metricHistory?.network}
        historyWindowSeconds={metricHistory?.windowSeconds}
        onOpenDetails={() => { onOpenDetail('network'); }}
        detailButtonLabel="Open network detail"
      />

    </div>
  );
};
