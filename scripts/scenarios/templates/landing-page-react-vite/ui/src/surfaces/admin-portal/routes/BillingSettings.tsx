import { useCallback, useEffect, useMemo, useState } from 'react';
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
import type { BundleCatalogEntry, PlanDisplayMetadata } from '../../../shared/api';
import { AlertTriangle, CreditCard, RefreshCw, ShieldCheck } from 'lucide-react';

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
}

const defaultStripeForm: StripeFormState = {
  publishableKey: '',
  secretKey: '',
  webhookSecret: '',
  dashboardUrl: '',
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
      setBundles(payload);
      const nextForms: Record<string, PriceFormState> = {};
      payload.forEach((entry) => {
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

  const renderStripeStatus = () => {
    if (!stripeSettings) {
      return null;
    }
    const badges = [
      {
        label: 'Publishable Key',
        ok: stripeSettings.publishable_key_set,
      },
      {
        label: 'Secret Key',
        ok: stripeSettings.secret_key_set,
      },
      {
        label: 'Webhook Secret',
        ok: stripeSettings.webhook_secret_set,
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

  const bundlePriceKeys = useMemo(() => Object.keys(priceForms), [priceForms]);

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
          {entry.prices.map((price) => {
            const key = `${entry.bundle.bundle_key}:${price.stripe_price_id}`;
            const formState = priceForms[key];
            if (!formState) {
              return null;
            }
            const dirty = isDirty(formState);
            return (
              <div key={price.stripe_price_id} className="rounded-xl border border-white/10 p-4">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{price.plan_name}</h3>
                    <p className="text-sm text-slate-400">{price.stripe_price_id} · {price.billing_interval}</p>
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
                    disabled={!dirty || formState.saving}
                    className="gap-2"
                  >
                    {formState.saving && <RefreshCw className="h-4 w-4 animate-spin" />}
                    {dirty ? 'Save changes' : 'Up to date'}
                  </Button>
                  {!price.display_enabled && (
                    <span className="text-xs text-slate-400">Hidden from landing page visitors</span>
                  )}
                </div>
              </div>
            );
          })}
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
