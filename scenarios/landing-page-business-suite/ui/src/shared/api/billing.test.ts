import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as billing from './billing';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

describe('billing API transport', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mockApiCall.mockResolvedValue({} as never);
  });

  it('uses settings, catalog, price, and checkout endpoints and rejects malformed required responses', async () => {
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: 'env' });
    await expect(billing.updateStripeSettings({ dashboard_url: 'https://dashboard.stripe.com' })).resolves.toMatchObject({ source: 'env' });
    await expect(billing.revealStripeSecret('secret_key')).resolves.toEqual({});
    await expect(billing.getBundleCatalog()).rejects.toThrow('Invalid bundle catalog response');
    await expect(billing.updateBundlePrice('starter plan', 'price/1', { plan_name: 'Starter' })).rejects.toThrow('Invalid plan response from update');
    await expect(billing.verifyStripePrice('lookup key')).rejects.toThrow('Invalid price verification response from Stripe');
    await expect(billing.createCheckoutSession({ price_id: 'price_1', customer_email: 'customer@example.com' })).rejects.toThrow('Invalid checkout session response');
    await expect(billing.createCreditsCheckoutSession({ price_id: 'price_credits', customer_email: 'customer@example.com' })).rejects.toThrow('Invalid credits checkout session response');
    await expect(billing.createBillingPortalSession('https://example.com/return', 'customer@example.com')).rejects.toThrow('Invalid billing portal response');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/settings/stripe');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/settings/stripe', expect.objectContaining({ method: 'PUT' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/settings/stripe/reveal?field=secret_key');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/bundles');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/bundles/starter%20plan/prices/price%2F1', expect.objectContaining({ method: 'PATCH' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/stripe/verify-price?key=lookup+key');
    expect(mockApiCall).toHaveBeenCalledWith('/billing/create-checkout-session', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/billing/create-credits-checkout-session', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/billing/portal-url?return_url=https%3A%2F%2Fexample.com%2Freturn&user=customer%40example.com');
  });

  it('uses Stripe plan import and lifecycle endpoints while failing closed for invalid payloads', async () => {
    await expect(billing.getStripeImportPreview()).rejects.toThrow('Invalid Stripe import preview response');
    await expect(billing.importStripePlans({ bundle_product_id: 'prod_1', selections: [] })).rejects.toThrow('Invalid Stripe import result');
    await expect(billing.createBundlePrice('starter', { stripe_price_id: 'price_1', plan_name: 'Starter', plan_tier: 'starter', billing_interval: 'month' })).rejects.toThrow('Invalid plan response from create');
    await expect(billing.deleteBundlePrice('starter plan', 'price/1')).resolves.toEqual({});
    expect(mockApiCall).toHaveBeenCalledWith('/admin/stripe/import-preview');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/stripe/import', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/bundles/starter/prices', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/bundles/starter%20plan/prices/price%2F1', { method: 'DELETE' });
  });

  it('uses coupon and mapping endpoints, validates read responses, and preserves destructive request contracts', async () => {
    await expect(billing.listCoupons()).rejects.toThrow('Invalid coupons list response');
    await expect(billing.createCoupon({ duration: 'forever', percent_off: 20 })).rejects.toThrow('Invalid coupon response');
    await expect(billing.getCoupon('coupon/1')).rejects.toThrow('Invalid coupon response');
    await expect(billing.updateCoupon('coupon/1', { name: 'Launch offer' })).rejects.toThrow('Invalid coupon response');
    await expect(billing.getCouponUsage()).rejects.toThrow('Invalid coupon usage response');
    await expect(billing.getCouponMappings()).rejects.toThrow('Invalid coupon mappings response');
    await expect(billing.getStripeCouponPreview()).rejects.toThrow('Invalid coupon import preview response');
    await expect(billing.deleteCoupon('coupon/1')).resolves.toBeUndefined();
    await expect(billing.setCouponForPlan('price/1', 'coupon/1')).resolves.toBeUndefined();
    await expect(billing.removeCouponFromPlan('price/1')).resolves.toBeUndefined();
    expect(mockApiCall).toHaveBeenCalledWith('/admin/coupons');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/coupons', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/coupons/coupon%2F1');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/coupons/coupon%2F1', expect.objectContaining({ method: 'PATCH' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/coupons/usage');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/coupon-mappings');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/stripe/coupons-preview');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/coupons/coupon%2F1', { method: 'DELETE' });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/plans/price%2F1/coupon', expect.objectContaining({ method: 'PUT' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/plans/price%2F1/coupon', { method: 'DELETE' });
  });

  it('returns validated checkout and portal sessions while omitting optional customer and query fields', async () => {
    const checkout = { session_id: 'cs_123', url: 'https://checkout.stripe.example/session' };
    mockApiCall
      .mockResolvedValueOnce({ session: checkout } as never)
      .mockResolvedValueOnce({ session: checkout } as never)
      .mockResolvedValueOnce({ session: checkout } as never)
      .mockResolvedValueOnce({ url: 'https://billing.stripe.example/portal' } as never)
      .mockResolvedValueOnce({ url: 'https://billing.stripe.example/portal' } as never);

    await expect(billing.createCheckoutSession({ price_id: 'price_1' })).resolves.toEqual(checkout);
    await expect(billing.createCheckoutSession({ price_id: 'price_2', customer_email: 'customer@example.com', success_url: 'https://app.example/success', cancel_url: 'https://app.example/cancel' })).resolves.toEqual(checkout);
    await expect(billing.createCreditsCheckoutSession({ price_id: 'price_credits', customer_email: 'customer@example.com' })).resolves.toEqual(checkout);
    await expect(billing.createBillingPortalSession()).resolves.toEqual({ url: 'https://billing.stripe.example/portal' });
    await expect(billing.createBillingPortalSession('https://app.example/return')).resolves.toEqual({ url: 'https://billing.stripe.example/portal' });

    expect(mockApiCall).toHaveBeenCalledWith('/billing/create-checkout-session', {
      method: 'POST',
      body: JSON.stringify({ price_id: 'price_1', success_url: undefined, cancel_url: undefined }),
    });
    expect(mockApiCall).toHaveBeenCalledWith('/billing/create-checkout-session', {
      method: 'POST',
      body: JSON.stringify({ price_id: 'price_2', success_url: 'https://app.example/success', cancel_url: 'https://app.example/cancel', customer_email: 'customer@example.com' }),
    });
    expect(mockApiCall).toHaveBeenCalledWith('/billing/portal-url');
    expect(mockApiCall).toHaveBeenCalledWith('/billing/portal-url?return_url=https%3A%2F%2Fapp.example%2Freturn');
  });

  it('normalizes Stripe setting snapshots from every supported config source', async () => {
    mockApiCall
      .mockResolvedValueOnce({
        snapshot: { publishableKeyPreview: 'pk_live_…', publishableKeySet: true, secretKeySet: true, webhookSecretSet: false, source: 2 },
        settings: { dashboardUrl: 'https://dashboard.stripe.com', updatedAt: '2026-01-01T00:00:00Z' },
      } as never)
      .mockResolvedValueOnce({ snapshot: { source: 0 }, settings: {} } as never);

    await expect(billing.getStripeSettings()).resolves.toMatchObject({
      publishable_key_preview: 'pk_live_…', publishable_key_set: true, secret_key_set: true,
      webhook_secret_set: false, source: 'database', dashboard_url: 'https://dashboard.stripe.com',
    });
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: 'env', publishable_key_set: false });
  });

  it('returns validated Stripe import, verification, and coupon read models', async () => {
    const coupon = { id: 'coupon_1', duration: 'once', times_redeemed: 0, valid: true, created: 1, is_intro_coupon: false };
    mockApiCall
      .mockResolvedValueOnce({ id: 'price_1', currency: 'usd', amount_cents: 1000, active: true } as never)
      .mockResolvedValueOnce({ products: [{ product_id: 'prod_1', product_name: 'Product', prices: [] }], total_prices: 0, conflict_count: 0, new_count: 0 } as never)
      .mockResolvedValueOnce({ imported: 1, overwritten: 0, skipped: 0 } as never)
      .mockResolvedValueOnce({ coupons: [coupon], intro_coupon_map: { price_1: 'coupon_1' } } as never)
      .mockResolvedValueOnce(coupon as never)
      .mockResolvedValueOnce([ { coupon_id: 'coupon_1', total_uses: 3, last_used_at: null } ] as never)
      .mockResolvedValueOnce({ mappings: { price_1: 'coupon_1' } } as never)
      .mockResolvedValueOnce({ coupons: [{ ...coupon, exists_locally: false }], total_coupons: 1, existing_count: 0, new_count: 1 } as never);

    await expect(billing.verifyStripePrice('price_1')).resolves.toMatchObject({ id: 'price_1' });
    await expect(billing.getStripeImportPreview()).resolves.toMatchObject({ total_prices: 0 });
    await expect(billing.importStripePlans({ bundle_product_id: 'prod_1', selections: [] })).resolves.toMatchObject({ imported: 1 });
    await expect(billing.listCoupons()).resolves.toMatchObject({ intro_coupon_map: { price_1: 'coupon_1' } });
    await expect(billing.getCoupon('coupon_1')).resolves.toMatchObject({ id: 'coupon_1' });
    await expect(billing.getCouponUsage()).resolves.toEqual([{ coupon_id: 'coupon_1', total_uses: 3, last_used_at: null }]);
    await expect(billing.getCouponMappings()).resolves.toEqual({ mappings: { price_1: 'coupon_1' } });
    await expect(billing.getStripeCouponPreview()).resolves.toMatchObject({ total_coupons: 1 });
  });
});
