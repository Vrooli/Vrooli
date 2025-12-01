import { Check } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';
import type { PlanOption, PricingOverview } from '../../../shared/api';

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

function buildTierFromPlan(option: PlanOption, bundle: PricingOverview['bundle'], highlighted: boolean) {
  const priceLabel = formatCurrency(option.amount_cents, option.currency);
  const introAmount = option.intro_amount_cents;
  const badge =
    option.intro_enabled && introAmount
      ? `${formatCurrency(introAmount, option.currency)} intro for ${option.intro_periods || 1} month${option.intro_periods === 1 ? '' : 's'}`
      : option.bonus_type
        ? option.bonus_type.replace('_', ' ')
        : undefined;

  const metaFeatures = option.metadata?.features;
  const baseFeatures = Array.isArray(metaFeatures) ? (metaFeatures as string[]) : [];
  const creditsLabel = bundle.display_credits_label || 'credits';
  const features = [
    `${formatCredits(option.monthly_included_credits, bundle.display_credits_multiplier, creditsLabel)} included`,
    ...(option.one_time_bonus_credits > 0
      ? [
          `Bonus ${formatCredits(option.one_time_bonus_credits, bundle.display_credits_multiplier, creditsLabel)}`,
        ]
      : []),
    ...baseFeatures,
  ];

  return {
    name: option.plan_name,
    description: option.plan_tier.charAt(0).toUpperCase() + option.plan_tier.slice(1),
    price: `${priceLabel} / ${option.billing_interval === 'month' ? 'month' : 'year'}`,
    features,
    cta_text: option.intro_enabled
      ? `Start ${formatCurrency(introAmount ?? option.amount_cents, option.currency)} intro`
      : 'Choose plan',
    cta_url: `/checkout?price_id=${option.stripe_price_id}`,
    highlighted,
    badge,
    subtitle: `Plan rank #${option.plan_rank}`,
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
        key={`${tier.name}-${tier.price}-${index}`}
        className={`relative rounded-3xl border p-8 transition-all duration-300 ${
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
  const monthlyPlansRaw = pricingOverview?.monthly;
  const yearlyPlansRaw = pricingOverview?.yearly;
  const monthlyPlans = Array.isArray(monthlyPlansRaw) ? monthlyPlansRaw : [];
  const yearlyPlans = Array.isArray(yearlyPlansRaw) ? yearlyPlansRaw : [];

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

        <div className="mt-12 grid items-start gap-8 md:grid-cols-3">
          {(pricingOverview && monthlyTiers.length > 0 ? monthlyTiers : fallbackTiers).map((tier, index) =>
            renderTier(tier, index)
          )}
        </div>

        {yearlyTiers.length > 0 && (
          <div className="mt-12 rounded-3xl border border-slate-200 bg-white p-6 text-sm text-slate-600">
            Yearly billing available on request — includes white-glove promotion support and export of the entire style
            pack for compliance archives.
          </div>
        )}
      </div>
    </section>
  );
}
