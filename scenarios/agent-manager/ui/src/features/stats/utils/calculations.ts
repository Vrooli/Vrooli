// Calculation utilities for stats

import type { RunStatusCounts, TimeSeriesBucket } from "../api/types";

/**
 * Calculate success rate from status counts
 */
export function calculateSuccessRate(counts: RunStatusCounts): number {
  const completed = counts.complete + counts.failed;
  if (completed === 0) return 0;
  return counts.complete / completed;
}

/**
 * Calculate throughput (runs per hour) from time series
 */
export function calculateThroughput(buckets: TimeSeriesBucket[], windowHours: number): number {
  const totalCompleted = buckets.reduce((sum, b) => sum + b.runsCompleted, 0);
  if (windowHours === 0) return 0;
  return totalCompleted / windowHours;
}

/**
 * Get window hours from preset
 */
export function getWindowHours(preset: string): number {
  switch (preset) {
    case "6h":
      return 6;
    case "12h":
      return 12;
    case "24h":
      return 24;
    case "7d":
      return 7 * 24;
    case "30d":
      return 30 * 24;
    default:
      return 24;
  }
}

/**
 * Calculate percent change between two values
 */
export function percentChange(current: number, previous: number): number {
  if (previous === 0) return current > 0 ? 100 : 0;
  return ((current - previous) / previous) * 100;
}

/**
 * Calculate total from time series buckets
 */
export function sumBucketField(
  buckets: TimeSeriesBucket[],
  field: keyof Pick<TimeSeriesBucket, "runsStarted" | "runsCompleted" | "runsFailed" | "totalCostUsd">
): number {
  return buckets.reduce((sum, b) => sum + b[field], 0);
}

/**
 * Calculate average from time series buckets (excluding zeros)
 */
export function avgBucketField(
  buckets: TimeSeriesBucket[],
  field: keyof Pick<TimeSeriesBucket, "avgDurationMs" | "totalCostUsd">
): number {
  const nonZero = buckets.filter((b) => b[field] > 0);
  if (nonZero.length === 0) return 0;
  return nonZero.reduce((sum, b) => sum + b[field], 0) / nonZero.length;
}
