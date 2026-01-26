import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { FormField, inputClassName, textareaClassName } from '../components/FormField';
import { SecretInput } from '../components/SecretInput';
import { Callout } from '../components/Callout';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Textarea } from '../../../shared/ui/input';
import { MetricsModeProvider } from '../../../shared/hooks/useMetrics';
import { PricingSection } from '../../public-landing/sections/PricingSection';
import { AlertTriangle, CreditCard, Package, RefreshCw, ShieldCheck } from 'lucide-react';
import { revealStripeSecret } from '../../../shared/api';
import { isDemoPlanOption } from '../../../shared/lib/pricingPlaceholders';
import { cn } from '../../../shared/lib/utils';
import { useBillingForm } from '../hooks/useBillingForm';
import {
  filterPricesByTab,
  normalizeInterval,
  getIntervalLabel,
  buildPricingPreviewData,
  isPriceFormDirty,
  getPriceIdentifier,
  type PriceFormState,
  type PricingPreviewData,
} from '../services/pricing.service';
import type { BundleCatalogEntry } from '../../../shared/api';

const PRICING_PREVIEW_CONTENT = {
  title: 'Landing pricing preview',
  subtitle: 'Updates instantly with unsaved copy changes so you can validate the three-card layout visitors will see.',
};

export function BillingSettings() {
  const {
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
  } = useBillingForm();

  const currentDashboardUrl = stripeSettings?.dashboard_url;

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Billing & Subscription"
          description="Configure Stripe integration, manage pricing plans, and control how your landing page displays subscription options."
          icon={CreditCard}
          iconBgClass="bg-emerald-500/10"
          iconColorClass="text-emerald-400"
          testId="billing-header"
        />

        <FormSection
          title="Stripe Configuration"
          description="Store publishable/restricted keys and link directly to the Stripe dashboard."
          icon={CreditCard}
          iconColorClass="text-blue-300"
          actions={currentDashboardUrl ? (
            <Button
              variant="ghost"
              size="sm"
              className="gap-2"
              onClick={() => window.open(currentDashboardUrl, '_blank', 'noopener,noreferrer')}
            >
              Open Stripe Dashboard
            </Button>
          ) : undefined}
        >
            {loadingStripe ? (
              <p className="text-slate-400">Loading Stripe settings...</p>
            ) : (
              <>
                {/* Status Badges */}
                {stripeStatusBadges.length > 0 && (
                  <div className="flex flex-wrap gap-3">
                    {stripeStatusBadges.map((badge) => (
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
                )}
                {stripeSettings?.source && (
                  <p className="text-xs uppercase tracking-wide text-slate-500">
                    Source: {stripeSettings.source === 'database' ? 'Admin override' : 'Environment variables'}
                  </p>
                )}
                {stripeError && <p className="text-sm text-rose-300">{stripeError}</p>}
                <form onSubmit={handleStripeSave} className="grid gap-4 md:grid-cols-2">
                  <FormField
                    label="Publishable Key"
                    helpText={stripeSettings?.publishable_key_set ? 'Publishable key is saved. Enter a new value to rotate.' : undefined}
                  >
                    <SecretInput
                      isSet={stripeSettings?.publishable_key_set ?? false}
                      value={stripeForm.publishableKey}
                      onChange={handleStripeInput('publishableKey')}
                      placeholder="pk_live_..."
                      onReveal={() => revealStripeSecret('publishable_key').then((r) => r.value)}
                    />
                  </FormField>
                  <FormField
                    label="Restricted Key (secret)"
                    helpText={stripeSettings?.secret_key_set ? 'Restricted key is saved. Enter a new value to rotate.' : undefined}
                  >
                    <SecretInput
                      isSet={stripeSettings?.secret_key_set ?? false}
                      value={stripeForm.secretKey}
                      onChange={handleStripeInput('secretKey')}
                      placeholder="rk_live_..."
                      onReveal={() => revealStripeSecret('secret_key').then((r) => r.value)}
                    />
                  </FormField>
                  <FormField
                    label="Webhook Secret"
                    helpText={stripeSettings?.webhook_secret_set ? 'Webhook secret is saved. Enter a new value to rotate.' : undefined}
                  >
                    <SecretInput
                      isSet={stripeSettings?.webhook_secret_set ?? false}
                      value={stripeForm.webhookSecret}
                      onChange={handleStripeInput('webhookSecret')}
                      placeholder="whsec_..."
                      onReveal={() => revealStripeSecret('webhook_secret').then((r) => r.value)}
                    />
                  </FormField>
                  <FormField label="Dashboard URL">
                    <input
                      type="url"
                      value={stripeForm.dashboardUrl}
                      onChange={handleStripeInput('dashboardUrl')}
                      placeholder="https://dashboard.stripe.com/..."
                      className={inputClassName}
                    />
                  </FormField>
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
        </FormSection>

        {/* Plan Display Manager */}
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
                onClick={toggleDemoPlaceholders}
                className="gap-2"
              >
                {includeDemoPlaceholders ? 'Hide demo placeholders' : 'Show demo placeholders'}
              </Button>
              <Button variant="ghost" size="sm" onClick={loadBundles} className="gap-2">
                <RefreshCw className="h-4 w-4" /> Reload catalog
              </Button>
            </div>
          </div>
          <div className="space-y-6">
            {loadingBundles && <p className="text-slate-400">Loading bundle catalog...</p>}
            {bundleError && (
              <Callout type="error" message={bundleError} />
            )}
            {!loadingBundles && !bundleError && bundles.length === 0 && (
              <Card className={LAYOUT.card.base}>
                <CardContent className="py-12 text-center">
                  <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-slate-800">
                    <Package className="h-8 w-8 text-slate-400" />
                  </div>
                  <h3 className="mb-2 text-lg font-semibold text-white">No pricing plans configured</h3>
                  <p className="mb-6 max-w-md mx-auto text-sm text-slate-400">
                    You don't have any pricing bundles set up yet. Enable "Show demo placeholders" to preview
                    the layout with sample plans, or connect your Stripe account to create real pricing tiers.
                  </p>
                  <Button
                    variant="outline"
                    onClick={toggleDemoPlaceholders}
                    className="gap-2"
                  >
                    {includeDemoPlaceholders ? 'Hide demo placeholders' : 'Show demo placeholders'}
                  </Button>
                </CardContent>
              </Card>
            )}
            {!loadingBundles && !bundleError && bundles.length > 0 && bundles.map((entry) => (
              <BundleCard
                key={entry.bundle.bundle_key}
                entry={entry}
                priceForms={priceForms}
                pricingTab={pricingTab}
                includeDemoPlaceholders={includeDemoPlaceholders}
                priceChecks={priceChecks}
                onPriceChange={handlePriceChange}
                onSavePrice={handleSavePrice}
                onVerifyPrice={handleVerifyPrice}
                onRemoveDemoPlan={removeDemoPlan}
              />
            ))}
          </div>
        </div>
      </div>
    </AdminLayout>
  );
}

// Extracted bundle card component for cleaner rendering
interface BundleCardProps {
  entry: BundleCatalogEntry;
  priceForms: Record<string, PriceFormState>;
  pricingTab: 'month' | 'year' | 'other';
  includeDemoPlaceholders: boolean;
  priceChecks: Record<string, { status: string; message?: string }>;
  onPriceChange: (bundleKey: string, priceId: string, field: any, transformer?: any) => (event: any) => void;
  onSavePrice: (bundleKey: string, priceId: string) => Promise<void>;
  onVerifyPrice: (bundleKey: string, priceId: string) => Promise<void>;
  onRemoveDemoPlan: (bundleKey: string, priceId: string) => void;
}

function BundleCard({
  entry,
  priceForms,
  pricingTab,
  includeDemoPlaceholders,
  priceChecks,
  onPriceChange,
  onSavePrice,
  onVerifyPrice,
  onRemoveDemoPlan,
}: BundleCardProps) {
  const visiblePrices = filterPricesByTab(entry.prices, pricingTab, includeDemoPlaceholders);
  const demoHidden = entry.prices.some(isDemoPlanOption) && !includeDemoPlaceholders;
  const previewData = buildPricingPreviewData(entry, priceForms, includeDemoPlaceholders);

  return (
    <Card className={LAYOUT.card.base}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>{entry.bundle.name}</span>
          <span className="text-xs text-slate-400">Bundle key: {entry.bundle.bundle_key}</span>
        </CardTitle>
        <CardDescription>
          Control which Stripe prices are surfaced on the landing page and customize their marketing copy.
        </CardDescription>
      </CardHeader>
      <CardContent className={LAYOUT.contentSpacing}>
        {demoHidden && (
          <Callout
            type="warning"
            message="Demo placeholders hidden. Turn them back on to see filler tiers until you add real Stripe prices."
          />
        )}
        <div className="grid gap-6 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
          <div className="space-y-6">
            {visiblePrices.map((price) => {
              const priceIdentifier = getPriceIdentifier(price);
              const key = `${entry.bundle.bundle_key}:${priceIdentifier}`;
              const formState = priceForms[key];
              if (!formState) return null;

              return (
                <PriceFormCard
                  key={key}
                  bundleKey={entry.bundle.bundle_key}
                  priceIdentifier={priceIdentifier}
                  price={price}
                  formState={formState}
                  priceCheck={priceChecks[key]}
                  onPriceChange={onPriceChange}
                  onSavePrice={onSavePrice}
                  onVerifyPrice={onVerifyPrice}
                  onRemoveDemoPlan={onRemoveDemoPlan}
                />
              );
            })}
          </div>
          <PlanPreview data={previewData} />
        </div>
      </CardContent>
    </Card>
  );
}

// Extracted price form card for cleaner rendering
interface PriceFormCardProps {
  bundleKey: string;
  priceIdentifier: string;
  price: any;
  formState: PriceFormState;
  priceCheck?: { status: string; message?: string };
  onPriceChange: (bundleKey: string, priceId: string, field: any, transformer?: any) => (event: any) => void;
  onSavePrice: (bundleKey: string, priceId: string) => Promise<void>;
  onVerifyPrice: (bundleKey: string, priceId: string) => Promise<void>;
  onRemoveDemoPlan: (bundleKey: string, priceId: string) => void;
}

function PriceFormCard({
  bundleKey,
  priceIdentifier,
  price,
  formState,
  priceCheck,
  onPriceChange,
  onSavePrice,
  onVerifyPrice,
  onRemoveDemoPlan,
}: PriceFormCardProps) {
  const dirty = isPriceFormDirty(formState);
  const demoPlan = formState.demo;

  return (
    <div className="rounded-xl border border-white/10 p-4">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold text-white">{price.plan_name}</h3>
          <div className="flex flex-wrap items-center gap-2 text-sm text-slate-400">
            <span>{getIntervalLabel(normalizeInterval(price.billing_interval))} · {price.currency.toUpperCase()}</span>
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
            onClick={() => onRemoveDemoPlan(bundleKey, priceIdentifier)}
          >
            Remove demo placeholder
          </Button>
        )}
        <label className="flex items-center gap-2 text-sm text-slate-200">
          <input
            type="checkbox"
            checked={formState.values.displayEnabled}
            onChange={onPriceChange(bundleKey, priceIdentifier, 'displayEnabled')}
            className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
          />
          Visible on landing page
        </label>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-3">
        <FormField label="Plan Name" className="md:col-span-2">
          <input
            type="text"
            value={formState.values.planName}
            onChange={onPriceChange(bundleKey, priceIdentifier, 'planName')}
            className={inputClassName}
          />
        </FormField>
        <FormField label="Display Weight">
          <input
            type="number"
            value={formState.values.displayWeight}
            onChange={onPriceChange(bundleKey, priceIdentifier, 'displayWeight')}
            className={inputClassName}
          />
        </FormField>
      </div>

      <FormField label="Stripe Price ID" className="mt-4">
        <input
          type="text"
          value={formState.values.stripePriceId}
          onChange={onPriceChange(bundleKey, priceIdentifier, 'stripePriceId')}
          placeholder="price_abc123 or lookup key if using Stripe aliases"
          className={inputClassName}
        />
        <div className="mt-2 flex items-center gap-2 text-xs">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="border border-white/10 bg-white/5 text-white"
            onClick={() => onVerifyPrice(bundleKey, priceIdentifier)}
          >
            Verify
          </Button>
          {priceCheck?.status === 'checking' && <span className="text-slate-300">Checking...</span>}
          {priceCheck?.status === 'ok' && <span className="text-emerald-300">{priceCheck.message || 'Verified'}</span>}
          {priceCheck?.status === 'error' && <span className="text-amber-200">{priceCheck.message || 'Verification failed'}</span>}
        </div>
        <p className="mt-1 text-xs text-slate-400">Paste the actual Stripe price ID or leave blank for free/CTA-only tiers.</p>
      </FormField>

      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <FormField label="Subtitle">
          <input
            type="text"
            value={formState.values.subtitle}
            onChange={onPriceChange(bundleKey, priceIdentifier, 'subtitle')}
            className={inputClassName}
          />
        </FormField>
        <FormField label="Badge">
          <input
            type="text"
            value={formState.values.badge}
            onChange={onPriceChange(bundleKey, priceIdentifier, 'badge')}
            className={inputClassName}
          />
        </FormField>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <FormField label="CTA Label">
          <input
            type="text"
            value={formState.values.ctaLabel}
            onChange={onPriceChange(bundleKey, priceIdentifier, 'ctaLabel')}
            className={inputClassName}
          />
        </FormField>
        <label className="mt-6 flex items-center gap-2 text-sm text-slate-200">
          <input
            type="checkbox"
            checked={formState.values.highlight}
            onChange={onPriceChange(bundleKey, priceIdentifier, 'highlight')}
            className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
          />
          Highlight tier (apply hero styling)
        </label>
      </div>

      <FormField label="Feature Bullets" className="mt-4">
        <Textarea
          value={formState.values.featuresText}
          onChange={onPriceChange(bundleKey, priceIdentifier, 'featuresText')}
          rows={4}
          className={textareaClassName}
          placeholder={'One feature per line\nDesktop downloads included\nWhite-glove onboarding'}
        />
      </FormField>

      {formState.error && (
        <p className="mt-3 text-sm text-rose-300">{formState.error}</p>
      )}

      <div className="mt-4 flex items-center gap-3">
        <Button
          type="button"
          onClick={() => onSavePrice(bundleKey, priceIdentifier)}
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
}

// Plan preview component
function PlanPreview({ data }: { data: PricingPreviewData }) {
  let statusMessage = 'Showing live preview of enabled monthly plans.';
  if (data.monthlyCount === 0 && data.placeholderCount > 0) {
    statusMessage = 'No saved monthly plans yet - displaying demo placeholders so the layout stays complete.';
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
          <div className="max-h-[640px] overflow-y-auto rounded-[28px] border border-white/10 bg-surface-darker p-1" onClickCapture={(event) => {
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
