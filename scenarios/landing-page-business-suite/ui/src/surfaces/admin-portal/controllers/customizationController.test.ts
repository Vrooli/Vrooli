import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Variant, VariantStats } from '../../../shared/api';
import {
  SNAPSHOT_DAYS,
  filterActiveVariants,
  filterArchivedVariants,
  getVariantWeight,
  calculateTotalWeight,
  determineWeightStatus,
  determineTrafficShareMode,
  normalizeTrafficShare,
  findStaleVariants,
  findNeverUpdatedVariants,
  findUnderperformingVariant,
  getAttentionCandidateSlugs,
  buildAttentionReasonsMap,
  filterVariantsByQuery,
  buildStatsMap,
  getTrendType,
} from './customizationController';

const createMockVariant = (overrides: Partial<Variant> = {}): Variant => ({
  id: 1,
  slug: 'test-variant',
  name: 'Test Variant',
  status: 'active',
  weight: 50,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

const createMockStats = (overrides: Partial<VariantStats> = {}): VariantStats => ({
  variant_id: 1,
  variant_slug: 'test-variant',
  variant_name: 'Test Variant',
  views: 100,
  cta_clicks: 10,
  conversions: 5,
  downloads: 3,
  conversion_rate: 5,
  trend: 'up',
  ...overrides,
});

describe('customizationController', () => {
  describe('SNAPSHOT_DAYS', () => {
    it('is 7 days', () => {
      expect(SNAPSHOT_DAYS).toBe(7);
    });
  });

  describe('filterActiveVariants', () => {
    it('filters only active variants', () => {
      const variants = [
        createMockVariant({ slug: 'a', status: 'active' }),
        createMockVariant({ slug: 'b', status: 'archived' }),
        createMockVariant({ slug: 'c', status: 'active' }),
      ];
      const result = filterActiveVariants(variants);
      expect(result).toHaveLength(2);
      expect(result.every((v) => v.status === 'active')).toBe(true);
    });
  });

  describe('filterArchivedVariants', () => {
    it('filters only archived variants', () => {
      const variants = [
        createMockVariant({ slug: 'a', status: 'active' }),
        createMockVariant({ slug: 'b', status: 'archived' }),
      ];
      const result = filterArchivedVariants(variants);
      expect(result).toHaveLength(1);
      expect(result[0].slug).toBe('b');
    });
  });

  describe('getVariantWeight', () => {
    it('returns draft weight if present', () => {
      const variant = createMockVariant({ slug: 'a', weight: 50 });
      const drafts = { a: 75 };
      expect(getVariantWeight(variant, drafts)).toBe(75);
    });

    it('returns variant weight if no draft', () => {
      const variant = createMockVariant({ slug: 'a', weight: 50 });
      expect(getVariantWeight(variant, {})).toBe(50);
    });

    it('returns 0 if no weight', () => {
      const variant = createMockVariant({ slug: 'a', weight: undefined });
      expect(getVariantWeight(variant, {})).toBe(0);
    });
  });

  describe('calculateTotalWeight', () => {
    it('sums weights from drafts and variants', () => {
      const variants = [
        createMockVariant({ slug: 'a', weight: 30 }),
        createMockVariant({ slug: 'b', weight: 40 }),
      ];
      const drafts = { a: 25 };
      expect(calculateTotalWeight(variants, drafts)).toBe(65); // 25 + 40
    });
  });

  describe('determineWeightStatus', () => {
    it('returns empty when no variants', () => {
      expect(determineWeightStatus(0, 0)).toBe('empty');
    });

    it('returns balanced when total is 100', () => {
      expect(determineWeightStatus(2, 100)).toBe('balanced');
    });

    it('returns over when total exceeds 100', () => {
      expect(determineWeightStatus(2, 150)).toBe('over');
    });

    it('returns under when total below 100', () => {
      expect(determineWeightStatus(2, 80)).toBe('under');
    });
  });

  describe('determineTrafficShareMode', () => {
    it('returns even when total is 0', () => {
      expect(determineTrafficShareMode(0)).toBe('even');
    });

    it('returns weighted when total > 0', () => {
      expect(determineTrafficShareMode(100)).toBe('weighted');
    });
  });

  describe('normalizeTrafficShare', () => {
    it('returns 0 when no variants', () => {
      expect(normalizeTrafficShare(50, 100, 0, 'weighted')).toBe(0);
    });

    it('returns even split in even mode', () => {
      expect(normalizeTrafficShare(0, 0, 4, 'even')).toBe(25);
    });

    it('returns proportional share in weighted mode', () => {
      expect(normalizeTrafficShare(25, 100, 4, 'weighted')).toBe(25);
      expect(normalizeTrafficShare(50, 100, 4, 'weighted')).toBe(50);
    });

    it('returns 0 when total is 0 in weighted mode', () => {
      expect(normalizeTrafficShare(0, 0, 4, 'weighted')).toBe(0);
    });
  });

  describe('findStaleVariants', () => {
    it('finds variants not updated in threshold days', () => {
      const oldDate = new Date();
      oldDate.setDate(oldDate.getDate() - 40);

      const variants = [
        createMockVariant({ slug: 'old', updated_at: oldDate.toISOString() }),
        createMockVariant({ slug: 'new', updated_at: new Date().toISOString() }),
      ];

      const result = findStaleVariants(variants);
      expect(result).toHaveLength(1);
      expect(result[0].variant.slug).toBe('old');
    });

    it('limits results', () => {
      const oldDate = new Date();
      oldDate.setDate(oldDate.getDate() - 40);

      const variants = Array.from({ length: 10 }, (_, i) =>
        createMockVariant({ slug: `variant-${i}`, updated_at: oldDate.toISOString() })
      );

      const result = findStaleVariants(variants, 3);
      expect(result).toHaveLength(3);
    });

    it('handles missing updated_at', () => {
      const variants = [createMockVariant({ slug: 'a', updated_at: undefined })];
      const result = findStaleVariants(variants);
      expect(result).toHaveLength(0);
    });
  });

  describe('findNeverUpdatedVariants', () => {
    it('finds variants without updated_at', () => {
      const variants = [
        createMockVariant({ slug: 'a', updated_at: undefined }),
        createMockVariant({ slug: 'b', updated_at: '2024-01-01' }),
        createMockVariant({ slug: 'c', updated_at: undefined }),
      ];
      const result = findNeverUpdatedVariants(variants);
      expect(result).toHaveLength(2);
    });
  });

  describe('findUnderperformingVariant', () => {
    it('finds variant with lowest conversion rate', () => {
      const variants = [
        createMockVariant({ slug: 'a' }),
        createMockVariant({ slug: 'b' }),
      ];
      const stats = [
        createMockStats({ variant_slug: 'a', conversion_rate: 10 }),
        createMockStats({ variant_slug: 'b', conversion_rate: 5 }),
      ];

      const result = findUnderperformingVariant(stats, variants);
      expect(result?.stats.variant_slug).toBe('b');
    });

    it('returns null for empty stats', () => {
      const variants = [createMockVariant({ slug: 'a' })];
      expect(findUnderperformingVariant([], variants)).toBeNull();
      expect(findUnderperformingVariant(undefined, variants)).toBeNull();
    });

    it('returns null for no active variants', () => {
      const stats = [createMockStats({ variant_slug: 'a' })];
      expect(findUnderperformingVariant(stats, [])).toBeNull();
    });

    it('only considers active variant stats', () => {
      const variants = [createMockVariant({ slug: 'a' })];
      const stats = [
        createMockStats({ variant_slug: 'a', conversion_rate: 10 }),
        createMockStats({ variant_slug: 'archived', conversion_rate: 1 }),
      ];

      const result = findUnderperformingVariant(stats, variants);
      expect(result?.stats.variant_slug).toBe('a');
    });
  });

  describe('getAttentionCandidateSlugs', () => {
    it('combines all attention sources', () => {
      const stale = [{ variant: createMockVariant({ slug: 'stale' }), daysSinceUpdate: 40 }];
      const neverUpdated = [createMockVariant({ slug: 'new' })];
      const underperforming = 'low';

      const result = getAttentionCandidateSlugs(stale, neverUpdated, underperforming);
      expect(result.size).toBe(3);
      expect(result.has('stale')).toBe(true);
      expect(result.has('new')).toBe(true);
      expect(result.has('low')).toBe(true);
    });

    it('handles undefined underperforming', () => {
      const result = getAttentionCandidateSlugs([], [], undefined);
      expect(result.size).toBe(0);
    });
  });

  describe('buildAttentionReasonsMap', () => {
    it('builds reasons map', () => {
      const stale = [{ variant: createMockVariant({ slug: 'a' }), daysSinceUpdate: 40 }];
      const neverUpdated = [createMockVariant({ slug: 'b' })];
      const underperforming = 'a';

      const result = buildAttentionReasonsMap(stale, neverUpdated, underperforming);
      expect(result.get('a')).toEqual(['Stale · 40d', 'Lowest conversion']);
      expect(result.get('b')).toEqual(['Never customized']);
    });
  });

  describe('filterVariantsByQuery', () => {
    const variants = [
      createMockVariant({ slug: 'hero-test', name: 'Hero Test' }),
      createMockVariant({ slug: 'pricing-ab', name: 'Pricing AB' }),
      createMockVariant({ slug: 'cta-variant', name: 'CTA Variant' }),
    ];

    it('filters by name', () => {
      const result = filterVariantsByQuery(variants, 'hero', false, new Set());
      expect(result).toHaveLength(1);
      expect(result[0].slug).toBe('hero-test');
    });

    it('filters by slug', () => {
      const result = filterVariantsByQuery(variants, 'cta', false, new Set());
      expect(result).toHaveLength(1);
    });

    it('is case insensitive', () => {
      const result = filterVariantsByQuery(variants, 'PRICING', false, new Set());
      expect(result).toHaveLength(1);
    });

    it('returns all for empty query', () => {
      const result = filterVariantsByQuery(variants, '', false, new Set());
      expect(result).toHaveLength(3);
    });

    it('filters by attention only', () => {
      const attention = new Set(['hero-test']);
      const result = filterVariantsByQuery(variants, '', true, attention);
      expect(result).toHaveLength(1);
      expect(result[0].slug).toBe('hero-test');
    });

    it('combines query and attention filters', () => {
      const attention = new Set(['hero-test', 'pricing-ab']);
      const result = filterVariantsByQuery(variants, 'hero', true, attention);
      expect(result).toHaveLength(1);
    });
  });

  describe('buildStatsMap', () => {
    it('builds map from stats array', () => {
      const stats = [
        createMockStats({ variant_slug: 'a' }),
        createMockStats({ variant_slug: 'b' }),
      ];
      const result = buildStatsMap(stats);
      expect(result.size).toBe(2);
      expect(result.get('a')).toBeDefined();
    });

    it('handles undefined', () => {
      const result = buildStatsMap(undefined);
      expect(result.size).toBe(0);
    });
  });

  describe('getTrendType', () => {
    it('returns correct trend type', () => {
      expect(getTrendType('up')).toBe('up');
      expect(getTrendType('down')).toBe('down');
      expect(getTrendType(undefined)).toBe('neutral');
    });
  });
});
