/**
 * Integration tests for app configuration and downloads workflow
 *
 * Tests multi-service interactions across:
 * - downloads.service
 * - branding.service
 * - customizationController
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  filterActiveVariants,
  filterArchivedVariants,
  getVariantWeight,
  calculateTotalWeight,
  determineWeightStatus,
  normalizeTrafficShare,
  findStaleVariants,
  filterVariantsByQuery,
  getAttentionCandidateSlugs,
} from '../../controllers/customizationController';
import type { Variant } from '../../../../shared/api';

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

describe('Downloads and App Configuration Workflow', () => {
  describe('Variant management workflow', () => {
    it('separates active and archived variants for display', () => {
      const variants = [
        createMockVariant({ id: 1, slug: 'a', status: 'active' }),
        createMockVariant({ id: 2, slug: 'b', status: 'archived' }),
        createMockVariant({ id: 3, slug: 'c', status: 'active' }),
        createMockVariant({ id: 4, slug: 'd', status: 'archived' }),
      ];

      const active = filterActiveVariants(variants);
      const archived = filterArchivedVariants(variants);

      expect(active).toHaveLength(2);
      expect(archived).toHaveLength(2);
      expect(active.map((v) => v.slug)).toEqual(['a', 'c']);
      expect(archived.map((v) => v.slug)).toEqual(['b', 'd']);
    });
  });

  describe('Traffic distribution workflow', () => {
    it('calculates total weight with draft overrides', () => {
      const variants = [
        createMockVariant({ slug: 'a', weight: 30 }),
        createMockVariant({ slug: 'b', weight: 40 }),
        createMockVariant({ slug: 'c', weight: 30 }),
      ];
      const drafts = { a: 50, b: 30 }; // Override a and b

      // Without drafts: 30 + 40 + 30 = 100
      // With drafts: 50 + 30 + 30 = 110

      const total = calculateTotalWeight(variants, drafts);
      expect(total).toBe(110);

      const status = determineWeightStatus(variants.length, total);
      expect(status).toBe('over');
    });

    it('normalizes traffic shares when over-allocated', () => {
      const variants = [
        createMockVariant({ slug: 'a', weight: 60 }),
        createMockVariant({ slug: 'b', weight: 60 }),
      ];
      const totalWeight = 120; // Over 100

      // Even though weight is 60, normalized should be 50%
      const shareA = normalizeTrafficShare(60, totalWeight, 2, 'weighted');
      expect(shareA).toBe(50);
    });

    it('distributes evenly when all weights are zero', () => {
      const variants = [
        createMockVariant({ slug: 'a', weight: 0 }),
        createMockVariant({ slug: 'b', weight: 0 }),
        createMockVariant({ slug: 'c', weight: 0 }),
        createMockVariant({ slug: 'd', weight: 0 }),
      ];

      // Each variant gets 25% in even mode
      const share = normalizeTrafficShare(0, 0, 4, 'even');
      expect(share).toBe(25);
    });

    it('achieves balanced 100% allocation', () => {
      const variants = [
        createMockVariant({ slug: 'a', weight: 25 }),
        createMockVariant({ slug: 'b', weight: 25 }),
        createMockVariant({ slug: 'c', weight: 50 }),
      ];

      const total = calculateTotalWeight(variants, {});
      expect(total).toBe(100);

      const status = determineWeightStatus(variants.length, total);
      expect(status).toBe('balanced');
    });
  });

  describe('Attention-requiring variants workflow', () => {
    it('identifies variants needing attention from multiple sources', () => {
      // Create stale variant (40+ days old)
      const staleDate = new Date();
      staleDate.setDate(staleDate.getDate() - 45);

      const variants = [
        createMockVariant({ slug: 'stale', updated_at: staleDate.toISOString() }),
        createMockVariant({ slug: 'never-updated', updated_at: undefined }),
        createMockVariant({ slug: 'current', updated_at: new Date().toISOString() }),
      ];

      const active = filterActiveVariants(variants);
      const staleVariants = findStaleVariants(active);

      expect(staleVariants).toHaveLength(1);
      expect(staleVariants[0].variant.slug).toBe('stale');
      expect(staleVariants[0].daysSinceUpdate).toBeGreaterThanOrEqual(45);

      // Collect attention candidates
      const neverUpdated = active.filter((v) => !v.updated_at);
      const attentionSlugs = getAttentionCandidateSlugs(staleVariants, neverUpdated, undefined);

      expect(attentionSlugs.has('stale')).toBe(true);
      expect(attentionSlugs.has('never-updated')).toBe(true);
      expect(attentionSlugs.has('current')).toBe(false);
    });
  });

  describe('Variant search and filter workflow', () => {
    it('filters by search query and attention flag combined', () => {
      const variants = [
        createMockVariant({ slug: 'hero-test', name: 'Hero Test' }),
        createMockVariant({ slug: 'hero-control', name: 'Hero Control' }),
        createMockVariant({ slug: 'pricing-test', name: 'Pricing Test' }),
      ];
      const attention = new Set(['hero-test', 'pricing-test']);

      // Search for "hero" with attention filter
      const results = filterVariantsByQuery(variants, 'hero', true, attention);

      // Should only match hero-test (matches query AND is attention)
      expect(results).toHaveLength(1);
      expect(results[0].slug).toBe('hero-test');
    });

    it('shows all attention variants when no query', () => {
      const variants = [
        createMockVariant({ slug: 'a', name: 'A' }),
        createMockVariant({ slug: 'b', name: 'B' }),
        createMockVariant({ slug: 'c', name: 'C' }),
      ];
      const attention = new Set(['a', 'c']);

      const results = filterVariantsByQuery(variants, '', true, attention);

      expect(results).toHaveLength(2);
      expect(results.map((v) => v.slug)).toEqual(['a', 'c']);
    });

    it('case-insensitive search works correctly', () => {
      const variants = [
        createMockVariant({ slug: 'hero-TEST', name: 'Hero TEST' }),
        createMockVariant({ slug: 'PRICING-test', name: 'PRICING Test' }),
      ];

      const results = filterVariantsByQuery(variants, 'hero', false, new Set());
      expect(results).toHaveLength(1);

      const results2 = filterVariantsByQuery(variants, 'PRICING', false, new Set());
      expect(results2).toHaveLength(1);
    });
  });

  describe('Complete variant lifecycle', () => {
    it('simulates creating and managing multiple variants', () => {
      // Start with no variants
      let variants: Variant[] = [];

      // Add first variant
      variants.push(createMockVariant({ id: 1, slug: 'control', weight: 50 }));
      expect(determineWeightStatus(1, 50)).toBe('under');

      // Add second variant to balance
      variants.push(createMockVariant({ id: 2, slug: 'test-a', weight: 50 }));
      expect(calculateTotalWeight(variants, {})).toBe(100);
      expect(determineWeightStatus(2, 100)).toBe('balanced');

      // Archive the test variant
      variants = variants.map((v) =>
        v.slug === 'test-a' ? { ...v, status: 'archived' as const } : v
      );

      const active = filterActiveVariants(variants);
      const archived = filterArchivedVariants(variants);

      expect(active).toHaveLength(1);
      expect(archived).toHaveLength(1);

      // Now under-allocated
      const activeTotal = calculateTotalWeight(active, {});
      expect(activeTotal).toBe(50);
      expect(determineWeightStatus(1, 50)).toBe('under');
    });
  });
});
