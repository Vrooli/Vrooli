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

/**
 * Merge memory and swap series into one sorted dataset keyed by timestamp.
 *
 * The two are plotted together deliberately: memory utilisation can sit in a
 * healthy band while swap fills, and that divergence is invisible on a
 * memory-only chart. Points are kept `undefined` rather than zero when a
 * series has no reading at that timestamp, so recharts breaks the line instead
 * of drawing a false drop to 0%.
 */
export const combineMemorySeries = (
  memorySeries?: ChartDataPoint[],
  swapSeries?: ChartDataPoint[]
) => {
  const combined = new Map<string, { timestamp: string; memory?: number; swap?: number }>();
  const upsert = (point: ChartDataPoint, key: 'memory' | 'swap') => {
    const value = Number(point.value);
    if (!Number.isFinite(value)) return;
    const existing = combined.get(point.timestamp) ?? { timestamp: point.timestamp };
    existing[key] = value;
    combined.set(point.timestamp, existing);
  };
  (memorySeries ?? []).forEach(point => upsert(point, 'memory'));
  (swapSeries ?? []).forEach(point => upsert(point, 'swap'));
  return Array.from(combined.values()).sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  );
};
