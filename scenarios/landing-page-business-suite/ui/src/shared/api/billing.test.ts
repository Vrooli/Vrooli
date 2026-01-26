import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  getStripeSettings,
  updateStripeSettings,
  getBundleCatalog,
  updateBundlePrice,
  verifyStripePrice,
  createCheckoutSession,
  createCreditsCheckoutSession,
  createBillingPortalSession,
  createBundlePrice,
  getStripeImportPreview,
  importStripePlans,
  type StripeSettingsResponse,
  type StripeSettingsUpdatePayload,
  type BundleCatalogResponse,
} from './billing';
import { ApiError } from './common';
import { createFetchMock, mockResponses, installFetchMock, getFetchCall } from '../test-utils/api-mocks';

vi.mock('@bufbuild/protobuf', () => ({
  fromJson: vi.fn((schema, data) => data),
}));

describe('billing API', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getStripeSettings', () => {
    it('returns flattened Stripe settings', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeyPreview: 'pk_test_xxx',
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: false,
          source: 2,
        },
        settings: {
          dashboardUrl: 'https://dashboard.stripe.com',
          updatedAt: '2024-01-01T00:00:00Z',
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getStripeSettings();

      expect(result.publishable_key_preview).toBe('pk_test_xxx');
      expect(result.publishable_key_set).toBe(true);
      expect(result.secret_key_set).toBe(true);
      expect(result.webhook_secret_set).toBe(false);
      expect(result.dashboard_url).toBe('https://dashboard.stripe.com');
    });

    it('handles timestamp with toJsonString method', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
        },
        settings: {
          updatedAt: {
            toJsonString: () => '2024-06-15T10:30:00Z',
          },
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getStripeSettings();

      expect(result.updated_at).toBe('2024-06-15T10:30:00Z');
    });

    it('handles timestamp with seconds/nanos', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
        },
        settings: {
          updatedAt: {
            seconds: 1718444100,
            nanos: 500000000,
          },
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getStripeSettings();

      expect(result.updated_at).toBeDefined();
      expect(new Date(result.updated_at!).getTime()).toBeGreaterThan(0);
    });

    it('handles Date instance timestamp', async () => {
      const testDate = new Date('2024-07-01T12:00:00Z');
      const protoResponse = {
        snapshot: {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
        },
        settings: {
          updatedAt: testDate,
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getStripeSettings();

      expect(result.updated_at).toBe(testDate.toISOString());
    });

    it('returns source from response', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
          source: 2,
        },
        settings: {},
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getStripeSettings();

      expect(result.source).toBeDefined();
      expect(typeof result.source).toBe('string');
    });

    it('returns source as string', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
          source: 1,
        },
        settings: {},
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getStripeSettings();

      expect(typeof result.source).toBe('string');
    });

    it('handles undefined source by falling back to env', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
          source: 0,
        },
        settings: {},
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getStripeSettings();

      expect(typeof result.source).toBe('string');
    });
  });

  describe('updateStripeSettings', () => {
    it('sends PUT request with payload', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
        },
        settings: {},
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const payload: StripeSettingsUpdatePayload = {
        publishable_key: 'pk_test_new',
        secret_key: 'sk_test_new',
      };

      await updateStripeSettings(payload);

      const [, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('PUT');
      expect(JSON.parse(options.body as string)).toEqual(payload);
    });

    it('returns updated settings', async () => {
      const protoResponse = {
        snapshot: {
          publishableKeyPreview: 'pk_test_new_xxx',
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
        },
        settings: {
          dashboardUrl: 'https://dashboard.stripe.com',
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await updateStripeSettings({
        publishable_key: 'pk_test_new',
      });

      expect(result.publishable_key_preview).toBe('pk_test_new_xxx');
    });
  });

  describe('getBundleCatalog', () => {
    it('returns bundle catalog', async () => {
      const response: BundleCatalogResponse = {
        bundles: [
          {
            bundle: {
              id: 1,
              bundle_key: 'main',
              name: 'Main Bundle',
              stripe_product_id: 'prod_123',
              credits_per_usd: 1000000,
              display_credits_multiplier: 0.001,
              display_credits_label: 'credits',
            },
            prices: [
              {
                plan_name: 'Pro',
                plan_tier: 'pro',
                billing_interval: 'month',
                amount_cents: 9900,
                currency: 'usd',
                intro_enabled: false,
                stripe_price_id: 'price_123',
                monthly_included_credits: 10000000,
                one_time_bonus_credits: 0,
                display_enabled: true,
                display_weight: 50,
              },
            ],
          },
        ],
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await getBundleCatalog();

      expect(result.bundles).toHaveLength(1);
      expect(result.bundles[0]?.bundle.bundle_key).toBe('main');
      expect(result.bundles[0]?.prices[0]?.plan_name).toBe('Pro');
    });
  });

  describe('updateBundlePrice', () => {
    it('sends PATCH request with correct URL and payload', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({
        plan_name: 'Pro Plus',
        plan_tier: 'pro',
        billing_interval: 'month',
        amount_cents: 9900,
        currency: 'usd',
        intro_enabled: false,
        stripe_price_id: 'price_123',
        monthly_included_credits: 0,
        one_time_bonus_credits: 0,
        display_enabled: true,
        display_weight: 75,
      }));

      await updateBundlePrice('main', 'price_123', {
        plan_name: 'Pro Plus',
        display_weight: 75,
      });

      const [url, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('PATCH');
      expect(url).toContain('/admin/bundles/main/prices/price_123');
      expect(JSON.parse(options.body as string)).toEqual({
        plan_name: 'Pro Plus',
        display_weight: 75,
      });
    });

    it('URL encodes bundle key and price id', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({
        plan_name: 'Encoded Plan',
        plan_tier: 'pro',
        billing_interval: 'month',
        amount_cents: 9900,
        currency: 'usd',
        intro_enabled: false,
        stripe_price_id: 'price+special',
        monthly_included_credits: 0,
        one_time_bonus_credits: 0,
        display_enabled: true,
        display_weight: 10,
      }));

      await updateBundlePrice('bundle/with/slashes', 'price+special', {});

      const [url] = getFetchCall(fetchMock);
      expect(url).toContain(encodeURIComponent('bundle/with/slashes'));
      expect(url).toContain(encodeURIComponent('price+special'));
    });
  });

  describe('createBundlePrice', () => {
    it('sends POST request with payload and returns plan', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({
        plan_name: 'New Plan',
        plan_tier: 'pro',
        billing_interval: 'month',
        amount_cents: 2900,
        currency: 'usd',
        intro_enabled: false,
        stripe_price_id: 'price_new',
        monthly_included_credits: 0,
        one_time_bonus_credits: 0,
        display_enabled: true,
        display_weight: 10,
      }));

      const result = await createBundlePrice('main', {
        stripe_price_id: 'price_new',
        plan_name: 'New Plan',
        plan_tier: 'pro',
        billing_interval: 'month',
        amount_cents: 2900,
      });

      const [url, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('POST');
      expect(url).toContain('/admin/bundles/main/prices');
      expect(result.plan_name).toBe('New Plan');
    });
  });

  describe('verifyStripePrice', () => {
    it('returns price details on success', async () => {
      const response = {
        id: 'price_123',
        lookup_key: 'pro_monthly',
        currency: 'usd',
        amount_cents: 9900,
        interval: 'month',
        active: true,
        product: 'prod_123',
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await verifyStripePrice('price_123');

      expect(result.id).toBe('price_123');
      expect(result.lookup_key).toBe('pro_monthly');
      expect(result.active).toBe(true);
    });

    it('includes lookup key in query', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({ id: 'price_123' }));

      await verifyStripePrice('pro_monthly');

      const [url] = getFetchCall(fetchMock);
      expect(url).toContain('key=pro_monthly');
    });

    it('throws on invalid price', async () => {
      fetchMock.mockResolvedValue(mockResponses.notFound('Price not found'));

      await expect(verifyStripePrice('invalid_price')).rejects.toBeInstanceOf(ApiError);
    });
  });

  describe('getStripeImportPreview', () => {
    it('returns import preview data', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({
        bundle_product_id: 'prod_123',
        bundle_product_found: true,
        bundle_plan_count: 0,
        products: [
          {
            product_id: 'prod_123',
            product_name: 'Main Bundle',
            is_current_bundle: true,
            prices: [
              {
                price_id: 'price_123',
                lookup_key: 'pro_monthly',
                currency: 'usd',
                amount_cents: 9900,
                interval: 'month',
                product_id: 'prod_123',
                product_name: 'Main Bundle',
                active: true,
                exists_locally: false,
              },
            ],
          },
        ],
        total_prices: 1,
        conflict_count: 0,
        new_count: 1,
      }));

      const result = await getStripeImportPreview();

      expect(result.total_prices).toBe(1);
      expect(result.products[0]?.prices[0]?.price_id).toBe('price_123');
    });
  });

  describe('importStripePlans', () => {
    it('returns import results', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({
        imported: 1,
        overwritten: 0,
        skipped: 0,
        errors: [],
      }));

      const result = await importStripePlans({
        bundle_product_id: 'prod_123',
        mode: 'replace',
        selections: [{ price_id: 'price_123', action: 'import' }],
      });

      expect(result.imported).toBe(1);
    });
  });

  describe('createCheckoutSession', () => {
    it('sends POST request with required fields', async () => {
      const session = {
        session_id: 'cs_123',
        url: 'https://checkout.stripe.com/session/123',
      };
      fetchMock.mockResolvedValue(mockResponses.success({ session }));

      await createCheckoutSession({
        price_id: 'price_123',
      });

      const [, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('POST');
      const body = JSON.parse(options.body as string);
      expect(body.price_id).toBe('price_123');
    });

    it('includes optional customer_email', async () => {
      const session = { session_id: 'cs_123', url: 'https://checkout.stripe.com/123' };
      fetchMock.mockResolvedValue(mockResponses.success({ session }));

      await createCheckoutSession({
        price_id: 'price_123',
        customer_email: 'user@example.com',
      });

      const [, options] = getFetchCall(fetchMock);
      const body = JSON.parse(options.body as string);
      expect(body.customer_email).toBe('user@example.com');
    });

    it('includes success and cancel URLs', async () => {
      const session = { session_id: 'cs_123', url: 'https://checkout.stripe.com/123' };
      fetchMock.mockResolvedValue(mockResponses.success({ session }));

      await createCheckoutSession({
        price_id: 'price_123',
        success_url: 'https://example.com/success',
        cancel_url: 'https://example.com/cancel',
      });

      const [, options] = getFetchCall(fetchMock);
      const body = JSON.parse(options.body as string);
      expect(body.success_url).toBe('https://example.com/success');
      expect(body.cancel_url).toBe('https://example.com/cancel');
    });

    it('returns session object directly', async () => {
      const session = {
        session_id: 'cs_123',
        url: 'https://checkout.stripe.com/session/123',
      };
      fetchMock.mockResolvedValue(mockResponses.success({ session }));

      const result = await createCheckoutSession({ price_id: 'price_123' });

      expect(result).toEqual(session);
    });
  });

  describe('createCreditsCheckoutSession', () => {
    it('sends POST request with required fields', async () => {
      const session = { session_id: 'cs_credits_123', url: 'https://checkout.stripe.com/123' };
      fetchMock.mockResolvedValue(mockResponses.success({ session }));

      await createCreditsCheckoutSession({
        price_id: 'price_credits_100',
        customer_email: 'user@example.com',
      });

      const [url, options] = getFetchCall(fetchMock);
      expect(options.method).toBe('POST');
      expect(url).toContain('/billing/create-credits-checkout-session');
      const body = JSON.parse(options.body as string);
      expect(body.price_id).toBe('price_credits_100');
      expect(body.customer_email).toBe('user@example.com');
    });
  });

  describe('createBillingPortalSession', () => {
    it('calls endpoint without params', async () => {
      const response = { url: 'https://billing.stripe.com/portal/123' };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await createBillingPortalSession();

      const [url] = getFetchCall(fetchMock);
      expect(url).toContain('/billing/portal-url');
      expect(url).not.toContain('?');
      expect(result.url).toBe('https://billing.stripe.com/portal/123');
    });

    it('includes return_url param when provided', async () => {
      const response = { url: 'https://billing.stripe.com/portal/123' };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      await createBillingPortalSession('https://example.com/return');

      const [url] = getFetchCall(fetchMock);
      expect(url).toContain('return_url=');
      expect(url).toContain(encodeURIComponent('https://example.com/return'));
    });

    it('includes user param when provided', async () => {
      const response = { url: 'https://billing.stripe.com/portal/123' };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      await createBillingPortalSession(undefined, 'user@example.com');

      const [url] = getFetchCall(fetchMock);
      expect(url).toContain('user=user%40example.com');
    });

    it('includes both params when provided', async () => {
      const response = { url: 'https://billing.stripe.com/portal/123' };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      await createBillingPortalSession('https://example.com', 'user@example.com');

      const [url] = getFetchCall(fetchMock);
      expect(url).toContain('return_url=');
      expect(url).toContain('user=');
    });
  });
});
