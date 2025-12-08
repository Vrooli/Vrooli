import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { createCheckoutSession, getPlans, type PlanOption, type PricingOverview } from '../../../shared/api';

function formatCurrency(amount: number, currency = 'usd') {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
    maximumFractionDigits: 0,
  }).format(amount / 100);
}

function describePlan(plan?: PlanOption) {
  if (!plan) return '';
  const interval = plan.billing_interval === 'year' ? 'year' : 'month';
  const price = typeof plan.amount_cents === 'number' && plan.amount_cents > 0 ? formatCurrency(plan.amount_cents, plan.currency) : 'Custom';
  return `${price} / ${interval}`;
}

function buildDefaultURLs() {
  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  return {
    success: origin ? `${origin}/?checkout=success` : '/?checkout=success',
    cancel: typeof window !== 'undefined' ? window.location.href : '/checkout',
  };
}

export function CheckoutPage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const priceParam = params.get('price_id') || '';

  const [pricing, setPricing] = useState<PricingOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [attemptKey, setAttemptKey] = useState(0);
  const startedRef = useRef(false);

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const plans = await getPlans();
        if (mounted) {
          setPricing(plans);
        }
      } catch (err) {
        if (mounted) {
          setError(err instanceof Error ? err.message : 'Failed to load plans');
        }
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => {
      mounted = false;
    };
  }, []);

  const selectedPlan = useMemo<PlanOption | undefined>(() => {
    if (!pricing) return undefined;
    const candidates = [...(pricing.monthly || []), ...(pricing.yearly || [])].filter((plan) => plan.display_enabled);
    if (priceParam) {
      const match = candidates.find((plan) => plan.stripe_price_id === priceParam);
      if (match) return match;
    }
    return candidates[0];
  }, [pricing, priceParam]);

  useEffect(() => {
    let cancelled = false;
    const startCheckout = async () => {
      if (!selectedPlan || startedRef.current) return;
      startedRef.current = true;
      setSubmitting(true);
      setSessionError(null);
      try {
        const urls = buildDefaultURLs();
        const session = await createCheckoutSession({
          price_id: selectedPlan.stripe_price_id,
          success_url: urls.success,
          cancel_url: urls.cancel,
        });

        if (!cancelled && session?.url) {
          window.location.href = session.url;
          return;
        }
        if (!cancelled) {
          setSessionError('Stripe did not return a checkout URL. Try again or contact support.');
          startedRef.current = false;
        }
      } catch (err) {
        if (!cancelled) {
          setSessionError(err instanceof Error ? err.message : 'Failed to start checkout.');
          startedRef.current = false;
        }
      } finally {
        if (!cancelled) {
          setSubmitting(false);
        }
      }
    };

    startCheckout();

    return () => {
      cancelled = true;
    };
  }, [selectedPlan, attemptKey]);

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center text-white">
        <div className="animate-pulse text-lg">Loading pricing…</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center text-white">
        <Card className="max-w-xl bg-slate-900 border-slate-800 text-white">
          <CardHeader>
            <CardTitle>Unable to load checkout</CardTitle>
            <CardDescription className="text-slate-300">{error}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button variant="ghost" onClick={() => navigate('/')}>
              Back to landing
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-indigo-950 text-white">
      <div className="mx-auto max-w-5xl px-6 py-16 space-y-10">
        <div className="flex flex-col gap-3">
          <button
            type="button"
            onClick={() => navigate(-1)}
            className="self-start rounded-full border border-white/10 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300 hover:border-white/30"
          >
            Go back
          </button>
          <h1 className="text-4xl font-semibold leading-tight">Secure checkout</h1>
          <p className="text-lg text-slate-300">
            We&apos;re sending you to Stripe to finish payment. Stripe will collect your email during checkout.
          </p>
        </div>

        <div className="grid gap-6 lg:grid-cols-[1.25fr_1fr]">
          <Card className="bg-slate-900 border-slate-800 shadow-2xl shadow-indigo-900/30">
            <CardHeader>
              <CardTitle>Plan details</CardTitle>
              <CardDescription className="text-slate-300">
                {selectedPlan ? selectedPlan.plan_name : 'No plan selected'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {selectedPlan ? (
                <div className="space-y-4">
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm uppercase tracking-[0.3em] text-indigo-300">Plan</p>
                        <h2 className="text-2xl font-bold">{selectedPlan.plan_name}</h2>
                        <p className="text-slate-300">{selectedPlan.plan_tier.toUpperCase()}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-3xl font-bold">{describePlan(selectedPlan)}</p>
                        {selectedPlan.intro_enabled && selectedPlan.intro_amount_cents != null && (
                          <p className="text-xs text-emerald-300">
                            Intro {formatCurrency(selectedPlan.intro_amount_cents, selectedPlan.currency)} for{' '}
                            {selectedPlan.intro_periods || 1} month{selectedPlan.intro_periods === 1 ? '' : 's'}
                          </p>
                        )}
                      </div>
                    </div>
                    {Array.isArray(selectedPlan.metadata?.features) && selectedPlan.metadata?.features.length > 0 && (
                      <div className="mt-4 grid gap-2 sm:grid-cols-2">
                        {(selectedPlan.metadata?.features as string[]).map((feature) => (
                          <div key={feature} className="rounded-xl border border-white/5 bg-black/20 px-3 py-2 text-sm text-slate-200">
                            {feature}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                  <p className="text-sm text-slate-400">
                    Payments are processed by Stripe. You&apos;ll get a portal link to manage billing after checkout.
                  </p>
                </div>
              ) : (
                <p className="text-slate-300">No active plans are available right now.</p>
              )}
            </CardContent>
          </Card>

          <Card className="bg-slate-900 border-slate-800 shadow-2xl shadow-indigo-900/30">
            <CardHeader>
              <CardTitle>Redirecting to Stripe</CardTitle>
              <CardDescription className="text-slate-300">Sit tight — we&apos;re creating your checkout session.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {sessionError ? (
                  <>
                    <p className="text-sm text-rose-300">{sessionError}</p>
                    <Button
                      onClick={() => {
                        startedRef.current = false;
                        setSessionError(null);
                        setAttemptKey((key) => key + 1);
                      }}
                      className="w-full"
                      variant="default"
                    >
                      Retry checkout
                    </Button>
                  </>
                ) : (
                  <Button disabled className="w-full">
                    {submitting ? 'Redirecting…' : 'Preparing checkout…'}
                  </Button>
                )}
                <Button variant="ghost" onClick={() => navigate('/')}>Back to landing</Button>
              </div>
              <p className="pt-4 text-xs text-slate-500">
                By continuing you agree to the terms and acknowledge this subscription powers the Silent Founder OS suite.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
