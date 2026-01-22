import { describe, it, expect } from 'vitest';
import type { TierLimit } from '../../../shared/api';
import {
  DEFAULT_TIER_VALUES,
  getEditKey,
  getTierLabel,
  getTierColor,
  isUnlimitedValue,
  parseEditedValue,
  buildTierLimitUpdate,
  getDisplayValue,
  collectCostBasedLimitKeys,
  findAICreditsLimit,
  calculateDoubledLimits,
  TIER_OPTIONS,
} from './tiers.service';

const createMockLimit = (overrides: Partial<TierLimit> = {}): TierLimit => ({
  id: '1',
  tier_id: 'solo',
  limit_type: 'cost_based',
  limit_key: 'ai_credits',
  limit_value: 5000000,
  cost_multiplier: 1000000,
  reset_period: 'monthly',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  display_dollars: 5,
  ...overrides,
});

describe('tiers.service', () => {
  describe('DEFAULT_TIER_VALUES', () => {
    it('has values for all tiers', () => {
      expect(DEFAULT_TIER_VALUES['free:ai_credits']).toBe('0');
      expect(DEFAULT_TIER_VALUES['solo:ai_credits']).toBe('5');
      expect(DEFAULT_TIER_VALUES['pro:ai_credits']).toBe('20');
      expect(DEFAULT_TIER_VALUES['studio:ai_credits']).toBe('100');
      expect(DEFAULT_TIER_VALUES['business:ai_credits']).toBe('unlimited');
    });
  });

  describe('getEditKey', () => {
    it('combines tier ID and limit key', () => {
      expect(getEditKey('solo', 'ai_credits')).toBe('solo:ai_credits');
    });
  });

  describe('getTierLabel', () => {
    it('returns label for known tiers', () => {
      expect(getTierLabel('free')).toBe('Free');
      expect(getTierLabel('solo')).toBe('Solo');
      expect(getTierLabel('pro')).toBe('Pro');
    });

    it('returns tier ID for unknown tiers', () => {
      expect(getTierLabel('unknown')).toBe('unknown');
    });
  });

  describe('getTierColor', () => {
    it('returns correct colors for tiers', () => {
      expect(getTierColor('free')).toBe('text-slate-400');
      expect(getTierColor('solo')).toBe('text-blue-400');
      expect(getTierColor('pro')).toBe('text-purple-400');
      expect(getTierColor('studio')).toBe('text-amber-400');
      expect(getTierColor('business')).toBe('text-emerald-400');
    });

    it('returns default color for unknown tier', () => {
      expect(getTierColor('unknown')).toBe('text-slate-400');
    });
  });

  describe('isUnlimitedValue', () => {
    it('returns true for negative values', () => {
      expect(isUnlimitedValue(-1)).toBe(true);
      expect(isUnlimitedValue(-100)).toBe(true);
    });

    it('returns false for zero and positive values', () => {
      expect(isUnlimitedValue(0)).toBe(false);
      expect(isUnlimitedValue(100)).toBe(false);
    });
  });

  describe('parseEditedValue', () => {
    it('returns isUnlimited for "unlimited"', () => {
      expect(parseEditedValue('unlimited')).toEqual({ isUnlimited: true });
    });

    it('returns isUnlimited for "-1"', () => {
      expect(parseEditedValue('-1')).toEqual({ isUnlimited: true });
    });

    it('returns displayDollars for valid numbers', () => {
      expect(parseEditedValue('50')).toEqual({ displayDollars: 50 });
      expect(parseEditedValue('99.99')).toEqual({ displayDollars: 99.99 });
    });

    it('returns null for invalid values', () => {
      expect(parseEditedValue('abc')).toBeNull();
      expect(parseEditedValue('-2')).toBeNull();
    });

    it('handles whitespace and case', () => {
      expect(parseEditedValue('  UNLIMITED  ')).toEqual({ isUnlimited: true });
      expect(parseEditedValue('  50  ')).toEqual({ displayDollars: 50 });
    });
  });

  describe('buildTierLimitUpdate', () => {
    it('builds unlimited update', () => {
      expect(buildTierLimitUpdate({ isUnlimited: true })).toEqual({ is_unlimited: true });
    });

    it('builds display dollars update', () => {
      expect(buildTierLimitUpdate({ displayDollars: 50 })).toEqual({ display_dollars: 50 });
    });
  });

  describe('getDisplayValue', () => {
    it('returns "unlimited" for negative limit value', () => {
      const limit = createMockLimit({ limit_value: -1 });
      expect(getDisplayValue(limit)).toBe('unlimited');
    });

    it('returns formatted display_dollars', () => {
      const limit = createMockLimit({ limit_value: 100, display_dollars: 5 });
      expect(getDisplayValue(limit)).toBe('5.00');
    });

    it('returns "0" for undefined display_dollars', () => {
      const limit = createMockLimit({ limit_value: 100, display_dollars: undefined });
      expect(getDisplayValue(limit)).toBe('0');
    });
  });

  describe('collectCostBasedLimitKeys', () => {
    it('collects cost-based limit keys', () => {
      const limits: Record<string, TierLimit[]> = {
        solo: [
          createMockLimit({ limit_key: 'ai_credits', limit_type: 'cost_based' }),
          createMockLimit({ limit_key: 'api_calls', limit_type: 'app_specific' }),
        ],
        pro: [createMockLimit({ limit_key: 'ai_credits', limit_type: 'cost_based' })],
      };

      const keys = collectCostBasedLimitKeys(limits);
      expect(keys.size).toBe(1);
      expect(keys.has('ai_credits')).toBe(true);
      expect(keys.has('api_calls')).toBe(false);
    });

    it('returns empty set for empty limits', () => {
      const keys = collectCostBasedLimitKeys({});
      expect(keys.size).toBe(0);
    });
  });

  describe('findAICreditsLimit', () => {
    it('finds ai_credits cost-based limit', () => {
      const tierLimits = [
        createMockLimit({ limit_key: 'other', limit_type: 'app_specific' }),
        createMockLimit({ limit_key: 'ai_credits', limit_type: 'cost_based' }),
      ];

      const result = findAICreditsLimit(tierLimits);
      expect(result?.limit_key).toBe('ai_credits');
      expect(result?.limit_type).toBe('cost_based');
    });

    it('returns undefined if not found', () => {
      const tierLimits = [
        createMockLimit({ limit_key: 'other', limit_type: 'app_specific' }),
      ];

      expect(findAICreditsLimit(tierLimits)).toBeUndefined();
    });

    it('returns undefined for undefined input', () => {
      expect(findAICreditsLimit(undefined)).toBeUndefined();
    });
  });

  describe('calculateDoubledLimits', () => {
    it('doubles all positive limits', () => {
      const limits: Record<string, TierLimit[]> = {
        solo: [createMockLimit({ tier_id: 'solo', display_dollars: 5, limit_value: 5000000 })],
        pro: [createMockLimit({ tier_id: 'pro', display_dollars: 20, limit_value: 20000000 })],
      };

      const result = calculateDoubledLimits(limits);
      expect(result['solo:ai_credits']).toBe('10');
      expect(result['pro:ai_credits']).toBe('40');
    });

    it('skips unlimited tiers', () => {
      const limits: Record<string, TierLimit[]> = {
        business: [createMockLimit({ tier_id: 'business', display_dollars: undefined, limit_value: -1 })],
      };

      const result = calculateDoubledLimits(limits);
      expect(result['business:ai_credits']).toBeUndefined();
    });

    it('returns empty object for empty limits', () => {
      const result = calculateDoubledLimits({});
      expect(Object.keys(result).length).toBe(0);
    });
  });

  describe('TIER_OPTIONS', () => {
    it('has options', () => {
      expect(TIER_OPTIONS.length).toBeGreaterThan(0);
    });
  });
});
