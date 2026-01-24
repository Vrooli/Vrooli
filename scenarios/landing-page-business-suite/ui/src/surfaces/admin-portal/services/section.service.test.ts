import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { LandingSection, ContentSection } from '../../../shared/api';
import {
  loadComparePreference,
  saveComparePreference,
  sortSectionsByOrder,
  findSectionByType,
  buildDefaultSectionContent,
  getVariantGuidance,
  COMPARE_STORAGE_KEY,
} from './section.service';

describe('section.service', () => {
  describe('loadComparePreference', () => {
    const originalWindow = global.window;

    afterEach(() => {
      global.window = originalWindow;
    });

    it('returns null when window is undefined (SSR)', () => {
      const windowBackup = global.window;
      // @ts-expect-error - simulating SSR
      delete global.window;

      const result = loadComparePreference('test-variant');

      expect(result).toBeNull();
      global.window = windowBackup;
    });

    it('returns null when localStorage is empty', () => {
      const mockStorage: Record<string, string> = {};
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation((key: string) => mockStorage[key] ?? null);

      const result = loadComparePreference('test-variant');

      expect(result).toBeNull();
    });

    it('returns saved preference for variant', () => {
      const stored = { 'test-variant': 'compare-variant' };
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => JSON.stringify(stored));

      const result = loadComparePreference('test-variant');

      expect(result).toBe('compare-variant');
    });

    it('returns null when variant not in storage', () => {
      const stored = { 'other-variant': 'compare-variant' };
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => JSON.stringify(stored));

      const result = loadComparePreference('test-variant');

      expect(result).toBeNull();
    });

    it('returns null on JSON parse error', () => {
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => 'invalid-json');

      const result = loadComparePreference('test-variant');

      expect(result).toBeNull();
    });
  });

  describe('saveComparePreference', () => {
    const originalWindow = global.window;
    let mockStorage: Record<string, string>;

    beforeEach(() => {
      mockStorage = {};
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation((key: string) => mockStorage[key] ?? null);
      vi.spyOn(Storage.prototype, 'setItem').mockImplementation((key: string, value: string) => {
        mockStorage[key] = value;
      });
    });

    afterEach(() => {
      vi.restoreAllMocks();
      global.window = originalWindow;
    });

    it('does nothing when window is undefined (SSR)', () => {
      const windowBackup = global.window;
      // @ts-expect-error - simulating SSR
      delete global.window;

      saveComparePreference('test-variant', 'compare-variant');

      expect(mockStorage[COMPARE_STORAGE_KEY]).toBeUndefined();
      global.window = windowBackup;
    });

    it('saves preference to localStorage', () => {
      saveComparePreference('test-variant', 'compare-variant');

      const storedStr = mockStorage[COMPARE_STORAGE_KEY];
      expect(storedStr).toBeDefined();
      const stored = JSON.parse(storedStr!);
      expect(stored['test-variant']).toBe('compare-variant');
    });

    it('merges with existing preferences', () => {
      mockStorage[COMPARE_STORAGE_KEY] = JSON.stringify({ 'existing-variant': 'existing-compare' });

      saveComparePreference('test-variant', 'compare-variant');

      const stored = JSON.parse(mockStorage[COMPARE_STORAGE_KEY]);
      expect(stored['existing-variant']).toBe('existing-compare');
      expect(stored['test-variant']).toBe('compare-variant');
    });

    it('removes preference when compareSlug is null', () => {
      mockStorage[COMPARE_STORAGE_KEY] = JSON.stringify({ 'test-variant': 'compare-variant' });

      saveComparePreference('test-variant', null);

      const stored = JSON.parse(mockStorage[COMPARE_STORAGE_KEY]);
      expect(stored['test-variant']).toBeUndefined();
    });

    it('handles localStorage errors silently', () => {
      vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
        throw new Error('QuotaExceededError');
      });

      expect(() => saveComparePreference('test-variant', 'compare-variant')).not.toThrow();
    });
  });

  describe('sortSectionsByOrder', () => {
    it('sorts sections by order ascending', () => {
      const sections: LandingSection[] = [
        { section_type: 'cta', content: {}, order: 3 },
        { section_type: 'hero', content: {}, order: 1 },
        { section_type: 'features', content: {}, order: 2 },
      ];

      const result = sortSectionsByOrder(sections);

      expect(result[0]?.section_type).toBe('hero');
      expect(result[1]?.section_type).toBe('features');
      expect(result[2]?.section_type).toBe('cta');
    });

    it('handles undefined order values (treats as 0)', () => {
      const sections: LandingSection[] = [
        { section_type: 'features', content: {}, order: 2 },
        { section_type: 'hero', content: {} } as LandingSection,
        { section_type: 'cta', content: {}, order: 1 },
      ];

      const result = sortSectionsByOrder(sections);

      expect(result[0]?.section_type).toBe('hero');
      expect(result[1]?.section_type).toBe('cta');
      expect(result[2]?.section_type).toBe('features');
    });

    it('does not mutate the original array', () => {
      const sections: LandingSection[] = [
        { section_type: 'cta', content: {}, order: 2 },
        { section_type: 'hero', content: {}, order: 1 },
      ];
      const originalFirst = sections[0];

      sortSectionsByOrder(sections);

      expect(sections[0]).toBe(originalFirst);
    });

    it('returns empty array for empty input', () => {
      const result = sortSectionsByOrder([]);
      expect(result).toEqual([]);
    });

    it('handles single section', () => {
      const sections: LandingSection[] = [{ section_type: 'hero', content: {}, order: 1 }];

      const result = sortSectionsByOrder(sections);

      expect(result).toHaveLength(1);
      expect(result[0]?.section_type).toBe('hero');
    });
  });

  describe('findSectionByType', () => {
    it('returns first matching section by type', () => {
      const sections: LandingSection[] = [
        { section_type: 'hero', content: { title: 'First Hero' }, order: 2 },
        { section_type: 'features', content: {}, order: 1 },
        { section_type: 'hero', content: { title: 'Second Hero' }, order: 3 },
      ];

      const result = findSectionByType(sections, 'hero');

      expect(result).not.toBeNull();
      expect(result?.content.title).toBe('First Hero');
    });

    it('returns section with lowest order when multiple match', () => {
      const sections: LandingSection[] = [
        { section_type: 'hero', content: { title: 'Order 2' }, order: 2 },
        { section_type: 'hero', content: { title: 'Order 1' }, order: 1 },
      ];

      const result = findSectionByType(sections, 'hero');

      expect(result?.content.title).toBe('Order 1');
    });

    it('returns null when no section matches type', () => {
      const sections: LandingSection[] = [
        { section_type: 'hero', content: {}, order: 1 },
        { section_type: 'features', content: {}, order: 2 },
      ];

      const result = findSectionByType(sections, 'pricing');

      expect(result).toBeNull();
    });

    it('returns null for empty sections array', () => {
      const result = findSectionByType([], 'hero');
      expect(result).toBeNull();
    });
  });

  describe('buildDefaultSectionContent', () => {
    it('returns object with default fields', () => {
      const result = buildDefaultSectionContent();

      expect(result).toHaveProperty('title', '');
      expect(result).toHaveProperty('subtitle', '');
      expect(result).toHaveProperty('cta_text', '');
      expect(result).toHaveProperty('cta_url', '');
    });

    it('returns a new object on each call', () => {
      const result1 = buildDefaultSectionContent();
      const result2 = buildDefaultSectionContent();

      expect(result1).not.toBe(result2);
      expect(result1).toEqual(result2);
    });
  });

  describe('getVariantGuidance', () => {
    it('returns guidance with key field', () => {
      const result = getVariantGuidance('some-variant');

      expect(result).toHaveProperty('key');
      expect(typeof result.key).toBe('string');
    });

    it('returns control key for unknown variant', () => {
      const result = getVariantGuidance('nonexistent-variant-xyz');

      expect(result.key).toBe('control');
    });

    it('handles undefined slug', () => {
      const result = getVariantGuidance(undefined);

      expect(result.key).toBe('control');
    });
  });

  describe('COMPARE_STORAGE_KEY', () => {
    it('has the expected constant value', () => {
      expect(COMPARE_STORAGE_KEY).toBe('landing-manager-section-editor-compare');
    });
  });
});
