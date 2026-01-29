import { useNavigate } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { FormField } from '../components/FormField';
import { inputClassName } from '../components/formFieldClasses';
import { SecretInput } from '../components/SecretInput';
import { Callout } from '../components/Callout';
import { PlanDisplayManager } from '../components/plans';
import { LAYOUT } from '../config/layout.constants';
import { Button } from '../../../shared/ui/button';
import { AlertTriangle, CreditCard, RefreshCw, ShieldCheck } from 'lucide-react';
import { revealStripeSecret } from '../../../shared/api';
import { useBillingForm } from '../hooks/useBillingForm';

export function BillingSettings() {
  const navigate = useNavigate();
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

  // Count total real plans (excluding demo placeholders)
  const totalPlanCount = bundles.reduce((count, entry) => {
    return count + entry.prices.filter((p) => !p.metadata?.__demo).length;
  }, 0);

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

        {/* Plan Preview (read-only) */}
        <PlanDisplayManager
          mode="preview"
          bundles={bundles}
          priceForms={priceForms}
          activeTab={pricingTab}
          onTabChange={setPricingTab}
          showDemoPlaceholders={includeDemoPlaceholders}
          onToggleDemoPlaceholders={toggleDemoPlaceholders}
          onReload={loadBundles}
          loading={loadingBundles}
          error={bundleError}
          onPriceChange={handlePriceChange}
          onSavePrice={handleSavePrice}
          onVerifyPrice={handleVerifyPrice}
          onRemoveDemoPlan={removeDemoPlan}
          priceChecks={priceChecks}
        />

        {/* Navigation callout to Plans admin page */}
        {!loadingBundles && totalPlanCount > 0 && (
          <Callout
            type="info"
            title="Plans Detected"
            message={`${totalPlanCount} plan${totalPlanCount !== 1 ? 's' : ''} found from Stripe. Edit plan display settings in the Plans admin page.`}
            actions={[
              { label: 'Manage Plans', onClick: () => navigate('/admin/tiers') }
            ]}
          />
        )}
      </div>
    </AdminLayout>
  );
}
