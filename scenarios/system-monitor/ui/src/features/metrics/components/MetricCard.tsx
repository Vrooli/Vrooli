import { ChevronDown, ChevronUp } from 'lucide-react';
import type {
  CardType,
  CPUMetrics,
  MemoryMetrics,
  NetworkMetrics,
  ChartDataPoint,
  DiskCardDetails,
  GPUCardDetails
} from '../../../types';
import { MetricSparkline } from './MetricSparkline';
import { expansionMap } from './expansions';
import { formatWindowLabel } from '../../../shared/utils/formatters';

interface MetricCardProps {
  type: CardType;
  label: string;
  unit: string;
  value?: number | null;
  isExpanded: boolean;
  onToggle: () => void;
  details?: CPUMetrics | MemoryMetrics | NetworkMetrics | DiskCardDetails | GPUCardDetails;
  alertCount: number;
  history?: ChartDataPoint[];
  historyWindowSeconds?: number;
  valueDomain?: [number, number];
  threshold?: number;
  historyUnit?: string;
  onOpenDetails?: () => void;
  detailButtonLabel?: string;
}

export const MetricCard = ({
  type,
  label,
  unit,
  value,
  isExpanded,
  onToggle,
  details,
  alertCount,
  history,
  historyWindowSeconds,
  valueDomain,
  threshold,
  historyUnit,
  onOpenDetails,
  detailButtonLabel = 'VIEW DETAIL'
}: MetricCardProps) => {
  const hasNumericValue = typeof value === 'number' && Number.isFinite(value);
  const resolvedValue = hasNumericValue ? (value as number) : null;

  const getProgressValue = (): number => {
    if (!hasNumericValue || resolvedValue === null) {
      return 0;
    }
    return Math.min(resolvedValue, 100);
  };

  const getValueColor = (): string => {
    if (!hasNumericValue || resolvedValue === null) {
      return 'var(--color-text-dim)';
    }
    if (resolvedValue >= 90) return 'var(--color-error)';
    if (resolvedValue >= 70) return 'var(--color-warning)';
    return 'var(--color-text-bright)';
  };

  const sparklineColor = (() => {
    switch (type) {
      case 'cpu':
        return 'var(--color-accent)';
      case 'memory':
        return 'var(--color-warning)';
      case 'network':
        return 'var(--color-accent)';
      case 'disk':
        return 'var(--color-info)';
      case 'gpu':
        return 'var(--color-info)';
      default:
        return 'var(--color-accent)';
    }
  })();

  const defaultThresholds: Partial<Record<CardType, number>> = {
    cpu: 75,
    memory: 80,
    disk: 80,
    gpu: 85
  };

  const sparklineThreshold = typeof threshold === 'number' ? threshold : defaultThresholds[type];

  const sparklineUnit = (() => {
    if (historyUnit) {
      return historyUnit;
    }
    if (type === 'network') {
      return ' connections';
    }
    if (type === 'gpu') {
      return '%';
    }
    if (unit === '%') {
      return '%';
    }
    return unit ? ` ${unit}` : undefined;
  })();

  const renderExpandedContent = () => {
    if (!isExpanded || !details) return null;
    const Expansion = expansionMap[type];
    if (!Expansion) return null;
    return <Expansion details={details} />;
  };

  return (
    <div
      className={`metric-card expandable ${isExpanded ? 'expanded' : ''}`}
      onClick={onToggle}
    >
      <div className="metric-header">
        <span className="metric-label">
          {label}
        </span>

        <div className="flex-row-center gap-sm">
          {alertCount > 0 && (
            <span className="alert-badge">
              {alertCount}
            </span>
          )}

          <span className="text-sm text-muted">
            {unit}
          </span>

          <span className="text-accent">
            {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
          </span>
        </div>
      </div>

      <div className="metric-value" style={{ color: getValueColor() }}>
        {resolvedValue !== null ? resolvedValue.toFixed(1) : '—'}
      </div>

      {history && history.length > 0 ? (
        <MetricSparkline
          data={history}
          color={sparklineColor}
          valueDomain={valueDomain}
          threshold={sparklineThreshold}
          unit={sparklineUnit}
          windowLabel={formatWindowLabel(historyWindowSeconds)}
        />
      ) : (
        <div className="metric-bar">
          <div
            className="metric-fill"
            style={{ width: `${getProgressValue()}%` }}
          />
        </div>
      )}

      {renderExpandedContent()}

      {onOpenDetails && (
        <div style={{ marginTop: 'var(--spacing-md)', display: 'flex', justifyContent: 'flex-end' }}>
          <button
            type="button"
            className="btn btn-action text-xs"
            style={{ letterSpacing: '0.08em' }}
            onClick={event => {
              event.stopPropagation();
              onOpenDetails?.();
            }}
          >
            {detailButtonLabel}
          </button>
        </div>
      )}
    </div>
  );
};
