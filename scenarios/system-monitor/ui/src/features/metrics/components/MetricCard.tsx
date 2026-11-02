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
import type { ChartInspection } from './MetricSparkline';
import { expansionMap } from './expansions';
import { formatWindowLabel } from '../../../shared/utils/formatters';

interface MetricCardProps {
  type: CardType;
  label: string;
  unit: string;
  value?: number | null;
  metric?: MetricValue;
  valueObservedAt?: string;
  secondaryText?: string;
  emptyHistoryMessage?: string;
  currentUnavailableReason?: string;
  isExpanded: boolean;
  onToggle: () => void;
  details?: CPUMetrics | MemoryMetrics | NetworkMetrics | DiskCardDetails | GPUCardDetails;
  alertCount: number;
  history?: ChartDataPoint[];
  historyWindowSeconds?: number;
  windowStart?: string;
  windowEnd?: string;
  sampleIntervalSeconds?: number;
  valueDomain?: [number, number];
  threshold?: number;
  historyUnit?: string;
  onOpenDetails?: () => void;
  detailButtonLabel?: string;
  inspectionStore?: Map<string, ChartInspection>;
}

/**
 * Severity is the card's state, expressed once and consumed twice — as the
 * figure's colour and as the leading-edge stripe's pattern. Two channels means
 * the state survives greyscale printing and colour-blind vision.
 */
type Severity = 'unread' | 'ungraded' | 'nominal' | 'elevated' | 'critical';

const SEVERITY_TEXT_CLASS: Record<Severity, string> = {
  unread: 'text-muted',
  // Read, but no bar exists to judge it against. It takes the same neutral ink
  // as `nominal` because the READING is trustworthy — what is missing is the
  // verdict, not the number.
  ungraded: 'text-heading',
  nominal: 'text-heading',
  elevated: 'text-warning',
  critical: 'text-error'
};

export const MetricCard = ({
  type,
  label,
  unit,
  value,
  metric,
  valueObservedAt,
  secondaryText,
  emptyHistoryMessage,
  currentUnavailableReason,
  isExpanded,
  onToggle,
  details,
  alertCount,
  history,
  historyWindowSeconds,
  windowStart,
  windowEnd,
  sampleIntervalSeconds,
  valueDomain,
  threshold,
  historyUnit,
  onOpenDetails,
  detailButtonLabel = 'View detail',
  inspectionStore
}: MetricCardProps) => {
  const state = metric?.state;
  const metricMeasured = state?.case === 'measured' ? state.value : undefined;
  const metricReason = currentUnavailableReason ?? (state?.case === 'unsupportedReason' || state?.case === 'failedError' ? state.value : undefined);
  const hasNumericValue = typeof metricMeasured === 'number'
    ? Number.isFinite(metricMeasured)
    : typeof value === 'number' && Number.isFinite(value);
  const resolvedValue = hasNumericValue ? (typeof metricMeasured === 'number' ? metricMeasured : value ?? null) : null;
  const isUnavailable = Boolean(currentUnavailableReason || (metric && state?.case !== 'measured'));
  const observedAt = metric?.observedAt ? timestampDate(metric.observedAt).toLocaleTimeString() : valueObservedAt ? new Date(valueObservedAt).toLocaleTimeString() : undefined;

  // A current quantity is not a server condition. Do not infer severity from
  // fixed client-side bands; the separate alert badge carries server state.
  const severity: Severity = isUnavailable || !hasNumericValue ? 'unread' : 'ungraded';

  /**
   * One rule for the whole grid: the trace is always the neutral series
   * colour, and severity is carried by the stripe and the threshold line
   * alone. Keying the trace off the card TYPE instead produced cards whose
   * stripe said "threshold crossed" in amber while the line underneath sat in
   * an unrelated blue — two channels stating different things about the same
   * reading. The phosphor accent is the app's one datum colour; a metric's
   * identity is its label, not its hue.
   */
  const sparklineColor = 'var(--chart-line-1)';

  const sparklineThreshold = threshold;

  /**
   * A connection count is an integer, and `#` is a symbol rather than a unit.
   * The grid still hands this card `unit="#"` (see MetricsGrid), so the card
   * resolves both here: counts read "499 connections", not "499.0 #".
   */
  const isCount = type === 'network' || unit === '#';
  const displayUnit = isCount ? 'connections' : unit;
  const valuePrecision = isCount ? 0 : 1;

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

  // A collector that reported "unsupported" or "failed" has no current reading,
  // so its history is stale by definition. Drawing it under the failure reason
  // reads as live telemetry contradicting its own error message.
  const showTrace = Boolean(history && history.length > 0);

  const renderExpandedContent = () => {
    if (!isExpanded || !details) return null;
    const Expansion = expansionMap[type];
    if (!Expansion) return null;
    return <Expansion details={details} />;
  };

  const figureText = isUnavailable && state?.case === 'failedError'
    ? '⚠'
    : resolvedValue !== null
      ? resolvedValue.toFixed(valuePrecision)
      : '—';
  const showUnit = Boolean(displayUnit) && resolvedValue !== null && !isUnavailable;
  const spokenValue = resolvedValue !== null
    ? `${resolvedValue.toFixed(valuePrecision)}${isCount ? ` ${displayUnit}` : displayUnit}`
    : '';

  return (
    <div
      className={`metric-card readout-card expandable ${isExpanded ? 'expanded' : ''}`}
      data-severity={severity}
      onClick={onToggle}
    >
      <span className="readout-card__stripe" aria-hidden="true" />

      <div className="metric-header">
        <span className="metric-label readout-card__label">
          {label}
        </span>

        <div className="flex-row-center gap-sm">
          {alertCount > 0 && (
            <span className="alert-badge">
              {alertCount}
            </span>
          )}

          <button
            type="button"
            className="readout-toggle"
            aria-expanded={isExpanded}
            aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${label}`}
            onClick={event => {
              event.stopPropagation();
              onToggle();
            }}
          >
            {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
          </button>
        </div>
      </div>

      {/* Figure and unit are one lockup on one baseline. */}
      <div className={`metric-value readout-figure ${SEVERITY_TEXT_CLASS[severity]}`}>
        <span className="readout-figure__value">{figureText}</span>
        {showUnit && <span className="readout-figure__unit">{displayUnit}</span>}
      </div>

      <div
        className="sr-only"
        role={hasNumericValue && !isUnavailable ? 'meter' : 'status'}
        aria-label={isUnavailable || !hasNumericValue ? `${label}: ${metricReason ?? 'not measured'}` : `${label}: ${spokenValue}`}
        aria-valuemin={hasNumericValue && !isUnavailable ? 0 : undefined}
        aria-valuemax={hasNumericValue && !isUnavailable && unit === '%' ? 100 : undefined}
        aria-valuenow={hasNumericValue && !isUnavailable && resolvedValue !== null ? resolvedValue : undefined}
      />
      {isUnavailable && (
        <span className="metric-state-reason readout-reason" title={metricReason}>
          {metricReason ?? 'Not measured'}
        </span>
      )}
      {observedAt && <div className="metric-observed-at readout-caption">Measured {observedAt}</div>}
      {secondaryText && <div className="readout-caption">{secondaryText}</div>}

      <div className="overview-chart" data-sm-style="sm-style-254feda622">
        {isUnavailable && showTrace ? <div className="readout-caption">Historical observations; current reading unavailable</div> : null}
        <MetricSparkline
          inspectionStore={inspectionStore}
          inspectionKey={type}
          data={history}
          emptyMessage={emptyHistoryMessage}
          windowStart={windowStart}
          windowEnd={windowEnd}
          height={112}
          sampleIntervalSeconds={sampleIntervalSeconds}
          color={sparklineColor}
          valueDomain={valueDomain}
          threshold={sparklineThreshold}
          unit={sparklineUnit}
          precision={historyUnit ? 1 : valuePrecision}
          windowLabel={formatWindowLabel(historyWindowSeconds)}
          ariaLabel={`${label} history`}
        />
      </div>

      {renderExpandedContent()}

      {onOpenDetails && (
        <div className="readout-detail">
          <button
            type="button"
            className="readout-detail__btn"
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
