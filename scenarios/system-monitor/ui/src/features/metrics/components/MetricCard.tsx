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
  detailButtonLabel = 'View detail'
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

  /**
   * The authored attention thresholds, by card. Network is absent ON PURPOSE:
   * its value is a connection count and nobody has declared what "too many"
   * is on this host. An absent entry means "ungraded", never "fine".
   */
  const defaultThresholds: Partial<Record<CardType, number>> = {
    cpu: 75,
    memory: 80,
    disk: 80,
    gpu: 85
  };

  const authoredThreshold = typeof threshold === 'number' ? threshold : defaultThresholds[type];
  /** Whether this card's value is on a 0-100 percentage scale. */
  const isPercentage = unit === '%';

  /**
   * `null` — not `0` — when nothing was measured. A zero-length bar is a
   * fabricated reading of zero, which is a different and false claim from
   * "this was never read"; callers must render the unread treatment instead.
   */
  const getProgressValue = (): number | null => {
    if (!hasNumericValue || resolvedValue === null || isUnavailable) {
      return null;
    }
    // The track is a 0-100 percentage track. Clamping a connection count of
    // 551 onto it drew a permanently full bar that meant nothing, so a
    // non-percentage metric gets no bar rather than a misleading one.
    if (!isPercentage) {
      return null;
    }
    return Math.min(resolvedValue, 100);
  };

  /**
   * Severity is graded against a bar that is APPROPRIATE TO THE UNIT.
   *
   * This previously compared every value against 70 and 90 regardless of what
   * the number meant. Three of the five cards are percentages, so that read
   * correctly — but the network card's value is a CONNECTION COUNT, and a host
   * with 551 open connections therefore sat permanently at `critical` with a
   * red stripe and a red figure. It was not reporting a busy host; it was
   * comparing connections against percent.
   *
   * So: a percentage grades on the percentage bars. Anything else grades only
   * against an explicitly authored `threshold`, and if none exists the card
   * reports `ungraded` — read, trustworthy, but with nothing to judge it
   * against. It deliberately does NOT fall through to `nominal`, because
   * "nobody set a limit" and "it is within limits" are different claims and
   * only one of them is true here.
   */
  const getSeverity = (): Severity => {
    if (isUnavailable || !hasNumericValue || resolvedValue === null) {
      return 'unread';
    }
    if (isPercentage) {
      if (resolvedValue >= 90) return 'critical';
      if (resolvedValue >= 70) return 'elevated';
      return 'nominal';
    }
    if (typeof authoredThreshold === 'number') {
      if (resolvedValue >= authoredThreshold) return 'elevated';
      return 'nominal';
    }
    return 'ungraded';
  };

  const severity = getSeverity();
  const progressValue = getProgressValue();

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

  const sparklineThreshold = authoredThreshold;

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
  const showTrace = !isUnavailable && Boolean(history && history.length > 0);

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
        aria-valuemax={hasNumericValue && !isUnavailable && type !== 'network' ? 100 : undefined}
        aria-valuenow={hasNumericValue && !isUnavailable && resolvedValue !== null ? resolvedValue : undefined}
      />
      {isUnavailable && (
        <span className="metric-state-reason readout-reason" title={metricReason}>
          {metricReason ?? 'Not measured'}
        </span>
      )}
      {observedAt && <div className="metric-observed-at readout-caption">Measured {observedAt}</div>}

      {showTrace ? (
        <div data-sm-style="sm-style-254feda622">
          <MetricSparkline
            data={history}
            color={sparklineColor}
            valueDomain={valueDomain}
            threshold={sparklineThreshold}
            unit={sparklineUnit}
            precision={historyUnit ? 1 : valuePrecision}
            windowLabel={formatWindowLabel(historyWindowSeconds)}
            ariaLabel={`${label} history${resolvedValue !== null ? `, latest ${spokenValue}` : ', not measured'}`}
          />
        </div>
      ) : progressValue !== null ? (
        <div className="metric-bar">
          <div
            className="metric-fill"
            style={{ width: `${progressValue}%` }}
          />
        </div>
      ) : severity === 'ungraded' ? (
        /*
         * Read, but not on a 0-100 scale, so there is no track to fill. It
         * must NOT fall through to the unread bar below: that treatment means
         * "never measured", and this value was measured — it simply has no
         * percentage to plot. Rendering nothing is the honest option.
         */
        null
      ) : (
        <div
          className="metric-bar readout-bar--unread"
          data-testid="metric-unread-bar"
          aria-hidden="true"
        />
      )}

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
