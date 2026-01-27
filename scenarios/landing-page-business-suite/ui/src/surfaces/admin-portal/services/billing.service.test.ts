import { describe, it, expect, vi, beforeEach } from 'vitest';
import type {
  StripeSettingsResponse,
  BundleCatalogEntry,
  BundleProduct,
  PlanOption,
  getStripeSettings,
  updateStripeSettings,
  getBundleCatalog,
  updateBundlePrice,
  verifyStripePrice,
} from '../../../shared/api';
import {
  loadStripeSettings,
  saveStripeSettings,
  loadBundleCatalog,
  savePriceForm,
  verifyPriceId,
  buildStripeStatusBadges,
  isStripeFullyConfigured,
  isStripePartiallyConfigured,
  hasStripeFormValues,
  DEFAULT_STRIPE_FORM,
  type StripeFormState,
  type PriceVerificationResult,
} from './billing.service';
import type { PriceFormState } from './pricing.service';

// Mock the API module
type GetStripeSettingsFn = typeof getStripeSettings;
type UpdateStripeSettingsFn = typeof updateStripeSettings;
type GetBundleCatalogFn = typeof getBundleCatalog;
type UpdateBundlePriceFn = typeof updateBundlePrice;
type VerifyStripePriceFn = typeof verifyStripePrice;
type UpdateBundlePriceResult = Awaited<ReturnType<UpdateBundlePriceFn>>;

const getStripeSettingsMock = vi.fn<Parameters<GetStripeSettingsFn>, ReturnType<GetStripeSettingsFn>>();
const updateStripeSettingsMock = vi.fn<Parameters<UpdateStripeSettingsFn>, ReturnType<UpdateStripeSettingsFn>>();
const getBundleCatalogMock = vi.fn<Parameters<GetBundleCatalogFn>, ReturnType<GetBundleCatalogFn>>();
const updateBundlePriceMock = vi.fn<Parameters<UpdateBundlePriceFn>, ReturnType<UpdateBundlePriceFn>>();
const verifyStripePriceMock = vi.fn<Parameters<VerifyStripePriceFn>, ReturnType<VerifyStripePriceFn>>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getStripeSettings: (...args: Parameters<GetStripeSettingsFn>) => getStripeSettingsMock(...args),
    updateStripeSettings: (...args: Parameters<UpdateStripeSettingsFn>) => updateStripeSettingsMock(...args),
    getBundleCatalog: (...args: Parameters<GetBundleCatalogFn>) => getBundleCatalogMock(...args),
    updateBundlePrice: (...args: Parameters<UpdateBundlePriceFn>) => updateBundlePriceMock(...args),
    verifyStripePrice: (...args: Parameters<VerifyStripePriceFn>) => verifyStripePriceMock(...args),
  };
});

const mockStripeSettings: StripeSettingsResponse = {
  publishable_key_preview: 'pk_test_xxx',
  publishable_key_set: true,
  secret_key_set: true,
  webhook_secret_set: false,
  dashboard_url: 'https://dashboard.stripe.com/test',
  source: 'database',
};

const mockBundle: BundleProduct = {
  id: 1,
  bundle_key: 'test_bundle',
  name: 'Test Bundle',
  stripe_product_id: 'prod_test',
  credits_per_usd: 1000000,
  display_credits_multiplier: 0.001,
  display_credits_label: 'credits',
};

const mockPlan: PlanOption = {
  plan_name: 'Test Plan',
  plan_tier: 'pro',
  billing_interval: 'month',
  amount_cents: 9900,
  currency: 'usd',
  intro_enabled: false,
  stripe_price_id: 'price_123',
  monthly_included_credits: 10000000,
  one_time_bonus_credits: 0,
  plan_rank: 1,
  display_enabled: true,
  display_weight: 50,
};

const mockUpdatedPlan: UpdateBundlePriceResult = {
  plan_name: 'Test Plan',
  plan_tier: 'pro',
  billing_interval: 'month',
  amount_cents: 9900,
  currency: 'usd',
  intro_enabled: false,
  stripe_price_id: 'price_123',
  monthly_included_credits: 10000000,
  one_time_bonus_credits: 0,
  plan_rank: 1,
  kind: 'subscription',
  display_enabled: true,
  display_weight: 50,
};

describe('billing.service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('loadStripeSettings', () => {
    it('calls getStripeSettings and returns the response', async () => {
      getStripeSettingsMock.mockResolvedValue(mockStripeSettings);

      const result = await loadStripeSettings();

      expect(getStripeSettingsMock).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockStripeSettings);
    });

    it('propagates errors from the API', async () => {
      const error = new Error('API failure');
      getStripeSettingsMock.mockRejectedValue(error);

      await expect(loadStripeSettings()).rejects.toThrow('API failure');
    });
  });

  describe('saveStripeSettings', () => {
    it('sends non-empty fields with snake_case keys', async () => {
      updateStripeSettingsMock.mockResolvedValue(mockStripeSettings);

      const form: StripeFormState = {
        publishableKey: 'pk_test_123',
        secretKey: 'sk_test_456',
        webhookSecret: '',
        dashboardUrl: 'https://dashboard.stripe.com',
      };

      await saveStripeSettings(form);

      expect(updateStripeSettingsMock).toHaveBeenCalledWith({
        publishable_key: 'pk_test_123',
        secret_key: 'sk_test_456',
        dashboard_url: 'https://dashboard.stripe.com',
      });
    });

    it('throws error when no fields have values', async () => {
      const form: StripeFormState = {
        publishableKey: '',
        secretKey: '',
        webhookSecret: '   ',
        dashboardUrl: '',
      };

      await expect(saveStripeSettings(form)).rejects.toThrow('Enter at least one field before saving.');
      expect(updateStripeSettingsMock).not.toHaveBeenCalled();
    });

    it('trims whitespace from values', async () => {
      updateStripeSettingsMock.mockResolvedValue(mockStripeSettings);

      const form: StripeFormState = {
        publishableKey: '  pk_test_123  ',
        secretKey: '',
        webhookSecret: '',
        dashboardUrl: '',
      };

      await saveStripeSettings(form);

      expect(updateStripeSettingsMock).toHaveBeenCalledWith({
        publishable_key: 'pk_test_123',
      });
    });
  });

  describe('hasStripeFormValues', () => {
    it('returns true when at least one field has a value', () => {
      const form: StripeFormState = {
        publishableKey: 'pk_test_123',
        secretKey: '',
        webhookSecret: '',
        dashboardUrl: '',
      };

      expect(hasStripeFormValues(form)).toBe(true);
    });

    it('returns false when all fields are empty', () => {
      const form: StripeFormState = {
        publishableKey: '',
        secretKey: '  ',
        webhookSecret: '',
        dashboardUrl: '   ',
      };

      expect(hasStripeFormValues(form)).toBe(false);
    });
  });

  describe('loadBundleCatalog', () => {
    it('calls getBundleCatalog and returns bundles', async () => {
      const mockBundles: BundleCatalogEntry[] = [{ bundle: mockBundle, prices: [mockPlan] }];
      getBundleCatalogMock.mockResolvedValue({ bundles: mockBundles });

      const result = await loadBundleCatalog();

      expect(getBundleCatalogMock).toHaveBeenCalledTimes(1);
      expect(result.bundles).toEqual(mockBundles);
    });
  });

  describe('savePriceForm', () => {
    it('calls updateBundlePrice with correct payload', async () => {
      updateBundlePriceMock.mockResolvedValue(mockUpdatedPlan);

      const formState: PriceFormState = {
        values: {
          stripePriceId: 'price_new',
          planName: 'Pro Plan',
          displayWeight: 75,
          displayEnabled: true,
          subtitle: 'Best for teams',
          badge: 'Popular',
          ctaLabel: 'Get Started',
          highlight: true,
          featuresText: 'Feature 1\nFeature 2',
        },
        original: {
          stripePriceId: 'price_123',
          planName: 'Test Plan',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        saving: false,
      };

      await savePriceForm('test_bundle', 'price_123', formState);

      expect(updateBundlePriceMock).toHaveBeenCalledWith('test_bundle', 'price_123', {
        stripe_price_id: 'price_new',
        plan_name: 'Pro Plan',
        display_weight: 75,
        display_enabled: true,
        subtitle: 'Best for teams',
        badge: 'Popular',
        cta_label: 'Get Started',
        highlight: true,
        features: ['Feature 1', 'Feature 2'],
      });
    });

    it('rejects empty stripe_price_id', async () => {
      const formState: PriceFormState = {
        values: {
          stripePriceId: '',
          planName: 'Plan',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        original: {
          stripePriceId: 'price_123',
          planName: 'Plan',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        saving: false,
      };

      await expect(savePriceForm('test_bundle', 'price_123', formState)).rejects.toThrow('Stripe price ID is required.');
      expect(updateBundlePriceMock).not.toHaveBeenCalled();
    });

    it('rejects empty plan names', async () => {
      const formState: PriceFormState = {
        values: {
          stripePriceId: 'price_123',
          planName: '',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        original: {
          stripePriceId: 'price_123',
          planName: '',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        saving: false,
      };

      await expect(savePriceForm('test_bundle', 'price_123', formState)).rejects.toThrow('Plan name is required.');
      expect(updateBundlePriceMock).not.toHaveBeenCalled();
    });
  });

  describe('verifyPriceId', () => {
    it('returns error for empty input', async () => {
      const result = await verifyPriceId('');

      expect(result.status).toBe('error');
      expect(result.message).toBe('Enter a Stripe price ID');
      expect(verifyStripePriceMock).not.toHaveBeenCalled();
    });

    it('returns error for whitespace-only input', async () => {
      const result = await verifyPriceId('   ');

      expect(result.status).toBe('error');
      expect(result.message).toBe('Enter a Stripe price ID');
    });

    it('returns ok with formatted info on successful verification', async () => {
      verifyStripePriceMock.mockResolvedValue({
        id: 'price_abc123',
        lookup_key: 'pro_monthly',
        interval: 'month',
        currency: 'usd',
        active: true,
      });

      const result = await verifyPriceId('price_abc123');

      expect(result.status).toBe('ok');
      expect(result.message).toContain('ID price_abc123');
      expect(result.message).toContain('lookup pro_monthly');
      expect(result.message).toContain('month');
      expect(result.message).toContain('USD');
    });

    it('indicates inactive status in message', async () => {
      verifyStripePriceMock.mockResolvedValue({
        id: 'price_inactive',
        active: false,
      });

      const result = await verifyPriceId('price_inactive');

      expect(result.status).toBe('ok');
      expect(result.message).toContain('inactive');
    });

    it('rejects non-price identifiers', async () => {
      const result = await verifyPriceId('some_key');

      expect(result.status).toBe('error');
      expect(result.message).toBe('Stripe price IDs must start with "price_".');
      expect(verifyStripePriceMock).not.toHaveBeenCalled();
    });

    it('returns error status on API failure', async () => {
      verifyStripePriceMock.mockRejectedValue(new Error('Price not found'));

      const result = await verifyPriceId('price_invalid');

      expect(result.status).toBe('error');
      expect(result.message).toBe('Price not found');
    });

    it('handles non-Error rejection', async () => {
      verifyStripePriceMock.mockRejectedValue('Unknown error');

      const result = await verifyPriceId('price_test');

      expect(result.status).toBe('error');
      expect(result.message).toBe('Unknown error');
    });
  });

  describe('buildStripeStatusBadges', () => {
    it('returns empty array when settings is null', () => {
      const badges = buildStripeStatusBadges(null);
      expect(badges).toEqual([]);
    });

    it('returns badges with correct ok states', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: true,
        secret_key_set: true,
        webhook_secret_set: false,
        source: 'database',
      };

      const badges = buildStripeStatusBadges(settings);

      expect(badges).toHaveLength(3);
      expect(badges[0]).toEqual({ label: 'Publishable Key', ok: true });
      expect(badges[1]).toEqual({ label: 'Restricted Key', ok: true });
      expect(badges[2]).toEqual({ label: 'Webhook Secret', ok: false });
    });

    it('uses publishable_key_set as source of truth (preview is only set when key is configured)', () => {
      // With the unified contract, publishable_key_preview is only populated
      // when publishable_key_set is true. The badge uses only *_set flags.
      const settings: StripeSettingsResponse = {
        publishable_key_preview: 'pk_test_xxx',
        publishable_key_set: true, // preview is only set when this is true
        secret_key_set: false,
        webhook_secret_set: false,
        source: 'env',
      };

      const badges = buildStripeStatusBadges(settings);

      expect(badges[0]?.ok).toBe(true);
    });

    it('marks all badges as not ok when nothing is set', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: false,
        secret_key_set: false,
        webhook_secret_set: false,
        source: 'env',
      };

      const badges = buildStripeStatusBadges(settings);

      expect(badges.every((badge) => !badge.ok)).toBe(true);
    });
  });

  describe('isStripeFullyConfigured', () => {
    it('returns false when settings is null', () => {
      expect(isStripeFullyConfigured(null)).toBe(false);
    });

    it('returns true when all three keys are set', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: true,
        secret_key_set: true,
        webhook_secret_set: true,
        source: 'database',
      };
      expect(isStripeFullyConfigured(settings)).toBe(true);
    });

    it('returns false when any key is missing', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: true,
        secret_key_set: true,
        webhook_secret_set: false,
        source: 'database',
      };
      expect(isStripeFullyConfigured(settings)).toBe(false);
    });

    it('returns false when no keys are set', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: false,
        secret_key_set: false,
        webhook_secret_set: false,
        source: 'env',
      };
      expect(isStripeFullyConfigured(settings)).toBe(false);
    });
  });

  describe('isStripePartiallyConfigured', () => {
    it('returns false when settings is null', () => {
      expect(isStripePartiallyConfigured(null)).toBe(false);
    });

    it('returns true when at least one key is set', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: true,
        secret_key_set: false,
        webhook_secret_set: false,
        source: 'env',
      };
      expect(isStripePartiallyConfigured(settings)).toBe(true);
    });

    it('returns true when all keys are set', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: true,
        secret_key_set: true,
        webhook_secret_set: true,
        source: 'database',
      };
      expect(isStripePartiallyConfigured(settings)).toBe(true);
    });

    it('returns false when no keys are set', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_set: false,
        secret_key_set: false,
        webhook_secret_set: false,
        source: 'env',
      };
      expect(isStripePartiallyConfigured(settings)).toBe(false);
    });
  });

  describe('DEFAULT_STRIPE_FORM', () => {
    it('has all empty string values', () => {
      expect(DEFAULT_STRIPE_FORM.publishableKey).toBe('');
      expect(DEFAULT_STRIPE_FORM.secretKey).toBe('');
      expect(DEFAULT_STRIPE_FORM.webhookSecret).toBe('');
      expect(DEFAULT_STRIPE_FORM.dashboardUrl).toBe('');
    });
  });
});
