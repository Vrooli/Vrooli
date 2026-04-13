import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const FALLBACK_PATH = '../../../../.vrooli/variants/fallback.json';

const basePricing = {
  bundle: {
    bundle_key: 'test_bundle',
    name: 'Test Bundle',
    stripe_product_id: 'price_test',
    credits_per_usd: 1000000,
    display_credits_multiplier: 0.001,
    display_credits_label: 'credits',
  },
  monthly: [],
  yearly: [],
  updated_at: '2025-01-01T00:00:00Z',
};

const baseVariant = {
  slug: 'fallback',
  name: 'Fallback Variant',
};

describe('getFallbackLandingConfig', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    vi.doUnmock(FALLBACK_PATH);
  });

  it('handles null sections by returning an empty array', async () => {
    vi.doMock(FALLBACK_PATH, () => ({
      default: {
        variant: baseVariant,
        sections: null,
        pricing: basePricing,
        downloads: null,
      },
    }));

    const module = await import('./fallbackLandingConfig');
    const config = module.getFallbackLandingConfig();

    expect(config.sections).toEqual([]);
    expect(config.variant.slug).toBe('fallback');
  });

  it('normalizes section order and enabled flags', async () => {
    vi.doMock(FALLBACK_PATH, () => ({
      default: {
        variant: baseVariant,
        sections: [
          { section_type: 'hero', content: {}, enabled: undefined },
          { section_type: 'cta', content: {}, order: 10, enabled: false },
        ],
        pricing: basePricing,
        downloads: [],
      },
    }));

    const module = await import('./fallbackLandingConfig');
    const config = module.getFallbackLandingConfig();

    expect(config.sections).toHaveLength(2);
    expect(config.sections[0].order).toBe(1);
    expect(config.sections[0].enabled).toBe(true);
    expect(config.sections[1].order).toBe(10);
    expect(config.sections[1].enabled).toBe(false);
  });
});
