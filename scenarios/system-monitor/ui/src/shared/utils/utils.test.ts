import { describe, expect, it } from 'vitest';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { buildSingleSeriesData, combineDiskSeries } from './chartData';
import { getHealthColor, getRiskLevelColor, getStatusColor } from './colors';
import { formatBytes, formatDurationElapsed, formatDurationSeconds, formatInteger, formatMbPerSecond, formatMegabytes, formatOptionalNumber, formatPercentage, formatProtoTimestamp, formatTime, formatTimeLabel, formatTimestampDisplay, formatWindowLabel, getUtilizationColor } from './formatters';
import { sortByTimestamp, toIsoString } from './timestamps';
import { bool, num, str } from './typeGuards';

describe('shared UI utility functions', () => {
  it('builds sorted and merged chart series while dropping invalid values', () => {
    expect(buildSingleSeriesData()).toEqual([]);
    expect(buildSingleSeriesData([
      { timestamp: '2026-01-02T00:00:00Z', value: 2 },
      { timestamp: 'bad', value: Number.NaN },
      { timestamp: '2026-01-01T00:00:00Z', value: '1' as unknown as number },
    ])).toEqual([
      { timestamp: '2026-01-01T00:00:00Z', value: 1 },
      { timestamp: '2026-01-02T00:00:00Z', value: 2 },
    ]);
    expect(combineDiskSeries(
      [{ timestamp: '2026-01-02T00:00:00Z', value: 3 }, { timestamp: '2026-01-01T00:00:00Z', value: 1 }],
      [{ timestamp: '2026-01-02T00:00:00Z', value: 4 }, { timestamp: '2026-01-03T00:00:00Z', value: 5 }],
    )).toEqual([
      { timestamp: '2026-01-01T00:00:00Z', read: 1, write: 0 },
      { timestamp: '2026-01-02T00:00:00Z', read: 3, write: 4 },
      { timestamp: '2026-01-03T00:00:00Z', read: 0, write: 5 },
    ]);
  });

  it('maps status and utilization values to semantic colors', () => {
    expect(getRiskLevelColor('high')).toContain('error');
    expect(getRiskLevelColor('medium')).toContain('warning');
    expect(getRiskLevelColor('low')).toContain('success');
    expect(getHealthColor(true)).toContain('success');
    expect(getHealthColor(false)).toContain('error');
    for (const status of ['critical', 'high', 'unhealthy', 'error']) expect(getStatusColor(status)).toContain('error');
    for (const status of ['medium', 'degraded', 'warning']) expect(getStatusColor(status)).toContain('warning');
    expect(getStatusColor('healthy')).toContain('success');
    expect(getUtilizationColor(90)).toContain('error');
    expect(getUtilizationColor(75)).toContain('warning');
    expect(getUtilizationColor(40)).toContain('success');
  });

  it('formats numbers, durations, windows, timestamps, and invalid values honestly', () => {
    expect(formatBytes()).toBe('—');
    expect(formatBytes(1024)).toBe('1.00 KB');
    expect(formatBytes(1024 * 1024 * 100)).toBe('100 MB');
    expect(formatMegabytes(512)).toBe('512 MB');
    expect(formatMegabytes(null)).toBe('—');
    expect(formatPercentage(12.34)).toBe('12.3%');
    expect(formatMbPerSecond(1.234)).toBe('1.23 MB/s');
    expect(formatInteger(1234.6)).toBe('1,235');
    expect(formatOptionalNumber(null)).toBe('—');
    expect(formatOptionalNumber(1.234, 2)).toBe('1.23');
    expect(formatDurationSeconds(0)).toBe('< 1s');
    expect(formatDurationSeconds(90)).toBe('1m 30s');
    expect(formatDurationSeconds(3600)).toBe('1h');
    expect(formatWindowLabel()).toBeUndefined();
    expect(formatWindowLabel(30)).toBe('Past 30s');
    expect(formatWindowLabel(300)).toBe('Past 5m');
    expect(formatWindowLabel(90)).toBe('Past 1.5m');
    expect(formatTimeLabel('bad')).toBe('bad');
    expect(formatTime('bad')).toBe('bad');
    expect(formatTimestampDisplay(undefined)).toBe('Unknown');
    expect(formatTimestampDisplay({ seconds: 0n, nanos: 0 })).toContain('1969');
    const ts = timestampFromDate(new Date('2026-01-01T00:00:00Z'));
    expect(formatProtoTimestamp(ts)).toMatch(/:00:00/);
    expect(formatDurationElapsed('bad')).toBe('unknown duration');
  });

  it('normalizes timestamps and narrows unknown values', () => {
    expect(toIsoString()).toMatch(/T/);
    expect(toIsoString(timestampFromDate(new Date('2026-01-01T00:00:00Z')))).toBe('2026-01-01T00:00:00.000Z');
    const sorted = sortByTimestamp([{ value: 1 }, { value: 3 }, { value: 2 }], item => item.value === 2 ? Number.NaN : item.value);
    expect(sorted.map(item => item.value)).toEqual([3, 1, 2]);
    expect(str('x')).toBe('x');
    expect(str(1)).toBeUndefined();
    expect(num(1)).toBe(1);
    expect(num('1')).toBeUndefined();
    expect(bool(true)).toBe(true);
    expect(bool('true')).toBeUndefined();
  });
});
