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
import type { MetricValue } from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { MetricSparkline } from './MetricSparkline';
import { expansionMap } from './expansions';
import { formatWindowLabel } from '../../../shared/utils/formatters';

interface MetricCardProps {
  type: CardType;
  label: string;
  unit: string;
  value?: number | null;
  metric?: MetricValue;
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
  metric,
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
  const state = metric?.state;
  const metricMeasured = state?.case === 'measured' ? state.value : undefined;
  const metricReason = state?.case === 'unsupportedReason' || state?.case === 'failedError' ? state.value : undefined;
  const hasNumericValue = typeof metricMeasured === 'number'
    ? Number.isFinite(metricMeasured)
    : typeof value === 'number' && Number.isFinite(value);
  const resolvedValue = hasNumericValue ? (typeof metricMeasured === 'number' ? metricMeasured : value ?? null) : null;
  const isUnavailable = Boolean(metric && state?.case !== 'measured');
  const observedAt = metric?.observedAt ? timestampDate(metric.observedAt).toLocaleTimeString() : undefined;

  const getProgressValue = (): number => {
    if (!hasNumericValue || resolvedValue === null) {
      return 0;
    }
    return Math.min(resolvedValue, 100);
  };

  const getValueColor = (): string => {
    if (!hasNumericValue || resolvedValue === null) {
      return 'var(--color-text-secondary)';
    }
    if (resolvedValue >= 90) return 'var(--color-error)';
    if (resolvedValue >= 70) return 'var(--color-warning)';
    return 'var(--color-text-heading)';
  };

  const sparklineColor = (() => {
    switch (type) {
      case 'cpu':
        return 'var(--color-primary)';
      case 'memory':
        return 'var(--color-warning)';
      case 'network':
        return 'var(--color-primary)';
      case 'disk':
        return 'var(--color-info)';
      case 'gpu':
        return 'var(--color-info)';
      default:
        return 'var(--color-primary)';
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

      <div className={`metric-value ${getValueColor() === 'var(--color-error)' ? 'text-error' : getValueColor() === 'var(--color-warning)' ? 'text-warning' : getValueColor() === 'var(--color-text-secondary)' ? 'text-muted' : 'text-heading'}`}>
        {isUnavailable && state?.case === 'failedError' ? '⚠' : resolvedValue !== null ? resolvedValue.toFixed(1) : '—'}
      </div>

      <div
        className="sr-only"
        role={hasNumericValue && !isUnavailable ? 'meter' : 'status'}
        aria-label={isUnavailable || !hasNumericValue ? `${label}: ${metricReason ?? 'not measured'}` : `${label}: ${resolvedValue?.toFixed(1)}${unit}`}
        aria-valuemin={hasNumericValue && !isUnavailable ? 0 : undefined}
        aria-valuemax={hasNumericValue && !isUnavailable && type !== 'network' ? 100 : undefined}
        aria-valuenow={hasNumericValue && !isUnavailable && resolvedValue !== null ? resolvedValue : undefined}
      />
      {isUnavailable && <span className="metric-state-reason" title={metricReason}>{metricReason ?? 'Not measured'}</span>}
      {observedAt && <div className="metric-observed-at">Measured {observedAt}</div>}

      {history && history.length > 0 ? (
        <div data-sm-style="sm-style-254feda622">
          <MetricSparkline
            data={history}
            color={sparklineColor}
            valueDomain={valueDomain}
            threshold={sparklineThreshold}
            unit={sparklineUnit}
            windowLabel={formatWindowLabel(historyWindowSeconds)}
            ariaLabel={`${label} history${resolvedValue !== null ? `, latest ${resolvedValue.toFixed(1)}${unit}` : ', not measured'}`}
          />
        </div>
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
        <div data-sm-style="sm-style-ffa5044e12">
          <button
            type="button"
            className="btn btn-action text-xs"
            data-sm-style="sm-style-84a0f8980c"
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
