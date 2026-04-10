/**
 * Unit tests for format utilities.
 * Tests pure formatting functions used across the UI.
 *
 * [REQ:LD-UI-TIMELINE] Tests formatRelativeTime, formatShortDate
 * [REQ:LD-UI-STORAGE] Tests formatBytes
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  formatRelativeTime,
  formatShortDate,
  formatDateTime,
  formatDate,
  formatBytes,
} from './format';

describe('formatRelativeTime', () => {
  beforeEach(() => {
    // Mock Date to ensure consistent tests
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-03-10T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns "just now" for timestamps less than 1 minute ago', () => {
    const now = new Date('2026-03-10T12:00:00Z').toISOString();
    expect(formatRelativeTime(now)).toBe('just now');
  });

  it('returns minutes ago for timestamps within an hour', () => {
    const fiveMinutesAgo = new Date('2026-03-10T11:55:00Z').toISOString();
    expect(formatRelativeTime(fiveMinutesAgo)).toBe('5m ago');

    const thirtyMinutesAgo = new Date('2026-03-10T11:30:00Z').toISOString();
    expect(formatRelativeTime(thirtyMinutesAgo)).toBe('30m ago');
  });

  it('returns hours ago for timestamps within a day', () => {
    const twoHoursAgo = new Date('2026-03-10T10:00:00Z').toISOString();
    expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');

    const twelveHoursAgo = new Date('2026-03-10T00:00:00Z').toISOString();
    expect(formatRelativeTime(twelveHoursAgo)).toBe('12h ago');
  });

  it('returns days ago for timestamps older than 24 hours', () => {
    const oneDayAgo = new Date('2026-03-09T12:00:00Z').toISOString();
    expect(formatRelativeTime(oneDayAgo)).toBe('1d ago');

    const threeDaysAgo = new Date('2026-03-07T12:00:00Z').toISOString();
    expect(formatRelativeTime(threeDaysAgo)).toBe('3d ago');
  });

  it('handles boundary case at 59 minutes', () => {
    const fiftyNineMinutesAgo = new Date('2026-03-10T11:01:00Z').toISOString();
    expect(formatRelativeTime(fiftyNineMinutesAgo)).toBe('59m ago');
  });

  it('handles boundary case at 23 hours', () => {
    const twentyThreeHoursAgo = new Date('2026-03-09T13:00:00Z').toISOString();
    expect(formatRelativeTime(twentyThreeHoursAgo)).toBe('23h ago');
  });
});

describe('formatShortDate', () => {
  it('formats date with weekday, month, and day', () => {
    // Note: toLocaleDateString output depends on locale but the format should be consistent
    const result = formatShortDate('2026-03-10T12:00:00Z');
    // Should contain day of week, month, and day number
    expect(result).toMatch(/\w{3}/); // weekday abbreviation
    expect(result).toMatch(/Mar/); // month
    expect(result).toMatch(/10/); // day
  });

  it('handles different dates correctly', () => {
    const result = formatShortDate('2026-01-15T08:30:00Z');
    expect(result).toMatch(/Jan/);
    expect(result).toMatch(/15/);
  });
});

describe('formatDateTime', () => {
  it('formats date with month, day, year and time', () => {
    const result = formatDateTime('2026-03-10T15:45:00Z');
    expect(result).toMatch(/Mar/);
    expect(result).toMatch(/2026/);
    // Time format varies by timezone but should include hour/minute
    expect(result).toMatch(/\d{1,2}:\d{2}/);
  });

  it('includes all date components', () => {
    // Use a mid-day time that won't shift across day boundaries in most timezones
    const result = formatDateTime('2026-03-10T12:00:00Z');
    expect(result).toMatch(/Mar/);
    expect(result).toMatch(/2026/);
    expect(result).toMatch(/\d{1,2}:\d{2}/);
  });
});

describe('formatDate', () => {
  it('formats date with month, day, and year', () => {
    const result = formatDate('2026-03-10T12:00:00Z');
    expect(result).toMatch(/Mar/);
    expect(result).toMatch(/10/);
    expect(result).toMatch(/2026/);
  });

  it('returns default fallback for undefined', () => {
    expect(formatDate(undefined)).toBe('-');
  });

  it('returns custom fallback for undefined', () => {
    expect(formatDate(undefined, 'N/A')).toBe('N/A');
  });

  it('returns fallback for invalid date string', () => {
    expect(formatDate('not-a-date')).toBe('-');
  });

  it('handles empty string', () => {
    expect(formatDate('')).toBe('-');
  });
});

describe('formatBytes', () => {
  it('returns "0 B" for zero bytes', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  it('formats bytes correctly', () => {
    expect(formatBytes(512)).toBe('512.0 B');
    expect(formatBytes(1023)).toBe('1023.0 B');
  });

  it('formats kilobytes correctly', () => {
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(10240)).toBe('10.0 KB');
  });

  it('formats megabytes correctly', () => {
    expect(formatBytes(1048576)).toBe('1.0 MB'); // 1 MB
    expect(formatBytes(1572864)).toBe('1.5 MB'); // 1.5 MB
    expect(formatBytes(104857600)).toBe('100.0 MB'); // 100 MB
  });

  it('formats gigabytes correctly', () => {
    expect(formatBytes(1073741824)).toBe('1.0 GB'); // 1 GB
    expect(formatBytes(2147483648)).toBe('2.0 GB'); // 2 GB
  });

  it('caps at GB for very large values', () => {
    // Very large value should still show as GB
    expect(formatBytes(10737418240)).toBe('10.0 GB'); // 10 GB
  });

  it('handles edge case near unit boundaries', () => {
    expect(formatBytes(1000)).toBe('1000.0 B');
    expect(formatBytes(1025)).toBe('1.0 KB'); // Just over 1 KB
  });
});
