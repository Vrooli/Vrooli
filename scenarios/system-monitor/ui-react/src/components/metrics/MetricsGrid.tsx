import type { MetricsResponse, DetailedMetrics, CardType } from '../../types';
import { MetricCard } from './MetricCard';

interface MetricsGridProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  expandedCards: Set<CardType>;
  onToggleCard: (cardType: string) => void;
}

export const MetricsGrid = ({ metrics, detailedMetrics, expandedCards, onToggleCard }: MetricsGridProps) => {
  return (
    <div className="grid grid-4" style={{ gap: 'var(--spacing-lg)' }}>
      
      {/* CPU Usage Card */}
      <MetricCard
        type="cpu"
        label="CPU USAGE"
        unit="%"
        value={metrics?.cpu_usage ?? 0}
        isExpanded={expandedCards.has('cpu')}
        onToggle={() => onToggleCard('cpu')}
        details={detailedMetrics?.cpu_details}
        alertCount={0} // TODO: Calculate based on thresholds
      />

      {/* Memory & Storage Card */}
      <MetricCard
        type="memory"
        label="MEMORY & STORAGE"
        unit="%"
        value={metrics?.memory_usage ?? 0}
        isExpanded={expandedCards.has('memory')}
        onToggle={() => onToggleCard('memory')}
        details={detailedMetrics?.memory_details}
        alertCount={0} // TODO: Calculate based on thresholds
      />

      {/* Network & Connections Card */}
      <MetricCard
        type="network"
        label="NETWORK & CONNECTIONS"
        unit="#"
        value={metrics?.tcp_connections ?? 0}
        isExpanded={expandedCards.has('network')}
        onToggle={() => onToggleCard('network')}
        details={detailedMetrics?.network_details}
        alertCount={0} // TODO: Calculate based on thresholds
      />

      {/* System Health Card */}
      <MetricCard
        type="system"
        label="SYSTEM HEALTH"
        unit="STATE"
        value="ONLINE"
        isExpanded={expandedCards.has('system')}
        onToggle={() => onToggleCard('system')}
        details={detailedMetrics?.system_details}
        alertCount={0} // TODO: Calculate based on thresholds
        isStatusCard={true}
      />

    </div>
  );
};