import { formatOwnerPrice } from '../../../public-landing/presentation/commerce';
import { isDemoPlanOption } from '../../../../shared/lib/pricingPlaceholders';
import { getIntervalLabel, normalizeInterval, type PricingPreviewData } from '../../services/pricing.service';

interface PlanPreviewProps { data: PricingPreviewData }

/** Private owner-facts preview. No alternate marketing renderer or transaction effects. */
export function PlanPreview({ data }: PlanPreviewProps) {
  const plans = [...data.overview.monthly, ...data.overview.yearly].filter(plan => plan.display_enabled && !isDemoPlanOption(plan));
  return <section className="rounded-xl border border-white/10 bg-slate-950/60 p-5" aria-label="Pricing owner preview">
    <h2 className="text-sm font-semibold text-white">Pricing preview</h2>
    <p className="mt-1 text-xs text-slate-400">Configured owner prices with unsaved form edits. Public presentation copy and publication are managed separately.</p>
    {plans.length === 0 ? <p className="mt-4 text-sm text-slate-300" role="status">No enabled configured plans to preview.</p> :
      <div className="mt-4 grid gap-4 md:grid-cols-2">{plans.map((plan, index) => {
        let money: string;
        try { money = formatOwnerPrice(plan, 'en'); } catch { money = 'Price unavailable'; }
        return <article key={plan.stripe_price_id || index} className="min-w-0 rounded-lg border border-white/10 p-4">
          <h3 className="break-words font-semibold">{plan.plan_name}</h3>
          <p className="mt-2 text-xl">{money}</p><p className="text-sm text-slate-400">{getIntervalLabel(normalizeInterval(plan.billing_interval))}</p>
          <p className="mt-2 break-all font-mono text-xs text-slate-400">{plan.stripe_price_id}</p>
          {typeof plan.metadata?.subtitle === 'string' && <p className="mt-3">{plan.metadata.subtitle}</p>}
          {typeof plan.metadata?.badge === 'string' && <p className="mt-2 text-sm">{plan.metadata.badge}</p>}
          {Array.isArray(plan.metadata?.features) && <ul className="mt-3 list-inside list-disc text-sm">{plan.metadata.features.filter((feature): feature is string => typeof feature === 'string').map((feature, i) => <li key={i}>{feature}</li>)}</ul>}
          {typeof plan.metadata?.cta_label === 'string' && <p className="mt-3 text-sm text-slate-400">Configured label: {plan.metadata.cta_label}</p>}
        </article>;
      })}</div>}
  </section>;
}
