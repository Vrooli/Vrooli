import { ArrowUpRight, Crown, Sparkles, Zap, Star, type LucideIcon } from 'lucide-react';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';
import type { SubscriptionTier } from '@stores/entitlementStore';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

interface TierComparisonItem {
  tier: SubscriptionTier;
  features: string[];
  highlight?: boolean;
}

const TIER_COMPARISON: TierComparisonItem[] = [
  {
    tier: 'solo',
    features: ['50 monthly executions', 'Live recording', 'Basic support'],
  },
  {
    tier: 'pro',
    features: ['200 monthly executions', 'AI-powered features', 'Watermark-free exports', 'Priority support'],
    highlight: true,
  },
  {
    tier: 'studio',
    features: ['Unlimited executions', 'All Pro features', 'Team collaboration', 'Custom branding'],
  },
];

const TIER_ICONS: Record<SubscriptionTier, LucideIcon> = {
  free: Zap,
  solo: Star,
  pro: Sparkles,
  studio: Crown,
  business: Crown,
};

export function UpgradePromptSection() {
  const { status, userEmail } = useEntitlementStore();

  // Don't show upgrade prompt for high tiers or when entitlements are disabled
  if (!status?.entitlements_enabled) {
    return null;
  }

  const currentTier = status?.tier || 'free';
  const isHighTier = currentTier === 'studio' || currentTier === 'business';

  if (isHighTier) {
    return null;
  }

  // Build upgrade URL with email pre-filled
  const upgradeUrl = userEmail
    ? `${LANDING_PAGE_URL}/pricing?email=${encodeURIComponent(userEmail)}`
    : `${LANDING_PAGE_URL}/pricing`;

  // Get tiers to show (tiers higher than current)
  const tierOrder: SubscriptionTier[] = ['free', 'solo', 'pro', 'studio', 'business'];
  const currentIndex = tierOrder.indexOf(currentTier);
  const availableUpgrades = TIER_COMPARISON.filter((item) => {
    const itemIndex = tierOrder.indexOf(item.tier);
    return itemIndex > currentIndex;
  });

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <ArrowUpRight size={20} className="text-flow-accent" />
          Upgrade Your Plan
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          Unlock more features and higher limits with a premium subscription.
        </p>
      </div>

      {/* Tier Comparison Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {availableUpgrades.map((item) => {
          const tierConfig = TIER_CONFIG[item.tier];
          const Icon = TIER_ICONS[item.tier];

          return (
            <div
              key={item.tier}
              className={`
                rounded-xl border p-4 transition-all
                ${item.highlight
                  ? `${tierConfig.bgColor} ${tierConfig.borderColor} ring-1 ring-${item.tier === 'pro' ? 'purple' : 'amber'}-500/30`
                  : 'border-gray-700 bg-gray-800/50 hover:border-gray-600'
                }
              `}
            >
              <div className="flex items-center gap-2 mb-3">
                <div className={`p-1.5 rounded-lg ${tierConfig.bgColor}`}>
                  <Icon size={18} className={tierConfig.color} />
                </div>
                <span className={`font-semibold ${tierConfig.color}`}>
                  {tierConfig.label}
                </span>
                {item.highlight && (
                  <span className="text-xs px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-300">
                    Popular
                  </span>
                )}
              </div>

              <ul className="space-y-2 mb-4">
                {item.features.map((feature, index) => (
                  <li key={index} className="flex items-start gap-2 text-sm text-gray-300">
                    <Zap size={14} className={`mt-0.5 flex-shrink-0 ${tierConfig.color}`} />
                    {feature}
                  </li>
                ))}
              </ul>
            </div>
          );
        })}
      </div>

      {/* Upgrade CTA */}
      <div className="flex flex-col sm:flex-row items-center gap-4 p-4 rounded-xl bg-gradient-to-r from-purple-900/30 to-blue-900/30 border border-purple-700/50">
        <div className="flex-1 text-center sm:text-left">
          <p className="font-medium text-surface">Ready to upgrade?</p>
          <p className="text-sm text-gray-400">
            Get more features and higher limits today.
          </p>
        </div>
        <a
          href={upgradeUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="
            flex items-center gap-2 px-6 py-3 rounded-lg
            bg-gradient-to-r from-purple-600 to-blue-600
            hover:from-purple-500 hover:to-blue-500
            text-white font-medium
            transition-all shadow-lg hover:shadow-purple-500/25
          "
        >
          <Crown size={18} />
          Upgrade Now
          <ArrowUpRight size={16} />
        </a>
      </div>
    </div>
  );
}

export default UpgradePromptSection;
