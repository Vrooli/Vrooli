import { Check } from 'lucide-react';
import { Button } from '../ui/button';
import { useMetrics } from '../../hooks/useMetrics';

interface PricingTier {
  name: string;
  price: string;
  description: string;
  features: string[];
  cta_text?: string;
  cta_url?: string;
  highlighted?: boolean;
}

interface PricingSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    tiers?: PricingTier[];
  };
}

export function PricingSection({ content }: PricingSectionProps) {
  const { trackCTAClick } = useMetrics();

  const tiers = content.tiers || [
    {
      name: 'Starter',
      price: '$29',
      description: 'Perfect for small projects',
      features: ['1 Landing Page', 'Basic Analytics', 'A/B Testing', 'Email Support'],
      cta_text: 'Get Started',
      cta_url: '/checkout?plan=starter',
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

  const handlePricingCTA = (tier: PricingTier) => {
    if (tier.cta_url) {
      trackCTAClick(`pricing-${tier.name.toLowerCase()}-cta`, {
        tier: tier.name,
        price: tier.price,
      });
      window.location.href = tier.cta_url;
    }
  };

  return (
    <section className="py-24 bg-slate-950 relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute top-1/2 left-0 w-96 h-96 bg-purple-500/10 rounded-full blur-3xl"></div>
      <div className="absolute top-1/2 right-0 w-96 h-96 bg-blue-500/10 rounded-full blur-3xl"></div>

      <div className="container relative z-10 mx-auto px-6">
        <div className="max-w-6xl mx-auto">
          {/* Header */}
          <div className="text-center mb-16 space-y-4">
            <h2 className="text-4xl lg:text-5xl font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
              {content.title || 'Simple, Transparent Pricing'}
            </h2>
            <p className="text-xl text-slate-400 max-w-2xl mx-auto">
              {content.subtitle || 'Choose the plan that fits your needs. No hidden fees, cancel anytime.'}
            </p>
          </div>

          {/* Pricing Tiers */}
          <div className="grid md:grid-cols-3 gap-8 items-start">
            {tiers.map((tier, index) => (
              <div
                key={index}
                className={`relative rounded-2xl border p-8 transition-all duration-300 ${
                  tier.highlighted
                    ? 'border-purple-500 bg-gradient-to-br from-purple-500/10 to-blue-500/10 scale-105 shadow-2xl shadow-purple-500/25'
                    : 'border-white/10 bg-white/5 hover:border-purple-500/50 hover:bg-white/10'
                }`}
                data-testid={`pricing-tier-${index}`}
              >
                {tier.highlighted && (
                  <div className="absolute -top-4 left-1/2 -translate-x-1/2 px-4 py-1 bg-gradient-to-r from-purple-600 to-blue-600 rounded-full text-sm font-semibold">
                    Most Popular
                  </div>
                )}

                <div className="space-y-6">
                  {/* Tier name and price */}
                  <div>
                    <h3 className="text-2xl font-bold text-white mb-2">{tier.name}</h3>
                    <p className="text-slate-400 text-sm mb-4">{tier.description}</p>
                    <div className="flex items-baseline gap-2">
                      <span className="text-5xl font-bold text-white">{tier.price}</span>
                      {tier.price !== 'Custom' && <span className="text-slate-400">/month</span>}
                    </div>
                  </div>

                  {/* CTA Button */}
                  <Button
                    className={`w-full ${
                      tier.highlighted
                        ? 'bg-gradient-to-r from-purple-600 to-blue-600 hover:from-purple-500 hover:to-blue-500'
                        : 'bg-white/10 hover:bg-white/20'
                    }`}
                    size="lg"
                    onClick={() => handlePricingCTA(tier)}
                    data-testid={`pricing-cta-${tier.name.toLowerCase()}`}
                  >
                    {tier.cta_text || 'Get Started'}
                  </Button>

                  {/* Features */}
                  <ul className="space-y-3">
                    {tier.features.map((feature, featureIndex) => (
                      <li key={featureIndex} className="flex items-start gap-3">
                        <Check className="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
                        <span className="text-slate-300">{feature}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
