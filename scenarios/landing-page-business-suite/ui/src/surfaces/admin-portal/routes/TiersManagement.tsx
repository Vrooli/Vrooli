import { useMemo } from 'react';
import { Layers, Calendar, CalendarDays, Eye, EyeOff } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { PlanDisplayManager } from '../components/plans';
import { LAYOUT } from '../config/layout.constants';
import { useBillingForm } from '../hooks/useBillingForm';
import { normalizeInterval } from '../services/pricing.service';
import { isDemoPlanOption } from '../../../shared/lib/pricingPlaceholders';

export function TiersManagement() {
  const {
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

  // Compute stats from bundles (excluding demo placeholders)
  const stats = useMemo(() => {
    let total = 0;
    let enabled = 0;
    let monthly = 0;
    let yearly = 0;

    bundles.forEach((entry) => {
      entry.prices.forEach((price) => {
        if (isDemoPlanOption(price)) return;
        total++;
        if (price.display_enabled) enabled++;
        const interval = normalizeInterval(price.billing_interval);
        if (interval === 'month') monthly++;
        if (interval === 'year') yearly++;
      });
    });

    return { total, enabled, monthly, yearly };
  }, [bundles]);

  return (
    <AdminLayout maxWidth="extraWide">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Plan Management"
          description="Configure pricing plans and control how subscription tiers appear to visitors."
          icon={Layers}
          iconBgClass="bg-purple-500/10"
          iconColorClass="text-purple-400"
          testId="tiers-management-header"
        />

        {/* Stats summary row */}
        {!loadingBundles && stats.total > 0 && (
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4" data-testid="tiers-stats">
            <StatCard
              label="Total Plans"
              value={stats.total}
              icon={Layers}
              iconColor="text-purple-300"
              iconBg="bg-purple-500/20"
            />
            <StatCard
              label="Enabled"
              value={stats.enabled}
              icon={stats.enabled > 0 ? Eye : EyeOff}
              iconColor={stats.enabled > 0 ? 'text-emerald-300' : 'text-slate-400'}
              iconBg={stats.enabled > 0 ? 'bg-emerald-500/20' : 'bg-slate-500/20'}
            />
            <StatCard
              label="Monthly"
              value={stats.monthly}
              icon={Calendar}
              iconColor="text-blue-300"
              iconBg="bg-blue-500/20"
            />
            <StatCard
              label="Yearly"
              value={stats.yearly}
              icon={CalendarDays}
              iconColor="text-amber-300"
              iconBg="bg-amber-500/20"
            />
          </div>
        )}

        {/* Plan Display Manager (edit mode) */}
        <PlanDisplayManager
          mode="edit"
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
      </div>
    </AdminLayout>
  );
}

interface StatCardProps {
  label: string;
  value: number;
  icon: React.ElementType;
  iconColor: string;
  iconBg: string;
}

function StatCard({
  label,
  value,
  icon: Icon,
  iconColor,
  iconBg,
}: StatCardProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
      <div className="flex items-center gap-3 mb-3">
        <div className={`p-2 rounded-lg ${iconBg}`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
        <p className="text-xs uppercase tracking-[0.3em] text-slate-500">{label}</p>
      </div>
      <p className="text-3xl font-semibold text-white">{value}</p>
    </div>
  );
}
