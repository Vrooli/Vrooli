import { describe, it, expect } from 'vitest';
import type { TierLimit } from '../api';
import {
  getEditKey,
  getTierLabel,
  getTierColor,
  isUnlimitedValue,
  parseEditedValue,
  buildTierLimitUpdate,
  getDisplayValue,
} from './tierUtils';

describe('tierUtils', () => {
  describe('getEditKey', () => {
    it('combines tier ID and limit key', () => {
      expect(getEditKey('pro', 'ai_credits')).toBe('pro:ai_credits');
    });

    it('handles empty strings', () => {
      expect(getEditKey('', '')).toBe(':');
      expect(getEditKey('pro', '')).toBe('pro:');
      expect(getEditKey('', 'ai_credits')).toBe(':ai_credits');
    });

    it('handles special characters', () => {
      expect(getEditKey('tier-1', 'limit_key')).toBe('tier-1:limit_key');
    });
  });

  describe('getTierLabel', () => {
    it('returns label for known tiers', () => {
      expect(getTierLabel('free')).toBe('Free');
      expect(getTierLabel('solo')).toBe('Solo');
      expect(getTierLabel('pro')).toBe('Pro');
      expect(getTierLabel('studio')).toBe('Studio');
      expect(getTierLabel('business')).toBe('Business');
    });

    it('returns tier ID for unknown tiers', () => {
      expect(getTierLabel('unknown')).toBe('unknown');
      expect(getTierLabel('')).toBe('');
      expect(getTierLabel('custom_tier')).toBe('custom_tier');
    });
  });

  describe('getTierColor', () => {
    it('returns correct colors for known tiers', () => {
      expect(getTierColor('free')).toBe('text-slate-400');
      expect(getTierColor('solo')).toBe('text-blue-400');
      expect(getTierColor('pro')).toBe('text-purple-400');
      expect(getTierColor('studio')).toBe('text-amber-400');
      expect(getTierColor('business')).toBe('text-emerald-400');
    });

    it('returns default color for unknown tiers', () => {
      expect(getTierColor('unknown')).toBe('text-slate-400');
      expect(getTierColor('')).toBe('text-slate-400');
    });
  });

  describe('isUnlimitedValue', () => {
    it('returns true for negative values', () => {
      expect(isUnlimitedValue(-1)).toBe(true);
      expect(isUnlimitedValue(-100)).toBe(true);
      expect(isUnlimitedValue(-0.5)).toBe(true);
    });

    it('returns false for zero and positive values', () => {
      expect(isUnlimitedValue(0)).toBe(false);
      expect(isUnlimitedValue(1)).toBe(false);
      expect(isUnlimitedValue(100)).toBe(false);
      expect(isUnlimitedValue(0.5)).toBe(false);
    });
  });

  describe('parseEditedValue', () => {
    it('parses "unlimited" as unlimited', () => {
      expect(parseEditedValue('unlimited')).toEqual({ isUnlimited: true });
      expect(parseEditedValue('UNLIMITED')).toEqual({ isUnlimited: true });
      expect(parseEditedValue('  unlimited  ')).toEqual({ isUnlimited: true });
    });

    it('parses "-1" as unlimited', () => {
      expect(parseEditedValue('-1')).toEqual({ isUnlimited: true });
      expect(parseEditedValue('  -1  ')).toEqual({ isUnlimited: true });
    });

    it('parses positive numbers as dollar values', () => {
      expect(parseEditedValue('5')).toEqual({ displayDollars: 5 });
      expect(parseEditedValue('5.00')).toEqual({ displayDollars: 5 });
      expect(parseEditedValue('5.50')).toEqual({ displayDollars: 5.5 });
      expect(parseEditedValue('0')).toEqual({ displayDollars: 0 });
      expect(parseEditedValue('0.99')).toEqual({ displayDollars: 0.99 });
    });

    it('handles floating point precision', () => {
      expect(parseEditedValue('10.99')).toEqual({ displayDollars: 10.99 });
      expect(parseEditedValue('100.50')).toEqual({ displayDollars: 100.5 });
    });

    it('returns null for invalid values', () => {
      expect(parseEditedValue('invalid')).toBeNull();
      expect(parseEditedValue('abc')).toBeNull();
      expect(parseEditedValue('')).toBeNull();
      expect(parseEditedValue('   ')).toBeNull();
    });

    it('returns null for negative non-unlimited values', () => {
      expect(parseEditedValue('-2')).toBeNull();
      expect(parseEditedValue('-5')).toBeNull();
      expect(parseEditedValue('-0.5')).toBeNull();
    });
  });

  describe('buildTierLimitUpdate', () => {
    it('builds unlimited update', () => {
      expect(buildTierLimitUpdate({ isUnlimited: true })).toEqual({
        is_unlimited: true,
      });
    });

    it('builds dollar amount update', () => {
      expect(buildTierLimitUpdate({ displayDollars: 5 })).toEqual({
        display_dollars: 5,
      });
      expect(buildTierLimitUpdate({ displayDollars: 0 })).toEqual({
        display_dollars: 0,
      });
      expect(buildTierLimitUpdate({ displayDollars: 10.99 })).toEqual({
        display_dollars: 10.99,
      });
    });
  });

  describe('getDisplayValue', () => {
    const createMockLimit = (overrides: Partial<TierLimit> = {}): TierLimit => ({
      id: '1',
      tier_id: 'pro',
      limit_type: 'cost_based',
      limit_key: 'ai_credits',
      limit_value: 500000000,
      cost_multiplier: 1000000,
      reset_period: 'monthly',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      display_dollars: 5,
      ...overrides,
    });

    it('returns "unlimited" for negative limit values', () => {
      const limit = createMockLimit({ limit_value: -1, display_dollars: undefined });
      expect(getDisplayValue(limit)).toBe('unlimited');
    });

    it('returns formatted dollar value for positive limits', () => {
      const limit = createMockLimit({ limit_value: 500000000, display_dollars: 5 });
      expect(getDisplayValue(limit)).toBe('5.00');
    });

    it('formats decimal dollar values correctly', () => {
      const limit = createMockLimit({ limit_value: 1099000000, display_dollars: 10.99 });
      expect(getDisplayValue(limit)).toBe('10.99');
    });

    it('returns "0" when display_dollars is undefined', () => {
      const limit = createMockLimit({ limit_value: 0, display_dollars: undefined });
      expect(getDisplayValue(limit)).toBe('0');
    });

    it('handles zero dollar value', () => {
      const limit = createMockLimit({ tier_id: 'free', limit_value: 0, display_dollars: 0 });
      expect(getDisplayValue(limit)).toBe('0.00');
    });
  });
});
