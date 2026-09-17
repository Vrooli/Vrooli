import { useState } from 'react';
import type { Action, Block, ResolvedActions } from './types';
import type { OwnerPrice, ResolvedPricing } from './commerce';
import { formatOwnerPrice, yearlySavingsPercent } from './commerce';
import { ActionLink } from './primitives';
import { downloadSystemUi as ui } from './systemUi';

type ReadyEntry = { ref: string; plan: OwnerPrice };

function Check() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

function PlanCard({ entry, block, actions, reason, locale, savings }: {
  entry: ReadyEntry; block: Block<'pricing'>; actions?: ResolvedActions; reason: string; locale: string; savings: number;
}) {
  const { ref, plan } = entry;
  const configuredActions = block.content.actions.filter(action => action.kind === 'purchase' && action.plan_ref === ref);
  let money: string;
  try { money = formatOwnerPrice(plan, locale); } catch { return <article className="commerce-card" key={ref}><p role="status">{reason}</p></article>; }
  const title = plan.display.name || configuredActions[0]?.label || plan.plan_tier;
  return <article className="commerce-card" data-price-ref={ref} data-highlight={plan.display.highlight || undefined}>
    {plan.display.badge && <span className="commerce-badge">{plan.display.badge}</span>}
    <header className="commerce-card-head">
      <h3>{title}</h3>
      {plan.display.subtitle && <p className="commerce-subtitle">{plan.display.subtitle}</p>}
    </header>
    <p className="commerce-price"><span className="commerce-amount">{money}</span>{plan.billing_interval !== 'one_time' && <span className="commerce-interval">{ui.perInterval[plan.billing_interval]}</span>}</p>
    <p className="commerce-billing"><span>{ui.billing[plan.billing_interval]}</span>{savings > 0 && <span className="commerce-save">{ui.save} {savings}%</span>}</p>
    {plan.display.features.length > 0 && <ul className="commerce-features">{plan.display.features.map((feature, index) => <li key={index}><Check />{feature}</li>)}</ul>}
    {plan.intro_enabled && <p className="commerce-note">{ui.standardPrice}</p>}
    <div className="commerce-actions">{configuredActions.map((action, index) => <ActionLink key={index} action={action} resolvedActions={actions} reason={reason} className={plan.display.highlight ? 'button-primary' : 'button-secondary'} />)}</div>
  </article>;
}

/** No transport, auth, checkout effect or fabricated free plan. Safe in private preview. */
export function PricingCards({ block, prices = {}, actions, reason, locale }: {
  block: Block<'pricing'>; prices?: ResolvedPricing; actions?: ResolvedActions; reason: string; locale: string;
}) {
  const entries = block.content.plan_refs.map(ref => ({ ref, value: prices[ref] }));
  const ready: ReadyEntry[] = entries.flatMap(entry => entry.value?.status === 'ready' ? [{ ref: entry.ref, plan: entry.value.plan }] : []);
  const unavailable = entries.filter(entry => entry.value?.status !== 'ready');
  const monthly = ready.filter(entry => entry.plan.billing_interval === 'month');
  const yearly = ready.filter(entry => entry.plan.billing_interval === 'year');
  const oneTime = ready.filter(entry => entry.plan.billing_interval === 'one_time');
  const hasToggle = monthly.length > 0 && yearly.length > 0;
  const [interval, setInterval] = useState<'month' | 'year'>('month');
  const active = hasToggle ? (interval === 'month' ? monthly : yearly) : [...monthly, ...yearly];
  const monthlyByTier = new Map(monthly.map(entry => [entry.plan.plan_tier, entry.plan]));
  const savingsFor = (plan: OwnerPrice) => {
    if (plan.billing_interval !== 'year') return 0;
    const month = monthlyByTier.get(plan.plan_tier);
    return month ? yearlySavingsPercent(plan, month) : 0;
  };
  const bestSavings = Math.max(0, ...yearly.map(entry => savingsFor(entry.plan)));
  return <div className="commerce">
    {hasToggle && <div className="commerce-toggle" role="group" aria-label={ui.billingIntervalLabel}>
      <button type="button" aria-pressed={interval === 'month'} onClick={() => setInterval('month')}>{ui.monthly}</button>
      <button type="button" aria-pressed={interval === 'year'} onClick={() => setInterval('year')}>{ui.yearly}{bestSavings > 0 && <span className="commerce-save">{ui.save} {bestSavings}%</span>}</button>
    </div>}
    <div className="commerce-grid" data-plan-count={active.length}>
      {active.map(entry => <PlanCard key={entry.ref} entry={entry} block={block} actions={actions} reason={reason} locale={locale} savings={savingsFor(entry.plan)} />)}
      {unavailable.map(entry => <article className="commerce-card" key={entry.ref}><p role="status">{entry.value?.status === 'unavailable' ? entry.value.reason || reason : reason}</p></article>)}
    </div>
    {oneTime.length > 0 && <div className="commerce-topups">
      <p className="commerce-topups-title">{ui.topupsTitle}</p>
      <p className="commerce-note">{ui.topupsNote}</p>
      <div className="commerce-topups-row">{oneTime.map(entry => {
        const configuredActions = block.content.actions.filter((action: Action) => action.kind === 'purchase' && action.plan_ref === entry.ref);
        let money: string;
        try { money = formatOwnerPrice(entry.plan, locale); } catch { return <span className="commerce-topup" key={entry.ref}><span role="status">{reason}</span></span>; }
        return <span className="commerce-topup" key={entry.ref} data-price-ref={entry.ref}>
          <b>{money}</b>{entry.plan.display.subtitle && <small>{entry.plan.display.subtitle}</small>}
          {configuredActions.map((action, index) => <ActionLink key={index} action={action} resolvedActions={actions} reason={reason} className="button-secondary" />)}
        </span>;
      })}</div>
    </div>}
    {ready.length > 0 && <p className="commerce-trust">{ui.trust}</p>}
  </div>;
}
