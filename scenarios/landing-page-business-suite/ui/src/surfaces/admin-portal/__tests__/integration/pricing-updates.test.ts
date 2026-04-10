/**
 * Integration tests for pricing/tier updates workflow
 *
 * Tests multi-service interactions across:
 * - tiers.service
 * - appLimits.service
 * - pricing.service
 */

import { describe, it, expect } from 'vitest';
import {
  parseEditedValue,
  buildTierLimitUpdate,
  getDisplayValue,
  getEditKey,
  TIER_OPTIONS,
} from '../../../../shared/lib/tierUtils';
import {
  calculateDoubledLimits,
  findAICreditsLimit,
  DEFAULT_TIER_VALUES,
} from '../../services/tiers.service';
import {
  collectLimitKeys,
  buildCreateLimitPayload,
  type NewLimitFormState,
} from '../../services/appLimits.service';
import type { TierLimit } from '../../../../shared/api';

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

describe('Pricing Updates Integration', () => {
  describe('Tier limit editing workflow', () => {
    it('parses user input and builds valid update payload', () => {
      // User enters "50" in the limit input
      const userInput = '50';

      // Parse the input
      const parsed = parseEditedValue(userInput);
      expect(parsed).toEqual({ displayDollars: 50 });

      // Build the API update payload
      if (parsed && !('isUnlimited' in parsed)) {
        const update = buildTierLimitUpdate(parsed);
        expect(update).toEqual({ display_dollars: 50 });
      }
    });

    it('handles unlimited value input', () => {
      // User enters "unlimited"
      const userInput = 'unlimited';

      const parsed = parseEditedValue(userInput);
      expect(parsed).toEqual({ isUnlimited: true });

      if (parsed && 'isUnlimited' in parsed) {
        const update = buildTierLimitUpdate(parsed);
        expect(update).toEqual({ is_unlimited: true });
      }
    });

    it('handles -1 as unlimited', () => {
      const userInput = '-1';

      const parsed = parseEditedValue(userInput);
      expect(parsed).toEqual({ isUnlimited: true });
    });

    it('rejects invalid input', () => {
      expect(parseEditedValue('abc')).toBeNull();
      expect(parseEditedValue('-5')).toBeNull();
      expect(parseEditedValue('')).toBeNull();
    });
  });

  describe('Display value round-trip', () => {
    it('displays and parses limit values consistently', () => {
      // Create a limit with display_dollars = 25
      const limit = createMockLimit({ display_dollars: 25, limit_value: 25000000 });

      // Get display value
      const displayValue = getDisplayValue(limit);
      expect(displayValue).toBe('25.00');

      // Parse the display value back
      const parsed = parseEditedValue(displayValue);
      expect(parsed).toEqual({ displayDollars: 25 });
    });

    it('handles unlimited display and parse round-trip', () => {
      const limit = createMockLimit({ limit_value: -1 });

      const displayValue = getDisplayValue(limit);
      expect(displayValue).toBe('unlimited');

      const parsed = parseEditedValue(displayValue);
      expect(parsed).toEqual({ isUnlimited: true });
    });
  });

  describe('Multi-tier operations', () => {
    it('calculates doubled limits across all tiers', () => {
      const limits: Record<string, TierLimit[]> = {
        solo: [createMockLimit({ tier_id: 'solo', display_dollars: 5, limit_value: 5000000 })],
        pro: [createMockLimit({ tier_id: 'pro', display_dollars: 20, limit_value: 20000000 })],
        studio: [createMockLimit({ tier_id: 'studio', display_dollars: 100, limit_value: 100000000 })],
        business: [createMockLimit({ tier_id: 'business', limit_value: -1, display_dollars: undefined })],
      };

      const doubled = calculateDoubledLimits(limits);

      expect(doubled['solo:ai_credits']).toBe('10');
      expect(doubled['pro:ai_credits']).toBe('40');
      expect(doubled['studio:ai_credits']).toBe('200');
      // Unlimited tiers should not be doubled
      expect(doubled['business:ai_credits']).toBeUndefined();
    });

    it('applies default tier values correctly', () => {
      // Verify default values match expected tier progression
      expect(DEFAULT_TIER_VALUES['free:ai_credits']).toBe('0');
      expect(DEFAULT_TIER_VALUES['solo:ai_credits']).toBe('5');
      expect(DEFAULT_TIER_VALUES['pro:ai_credits']).toBe('20');
      expect(DEFAULT_TIER_VALUES['studio:ai_credits']).toBe('100');
      expect(DEFAULT_TIER_VALUES['business:ai_credits']).toBe('unlimited');
    });
  });

  describe('App-specific limits workflow', () => {
    it('creates new limit with correct internal value conversion', () => {
      const form: NewLimitFormState = {
        tier_id: 'solo',
        limit_key: 'workflow_exports',
        display_dollars: '50',
      };

      const payload = buildCreateLimitPayload(form, 'browser-automation-studio');

      expect(payload.tier_id).toBe('solo');
      expect(payload.limit_key).toBe('workflow_exports');
      expect(payload.app_bundle_key).toBe('browser-automation-studio');
      // 50 dollars * 100 cents * 1000000 cost_multiplier
      expect(payload.limit_value).toBe(5000000000);
    });

    it('collects unique limit keys across tiers', () => {
      const limits: Record<string, TierLimit[]> = {
        solo: [
          createMockLimit({ limit_key: 'ai_credits' }),
          createMockLimit({ limit_key: 'workflow_exports' }),
        ],
        pro: [
          createMockLimit({ limit_key: 'ai_credits' }),
          createMockLimit({ limit_key: 'api_calls' }),
        ],
      };

      const keys = collectLimitKeys(limits);

      expect(keys.size).toBe(3);
      expect(keys.has('ai_credits')).toBe(true);
      expect(keys.has('workflow_exports')).toBe(true);
      expect(keys.has('api_calls')).toBe(true);
    });

    it('generates correct edit keys for tier/limit combinations', () => {
      expect(getEditKey('solo', 'ai_credits')).toBe('solo:ai_credits');
      expect(getEditKey('pro', 'workflow_exports')).toBe('pro:workflow_exports');
    });
  });

  describe('Tier options consistency', () => {
    it('has all expected tiers', () => {
      const tierValues = TIER_OPTIONS.map((t) => t.value);

      expect(tierValues).toContain('free');
      expect(tierValues).toContain('solo');
      expect(tierValues).toContain('pro');
      expect(tierValues).toContain('studio');
      expect(tierValues).toContain('business');
    });

    it('tiers have labels and values', () => {
      TIER_OPTIONS.forEach((tier) => {
        expect(tier.value).toBeTruthy();
        expect(tier.label).toBeTruthy();
      });
    });
  });

  describe('Finding AI credits limit', () => {
    it('finds cost-based ai_credits limit', () => {
      const limits = [
        createMockLimit({ limit_key: 'other', limit_type: 'app_specific' }),
        createMockLimit({ limit_key: 'ai_credits', limit_type: 'cost_based' }),
      ];

      const found = findAICreditsLimit(limits);
      expect(found?.limit_key).toBe('ai_credits');
      expect(found?.limit_type).toBe('cost_based');
    });

    it('ignores app-specific ai_credits', () => {
      const limits = [
        createMockLimit({ limit_key: 'ai_credits', limit_type: 'app_specific' }),
      ];

      const found = findAICreditsLimit(limits);
      expect(found).toBeUndefined();
    });
  });
});
