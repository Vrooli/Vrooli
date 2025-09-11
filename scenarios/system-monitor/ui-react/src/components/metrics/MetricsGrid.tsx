import { useState, useEffect } from 'react';
import type { MetricsResponse, DetailedMetrics, CardType } from '../../types';
import { MetricCard } from './MetricCard';

interface MetricsGridProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  expandedCards: Set<CardType>;
  onToggleCard: (cardType: string) => void;
}

export const MetricsGrid = ({ metrics, detailedMetrics, expandedCards, onToggleCard }: MetricsGridProps) => {
  const [isDesktop, setIsDesktop] = useState(window.innerWidth >= 1024);
  
  useEffect(() => {
    const handleResize = () => {
      setIsDesktop(window.innerWidth >= 1024);
    };
    
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);
  
  const handleCardToggle = (cardType: string) => {
    if (isDesktop) {
      // On desktop, toggle all metric cards together
      const allMetricCards = ['cpu', 'memory', 'network', 'system'];
      const isAnyExpanded = allMetricCards.some(card => expandedCards.has(card as CardType));
      
      // If any card is expanded, collapse all. If none are expanded, expand all.
      if (isAnyExpanded) {
        // Collapse all metric cards
        allMetricCards.forEach(card => {
          if (expandedCards.has(card as CardType)) {
            onToggleCard(card);
          }
        });
      } else {
        // Expand all metric cards
        allMetricCards.forEach(card => {
          if (!expandedCards.has(card as CardType)) {
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
      />

      {/* Memory & Storage Card */}
      <MetricCard
        type="memory"
        label="MEMORY & STORAGE"
        unit="%"
        value={metrics?.memory_usage ?? 0}
        isExpanded={expandedCards.has('memory')}
        onToggle={() => handleCardToggle('memory')}
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
        onToggle={() => handleCardToggle('network')}
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
        onToggle={() => handleCardToggle('system')}
        details={detailedMetrics?.system_details}
        alertCount={0} // TODO: Calculate based on thresholds
        isStatusCard={true}
      />

    </div>
  );
};