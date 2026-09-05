import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as billing from './billing';
import { apiCall } from './common';

const paymentsClient = vi.hoisted(() => ({ createCheckoutSession: vi.fn(), getBillingPortal: vi.fn() }));
const stripeSettingsClient = vi.hoisted(() => ({ getStripeSettings: vi.fn(), updateStripeSettings: vi.fn(), revealStripeSecret: vi.fn() }));
const bundleAdminClient = vi.hoisted(() => ({ listBundleCatalog: vi.fn(), updateBundlePrice: vi.fn() }));
const couponAdminClient = vi.hoisted(() => ({ listCoupons: vi.fn(), createCoupon: vi.fn(), getCoupon: vi.fn(), updateCoupon: vi.fn(), deleteCoupon: vi.fn(), listCouponUsage: vi.fn(), getCouponMappings: vi.fn(), setCouponForPlan: vi.fn(), removeCouponFromPlan: vi.fn(), getCouponImportPreview: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn((service: { typeName?: string }) => {
  if (service.typeName === 'landing_page_business_suite.v1.StripeSettingsService') return stripeSettingsClient;
  if (service.typeName === 'landing_page_business_suite.v1.BundleAdminService') return bundleAdminClient;
  if (service.typeName === 'landing_page_business_suite.v1.CouponAdminService') return couponAdminClient;
  return paymentsClient;
}) }));
vi.mock('./common', () => ({ apiCall: vi.fn(), CONNECT_API_BASE: 'http://api.example.test' }));
const mockApiCall = vi.mocked(apiCall);

describe('billing API transport', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mockApiCall.mockResolvedValue({});
    paymentsClient.createCheckoutSession.mockResolvedValue({});
    paymentsClient.getBillingPortal.mockResolvedValue({});
		stripeSettingsClient.getStripeSettings.mockResolvedValue({});
		stripeSettingsClient.updateStripeSettings.mockResolvedValue({});
		stripeSettingsClient.revealStripeSecret.mockResolvedValue({});
    bundleAdminClient.listBundleCatalog.mockResolvedValue({});
    bundleAdminClient.updateBundlePrice.mockResolvedValue({});
    couponAdminClient.listCoupons.mockResolvedValue({}); couponAdminClient.createCoupon.mockResolvedValue({});
    couponAdminClient.getCoupon.mockResolvedValue({}); couponAdminClient.updateCoupon.mockResolvedValue({});
    couponAdminClient.deleteCoupon.mockResolvedValue({ deleted: true }); couponAdminClient.listCouponUsage.mockResolvedValue({});
    couponAdminClient.getCouponMappings.mockResolvedValue({}); couponAdminClient.setCouponForPlan.mockResolvedValue({ assigned: true });
    couponAdminClient.removeCouponFromPlan.mockResolvedValue({ removed: true }); couponAdminClient.getCouponImportPreview.mockResolvedValue({});
  });

  it('uses settings, catalog, price, and checkout endpoints and rejects malformed required responses', async () => {
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: 'env' });
    await expect(billing.updateStripeSettings({ dashboard_url: 'https://dashboard.stripe.com' })).resolves.toMatchObject({ source: 'env' });
    await expect(billing.revealStripeSecret('secret_key')).resolves.toEqual({});
    await expect(billing.getBundleCatalog()).resolves.toEqual({ bundles: [] });
    await expect(billing.updateBundlePrice('starter plan', 'price/1', { plan_name: 'Starter' })).rejects.toThrow('Invalid plan response from update');
    await expect(billing.verifyStripePrice('lookup key')).rejects.toThrow('Invalid price verification response from Stripe');
    await expect(billing.createCheckoutSession({ price_id: 'price_1', customer_email: 'customer@example.com' })).rejects.toThrow('Invalid checkout session response');
    await expect(billing.createCreditsCheckoutSession({ price_id: 'price_credits', customer_email: 'customer@example.com' })).rejects.toThrow('Invalid credits checkout session response');
    await expect(billing.createBillingPortalSession('https://example.com/return', 'customer@example.com')).rejects.toThrow('Invalid billing portal response');
		expect(stripeSettingsClient.getStripeSettings).toHaveBeenCalledWith({});
		expect(stripeSettingsClient.updateStripeSettings).toHaveBeenCalledWith({ dashboardUrl: 'https://dashboard.stripe.com', publishableKey: undefined, secretKey: undefined, webhookSecret: undefined, anomalyWebhookUrl: undefined, anomalyWebhookEnabled: undefined, anomalyRateLimits: undefined });
		expect(stripeSettingsClient.revealStripeSecret).toHaveBeenCalledWith({ field: 'secret_key' });
    expect(bundleAdminClient.listBundleCatalog).toHaveBeenCalledWith({});
    expect(bundleAdminClient.updateBundlePrice).toHaveBeenCalledWith(expect.objectContaining({
      bundleKey: 'starter plan', priceId: 'price/1', planName: 'Starter', featuresPresent: false,
    }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/stripe/verify-price?key=lookup+key');
    expect(paymentsClient.createCheckoutSession).toHaveBeenCalledTimes(2);
    expect(paymentsClient.getBillingPortal).toHaveBeenCalledWith({ returnUrl: 'https://example.com/return' });
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
    couponAdminClient.listCoupons.mockResolvedValueOnce({ coupons: [{}] });
    await expect(billing.listCoupons()).rejects.toThrow('Invalid coupons list response');
    await expect(billing.createCoupon({ duration: 'forever', percent_off: 20 })).rejects.toThrow('Invalid coupon response');
    await expect(billing.getCoupon('coupon/1')).rejects.toThrow('Invalid coupon response');
    await expect(billing.updateCoupon('coupon/1', { name: 'Launch offer' })).rejects.toThrow('Invalid coupon response');
    await expect(billing.getCouponUsage()).resolves.toEqual([]);
    await expect(billing.getCouponMappings()).resolves.toEqual({ mappings: {} });
    await expect(billing.getStripeCouponPreview()).rejects.toThrow('Invalid coupon import preview response');
    await expect(billing.deleteCoupon('coupon/1')).resolves.toBeUndefined();
    await expect(billing.setCouponForPlan('price/1', 'coupon/1')).resolves.toBeUndefined();
    await expect(billing.removeCouponFromPlan('price/1')).resolves.toBeUndefined();
    expect(couponAdminClient.listCoupons).toHaveBeenCalledWith({});
    expect(couponAdminClient.createCoupon).toHaveBeenCalledWith(expect.objectContaining({ duration: 3, percentOff: 20 }));
    expect(couponAdminClient.getCoupon).toHaveBeenCalledWith({ couponId: 'coupon/1' });
    expect(couponAdminClient.updateCoupon).toHaveBeenCalledWith({ couponId: 'coupon/1', name: 'Launch offer' });
    expect(couponAdminClient.listCouponUsage).toHaveBeenCalledWith({});
    expect(couponAdminClient.getCouponMappings).toHaveBeenCalledWith({});
    expect(couponAdminClient.getCouponImportPreview).toHaveBeenCalledWith({});
    expect(couponAdminClient.deleteCoupon).toHaveBeenCalledWith({ couponId: 'coupon/1' });
    expect(couponAdminClient.setCouponForPlan).toHaveBeenCalledWith({ priceId: 'price/1', couponId: 'coupon/1' });
    expect(couponAdminClient.removeCouponFromPlan).toHaveBeenCalledWith({ priceId: 'price/1' });
  });

  it('returns validated checkout and portal sessions while omitting optional customer and query fields', async () => {
    const checkout = { session_id: 'cs_123', url: 'https://checkout.stripe.example/session' };
    const connectCheckout = { sessionId: 'cs_123', url: checkout.url, customerEmail: '', stripePriceId: '', amountCents: 0n, currency: '', successUrl: '', cancelUrl: '' };
    paymentsClient.createCheckoutSession
      .mockResolvedValueOnce({ session: connectCheckout })
      .mockResolvedValueOnce({ session: connectCheckout })
      .mockResolvedValueOnce({ session: connectCheckout });
    paymentsClient.getBillingPortal
      .mockResolvedValueOnce({ url: 'https://billing.stripe.example/portal' })
      .mockResolvedValueOnce({ url: 'https://billing.stripe.example/portal' });

    await expect(billing.createCheckoutSession({ price_id: 'price_1' })).resolves.toEqual(checkout);
    await expect(billing.createCheckoutSession({ price_id: 'price_2', customer_email: 'customer@example.com', success_url: 'https://app.example/success', cancel_url: 'https://app.example/cancel' })).resolves.toEqual(checkout);
    await expect(billing.createCreditsCheckoutSession({ price_id: 'price_credits', customer_email: 'customer@example.com' })).resolves.toEqual(checkout);
    await expect(billing.createBillingPortalSession()).resolves.toEqual({ url: 'https://billing.stripe.example/portal' });
    await expect(billing.createBillingPortalSession('https://app.example/return')).resolves.toEqual({ url: 'https://billing.stripe.example/portal' });

    expect(paymentsClient.createCheckoutSession).toHaveBeenCalledWith(expect.objectContaining({ priceId: 'price_1' }));
    expect(paymentsClient.createCheckoutSession).toHaveBeenCalledWith(expect.objectContaining({ priceId: 'price_credits' }));
    expect(paymentsClient.getBillingPortal).toHaveBeenCalledWith({ returnUrl: '' });
    expect(paymentsClient.getBillingPortal).toHaveBeenCalledWith({ returnUrl: 'https://app.example/return' });
  });

  it('preserves populated generated checkout fields in the legacy UI shape', async () => {
    paymentsClient.createCheckoutSession.mockResolvedValue({
      session: {
        sessionId: 'cs_full', url: 'https://checkout.stripe.example/full', customerEmail: 'buyer@example.test', stripePriceId: 'price_full',
        amountCents: 4900n, currency: 'usd', successUrl: 'https://app.example/success', cancelUrl: 'https://app.example/cancel',
      },
    });
    await expect(billing.createCheckoutSession({ price_id: 'price_full' })).resolves.toEqual({
      session_id: 'cs_full', url: 'https://checkout.stripe.example/full', customer_email: 'buyer@example.test', stripe_price_id: 'price_full',
      amount_cents: 4900, currency: 'usd', success_url: 'https://app.example/success', cancel_url: 'https://app.example/cancel',
    });
  });

  it('normalizes BundleAdmin Connect responses and retains explicit empty feature updates', async () => {
    const price = {
      planName: 'Pro', planTier: 'pro', billingInterval: 1, amountCents: 4900n, currency: 'usd',
      stripePriceId: 'price_pro', monthlyIncludedCredits: 100n, oneTimeBonusCredits: 0n,
      displayEnabled: true, displayWeight: 10,
    };
    bundleAdminClient.listBundleCatalog.mockResolvedValue({
      bundles: [{ bundle: { bundleKey: 'business', name: 'Business', stripeProductId: 'prod_business', creditsPerUsd: 1000000n, displayCreditsMultiplier: 1, displayCreditsLabel: 'credits' }, prices: [price] }],
    });
    bundleAdminClient.updateBundlePrice.mockResolvedValue({ price });

    await expect(billing.getBundleCatalog()).resolves.toMatchObject({ bundles: [{ bundle: { bundle_key: 'business' }, prices: [{ stripe_price_id: 'price_pro', amount_cents: 4900 }] }] });
    await expect(billing.updateBundlePrice('business', 'price_pro', { features: [] })).resolves.toMatchObject({ stripe_price_id: 'price_pro' });
    expect(bundleAdminClient.updateBundlePrice).toHaveBeenLastCalledWith(expect.objectContaining({ features: [], featuresPresent: true }));
  });

  it('normalizes non-default generated bundle plan enum variants', async () => {
    const base = { planName: 'Variant', planTier: 'variant', amountCents: 1200n, currency: 'usd', stripePriceId: 'price_variant', monthlyIncludedCredits: 0n, oneTimeBonusCredits: 0n, displayEnabled: true, displayWeight: 0 };
    bundleAdminClient.listBundleCatalog.mockResolvedValue({ bundles: [{ bundle: { bundleKey: 'business', name: 'Business', creditsPerUsd: 1000000n, displayCreditsMultiplier: 1, displayCreditsLabel: 'credits' }, prices: [
      { ...base, billingInterval: 2, kind: 2, introType: 2 },
      { ...base, planName: 'Contribution', planTier: 'contribution', stripePriceId: 'price_contribution', billingInterval: 3, kind: 3, introType: 1 },
    ] }] });

    await expect(billing.getBundleCatalog()).resolves.toMatchObject({ bundles: [{ prices: [
      { billing_interval: 'year', kind: 'credits_topup', intro_type: 'percentage' },
      { billing_interval: 'one_time', kind: 'supporter_contribution', intro_type: 'flat_amount' },
    ] }] });
  });

  it('preserves populated generated bundle optional fields and metadata', async () => {
    bundleAdminClient.listBundleCatalog.mockResolvedValue({ bundles: [{
      bundle: { bundleKey: 'business', name: 'Business', stripeProductId: 'prod_business', creditsPerUsd: 1000000n, displayCreditsMultiplier: 2, displayCreditsLabel: 'tokens', environment: 'production', metadata: { region: { toJson: () => ({ value: 'us-east-1' }) } } },
      prices: [{ planName: 'Flexible', planTier: 'flexible', billingInterval: 1, amountCents: 2500n, currency: 'usd', introEnabled: true, introAmountCents: 250n, introPeriods: 2n, introPriceLookupKey: 'intro_flexible', stripePriceId: 'price_flexible', monthlyIncludedCredits: 10n, oneTimeBonusCredits: 5n, planRank: 3n, bonusType: 'launch', isVariableAmount: true, displayEnabled: true, displayWeight: 4n, bundleKey: 'business', metadata: { label: { toJson: () => ({ text: 'Flexible' }) } } }],
    }] });

    await expect(billing.getBundleCatalog()).resolves.toMatchObject({ bundles: [{ bundle: { environment: 'production', metadata: { region: { value: 'us-east-1' } } }, prices: [{ intro_amount_cents: 250, intro_periods: 2, intro_price_lookup_key: 'intro_flexible', plan_rank: 3, bonus_type: 'launch', is_variable_amount: true, metadata: { label: { text: 'Flexible' } } }] }] });
  });

  it('normalizes protobuf zero values for a partially configured bundle plan', async () => {
    bundleAdminClient.listBundleCatalog.mockResolvedValue({ bundles: [{ bundle: {}, prices: [{ billingInterval: 1, amountCents: 0n, currency: 'usd', stripePriceId: 'price_zero_values' }] }] });
    await expect(billing.getBundleCatalog()).resolves.toMatchObject({ bundles: [{ prices: [{ plan_name: '', plan_tier: '', monthly_included_credits: 0, one_time_bonus_credits: 0, display_weight: 0 }] }] });
  });


  it('normalizes Stripe setting snapshots from every supported config source', async () => {
    stripeSettingsClient.getStripeSettings
      .mockResolvedValueOnce({
        snapshot: { publishableKeyPreview: 'pk_live_…', publishableKeySet: true, secretKeySet: true, webhookSecretSet: false, source: 2 },
        settings: { dashboardUrl: 'https://dashboard.stripe.com', updatedAt: '2026-01-01T00:00:00Z' },
      })
      .mockResolvedValueOnce({ snapshot: { source: 0 }, settings: {} })
      .mockResolvedValueOnce({ snapshot: { source: 'managed' }, settings: {} })
      .mockResolvedValueOnce({ snapshot: { source: 42 }, settings: {} })
      .mockResolvedValueOnce({ snapshot: { source: {} }, settings: {} })
      .mockResolvedValueOnce({ snapshot: { source: 1 }, settings: {} });

    await expect(billing.getStripeSettings()).resolves.toMatchObject({
      publishable_key_preview: 'pk_live_…', publishable_key_set: true, secret_key_set: true,
      webhook_secret_set: false, source: 'database', dashboard_url: 'https://dashboard.stripe.com',
    });
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: 'env', publishable_key_set: false });
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: 'managed' });
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: '42' });
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: 'env' });
    await expect(billing.getStripeSettings()).resolves.toMatchObject({ source: 'env' });
  });

  it('returns validated Stripe import, verification, and coupon read models', async () => {
    const coupon = { id: 'coupon_1', duration: 1, timesRedeemed: 0, valid: true, created: 1n, isIntroCoupon: false };
    mockApiCall
      .mockResolvedValueOnce({ id: 'price_1', currency: 'usd', amount_cents: 1000, active: true })
      .mockResolvedValueOnce({ products: [{ product_id: 'prod_1', product_name: 'Product', prices: [] }], total_prices: 0, conflict_count: 0, new_count: 0 })
      .mockResolvedValueOnce({ imported: 1, overwritten: 0, skipped: 0 });
    couponAdminClient.listCoupons.mockResolvedValueOnce({ coupons: [coupon], introCouponMap: { price_1: 'coupon_1' } });
    couponAdminClient.getCoupon.mockResolvedValueOnce({ coupon });
    couponAdminClient.listCouponUsage.mockResolvedValueOnce({ usage: [{ couponId: 'coupon_1', totalUses: 3n }] });
    couponAdminClient.getCouponMappings.mockResolvedValueOnce({ mappings: { price_1: 'coupon_1' } });
    couponAdminClient.getCouponImportPreview.mockResolvedValueOnce({ coupons: [{ ...coupon, existsLocally: false }], totalCoupons: 1, existingCount: 0, newCount: 1 });

    await expect(billing.verifyStripePrice('price_1')).resolves.toMatchObject({ id: 'price_1' });
    await expect(billing.getStripeImportPreview()).resolves.toMatchObject({ total_prices: 0 });
    await expect(billing.importStripePlans({ bundle_product_id: 'prod_1', selections: [] })).resolves.toMatchObject({ imported: 1 });
    await expect(billing.listCoupons()).resolves.toMatchObject({ intro_coupon_map: { price_1: 'coupon_1' } });
    await expect(billing.getCoupon('coupon_1')).resolves.toMatchObject({ id: 'coupon_1' });
    await expect(billing.getCouponUsage()).resolves.toEqual([{ coupon_id: 'coupon_1', total_uses: 3, last_used_at: null }]);
    await expect(billing.getCouponMappings()).resolves.toEqual({ mappings: { price_1: 'coupon_1' } });
    await expect(billing.getStripeCouponPreview()).resolves.toMatchObject({ total_coupons: 1 });
  });

  it('preserves every supported coupon duration and optional coupon fields', async () => {
    const once = { id: 'once', name: 'One time', amountOff: 500n, currency: 'usd', duration: 1, durationInMonths: 1, maxRedemptions: 2, redeemBy: 10n, timesRedeemed: 1, valid: true, created: 2n, isIntroCoupon: true, introTier: 'starter' };
    const repeating = { id: 'repeating', percentOff: 15, duration: 2, durationInMonths: 3, timesRedeemed: 0, valid: true, created: 3n, isIntroCoupon: false };
    const forever = { id: 'forever', duration: 3, timesRedeemed: 0, valid: true, created: 4n, isIntroCoupon: false };
    couponAdminClient.listCoupons.mockResolvedValueOnce({ coupons: [once, repeating, forever], introCouponMap: {} });
    couponAdminClient.getCouponImportPreview.mockResolvedValueOnce({ coupons: [{ ...once, existsLocally: true }, { ...repeating, existsLocally: false }, { ...forever, existsLocally: false }], totalCoupons: 3, existingCount: 1, newCount: 2 });
    couponAdminClient.createCoupon.mockResolvedValueOnce({ coupon: once }).mockResolvedValueOnce({ coupon: repeating });

    await expect(billing.listCoupons()).resolves.toMatchObject({ coupons: [{ duration: 'once', amount_off: 500, intro_tier: 'starter' }, { duration: 'repeating', percent_off: 15 }, { duration: 'forever' }] });
    await expect(billing.getStripeCouponPreview()).resolves.toMatchObject({ total_coupons: 3, existing_count: 1, new_count: 2 });
    await expect(billing.createCoupon({ duration: 'once', amount_off: 500, currency: 'usd' })).resolves.toMatchObject({ duration: 'once' });
    await expect(billing.createCoupon({ duration: 'repeating', percent_off: 15, duration_in_months: 3 })).resolves.toMatchObject({ duration: 'repeating' });
    expect(couponAdminClient.createCoupon).toHaveBeenCalledWith(expect.objectContaining({ duration: 1, amountOff: 500n, currency: 'usd' }));
    expect(couponAdminClient.createCoupon).toHaveBeenCalledWith(expect.objectContaining({ duration: 2, percentOff: 15, durationInMonths: 3 }));
  });

  it('fails closed for declined coupon mutations and malformed typed coupon reads', async () => {
    couponAdminClient.deleteCoupon.mockResolvedValueOnce({ deleted: false });
    couponAdminClient.setCouponForPlan.mockResolvedValueOnce({ assigned: false });
    couponAdminClient.removeCouponFromPlan.mockResolvedValueOnce({ removed: false });
    couponAdminClient.listCouponUsage.mockResolvedValueOnce({ usage: [{ couponId: '', totalUses: -1n, lastUsedAt: '2026-01-01T00:00:00Z' }] });
    couponAdminClient.getCouponMappings.mockResolvedValueOnce({ mappings: { price_bad: 42 } });
    couponAdminClient.getCouponImportPreview.mockResolvedValueOnce({ coupons: [{ id: 'unknown', duration: 0, timesRedeemed: 0, valid: true, existsLocally: false }], totalCoupons: 0, existingCount: 0, newCount: 0 });

    await expect(billing.deleteCoupon('coupon_1')).rejects.toThrow('Coupon deletion was not confirmed');
    await expect(billing.setCouponForPlan('price_1', 'coupon_1')).rejects.toThrow('Coupon assignment was not confirmed');
    await expect(billing.removeCouponFromPlan('price_1')).rejects.toThrow('Coupon removal was not confirmed');
    await expect(billing.getCouponUsage()).rejects.toThrow('Invalid coupon usage response');
    await expect(billing.getCouponMappings()).rejects.toThrow('Invalid coupon mappings response');
    await expect(billing.getStripeCouponPreview()).resolves.toMatchObject({ coupons: [], total_coupons: 0 });
  });
});
