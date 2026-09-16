import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { AlertTriangle, ArrowLeft, ArrowRight, Check, Lock, RefreshCw, SearchX, Sparkles, WifiOff } from 'lucide-react';
import { SiteShell } from '../site/SiteShell';
import { createCheckoutSession, getPlans, isApiError, type PlanOption, type PricingOverview } from '../../../shared/api';
import { getAttributionContext } from '../../../shared/lib/attribution';

const SHELL_META = { title: 'Checkout', description: 'Secure checkout powered by Stripe.', noindex: true };

function formatCurrency(amount: number, currency = 'usd') {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
    maximumFractionDigits: 0,
  }).format(amount / 100);
}

function describePlan(plan?: PlanOption) {
  if (!plan) return '';
  const interval = plan.billing_interval === 'one_time' ? 'one-time' : plan.billing_interval === 'year' ? 'year' : 'month';
  const price = typeof plan.amount_cents === 'number' && plan.amount_cents > 0 ? formatCurrency(plan.amount_cents, plan.currency) : 'Custom';
  return interval === 'one-time' ? price : `${price} / ${interval}`;
}

function getPlanFeatures(plan: PlanOption): string[] {
  const metadata = plan.metadata as { features?: unknown } | undefined;
  return Array.isArray(metadata?.features)
    ? metadata.features.filter((feature): feature is string => typeof feature === 'string')
    : [];
}

function buildDefaultURLs() {
  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  return {
    success: origin ? `${origin}/thank-you?type=checkout` : '/thank-you?type=checkout',
    cancel: typeof window !== 'undefined' ? window.location.href : '/checkout',
  };
}

interface ErrorState {
  message: string;
  type: 'network' | 'server' | 'validation' | 'unknown';
  retryable: boolean;
}

function classifyErrorState(err: unknown): ErrorState {
  if (isApiError(err, 'network') || isApiError(err, 'timeout')) {
    return {
      message: 'Unable to connect. Please check your internet connection.',
      type: 'network',
      retryable: true,
    };
  }
  if (isApiError(err, 'server_error')) {
    return {
      message: err.userMessage,
      type: 'server',
      retryable: true,
    };
  }
  if (isApiError(err, 'rate_limited')) {
    return {
      message: err.userMessage,
      type: 'server',
      retryable: true,
    };
  }
  if (isApiError(err, 'validation')) {
    return {
      message: err.userMessage,
      type: 'validation',
      retryable: false,
    };
  }
  if (isApiError(err, 'not_found')) {
    return {
      message: err.userMessage,
      type: 'validation',
      retryable: false,
    };
  }
  const detail = err instanceof Error ? err.message : String(err ?? '');
  if (/stripe configuration|create checkout session/i.test(detail)) {
    return {
      message: 'Payments are temporarily unavailable. Please try again later.',
      type: 'server',
      retryable: true,
    };
  }
  return {
    message: err instanceof Error ? err.message : 'Something went wrong. Please try again.',
    type: 'unknown',
    retryable: true,
  };
}

export function CheckoutPage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const priceParam = params.get('price_id') || '';
  const planParam = params.get('plan')?.trim().toLowerCase() || '';
  const businessAccountId = params.get('business_account_id')?.trim() || undefined;
  const freeRequested = planParam === 'free';

  const [pricing, setPricing] = useState<PricingOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [sessionError, setSessionError] = useState<ErrorState | null>(null);
  const [attemptKey, setAttemptKey] = useState(0);
  const [loadAttemptKey, setLoadAttemptKey] = useState(0);
  const startedRef = useRef(false);
  const loadRequestRef = useRef(0);

  useEffect(() => {
    const requestId = ++loadRequestRef.current;
    const loadPlans = async () => {
      if (freeRequested) {
        setLoading(false);
        return;
      }
      setLoading(true);
      setError(null);
      try {
        const plans = await getPlans();
        if (loadRequestRef.current === requestId) {
          setPricing(plans);
        }
      } catch (err) {
        if (loadRequestRef.current === requestId) {
          setError(classifyErrorState(err));
        }
      } finally {
        if (loadRequestRef.current === requestId) setLoading(false);
      }
    };
    void loadPlans();
    return () => {
      if (loadRequestRef.current === requestId) {
        loadRequestRef.current += 1;
      }
    };
  }, [freeRequested, loadAttemptKey]);

  const selectedPlan = useMemo<PlanOption | undefined>(() => {
    if (!pricing) return undefined;
    const monthlyPlans = Array.isArray(pricing.monthly) ? pricing.monthly : [];
    const yearlyPlans = Array.isArray(pricing.yearly) ? pricing.yearly : [];
    const creditTopupPlans = Array.isArray(pricing.credit_topups) ? pricing.credit_topups : [];
    const candidates = [...monthlyPlans, ...yearlyPlans, ...creditTopupPlans].filter((plan) => plan.display_enabled);
    if (priceParam) {
      return candidates.find((plan) => plan.stripe_price_id === priceParam);
    }
    if (planParam) {
      return candidates.find((plan) => (
        plan.plan_tier.trim().toLowerCase() === planParam
        || plan.plan_name.trim().toLowerCase() === planParam
      ));
    }
    return candidates[0];
  }, [planParam, pricing, priceParam]);
  const selectedPlanFeatures = useMemo(
    () => (selectedPlan ? getPlanFeatures(selectedPlan) : []),
    [selectedPlan],
  );

  useEffect(() => {
    let cancelled = false;
    const startCheckout = async () => {
      if (freeRequested || !selectedPlan || startedRef.current) return;
      startedRef.current = true;
      setSubmitting(true);
      setSessionError(null);
      try {
        const urls = buildDefaultURLs();
        const session = await createCheckoutSession({
          price_id: selectedPlan.stripe_price_id,
          success_url: urls.success,
          cancel_url: urls.cancel,
          business_account_id: businessAccountId,
          attribution: getAttributionContext(new URLSearchParams(window.location.search).get('variant_slug') ?? new URLSearchParams(window.location.search).get('variant')),
        });

        if (!cancelled && session.url) {
          window.location.href = session.url;
          return;
        }
        if (!cancelled) {
          setSessionError({
            message: 'Stripe did not return a checkout URL. Try again or contact support.',
            type: 'server',
            retryable: true,
          });
          startedRef.current = false;
        }
      } catch (err) {
        if (!cancelled) {
          setSessionError(classifyErrorState(err));
          startedRef.current = false;
        }
      } finally {
        if (!cancelled) {
          setSubmitting(false);
        }
      }
    };

    void startCheckout();

    return () => {
      cancelled = true;
    };
  }, [attemptKey, businessAccountId, freeRequested, selectedPlan]);

  const back = <button type="button" className="button button-secondary" onClick={() => { navigate('/'); }}><ArrowLeft aria-hidden="true" />Back to home</button>;
  const status = (props: { icon: ReactNode; tone?: 'warning' | 'danger' | 'success'; title: string; body: ReactNode; actions: ReactNode }) => (
    <SiteShell meta={SHELL_META} width="narrow">
      <section className="site-card site-status-card" data-tone={props.tone}>
        <span className="site-status-mark" aria-hidden="true">{props.icon}</span>
        <h1>{props.title}</h1>
        <p className="site-lede">{props.body}</p>
        <div className="site-actions">{props.actions}</div>
      </section>
    </SiteShell>
  );

  if (loading) {
    return (
      <SiteShell meta={SHELL_META} width="narrow">
        <section className="site-card site-status-card" aria-busy="true">
          <span className="site-status-mark" aria-hidden="true"><span className="site-spinner" /></span>
          <h1>Loading pricing…</h1>
          <p className="site-lede">Getting the latest plan details.</p>
        </section>
      </SiteShell>
    );
  }

  if (freeRequested) {
    return status({
      icon: <Sparkles />, tone: 'success', title: 'Start with the free edition',
      body: 'Free access does not require payment. Continue to the landing page to choose a download.',
      actions: <><button type="button" className="button button-primary" onClick={() => { navigate('/#downloads-section'); }}>View free downloads<ArrowRight aria-hidden="true" /></button>{back}</>,
    });
  }

  if (pricing && (priceParam || planParam) && !selectedPlan) {
    return status({
      icon: <SearchX />, tone: 'danger', title: 'Plan unavailable',
      body: 'The requested plan is not available. Choose a current plan from the landing page and try again.',
      actions: back,
    });
  }

  if (error) {
    return status({
      icon: error.type === 'network' ? <WifiOff /> : <AlertTriangle />, tone: error.type === 'network' ? 'warning' : 'danger',
      title: error.type === 'network' ? 'Connection issue' : 'Unable to load checkout',
      body: error.message,
      actions: <>{error.retryable && <button type="button" className="button button-primary" onClick={() => { setLoadAttemptKey((k) => k + 1); }}><RefreshCw aria-hidden="true" />Try again</button>}{back}</>,
    });
  }

  return (
    <SiteShell meta={SHELL_META}>
      <div className="site-checkout">
        <header className="site-hero">
          <p className="eyebrow">Secure checkout</p>
          <h1>You&apos;re almost there.</h1>
          <p className="site-lede">
            We&apos;re sending you to Stripe to finish payment. Stripe will collect your email during checkout.
          </p>
        </header>

        <div className="site-checkout-grid">
          <section className="site-card site-checkout-plan" aria-label="Plan details">
            {selectedPlan ? (
              <>
                <div className="site-checkout-plan-head">
                  <div>
                    <p className="site-checkout-kicker">{selectedPlan.plan_tier.toUpperCase()}</p>
                    <h2>{selectedPlan.plan_name}</h2>
                  </div>
                  <div className="site-checkout-price">
                    <p>{describePlan(selectedPlan)}</p>
                    {selectedPlan.intro_enabled && selectedPlan.intro_amount_cents != null && (
                      <small>
                        Intro {formatCurrency(selectedPlan.intro_amount_cents, selectedPlan.currency)} for{' '}
                        {selectedPlan.intro_periods || 1} month{selectedPlan.intro_periods === 1 ? '' : 's'}
                      </small>
                    )}
                  </div>
                </div>
                {selectedPlanFeatures.length > 0 && (
                  <ul className="site-checkout-features">
                    {selectedPlanFeatures.map((feature) => (
                      <li key={feature}><Check aria-hidden="true" />{feature}</li>
                    ))}
                  </ul>
                )}
                <p className="site-checkout-note"><Lock aria-hidden="true" />Payments are processed by Stripe. You&apos;ll get a portal link to manage billing after checkout.</p>
              </>
            ) : (
              <p className="site-lede">No active plans are available right now.</p>
            )}
          </section>

          <section className="site-card site-checkout-status" aria-live="polite">
            {sessionError ? (
              <>
                <div className="site-alert" data-tone={sessionError.type === 'network' ? 'warning' : 'danger'} role="alert">
                  {sessionError.type === 'network' ? <WifiOff aria-hidden="true" /> : <AlertTriangle aria-hidden="true" />}
                  <div>
                    <p className="site-alert-title">{sessionError.type === 'network' ? 'Connection issue' : 'Checkout failed'}</p>
                    <p>{sessionError.message}</p>
                  </div>
                </div>
                {sessionError.retryable && (
                  <button
                    type="button"
                    className="button button-primary"
                    onClick={() => {
                      startedRef.current = false;
                      setSessionError(null);
                      setAttemptKey((key) => key + 1);
                    }}
                  >
                    <RefreshCw aria-hidden="true" />
                    Retry checkout
                  </button>
                )}
              </>
            ) : (
              <>
                <span className="site-status-mark" aria-hidden="true"><span className="site-spinner" /></span>
                <h2>Redirecting to Stripe</h2>
                <p className="site-checkout-note">Sit tight — we&apos;re creating your checkout session.</p>
                <button type="button" className="button button-primary" disabled>
                  {submitting ? 'Redirecting…' : 'Preparing checkout…'}
                </button>
              </>
            )}
            {back}
            <p className="site-checkout-fine">
              By continuing you agree to the <Link to="/terms">terms</Link> and acknowledge this subscription{pricing?.bundle.name.trim() ? ` is for ${pricing.bundle.name.trim()}` : ' is for the selected plan'}. See our <Link to="/privacy">privacy policy</Link>.
            </p>
          </section>
        </div>
      </div>
    </SiteShell>
  );
}
