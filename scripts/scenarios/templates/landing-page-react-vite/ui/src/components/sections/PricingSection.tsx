import { Check } from 'lucide-react';
import { Button } from '../ui/button';
import { useMetrics } from '../../hooks/useMetrics';
import type { PlanOption, PricingOverview } from '../../lib/api';

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

    return (
      <div
        key={`${tier.name}-${tier.price}-${index}`}
        className={`relative rounded-2xl border p-8 transition-all duration-300 ${
          tier.highlighted
            ? 'border-purple-500 bg-gradient-to-br from-purple-500/10 to-blue-500/10 scale-105 shadow-2xl shadow-purple-500/25'
            : 'border-white/10 bg-white/5 hover:border-purple-500/50 hover:bg-white/10'
        }`}
        data-testid={`pricing-tier-${index}`}
      >
        {tier.badge && (
          <div className="absolute -top-4 left-1/2 -translate-x-1/2 px-4 py-1 bg-gradient-to-r from-purple-600 to-blue-600 rounded-full text-sm font-semibold">
            {tier.badge}
          </div>
        )}

        <div className="space-y-6">
          <div>
            <h3 className="text-2xl font-bold text-white mb-2">{tier.name}</h3>
            {tier.subtitle && <p className="text-xs text-purple-300 uppercase tracking-wider mb-2">{tier.subtitle}</p>}
            <p className="text-slate-400 text-sm mb-4">{tier.description}</p>
            <div className="flex items-baseline gap-2">
              <span className="text-5xl font-bold text-white">{tier.price}</span>
            </div>
          </div>

          <Button
            className={`w-full ${
              tier.highlighted
                ? 'bg-gradient-to-r from-purple-600 to-blue-600 hover:from-purple-500 hover:to-blue-500'
                : 'bg-white/10 hover:bg-white/20'
            }`}
            size="lg"
            onClick={handleClick}
            data-testid={`pricing-cta-${tier.name.toLowerCase().replace(/\s+/g, '-')}`}
          >
            {tier.cta_text || 'Get Started'}
          </Button>

          <ul className="space-y-3">
            {tier.features.map((feature, featureIndex) => (
              <li key={`${feature}-${featureIndex}`} className="flex items-start gap-3">
                <Check className="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
                <span className="text-slate-300">{feature}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    );
  };

  const monthlyTiers =
    pricingOverview && pricingOverview.bundle
      ? pricingOverview.monthly.map((option, index) =>
          buildTierFromPlan(option, pricingOverview.bundle, index === 0)
        )
      : [];
  const yearlyTiers =
    pricingOverview && pricingOverview.bundle
      ? pricingOverview.yearly.map((option, index) =>
          buildTierFromPlan(option, pricingOverview.bundle, index === 0)
        )
      : [];

  const fallbackTiers = content.tiers || [
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
  ];

  return (
    <section className="py-24 bg-slate-950 relative overflow-hidden">
      <div className="absolute top-1/2 left-0 w-96 h-96 bg-purple-500/10 rounded-full blur-3xl"></div>
      <div className="absolute top-1/2 right-0 w-96 h-96 bg-blue-500/10 rounded-full blur-3xl"></div>

      <div className="container relative z-10 mx-auto px-6">
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16 space-y-4">
            <h2 className="text-4xl lg:text-5xl font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
              {content.title || 'Simple, Transparent Pricing'}
            </h2>
            <p className="text-xl text-slate-400 max-w-2xl mx-auto">
              {content.subtitle || 'Choose the plan that fits your needs. No hidden fees, cancel anytime.'}
            </p>
          </div>

          {pricingOverview && monthlyTiers.length > 0 ? (
            <>
              <div className="mb-8 text-center">
                <span className="text-sm uppercase tracking-[0.3em] text-slate-500">Monthly</span>
              </div>
              <div className="grid md:grid-cols-3 gap-8 items-start mb-14">
                {monthlyTiers.map((tier, index) => renderTier(tier, index))}
              </div>
              {yearlyTiers.length > 0 && (
                <>
                  <div className="mb-8 text-center">
                    <span className="text-sm uppercase tracking-[0.3em] text-slate-500">Yearly</span>
                  </div>
                  <div className="grid md:grid-cols-3 gap-8 items-start">
                    {yearlyTiers.map((tier, index) => renderTier(tier, index))}
                  </div>
                </>
              )}
            </>
          ) : (
            <div className="grid md:grid-cols-3 gap-8 items-start">
              {fallbackTiers.map((tier, index) => renderTier(tier, index))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
