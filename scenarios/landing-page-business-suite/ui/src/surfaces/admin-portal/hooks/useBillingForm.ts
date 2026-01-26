import { useCallback, useEffect, useState } from 'react';
import { getApiErrorMessage, type StripeSettingsResponse, type BundleCatalogEntry } from '../../../shared/api';
import {
  loadStripeSettings,
  saveStripeSettings,
  loadBundleCatalog,
  savePriceForm,
  verifyPriceId,
  buildStripeStatusBadges,
  DEFAULT_STRIPE_FORM,
  type StripeFormState,
  type PriceVerificationResult,
} from '../services/billing.service';
import {
  enrichBundlesWithDemo,
  ensureBundleForDemo,
  buildPriceFormsFromBundles,
  isPriceFormDirty,
  type PriceFormState,
  type PriceFormValues,
} from '../services/pricing.service';

/**
 * Reactive hook for billing settings form management
 *
 * Provides state and handlers for:
 * - Stripe configuration form
 * - Bundle catalog loading
 * - Price form editing and saving
 * - Price verification
 */
export function useBillingForm() {
  // Stripe settings state
  const [stripeSettings, setStripeSettings] = useState<StripeSettingsResponse | null>(null);
  const [stripeForm, setStripeForm] = useState<StripeFormState>(DEFAULT_STRIPE_FORM);
  const [loadingStripe, setLoadingStripe] = useState(true);
  const [savingStripe, setSavingStripe] = useState(false);
  const [stripeError, setStripeError] = useState<string | null>(null);

  // Bundles state
  const [bundles, setBundles] = useState<BundleCatalogEntry[]>([]);
  const [priceForms, setPriceForms] = useState<Record<string, PriceFormState>>({});
  const [bundleError, setBundleError] = useState<string | null>(null);
  const [loadingBundles, setLoadingBundles] = useState(true);
  const [includeDemoPlaceholders, setIncludeDemoPlaceholders] = useState(false);

  // Price verification state
  const [priceChecks, setPriceChecks] = useState<Record<string, PriceVerificationResult>>({});

  // Pricing tab state
  const [pricingTab, setPricingTab] = useState<'month' | 'year' | 'other'>('month');

  /**
   * Load Stripe settings from API
   */
  const loadStripe = useCallback(async () => {
    setLoadingStripe(true);
    setStripeError(null);
    try {
      const data = await loadStripeSettings();
      setStripeSettings(data);
    } catch (error) {
      setStripeError(getApiErrorMessage(error, 'Failed to load Stripe settings'));
    } finally {
      setLoadingStripe(false);
    }
  }, []);

  /**
   * Load bundles from API
   */
  const loadBundles = useCallback(async () => {
    setLoadingBundles(true);
    setBundleError(null);
    try {
      const { bundles: payload } = await loadBundleCatalog();
      // Ensure at least one bundle exists for demo display when empty
      const bundlesWithDemo = ensureBundleForDemo(payload, includeDemoPlaceholders);
      // Then enrich with demo placeholder plans
      const enrichedBundles = enrichBundlesWithDemo(bundlesWithDemo, includeDemoPlaceholders);
      setBundles(enrichedBundles);
      setPriceForms(buildPriceFormsFromBundles(enrichedBundles));
      setPriceChecks({});
    } catch (error) {
      setBundleError(getApiErrorMessage(error, 'Failed to load bundle catalog'));
    } finally {
      setLoadingBundles(false);
    }
  }, [includeDemoPlaceholders]);

  // Initial load
  useEffect(() => {
    loadStripe();
    loadBundles();
  }, [loadStripe, loadBundles]);

  /**
   * Handle Stripe form input changes
   */
  const handleStripeInput = useCallback(
    (field: keyof StripeFormState) => (event: React.ChangeEvent<HTMLInputElement>) => {
      setStripeForm((prev) => ({ ...prev, [field]: event.target.value }));
    },
    []
  );

  /**
   * Save Stripe settings
   */
  const handleStripeSave = useCallback(async (event: React.FormEvent) => {
    event.preventDefault();
    setSavingStripe(true);
    setStripeError(null);
    try {
      const updated = await saveStripeSettings(stripeForm);
      setStripeSettings(updated);
      setStripeForm(DEFAULT_STRIPE_FORM);
    } catch (error) {
      setStripeError(getApiErrorMessage(error, 'Failed to update Stripe settings'));
    } finally {
      setSavingStripe(false);
    }
  }, [stripeForm]);

  /**
   * Handle price form field changes
   */
  const handlePriceChange = useCallback(
    (
      bundleKey: string,
      priceId: string,
      field: keyof PriceFormValues,
      transformer?: (value: string) => string | number
    ) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      const key = `${bundleKey}:${priceId}`;
      setPriceForms((prev) => {
        const current = prev[key];
        if (!current) return prev;

        const rawValue = field === 'highlight' || field === 'displayEnabled'
          ? (event.target as HTMLInputElement).checked
          : event.target.value;

        let nextValue: string | number | boolean = rawValue;
        if (typeof transformer === 'function') {
          nextValue = transformer(String(rawValue));
        }

        const nextValues: PriceFormValues = { ...current.values };
        if (field === 'displayWeight') {
          nextValues.displayWeight = Number(nextValue) || 0;
        } else if (field === 'displayEnabled') {
          nextValues.displayEnabled = Boolean(nextValue);
        } else if (field === 'highlight') {
          nextValues.highlight = Boolean(nextValue);
        } else if (field === 'featuresText') {
          nextValues.featuresText = String(nextValue);
        } else if (field === 'stripePriceId') {
          nextValues.stripePriceId = String(nextValue);
        } else {
          const mutable = nextValues as unknown as Record<string, unknown>;
          mutable[field] = nextValue;
        }

        return {
          ...prev,
          [key]: {
            ...current,
            values: nextValues,
            error: undefined,
          },
        };
      });

      if (field === 'stripePriceId') {
        setPriceChecks((prev) => {
          if (!prev[key]) return prev;
          const next = { ...prev };
          delete next[key];
          return next;
        });
      }
    },
    []
  );

  /**
   * Save a single price form
   */
  const handleSavePrice = useCallback(async (bundleKey: string, priceId: string) => {
    const key = `${bundleKey}:${priceId}`;
    const formState = priceForms[key];
    if (!formState || !isPriceFormDirty(formState)) return;

    if (formState.demo) {
      setPriceForms((prev) => ({
        ...prev,
        [key]: {
          ...formState,
          error: 'Demo plans cannot be saved. Connect Stripe billing to replace this slot.',
        },
      }));
      return;
    }

    const planName = formState.values.planName.trim();
    const stripePriceId = formState.values.stripePriceId.trim();
    if (!planName) {
      setPriceForms((prev) => ({
        ...prev,
        [key]: {
          ...formState,
          error: 'Plan name is required.',
        },
      }));
      return;
    }
    if (!stripePriceId) {
      setPriceForms((prev) => ({
        ...prev,
        [key]: {
          ...formState,
          error: 'Stripe price ID is required.',
        },
      }));
      return;
    }
    if (!stripePriceId.startsWith('price_')) {
      setPriceForms((prev) => ({
        ...prev,
        [key]: {
          ...formState,
          error: 'Stripe price IDs must start with "price_".',
        },
      }));
      return;
    }
    if (stripePriceId !== formState.original.stripePriceId.trim()) {
      const check = priceChecks[key];
      if (!check || check.status !== 'ok') {
        setPriceForms((prev) => ({
          ...prev,
          [key]: {
            ...formState,
            error: 'Verify the new Stripe price ID before saving changes.',
          },
        }));
        return;
      }
    }

    setPriceForms((prev) => ({
      ...prev,
      [key]: { ...formState, saving: true, error: undefined },
    }));

    try {
      await savePriceForm(bundleKey, priceId, formState);
      setPriceForms((prev) => {
        const existing = prev[key];
        if (!existing) return prev;
        return {
          ...prev,
          [key]: {
            ...existing,
            saving: false,
            original: { ...formState.values },
          },
        };
      });
      loadBundles();
    } catch (error) {
      setPriceForms((prev) => {
        const existing = prev[key];
        if (!existing) return prev;
        return {
          ...prev,
          [key]: {
            ...existing,
            saving: false,
            error: getApiErrorMessage(error, 'Failed to update price'),
          },
        };
      });
    }
  }, [priceForms, priceChecks, loadBundles]);

  /**
   * Verify a Stripe price ID
   */
  const handleVerifyPrice = useCallback(async (bundleKey: string, priceId: string) => {
    const key = `${bundleKey}:${priceId}`;
    const formState = priceForms[key];
    const value = formState?.values.stripePriceId.trim() || '';

    setPriceChecks((prev) => ({ ...prev, [key]: { status: 'checking' } }));

    const result = await verifyPriceId(value);
    setPriceChecks((prev) => ({ ...prev, [key]: result }));
  }, [priceForms]);

  /**
   * Remove a demo placeholder plan from the UI
   */
  const removeDemoPlan = useCallback((bundleKey: string, priceId: string) => {
    setBundles((prev) =>
      prev.map((entry) =>
        entry.bundle.bundle_key !== bundleKey
          ? entry
          : {
              ...entry,
              prices: entry.prices.filter((price) => {
                const identifier =
                  price.stripe_price_id ||
                  (price.metadata && (price.metadata as Record<string, unknown>).__price_pk?.toString());
                return identifier !== priceId;
              }),
            }
      )
    );
    setPriceForms((prev) => {
      const next = { ...prev };
      delete next[`${bundleKey}:${priceId}`];
      return next;
    });
  }, []);

  /**
   * Toggle demo placeholders visibility
   */
  const toggleDemoPlaceholders = useCallback(() => {
    setIncludeDemoPlaceholders((prev) => !prev);
  }, []);

  // Build computed values
  const stripeStatusBadges = buildStripeStatusBadges(stripeSettings);

  return {
    // Stripe state
    stripeSettings,
    stripeForm,
    loadingStripe,
    savingStripe,
    stripeError,
    stripeStatusBadges,
    handleStripeInput,
    handleStripeSave,
    loadStripe,

    // Bundles state
    bundles,
    priceForms,
    bundleError,
    loadingBundles,
    loadBundles,
    includeDemoPlaceholders,
    toggleDemoPlaceholders,

    // Price forms
    handlePriceChange,
    handleSavePrice,
    handleVerifyPrice,
    priceChecks,
    removeDemoPlan,

    // Tab state
    pricingTab,
    setPricingTab,
  };
}
