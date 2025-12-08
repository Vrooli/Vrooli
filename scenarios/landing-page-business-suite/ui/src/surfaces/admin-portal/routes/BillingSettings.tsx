import { useCallback, useEffect, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import {
  getStripeSettings,
  updateStripeSettings,
  getBundleCatalog,
  updateBundlePrice,
  verifyStripePrice,
  type StripeSettingsResponse,
} from '../../../shared/api';
import type { BundleCatalogEntry, PlanDisplayMetadata, PlanOption, PricingOverview } from '../../../shared/api';
import { MetricsModeProvider } from '../../../shared/hooks/useMetrics';
import { PricingSection } from '../../public-landing/sections/PricingSection';
import { AlertTriangle, CreditCard, RefreshCw, ShieldCheck } from 'lucide-react';
import { injectDemoPlansForBundle, isDemoPlanOption } from '../../../shared/lib/pricingPlaceholders';
import { cn } from '../../../shared/lib/utils';

interface StripeFormState {
  publishableKey: string;
  secretKey: string;
  webhookSecret: string;
  dashboardUrl: string;
}

interface PriceFormValues {
  stripePriceId: string;
  planName: string;
  displayWeight: number;
  displayEnabled: boolean;
  subtitle: string;
  badge: string;
  ctaLabel: string;
  highlight: boolean;
  featuresText: string;
}

interface PriceFormState {
  values: PriceFormValues;
  original: PriceFormValues;
  saving: boolean;
  error?: string;
  demo?: boolean;
}

interface PriceCheck {
  status: 'idle' | 'checking' | 'ok' | 'error';
  message?: string;
}

const defaultStripeForm: StripeFormState = {
  publishableKey: '',
  secretKey: '',
  webhookSecret: '',
  dashboardUrl: '',
};

const PRICING_PREVIEW_CONTENT = {
  title: 'Landing pricing preview',
  subtitle: 'Updates instantly with unsaved copy changes so you can validate the three-card layout visitors will see.',
};

const buildPriceValues = (
  metadata: PlanDisplayMetadata | undefined,
  defaults: { planName: string; displayWeight: number; displayEnabled: boolean; priceId: string },
): PriceFormValues => {
  const features = Array.isArray(metadata?.features)
    ? (metadata?.features as string[]).map((entry) => String(entry))
    : [];

  return {
    stripePriceId: defaults.priceId,
    planName: defaults.planName,
    displayWeight: defaults.displayWeight,
    displayEnabled: defaults.displayEnabled,
    subtitle: (metadata?.subtitle as string) || '',
    badge: (metadata?.badge as string) || '',
    ctaLabel: (metadata?.cta_label as string) || '',
    highlight: Boolean(metadata?.highlight),
    featuresText: features.join('\n'),
  };
};

const isDirty = (state: PriceFormState): boolean => {
  return JSON.stringify(state.original) !== JSON.stringify(state.values);
};

export function BillingSettings() {
  const [stripeSettings, setStripeSettings] = useState<StripeSettingsResponse | null>(null);
  const [stripeForm, setStripeForm] = useState<StripeFormState>(defaultStripeForm);
  const [loadingStripe, setLoadingStripe] = useState(true);
  const [savingStripe, setSavingStripe] = useState(false);
  const [stripeError, setStripeError] = useState<string | null>(null);

  const [bundles, setBundles] = useState<BundleCatalogEntry[]>([]);
  const [priceForms, setPriceForms] = useState<Record<string, PriceFormState>>({});
  const [bundleError, setBundleError] = useState<string | null>(null);
  const [loadingBundles, setLoadingBundles] = useState(true);
  const [includeDemoPlaceholders, setIncludeDemoPlaceholders] = useState(false);
  const [pricingTab, setPricingTab] = useState<'month' | 'year' | 'other'>('month');
  const [priceChecks, setPriceChecks] = useState<Record<string, PriceCheck>>({});

  const loadStripe = useCallback(async () => {
    setLoadingStripe(true);
    setStripeError(null);
    try {
      const data = await getStripeSettings();
      setStripeSettings(data);
    } catch (error) {
      setStripeError(error instanceof Error ? error.message : 'Failed to load Stripe settings');
    } finally {
      setLoadingStripe(false);
    }
  }, []);

  const loadBundles = useCallback(async () => {
    setLoadingBundles(true);
    setBundleError(null);
    try {
      const { bundles: payload } = await getBundleCatalog();
      const enrichedBundles = payload.map((entry) =>
        includeDemoPlaceholders ? injectDemoPlansForBundle(entry) : entry,
      );
      setBundles(enrichedBundles);
      const nextForms: Record<string, PriceFormState> = {};
      enrichedBundles.forEach((entry) => {
        entry.prices.forEach((price) => {
          const priceIdentifier =
            price.stripe_price_id ||
            (price.metadata && (price.metadata as Record<string, unknown>).__price_pk?.toString()) ||
            price.plan_name;
          const key = `${entry.bundle.bundle_key}:${priceIdentifier}`;
          const values = buildPriceValues(price.metadata, {
            priceId: price.stripe_price_id,
            planName: price.plan_name,
            displayWeight: price.display_weight,
            displayEnabled: price.display_enabled,
          });
          nextForms[key] = {
            values,
            original: { ...values },
            saving: false,
            demo: isDemoPlanOption(price),
          };
        });
      });
      setPriceForms(nextForms);
      // Reset price check statuses when bundles reload
      setPriceChecks({});
    } catch (error) {
      setBundleError(error instanceof Error ? error.message : 'Failed to load bundle catalog');
    } finally {
      setLoadingBundles(false);
    }
  }, [includeDemoPlaceholders]);

  useEffect(() => {
    loadStripe();
    loadBundles();
  }, [loadStripe, loadBundles]);

  const handleStripeInput = (field: keyof StripeFormState) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setStripeForm((prev) => ({ ...prev, [field]: event.target.value }));
  };

  const handleStripeSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSavingStripe(true);
    setStripeError(null);
    try {
      const payload: Record<string, string> = {};
      (Object.keys(stripeForm) as (keyof StripeFormState)[]).forEach((key) => {
        const value = stripeForm[key].trim();
        if (value.length > 0) {
          const apiKey = key === 'dashboardUrl' ? 'dashboard_url' : key.replace(/[A-Z]/g, (match) => `_${match.toLowerCase()}`);
          payload[apiKey] = value;
        }
      });

      if (Object.keys(payload).length === 0) {
        setStripeError('Enter at least one field before saving.');
        setSavingStripe(false);
        return;
      }

      const updated = await updateStripeSettings(payload);
      setStripeSettings(updated);
      setStripeForm(defaultStripeForm);
    } catch (error) {
      setStripeError(error instanceof Error ? error.message : 'Failed to update Stripe settings');
    } finally {
      setSavingStripe(false);
    }
  };

  const currentDashboardUrl = stripeSettings?.dashboard_url;
  const publishablePreview = stripeSettings?.publishable_key_preview;

  const renderStripeStatus = () => {
    if (!stripeSettings) {
      return null;
    }
    const publishableSet = stripeSettings.publishable_key_set || Boolean(stripeSettings.publishable_key_preview);
    const secretSet = stripeSettings.secret_key_set;
    const webhookSet = stripeSettings.webhook_secret_set;
    const badges = [
      {
        label: 'Publishable Key',
        ok: publishableSet,
      },
      {
        label: 'Restricted Key',
        ok: secretSet,
      },
      {
        label: 'Webhook Secret',
        ok: webhookSet,
      },
    ];

    return (
      <div className="flex flex-wrap gap-3">
        {badges.map((badge) => (
          <span
            key={badge.label}
            className={
              'inline-flex items-center gap-2 rounded-full px-3 py-1 text-sm ' +
              (badge.ok ? 'bg-emerald-500/10 text-emerald-200' : 'bg-rose-500/10 text-rose-200')
            }
          >
            {badge.ok ? <ShieldCheck className="h-4 w-4" /> : <AlertTriangle className="h-4 w-4" />}
            {badge.label}
          </span>
        ))}
      </div>
    );
  };

  const handlePriceChange = (
    bundleKey: string,
    priceId: string,
    field: keyof PriceFormValues,
    transformer?: (value: string) => string | number,
  ) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const key = `${bundleKey}:${priceId}`;
    setPriceForms((prev) => {
      const current = prev[key];
      if (!current) {
        return prev;
      }
      const rawValue = field === 'highlight' || field === 'displayEnabled' ? (event.target as HTMLInputElement).checked : event.target.value;
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
        (nextValues as Record<string, unknown>)[field] = nextValue;
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
  };

  const handleSavePrice = async (bundleKey: string, priceId: string) => {
    const key = `${bundleKey}:${priceId}`;
    const formState = priceForms[key];
    if (!formState || !isDirty(formState)) {
      return;
    }

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

    const features = formState.values.featuresText
      .split('\n')
      .map((entry) => entry.trim())
      .filter(Boolean);

    const stripePriceId = formState.values.stripePriceId.trim();

    setPriceForms((prev) => ({
      ...prev,
      [key]: {
        ...formState,
        saving: true,
        error: undefined,
      },
    }));

    try {
      await updateBundlePrice(bundleKey, priceId, {
        stripe_price_id: stripePriceId === '' ? '' : stripePriceId,
        plan_name: formState.values.planName.trim() || undefined,
        display_weight: formState.values.displayWeight,
        display_enabled: formState.values.displayEnabled,
        subtitle: formState.values.subtitle.trim() || undefined,
        badge: formState.values.badge.trim() || undefined,
        cta_label: formState.values.ctaLabel.trim() || undefined,
        highlight: formState.values.highlight,
        features,
      });

      setPriceForms((prev) => ({
        ...prev,
        [key]: {
          ...prev[key],
          saving: false,
          original: { ...formState.values },
        },
      }));
      loadBundles();
    } catch (error) {
      setPriceForms((prev) => ({
        ...prev,
        [key]: {
          ...prev[key],
          saving: false,
          error: error instanceof Error ? error.message : 'Failed to update price',
        },
      }));
    }
  };

  const handleVerifyPrice = async (bundleKey: string, priceId: string) => {
    const key = `${bundleKey}:${priceId}`;
    const formState = priceForms[key];
    const value = formState?.values.stripePriceId.trim() || '';
    if (!value) {
      setPriceChecks((prev) => ({
        ...prev,
        [key]: { status: 'error', message: 'Enter a Stripe price ID or lookup key' },
      }));
      return;
    }
    setPriceChecks((prev) => ({ ...prev, [key]: { status: 'checking' } }));
    try {
      const info = await verifyStripePrice(value);
      const parts = [
        info.id ? `ID ${info.id}` : null,
        info.lookup_key ? `lookup ${info.lookup_key}` : null,
        info.interval ? info.interval : null,
        info.currency ? info.currency.toUpperCase() : null,
        info.active === false ? 'inactive' : null,
      ].filter(Boolean);
      setPriceChecks((prev) => ({
        ...prev,
        [key]: {
          status: 'ok',
          message: parts.length > 0 ? parts.join(' · ') : 'Verified',
        },
      }));
    } catch (error) {
      setPriceChecks((prev) => ({
        ...prev,
        [key]: {
          status: 'error',
          message: error instanceof Error ? error.message : 'Verification failed',
        },
      }));
    }
  };

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
            },
      ),
    );
    setPriceForms((prev) => {
      const next = { ...prev };
      delete next[`${bundleKey}:${priceId}`];
      return next;
    });
  }, []);

  const renderBundleCards = () => {
    if (loadingBundles) {
      return <p className="text-slate-400">Loading bundle catalog…</p>;
    }

    if (bundleError) {
      return (
        <div className="rounded-xl border border-rose-500/40 bg-rose-500/5 p-4 text-rose-200">
          {bundleError}
        </div>
      );
    }

    return bundles.map((entry) => {
      const visiblePrices = entry.prices.filter((price) => {
        const interval = normalizeInterval(price.billing_interval);
        if (!includeDemoPlaceholders && isDemoPlanOption(price)) {
          return false;
        }
        if (pricingTab === 'month') return interval === 'month';
        if (pricingTab === 'year') return interval === 'year';
        return interval === 'one_time' || interval === 'other';
      });
      const demoHidden = entry.prices.some(isDemoPlanOption) && !includeDemoPlaceholders;
      return (
      <Card key={entry.bundle.bundle_key} className="border-white/10 bg-slate-900/40">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>{entry.bundle.name}</span>
            <span className="text-xs text-slate-400">Bundle key: {entry.bundle.bundle_key}</span>
          </CardTitle>
          <CardDescription>
            Control which Stripe prices are surfaced on the landing page and customize their marketing copy.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {demoHidden && (
            <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-amber-200">
              Demo placeholders hidden. Turn them back on to see filler tiers until you add real Stripe prices.
            </div>
          )}
          <div className="grid gap-6 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
            <div className="space-y-6">
          {visiblePrices.map((price) => {
            const priceIdentifier =
              price.stripe_price_id ||
              (price.metadata && (price.metadata as Record<string, unknown>).__price_pk?.toString()) ||
              price.plan_name;
            const key = `${entry.bundle.bundle_key}:${priceIdentifier}`;
            const formState = priceForms[key];
            if (!formState) {
              return null;
            }
            const dirty = isDirty(formState);
            const demoPlan = formState.demo;
            return (
              <div key={key} className="rounded-xl border border-white/10 p-4">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{price.plan_name}</h3>
                    <div className="flex flex-wrap items-center gap-2 text-sm text-slate-400">
                      <span>{intervalLabel(normalizeInterval(price.billing_interval))} · {price.currency.toUpperCase()}</span>
                      <span
                        className={cn(
                          'inline-flex items-center gap-2 rounded-full border px-2 py-0.5 text-xs',
                          demoPlan
                            ? 'border-amber-500/50 bg-amber-500/10 text-amber-200'
                            : 'border-emerald-500/40 bg-emerald-500/10 text-emerald-100'
                        )}
                      >
                        {demoPlan ? 'Demo placeholder (not saved)' : `Stripe price: ${price.stripe_price_id || 'None (free/CTA)'}`}
                      </span>
                    </div>
                  </div>
                  {demoPlan && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="gap-2 text-amber-200 hover:text-amber-100"
                      onClick={() => removeDemoPlan(entry.bundle.bundle_key, priceIdentifier)}
                    >
                      Remove demo placeholder
                    </Button>
                  )}
                  <label className="flex items-center gap-2 text-sm text-slate-200">
                    <input
                      type="checkbox"
                      checked={formState.values.displayEnabled}
                          onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'displayEnabled')}
                          className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
                        />
                        Visible on landing page
                      </label>
                    </div>

                    <div className="mt-4 grid gap-4 md:grid-cols-3">
                      <div className="md:col-span-2">
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Plan Name</label>
                        <input
                          type="text"
                          value={formState.values.planName}
                          onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'planName')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Display Weight</label>
                        <input
                          type="number"
                          value={formState.values.displayWeight}
                          onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'displayWeight')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                    </div>

                    <div className="mt-4">
                      <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Stripe Price ID</label>
                      <input
                        type="text"
                        value={formState.values.stripePriceId}
                        onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'stripePriceId')}
                        placeholder="price_abc123 or lookup key if using Stripe aliases"
                        className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      />
                      <div className="mt-2 flex items-center gap-2 text-xs">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="border border-white/10 bg-white/5 text-white"
                          onClick={() => handleVerifyPrice(entry.bundle.bundle_key, priceIdentifier)}
                        >
                          Verify
                        </Button>
                        {priceChecks[`${entry.bundle.bundle_key}:${priceIdentifier}`]?.status === 'checking' && (
                          <span className="text-slate-300">Checking…</span>
                        )}
                        {priceChecks[`${entry.bundle.bundle_key}:${priceIdentifier}`]?.status === 'ok' && (
                          <span className="text-emerald-300">
                            {priceChecks[`${entry.bundle.bundle_key}:${priceIdentifier}`]?.message || 'Verified'}
                          </span>
                        )}
                        {priceChecks[`${entry.bundle.bundle_key}:${priceIdentifier}`]?.status === 'error' && (
                          <span className="text-amber-200">
                            {priceChecks[`${entry.bundle.bundle_key}:${priceIdentifier}`]?.message || 'Verification failed'}
                          </span>
                        )}
                      </div>
                      <p className="mt-1 text-xs text-slate-400">Paste the actual Stripe price ID or leave blank for free/CTA-only tiers.</p>
                    </div>

                    <div className="mt-4 grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Subtitle</label>
                        <input
                          type="text"
                          value={formState.values.subtitle}
                          onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'subtitle')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Badge</label>
                        <input
                          type="text"
                          value={formState.values.badge}
                          onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'badge')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                    </div>

                    <div className="mt-4 grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">CTA Label</label>
                        <input
                          type="text"
                          value={formState.values.ctaLabel}
                          onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'ctaLabel')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <label className="mt-6 flex items-center gap-2 text-sm text-slate-200">
                        <input
                          type="checkbox"
                          checked={formState.values.highlight}
                          onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'highlight')}
                          className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
                        />
                        Highlight tier (apply hero styling)
                      </label>
                    </div>

                    <div className="mt-4">
                      <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Feature Bullets</label>
                      <textarea
                        value={formState.values.featuresText}
                        onChange={handlePriceChange(entry.bundle.bundle_key, priceIdentifier, 'featuresText')}
                        rows={4}
                        className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        placeholder={'One feature per line\nDesktop downloads included\nWhite-glove onboarding'}
                      />
                    </div>

                {formState.error && (
                  <p className="mt-3 text-sm text-rose-300">{formState.error}</p>
                )}

                <div className="mt-4 flex items-center gap-3">
                  <Button
                    type="button"
                    onClick={() => handleSavePrice(entry.bundle.bundle_key, priceIdentifier)}
                    disabled={!dirty || formState.saving || demoPlan}
                    className="gap-2"
                  >
                    {formState.saving && <RefreshCw className="h-4 w-4 animate-spin" />}
                    {demoPlan ? 'Demo plan' : dirty ? 'Save changes' : 'Up to date'}
                  </Button>
                  {!price.display_enabled && (
                    <span className="text-xs text-slate-400">Hidden from landing page visitors</span>
                  )}
                  {demoPlan && (
                    <span className="text-xs text-amber-300">Connect Stripe & reload to edit this slot.</span>
                  )}
                </div>
              </div>
            );
          })}
            </div>
            <PlanPreview data={buildPricingPreviewData(entry, priceForms, includeDemoPlaceholders)} />
          </div>
        </CardContent>
      </Card>
    );
    });
  };

  return (
    <AdminLayout>
      <div className="space-y-10">
        <Card className="border-white/10 bg-slate-900/60">
          <CardHeader className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <CreditCard className="h-5 w-5 text-blue-300" /> Stripe Configuration
              </CardTitle>
              <CardDescription>Store publishable/restricted keys and link directly to the Stripe dashboard.</CardDescription>
            </div>
            {currentDashboardUrl && (
              <Button
                variant="ghost"
                size="sm"
                className="gap-2"
                onClick={() => window.open(currentDashboardUrl, '_blank', 'noopener,noreferrer')}
              >
                Open Stripe Dashboard
              </Button>
            )}
          </CardHeader>
          <CardContent className="space-y-6">
            {loadingStripe ? (
              <p className="text-slate-400">Loading Stripe settings…</p>
            ) : (
              <>
                {renderStripeStatus()}
                {stripeSettings?.source && (
                  <p className="text-xs uppercase tracking-wide text-slate-500">
                    Source: {stripeSettings.source === 'database' ? 'Admin override' : 'Environment variables'}
                  </p>
                )}
                {stripeError && <p className="text-sm text-rose-300">{stripeError}</p>}
                <form onSubmit={handleStripeSubmit} className="grid gap-4 md:grid-cols-2">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Publishable Key</label>
                    <input
                      type="text"
                      value={stripeForm.publishableKey}
                      onChange={handleStripeInput('publishableKey')}
                      placeholder={publishablePreview ? `${publishablePreview} (saved)` : 'pk_live_...'}
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                    {publishablePreview && (
                      <p className="mt-1 text-xs text-slate-400">Saved (preview): {publishablePreview}</p>
                    )}
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Restricted Key (secret)</label>
                    <input
                      type="text"
                      value={stripeForm.secretKey}
                      onChange={handleStripeInput('secretKey')}
                      placeholder={stripeSettings?.secret_key_set ? 'Saved restricted key (not shown)' : 'rk_live_...'}
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                    {stripeSettings?.secret_key_set && (
                      <p className="mt-1 text-xs text-slate-400">Restricted key is saved. Enter a new value to rotate.</p>
                    )}
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Webhook Secret</label>
                    <input
                      type="text"
                      value={stripeForm.webhookSecret}
                      onChange={handleStripeInput('webhookSecret')}
                      placeholder={stripeSettings?.webhook_secret_set ? 'Saved webhook secret (not shown)' : 'whsec_...'}
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                    {stripeSettings?.webhook_secret_set && (
                      <p className="mt-1 text-xs text-slate-400">Webhook secret is saved. Enter a new value to rotate.</p>
                    )}
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Dashboard URL</label>
                    <input
                      type="url"
                      value={stripeForm.dashboardUrl}
                      onChange={handleStripeInput('dashboardUrl')}
                      placeholder="https://dashboard.stripe.com/..."
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                  </div>
                  <div className="md:col-span-2 flex items-center gap-3">
                    <Button type="submit" className="gap-2" disabled={savingStripe}>
                      {savingStripe && <RefreshCw className="h-4 w-4 animate-spin" />}
                      Save Stripe Settings
                    </Button>
                    <Button type="button" variant="ghost" size="sm" onClick={loadStripe} className="gap-2">
                      <RefreshCw className="h-4 w-4" />
                      Refresh
                    </Button>
                  </div>
                </form>
              </>
            )}
          </CardContent>
        </Card>

        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-semibold text-white">Plan Display Manager</h2>
            <div className="flex items-center gap-2">
              <div className="flex overflow-hidden rounded-lg border border-white/10">
                {(['month', 'year', 'other'] as const).map((tab) => (
                  <button
                    key={tab}
                    className={cn(
                      'px-3 py-1 text-sm transition-colors',
                      pricingTab === tab
                        ? 'bg-white/10 text-white'
                        : 'bg-transparent text-slate-300 hover:bg-white/5'
                    )}
                    onClick={() => setPricingTab(tab)}
                  >
                    {tab === 'month' ? 'Monthly' : tab === 'year' ? 'Yearly' : 'Other'}
                  </button>
                ))}
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setIncludeDemoPlaceholders((prev) => !prev)}
                className="gap-2"
              >
                {includeDemoPlaceholders ? 'Hide demo placeholders' : 'Show demo placeholders'}
              </Button>
              <Button variant="ghost" size="sm" onClick={loadBundles} className="gap-2">
                <RefreshCw className="h-4 w-4" /> Reload catalog
              </Button>
            </div>
          </div>
          <div className="space-y-6">{renderBundleCards()}</div>
        </div>
      </div>
    </AdminLayout>
  );
}

interface PricingPreviewData {
  overview: PricingOverview;
  monthlyCount: number;
  placeholderCount: number;
}

type IntervalSlug = 'month' | 'year' | 'one_time' | 'other';

const normalizeInterval = (value: PlanOption['billing_interval'] | string | number | null | undefined): IntervalSlug => {
  if (typeof value === 'number') {
    if (value === 1) return 'month';
    if (value === 2) return 'year';
    if (value === 3) return 'one_time';
  }
  const raw = String(value ?? '').toLowerCase();
  if (raw.includes('month')) return 'month';
  if (raw.includes('year')) return 'year';
  if (raw.includes('one_time') || raw.includes('one-time') || raw.includes('onetime')) return 'one_time';
  return 'other';
};

const intervalLabel = (slug: IntervalSlug) => {
  switch (slug) {
    case 'month':
      return 'Monthly';
    case 'year':
      return 'Yearly';
    case 'one_time':
      return 'One-time';
    default:
      return 'Other';
  }
};

function buildPricingPreviewData(entry: BundleCatalogEntry, priceForms: Record<string, PriceFormState>, includeDemo: boolean): PricingPreviewData {
  const enhancedPlans = entry.prices
    .filter((plan) => includeDemo || !isDemoPlanOption(plan))
    .map((price) => applyFormOverrides(entry.bundle.bundle_key, price, priceForms));
  const monthlyPlans = sortPlans(
    enhancedPlans.filter((plan) => normalizeInterval(plan.billing_interval) === 'month' && plan.display_enabled),
  );
  const yearlyPlans = sortPlans(
    enhancedPlans.filter((plan) => normalizeInterval(plan.billing_interval) === 'year' && plan.display_enabled),
  );

  const placeholderCount = monthlyPlans.filter((plan) => isDemoPlanOption(plan)).length;
  const monthlyCount = monthlyPlans.length - placeholderCount;

  return {
    overview: {
      bundle: entry.bundle,
      monthly: monthlyPlans,
      yearly: yearlyPlans,
      updated_at: new Date().toISOString(),
    },
    monthlyCount,
    placeholderCount,
  };
}

function applyFormOverrides(bundleKey: string, price: PlanOption, priceForms: Record<string, PriceFormState>): PlanOption {
  const priceIdentifier =
    price.stripe_price_id ||
    (price.metadata && (price.metadata as Record<string, unknown>).__price_pk?.toString()) ||
    price.plan_name;
  const key = `${bundleKey}:${priceIdentifier}`;
  const formState = priceForms[key];
  if (!formState) {
    return { ...price };
  }

  const nextMetadata: PlanDisplayMetadata = {
    ...(price.metadata ?? {}),
  };

  const setOrDelete = (field: keyof PlanDisplayMetadata, value: string) => {
    const trimmed = value.trim();
    if (trimmed.length > 0) {
      nextMetadata[field] = trimmed;
    } else {
      delete nextMetadata[field];
    }
  };

  setOrDelete('subtitle', formState.values.subtitle);
  setOrDelete('badge', formState.values.badge);
  setOrDelete('cta_label', formState.values.ctaLabel);
  nextMetadata.highlight = formState.values.highlight || undefined;
  const features = parseFeaturesText(formState.values.featuresText);
  if (features.length > 0) {
    nextMetadata.features = features;
  } else {
    delete nextMetadata.features;
  }

  const metadata = Object.keys(nextMetadata).length > 0 ? nextMetadata : undefined;
  const planName = formState.values.planName.trim().length > 0 ? formState.values.planName.trim() : price.plan_name;

  return {
    ...price,
    plan_name: planName,
    display_weight: formState.values.displayWeight,
    display_enabled: formState.values.displayEnabled,
    metadata,
  };
}

function parseFeaturesText(raw: string): string[] {
  return raw
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function sortPlans(plans: PlanOption[]): PlanOption[] {
  return [...plans].sort((a, b) => {
    if (a.display_weight === b.display_weight) {
      const aRank = typeof a.plan_rank === 'number' ? a.plan_rank : Number.MAX_SAFE_INTEGER;
      const bRank = typeof b.plan_rank === 'number' ? b.plan_rank : Number.MAX_SAFE_INTEGER;
      return aRank - bRank;
    }
    return b.display_weight - a.display_weight;
  });
}

function PlanPreview({ data }: { data: PricingPreviewData }) {
  let statusMessage = 'Showing live preview of enabled monthly plans.';
  if (data.monthlyCount === 0 && data.placeholderCount > 0) {
    statusMessage = 'No saved monthly plans yet — displaying demo placeholders so the layout stays complete.';
  } else if (data.placeholderCount > 0) {
    statusMessage = `Showing ${data.monthlyCount} saved plan${data.monthlyCount === 1 ? '' : 's'} plus ${data.placeholderCount} demo placeholder${data.placeholderCount === 1 ? '' : 's'} to fill the preview.`;
  } else if (data.monthlyCount > 0) {
    statusMessage = `Showing ${data.monthlyCount} saved monthly plan${data.monthlyCount === 1 ? '' : 's'}.`;
  }

  return (
    <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-white">Live Pricing Preview</p>
          <p className="text-xs text-slate-400">{statusMessage}</p>
        </div>
      </div>
      <MetricsModeProvider mode="preview">
        <div className="relative mt-4">
          <div className="max-h-[640px] overflow-y-auto rounded-[28px] border border-white/10 bg-[#030712] p-1" onClickCapture={(event) => {
            event.preventDefault();
            event.stopPropagation();
          }} onKeyDownCapture={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              event.stopPropagation();
            }
          }}>
            <div className="pointer-events-none">
              <PricingSection content={PRICING_PREVIEW_CONTENT} pricingOverview={data.overview} />
            </div>
          </div>
        </div>
      </MetricsModeProvider>
    </div>
  );
}
