import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { StripeSettingsResponse, BundleCatalogEntry, BundleProduct, PlanOption } from '../../../shared/api';
import {
  loadStripeSettings,
  saveStripeSettings,
  loadBundleCatalog,
  savePriceForm,
  verifyPriceId,
  buildStripeStatusBadges,
  hasStripeFormValues,
  DEFAULT_STRIPE_FORM,
  type StripeFormState,
  type PriceVerificationResult,
} from './billing.service';
import type { PriceFormState } from './pricing.service';

// Mock the API module
const getStripeSettingsMock = vi.fn();
const updateStripeSettingsMock = vi.fn();
const getBundleCatalogMock = vi.fn();
const updateBundlePriceMock = vi.fn();
const verifyStripePriceMock = vi.fn();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getStripeSettings: (...args: unknown[]) => getStripeSettingsMock(...args),
    updateStripeSettings: (...args: unknown[]) => updateStripeSettingsMock(...args),
    getBundleCatalog: (...args: unknown[]) => getBundleCatalogMock(...args),
    updateBundlePrice: (...args: unknown[]) => updateBundlePriceMock(...args),
    verifyStripePrice: (...args: unknown[]) => verifyStripePriceMock(...args),
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
      updateBundlePriceMock.mockResolvedValue({});

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

    it('sends empty string for stripe_price_id when cleared', async () => {
      updateBundlePriceMock.mockResolvedValue({});

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

      await savePriceForm('test_bundle', 'price_123', formState);

      expect(updateBundlePriceMock).toHaveBeenCalledWith('test_bundle', 'price_123', expect.objectContaining({
        stripe_price_id: '',
      }));
    });

    it('omits undefined values for empty optional fields', async () => {
      updateBundlePriceMock.mockResolvedValue({});

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

      await savePriceForm('test_bundle', 'price_123', formState);

      const call = updateBundlePriceMock.mock.calls[0][2];
      expect(call.plan_name).toBeUndefined();
      expect(call.subtitle).toBeUndefined();
      expect(call.badge).toBeUndefined();
      expect(call.cta_label).toBeUndefined();
    });
  });

  describe('verifyPriceId', () => {
    it('returns error for empty input', async () => {
      const result = await verifyPriceId('');

      expect(result.status).toBe('error');
      expect(result.message).toBe('Enter a Stripe price ID or lookup key');
      expect(verifyStripePriceMock).not.toHaveBeenCalled();
    });

    it('returns error for whitespace-only input', async () => {
      const result = await verifyPriceId('   ');

      expect(result.status).toBe('error');
      expect(result.message).toBe('Enter a Stripe price ID or lookup key');
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

    it('returns simple "Verified" when no details available', async () => {
      verifyStripePriceMock.mockResolvedValue({});

      const result = await verifyPriceId('some_key');

      expect(result.status).toBe('ok');
      expect(result.message).toBe('Verified');
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
      expect(result.message).toBe('Verification failed');
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

    it('marks publishable key as ok when preview is set', () => {
      const settings: StripeSettingsResponse = {
        publishable_key_preview: 'pk_test_xxx',
        publishable_key_set: false,
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

  describe('DEFAULT_STRIPE_FORM', () => {
    it('has all empty string values', () => {
      expect(DEFAULT_STRIPE_FORM.publishableKey).toBe('');
      expect(DEFAULT_STRIPE_FORM.secretKey).toBe('');
      expect(DEFAULT_STRIPE_FORM.webhookSecret).toBe('');
      expect(DEFAULT_STRIPE_FORM.dashboardUrl).toBe('');
    });
  });
});
