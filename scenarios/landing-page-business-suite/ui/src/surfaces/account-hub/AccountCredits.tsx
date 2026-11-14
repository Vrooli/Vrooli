import { useEffect, useState } from 'react';
import { Link, Navigate } from 'react-router-dom';
import { useUserAuth } from '../../app/providers/useUserAuth';
import { apiGet } from '../../shared/api/common';
import { getPlans } from '../../shared/api/landing';
import { createCreditsCheckoutSession } from '../../shared/api/billing';
import type { PricingOverview } from '../../shared/api/types';
import { SiteShell } from '../public-landing/site/SiteShell';
import { Select } from '@vrooli/react-component-library/Select/1';
import { StatusBadge } from '@vrooli/react-component-library/StatusBadge/1';
import './account-hub.css';

type Account = { id: string; display_name: string; billing_email: string };
type Catalog = { accounts: Account[]; pricing: PricingOverview };
type State = { kind: 'loading' } | { kind: 'error'; message: string } | { kind: 'ready'; catalog: Catalog; account: string; checkout: boolean; message?: string };
type Summary = { limits: Record<string, number>; remaining: Record<string, number>; wallet: { available: number; on_hold: number; livemode: boolean; holds: { reason: string; credits: number }[]; history: { feature_id: string; credits: number; calls: number }[] } };
type Activity = { kind: 'loading'; account: string } | { kind: 'error'; account: string } | { kind: 'ready'; account: string; summary: Summary };

function CreditActivity({ account, pricing }: { account: string; pricing: PricingOverview }) {
  const [activity, setActivity] = useState<Activity>({ kind: 'loading', account });
  const [refresh, setRefresh] = useState(0);
  const [feature, setFeature] = useState('all');
  useEffect(() => {
    let active = true;
    setActivity({ kind: 'loading', account });
    void apiGet<Summary>(`/usage/business-account-summary?business_account_id=${encodeURIComponent(account)}`).then(summary => {
      if (active) setActivity({ kind: 'ready', account, summary });
    }).catch(() => { if (active) setActivity({ kind: 'error', account }); });
    return () => { active = false; };
  }, [account, refresh]);
  const amount = (value: number) => (value * pricing.bundle.display_credits_multiplier).toLocaleString(undefined, { maximumFractionDigits: 3 });
  const label = pricing.bundle.display_credits_label;
  const current = activity.account === account ? activity : null;
  if (!current || current.kind === 'loading') return <p role="status">Checking available credits…</p>;
  if (current.kind === 'error') return <div role="alert" className="account-hub-status">Your balance could not be loaded. <button type="button" className="button button-secondary" onClick={() => { setRefresh(value => value + 1); }}>Try again</button></div>;
  const { wallet, limits, remaining } = current.summary;
  const cap = limits.ai_credits;
  const capRemaining = remaining.ai_credits;
  const history = wallet.history.filter(item => feature === 'all' || item.feature_id === feature);
  return <>
    <div className="credits-toolbar"><StatusBadge tone={wallet.livemode ? 'neutral' : 'warning'}>{wallet.livemode ? 'Account balance' : 'Test mode'}</StatusBadge><button type="button" className="button button-quiet" onClick={() => { setRefresh(value => value + 1); }}>Refresh balance</button></div>
    <div className="account-hub-grid credits-balances">
      <section className="account-card" aria-labelledby="available-title"><h2 id="available-title">Available</h2><p className="account-credits-balance" data-testid="credits-available">{amount(wallet.available)} <span>{label}</span></p><p className="account-fine">Ready for your next AI request.</p></section>
      <section className="account-card" aria-labelledby="on-hold-title"><h2 id="on-hold-title">On hold</h2><p className="account-credits-balance" data-testid="credits-on-hold">{amount(wallet.on_hold)} <span>{label}</span></p>{wallet.holds.length ? <ul className="credits-holds">{wallet.holds.map(hold => <li key={hold.reason}><span>{hold.reason}</span><b>{amount(hold.credits)} {label}</b></li>)}</ul> : <p className="account-fine">No credits are currently held.</p>}</section>
      <section className="account-card" aria-labelledby="cap-title"><h2 id="cap-title">Monthly spending cap</h2><p className="account-credits-balance">{cap === undefined ? 'Unavailable' : cap < 0 ? 'No cap' : amount(cap)} <span>{cap === undefined || cap < 0 ? '' : label}</span></p><p className="account-fine">{capRemaining === undefined ? 'Your spending cap could not be determined.' : capRemaining < 0 ? 'Your plan has no monthly credit cap.' : `${amount(Math.max(0, capRemaining))} ${label} left under this cap, including requests in progress.`}</p></section>
    </div>
    <section className="account-card credits-history" aria-labelledby="history-title"><div className="credits-toolbar"><h2 id="history-title">History by feature</h2><Select aria-label="Filter history by feature" value={feature} onValueChange={setFeature} options={[{ value: 'all', label: 'All features' }, ...wallet.history.map(item => ({ value: item.feature_id, label: item.feature_id || 'Other usage' }))]} /></div>
      {history.length ? <div className="credits-table-wrap"><table><thead><tr><th scope="col">Feature</th><th scope="col">Requests</th><th scope="col">Credits used</th></tr></thead><tbody>{history.map(item => <tr key={item.feature_id}><th scope="row">{item.feature_id || 'Other usage'}</th><td>{item.calls.toLocaleString()}</td><td>{amount(item.credits)} {label}</td></tr>)}</tbody></table></div> : <p className="account-fine">No usage yet. AI requests will appear here after they settle.</p>}
    </section>
  </>;
}

export default function AccountCredits() {
  const auth = useUserAuth();
  const [state, setState] = useState<State>({ kind: 'loading' });
  useEffect(() => {
    if (!auth.isAuthenticated) return;
    let active = true;
    void Promise.all([apiGet<{ accounts: Account[] }>('/business-accounts'), getPlans()]).then(([result, pricing]) => {
      if (active) setState({ kind: 'ready', catalog: { accounts: result.accounts, pricing }, account: result.accounts[0]?.id ?? '', checkout: false });
    }).catch(() => { if (active) setState({ kind: 'error', message: 'We could not load your credits. Please try again.' }); });
    return () => { active = false; };
  }, [auth.isAuthenticated]);

  if (!auth.isSessionLoading && !auth.isAuthenticated) return <Navigate to="/auth/login?next=%2Faccount%2Fcredits" replace />;

  const buy = async (price: string) => {
    if (state.kind !== 'ready' || state.checkout) return;
    const account = state.catalog.accounts.find(item => item.id === state.account);
    if (!account) return;
    setState({ ...state, checkout: true, message: undefined });
    try {
      const session = await createCreditsCheckoutSession({ price_id: price, customer_email: account.billing_email, business_account_id: account.id, success_url: `${window.location.origin}/account/credits?checkout=complete`, cancel_url: `${window.location.origin}/account/credits?checkout=cancelled` });
      window.location.assign(session.url);
    } catch {
      setState({ ...state, checkout: false, message: 'Checkout could not be opened. Your credits have not changed. Please try again.' });
    }
  };

  return <SiteShell meta={{ title: 'Your credits', description: 'Manage credits and usage.', noindex: true }} width="wide">
    <header className="account-hub-header"><Link to="/account">Your account</Link><p className="account-hub-eyebrow">Credits</p><h1>Your credits</h1><p className="account-hub-intro">One balance for AI across your apps.</p></header>
    {(auth.isSessionLoading || state.kind === 'loading') && <p role="status">Loading your credits…</p>}
    {state.kind === 'error' && <p role="alert">{state.message}</p>}
    {state.kind === 'ready' && <>
      <label className="account-fine credits-account">Billing account <Select aria-label="Billing account" value={state.account} disabled={state.checkout} onValueChange={account => { setState({ ...state, account }); }} options={state.catalog.accounts.map(account => ({ value: account.id, label: account.display_name }))} /></label>
      {state.message && <p role="alert" className="account-hub-status">{state.message}</p>}
      {state.account && <CreditActivity key={state.account} account={state.account} pricing={state.catalog.pricing} />}
      <section className="account-card" aria-labelledby="add-credits-title"><h2 id="add-credits-title">Add credits</h2><p className="account-fine">Choose a top-up. Payment is completed securely with Stripe; credits arrive when payment is confirmed.</p>
        <div className="account-card-actions">{state.catalog.pricing.credit_topups?.filter(plan => plan.display_enabled).map(plan => <button className="button button-primary" key={plan.stripe_price_id} type="button" disabled={state.checkout || !state.account} onClick={() => { void buy(plan.stripe_price_id); }}>{plan.plan_name} · {new Intl.NumberFormat(undefined, { style: 'currency', currency: plan.currency }).format(plan.amount_cents / 100)}</button>)}</div>
      </section>
    </>}
  </SiteShell>;
}
