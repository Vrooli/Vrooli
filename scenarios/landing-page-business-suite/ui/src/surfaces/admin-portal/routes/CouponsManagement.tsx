import { Tag, Plus, RefreshCcw, CheckCircle, AlertTriangle, Settings } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Callout } from '../components/Callout';
import { Card, CardContent } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { LAYOUT } from '../config/layout.constants';
import { CouponCard, CreateCouponModal } from '../components/coupons';
import { useCouponsManagement, type CouponFilter } from '../hooks/useCouponsManagement';

const filterOptions: { value: CouponFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'expired', label: 'Expired' },
];

export function CouponsManagement() {
  const {
    filteredCoupons,
    introCouponMap,
    usageStats,
    filter,
    setFilter,
    totalCount,
    activeCount,
    introConfiguredCount,
    loading,
    error,
    createModalOpen,
    creating,
    createError,
    deletingId,
    loadCoupons,
    openCreateModal,
    closeCreateModal,
    handleCreate,
    handleDelete,
  } = useCouponsManagement();

  // Create a map for quick lookup of usage stats
  const usageStatsMap = Object.fromEntries(usageStats.map((s) => [s.coupon_id, s]));

  if (loading) {
    return (
      <AdminLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <p className="text-slate-400">Loading coupons...</p>
        </div>
      </AdminLayout>
    );
  }

  if (error) {
    const isAuthError = error.toLowerCase().includes('authentication');
    return (
      <AdminLayout>
        <div className="space-y-4">
          <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4">
            <p className="text-red-400">Error: {error}</p>
            <Button onClick={loadCoupons} variant="outline" className="mt-4">
              Retry
            </Button>
          </div>
          {isAuthError && (
            <Callout
              type="info"
              title="Using a Restricted Stripe Key?"
              message="If you're using a restricted API key, ensure it has Coupons permissions enabled in the Stripe dashboard: Developers → API keys → Edit key → Coupons → Read and Write."
            />
          )}
        </div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.pageSpacing}>
        <div className="flex items-start justify-between gap-4">
          <PageHeader
            variant="icon-title"
            title="Coupon Management"
            description="Create, view, and manage Stripe coupons for discounts and intro pricing."
            icon={Tag}
            iconBgClass="bg-emerald-500/10"
            iconColorClass="text-emerald-400"
            testId="coupons-management-header"
          />
          <div className="flex items-center gap-2 flex-shrink-0">
            <Button
              size="sm"
              onClick={openCreateModal}
              className="gap-2"
            >
              <Plus className="h-4 w-4" />
              Create Coupon
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={loadCoupons}
              className="gap-2"
            >
              <RefreshCcw className="h-4 w-4" />
              Refresh
            </Button>
          </div>
        </div>

        {/* Stats summary row */}
        <div className="grid grid-cols-2 gap-4 md:grid-cols-3" data-testid="coupons-stats">
          <StatCard
            label="Total Coupons"
            value={totalCount}
            icon={Tag}
            iconColor="text-emerald-300"
            iconBg="bg-emerald-500/20"
          />
          <StatCard
            label="Active"
            value={activeCount}
            icon={CheckCircle}
            iconColor={activeCount > 0 ? 'text-green-300' : 'text-slate-400'}
            iconBg={activeCount > 0 ? 'bg-green-500/20' : 'bg-slate-500/20'}
          />
          <StatCard
            label="Intro Configured"
            value={introConfiguredCount}
            icon={Settings}
            iconColor={introConfiguredCount > 0 ? 'text-cyan-300' : 'text-slate-400'}
            iconBg={introConfiguredCount > 0 ? 'bg-cyan-500/20' : 'bg-slate-500/20'}
          />
        </div>

        {/* Intro pricing callout */}
        {introCouponMap && Object.keys(introCouponMap).length > 0 && (
          <Callout
            type="info"
            title="Intro Pricing Configured"
            message={`Environment variables are set for ${Object.keys(introCouponMap).length} tier(s): ${Object.entries(introCouponMap)
              .map(([tier, couponId]) => `${tier} = ${couponId}`)
              .join(', ')}`}
          />
        )}

        {/* Filter tabs */}
        <Card className={`${LAYOUT.card.base}`}>
          <CardContent className="py-3">
            <div className="flex items-center gap-2">
              {filterOptions.map((option) => (
                <button
                  key={option.value}
                  onClick={() => setFilter(option.value)}
                  className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
                    filter === option.value
                      ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
                      : 'text-slate-400 hover:text-white hover:bg-white/5'
                  }`}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Coupon list */}
        {filteredCoupons.length === 0 ? (
          <Card className={LAYOUT.card.base}>
            <CardContent className="py-12 text-center">
              <Tag className="h-12 w-12 mx-auto text-slate-500 mb-4" />
              <p className="text-slate-400">No coupons found</p>
              <p className="text-sm text-slate-500 mt-1">
                {filter !== 'all'
                  ? 'Try adjusting your filter'
                  : 'Create a coupon to get started'}
              </p>
              {filter === 'all' && (
                <Button onClick={openCreateModal} className="mt-4 gap-2">
                  <Plus className="h-4 w-4" />
                  Create Coupon
                </Button>
              )}
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {filteredCoupons.map((coupon) => (
              <CouponCard
                key={coupon.id}
                coupon={coupon}
                usageStats={usageStatsMap[coupon.id]}
                onDelete={handleDelete}
                isDeleting={deletingId === coupon.id}
              />
            ))}
          </div>
        )}

        {/* Warning for missing intro coupons */}
        {introCouponMap && Object.keys(introCouponMap).length > 0 && filteredCoupons.length > 0 && (
          <>
            {Object.entries(introCouponMap)
              .filter(([, couponId]) => !filteredCoupons.some((c) => c.id === couponId))
              .map(([tier, couponId]) => (
                <Callout
                  key={tier}
                  type="warning"
                  title="Missing Intro Coupon"
                  message={`Coupon "${couponId}" is configured for the ${tier} tier but does not exist in Stripe. Create this coupon or update the INTRO_COUPON_${tier.toUpperCase()} environment variable.`}
                  icon={AlertTriangle}
                />
              ))}
          </>
        )}
      </div>

      {/* Create modal */}
      <CreateCouponModal
        isOpen={createModalOpen}
        onClose={closeCreateModal}
        onCreate={handleCreate}
        creating={creating}
        error={createError}
      />
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
