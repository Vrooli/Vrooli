import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { WaitlistEmail } from '../../../shared/api';
import {
  formatDate,
  calculateStats,
  filterEmailsBySource,
  searchEmails,
  sortEmailsByDate,
  removeEmailFromList,
  getUniqueSources,
} from './waitlist.service';

const createMockEmail = (overrides: Partial<WaitlistEmail> = {}): WaitlistEmail => ({
  id: 1,
  email: 'test@example.com',
  source: 'coming_soon',
  created_at: '2024-01-15T10:30:00Z',
  ...overrides,
});

describe('waitlist.service', () => {
  describe('formatDate', () => {
    it('formats date string to locale string', () => {
      const result = formatDate('2024-01-15T10:30:00Z');
      // Just verify it returns a non-empty string (locale dependent)
      expect(typeof result).toBe('string');
      expect(result.length).toBeGreaterThan(0);
    });

    it('handles different date formats', () => {
      const result = formatDate('2024-12-25T23:59:59Z');
      expect(typeof result).toBe('string');
    });
  });

  describe('calculateStats', () => {
    it('calculates correct stats for empty list', () => {
      const stats = calculateStats([]);
      expect(stats.totalSignups).toBe(0);
      expect(stats.comingSoonCount).toBe(0);
    });

    it('calculates correct stats for mixed sources', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'coming_soon' }),
        createMockEmail({ id: 2, source: 'coming_soon' }),
        createMockEmail({ id: 3, source: 'newsletter' }),
        createMockEmail({ id: 4, source: 'referral' }),
      ];

      const stats = calculateStats(emails);
      expect(stats.totalSignups).toBe(4);
      expect(stats.comingSoonCount).toBe(2);
    });

    it('calculates correct stats when all from coming_soon', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'coming_soon' }),
        createMockEmail({ id: 2, source: 'coming_soon' }),
      ];

      const stats = calculateStats(emails);
      expect(stats.totalSignups).toBe(2);
      expect(stats.comingSoonCount).toBe(2);
    });

    it('calculates correct stats when none from coming_soon', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'newsletter' }),
        createMockEmail({ id: 2, source: 'referral' }),
      ];

      const stats = calculateStats(emails);
      expect(stats.totalSignups).toBe(2);
      expect(stats.comingSoonCount).toBe(0);
    });
  });

  describe('filterEmailsBySource', () => {
    const emails: WaitlistEmail[] = [
      createMockEmail({ id: 1, source: 'coming_soon' }),
      createMockEmail({ id: 2, source: 'coming_soon' }),
      createMockEmail({ id: 3, source: 'newsletter' }),
    ];

    it('filters emails by source', () => {
      const result = filterEmailsBySource(emails, 'coming_soon');
      expect(result).toHaveLength(2);
      expect(result.every((e) => e.source === 'coming_soon')).toBe(true);
    });

    it('returns empty array for non-matching source', () => {
      const result = filterEmailsBySource(emails, 'nonexistent');
      expect(result).toHaveLength(0);
    });

    it('returns all emails for empty source array', () => {
      const result = filterEmailsBySource([], 'coming_soon');
      expect(result).toHaveLength(0);
    });
  });

  describe('searchEmails', () => {
    const emails: WaitlistEmail[] = [
      createMockEmail({ id: 1, email: 'john@example.com' }),
      createMockEmail({ id: 2, email: 'jane@example.com' }),
      createMockEmail({ id: 3, email: 'bob@test.org' }),
    ];

    it('searches by email substring', () => {
      const result = searchEmails(emails, 'example');
      expect(result).toHaveLength(2);
    });

    it('search is case insensitive', () => {
      const result = searchEmails(emails, 'JOHN');
      expect(result).toHaveLength(1);
      expect(result[0].email).toBe('john@example.com');
    });

    it('returns all emails for empty query', () => {
      const result = searchEmails(emails, '');
      expect(result).toHaveLength(3);
    });

    it('returns all emails for whitespace-only query', () => {
      const result = searchEmails(emails, '   ');
      expect(result).toHaveLength(3);
    });

    it('returns empty array for non-matching query', () => {
      const result = searchEmails(emails, 'nonexistent');
      expect(result).toHaveLength(0);
    });
  });

  describe('sortEmailsByDate', () => {
    const emails: WaitlistEmail[] = [
      createMockEmail({ id: 1, created_at: '2024-01-15T10:00:00Z' }),
      createMockEmail({ id: 2, created_at: '2024-01-16T10:00:00Z' }),
      createMockEmail({ id: 3, created_at: '2024-01-14T10:00:00Z' }),
    ];

    it('sorts by date descending by default (newest first)', () => {
      const result = sortEmailsByDate(emails);
      expect(result[0].id).toBe(2);
      expect(result[1].id).toBe(1);
      expect(result[2].id).toBe(3);
    });

    it('sorts by date ascending when specified', () => {
      const result = sortEmailsByDate(emails, true);
      expect(result[0].id).toBe(3);
      expect(result[1].id).toBe(1);
      expect(result[2].id).toBe(2);
    });

    it('does not mutate original array', () => {
      const original = [...emails];
      sortEmailsByDate(emails);
      expect(emails).toEqual(original);
    });
  });

  describe('removeEmailFromList', () => {
    const emails: WaitlistEmail[] = [
      createMockEmail({ id: 1 }),
      createMockEmail({ id: 2 }),
      createMockEmail({ id: 3 }),
    ];

    it('removes email by id', () => {
      const result = removeEmailFromList(emails, 2);
      expect(result).toHaveLength(2);
      expect(result.find((e) => e.id === 2)).toBeUndefined();
    });

    it('returns same array if id not found', () => {
      const result = removeEmailFromList(emails, 999);
      expect(result).toHaveLength(3);
    });

    it('does not mutate original array', () => {
      const original = [...emails];
      removeEmailFromList(emails, 1);
      expect(emails).toEqual(original);
    });

    it('handles empty array', () => {
      const result = removeEmailFromList([], 1);
      expect(result).toHaveLength(0);
    });
  });

  describe('getUniqueSources', () => {
    it('returns unique sorted sources', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'newsletter' }),
        createMockEmail({ id: 2, source: 'coming_soon' }),
        createMockEmail({ id: 3, source: 'newsletter' }),
        createMockEmail({ id: 4, source: 'referral' }),
      ];

      const result = getUniqueSources(emails);
      expect(result).toEqual(['coming_soon', 'newsletter', 'referral']);
    });

    it('returns empty array for empty list', () => {
      const result = getUniqueSources([]);
      expect(result).toEqual([]);
    });

    it('returns single source when all same', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'coming_soon' }),
        createMockEmail({ id: 2, source: 'coming_soon' }),
      ];

      const result = getUniqueSources(emails);
      expect(result).toEqual(['coming_soon']);
    });
  });
});
