import { Eye, EyeOff, Star } from 'lucide-react';
import { cn } from '../../../../shared/lib/utils';
import { normalizeInterval, getIntervalLabel, type PriceFormState } from '../../services/pricing.service';
import type { PlanOption } from '../../../../shared/api';

export interface PriceReadOnlyCardProps {
  price: PlanOption;
  formState: PriceFormState;
}

export function PriceReadOnlyCard({ price, formState }: PriceReadOnlyCardProps) {
  const demoPlan = formState.demo;
  const isEnabled = formState.values.displayEnabled;
  const isHighlighted = formState.values.highlight;

  return (
    <div
      className={cn(
        'flex items-center justify-between py-4 first:pt-0 last:pb-0',
        !isEnabled && 'opacity-60'
      )}
    >
      <div className="flex items-center gap-3 min-w-0">
        <div
          className={cn(
            'flex h-8 w-8 items-center justify-center rounded-lg',
            isEnabled ? 'bg-emerald-500/10' : 'bg-slate-500/10'
          )}
        >
          {isEnabled ? (
            <Eye className="h-4 w-4 text-emerald-400" />
          ) : (
            <EyeOff className="h-4 w-4 text-slate-400" />
          )}
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-white truncate">{price.plan_name}</span>
            {isHighlighted && (
              <Star className="h-3.5 w-3.5 text-amber-400 fill-amber-400 flex-shrink-0" />
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
            <span>{getIntervalLabel(normalizeInterval(price.billing_interval))}</span>
            <span>·</span>
            <span>{price.currency.toUpperCase()}</span>
            {price.stripe_price_id && (
              <>
                <span>·</span>
                <span className="font-mono text-slate-500 truncate max-w-[120px]">
                  {price.stripe_price_id}
                </span>
              </>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0">
        <span
          className={cn(
            'inline-flex items-center rounded-full px-2 py-0.5 text-xs',
            demoPlan
              ? 'border border-amber-500/50 bg-amber-500/10 text-amber-200'
              : isEnabled
                ? 'bg-emerald-500/10 text-emerald-200'
                : 'bg-slate-500/10 text-slate-400'
          )}
        >
          {demoPlan ? 'Demo' : isEnabled ? 'Visible' : 'Hidden'}
        </span>
      </div>
    </div>
  );
}
