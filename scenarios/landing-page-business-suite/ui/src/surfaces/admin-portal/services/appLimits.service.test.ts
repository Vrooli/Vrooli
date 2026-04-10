import { describe, it, expect } from 'vitest';
import type { TierLimit } from '../../../shared/api';
import {
  getEditKey,
  getTierLabel,
  getTierColor,
  parseEditedValue,
  buildTierLimitUpdate,
  getDisplayValue,
  isUnlimitedValue,
} from '../../../shared/lib/tierUtils';
import {
  APP_OPTIONS,
  DEFAULT_NEW_LIMIT,
  collectLimitKeys,
  validateNewLimitForm,
  buildCreateLimitPayload,
  getSelectedAppLabel,
  type NewLimitFormState,
} from './appLimits.service';

const createMockLimit = (overrides: Partial<TierLimit> = {}): TierLimit => ({
  id: '1',
  tier_id: 'solo',
  limit_type: 'app_specific',
  limit_key: 'workflow_exports',
  limit_value: 10000000,
  cost_multiplier: 1000000,
  app_bundle_key: 'browser-automation-studio',
  reset_period: 'monthly',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  display_dollars: 100,
  ...overrides,
});

describe('appLimits.service', () => {
  describe('APP_OPTIONS', () => {
    it('contains at least one app option', () => {
      expect(APP_OPTIONS.length).toBeGreaterThan(0);
    });

    it('has browser-automation-studio as first option', () => {
      expect(APP_OPTIONS[0].value).toBe('browser-automation-studio');
    });
  });

  describe('DEFAULT_NEW_LIMIT', () => {
    it('has default tier_id of solo', () => {
      expect(DEFAULT_NEW_LIMIT.tier_id).toBe('solo');
    });

    it('has empty limit_key and display_dollars', () => {
      expect(DEFAULT_NEW_LIMIT.limit_key).toBe('');
      expect(DEFAULT_NEW_LIMIT.display_dollars).toBe('');
    });
  });

  describe('getEditKey', () => {
    it('combines tier ID and limit key with colon', () => {
      expect(getEditKey('solo', 'workflow_exports')).toBe('solo:workflow_exports');
    });

    it('handles empty strings', () => {
      expect(getEditKey('', '')).toBe(':');
    });

    it('handles special characters', () => {
      expect(getEditKey('pro', 'api:calls')).toBe('pro:api:calls');
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
    });
  });

  describe('getTierColor', () => {
    it('returns correct color class for each tier', () => {
      expect(getTierColor('free')).toBe('text-slate-400');
      expect(getTierColor('solo')).toBe('text-blue-400');
      expect(getTierColor('pro')).toBe('text-purple-400');
      expect(getTierColor('studio')).toBe('text-amber-400');
      expect(getTierColor('business')).toBe('text-emerald-400');
    });

    it('returns default color for unknown tiers', () => {
      expect(getTierColor('unknown')).toBe('text-slate-400');
    });
  });

  describe('collectLimitKeys', () => {
    it('collects unique limit keys from all tiers', () => {
      const limits: Record<string, TierLimit[]> = {
        solo: [
          createMockLimit({ limit_key: 'workflow_exports' }),
          createMockLimit({ limit_key: 'api_calls' }),
        ],
        pro: [
          createMockLimit({ limit_key: 'workflow_exports' }),
          createMockLimit({ limit_key: 'storage' }),
        ],
      };

      const keys = collectLimitKeys(limits);

      expect(keys.size).toBe(3);
      expect(keys.has('workflow_exports')).toBe(true);
      expect(keys.has('api_calls')).toBe(true);
      expect(keys.has('storage')).toBe(true);
    });

    it('returns empty set for empty limits', () => {
      const keys = collectLimitKeys({});
      expect(keys.size).toBe(0);
    });

    it('handles tiers with empty arrays', () => {
      const limits: Record<string, TierLimit[]> = {
        solo: [],
        pro: [createMockLimit({ limit_key: 'test' })],
      };

      const keys = collectLimitKeys(limits);
      expect(keys.size).toBe(1);
    });
  });

  describe('parseEditedValue', () => {
    it('returns isUnlimited for "unlimited"', () => {
      const result = parseEditedValue('unlimited');
      expect(result).toEqual({ isUnlimited: true });
    });

    it('returns isUnlimited for "UNLIMITED" (case insensitive)', () => {
      const result = parseEditedValue('UNLIMITED');
      expect(result).toEqual({ isUnlimited: true });
    });

    it('returns isUnlimited for "-1"', () => {
      const result = parseEditedValue('-1');
      expect(result).toEqual({ isUnlimited: true });
    });

    it('returns displayDollars for valid numbers', () => {
      expect(parseEditedValue('100')).toEqual({ displayDollars: 100 });
      expect(parseEditedValue('99.99')).toEqual({ displayDollars: 99.99 });
      expect(parseEditedValue('0')).toEqual({ displayDollars: 0 });
    });

    it('returns null for invalid values', () => {
      expect(parseEditedValue('abc')).toBeNull();
      expect(parseEditedValue('-2')).toBeNull();
      expect(parseEditedValue('')).toBeNull();
    });

    it('handles whitespace', () => {
      expect(parseEditedValue('  100  ')).toEqual({ displayDollars: 100 });
      expect(parseEditedValue('  unlimited  ')).toEqual({ isUnlimited: true });
    });
  });

  describe('buildTierLimitUpdate', () => {
    it('builds unlimited update', () => {
      const result = buildTierLimitUpdate({ isUnlimited: true });
      expect(result).toEqual({ is_unlimited: true });
    });

    it('builds display dollars update', () => {
      const result = buildTierLimitUpdate({ displayDollars: 50 });
      expect(result).toEqual({ display_dollars: 50 });
    });
  });

  describe('validateNewLimitForm', () => {
    it('returns error for empty limit_key', () => {
      const form: NewLimitFormState = {
        tier_id: 'solo',
        limit_key: '',
        display_dollars: '100',
      };
      expect(validateNewLimitForm(form)).toBe('Please enter a limit key');
    });

    it('returns error for whitespace-only limit_key', () => {
      const form: NewLimitFormState = {
        tier_id: 'solo',
        limit_key: '   ',
        display_dollars: '100',
      };
      expect(validateNewLimitForm(form)).toBe('Please enter a limit key');
    });

    it('returns null for valid form', () => {
      const form: NewLimitFormState = {
        tier_id: 'solo',
        limit_key: 'workflow_exports',
        display_dollars: '100',
      };
      expect(validateNewLimitForm(form)).toBeNull();
    });
  });

  describe('buildCreateLimitPayload', () => {
    it('builds correct payload structure', () => {
      const form: NewLimitFormState = {
        tier_id: 'pro',
        limit_key: 'workflow_exports',
        display_dollars: '100',
      };

      const result = buildCreateLimitPayload(form, 'browser-automation-studio');

      expect(result.tier_id).toBe('pro');
      expect(result.limit_type).toBe('app_specific');
      expect(result.limit_key).toBe('workflow_exports');
      expect(result.limit_value).toBe(10000000000); // 100 * 100 * 1000000
      expect(result.cost_multiplier).toBe(1000000);
      expect(result.app_bundle_key).toBe('browser-automation-studio');
      expect(result.reset_period).toBe('monthly');
    });

    it('trims limit_key whitespace', () => {
      const form: NewLimitFormState = {
        tier_id: 'pro',
        limit_key: '  my_limit  ',
        display_dollars: '50',
      };

      const result = buildCreateLimitPayload(form, 'test-app');
      expect(result.limit_key).toBe('my_limit');
    });

    it('handles empty display_dollars as 0', () => {
      const form: NewLimitFormState = {
        tier_id: 'solo',
        limit_key: 'test',
        display_dollars: '',
      };

      const result = buildCreateLimitPayload(form, 'test-app');
      expect(result.limit_value).toBe(0);
    });

    it('handles invalid display_dollars as 0', () => {
      const form: NewLimitFormState = {
        tier_id: 'solo',
        limit_key: 'test',
        display_dollars: 'invalid',
      };

      const result = buildCreateLimitPayload(form, 'test-app');
      expect(result.limit_value).toBe(0);
    });
  });

  describe('getDisplayValue', () => {
    it('returns "unlimited" for negative limit_value', () => {
      const limit = createMockLimit({ limit_value: -1 });
      expect(getDisplayValue(limit)).toBe('unlimited');
    });

    it('returns formatted display_dollars', () => {
      const limit = createMockLimit({ limit_value: 100, display_dollars: 100 });
      expect(getDisplayValue(limit)).toBe('100.00');
    });

    it('returns "0" for undefined display_dollars', () => {
      const limit = createMockLimit({ limit_value: 100, display_dollars: undefined });
      expect(getDisplayValue(limit)).toBe('0');
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

  describe('getSelectedAppLabel', () => {
    it('returns label for known app', () => {
      expect(getSelectedAppLabel('browser-automation-studio')).toBe('Browser Automation Studio');
    });

    it('returns app value for unknown app', () => {
      expect(getSelectedAppLabel('unknown-app')).toBe('unknown-app');
    });
  });
});
