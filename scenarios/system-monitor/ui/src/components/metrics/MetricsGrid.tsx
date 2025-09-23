import { useState, useEffect, useMemo } from 'react';
import type {
  MetricsResponse,
  DetailedMetrics,
  CardType,
  MetricHistory,
  StorageIOInfo,
  DiskCardDetails,
  ChartDataPoint
} from '../../types';
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
  const [isDesktop, setIsDesktop] = useState(window.innerWidth >= 1024);
  
  useEffect(() => {
    const handleResize = () => {
      setIsDesktop(window.innerWidth >= 1024);
    };
    
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

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
    if (!detailedMetrics?.memory_details?.disk_usage) {
      return undefined;
    }
    return {
      diskUsage: detailedMetrics.memory_details.disk_usage,
      storageIO: storageIO ?? undefined,
      lastUpdated: diskLastUpdated ?? detailedMetrics.timestamp
    };
  }, [detailedMetrics, storageIO, diskLastUpdated]);
  
  const handleCardToggle = (cardType: CardType) => {
    if (isDesktop) {
      // On desktop, toggle all metric cards together
      const allMetricCards: CardType[] = ['cpu', 'memory', 'disk', 'network'];
      const isAnyExpanded = allMetricCards.some(card => expandedCards.has(card));
      
      // If any card is expanded, collapse all. If none are expanded, expand all.
      if (isAnyExpanded) {
        // Collapse all metric cards
        allMetricCards.forEach(card => {
          if (expandedCards.has(card)) {
            onToggleCard(card);
          }
        });
      } else {
        // Expand all metric cards
        allMetricCards.forEach(card => {
          if (!expandedCards.has(card)) {
            onToggleCard(card);
          }
        });
      }
    } else {
      // On mobile/tablet, toggle individual cards
      onToggleCard(cardType);
    }
  };
  return (
    <div className="grid grid-4" style={{ gap: 'var(--spacing-lg)' }}>
      
      {/* CPU Usage Card */}
      <MetricCard
        type="cpu"
        label="CPU USAGE"
        unit="%"
        value={metrics?.cpu_usage ?? 0}
        isExpanded={expandedCards.has('cpu')}
        onToggle={() => handleCardToggle('cpu')}
        details={detailedMetrics?.cpu_details}
        alertCount={0} // TODO: Calculate based on thresholds
        history={metricHistory?.cpu}
        historyWindowSeconds={metricHistory?.windowSeconds}
        valueDomain={[0, 100]}
        onOpenDetails={() => onOpenDetail('cpu')}
        detailButtonLabel="OPEN DETAIL"
      />

      {/* Memory Card */}
      <MetricCard
        type="memory"
        label="MEMORY"
        unit="%"
        value={metrics?.memory_usage ?? 0}
        isExpanded={expandedCards.has('memory')}
        onToggle={() => handleCardToggle('memory')}
        details={detailedMetrics?.memory_details}
        alertCount={0} // TODO: Calculate based on thresholds
        history={metricHistory?.memory}
        historyWindowSeconds={metricHistory?.windowSeconds}
        valueDomain={[0, 100]}
        onOpenDetails={() => onOpenDetail('memory')}
        detailButtonLabel="OPEN DETAIL"
      />

      {/* Disk Card */}
      <MetricCard
        type="disk"
        label="DISK"
        unit="%"
        value={diskDetails?.diskUsage.percent ?? 0}
        isExpanded={expandedCards.has('disk')}
        onToggle={() => handleCardToggle('disk')}
        details={diskDetails}
        alertCount={0}
        history={diskIOHistory}
        historyWindowSeconds={metricHistory?.windowSeconds}
        historyUnit=" MB/s"
        onOpenDetails={() => onOpenDetail('disk')}
        detailButtonLabel="OPEN DETAIL"
      />

      {/* Network & Connections Card */}
      <MetricCard
        type="network"
        label="NETWORK & CONNECTIONS"
        unit="#"
        value={metrics?.tcp_connections ?? 0}
        isExpanded={expandedCards.has('network')}
        onToggle={() => handleCardToggle('network')}
        details={detailedMetrics?.network_details}
        alertCount={0} // TODO: Calculate based on thresholds
        history={metricHistory?.network}
        historyWindowSeconds={metricHistory?.windowSeconds}
        onOpenDetails={() => onOpenDetail('network')}
        detailButtonLabel="OPEN DETAIL"
      />

    </div>
  );
};
