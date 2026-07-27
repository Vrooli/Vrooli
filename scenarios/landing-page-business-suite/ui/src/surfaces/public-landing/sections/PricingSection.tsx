import { memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Check } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetricsHook';
import { createCheckoutSession, type PlanOption, type PricingOverview, type StripeCoupon } from '../../../shared/api';
import { isDemoPlanOption } from '../../../shared/lib/pricingPlaceholders';
import { normalizeInterval } from '../../../shared/lib/pricingIntervals';
import { getFallbackLandingConfig } from '../../../shared/lib/fallbackLandingConfig';
import { formatDiscountBadge, getCouponSummary } from '../../../shared/lib/pricingCalculations';

interface PricingTier {
  name: string;
  price: string;
  description: string;
  features: string[];
  cta_text?: string;
  cta_url?: string;
  highlighted?: boolean;
  badge?: string;
  subtitle?: string;
}

interface PricingSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    tiers?: PricingTier[];
  };
  pricingOverview?: PricingOverview;
  /** Optional coupon mappings (priceID -> couponID) for showing discount badges */
  couponMappings?: Record<string, string>;
  /** Optional available coupons for looking up coupon details */
  availableCoupons?: StripeCoupon[];
}

function formatCurrency(amount: number, currency = 'usd') {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
    maximumFractionDigits: 0,
  }).format(amount / 100);
}

function formatCredits(amount: number, multiplier: number, label: string) {
  const value = Math.round(amount * multiplier);
  if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(1).replace(/\.0$/, '')}M ${label}`;
  }
  if (value >= 1_000) {
    return `${(value / 1_000).toFixed(1).replace(/\.0$/, '')}k ${label}`;
  }
  return `${String(value)} ${label}`;
}

function ensureFreeTier(pricing: PricingOverview | null, fallbackPricing: PricingOverview | null): PricingOverview | null {
  if (!pricing) return fallbackPricing;
  const monthlyPlans = Array.isArray(pricing.monthly) ? pricing.monthly : [];
  const yearlyPlans = Array.isArray(pricing.yearly) ? pricing.yearly : [];
  const hasFree =
    [...monthlyPlans, ...yearlyPlans].some(
      (p) => typeof p.plan_tier === 'string' && p.plan_tier.toLowerCase() === 'free',
    );
  if (hasFree) return pricing;
  const fallbackMonthlyPlans = fallbackPricing && Array.isArray(fallbackPricing.monthly)
    ? fallbackPricing.monthly
    : [];
  const fallbackYearlyPlans = fallbackPricing && Array.isArray(fallbackPricing.yearly)
    ? fallbackPricing.yearly
    : [];
  const fallbackFree =
    [...fallbackMonthlyPlans, ...fallbackYearlyPlans].find(
      (p) => typeof p.plan_tier === 'string' && p.plan_tier.toLowerCase() === 'free',
    );
  if (!fallbackFree) return pricing;
  const merged: PricingOverview = {
    ...pricing,
    monthly: [fallbackFree, ...monthlyPlans],
    yearly: yearlyPlans,
  };
  return merged;
}

interface BuildTierOptions {
  option: PlanOption;
  bundle: PricingOverview['bundle'];
  fallbackHighlight: boolean;
  interval: 'month' | 'year';
  /** Optional coupon assigned to this plan for discount badge display */
  assignedCoupon?: StripeCoupon;
}

function buildTierFromPlan({ option, bundle, fallbackHighlight, interval, assignedCoupon }: BuildTierOptions) {
  const amount = typeof option.amount_cents === 'number' ? option.amount_cents : 0;
  const hasAmount = amount > 0;
  const isFree = amount === 0;
  const priceLabel = hasAmount ? formatCurrency(amount, option.currency) : isFree ? 'Free' : 'Custom';
  const introAmount = option.intro_amount_cents ?? 0;
  const hasIntroAmount = option.intro_amount_cents != null;
  const metadata = option.metadata || {};
  const metaFeatures = Array.isArray(metadata.features) ? (metadata.features) : [];
  const creditsLabel = bundle.display_credits_label || 'credits';
  const badgeOverride = typeof metadata.badge === 'string' ? (metadata.badge) : undefined;
  const bonusLabel =
    typeof option.bonus_type === 'string' && option.bonus_type.trim().toLowerCase() !== 'none'
      ? option.bonus_type.replace('_', ' ')
      : undefined;

  // Determine intro pricing badge - prioritize assigned coupon, then plan's intro settings
  let introBadge: string | undefined;
  if (assignedCoupon && hasAmount) {
    // Use shared utility to format coupon discount badge
    introBadge = formatDiscountBadge(amount, assignedCoupon, interval);
  } else if (option.intro_enabled && hasIntroAmount) {
    // Fall back to plan's native intro pricing config
    introBadge = `${formatCurrency(introAmount, option.currency)} intro for ${String(option.intro_periods || 1)} month${option.intro_periods === 1 ? '' : 's'}`;
  }

  const badge = badgeOverride || introBadge || bonusLabel;

  const features = [
    `${formatCredits(option.monthly_included_credits, bundle.display_credits_multiplier, creditsLabel)} included`,
    ...(option.one_time_bonus_credits > 0
      ? [
          `Bonus ${formatCredits(option.one_time_bonus_credits, bundle.display_credits_multiplier, creditsLabel)}`,
        ]
      : []),
    ...metaFeatures,
  ];

  const subtitle =
    typeof metadata.subtitle === 'string' && metadata.subtitle.trim().length > 0
      ? (metadata.subtitle)
      : Number.isFinite(option.plan_rank)
        ? `Plan rank #${String(option.plan_rank)}`
        : 'Flexible access';

  // Determine CTA text - prioritize assigned coupon info
  let ctaText: string;
  if (typeof metadata.cta_label === 'string' && metadata.cta_label.trim().length > 0) {
    ctaText = metadata.cta_label;
  } else if (assignedCoupon && hasAmount) {
    // Show coupon-aware CTA
    ctaText = `Start with ${getCouponSummary(assignedCoupon)}`;
  } else if (option.intro_enabled && hasIntroAmount) {
    ctaText = `Start ${formatCurrency(introAmount, option.currency)} intro`;
  } else {
    ctaText = 'Choose plan';
  }

  const highlighted = metadata.highlight === true ? true : fallbackHighlight;
  const directDownloadCTA = isFree || option.kind === 'supporter_contribution';
  const downloadHref = '#downloads-section';

  return {
    name: option.plan_name,
    description: option.plan_tier.charAt(0).toUpperCase() + option.plan_tier.slice(1),
    price: hasAmount
      ? `${priceLabel} / ${interval === 'month' ? 'month' : 'year'}`
      : isFree
        ? 'Free'
        : 'Contact sales',
    features,
    cta_text: directDownloadCTA ? 'Download' : ctaText,
    cta_url: directDownloadCTA ? downloadHref : option.stripe_price_id ? `/checkout?price_id=${option.stripe_price_id}` : undefined,
    highlighted,
    badge,
    subtitle,
  };
}

function getTierFeatures(tier: PricingTier): string[] {
  if (Array.isArray(tier.features)) {
    return tier.features;
  }
  return [];
}

function resolvePriceId(ctaUrl?: string) {
  if (!ctaUrl) return null;
  try {
    const baseUrl = typeof window !== 'undefined' ? window.location.origin : 'https://example.com';
    const url = new URL(ctaUrl, baseUrl);
    return url.searchParams.get('price_id');
  } catch {
    return null;
  }
}

const DEFAULT_FALLBACK_TIERS: PricingTier[] = [
  {
    name: 'Solo',
    price: '$39',
    description: 'Ship silently with Vrooli Ascension',
    features: [
      'Vrooli Ascension desktop + updates',
      'Unlimited workflows & retries',
      'Auto screen-recording exports',
      'Email support',
    ],
    cta_text: 'Start with Ascension',
    cta_url: '/checkout?plan=solo',
    highlighted: false,
    badge: 'Founder-friendly',
  },
  {
    name: 'Studio',
    price: '$119',
    description: 'Add marketing polish and concierge setup',
    features: [
      'Everything in Solo',
      'Motion presets & branding overlays',
      'Concierge workflow setup',
      'Priority support',
    ],
    cta_text: 'Book a setup call',
    cta_url: '/book-setup',
    highlighted: true,
    badge: 'Most popular',
  },
  {
    name: 'Team',
    price: '$189',
    description: 'Seat-based access with shared billing and audit trails',
    features: [
      'Everything in Studio',
      'Seat-based entitlements for installers',
      'Shared analytics and usage caps',
      'Email + chat support',
    ],
    cta_text: 'Start team trial',
    cta_url: '/checkout?plan=team',
    badge: 'New',
  },
  {
    name: 'Founder OS',
    price: 'Custom',
    description: 'Bundle future Vrooli business apps as they launch',
    features: [
      'Studio benefits',
      'Early access to new apps',
      'Shared analytics & credits',
      'Coaching with the creator',
    ],
    cta_text: 'Join the waitlist',
    cta_url: '/waitlist',
  },
];
const EMPTY_PLAN_OPTIONS: PlanOption[] = [];

interface PricingTierCardProps {
  tier: PricingTier;
  index: number;
  redirectingPrice: string | null;
  onSelect: (tier: PricingTier) => void;
}

const PricingTierCard = memo(function PricingTierCard({
  tier,
  index,
  redirectingPrice,
  onSelect,
}: PricingTierCardProps) {
  const highlight = tier.highlighted;
  const priceId = resolvePriceId(tier.cta_url);
  const loading = priceId !== null && redirectingPrice === priceId;

  return (
    <div
      className={`relative h-full overflow-visible rounded-3xl border p-8 pt-10 transition-all duration-300 ${
        highlight
          ? 'border-accent/50 bg-gradient-to-b from-surface-primary via-surface-deep to-surface-darker text-white shadow-[0_30px_80px_-40px_rgba(var(--color-accent),0.45)]'
          : 'border-slate-200 bg-white text-slate-900 hover:-translate-y-1 shadow-[0_20px_60px_-48px_rgba(0,0,0,0.4)]'
      }`}
      data-testid={`pricing-tier-${String(index)}`}
    >
      {highlight && <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(var(--color-accent),0.12),transparent_32%)]" />}
      {tier.badge && (
        <div
          className={`absolute left-1/2 top-0 -translate-x-1/2 -translate-y-1/2 rounded-full px-4 py-1 text-sm font-semibold ${
            highlight
              ? 'bg-surface-primary text-white ring-2 ring-accent/40'
              : 'bg-surface-primary text-white'
          }`}
          style={{ zIndex: 5 }}
        >
          {tier.badge}
        </div>
      )}

      <div className="space-y-6">
        <div className="space-y-2">
          <h3 className={`text-2xl font-bold ${highlight ? 'text-white' : 'text-slate-900'}`}>{tier.name}</h3>
          {tier.subtitle && (
            <p className={`text-xs uppercase tracking-[0.3em] ${highlight ? 'text-accent' : 'text-slate-500'}`}>
              {tier.subtitle}
            </p>
          )}
          <p className={`${highlight ? 'text-slate-200' : 'text-slate-600'} text-sm`}>{tier.description}</p>
          <div className="flex items-baseline gap-2">
            <span className={`text-5xl font-bold ${highlight ? 'text-white' : 'text-slate-900'}`}>{tier.price}</span>
          </div>
        </div>

        <Button
          variant={highlight ? 'default' : 'ghost'}
          className={`w-full ${highlight ? '' : 'bg-slate-900/5 text-slate-900 hover:bg-slate-900/10'}`}
          size="lg"
          onClick={() => { onSelect(tier); }}
          disabled={loading}
          data-testid={`pricing-cta-${tier.name.toLowerCase().replace(/\s+/g, '-')}`}
        >
          {loading ? 'Redirecting...' : tier.cta_text || 'Get Started'}
        </Button>

        <ul className="space-y-3">
          {getTierFeatures(tier).map((feature, featureIndex) => (
            <li key={`${feature}-${String(featureIndex)}`} className="flex items-start gap-3">
              <Check className={`w-5 h-5 flex-shrink-0 mt-0.5 ${highlight ? 'text-success' : 'text-accent'}`} />
              <span className={highlight ? 'text-slate-200' : 'text-slate-600'}>{feature}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
});

export function PricingSection({ content, pricingOverview, couponMappings, availableCoupons }: PricingSectionProps) {
  const { trackCTAClick } = useMetrics();
  const [activeInterval, setActiveInterval] = useState<'monthly' | 'yearly'>('monthly');
  const [stickyDismissed, setStickyDismissed] = useState(false);
  const [redirectingPrice, setRedirectingPrice] = useState<string | null>(null);
  const [sessionError, setSessionError] = useState<string | null>(null);

  const fallbackPricing = useMemo(() => getFallbackLandingConfig().pricing, []);
  const pricing = useMemo(() => ensureFreeTier(pricingOverview ?? null, fallbackPricing ?? null), [pricingOverview, fallbackPricing]);

  const buildDefaultURLs = useMemo(() => {
    const origin = typeof window !== 'undefined' ? window.location.origin : '';
    return {
      success: origin ? `${origin}/?checkout=success` : '/?checkout=success',
      cancel: typeof window !== 'undefined' ? window.location.href : '/checkout',
    };
  }, []);

  const handleTierSelect = useCallback(async (tier: PricingTier) => {
    if (!tier.cta_url) return;
    const priceId = resolvePriceId(tier.cta_url);
    const metricKey = `pricing-${tier.name.toLowerCase().replace(/\s+/g, '-')}-cta`;
    trackCTAClick(metricKey, {
      tier: tier.name,
      price: tier.price,
    });

    if (!priceId) {
      window.location.href = tier.cta_url;
      return;
    }

    setSessionError(null);
    setRedirectingPrice(priceId);
    try {
      const session = await createCheckoutSession({
        price_id: priceId,
        success_url: buildDefaultURLs.success,
        cancel_url: buildDefaultURLs.cancel,
      });
      if (session.url) {
        window.location.href = session.url;
        return;
      }
      setSessionError('Stripe did not return a checkout URL. Try again.');
    } catch (err) {
      setSessionError(err instanceof Error ? err.message : 'Failed to start checkout.');
    } finally {
      setRedirectingPrice(null);
    }
  }, [buildDefaultURLs.cancel, buildDefaultURLs.success, trackCTAClick]);

  const bundle = pricing?.bundle;
  const monthlyPlansRaw = pricing?.monthly ?? EMPTY_PLAN_OPTIONS;
  const yearlyPlansRaw = pricing?.yearly ?? EMPTY_PLAN_OPTIONS;
  const monthlyPlans = useMemo(
    () =>
      bundle
        ? monthlyPlansRaw.filter(
            (plan) =>
              !isDemoPlanOption(plan) &&
              (normalizeInterval(plan.billing_interval) === 'month' ||
                (typeof plan.plan_tier === 'string' && plan.plan_tier.toLowerCase() === 'free')) &&
              typeof plan.amount_cents === 'number' &&
              plan.amount_cents >= 0,
          )
        : [],
    [bundle, monthlyPlansRaw],
  );
  const yearlyPlans = useMemo(
    () =>
      bundle
        ? yearlyPlansRaw.filter(
            (plan) =>
              !isDemoPlanOption(plan) &&
              normalizeInterval(plan.billing_interval) === 'year' &&
              typeof plan.amount_cents === 'number' &&
              plan.amount_cents >= 0,
          )
        : [],
    [bundle, yearlyPlansRaw],
  );

  const sortByAmount = useCallback((plans: PlanOption[]) =>
    [...plans].sort((a, b) => {
      const aHas = typeof a.amount_cents === 'number' && a.amount_cents > 0;
      const bHas = typeof b.amount_cents === 'number' && b.amount_cents > 0;
      if (!aHas && !bHas) return 0;
      if (!aHas) return 1;
      if (!bHas) return -1;
      return a.amount_cents - b.amount_cents;
    }), []);

  // Helper to find assigned coupon for a plan
  const findAssignedCoupon = useCallback((priceId: string): StripeCoupon | undefined => {
    if (!couponMappings || !availableCoupons) return undefined;
    const couponId = couponMappings[priceId];
    if (!couponId) return undefined;
    return availableCoupons.find((c) => c.id === couponId);
  }, [availableCoupons, couponMappings]);

  const monthlyTiers = useMemo(
    () =>
      bundle && monthlyPlans.length > 0
        ? sortByAmount(monthlyPlans).map((option, index) =>
            buildTierFromPlan({
              option,
              bundle,
              fallbackHighlight: index === 0,
              interval: 'month',
              assignedCoupon: findAssignedCoupon(option.stripe_price_id),
            })
          )
        : [],
    [bundle, findAssignedCoupon, monthlyPlans, sortByAmount],
  );
  const yearlyTiers = useMemo(
    () =>
      bundle && yearlyPlans.length > 0
        ? sortByAmount(yearlyPlans).map((option, index) =>
            buildTierFromPlan({
              option,
              bundle,
              fallbackHighlight: index === 0,
              interval: 'year',
              assignedCoupon: findAssignedCoupon(option.stripe_price_id),
            })
          )
        : [],
    [bundle, findAssignedCoupon, sortByAmount, yearlyPlans],
  );

  const freeTier = useMemo<PricingTier>(
    () => ({
      name: 'Free Monthly',
      price: 'Free',
      description: '50 runs/month with builder and watermark exports',
      features: ['50 runs/month', 'Builder + replay viewer (watermark)', 'Email support'],
      cta_text: 'Download',
      cta_url: '#downloads-section',
      highlighted: monthlyTiers.length === 0,
      badge: 'Start free',
      subtitle: 'Free',
    }),
    [monthlyTiers.length],
  );

  const monthlyWithFree = useMemo(
    () =>
      monthlyTiers.length > 0 && monthlyTiers.some((tier) => tier.name.toLowerCase().includes('free'))
        ? monthlyTiers
        : [freeTier, ...monthlyTiers],
    [freeTier, monthlyTiers],
  );

  const fallbackTiers = useMemo(
    () =>
      (content.tiers || DEFAULT_FALLBACK_TIERS).map((tier) => ({
        ...tier,
        features: getTierFeatures(tier),
      })),
    [content.tiers],
  );

  const hasYearly = yearlyTiers.length > 0;
  useEffect(() => {
    if (!hasYearly && activeInterval === 'yearly') {
      setActiveInterval('monthly');
    }
  }, [activeInterval, hasYearly]);

  // The required free tier may be prepended to the monthly plans. Savings must
  // compare paid plans; using a $0 anchor silently hid a valid yearly discount.
  const monthlyAnchor = monthlyPlansRaw.find((plan) =>
    typeof plan.amount_cents === 'number' && plan.amount_cents > 0,
  );
  const yearlyAnchor = yearlyPlansRaw.find((plan) =>
    typeof plan.amount_cents === 'number' && plan.amount_cents > 0,
  );
  const savingsPercent =
    monthlyAnchor && yearlyAnchor
      ? Math.max(
          0,
          Math.round(
            (1 - yearlyAnchor.amount_cents / 12 / (monthlyAnchor.amount_cents || 1)) * 100,
          ),
        )
      : null;
  const savingsLabel = savingsPercent && savingsPercent > 0 ? `Save ${String(savingsPercent)}% yearly` : null;

  const effectiveInterval =
    bundle && activeInterval === 'yearly' && hasYearly ? 'yearly' : 'monthly';
  const tiersToRender = useMemo(
    () =>
      bundle
        ? effectiveInterval === 'yearly'
          ? yearlyTiers.length > 0
            ? yearlyTiers
            : monthlyWithFree
          : monthlyWithFree
        : fallbackTiers.slice(0, 4),
    [bundle, effectiveInterval, fallbackTiers, monthlyWithFree, yearlyTiers],
  );

  const paddedTiers = useMemo(() => {
    if (!bundle || tiersToRender.length >= 3) {
      return tiersToRender;
    }
    const existing = new Set(tiersToRender.map((tier) => tier.name.toLowerCase()));
    const fillers = fallbackTiers.filter((tier) => !existing.has(tier.name.toLowerCase()));
    return [...tiersToRender, ...fillers].slice(0, 3);
  }, [bundle, tiersToRender, fallbackTiers]);

  const featuredTier = paddedTiers.find((tier) => tier.highlighted) || paddedTiers[0];

  return (
    <section className="bg-surface-alt py-24 text-slate-900">
      <div className="container mx-auto px-6">
        <div className="max-w-4xl space-y-4 text-left">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">Pricing</p>
          <h2 className="text-4xl font-semibold">{content.title || 'Built to scale from demo to production'}</h2>
          <p className="text-lg text-slate-600">
            {content.subtitle ||
              'Silent Founder OS starts with Vrooli Ascension. Early adopters lock pricing as we drop new business apps into the same stack.'}
          </p>
        </div>

        {bundle && hasYearly && (
          <div className="mt-8 inline-flex w-full max-w-xl flex-wrap items-center gap-2 rounded-full border border-slate-300 bg-white/80 p-1 text-sm font-semibold text-slate-600">
            {(['monthly', 'yearly'] as const).map((mode) => {
              const active = effectiveInterval === mode;
              return (
                <button
                  key={mode}
                  type="button"
                  onClick={() => { setActiveInterval(mode); }}
                  className={`flex-1 rounded-full px-4 py-2 transition ${
                    active ? 'bg-slate-900 text-white shadow' : 'text-slate-600 hover:text-slate-900'
                  }`}
                >
                  {mode === 'monthly' ? 'Monthly billing' : 'Yearly billing'}
                </button>
              );
            })}
            {savingsLabel && effectiveInterval === 'yearly' && (
              <span className="rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-700">
                {savingsLabel}
              </span>
            )}
          </div>
        )}

        <div className="mt-12 -mx-6 flex snap-x snap-mandatory gap-4 overflow-x-auto overflow-y-visible pb-6 pt-4 md:mx-0">
          {paddedTiers.map((tier, index) => (
            <div
              key={`${tier.name}-${tier.price}-${String(index)}`}
              className="min-w-[82%] flex-shrink-0 snap-center md:min-w-[360px] lg:min-w-[380px]"
            >
              <PricingTierCard
                tier={tier}
                index={index}
                redirectingPrice={redirectingPrice}
                onSelect={(selectedTier) => { void handleTierSelect(selectedTier); }}
              />
            </div>
          ))}
        </div>

        {sessionError && (
          <div className="mt-6 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
            {sessionError}
          </div>
        )}

      </div>
      {featuredTier && !stickyDismissed && (
        <div className="fixed bottom-4 left-1/2 z-20 w-[min(480px,92vw)] -translate-x-1/2 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl shadow-slate-700/30 md:hidden">
          <div className="flex items-center justify-between gap-3 px-4 py-3">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Featured</p>
              <p className="text-sm font-semibold text-slate-900">{featuredTier.name}</p>
              <p className="text-xs text-slate-500">{featuredTier.price}</p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                onClick={() => { void handleTierSelect(featuredTier); }}
                disabled={
                  resolvePriceId(featuredTier.cta_url) !== null &&
                  redirectingPrice === resolvePriceId(featuredTier.cta_url)
                }
              >
                {resolvePriceId(featuredTier.cta_url) !== null &&
                redirectingPrice === resolvePriceId(featuredTier.cta_url)
                  ? 'Redirecting...'
                  : featuredTier.cta_text || 'Choose'}
              </Button>
              <button
                type="button"
                className="text-xs text-slate-500 hover:text-slate-700"
                onClick={() => { setStickyDismissed(true); }}
                aria-label="Dismiss pricing sticky"
              >
                ✕
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
