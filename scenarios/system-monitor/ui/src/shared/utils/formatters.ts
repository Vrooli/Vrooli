/**
 * Shared formatting utilities for the system-monitor UI.
 *
 * Extracted from MetricCard, MetricDetailViews, and InfrastructureMonitor
 * to eliminate duplication (React Coherence audit).
 */

import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { timestampDate } from '@bufbuild/protobuf/wkt';

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

/** Format a timestamp into a short time string (e.g. "14:05:30"). */
export const formatTimeLabel = (timestamp: string): string => {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
};

/** Format a number as a rounded, locale-aware integer string. */
export const formatInteger = (value: number): string => Math.round(value).toLocaleString();

/** Format a number as MB/s (e.g. "1.23 MB/s"). */
export const formatMbPerSecond = (value: number): string => `${value.toFixed(2)} MB/s`;

/** Return a CSS color variable based on utilization thresholds. */
export const getUtilizationColor = (percent: number): string =>
  percent >= 85 ? 'var(--color-error)' : percent >= 70 ? 'var(--color-warning)' : 'var(--color-success)';

/** Format seconds into a compact duration (e.g. "42s", "5m 30s", "2h 15m"). */
export const formatDurationSeconds = (seconds: number): string => {
  const rounded = Math.round(seconds);
  if (rounded < 1) return '< 1s';
  if (rounded < 60) return `${rounded}s`;
  if (rounded < 3600) {
    const mins = Math.floor(rounded / 60);
    const secs = rounded % 60;
    return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
  }
  const hours = Math.floor(rounded / 3600);
  const mins = Math.floor((rounded % 3600) / 60);
  return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
};

/** Format elapsed time since a start timestamp (e.g. "12s elapsed", "5m elapsed"). */
export const formatDurationElapsed = (startTime: string): string => {
  const start = new Date(startTime);
  if (Number.isNaN(start.getTime())) {
    return 'unknown duration';
  }
  const seconds = Math.max(0, Math.floor((Date.now() - start.getTime()) / 1000));
  if (seconds < 60) {
    return `${seconds}s elapsed`;
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    const remaining = seconds % 60;
    return `${minutes}m ${remaining}s elapsed`;
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m elapsed`;
};

/** Format a timestamp for chart axes (e.g. "14:05"). */
export const formatChartTime = (timestamp: string): string => {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
};

/** Format a protobuf Timestamp to a locale string, or "Unknown". */
export const formatTimestampDisplay = (ts: { seconds?: bigint; nanos?: number } | undefined): string => {
  if (!ts) {
    return 'Unknown';
  }
  const ms = Number(ts.seconds ?? 0n) * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000);
  const date = new Date(ms);
  if (Number.isNaN(date.getTime())) {
    return 'Unknown';
  }
  return date.toLocaleString();
};

/** Format a history window in seconds as a label (e.g. "Past 30s", "Past 5.0m"). */
export const formatWindowLabel = (seconds?: number): string | undefined => {
  if (!seconds) return undefined;
  if (seconds < 60) {
    return `Past ${seconds}s`;
  }
  const minutes = seconds / 60;
  if (Number.isInteger(minutes)) {
    return `Past ${minutes.toFixed(0)}m`;
  }
  return `Past ${minutes.toFixed(1)}m`;
};

/** Format an optional number to a fixed-decimal string, or em-dash. */
export const formatOptionalNumber = (value: number | undefined | null, decimals = 1): string =>
  value != null && Number.isFinite(value) ? value.toFixed(decimals) : '—';

/** Format a timestamp (ISO string or epoch ms) into a locale time string. */
export const formatTime = (isoOrMs: string | number): string => {
  const date = new Date(isoOrMs);
  if (Number.isNaN(date.getTime())) {
    return String(isoOrMs);
  }
  return date.toLocaleTimeString();
};

/** Format a protobuf Timestamp to a short time label (composes timestampDate + formatTimeLabel). */
export const formatProtoTimestamp = (ts: Timestamp): string =>
  formatTimeLabel(timestampDate(ts).toISOString());
