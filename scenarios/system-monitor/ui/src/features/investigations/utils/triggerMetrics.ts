/**
 * Pure utility functions for mapping system metrics to trigger cards.
 *
 * This module is the single source of truth for:
 * - Which system metrics map to which trigger IDs
 * - How to compute progress (0-1) toward a trigger threshold
 * - How to format metric values for display
 *
 * All functions are pure (no side effects, no React, no API calls)
 * so they can be unit-tested without any mocking.
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Raw metric values fetched from multiple API endpoints. */
export interface SystemMetricSources {
  /** From /metrics/current */
  cpuUsage?: number;
  /** From /metrics/current (% used) */
  memoryUsage?: number;
  /** From /metrics/current */
  tcpConnections?: number;
  /** From /metrics/detailed → memory_details.disk_usage.percent */
  diskUsagePercent?: number;
  /** From /metrics/process-monitor → process_health (zombies + high-thread-count) */
  anomalousProcessCount?: number;
}

/** Trigger condition direction. */
export type TriggerDirection = 'above' | 'below';

/** Minimal trigger shape needed for metric computations. */
export interface TriggerMetricInput {
  id: string;
  threshold: number;
  condition: TriggerDirection;
  unit: string;
}

// ---------------------------------------------------------------------------
// Metric mapping
// ---------------------------------------------------------------------------

/**
 * Build a lookup of trigger ID → current metric value from raw system metrics.
 *
 * The memory_pressure trigger measures *available* memory (100 - used),
 * since its condition is "below" (fires when available drops below threshold).
 */
export function buildMetricValues(sources: SystemMetricSources): Record<string, number> {
  const values: Record<string, number> = {};

  if (typeof sources.cpuUsage === 'number') {
    values['high_cpu'] = sources.cpuUsage;
  }
  if (typeof sources.memoryUsage === 'number') {
    // API reports % used; trigger threshold is on % available
    values['memory_pressure'] = 100 - sources.memoryUsage;
  }
  if (typeof sources.tcpConnections === 'number') {
    values['network_connections'] = sources.tcpConnections;
  }
  if (typeof sources.diskUsagePercent === 'number') {
    values['disk_space'] = sources.diskUsagePercent;
  }
  if (typeof sources.anomalousProcessCount === 'number') {
    values['process_anomaly'] = sources.anomalousProcessCount;
  }

  return values;
}

// ---------------------------------------------------------------------------
// Progress computation
// ---------------------------------------------------------------------------

/**
 * Compute how close a trigger is to firing, as a 0–1 ratio.
 *
 * For "above" triggers (e.g. CPU > 95%):
 *   progress = currentValue / threshold
 *   0% CPU  → 0.0,  47% CPU → ~0.5,  95% CPU → 1.0
 *
 * For "below" triggers (e.g. available memory < 10%):
 *   The trigger fires when the value drops *below* the threshold.
 *   We define a "safe" reference at 100 for percentage units, or
 *   threshold × 10 for count-based units, then compute:
 *     progress = 1 − (value − threshold) / (safeRef − threshold)
 *   So: value at safeRef → 0.0 (safe), value at threshold → 1.0 (firing).
 *
 * Returns 0 when currentValue is unavailable.
 */
export function computeTriggerProgress(
  currentValue: number | undefined,
  threshold: number,
  condition: TriggerDirection,
  unit: string,
): number {
  if (typeof currentValue !== 'number') return 0;
  if (threshold <= 0) return 0;

  if (condition === 'above') {
    return clamp01(currentValue / threshold);
  }

  // "below" condition: bar fills as value drops toward threshold
  const safeRef = unit === '%' ? 100 : threshold * 10;
  const range = safeRef - threshold;
  if (range <= 0) return 1;
  return clamp01(1 - (currentValue - threshold) / range);
}

// ---------------------------------------------------------------------------
// Color mapping
// ---------------------------------------------------------------------------

/** Map a 0–1 progress ratio to a CSS color variable. */
export function getProgressColor(progress: number): string {
  if (progress < 0.5) return 'var(--color-success)';
  if (progress < 0.8) return 'var(--color-warning)';
  return 'var(--color-error)';
}

// ---------------------------------------------------------------------------
// Display formatting
// ---------------------------------------------------------------------------

/** Format a metric value for display alongside its unit. */
export function formatMetricValue(value: number, unit: string): string {
  if (unit === '%') return `${Math.round(value)}%`;
  if (Number.isInteger(value)) return `${value}`;
  return value.toFixed(1);
}

/** Format a trigger's current/threshold readout string. */
export function formatTriggerReadout(
  currentValue: number | undefined,
  threshold: number,
  unit: string,
): string {
  if (typeof currentValue !== 'number') {
    return `— / ${threshold}${unit}`;
  }
  return `${formatMetricValue(currentValue, unit)} / ${threshold}${unit}`;
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

function clamp01(value: number): number {
  return Math.min(Math.max(value, 0), 1);
}
