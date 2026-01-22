import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import type { StripeSettingsResponse, BundleCatalogEntry, BundleProduct, PlanOption } from '../../../shared/api';
import { useBillingForm } from './useBillingForm';

// Mock the billing service
const loadStripeSettingsMock = vi.fn();
const saveStripeSettingsMock = vi.fn();
const loadBundleCatalogMock = vi.fn();
const savePriceFormMock = vi.fn();
const verifyPriceIdMock = vi.fn();

vi.mock('../services/billing.service', async () => {
  const actual = await vi.importActual<typeof import('../services/billing.service')>('../services/billing.service');
  return {
    ...actual,
    loadStripeSettings: (...args: unknown[]) => loadStripeSettingsMock(...args),
    saveStripeSettings: (...args: unknown[]) => saveStripeSettingsMock(...args),
    loadBundleCatalog: (...args: unknown[]) => loadBundleCatalogMock(...args),
    savePriceForm: (...args: unknown[]) => savePriceFormMock(...args),
    verifyPriceId: (...args: unknown[]) => verifyPriceIdMock(...args),
  };
});

// Mock the pricing service
vi.mock('../services/pricing.service', async () => {
  const actual = await vi.importActual<typeof import('../services/pricing.service')>('../services/pricing.service');
  return {
    ...actual,
    enrichBundlesWithDemo: (bundles: BundleCatalogEntry[], includeDemo: boolean) => {
      if (!includeDemo) return bundles;
      // Return bundles with a demo plan added
      return bundles.map((entry) => ({
        ...entry,
        prices: [
          ...entry.prices,
          {
            plan_name: 'Demo Plan',
            plan_tier: 'demo',
            stripe_price_id: 'demo_123',
            billing_interval: 'month',
            amount_cents: 0,
            currency: 'usd',
            intro_enabled: false,
            monthly_included_credits: 0,
            one_time_bonus_credits: 0,
            display_enabled: true,
            display_weight: 50,
            metadata: { __demo_placeholder: true },
          } as PlanOption,
        ],
      }));
    },
  };
});

const mockStripeSettings: StripeSettingsResponse = {
  publishable_key_preview: 'pk_test_xxx',
  publishable_key_set: true,
  secret_key_set: true,
  webhook_secret_set: false,
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
  plan_name: 'Pro Plan',
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

const mockBundles: BundleCatalogEntry[] = [
  {
    bundle: mockBundle,
    prices: [mockPlan],
  },
];

describe('useBillingForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    loadStripeSettingsMock.mockResolvedValue(mockStripeSettings);
    loadBundleCatalogMock.mockResolvedValue({ bundles: mockBundles });
  });

  describe('initial load', () => {
    it('loads Stripe settings on mount', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingStripe).toBe(false);
      });

      expect(loadStripeSettingsMock).toHaveBeenCalledTimes(1);
      expect(result.current.stripeSettings).toEqual(mockStripeSettings);
    });

    it('loads bundle catalog on mount', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      expect(loadBundleCatalogMock).toHaveBeenCalledTimes(1);
      expect(result.current.bundles).toHaveLength(1);
    });

    it('builds price forms from bundles', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      expect(result.current.priceForms['test_bundle:price_123']).toBeDefined();
      expect(result.current.priceForms['test_bundle:price_123'].values.planName).toBe('Pro Plan');
    });

    it('handles Stripe settings load error', async () => {
      loadStripeSettingsMock.mockRejectedValue(new Error('API failure'));

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingStripe).toBe(false);
      });

      expect(result.current.stripeError).toBe('API failure');
      expect(result.current.stripeSettings).toBeNull();
    });

    it('handles bundle catalog load error', async () => {
      loadBundleCatalogMock.mockRejectedValue(new Error('Bundle API failure'));

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      expect(result.current.bundleError).toBe('Bundle API failure');
    });
  });

  describe('Stripe form handling', () => {
    it('updates Stripe form field on input change', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingStripe).toBe(false);
      });

      act(() => {
        const handler = result.current.handleStripeInput('publishableKey');
        handler({ target: { value: 'pk_test_new' } } as React.ChangeEvent<HTMLInputElement>);
      });

      expect(result.current.stripeForm.publishableKey).toBe('pk_test_new');
    });

    it('saves Stripe settings and resets form', async () => {
      saveStripeSettingsMock.mockResolvedValue(mockStripeSettings);

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingStripe).toBe(false);
      });

      act(() => {
        result.current.handleStripeInput('publishableKey')({
          target: { value: 'pk_test_new' },
        } as React.ChangeEvent<HTMLInputElement>);
      });

      await act(async () => {
        await result.current.handleStripeSave({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(saveStripeSettingsMock).toHaveBeenCalled();
      expect(result.current.stripeForm.publishableKey).toBe('');
    });

    it('handles Stripe save error', async () => {
      saveStripeSettingsMock.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingStripe).toBe(false);
      });

      act(() => {
        result.current.handleStripeInput('publishableKey')({
          target: { value: 'pk_test_new' },
        } as React.ChangeEvent<HTMLInputElement>);
      });

      await act(async () => {
        await result.current.handleStripeSave({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(result.current.stripeError).toBe('Save failed');
    });
  });

  describe('price form handling', () => {
    it('updates price form field on change', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      act(() => {
        const handler = result.current.handlePriceChange('test_bundle', 'price_123', 'planName');
        handler({ target: { value: 'Updated Plan' } } as React.ChangeEvent<HTMLInputElement>);
      });

      expect(result.current.priceForms['test_bundle:price_123'].values.planName).toBe('Updated Plan');
    });

    it('handles displayWeight as number', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      act(() => {
        const handler = result.current.handlePriceChange('test_bundle', 'price_123', 'displayWeight');
        handler({ target: { value: '75' } } as React.ChangeEvent<HTMLInputElement>);
      });

      expect(result.current.priceForms['test_bundle:price_123'].values.displayWeight).toBe(75);
    });

    it('handles boolean fields (highlight)', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      act(() => {
        const handler = result.current.handlePriceChange('test_bundle', 'price_123', 'highlight');
        handler({ target: { checked: true } } as unknown as React.ChangeEvent<HTMLInputElement>);
      });

      expect(result.current.priceForms['test_bundle:price_123'].values.highlight).toBe(true);
    });

    it('handles displayEnabled boolean', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      act(() => {
        const handler = result.current.handlePriceChange('test_bundle', 'price_123', 'displayEnabled');
        handler({ target: { checked: false } } as unknown as React.ChangeEvent<HTMLInputElement>);
      });

      expect(result.current.priceForms['test_bundle:price_123'].values.displayEnabled).toBe(false);
    });
  });

  describe('price form saving', () => {
    it('saves dirty price form', async () => {
      savePriceFormMock.mockResolvedValue({});
      loadBundleCatalogMock.mockResolvedValue({ bundles: mockBundles });

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      act(() => {
        result.current.handlePriceChange('test_bundle', 'price_123', 'planName')({
          target: { value: 'Modified Plan' },
        } as React.ChangeEvent<HTMLInputElement>);
      });

      await act(async () => {
        await result.current.handleSavePrice('test_bundle', 'price_123');
      });

      expect(savePriceFormMock).toHaveBeenCalled();
    });

    it('does not save clean price form', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      await act(async () => {
        await result.current.handleSavePrice('test_bundle', 'price_123');
      });

      expect(savePriceFormMock).not.toHaveBeenCalled();
    });

    it('handles save error', async () => {
      savePriceFormMock.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      act(() => {
        result.current.handlePriceChange('test_bundle', 'price_123', 'planName')({
          target: { value: 'Modified' },
        } as React.ChangeEvent<HTMLInputElement>);
      });

      await act(async () => {
        await result.current.handleSavePrice('test_bundle', 'price_123');
      });

      expect(result.current.priceForms['test_bundle:price_123'].error).toBe('Save failed');
    });
  });

  describe('price verification', () => {
    it('verifies Stripe price ID', async () => {
      verifyPriceIdMock.mockResolvedValue({ status: 'ok', message: 'Verified' });

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      await act(async () => {
        await result.current.handleVerifyPrice('test_bundle', 'price_123');
      });

      expect(verifyPriceIdMock).toHaveBeenCalledWith('price_123');
      expect(result.current.priceChecks['test_bundle:price_123'].status).toBe('ok');
    });

    it('shows checking status during verification', async () => {
      let resolveVerify: (value: unknown) => void;
      verifyPriceIdMock.mockReturnValue(
        new Promise((resolve) => {
          resolveVerify = resolve;
        })
      );

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      act(() => {
        result.current.handleVerifyPrice('test_bundle', 'price_123');
      });

      expect(result.current.priceChecks['test_bundle:price_123'].status).toBe('checking');

      await act(async () => {
        resolveVerify!({ status: 'ok', message: 'Done' });
      });
    });
  });

  describe('demo placeholders', () => {
    it('toggles demo placeholders', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      expect(result.current.includeDemoPlaceholders).toBe(false);

      act(() => {
        result.current.toggleDemoPlaceholders();
      });

      expect(result.current.includeDemoPlaceholders).toBe(true);
    });

    it('removes demo plan from UI', async () => {
      // Start with demo placeholders enabled
      loadBundleCatalogMock.mockResolvedValue({
        bundles: [
          {
            bundle: mockBundle,
            prices: [
              mockPlan,
              {
                ...mockPlan,
                stripe_price_id: 'demo_plan',
                plan_name: 'Demo Plan',
                metadata: { __demo_placeholder: true },
              },
            ],
          },
        ],
      });

      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      const initialPriceCount = result.current.bundles[0].prices.length;

      act(() => {
        result.current.removeDemoPlan('test_bundle', 'demo_plan');
      });

      expect(result.current.bundles[0].prices.length).toBe(initialPriceCount - 1);
      expect(result.current.priceForms['test_bundle:demo_plan']).toBeUndefined();
    });
  });

  describe('pricing tab', () => {
    it('initializes with month tab', async () => {
      const { result } = renderHook(() => useBillingForm());

      expect(result.current.pricingTab).toBe('month');
    });

    it('allows changing pricing tab', async () => {
      const { result } = renderHook(() => useBillingForm());

      act(() => {
        result.current.setPricingTab('year');
      });

      expect(result.current.pricingTab).toBe('year');
    });
  });

  describe('Stripe status badges', () => {
    it('builds status badges from settings', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingStripe).toBe(false);
      });

      expect(result.current.stripeStatusBadges).toHaveLength(3);
      expect(result.current.stripeStatusBadges[0]).toEqual({ label: 'Publishable Key', ok: true });
      expect(result.current.stripeStatusBadges[1]).toEqual({ label: 'Restricted Key', ok: true });
      expect(result.current.stripeStatusBadges[2]).toEqual({ label: 'Webhook Secret', ok: false });
    });
  });

  describe('reload functions', () => {
    it('allows reloading Stripe settings', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingStripe).toBe(false);
      });

      loadStripeSettingsMock.mockClear();

      await act(async () => {
        await result.current.loadStripe();
      });

      expect(loadStripeSettingsMock).toHaveBeenCalled();
    });

    it('allows reloading bundles', async () => {
      const { result } = renderHook(() => useBillingForm());

      await waitFor(() => {
        expect(result.current.loadingBundles).toBe(false);
      });

      loadBundleCatalogMock.mockClear();

      await act(async () => {
        await result.current.loadBundles();
      });

      expect(loadBundleCatalogMock).toHaveBeenCalled();
    });
  });
});
