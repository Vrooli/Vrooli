import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Button } from '../../../shared/ui/button';
import {
  AlertTriangle,
  AppWindow,
  ChevronDown,
  CreditCard,
  ExternalLink,
  Home,
  Palette,
  RefreshCw,
  Users,
} from 'lucide-react';
import { LAYOUT } from '../config/layout.constants';
import { useLandingVariant } from '../../../app/providers/useLandingVariant';
import { useAdminHome } from '../hooks/useAdminHome';
import { isStripeFullyConfigured } from '../services/billing.service';

/**
 * Admin home page - simplified dashboard with quick navigation to section dashboards
 *
 * Provides:
 * - Quick flow cards (one per dropdown group)
 * - Compact stats bar
 * - Collapsible danger zone for reset
 */
export function AdminHome() {
  const navigate = useNavigate();
  const { variant: liveVariant } = useLandingVariant();

  const {
    healthSnapshot,
    healthLoading,
    stripeSettings,
    stripeLoading,
    resettingDemoData,
    resetMessage,
    resetError,
    showResetConfirm,
    setShowResetConfirm,
    handleResetDemoData,
  } = useAdminHome();

  const [dangerExpanded, setDangerExpanded] = useState(false);

  const previewPublicLanding = () => {
    window.open('/', '_blank', 'noopener,noreferrer');
  };

  // Compute quick stats
  const activeVariants = healthLoading ? '...' : (healthSnapshot?.activeCount ?? 0);
  const trafficAllocated = healthLoading ? '...' : `${Math.max(0, Math.round(healthSnapshot?.totalWeight ?? 0))}%`;
  const stripeConfigured = stripeLoading
    ? '...'
    : isStripeFullyConfigured(stripeSettings)
      ? 'Yes'
      : 'No';
  const liveVariantName = liveVariant?.name ?? liveVariant?.slug ?? 'None';

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.sectionSpacing}>
        <PageHeader
          variant="icon-title"
          title="Landing Page Business Suite Admin"
          icon={Home}
          iconBgClass="bg-slate-500/10"
          iconColorClass="text-slate-400"
          testId="admin-home-header"
        />

        {/* Purpose Statement */}
        <div className="mb-8 rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6" data-testid="admin-purpose">
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <div>
              <h2 className="text-xl font-semibold text-white mb-2">
                Manage your landing page and business
              </h2>
              <p className="text-slate-400 text-sm">
                Use the section dashboards below to customize your landing page, configure payments, manage apps, and engage with users.
              </p>
            </div>
            <Button
              variant="outline"
              onClick={previewPublicLanding}
              className="gap-2 shrink-0"
              data-testid="admin-preview-landing"
            >
              <ExternalLink className="h-4 w-4" />
              Preview landing
            </Button>
          </div>
        </div>

        {/* Quick Flows - One per dropdown group */}
        <div className="mb-8" data-testid="admin-quick-flows">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500 mb-4">
            Quick navigation
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <QuickFlowCard
              title="Landing"
              description="Customize variants, analytics, branding, and AI improvements"
              icon={Palette}
              iconBg="bg-purple-500/20"
              iconColor="text-purple-300"
              onClick={() => navigate('/admin/landing')}
              testId="flow-landing"
            />
            <QuickFlowCard
              title="Billing"
              description="Configure Stripe, manage plans, and set up AI keys"
              icon={CreditCard}
              iconBg="bg-amber-500/20"
              iconColor="text-amber-300"
              onClick={() => navigate('/admin/billing-home')}
              testId="flow-billing"
            />
            <QuickFlowCard
              title="Apps"
              description="Set up downloads, monitor usage, and configure limits"
              icon={AppWindow}
              iconBg="bg-blue-500/20"
              iconColor="text-blue-300"
              onClick={() => navigate('/admin/apps')}
              testId="flow-apps"
            />
            <QuickFlowCard
              title="Users"
              description="Manage accounts, triage feedback, and handle waitlist"
              icon={Users}
              iconBg="bg-emerald-500/20"
              iconColor="text-emerald-300"
              onClick={() => navigate('/admin/users')}
              testId="flow-users"
            />
          </div>
        </div>

        {/* Compact Stats Bar */}
        <div
          className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-4"
          data-testid="admin-stats-bar"
        >
          <div className="grid gap-4 text-center md:grid-cols-4">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Active variants
              </p>
              <p className="text-2xl font-semibold text-white mt-1">
                {activeVariants}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Traffic allocated
              </p>
              <p className="text-2xl font-semibold text-white mt-1">
                {trafficAllocated}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Stripe configured
              </p>
              <p className={`text-2xl font-semibold mt-1 ${stripeConfigured === 'Yes' ? 'text-emerald-300' : stripeConfigured === 'No' ? 'text-amber-300' : 'text-white'}`}>
                {stripeConfigured}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Live variant
              </p>
              <p className="text-lg font-semibold text-white mt-1 truncate" title={liveVariantName}>
                {liveVariantName}
              </p>
            </div>
          </div>
        </div>

        {/* Collapsible Danger Zone */}
        <div
          className="rounded-2xl border border-rose-500/30 bg-rose-500/5"
          data-testid="admin-danger-zone"
        >
          <button
            type="button"
            onClick={() => setDangerExpanded(!dangerExpanded)}
            className="w-full p-4 flex items-center justify-between text-left hover:bg-rose-500/10 transition-colors rounded-2xl"
            data-testid="admin-danger-toggle"
          >
            <div className="flex items-center gap-3">
              <AlertTriangle className="h-5 w-5 text-rose-400" />
              <div>
                <h3 className="font-semibold text-white">Danger Zone</h3>
                <p className="text-sm text-rose-100/60">Reset to template defaults</p>
              </div>
            </div>
            <ChevronDown className={`h-5 w-5 text-rose-400 transition-transform ${dangerExpanded ? 'rotate-180' : ''}`} />
          </button>

          {dangerExpanded && (
            <div className="px-4 pb-4 pt-2 border-t border-rose-500/20">
              <p className="text-sm text-rose-100/80 mb-4">
                Re-seed all landing page content from the fallback template. Use this to apply updated template content or restore defaults.
              </p>

              {resetError && (
                <p className="mb-3 text-sm text-rose-200" data-testid="admin-reset-error">{resetError}</p>
              )}
              {resetMessage && (
                <p className="mb-3 text-sm text-emerald-200" data-testid="admin-reset-success">{resetMessage}</p>
              )}

              {!showResetConfirm ? (
                <Button
                  variant="outline"
                  className="gap-2 border-rose-500/50 text-rose-200 hover:bg-rose-500/10"
                  onClick={() => setShowResetConfirm(true)}
                  disabled={resettingDemoData}
                  data-testid="admin-reset-demo-btn"
                >
                  <RefreshCw className={`h-4 w-4 ${resettingDemoData ? 'animate-spin' : ''}`} />
                  {resettingDemoData ? 'Resetting...' : 'Reset demo data'}
                </Button>
              ) : (
                <div className="rounded-xl border border-rose-500/50 bg-rose-500/10 p-4 space-y-4" data-testid="admin-reset-confirm-dialog">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="h-5 w-5 text-rose-400 mt-0.5 flex-shrink-0" />
                    <div className="space-y-2">
                      <p className="font-semibold text-rose-100">Are you sure you want to reset?</p>
                      <p className="text-sm text-rose-200/80">This action will permanently delete:</p>
                      <ul className="text-sm text-rose-200/80 list-disc list-inside space-y-1">
                        <li>All variant customizations</li>
                        <li>All section content edits</li>
                        <li>Download app configurations</li>
                        <li>Bundle pricing settings</li>
                      </ul>
                      <p className="text-sm text-rose-200/80 font-medium">
                        Data will be replaced with the current template fallback. This cannot be undone.
                      </p>
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    <Button
                      variant="outline"
                      className="gap-2"
                      onClick={() => setShowResetConfirm(false)}
                      disabled={resettingDemoData}
                    >
                      Cancel
                    </Button>
                    <Button
                      className="gap-2 bg-rose-600 hover:bg-rose-700 text-white"
                      onClick={handleResetDemoData}
                      disabled={resettingDemoData}
                      data-testid="admin-reset-confirm-btn"
                    >
                      <RefreshCw className={`h-4 w-4 ${resettingDemoData ? 'animate-spin' : ''}`} />
                      {resettingDemoData ? 'Resetting...' : 'Yes, reset everything'}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
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
}

function QuickFlowCard({
  title,
  description,
  icon: Icon,
  iconBg,
  iconColor,
  onClick,
  testId,
}: QuickFlowCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group rounded-xl border border-white/10 bg-white/5 p-5 text-left hover:bg-white/10 transition-all"
      data-testid={testId}
    >
      <div className="flex items-center gap-3 mb-3">
        <div className={`p-3 rounded-xl ${iconBg}`}>
          <Icon className={`h-6 w-6 ${iconColor}`} />
        </div>
        <h3 className="text-xl font-semibold text-white">{title}</h3>
      </div>
      <p className="text-sm text-slate-400 mb-3">{description}</p>
      <span className="inline-flex items-center gap-1 text-sm font-semibold text-slate-300 group-hover:translate-x-1 transition-transform">
        Open dashboard →
      </span>
    </button>
  );
}
