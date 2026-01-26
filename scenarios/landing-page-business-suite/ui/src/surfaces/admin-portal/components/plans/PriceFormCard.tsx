import { RefreshCw } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import { Textarea } from '../../../../shared/ui/input';
import { FormField, inputClassName, textareaClassName } from '../FormField';
import { cn } from '../../../../shared/lib/utils';
import {
  normalizeInterval,
  getIntervalLabel,
  isPriceFormDirty,
  type PriceFormState,
  type PriceFormValues,
} from '../../services/pricing.service';
import type { PlanOption } from '../../../../shared/api';

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
}: PriceFormCardProps) {
  const dirty = isPriceFormDirty(formState);
  const demoPlan = formState.demo;

  return (
    <div className="py-4 first:pt-0 last:pb-0">
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
