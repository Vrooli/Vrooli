import { useEffect, useState } from 'react';
import { Link, Navigate, useNavigate } from 'react-router-dom';
import { useUserAuth } from '../../app/providers/useUserAuth';
import { getCreditInfo, getEntitlements } from '../../shared/api/account';
import { createBillingPortalSession } from '../../shared/api/billing';
import { getLandingConfig } from '../../shared/api/landing';
import type { CreditInfo, EntitlementPayload } from '../../shared/api/types';
import { decodeProductPresentation } from '../public-landing/presentation/decode';
import type { Spotlight } from '../public-landing/presentation/types';
import { SiteShell } from '../public-landing/site/SiteShell';
import './account-hub.css';

const copy = {
  title: 'Your account',
  eyebrow: 'Account',
  intro: 'Your plan, credits, apps, and security in one place.',
  plan: 'Plan',
  planNone: 'No subscription yet',
  planNoneDetail: 'A subscription unlocks every app in the suite, on desktop and web.',
  planError: 'We could not load your plan right now. Try again in a moment.',
  planLoading: 'Checking your plan…',
  seePlans: 'See plans',
  manageBilling: 'Manage billing',
  billingNote: 'Invoices, payment method, plan changes, and cancellation are handled in the secure Stripe billing portal.',
  billingFailed: 'The billing portal could not be opened. If this keeps happening, verify your identity under Security and try again.',
  billingCycle: (day: number) => `Billing cycle starts on day ${day} of each month.`,
  credits: 'Credits',
  creditsDetail: 'Credits cover metered AI usage across the suite. They never expire.',
  buyCredits: 'Buy credits',
  apps: 'Your apps',
  appsDetail: 'Every suite app is included with your account.',
  openApp: 'View',
  downloadApp: 'Download',
  security: 'Security',
  securityDetail: 'Review signed-in devices, add passkeys, and sign out other sessions.',
  securityOpen: 'Manage security',
  profile: 'Profile',
  signedInAs: 'Signed in as',
  verified: 'Verified',
  unverified: 'Unverified',
  signOut: 'Sign out',
  loading: 'Loading your account…',
  statuses: { active: 'Active', trialing: 'Trial', past_due: 'Past due', canceled: 'Canceled', inactive: 'Inactive' } as Record<string, string>,
};

function tierName(tier?: string): string {
  if (!tier) return '';
  return tier.charAt(0).toUpperCase() + tier.slice(1);
}

export function AccountHub() {
  const auth = useUserAuth();
  const navigate = useNavigate();
  const [entitlements, setEntitlements] = useState<EntitlementPayload | null>(null);
  const [entitlementsState, setEntitlementsState] = useState<'loading' | 'ready' | 'error'>('loading');
  const [portalBusy, setPortalBusy] = useState(false);
  const [message, setMessage] = useState('');
  const [spotlights, setSpotlights] = useState<Spotlight[]>([]);

  // The suite's public app list comes from the homepage document, read
  // without a visitor identity: account views never join experiments.
  useEffect(() => {
    if (!auth.isAuthenticated) return;
    let live = true;
    getLandingConfig(undefined, undefined, { route: '/' })
      .then(config => {
        if (!live || !config.presentation) return;
        try { setSpotlights(decodeProductPresentation(config.presentation).spotlights.filter(app => app.slug && app.detail_route)); } catch { /* apps list stays empty */ }
      })
      .catch(() => undefined);
    return () => { live = false; };
  }, [auth.isAuthenticated]);

  const [creditInfo, setCreditInfo] = useState<CreditInfo | null>(null);
  useEffect(() => {
    if (!auth.isAuthenticated) return;
    let live = true;
    setEntitlementsState('loading');
    getEntitlements()
      .then(value => { if (live) { setEntitlements(value); setEntitlementsState('ready'); } })
      .catch(() => { if (live) setEntitlementsState('error'); });
    // The credits owner carries the display label and multiplier; the
    // entitlement payload only proves a balance exists.
    getCreditInfo()
      .then(value => { if (live) setCreditInfo(value); })
      .catch(() => undefined);
    return () => { live = false; };
  }, [auth.isAuthenticated]);

  if (auth.isSessionLoading) {
    return <SiteShell meta={{ title: copy.title, description: copy.intro, noindex: true }}><p role="status">{copy.loading}</p></SiteShell>;
  }
  if (!auth.isAuthenticated) return <Navigate to="/auth/login?next=%2Faccount" replace />;

  const subscription = entitlements?.subscription;
  const hasPlan = subscription && ['active', 'trialing', 'past_due'].includes(subscription.status);
  const credits = creditInfo ?? entitlements?.credits ?? null;
  const creditBalance = credits ? (credits.balance_credits * credits.display_credits_multiplier) : null;
  const billingCycleStart = entitlements?.billing_cycle_start ?? 0;

  const openBillingPortal = async () => {
    setPortalBusy(true); setMessage('');
    try {
      const portal = await createBillingPortalSession(window.location.href);
      if (!portal.url) throw new Error('missing portal url');
      window.location.href = portal.url;
    } catch {
      setMessage(copy.billingFailed);
      setPortalBusy(false);
    }
  };

  return <SiteShell meta={{ title: copy.title, description: copy.intro, noindex: true }} width="wide">
    <header className="account-hub-header">
      <p className="account-hub-eyebrow">{copy.eyebrow}</p>
      <h1>{copy.title}</h1>
      <p className="account-hub-intro">{copy.intro}</p>
      {message && <p role="status" className="account-hub-status">{message}</p>}
    </header>
    <div className="account-hub-grid">
      <section className="account-card" aria-labelledby="account-plan-title">
        <h2 id="account-plan-title">{copy.plan}</h2>
        {entitlementsState === 'loading' && <p role="status">{copy.planLoading}</p>}
        {entitlementsState === 'error' && <p role="status">{copy.planError}</p>}
        {entitlementsState === 'ready' && (hasPlan
          ? <>
            <p className="account-plan-name">{tierName(subscription?.plan_tier) || 'Subscription'}
              <span className="account-chip" data-status={subscription?.status}>{copy.statuses[subscription?.status ?? ''] ?? subscription?.status}</span></p>
            {billingCycleStart > 0 && <p className="account-fine">{copy.billingCycle(billingCycleStart)}</p>}
            <p className="account-fine">{copy.billingNote}</p>
            <div className="account-card-actions">
              <button type="button" className="button button-primary" onClick={() => { void openBillingPortal(); }} disabled={portalBusy}>{copy.manageBilling}</button>
              <Link className="button button-secondary" to="/#plans">{copy.seePlans}</Link>
            </div>
          </>
          : <>
            <p className="account-plan-name">{copy.planNone}</p>
            <p className="account-fine">{copy.planNoneDetail}</p>
            <div className="account-card-actions">
              <Link className="button button-primary" to="/#plans">{copy.seePlans}</Link>
            </div>
          </>)}
      </section>
      <section className="account-card" aria-labelledby="account-credits-title">
        <h2 id="account-credits-title">{copy.credits}</h2>
        {entitlementsState === 'loading' && <p role="status">{copy.planLoading}</p>}
        {entitlementsState === 'error' && <p role="status">{copy.planError}</p>}
        {entitlementsState === 'ready' && <>
          <p className="account-credits-balance">{creditBalance === null ? '0' : creditBalance.toLocaleString()}<span> {credits?.display_credits_label ?? 'credits'}</span></p>
          <p className="account-fine">{copy.creditsDetail}</p>
          <div className="account-card-actions">
            <Link className="button button-secondary" to="/#plans">{copy.buyCredits}</Link>
          </div>
        </>}
      </section>
      <section className="account-card account-card-apps" aria-labelledby="account-apps-title">
        <h2 id="account-apps-title">{copy.apps}</h2>
        <p className="account-fine">{copy.appsDetail}</p>
        <ul className="account-app-list">
          {spotlights.map(app => <li key={app.app_key}>
            <div className="account-app-copy"><b>{app.name}</b>{app.tagline && <span>{app.tagline}</span>}</div>
            <div className="account-app-actions">
              <Link className="button button-quiet" to={app.detail_route}>{copy.openApp}</Link>
              <Link className="button button-secondary" to={`${app.detail_route}/download`}>{copy.downloadApp}</Link>
            </div>
          </li>)}
        </ul>
      </section>
      <section className="account-card" aria-labelledby="account-security-title">
        <h2 id="account-security-title">{copy.security}</h2>
        <p className="account-fine">{copy.securityDetail}</p>
        <div className="account-card-actions">
          <Link className="button button-secondary" to="/account/security">{copy.securityOpen}</Link>
        </div>
      </section>
      <section className="account-card" aria-labelledby="account-profile-title">
        <h2 id="account-profile-title">{copy.profile}</h2>
        <p className="account-fine">{copy.signedInAs}</p>
        <p className="account-profile-email">{auth.user?.email}
          <span className="account-chip" data-status={auth.user?.email_verified ? 'active' : 'inactive'}>{auth.user?.email_verified ? copy.verified : copy.unverified}</span></p>
        <div className="account-card-actions">
          <button type="button" className="button button-quiet" onClick={() => { void auth.logout().then(() => { navigate('/', { replace: true }); }); }}>{copy.signOut}</button>
        </div>
      </section>
    </div>
  </SiteShell>;
}

export default AccountHub;
