import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  isValidDate,
  formatDateTime,
  formatDateOnly,
  formatMonthYear,
  calculateDaysSince,
  formatRelativeTime,
  formatFeedbackTimestamp,
  getCurrentPeriod,
  navigatePeriod,
  isCurrentPeriod,
  DAY_MS,
} from './dateFormatters';

describe('dateFormatters', () => {
  describe('isValidDate', () => {
    it('returns true for valid ISO date strings', () => {
      expect(isValidDate('2024-01-15T14:30:00Z')).toBe(true);
      expect(isValidDate('2024-01-15')).toBe(true);
      expect(isValidDate('2024-01-15T14:30:00.000Z')).toBe(true);
    });

    it('returns false for null or undefined', () => {
      expect(isValidDate(null)).toBe(false);
      expect(isValidDate(undefined)).toBe(false);
    });

    it('returns false for empty string', () => {
      expect(isValidDate('')).toBe(false);
    });

    it('returns false for invalid date strings', () => {
      expect(isValidDate('not-a-date')).toBe(false);
      expect(isValidDate('2024-13-45')).toBe(false);
    });
  });

  describe('formatDateTime', () => {
    it('formats date with full style by default', () => {
      const result = formatDateTime('2024-01-15T14:30:00Z', 'full');
      // The exact format depends on locale, but it should contain the date and time
      expect(result).toContain('2024');
      expect(result).not.toBe('-');
    });

    it('formats date with short style', () => {
      const result = formatDateTime('2024-01-15T14:30:00Z', 'short');
      // Short format uses 2-digit year
      expect(result).toContain('24');
      expect(result).not.toBe('-');
    });

    it('returns "-" for null', () => {
      expect(formatDateTime(null)).toBe('-');
    });

    it('returns "-" for undefined', () => {
      expect(formatDateTime(undefined)).toBe('-');
    });

    it('returns "-" for invalid date', () => {
      expect(formatDateTime('not-a-date')).toBe('-');
    });

    it('defaults to full style', () => {
      const fullResult = formatDateTime('2024-01-15T14:30:00Z');
      const explicitFullResult = formatDateTime('2024-01-15T14:30:00Z', 'full');
      expect(fullResult).toBe(explicitFullResult);
    });
  });

  describe('formatDateOnly', () => {
    it('formats date without time', () => {
      const result = formatDateOnly('2024-01-15T14:30:00Z');
      expect(result).not.toBeNull();
      expect(result).toContain('2024');
      // Should not contain time indicators
      expect(result).not.toContain(':');
    });

    it('returns null for null input', () => {
      expect(formatDateOnly(null)).toBeNull();
    });

    it('returns null for undefined input', () => {
      expect(formatDateOnly(undefined)).toBeNull();
    });

    it('returns null for invalid date', () => {
      expect(formatDateOnly('not-a-date')).toBeNull();
    });

    it('handles date-only strings', () => {
      const result = formatDateOnly('2024-01-15');
      expect(result).not.toBeNull();
    });
  });

  describe('formatMonthYear', () => {
    it('formats YYYY-MM to "Month Year"', () => {
      expect(formatMonthYear('2024-01')).toBe('January 2024');
      expect(formatMonthYear('2024-12')).toBe('December 2024');
      expect(formatMonthYear('2023-06')).toBe('June 2023');
    });

    it('handles single-digit months', () => {
      expect(formatMonthYear('2024-1')).toBe('January 2024');
    });

    it('returns input as-is for invalid format', () => {
      expect(formatMonthYear('invalid')).toBe('invalid');
      expect(formatMonthYear('')).toBe('');
    });
  });

  describe('calculateDaysSince', () => {
    beforeEach(() => {
      // Fix the current time for consistent testing
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-01-20T12:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns 0 for today', () => {
      expect(calculateDaysSince('2024-01-20T00:00:00Z')).toBe(0);
    });

    it('returns 1 for yesterday', () => {
      expect(calculateDaysSince('2024-01-19T00:00:00Z')).toBe(1);
    });

    it('returns correct count for multiple days ago', () => {
      expect(calculateDaysSince('2024-01-15T00:00:00Z')).toBe(5);
      expect(calculateDaysSince('2024-01-10T00:00:00Z')).toBe(10);
    });

    it('returns null for null input', () => {
      expect(calculateDaysSince(null)).toBeNull();
    });

    it('returns null for undefined input', () => {
      expect(calculateDaysSince(undefined)).toBeNull();
    });

    it('returns null for invalid date', () => {
      expect(calculateDaysSince('not-a-date')).toBeNull();
    });

    it('returns 0 for future dates (floors to 0)', () => {
      expect(calculateDaysSince('2024-01-25T00:00:00Z')).toBe(0);
    });
  });

  describe('formatRelativeTime', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-01-20T12:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns "today" for today', () => {
      expect(formatRelativeTime('2024-01-20T00:00:00Z')).toBe('today');
    });

    it('returns "yesterday" for yesterday', () => {
      expect(formatRelativeTime('2024-01-19T00:00:00Z')).toBe('yesterday');
    });

    it('returns "{days} days ago" for older dates', () => {
      expect(formatRelativeTime('2024-01-15T00:00:00Z')).toBe('5 days ago');
      expect(formatRelativeTime('2024-01-10T00:00:00Z')).toBe('10 days ago');
    });

    it('returns "Never" for null', () => {
      expect(formatRelativeTime(null)).toBe('Never');
    });

    it('returns "Never" for undefined', () => {
      expect(formatRelativeTime(undefined)).toBe('Never');
    });

    it('returns "Never" for invalid date', () => {
      expect(formatRelativeTime('not-a-date')).toBe('Never');
    });

    describe('with custom options', () => {
      it('uses custom nullLabel', () => {
        expect(formatRelativeTime(null, { nullLabel: 'Never customized' })).toBe('Never customized');
      });

      it('uses custom todayLabel', () => {
        expect(formatRelativeTime('2024-01-20T00:00:00Z', { todayLabel: 'Just now' })).toBe('just now');
      });

      it('uses custom yesterdayLabel', () => {
        expect(formatRelativeTime('2024-01-19T00:00:00Z', { yesterdayLabel: 'One day ago' })).toBe('one day ago');
      });

      it('uses custom daysAgoTemplate', () => {
        expect(
          formatRelativeTime('2024-01-15T00:00:00Z', { daysAgoTemplate: '{days}d ago' })
        ).toBe('5d ago');
      });

      it('adds prefix to all labels', () => {
        expect(formatRelativeTime('2024-01-20T00:00:00Z', { prefix: 'Updated ' })).toBe('Updated today');
        expect(formatRelativeTime('2024-01-19T00:00:00Z', { prefix: 'Updated ' })).toBe('Updated yesterday');
        expect(formatRelativeTime('2024-01-15T00:00:00Z', { prefix: 'Updated ' })).toBe('Updated 5 days ago');
      });

      it('does not apply prefix to nullLabel', () => {
        expect(formatRelativeTime(null, { prefix: 'Updated ', nullLabel: 'Never' })).toBe('Never');
      });
    });
  });

  describe('formatFeedbackTimestamp', () => {
    it('formats date in detailed feedback format', () => {
      const result = formatFeedbackTimestamp('2024-01-15T14:30:00Z');
      // Should contain month abbreviation, day, year, and time
      expect(result).toContain('Jan');
      expect(result).toContain('15');
      expect(result).toContain('2024');
    });

    it('returns input as-is for invalid date', () => {
      expect(formatFeedbackTimestamp('not-a-date')).toBe('not-a-date');
    });

    it('handles different months correctly', () => {
      expect(formatFeedbackTimestamp('2024-06-01T10:00:00Z')).toContain('Jun');
      expect(formatFeedbackTimestamp('2024-12-25T10:00:00Z')).toContain('Dec');
    });
  });

  describe('getCurrentPeriod', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns current YYYY-MM', () => {
      vi.setSystemTime(new Date('2024-01-15T12:00:00Z'));
      expect(getCurrentPeriod()).toBe('2024-01');
    });

    it('pads single-digit months with zero', () => {
      vi.setSystemTime(new Date('2024-05-15T12:00:00Z'));
      expect(getCurrentPeriod()).toBe('2024-05');
    });

    it('handles December correctly', () => {
      vi.setSystemTime(new Date('2024-12-15T12:00:00Z'));
      expect(getCurrentPeriod()).toBe('2024-12');
    });
  });

  describe('navigatePeriod', () => {
    it('navigates forward one month', () => {
      expect(navigatePeriod('2024-01', 1)).toBe('2024-02');
      expect(navigatePeriod('2024-05', 1)).toBe('2024-06');
    });

    it('navigates backward one month', () => {
      expect(navigatePeriod('2024-02', -1)).toBe('2024-01');
      expect(navigatePeriod('2024-06', -1)).toBe('2024-05');
    });

    it('handles year boundary going forward', () => {
      expect(navigatePeriod('2024-12', 1)).toBe('2025-01');
    });

    it('handles year boundary going backward', () => {
      expect(navigatePeriod('2024-01', -1)).toBe('2023-12');
    });

    it('handles multi-month navigation', () => {
      expect(navigatePeriod('2024-01', 3)).toBe('2024-04');
      expect(navigatePeriod('2024-06', -6)).toBe('2023-12');
    });
  });

  describe('isCurrentPeriod', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-01-15T12:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns true for current month', () => {
      expect(isCurrentPeriod('2024-01')).toBe(true);
    });

    it('returns false for past months', () => {
      expect(isCurrentPeriod('2023-12')).toBe(false);
      expect(isCurrentPeriod('2024-00')).toBe(false); // Would be invalid but still false
    });

    it('returns false for future months', () => {
      expect(isCurrentPeriod('2024-02')).toBe(false);
      expect(isCurrentPeriod('2025-01')).toBe(false);
    });
  });

  describe('DAY_MS constant', () => {
    it('equals milliseconds in a day', () => {
      expect(DAY_MS).toBe(24 * 60 * 60 * 1000);
      expect(DAY_MS).toBe(86400000);
    });
  });
});
