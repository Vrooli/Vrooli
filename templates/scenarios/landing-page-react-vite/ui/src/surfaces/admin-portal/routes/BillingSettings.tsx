import { useCallback, useEffect, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { create, clone } from '@bufbuild/protobuf';
import { PlanOptionSchema, PricingOverviewSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';
import {
  getStripeSettings,
  updateStripeSettings,
  getBundleCatalog,
  updateBundlePrice,
  BillingInterval,
  ConfigSource,
  jsonMapToRecord,
  recordToJsonMap,
  type GetStripeSettingsResponse,
  type StripeSettingsUpdate,
  type BundlePriceUpdate,
} from '../../../shared/api';
import type { BundleCatalogEntry, PlanOption, PricingOverview } from '../../../shared/api';
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

const asString = (value: unknown): string => (typeof value === 'string' ? value : '');

const BILLING_INTERVAL_LABELS: Record<BillingInterval, string> = {
  [BillingInterval.UNSPECIFIED]: 'unspecified',
  [BillingInterval.MONTH]: 'month',
  [BillingInterval.YEAR]: 'year',
  [BillingInterval.ONE_TIME]: 'one_time',
};

const billingIntervalLabel = (interval: BillingInterval): string => BILLING_INTERVAL_LABELS[interval] ?? 'unspecified';

const buildPriceValues = (metadata: Record<string, unknown>, defaults: { planName: string; displayWeight: number; displayEnabled: boolean }): PriceFormValues => {
  const features = Array.isArray(metadata.features)
    ? metadata.features.map((entry) => String(entry))
    : [];

  return {
    planName: defaults.planName,
    displayWeight: defaults.displayWeight,
    displayEnabled: defaults.displayEnabled,
    subtitle: asString(metadata.subtitle),
    badge: asString(metadata.badge),
    ctaLabel: asString(metadata.cta_label),
    highlight: Boolean(metadata.highlight),
    featuresText: features.join('\n'),
  };
};

const isDirty = (state: PriceFormState): boolean => {
  return JSON.stringify(state.original) !== JSON.stringify(state.values);
};

export function BillingSettings() {
  const [stripeSettings, setStripeSettings] = useState<GetStripeSettingsResponse | null>(null);
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
      const payload = await getBundleCatalog();
      const enrichedBundles = payload.map((entry) => injectDemoPlansForBundle(entry));
      setBundles(enrichedBundles);
      const nextForms: Record<string, PriceFormState> = {};
      enrichedBundles.forEach((entry) => {
        entry.prices.forEach((price) => {
          const key = `${entry.bundle?.bundleKey ?? ''}:${price.stripePriceId}`;
          const values = buildPriceValues(jsonMapToRecord(price.metadata), {
            planName: price.planName,
            displayWeight: price.displayWeight,
            displayEnabled: price.displayEnabled,
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
      const payload: StripeSettingsUpdate = {};
      (Object.keys(stripeForm) as (keyof StripeFormState)[]).forEach((key) => {
        const value = stripeForm[key].trim();
        if (value.length > 0) {
          payload[key] = value;
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

  const currentDashboardUrl = stripeSettings?.settings?.dashboardUrl;

  const renderStripeStatus = () => {
    if (!stripeSettings) {
      return null;
    }
    const snapshot = stripeSettings.snapshot;
    const badges = [
      {
        label: 'Publishable Key',
        ok: Boolean(snapshot?.publishableKeySet),
      },
      {
        label: 'Secret Key',
        ok: Boolean(snapshot?.secretKeySet),
      },
      {
        label: 'Webhook Secret',
        ok: Boolean(snapshot?.webhookSecretSet),
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
        // Remaining string fields: planName, subtitle, badge, ctaLabel.
        nextValues[field] = String(nextValue);
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
      const update: BundlePriceUpdate = {
        planName: formState.values.planName.trim() || undefined,
        displayWeight: formState.values.displayWeight,
        displayEnabled: formState.values.displayEnabled,
        subtitle: formState.values.subtitle.trim() || undefined,
        badge: formState.values.badge.trim() || undefined,
        ctaLabel: formState.values.ctaLabel.trim() || undefined,
        highlight: formState.values.highlight,
        features,
      };
      await updateBundlePrice(bundleKey, priceId, update);

      setPriceForms((prev) => {
        const existing = prev[key];
        if (!existing) return prev;
        return {
          ...prev,
          [key]: { ...existing, saving: false, original: { ...formState.values } },
        };
      });
    } catch (error) {
      setPriceForms((prev) => {
        const existing = prev[key];
        if (!existing) return prev;
        return {
          ...prev,
          [key]: {
            ...existing,
            saving: false,
            error: error instanceof Error ? error.message : 'Failed to update price',
          },
        };
      });
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
      <Card key={(entry.bundle?.bundleKey ?? '')} className="border-white/10 bg-slate-900/40">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>{entry.bundle?.name}</span>
            <span className="text-xs text-slate-400">Bundle key: {(entry.bundle?.bundleKey ?? '')}</span>
          </CardTitle>
          <CardDescription>
            Control which Stripe prices are surfaced on the landing page and customize their marketing copy.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-6 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
            <div className="space-y-6">
          {entry.prices.map((price) => {
            const key = `${(entry.bundle?.bundleKey ?? '')}:${price.stripePriceId}`;
            const formState = priceForms[key];
            if (!formState) {
              return null;
            }
            const dirty = isDirty(formState);
            const demoPlan = formState.demo;
            return (
              <div key={price.stripePriceId} className="rounded-xl border border-white/10 p-4">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{price.planName}</h3>
                    <p className="text-sm text-slate-400">{price.stripePriceId} · {billingIntervalLabel(price.billingInterval)}</p>
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
                          onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'displayEnabled')}
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
                          onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'planName')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Display Weight</label>
                        <input
                          type="number"
                          value={formState.values.displayWeight}
                          onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'displayWeight')}
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
                          onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'subtitle')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Badge</label>
                        <input
                          type="text"
                          value={formState.values.badge}
                          onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'badge')}
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
                          onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'ctaLabel')}
                          className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <label className="mt-6 flex items-center gap-2 text-sm text-slate-200">
                        <input
                          type="checkbox"
                          checked={formState.values.highlight}
                          onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'highlight')}
                          className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
                        />
                        Highlight tier (apply hero styling)
                      </label>
                    </div>

                    <div className="mt-4">
                      <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Feature Bullets</label>
                      <textarea
                        value={formState.values.featuresText}
                        onChange={handlePriceChange((entry.bundle?.bundleKey ?? ''), price.stripePriceId, 'featuresText')}
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
                    onClick={() => handleSavePrice((entry.bundle?.bundleKey ?? ''), price.stripePriceId)}
                    disabled={!dirty || formState.saving || demoPlan}
                    className="gap-2"
                  >
                    {formState.saving && <RefreshCw className="h-4 w-4 animate-spin" />}
                    {demoPlan ? 'Demo plan' : dirty ? 'Save changes' : 'Up to date'}
                  </Button>
                  {!price.displayEnabled && (
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
              <CardDescription>Store publishable/secret keys and link directly to the Stripe dashboard.</CardDescription>
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
                {stripeSettings?.snapshot && (
                  <p className="text-xs uppercase tracking-wide text-slate-500">
                    Source: {stripeSettings.snapshot.source === ConfigSource.DATABASE ? 'Admin override' : 'Environment variables'}
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
                      placeholder="pk_live_..."
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Secret Key</label>
                    <input
                      type="text"
                      value={stripeForm.secretKey}
                      onChange={handleStripeInput('secretKey')}
                      placeholder="sk_live_..."
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">Webhook Secret</label>
                    <input
                      type="text"
                      value={stripeForm.webhookSecret}
                      onChange={handleStripeInput('webhookSecret')}
                      placeholder="whsec_..."
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
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
  const enhancedPlans = entry.prices.map((price) => applyFormOverrides((entry.bundle?.bundleKey ?? ''), price, priceForms));
  const monthlyPlans = sortPlans(
    enhancedPlans.filter((plan) => plan.billingInterval === BillingInterval.MONTH && plan.displayEnabled),
  );
  const yearlyPlans = sortPlans(
    enhancedPlans.filter((plan) => plan.billingInterval === BillingInterval.YEAR && plan.displayEnabled),
  );

  const placeholderCount = monthlyPlans.filter((plan) => isDemoPlanOption(plan)).length;
  const monthlyCount = monthlyPlans.length - placeholderCount;

  return {
    overview: create(PricingOverviewSchema, {
      bundle: entry.bundle,
      monthly: monthlyPlans,
      yearly: yearlyPlans,
    }),
    monthlyCount,
    placeholderCount,
  };
}

function applyFormOverrides(bundleKey: string, price: PlanOption, priceForms: Record<string, PriceFormState>): PlanOption {
  const key = `${bundleKey}:${price.stripePriceId}`;
  const formState = priceForms[key];
  if (!formState) {
    return price;
  }

  const nextMetadata = jsonMapToRecord(price.metadata);

  const setOrDelete = (field: string, value: string) => {
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
  if (formState.values.highlight) {
    nextMetadata.highlight = true;
  } else {
    delete nextMetadata.highlight;
  }
  const features = parseFeaturesText(formState.values.featuresText);
  if (features.length > 0) {
    nextMetadata.features = features;
  } else {
    delete nextMetadata.features;
  }

  const planName = formState.values.planName.trim().length > 0 ? formState.values.planName.trim() : price.planName;

  const next = clone(PlanOptionSchema, price);
  next.planName = planName;
  next.displayWeight = formState.values.displayWeight;
  next.displayEnabled = formState.values.displayEnabled;
  next.metadata = recordToJsonMap(nextMetadata);
  return next;
}

function parseFeaturesText(raw: string): string[] {
  return raw
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function sortPlans(plans: PlanOption[]): PlanOption[] {
  return [...plans].sort((a, b) => {
    if (a.displayWeight === b.displayWeight) {
      return a.planRank - b.planRank;
    }
    return b.displayWeight - a.displayWeight;
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
