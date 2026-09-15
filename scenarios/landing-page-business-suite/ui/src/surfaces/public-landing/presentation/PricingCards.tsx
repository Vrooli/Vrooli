import type { Block, ResolvedActions } from './types';
import type { ResolvedPricing } from './commerce';
import { formatOwnerPrice } from './commerce';
import { ActionLink } from './primitives';
import { downloadSystemUi as ui } from './systemUi';

/** No transport, auth, checkout effect or fabricated free plan. Safe in private preview. */
export function PricingCards({ block, prices = {}, actions, reason, locale }: {
  block: Block<'pricing'>; prices?: ResolvedPricing; actions?: ResolvedActions; reason: string; locale: string;
}) {
  return <div className="commerce-grid">{block.content.plan_refs.map(ref => {
    const value = prices[ref];
    if (value?.status !== 'ready') return <article className="commerce-card" key={ref}><p role="status">{value?.reason || reason}</p></article>;
    const plan = value.plan;
    const configuredActions = block.content.actions.filter(action => action.kind === 'purchase' && action.plan_ref === ref);
    let money: string;
    try { money = formatOwnerPrice(plan, locale); } catch { return <article className="commerce-card" key={ref}><p role="status">{reason}</p></article>; }
    return <article className="commerce-card" key={ref} data-price-ref={ref}>
      {configuredActions.map((action, index) => <h3 key={index}>{action.label}</h3>)}<p className="commerce-price">{money}</p><p>{ui.billing[plan.billing_interval]}</p>
      {plan.intro_enabled && <p className="commerce-note">{ui.standardPrice}</p>}
      <div className="commerce-actions">{configuredActions.map((action, index) => <ActionLink key={index} action={action} resolvedActions={actions} reason={reason} />)}</div>
    </article>;
  })}</div>;
}
