import { useMemo, type DragEvent, type KeyboardEvent, type MouseEvent } from 'react';
import { ChevronDown, GripVertical, Loader2, RefreshCw, ShieldCheck, Trash2, Tag, X } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import { Textarea } from '../../../../shared/ui/textarea';
import { Card, CardContent, CardHeader } from '../../../../shared/ui/card';
import { FormField } from '../FormField';
import { inputClassName, textareaClassName } from '../formFieldClasses';
import { cn } from '../../../../shared/lib/utils';
import {
  normalizeInterval,
  getIntervalLabel,
  isPriceFormDirty,
  type PriceFormState,
  type PriceFormValues,
} from '../../services/pricing.service';
import type { PlanOption, StripeCoupon } from '../../../../shared/api';
import { formatDiscountPreview, getCouponSummary } from '../../../../shared/lib/pricingCalculations';

interface PriceVerificationResult {
  status: string;
  message?: string;
}

export interface PriceFormCardProps {
  bundleKey: string;
  priceIdentifier: string;
  price: PlanOption;
  formState: PriceFormState;
  priceCheck?: PriceVerificationResult;
  onPriceChange: (bundleKey: string, priceId: string, field: keyof PriceFormValues, transformer?: (value: string) => string | number) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  onSavePrice: (bundleKey: string, priceId: string) => Promise<void>;
  onVerifyPrice: (bundleKey: string, priceId: string) => Promise<void>;
  onRemoveDemoPlan: (bundleKey: string, priceId: string) => void;
  onDeletePlan?: (bundleKey: string, priceId: string) => void;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
  planIndex?: number;
  draggable?: boolean;
  onDragStart?: (event: DragEvent<HTMLElement>) => void;
  onDragOver?: (event: DragEvent<HTMLElement>) => void;
  onDragLeave?: () => void;
  onDrop?: (event: DragEvent<HTMLElement>) => void;
  onDragEnd?: () => void;
  isDragging?: boolean;
  isDragOver?: boolean;
  // Coupon mapping props
  availableCoupons?: StripeCoupon[];
  assignedCoupon?: StripeCoupon;
  onAssignCoupon?: (priceId: string, couponId: string) => Promise<{ success: boolean; error?: string }>;
  onUnassignCoupon?: (priceId: string) => Promise<{ success: boolean; error?: string }>;
  couponSaving?: boolean;
}

export function PriceFormCard({
  bundleKey,
  priceIdentifier,
  price,
  formState,
  priceCheck,
  onPriceChange,
  onSavePrice,
  onVerifyPrice,
  onRemoveDemoPlan,
  onDeletePlan,
  isCollapsed = true,
  onToggleCollapse,
  planIndex,
  draggable,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
  onDragEnd,
  isDragging,
  isDragOver,
  availableCoupons,
  assignedCoupon,
  onAssignCoupon,
  onUnassignCoupon,
  couponSaving,
}: PriceFormCardProps) {
  const dirty = isPriceFormDirty(formState);
  const demoPlan = formState.demo;
  const currencyLabel = price.currency ? price.currency.toUpperCase() : 'USD';
  const planLabel = formState.values.planName.trim() || price.plan_name;
  const stripePriceSummary = formState.values.stripePriceId || price.stripe_price_id || 'None (free/CTA)';

  const verificationStatus = priceCheck?.status || 'idle';
  const verifyDisabled = formState.saving || verificationStatus === 'checking';
  const verificationIcon = useMemo(() => {
    if (verificationStatus === 'checking') {
      return <Loader2 className="h-4 w-4 animate-spin text-slate-400" />;
    }
    return (
      <ShieldCheck
        className={cn(
          'h-4 w-4',
          verificationStatus === 'ok'
            ? 'text-emerald-400'
            : verificationStatus === 'error'
              ? 'text-amber-300'
              : 'text-slate-500'
        )}
      />
    );
  }, [verificationStatus]);

  const handleHeaderKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!onToggleCollapse) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onToggleCollapse();
    }
  };

  const stopPropagation = (event: MouseEvent) => {
    event.stopPropagation();
  };

  return (
    <Card
      className={cn(
        'border-white/10 bg-slate-900/50 shadow-none transition-all',
        isDragging && 'opacity-60 scale-[0.99]',
        isDragOver && 'ring-2 ring-blue-500/40 ring-offset-2 ring-offset-slate-950'
      )}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      <CardHeader
        className="p-4"
        onClick={onToggleCollapse}
        role={onToggleCollapse ? 'button' : undefined}
        tabIndex={onToggleCollapse ? 0 : undefined}
        onKeyDown={onToggleCollapse ? handleHeaderKeyDown : undefined}
        aria-expanded={onToggleCollapse ? !isCollapsed : undefined}
      >
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 min-w-0">
            <button
              type="button"
              className={cn(
                'mt-1 flex items-center gap-2 text-slate-400',
                draggable ? 'cursor-grab active:cursor-grabbing' : 'cursor-default opacity-60'
              )}
              title={draggable ? 'Drag to reorder' : undefined}
              onClick={stopPropagation}
              draggable={draggable}
              onDragStart={onDragStart}
              onDragEnd={onDragEnd}
              aria-label="Drag to reorder plan"
            >
              <GripVertical className="h-4 w-4" />
              {planIndex !== undefined && (
                <span className="text-xs text-slate-500 font-mono">#{planIndex + 1}</span>
              )}
            </button>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="text-lg font-semibold text-white truncate">{planLabel}</h3>
                {dirty && !demoPlan && (
                  <span className="rounded-full border border-blue-500/40 bg-blue-500/10 px-2 py-0.5 text-xs text-blue-200">
                    Unsaved changes
                  </span>
                )}
                <span
                  className={cn(
                    'inline-flex items-center gap-2 rounded-full border px-2 py-0.5 text-xs',
                    demoPlan
                      ? 'border-amber-500/50 bg-amber-500/10 text-amber-200'
                      : 'border-emerald-500/40 bg-emerald-500/10 text-emerald-100'
                  )}
                >
                  {demoPlan ? 'Demo placeholder (not saved)' : `Stripe price: ${stripePriceSummary}`}
                </span>
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
                <span>{getIntervalLabel(normalizeInterval(price.billing_interval))}</span>
                <span>·</span>
                <span>{currencyLabel}</span>
                <span>·</span>
                <span>Weight {formState.values.displayWeight}</span>
                {formState.values.stripePriceId && (
                  <>
                    <span>·</span>
                    <span className="font-mono text-slate-500 truncate max-w-[160px]">
                      {formState.values.stripePriceId}
                    </span>
                  </>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0" onClick={stopPropagation}>
            <label className="flex items-center gap-2 text-xs text-slate-200">
              <input
                type="checkbox"
                checked={formState.values.displayEnabled}
                onChange={onPriceChange(bundleKey, priceIdentifier, 'displayEnabled')}
                className="h-4 w-4 rounded border-slate-600 bg-slate-900 text-blue-500"
              />
              Visible
            </label>
            {demoPlan ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="gap-2 text-amber-200 hover:text-amber-100"
                onClick={() => { onRemoveDemoPlan(bundleKey, priceIdentifier); }}
              >
                Remove demo placeholder
              </Button>
            ) : (
              onDeletePlan && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="text-rose-300 hover:text-rose-200 hover:bg-rose-500/10"
                  onClick={() => { onDeletePlan(bundleKey, priceIdentifier); }}
                  disabled={formState.saving}
                  title="Delete plan"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              )
            )}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onToggleCollapse}
              className="h-8 w-8 p-0"
              title={isCollapsed ? 'Expand plan' : 'Collapse plan'}
            >
              <ChevronDown className={cn('h-4 w-4 transition-transform', !isCollapsed && 'rotate-180')} />
            </Button>
          </div>
        </div>
      </CardHeader>

      {!isCollapsed && (
        <CardContent className="pt-0 space-y-4">
          <div className="grid gap-4 md:grid-cols-3">
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

          <FormField label="Stripe Price ID">
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2">
                {verificationIcon}
              </span>
              <input
                type="text"
                value={formState.values.stripePriceId}
                onChange={onPriceChange(bundleKey, priceIdentifier, 'stripePriceId')}
                placeholder="price_abc123"
                className={cn(inputClassName, 'pl-10 pr-24')}
              />
              <button
                type="button"
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded-md border border-white/10 bg-white/5 px-2.5 py-1 text-xs font-medium text-white hover:bg-white/10 disabled:opacity-50"
                onClick={() => { void onVerifyPrice(bundleKey, priceIdentifier); }}
                disabled={verifyDisabled}
              >
                {verificationStatus === 'checking' ? 'Checking...' : 'Verify'}
              </button>
            </div>
            <div className="mt-2 flex items-center gap-2 text-xs">
              {verificationStatus === 'checking' && <span className="text-slate-300">Checking...</span>}
              {verificationStatus === 'ok' && <span className="text-emerald-300">{priceCheck?.message || 'Verified'}</span>}
              {verificationStatus === 'error' && <span className="text-amber-200">{priceCheck?.message || 'Verification failed'}</span>}
            </div>
            <p className="mt-1 text-xs text-slate-400">Use the Stripe price ID (starts with price_). Create a $0 price in Stripe for free tiers.</p>
          </FormField>

          <div className="grid gap-4 md:grid-cols-2">
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

          <div className="grid gap-4 md:grid-cols-2">
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

          <FormField label="Feature Bullets">
            <Textarea
              value={formState.values.featuresText}
              onChange={onPriceChange(bundleKey, priceIdentifier, 'featuresText')}
              rows={4}
              className={textareaClassName}
              placeholder={'One feature per line\nDesktop downloads included\nWhite-glove onboarding'}
            />
          </FormField>

          {/* Coupon assignment section */}
          {availableCoupons && onAssignCoupon && onUnassignCoupon && !demoPlan && (
            <FormField label="Intro Coupon">
              <div className="space-y-2">
                {assignedCoupon ? (
                  <div className="flex items-center gap-2 p-3 rounded-md bg-emerald-500/10 border border-emerald-500/30">
                    <Tag className="h-4 w-4 text-emerald-400" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-emerald-200 truncate">
                        {assignedCoupon.name || assignedCoupon.id}
                      </p>
                      <p className="text-xs text-emerald-300/70">
                        {getCouponSummary(assignedCoupon)}
                      </p>
                      {price.amount_cents > 0 && (
                        <p className="text-xs text-emerald-300 mt-0.5">
                          {formatDiscountPreview(price.amount_cents, assignedCoupon)}
                        </p>
                      )}
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      aria-label="Unassign intro coupon"
                      className="h-7 w-7 p-0 text-emerald-300 hover:text-white hover:bg-emerald-500/20"
                      onClick={() => { void onUnassignCoupon(priceIdentifier); }}
                      disabled={couponSaving}
                    >
                      {couponSaving ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <X className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                ) : (
                  <select
                    className={cn(inputClassName, 'cursor-pointer')}
                    value=""
                    onChange={(e) => {
                      if (e.target.value) {
                        void onAssignCoupon(priceIdentifier, e.target.value);
                      }
                    }}
                    disabled={couponSaving || availableCoupons.length === 0}
                  >
                    <option value="">
                      {availableCoupons.length === 0 ? 'No coupons available' : 'Select a coupon...'}
                    </option>
                    {availableCoupons.map((coupon) => (
                      <option key={coupon.id} value={coupon.id}>
                        {coupon.name || coupon.id} - {getCouponSummary(coupon)}
                      </option>
                    ))}
                  </select>
                )}
                <p className="text-xs text-slate-400">
                  Assign a Stripe coupon to apply during checkout for new customers on this plan.
                </p>
              </div>
            </FormField>
          )}

          {formState.error && (
            <p className="mt-3 text-sm text-rose-300">{formState.error}</p>
          )}

          <div className="mt-4 flex items-center gap-3">
            <Button
              type="button"
              onClick={() => { void onSavePrice(bundleKey, priceIdentifier); }}
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
        </CardContent>
      )}
    </Card>
  );
}
