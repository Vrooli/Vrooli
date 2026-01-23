import { describe, it, expect } from 'vitest';
import {
  calculateTotalUsage,
  calculateUsagePercentage,
  sortByUsageDesc,
  getTopUsers,
  getSortedAppTotals,
  getLimitedRecords,
} from './usage.service';

describe('usage.service', () => {
  describe('calculateTotalUsage', () => {
    it('sums all user totals', () => {
      const userTotals = {
        user1: 100,
        user2: 200,
        user3: 300,
      };
      expect(calculateTotalUsage(userTotals)).toBe(600);
    });

    it('returns 0 for empty object', () => {
      expect(calculateTotalUsage({})).toBe(0);
    });

    it('returns 0 for undefined', () => {
      expect(calculateTotalUsage(undefined)).toBe(0);
    });
  });

  describe('calculateUsagePercentage', () => {
    it('calculates correct percentage', () => {
      expect(calculateUsagePercentage(50, 200)).toBe(25);
      expect(calculateUsagePercentage(100, 100)).toBe(100);
    });

    it('returns 0 when total is 0', () => {
      expect(calculateUsagePercentage(50, 0)).toBe(0);
    });
  });

  describe('sortByUsageDesc', () => {
    it('sorts entries by value descending', () => {
      const entries: [string, number][] = [
        ['a', 10],
        ['b', 30],
        ['c', 20],
      ];
      const result = sortByUsageDesc(entries);
      expect(result[0][1]).toBe(30);
      expect(result[1][1]).toBe(20);
      expect(result[2][1]).toBe(10);
    });

    it('does not mutate original array', () => {
      const entries: [string, number][] = [
        ['a', 10],
        ['b', 30],
      ];
      const original = [...entries];
      sortByUsageDesc(entries);
      expect(entries).toEqual(original);
    });
  });

  describe('getTopUsers', () => {
    it('returns top N users sorted by usage', () => {
      const userTotals = {
        user1: 100,
        user2: 300,
        user3: 200,
      };
      const result = getTopUsers(userTotals, 2);
      expect(result).toHaveLength(2);
      expect(result[0].user).toBe('user2');
      expect(result[0].usage).toBe(300);
      expect(result[1].user).toBe('user3');
    });

    it('returns all users if less than limit', () => {
      const userTotals = {
        user1: 100,
        user2: 200,
      };
      const result = getTopUsers(userTotals, 10);
      expect(result).toHaveLength(2);
    });

    it('uses default limit of 10', () => {
      const userTotals: Record<string, number> = {};
      for (let i = 0; i < 15; i++) {
        userTotals[`user${i}`] = i * 10;
      }
      const result = getTopUsers(userTotals);
      expect(result).toHaveLength(10);
    });
  });

  describe('getSortedAppTotals', () => {
    it('returns apps sorted with percentages', () => {
      const appTotals = {
        app1: 200,
        app2: 800,
      };
      const result = getSortedAppTotals(appTotals);
      expect(result).toHaveLength(2);
      expect(result[0].app).toBe('app2');
      expect(result[0].usage).toBe(800);
      expect(result[0].percentage).toBe(80);
      expect(result[1].app).toBe('app1');
      expect(result[1].percentage).toBe(20);
    });

    it('handles empty totals', () => {
      const result = getSortedAppTotals({});
      expect(result).toHaveLength(0);
    });
  });

  describe('getLimitedRecords', () => {
    it('returns limited records', () => {
      const records = Array.from({ length: 50 }, (_, i) => ({
        id: String(i),
        user_identity: `user${i}`,
        billing_period: '2024-01',
        limit_key: 'ai_credits',
        usage_amount: i * 10,
        app_bundle_key: 'test',
        last_operation_at: '2024-01-01',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }));
      const result = getLimitedRecords(records, 20);
      expect(result).toHaveLength(20);
    });

    it('returns all if less than limit', () => {
      const records = Array.from({ length: 5 }, (_, i) => ({
        id: String(i),
        user_identity: `user${i}`,
        billing_period: '2024-01',
        limit_key: 'ai_credits',
        usage_amount: i * 10,
        app_bundle_key: 'test',
        last_operation_at: '2024-01-01',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }));
      const result = getLimitedRecords(records, 20);
      expect(result).toHaveLength(5);
    });

    it('uses default limit of 20', () => {
      const records = Array.from({ length: 50 }, (_, i) => ({
        id: String(i),
        user_identity: `user${i}`,
        billing_period: '2024-01',
        limit_key: 'ai_credits',
        usage_amount: i * 10,
        app_bundle_key: 'test',
        last_operation_at: '2024-01-01',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }));
      const result = getLimitedRecords(records);
      expect(result).toHaveLength(20);
    });
  });
});
