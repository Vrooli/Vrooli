import { Trash2, Tag, Clock, Infinity as InfinityIcon, Repeat, AlertTriangle, Pencil } from 'lucide-react';
import { Card, CardContent } from '../../../../shared/ui/card';
import { Button } from '../../../../shared/ui/button';
import type { StripeCoupon, CouponUsageStats } from '../../../../shared/api/billing';

interface CouponCardProps {
  coupon: StripeCoupon;
  usageStats?: CouponUsageStats;
  onDelete: (couponId: string) => Promise<{ success: boolean; error?: string }>;
  onEdit?: (coupon: StripeCoupon) => void;
  isDeleting: boolean;
}

/**
 * Format discount amount for display
 */
function formatDiscount(coupon: StripeCoupon): string {
  if (coupon.percent_off != null) {
    return `${String(coupon.percent_off)}% off`;
  }
  if (coupon.amount_off != null) {
    const currency = (coupon.currency || 'usd').toUpperCase();
    const amount = coupon.amount_off / 100;
    return `$${amount.toFixed(2)} ${currency} off`;
  }
  return 'Unknown discount';
}

/**
 * Get duration badge info
 */
function getDurationBadge(coupon: StripeCoupon): { label: string; icon: React.ReactNode; color: string } {
  switch (coupon.duration) {
    case 'once':
      return {
        label: 'Once',
        icon: <Clock className="h-3 w-3" />,
        color: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
      };
    case 'repeating':
      return {
        label: `${String(coupon.duration_in_months ?? 0)} months`,
        icon: <Repeat className="h-3 w-3" />,
        color: 'bg-purple-500/20 text-purple-300 border-purple-500/30',
      };
    case 'forever':
      return {
        label: 'Forever',
        icon: <InfinityIcon className="h-3 w-3" />,
        color: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
      };
    default:
      return {
        label: coupon.duration,
        icon: <Clock className="h-3 w-3" />,
        color: 'bg-slate-500/20 text-slate-300 border-slate-500/30',
      };
  }
}

/**
 * Format Unix timestamp to readable date
 */
function formatDate(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function CouponCard({ coupon, usageStats, onDelete, onEdit, isDeleting }: CouponCardProps) {
  const durationBadge = getDurationBadge(coupon);
  const usageCount = usageStats?.total_uses ?? 0;

  const handleDelete = async () => {
    const warning = coupon.is_intro_coupon
      ? `This coupon is configured for intro pricing (${coupon.intro_tier ?? 'unknown'} tier). Deleting it will break intro pricing for that tier. Are you sure?`
      : 'Are you sure you want to delete this coupon? This cannot be undone.';

    if (!confirm(warning)) return;
    await onDelete(coupon.id);
  };

  return (
    <Card className="border-white/10 bg-slate-900/40 hover:bg-slate-900/60 transition-colors">
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-4">
          {/* Left section: ID, name, discount */}
          <div className="flex items-start gap-3 min-w-0 flex-1">
            <div className="p-2 rounded-lg bg-emerald-500/20 flex-shrink-0">
              <Tag className="h-5 w-5 text-emerald-300" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-mono text-sm font-medium text-white truncate">
                  {coupon.id}
                </span>
                {coupon.is_intro_coupon && (
                  <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-cyan-500/20 text-cyan-300 border border-cyan-500/30">
                    Intro: {coupon.intro_tier}
                  </span>
                )}
                {!coupon.valid && (
                  <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-red-500/20 text-red-300 border border-red-500/30">
                    <AlertTriangle className="h-3 w-3" />
                    Expired
                  </span>
                )}
              </div>
              {coupon.name && (
                <p className="text-sm text-slate-400 truncate mt-0.5">{coupon.name}</p>
              )}
              <p className="text-lg font-semibold text-emerald-400 mt-1">
                {formatDiscount(coupon)}
              </p>
            </div>
          </div>

          {/* Right section: badges and actions */}
          <div className="flex flex-col items-end gap-2 flex-shrink-0">
            {/* Duration badge */}
            <span className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full border ${durationBadge.color}`}>
              {durationBadge.icon}
              {durationBadge.label}
            </span>

            {/* Action buttons */}
            <div className="flex items-center gap-1">
              {onEdit && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => { onEdit(coupon); }}
                  className="text-slate-400 hover:text-blue-400 hover:bg-blue-500/10 h-8 w-8 p-0"
                  title="Edit coupon"
                >
                  <Pencil className="h-4 w-4" />
                </Button>
              )}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => { void handleDelete(); }}
                disabled={isDeleting}
                className="text-slate-400 hover:text-red-400 hover:bg-red-500/10 h-8 w-8 p-0"
                title="Delete coupon"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        {/* Bottom row: usage and metadata */}
        <div className="flex items-center gap-4 mt-3 pt-3 border-t border-white/5 text-xs text-slate-500">
          <span>
            Used: {coupon.times_redeemed}
            {coupon.max_redemptions != null && ` / ${String(coupon.max_redemptions)}`}
          </span>
          {usageCount > 0 && (
            <span className="text-cyan-400">
              {usageCount} local intro use{usageCount !== 1 ? 's' : ''}
            </span>
          )}
          <span>Created: {formatDate(coupon.created)}</span>
          {coupon.redeem_by != null && (
            <span className={coupon.valid ? 'text-slate-400' : 'text-red-400'}>
              Expires: {formatDate(coupon.redeem_by)}
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
