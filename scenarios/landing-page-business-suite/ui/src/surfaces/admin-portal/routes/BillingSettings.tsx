import { useCallback, useEffect, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import {
  getStripeSettings,
  updateStripeSettings,
  getBundleCatalog,
  updateBundlePrice,
  type StripeSettingsResponse,
} from '../../../shared/api';
import type { BundleCatalogEntry, PlanDisplayMetadata, PlanOption, PricingOverview } from '../../../shared/api';
import { MetricsModeProvider } from '../../../shared/hooks/useMetrics';
import { PricingSection } from '../../public-landing/sections/PricingSection';
import { AlertTriangle, CreditCard, RefreshCw, ShieldCheck } from 'lucide-react';
import { injectDemoPlansForBundle, isDemoPlanOption } from '../../../shared/lib/pricingPlaceholders';

interface StripeFormState {
  publishableKey: string;
  secretKey: string;
  webhookSecret: string;
  dashboardUrl: string;
}

interface PriceFormValues {
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

const buildPriceValues = (metadata: PlanDisplayMetadata | undefined, defaults: { planName: string; displayWeight: number; displayEnabled: boolean }): PriceFormValues => {
  const features = Array.isArray(metadata?.features)
    ? (metadata?.features as string[]).map((entry) => String(entry))
    : [];

  return {
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
      const enrichedBundles = payload.map((entry) => injectDemoPlansForBundle(entry));
      setBundles(enrichedBundles);
      const nextForms: Record<string, PriceFormState> = {};
      enrichedBundles.forEach((entry) => {
        entry.prices.forEach((price) => {
          const key = `${entry.bundle.bundle_key}:${price.stripe_price_id}`;
          const values = buildPriceValues(price.metadata, {
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
    } catch (error) {
      setBundleError(error instanceof Error ? error.message : 'Failed to load bundle catalog');
    } finally {
      setLoadingBundles(false);
    }
  }, []);

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

    return bundles.map((entry) => (
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
          <div className="grid gap-6 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
            <div className="space-y-6">
          {entry.prices.map((price) => {
            const key = `${entry.bundle.bundle_key}:${price.stripe_price_id}`;
            const formState = priceForms[key];
            if (!formState) {
              return null;
            }
            const dirty = isDirty(formState);
            const demoPlan = formState.demo;
            return (
              <div key={price.stripe_price_id} className="rounded-xl border border-white/10 p-4">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{price.plan_name}</h3>
                    <p className="text-sm text-slate-400">{price.stripe_price_id} · {price.billing_interval}</p>
                    {demoPlan && (
                      <p className="text-xs font-semibold uppercase tracking-wide text-amber-300">
                        Demo placeholder · connect Stripe to replace this slot
                      </p>
                    )}
                  </div>
                  <label className="flex items-center gap-2 text-sm text-slate-200">
                    <input
                      type="checkbox"
                      checked={formState.values.displayEnabled}
                          onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'displayEnabled')}
                          className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
                        />
                        Visible on landing page
                      </label>
                    </div>

                    <div className="mt-4 grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Plan Name</label>
                        <input
                          type="text"
                          value={formState.values.planName}
                          onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'planName')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Display Weight</label>
                        <input
                          type="number"
                          value={formState.values.displayWeight}
                          onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'displayWeight')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                    </div>

                    <div className="mt-4 grid gap-4 md:grid-cols-2">
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Subtitle</label>
                        <input
                          type="text"
                          value={formState.values.subtitle}
                          onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'subtitle')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Badge</label>
                        <input
                          type="text"
                          value={formState.values.badge}
                          onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'badge')}
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
                          onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'ctaLabel')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <label className="mt-6 flex items-center gap-2 text-sm text-slate-200">
                        <input
                          type="checkbox"
                          checked={formState.values.highlight}
                          onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'highlight')}
                          className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
                        />
                        Highlight tier (apply hero styling)
                      </label>
                    </div>

                    <div className="mt-4">
                      <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Feature Bullets</label>
                      <textarea
                        value={formState.values.featuresText}
                        onChange={handlePriceChange(entry.bundle.bundle_key, price.stripe_price_id, 'featuresText')}
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
                    onClick={() => handleSavePrice(entry.bundle.bundle_key, price.stripe_price_id)}
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
            <PlanPreview data={buildPricingPreviewData(entry, priceForms)} />
          </div>
        </CardContent>
      </Card>
    ));
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
            <Button variant="ghost" size="sm" onClick={loadBundles} className="gap-2">
              <RefreshCw className="h-4 w-4" /> Reload catalog
            </Button>
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

function buildPricingPreviewData(entry: BundleCatalogEntry, priceForms: Record<string, PriceFormState>): PricingPreviewData {
  const enhancedPlans = entry.prices.map((price) => applyFormOverrides(entry.bundle.bundle_key, price, priceForms));
  const monthlyPlans = sortPlans(
    enhancedPlans.filter((plan) => plan.billing_interval === 'month' && plan.display_enabled),
  );
  const yearlyPlans = sortPlans(
    enhancedPlans.filter((plan) => plan.billing_interval === 'year' && plan.display_enabled),
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
  const key = `${bundleKey}:${price.stripe_price_id}`;
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
