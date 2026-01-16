import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { WifiOff, RefreshCw, AlertTriangle } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { createCheckoutSession, getPlans, isApiError, type PlanOption, type PricingOverview } from '../../../shared/api';

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
      message: 'Our servers are experiencing issues. Please try again later.',
      type: 'server',
      retryable: true,
    };
  }
  if (isApiError(err, 'validation')) {
    return {
      message: isApiError(err) ? err.userMessage : 'Invalid request. Please try again.',
      type: 'validation',
      retryable: false,
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

  const [pricing, setPricing] = useState<PricingOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [sessionError, setSessionError] = useState<ErrorState | null>(null);
  const [attemptKey, setAttemptKey] = useState(0);
  const [loadAttemptKey, setLoadAttemptKey] = useState(0);
  const startedRef = useRef(false);

  useEffect(() => {
    let mounted = true;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const plans = await getPlans();
        if (mounted) {
          setPricing(plans);
        }
      } catch (err) {
        if (mounted) {
          setError(classifyErrorState(err));
        }
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => {
      mounted = false;
    };
  }, [loadAttemptKey]);

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
    const ErrorIcon = error.type === 'network' ? WifiOff : AlertTriangle;
    const borderColor = error.type === 'network' ? 'border-amber-500/30' : 'border-rose-500/30';
    const iconColor = error.type === 'network' ? 'text-amber-400' : 'text-rose-400';
    const bgColor = error.type === 'network' ? 'bg-amber-500/10' : 'bg-rose-500/10';

    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center text-white px-6">
        <Card className={`max-w-xl bg-slate-900 ${borderColor} text-white`}>
          <CardHeader>
            <div className={`w-12 h-12 rounded-full ${bgColor} flex items-center justify-center mb-4`}>
              <ErrorIcon className={`w-6 h-6 ${iconColor}`} />
            </div>
            <CardTitle>
              {error.type === 'network' ? 'Connection Issue' : 'Unable to Load Checkout'}
            </CardTitle>
            <CardDescription className="text-slate-300">{error.message}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 sm:flex-row">
            {error.retryable && (
              <Button
                onClick={() => setLoadAttemptKey((k) => k + 1)}
                className="gap-2"
              >
                <RefreshCw className="w-4 h-4" />
                Try Again
              </Button>
            )}
            <Button variant="ghost" onClick={() => navigate('/')}>
              Back to Landing
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
                  <div className="space-y-4">
                    <div className={`rounded-lg border p-4 ${
                      sessionError.type === 'network'
                        ? 'border-amber-500/20 bg-amber-500/10'
                        : 'border-rose-500/20 bg-rose-500/10'
                    }`}>
                      <div className="flex items-start gap-3">
                        {sessionError.type === 'network' ? (
                          <WifiOff className="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
                        ) : (
                          <AlertTriangle className="w-5 h-5 text-rose-400 flex-shrink-0 mt-0.5" />
                        )}
                        <div>
                          <p className={`text-sm font-medium ${
                            sessionError.type === 'network' ? 'text-amber-300' : 'text-rose-300'
                          }`}>
                            {sessionError.type === 'network' ? 'Connection Issue' : 'Checkout Failed'}
                          </p>
                          <p className={`text-sm mt-1 ${
                            sessionError.type === 'network' ? 'text-amber-200/80' : 'text-rose-200/80'
                          }`}>
                            {sessionError.message}
                          </p>
                        </div>
                      </div>
                    </div>
                    {sessionError.retryable && (
                      <Button
                        onClick={() => {
                          startedRef.current = false;
                          setSessionError(null);
                          setAttemptKey((key) => key + 1);
                        }}
                        className="w-full gap-2"
                        variant="default"
                      >
                        <RefreshCw className="w-4 h-4" />
                        Retry Checkout
                      </Button>
                    )}
                  </div>
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
