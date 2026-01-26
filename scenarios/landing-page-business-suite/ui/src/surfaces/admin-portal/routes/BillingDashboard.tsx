import { useNavigate } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { StatusBadgeGrid } from '../components/StatusBadge';
import { Button } from '../../../shared/ui/button';
import {
  CreditCard,
  Key,
  Layers,
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Info,
} from 'lucide-react';
import { LAYOUT } from '../config/layout.constants';
import { useAdminHome } from '../hooks/useAdminHome';
import { isStripeFullyConfigured, isStripePartiallyConfigured } from '../services/billing.service';

/**
 * Billing Dashboard - Entry point for payment and monetization management
 *
 * Provides quick flows, stats, and navigation for:
 * - Stripe configuration
 * - Subscription plans/tiers
 * - AI API keys
 */
export function BillingDashboard() {
  const navigate = useNavigate();

  const {
    stripeSettings,
    stripeLoading,
    stripeError,
    refreshStripeStatus,
  } = useAdminHome();

  const stripeConfigured = isStripeFullyConfigured(stripeSettings);
  const stripePartial = isStripePartiallyConfigured(stripeSettings);

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.sectionSpacing}>
        <PageHeader
          variant="icon-title"
          title="Billing Dashboard"
          icon={CreditCard}
          iconBgClass="bg-amber-500/10"
          iconColorClass="text-amber-400"
          testId="billing-dashboard-header"
        />

        {/* Quick Flows */}
        <div className="mb-8" data-testid="billing-quick-flows">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500 mb-4">
            Quick flows
          </p>
          <div className="grid gap-4 md:grid-cols-3">
            <QuickFlowCard
              title="Configure Stripe"
              description="Set up Stripe API keys for payment processing"
              icon={CreditCard}
              iconBg="bg-amber-500/20"
              iconColor="text-amber-300"
              onClick={() => navigate('/admin/billing')}
              testId="flow-stripe"
            />
            <QuickFlowCard
              title="Manage plans"
              description="Configure subscription tiers and pricing"
              icon={Layers}
              iconBg="bg-purple-500/20"
              iconColor="text-purple-300"
              onClick={() => navigate('/admin/tiers')}
              testId="flow-plans"
              badge="Soon"
            />
            <QuickFlowCard
              title="AI API keys"
              description="Configure keys for AI providers"
              icon={Key}
              iconBg="bg-blue-500/20"
              iconColor="text-blue-300"
              onClick={() => navigate('/admin/api-keys')}
              testId="flow-api-keys"
            />
          </div>
        </div>

        {/* Stripe Status */}
        <div
          className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6"
          data-testid="billing-stripe-status"
        >
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-6">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Stripe configuration
              </p>
              <h2 className="text-xl font-semibold text-white mt-1">
                Payment processing status
              </h2>
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={refreshStripeStatus}
                className="gap-2"
                data-testid="billing-stripe-refresh"
              >
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
              <Button
                size="sm"
                onClick={() => navigate('/admin/billing')}
              >
                Configure Stripe
              </Button>
            </div>
          </div>

          {stripeLoading ? (
            <div className="grid gap-3 sm:grid-cols-3">
              {[0, 1, 2].map((i) => (
                <div key={i} className="h-16 rounded-xl bg-white/5 animate-pulse" />
              ))}
            </div>
          ) : stripeError ? (
            <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100 flex items-center gap-3">
              <AlertTriangle className="h-4 w-4" />
              <span>{stripeError}</span>
              <Button size="sm" variant="ghost" onClick={refreshStripeStatus}>
                Retry
              </Button>
            </div>
          ) : (
            <>
              <StatusBadgeGrid
                columns={3}
                badges={[
                  {
                    label: 'Publishable key',
                    status: stripeSettings?.publishable_key_set ? 'success' : 'warning',
                    description: stripeSettings?.publishable_key_set ? 'Configured' : 'Missing',
                  },
                  {
                    label: 'Secret key',
                    status: stripeSettings?.secret_key_set ? 'success' : 'warning',
                    description: stripeSettings?.secret_key_set ? 'Configured' : 'Missing',
                  },
                  {
                    label: 'Webhook secret',
                    status: stripeSettings?.webhook_secret_set ? 'success' : 'warning',
                    description: stripeSettings?.webhook_secret_set ? 'Configured' : 'Missing',
                  },
                ]}
              />
              {stripeSettings && (
                <div className="mt-4 text-xs text-slate-500 flex flex-wrap gap-4">
                  {stripeSettings.source && (
                    <span>
                      Source: {stripeSettings.source === 'database' ? 'Admin override' : 'Environment variables'}
                    </span>
                  )}
                  {stripeSettings.updated_at && (
                    <span>Last updated: {new Date(stripeSettings.updated_at).toLocaleString()}</span>
                  )}
                </div>
              )}
            </>
          )}
        </div>

        {/* Monetization Guidance */}
        <div
          className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6"
          data-testid="billing-guidance"
        >
          <div className="flex items-start gap-4">
            {stripeConfigured ? (
              <div className="p-3 rounded-xl bg-emerald-500/20">
                <CheckCircle className="h-6 w-6 text-emerald-300" />
              </div>
            ) : (
              <div className="p-3 rounded-xl bg-amber-500/20">
                <Info className="h-6 w-6 text-amber-300" />
              </div>
            )}
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-white mb-2">
                {stripeConfigured
                  ? 'Payments are ready'
                  : stripePartial
                    ? 'Stripe partially configured'
                    : 'Set up Stripe to accept payments'}
              </h3>
              <p className="text-sm text-slate-400 mb-4">
                {stripeConfigured
                  ? 'Your Stripe integration is fully configured. Customers can now make purchases through your landing page.'
                  : stripePartial
                    ? 'Some Stripe keys are missing. Complete the configuration to enable payment processing.'
                    : 'Configure your Stripe API keys to enable payment processing. You\'ll need your publishable key, secret key, and webhook secret from the Stripe dashboard.'}
              </p>
              {!stripeConfigured && (
                <div className="flex flex-wrap gap-3">
                  <Button
                    size="sm"
                    onClick={() => navigate('/admin/billing')}
                    data-testid="billing-guidance-setup"
                  >
                    {stripePartial ? 'Complete setup' : 'Set up Stripe'}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => window.open('https://dashboard.stripe.com/apikeys', '_blank', 'noopener,noreferrer')}
                  >
                    Open Stripe dashboard
                  </Button>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </AdminLayout>
  );
}

interface QuickFlowCardProps {
  title: string;
  description: string;
  icon: React.ElementType;
  iconBg: string;
  iconColor: string;
  onClick: () => void;
  testId: string;
  badge?: string;
}

function QuickFlowCard({
  title,
  description,
  icon: Icon,
  iconBg,
  iconColor,
  onClick,
  testId,
  badge,
}: QuickFlowCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group rounded-xl border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 transition-all"
      data-testid={testId}
    >
      <div className="flex items-center gap-3 mb-2">
        <div className={`p-2 rounded-lg ${iconBg}`}>
          <Icon className={`h-4 w-4 ${iconColor}`} />
        </div>
        <p className="font-semibold text-white">{title}</p>
        {badge && (
          <span className="text-[10px] text-slate-500 uppercase">{badge}</span>
        )}
      </div>
      <p className="text-sm text-slate-400">{description}</p>
      <span className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-slate-300 group-hover:translate-x-1 transition-transform">
        Go →
      </span>
    </button>
  );
}
