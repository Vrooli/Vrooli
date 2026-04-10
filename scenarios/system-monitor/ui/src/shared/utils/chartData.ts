/**
 * Pure data-builder functions for chart series.
 *
 * Extracted from metricHelpers.tsx so that the JSX render helpers can stay
 * in that file while pure transforms live in shared/utils.
 */

import type { ChartDataPoint } from '../../types';

/** Normalise a time series into sorted {timestamp, value} tuples. */
export const buildSingleSeriesData = (series?: ChartDataPoint[]) => {
  if (!series || series.length === 0) {
    return [] as Array<{ timestamp: string; value: number }>;
  }
  return [...series]
    .map(point => ({ timestamp: point.timestamp, value: Number(point.value) }))
    .filter(point => !Number.isNaN(point.value))
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
};

/** Merge read/write disk series into a single sorted array. */
export const combineDiskSeries = (readSeries?: ChartDataPoint[], writeSeries?: ChartDataPoint[]) => {
  const combined = new Map<string, { timestamp: string; read: number; write: number }>();
  (readSeries ?? []).forEach(point => {
    const existing = combined.get(point.timestamp) ?? { timestamp: point.timestamp, read: 0, write: 0 };
    existing.read = Number(point.value) || 0;
    combined.set(point.timestamp, existing);
  });
  (writeSeries ?? []).forEach(point => {
    const existing = combined.get(point.timestamp) ?? { timestamp: point.timestamp, read: 0, write: 0 };
    existing.write = Number(point.value) || 0;
    combined.set(point.timestamp, existing);
  });
  return Array.from(combined.values()).sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
};
