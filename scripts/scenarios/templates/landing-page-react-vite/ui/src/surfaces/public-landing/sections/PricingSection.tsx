import { useEffect, useState } from 'react';
import { Check } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';
import type { PlanOption, PricingOverview } from '../../../shared/api';
import { ensureDemoPlansForDisplay } from '../../../shared/lib/pricingPlaceholders';

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
  return `${value} ${label}`;
}

function buildTierFromPlan(option: PlanOption, bundle: PricingOverview['bundle'], fallbackHighlight: boolean) {
  const priceLabel = formatCurrency(option.amount_cents, option.currency);
  const introAmount = option.intro_amount_cents;
  const metadata = option.metadata || {};
  const metaFeatures = Array.isArray(metadata.features) ? (metadata.features as string[]) : [];
  const creditsLabel = bundle.display_credits_label || 'credits';
  const badgeOverride = typeof metadata.badge === 'string' ? (metadata.badge as string) : undefined;
  const badge =
    badgeOverride ||
    (option.intro_enabled && introAmount
      ? `${formatCurrency(introAmount, option.currency)} intro for ${option.intro_periods || 1} month${option.intro_periods === 1 ? '' : 's'}`
      : option.bonus_type
        ? option.bonus_type.replace('_', ' ')
        : undefined);

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
      ? (metadata.subtitle as string)
      : `Plan rank #${option.plan_rank}`;
  const ctaText =
    typeof metadata.cta_label === 'string' && metadata.cta_label.trim().length > 0
      ? (metadata.cta_label as string)
      : option.intro_enabled
        ? `Start ${formatCurrency(introAmount ?? option.amount_cents, option.currency)} intro`
        : 'Choose plan';

  const highlighted = metadata.highlight === true ? true : fallbackHighlight;

  return {
    name: option.plan_name,
    description: option.plan_tier.charAt(0).toUpperCase() + option.plan_tier.slice(1),
    price: `${priceLabel} / ${option.billing_interval === 'month' ? 'month' : 'year'}`,
    features,
    cta_text: ctaText,
    cta_url: `/checkout?price_id=${option.stripe_price_id}`,
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

export function PricingSection({ content, pricingOverview }: PricingSectionProps) {
  const { trackCTAClick } = useMetrics();
  const [activeInterval, setActiveInterval] = useState<'monthly' | 'yearly'>('monthly');

  const renderTier = (tier: PricingTier, index: number) => {
    const handleClick = () => {
      if (!tier.cta_url) return;
      trackCTAClick(`pricing-${tier.name.toLowerCase().replace(/\s+/g, '-')}-cta`, {
        tier: tier.name,
        price: tier.price,
      });
      window.location.href = tier.cta_url;
    };

    const highlight = tier.highlighted;
    return (
      <div
        className={`relative h-full rounded-3xl border p-8 transition-all duration-300 ${
          highlight
            ? 'border-[#F97316]/40 bg-[#0F172A] text-white shadow-2xl shadow-[#F97316]/20'
            : 'border-slate-200 bg-white text-slate-900 hover:-translate-y-1'
        }`}
        data-testid={`pricing-tier-${index}`}
      >
        {tier.badge && (
          <div className={`absolute -top-4 left-1/2 -translate-x-1/2 rounded-full px-4 py-1 text-sm font-semibold ${highlight ? 'bg-white/10 text-white' : 'bg-[#0F172A] text-white'}`}>
            {tier.badge}
          </div>
        )}

        <div className="space-y-6">
          <div>
            <h3 className={`text-2xl font-bold ${highlight ? 'text-white' : 'text-slate-900'} mb-2`}>{tier.name}</h3>
            {tier.subtitle && (
              <p className={`text-xs uppercase tracking-wider mb-2 ${highlight ? 'text-[#F97316]' : 'text-slate-500'}`}>
                {tier.subtitle}
              </p>
            )}
            <p className={`${highlight ? 'text-slate-300' : 'text-slate-500'} text-sm mb-4`}>{tier.description}</p>
            <div className="flex items-baseline gap-2">
              <span className={`text-5xl font-bold ${highlight ? 'text-white' : 'text-slate-900'}`}>{tier.price}</span>
            </div>
          </div>

          <Button
            variant={highlight ? 'default' : 'ghost'}
            className={`w-full ${highlight ? '' : 'bg-slate-900/5 text-slate-900 hover:bg-slate-900/10'}`}
            size="lg"
            onClick={handleClick}
            data-testid={`pricing-cta-${tier.name.toLowerCase().replace(/\s+/g, '-')}`}
          >
            {tier.cta_text || 'Get Started'}
          </Button>

          <ul className="space-y-3">
            {getTierFeatures(tier).map((feature, featureIndex) => (
              <li key={`${feature}-${featureIndex}`} className="flex items-start gap-3">
                <Check className={`w-5 h-5 flex-shrink-0 mt-0.5 ${highlight ? 'text-[#10B981]' : 'text-[#F97316]'}`} />
                <span className={highlight ? 'text-slate-200' : 'text-slate-600'}>{feature}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    );
  };

  const bundle = pricingOverview?.bundle;
  const monthlyPlansRaw = Array.isArray(pricingOverview?.monthly) ? (pricingOverview?.monthly as PlanOption[]) : [];
  const yearlyPlansRaw = Array.isArray(pricingOverview?.yearly) ? (pricingOverview?.yearly as PlanOption[]) : [];
  const monthlyPlans = bundle ? ensureDemoPlansForDisplay(bundle, monthlyPlansRaw, 3) : [];
  const yearlyPlans = bundle ? yearlyPlansRaw : [];

  const monthlyTiers =
    bundle && monthlyPlans.length > 0
      ? monthlyPlans.map((option, index) => buildTierFromPlan(option, bundle, index === 0))
      : [];
  const yearlyTiers =
    bundle && yearlyPlans.length > 0
      ? yearlyPlans.map((option, index) => buildTierFromPlan(option, bundle, index === 0))
      : [];

  const fallbackTiers = (content.tiers || [
    {
      name: 'Starter',
      price: '$29',
      description: 'Perfect for small projects',
      features: ['1 Landing Page', 'Basic Analytics', 'A/B Testing', 'Email Support'],
      cta_text: 'Get Started',
      cta_url: '/checkout?plan=starter',
      highlighted: false,
    },
    {
      name: 'Professional',
      price: '$99',
      description: 'For growing businesses',
      features: ['Unlimited Pages', 'Advanced Analytics', 'A/B Testing', 'Priority Support', 'Custom Domain', 'Agent Customization'],
      cta_text: 'Start Free Trial',
      cta_url: '/checkout?plan=professional',
      highlighted: true,
    },
    {
      name: 'Enterprise',
      price: 'Custom',
      description: 'For large organizations',
      features: ['Everything in Professional', 'White Label', 'Dedicated Support', 'SLA Guarantee', 'Custom Integrations'],
      cta_text: 'Contact Sales',
      cta_url: '/contact',
    },
  ]).map((tier) => ({
    ...tier,
    features: getTierFeatures(tier),
  }));

  const hasYearly = yearlyTiers.length > 0;
  useEffect(() => {
    if (!hasYearly && activeInterval === 'yearly') {
      setActiveInterval('monthly');
    }
  }, [activeInterval, hasYearly]);

  const effectiveInterval =
    bundle && activeInterval === 'yearly' && hasYearly ? 'yearly' : 'monthly';
  const tiersToRender = bundle
    ? effectiveInterval === 'yearly'
      ? yearlyTiers.length > 0
        ? yearlyTiers
        : monthlyTiers
      : monthlyTiers.length > 0
        ? monthlyTiers
        : []
    : fallbackTiers;

  return (
    <section className="bg-[#F6F5F2] py-24 text-slate-900">
      <div className="container mx-auto px-6">
        <div className="max-w-4xl space-y-4 text-left">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">Pricing</p>
          <h2 className="text-4xl font-semibold">{content.title || 'Built to scale from demo to production'}</h2>
          <p className="text-lg text-slate-600">
            {content.subtitle ||
              'Every plan inherits the Clause-grade styling pack, download governance, and analytics instrumentation. Seats scale, guardrails stay intact.'}
          </p>
        </div>

        {bundle && hasYearly && (
          <div className="mt-8 inline-flex w-full max-w-xl flex-wrap gap-2 rounded-full border border-slate-300 bg-white/80 p-1 text-sm font-semibold text-slate-600">
            {(['monthly', 'yearly'] as const).map((mode) => {
              const active = effectiveInterval === mode;
              return (
                <button
                  key={mode}
                  type="button"
                  onClick={() => setActiveInterval(mode)}
                  className={`flex-1 rounded-full px-4 py-2 transition ${
                    active ? 'bg-slate-900 text-white shadow' : 'text-slate-600 hover:text-slate-900'
                  }`}
                >
                  {mode === 'monthly' ? 'Monthly billing' : 'Yearly billing'}
                </button>
              );
            })}
          </div>
        )}

        <div className="mt-12 -mx-6 flex snap-x snap-mandatory gap-4 overflow-x-auto pb-6 md:mx-0 md:grid md:grid-cols-3 md:gap-8 md:overflow-visible md:pb-0 md:snap-none">
          {tiersToRender.map((tier, index) => (
            <div
              key={`${tier.name}-${tier.price ?? 'n/a'}-${index}`}
              className="min-w-[82%] flex-shrink-0 snap-center md:min-w-0 md:flex-shrink"
            >
              {renderTier(tier, index)}
            </div>
          ))}
        </div>

        {hasYearly && (
          <div className="mt-12 rounded-3xl border border-slate-200 bg-white p-6 text-sm text-slate-600">
            Yearly billing available on request — includes white-glove promotion support and export of the entire style
            pack for compliance archives.
          </div>
        )}
      </div>
    </section>
  );
}
