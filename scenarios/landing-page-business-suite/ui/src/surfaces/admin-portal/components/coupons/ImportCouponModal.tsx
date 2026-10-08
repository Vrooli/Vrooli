import { useMemo, useState } from 'react';
import { Check, Loader2, RefreshCw, Tag, Clock, Repeat, Infinity as InfinityIcon, ExternalLink } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../../../shared/ui/dialog';
import { cn } from '../../../../shared/lib/utils';
import { Callout } from '../Callout';
import type { UseCouponImportReturn } from '../../hooks/useCouponImport';
import type { CouponImportPreviewItem } from '../../../../shared/api/billing';

type CouponFilter = 'all' | 'active' | 'inactive' | 'assigned';
const COUPON_FILTERS: Array<{ key: CouponFilter; label: string }> = [
  { key: 'all', label: 'All' },
  { key: 'active', label: 'Active' },
  { key: 'assigned', label: 'Assigned' },
  { key: 'inactive', label: 'Inactive' },
];

interface ImportCouponModalProps {
  couponImport: UseCouponImportReturn;
}

/**
 * Format discount amount for display
 */
function formatDiscount(coupon: CouponImportPreviewItem): string {
  if (typeof coupon.percent_off === 'number' && coupon.percent_off > 0) {
    return `${String(coupon.percent_off)}% off`;
  }
  if (typeof coupon.amount_off === 'number' && coupon.amount_off > 0) {
    const currency = (coupon.currency || 'usd').toUpperCase();
    const amount = coupon.amount_off / 100;
    return `$${amount.toFixed(2)} ${currency} off`;
  }
  return 'Unknown discount';
}

/**
 * Get duration badge info
 */
function getDurationBadge(coupon: CouponImportPreviewItem): { label: string; icon: React.ReactNode; colorClass: string } {
  switch (coupon.duration) {
    case 'once':
      return {
        label: 'Once',
        icon: <Clock className="h-3 w-3" />,
        colorClass: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
      };
    case 'repeating':
      return {
        label: `${String(coupon.duration_in_months ?? 0)} months`,
        icon: <Repeat className="h-3 w-3" />,
        colorClass: 'bg-purple-500/20 text-purple-300 border-purple-500/30',
      };
    case 'forever':
      return {
        label: 'Forever',
        icon: <InfinityIcon className="h-3 w-3" />,
        colorClass: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
      };
    default:
      return {
        label: 'Unknown',
        icon: null,
        colorClass: 'bg-slate-500/20 text-slate-300 border-slate-500/30',
      };
  }
}

export function ImportCouponModal({ couponImport }: ImportCouponModalProps) {
  const {
    isModalOpen,
    closeModal,
    preview,
    loading,
    error,
    refreshPreview,
  } = couponImport;

  const [filter, setFilter] = useState<CouponFilter>('all');

  const filteredCoupons = useMemo(() => {
    if (!preview?.coupons) return [];

    switch (filter) {
      case 'active':
        return preview.coupons.filter((c) => c.valid);
      case 'inactive':
        return preview.coupons.filter((c) => !c.valid);
      case 'assigned':
        return preview.coupons.filter((c) => c.exists_locally);
      default:
        return preview.coupons;
    }
  }, [preview, filter]);

  const stats = useMemo(() => {
    if (!preview) return { total: 0, active: 0, assigned: 0, inactive: 0 };
    return {
      total: preview.total_coupons,
      active: preview.coupons.filter((c) => c.valid).length,
      assigned: preview.existing_count,
      inactive: preview.coupons.filter((c) => !c.valid).length,
    };
  }, [preview]);

  return (
    <Dialog open={isModalOpen} onOpenChange={(open) => { if (!open) { closeModal(); } }}>
      <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tag className="h-5 w-5" />
            Stripe Coupons
          </DialogTitle>
          <DialogDescription>
            View coupons from your Stripe account. To assign a coupon to a plan, go to the Plan Display Manager
            and select the coupon in the plan's "Intro Coupon" section.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4 space-y-4">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-slate-400" />
              <span className="ml-3 text-slate-400">Loading coupons from Stripe...</span>
            </div>
          )}

          {error && <Callout type="error" message={error} />}

          {!loading && !error && preview && (
            <>
              {/* Stats bar */}
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-4">
                  <span className="text-slate-400">
                    {stats.total} coupon{stats.total !== 1 ? 's' : ''} found
                  </span>
                  <span className="text-emerald-400">
                    {stats.active} active
                  </span>
                  {stats.assigned > 0 && (
                    <span className="text-blue-400">
                      {stats.assigned} assigned to plans
                    </span>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => { void refreshPreview(); }}
                  disabled={loading}
                  className="gap-1"
                >
                  <RefreshCw className="h-3 w-3" />
                  Refresh
                </Button>
              </div>

              {/* Filter tabs */}
              <div className="flex gap-1 p-1 bg-slate-800/50 rounded-lg w-fit">
                {COUPON_FILTERS.map(({ key, label }) => (
                  <button
                    key={key}
                    type="button"
                    onClick={() => { setFilter(key); }}
                    className={cn(
                      'px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
                      filter === key
                        ? 'bg-slate-700 text-white'
                        : 'text-slate-400 hover:text-white'
                    )}
                  >
                    {label}
                  </button>
                ))}
              </div>

              {/* Coupon list */}
              <div className="space-y-2">
                {filteredCoupons.length === 0 ? (
                  <div className="py-8 text-center text-sm text-slate-400">
                    No coupons match the current filter.
                  </div>
                ) : (
                  filteredCoupons.map((coupon) => {
                    const duration = getDurationBadge(coupon);
                    return (
                      <div
                        key={coupon.id}
                        className={cn(
                          'p-3 rounded-lg border transition-colors',
                          coupon.valid
                            ? 'bg-slate-800/50 border-slate-700/50'
                            : 'bg-slate-900/50 border-slate-800/50 opacity-60'
                        )}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="font-medium text-white truncate">
                                {coupon.name || coupon.id}
                              </span>
                              {coupon.name && (
                                <code className="text-xs text-slate-500 bg-slate-800 px-1 rounded">
                                  {coupon.id}
                                </code>
                              )}
                              {coupon.exists_locally && (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full bg-blue-500/20 text-blue-300 border border-blue-500/30">
                                  <Check className="h-3 w-3" />
                                  Assigned
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-3 mt-1">
                              <span className="text-sm text-emerald-400 font-medium">
                                {formatDiscount(coupon)}
                              </span>
                              <span
                                className={cn(
                                  'inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full border',
                                  duration.colorClass
                                )}
                              >
                                {duration.icon}
                                {duration.label}
                              </span>
                              {!coupon.valid && (
                                <span className="text-xs text-rose-400">
                                  Expired/Invalid
                                </span>
                              )}
                            </div>
                            {coupon.times_redeemed > 0 && (
                              <p className="text-xs text-slate-500 mt-1">
                                {coupon.times_redeemed} redemption{coupon.times_redeemed !== 1 ? 's' : ''}
                              </p>
                            )}
                          </div>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>

              {/* Info box */}
              <Callout
                type="info"
                message="To assign a coupon to a plan, open the plan in the Plan Display Manager and select the coupon from the 'Intro Coupon' dropdown. The coupon will be automatically applied during checkout for eligible customers."
              />
            </>
          )}
        </div>

        <DialogFooter className="border-t border-slate-700/50 pt-4">
          <Button variant="ghost" onClick={closeModal}>
            Close
          </Button>
          <a
            href="https://dashboard.stripe.com/coupons"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-slate-700 hover:bg-slate-600 rounded-md transition-colors"
          >
            <ExternalLink className="h-4 w-4" />
            Manage in Stripe
          </a>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
