/**
 * Shared formatting utilities for the system-monitor UI.
 *
 * Extracted from MetricCard, MetricDetailViews, and InfrastructureMonitor
 * to eliminate duplication (React Coherence audit).
 */

/** Format a byte count into a human-readable string (e.g. "1.23 GB"). */
export const formatBytes = (value?: number): string => {
  if (!Number.isFinite(value ?? NaN) || (value ?? 0) <= 0) {
    return '—';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const absolute = Math.max(0, value ?? 0);
  if (absolute === 0) {
    return '0 B';
  }
  const exponent = Math.min(Math.floor(Math.log(absolute) / Math.log(1024)), units.length - 1);
  const scaled = absolute / Math.pow(1024, exponent);
  const precision = scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2;
  return `${scaled.toFixed(precision)} ${units[exponent]}`;
};

/** Format a megabyte value (e.g. "512 MB"). */
export const formatMegabytes = (value?: number | null): string => {
  if (!Number.isFinite(value ?? NaN)) {
    return '—';
  }
  return `${Number(value).toFixed(0)} MB`;
};

/** Format a percentage value (e.g. "72.3%"). */
export const formatPercentage = (value?: number | null): string => {
  if (!Number.isFinite(value ?? NaN)) {
    return '—';
  }
  return `${Number(value).toFixed(1)}%`;
};

/** Return a CSS color variable based on utilization thresholds. */
export const getUtilizationColor = (percent: number): string =>
  percent >= 85 ? 'var(--color-error)' : percent >= 70 ? 'var(--color-warning)' : 'var(--color-success)';
