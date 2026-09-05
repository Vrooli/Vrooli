import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const FALLBACK_PATH = '../../../../.vrooli/fallback/fallback.json';

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
    expect(config.sections[0]?.order).toBe(1);
    expect(config.sections[0]?.enabled).toBe(true);
    expect(config.sections[1]?.order).toBe(10);
    expect(config.sections[1]?.enabled).toBe(false);
  });

  it('omits incomplete pricing while preserving fallback axes and a usable default header', async () => {
    vi.doMock(FALLBACK_PATH, () => ({
      default: {
        variant: baseVariant,
        axes: { audience: 'builders' },
        sections: undefined,
        pricing: { monthly: [] },
      },
    }));

    const module = await import('./fallbackLandingConfig');
    const config = module.getFallbackLandingConfig();

    expect(config.pricing).toBeUndefined();
    expect(config.downloads).toEqual([]);
    expect(config.variant.axes).toEqual({ audience: 'builders' });
    expect(config.header).toMatchObject({ branding: { label: 'Fallback Variant' }, behavior: { sticky: true } });
  });

  it('fills partial fallback plans and returns an isolated clone on every access', async () => {
    vi.doMock(FALLBACK_PATH, () => ({
      default: {
        variant: { ...baseVariant, axes: { embedded: 'variant' } },
        axes: { ignored: 'payload' },
        sections: [{ section_type: 'pricing', content: {}, order: Number.NaN }],
        pricing: {
          ...basePricing,
          updated_at: undefined,
          monthly: [{ plan_name: 'Free', plan_tier: 'free' }],
          yearly: null,
        },
        downloads: [],
      },
    }));

    const module = await import('./fallbackLandingConfig');
    const first = module.getFallbackLandingConfig();
    const plan = first.pricing?.monthly[0];
    expect(first.sections[0]?.order).toBe(1);
    expect(first.variant.axes).toEqual({ embedded: 'variant' });
    expect(plan).toMatchObject({ billing_interval: 'month', amount_cents: 0, currency: 'usd', stripe_price_id: '', display_enabled: true });
    expect(first.pricing?.updated_at).toMatch(/^\d{4}-\d{2}-\d{2}T/);

    first.variant.name = 'Mutated';
    expect(module.getFallbackLandingConfig().variant.name).toBe('Fallback Variant');
  });
});
