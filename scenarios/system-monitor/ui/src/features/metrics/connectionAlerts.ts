import type { MetricValue } from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';
import type { ChartDataPoint } from '../../types';

/**
 * Connection-count alerting.
 *
 * The card previously hard-coded `alertCount={0}`, so on 2026-08-21 a host
 * carrying 57,706 established TCP connections — 93% of them held by a single
 * leaking process — displayed the number without ever raising an alert.
 *
 * Two rules, because one is not enough:
 *
 * 1. Level. A high absolute count is worth flagging on its own.
 * 2. Growth. A leak is a *shape*, not a level: it climbs monotonically from a
 *    normal-looking baseline. Waiting for the level rule means waiting until the
 *    host is already in trouble, so sustained growth alerts earlier and needs no
 *    per-host tuning.
 */

/** A count at or above this is unambiguously wrong for a single host. */
export const CONNECTIONS_CRITICAL = 20_000;

/** Elevated: plausible under real load, but worth a look. */
export const CONNECTIONS_WARNING = 5_000;

/**
 * Growth detection only engages above this floor. Below it, doubling from 20 to
 * 40 connections is ordinary activity rather than a leak.
 */
export const GROWTH_FLOOR = 500;

/** Samples required before a trend claim is meaningful. */
export const GROWTH_MIN_SAMPLES = 5;

/** Multiple of the window's opening value that counts as sustained growth. */
export const GROWTH_FACTOR = 2;

export interface ConnectionAlert {
  severity: 'warning' | 'critical';
  reason: string;
}

/**
 * Detects a monotone-enough climb. A strict monotonicity requirement would be
 * defeated by a single sample of jitter, so this asks that the series rise
 * across the window, end at its own maximum, and never give back much ground.
 */
const isSustainedGrowth = (history: readonly ChartDataPoint[]): boolean => {
  if (history.length < GROWTH_MIN_SAMPLES) {
    return false;
  }
  const values = history.map((point) => point.value);
  const first = values[0] ?? 0;
  const last = values[values.length - 1] ?? 0;

  if (last < GROWTH_FLOOR) {
    return false;
  }
  if (first <= 0 || last < first * GROWTH_FACTOR) {
    return false;
  }
  // The series must still be climbing at the end, not recovering from a spike.
  const peak = Math.max(...values);
  if (last < peak * 0.9) {
    return false;
  }
  // No sustained retreat: a genuine leak does not release what it took.
  let deepestDrop = 0;
  for (let index = 1; index < values.length; index++) {
    const previous = values[index - 1] ?? 0;
    const current = values[index] ?? 0;
    deepestDrop = Math.max(deepestDrop, previous - current);
  }
  return deepestDrop <= (last - first) * 0.5;
};

/**
 * Returns the alerts raised by the current connection count and its recent
 * history. `connections` of undefined means "not read yet", which is not an
 * alert — an unread metric must never render as a healthy zero.
 */
export const connectionAlerts = (
  connections: number | undefined | null,
  history?: readonly ChartDataPoint[]
): ConnectionAlert[] => {
  if (connections === undefined || connections === null || !Number.isFinite(connections)) {
    return [];
  }

  const alerts: ConnectionAlert[] = [];

  if (connections >= CONNECTIONS_CRITICAL) {
    alerts.push({
      severity: 'critical',
      reason: `${connections.toLocaleString()} established connections`
    });
  } else if (connections >= CONNECTIONS_WARNING) {
    alerts.push({
      severity: 'warning',
      reason: `${connections.toLocaleString()} established connections`
    });
  }

  if (history && isSustainedGrowth(history)) {
    alerts.push({
      severity: 'warning',
      reason: 'connection count climbing steadily'
    });
  }

  return alerts;
};

/** Convenience for the card, which renders a count badge. */
export const connectionAlertCount = (
  connections: number | undefined | null,
  history?: readonly ChartDataPoint[]
): number => connectionAlerts(connections, history).length;

/**
 * Extracts the measured number from a MetricValue, or undefined for any
 * non-measured state (unsupported, failed, stale, not-yet-sampled).
 *
 * This distinction is the whole point: a metric that could not be read must not
 * be alerted on, and must never be coerced to 0 — an unread connection count and
 * a genuinely idle host are opposite conditions.
 */
export const measuredConnections = (metric?: MetricValue): number | undefined =>
  metric?.state?.case === 'measured' ? metric.state.value : undefined;

/** Alert count for a card holding a MetricValue rather than a bare number. */
export const metricConnectionAlertCount = (
  metric?: MetricValue,
  history?: readonly ChartDataPoint[]
): number => connectionAlertCount(measuredConnections(metric), history);
