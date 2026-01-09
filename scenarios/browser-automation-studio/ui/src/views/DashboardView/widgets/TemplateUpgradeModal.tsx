import { ArrowUpRight, Crown, CreditCard, Sparkles, X, Zap, Star, type LucideIcon } from 'lucide-react';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';
import type { SubscriptionTier } from '@stores/entitlementStore';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import { TierBadge } from '@shared/ui';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

// TODO: Update this URL to the actual buy credits page
const BUY_CREDITS_URL = 'https://example.com/buy-credits';

interface TierFeature {
  tier: SubscriptionTier;
  aiCredits: string;
  features: string[];
  highlight?: boolean;
}

const TIER_AI_FEATURES: TierFeature[] = [
  {
    tier: 'pro',
    aiCredits: '500 AI credits/month',
    features: ['AI workflow generation', 'Smart step suggestions', 'Natural language editing'],
    highlight: true,
  },
  {
    tier: 'studio',
    aiCredits: '2,000 AI credits/month',
    features: ['All Pro features', 'Priority AI processing', 'Advanced AI models'],
  },
  {
    tier: 'business',
    aiCredits: 'Unlimited AI credits',
    features: ['All Studio features', 'Custom AI fine-tuning', 'Dedicated support'],
  },
];

const TIER_ICONS: Record<SubscriptionTier, LucideIcon> = {
  free: Zap,
  solo: Star,
  pro: Sparkles,
  studio: Crown,
  business: Crown,
};

interface TemplateUpgradeModalProps {
  isOpen: boolean;
  onClose: () => void;
  onOpenSettings?: (tab?: string) => void;
}

export function TemplateUpgradeModal({ isOpen, onClose, onOpenSettings }: TemplateUpgradeModalProps) {
  const { userEmail, status } = useEntitlementStore();
  const currentTier = status?.tier || 'free';

  const upgradeUrl = userEmail
    ? `${LANDING_PAGE_URL}/pricing?email=${encodeURIComponent(userEmail)}`
    : `${LANDING_PAGE_URL}/pricing`;

  // Filter to show tiers higher than current
  const tierOrder: SubscriptionTier[] = ['free', 'solo', 'pro', 'studio', 'business'];
  const currentIndex = tierOrder.indexOf(currentTier);
  const availableUpgrades = TIER_AI_FEATURES.filter((item) => {
    const itemIndex = tierOrder.indexOf(item.tier);
    return itemIndex > currentIndex;
  });

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel="AI credits required for templates"
    >
      <div className="relative">
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute top-0 right-0 p-1 text-gray-400 hover:text-gray-200 transition-colors"
        >
          <X size={20} />
        </button>

        {/* Header */}
        <div className="flex items-center gap-3 mb-4">
          <div className="p-3 rounded-full bg-purple-500/20">
            <Sparkles size={24} className="text-purple-400" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-surface">
              AI Credits Required
            </h2>
            <p className="text-sm text-gray-400 mt-0.5">
              Templates use AI to automate your browser
            </p>
          </div>
        </div>

        {/* Explanation */}
        <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700 mb-4">
          <p className="text-gray-300 text-sm">
            Templates use AI to understand websites and perform complex actions automatically.
            Upgrade your plan or purchase credits to unlock this feature.
          </p>
        </div>

        {/* Current Tier Info */}
        {status && (
          <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30 border border-gray-700 mb-4">
            <span className="text-sm text-gray-400">Your current plan:</span>
            <TierBadge tier={status.tier} size="sm" />
          </div>
        )}

        {/* Tier Comparison Cards */}
        <div className="space-y-3 mb-6">
          <h3 className="text-sm font-medium text-gray-400">Plans with AI access:</h3>
          <div className="grid gap-3">
            {availableUpgrades.map((item) => {
              const tierConfig = TIER_CONFIG[item.tier];
              const Icon = TIER_ICONS[item.tier];

              return (
                <div
                  key={item.tier}
                  className={`
                    rounded-xl border p-4 transition-all
                    ${item.highlight
                      ? `${tierConfig.bgColor} ${tierConfig.borderColor} ring-1 ring-purple-500/30`
                      : 'border-gray-700 bg-gray-800/50'
                    }
                  `}
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <div className={`p-1.5 rounded-lg ${tierConfig.bgColor}`}>
                        <Icon size={16} className={tierConfig.color} />
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
                    <span className="text-sm text-gray-300 font-medium">
                      {item.aiCredits}
                    </span>
                  </div>

                  <ul className="flex flex-wrap gap-x-4 gap-y-1">
                    {item.features.map((feature, index) => (
                      <li key={index} className="flex items-center gap-1.5 text-xs text-gray-400">
                        <Zap size={10} className={tierConfig.color} />
                        {feature}
                      </li>
                    ))}
                  </ul>
                </div>
              );
            })}
          </div>
        </div>

        {/* Actions */}
        <div className="space-y-3">
          {/* Buy Credits Button */}
          <a
            href={BUY_CREDITS_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="
              flex items-center justify-center gap-2 w-full px-4 py-3 rounded-lg
              bg-gradient-to-r from-green-600 to-emerald-600
              hover:from-green-500 hover:to-emerald-500
              text-white font-medium
              transition-all shadow-lg hover:shadow-green-500/25
            "
          >
            <CreditCard size={18} />
            Buy AI Credits
            <ArrowUpRight size={16} />
          </a>

          {/* Upgrade Plan Button */}
          <a
            href={upgradeUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="
              flex items-center justify-center gap-2 w-full px-4 py-3 rounded-lg
              bg-gradient-to-r from-purple-600 to-blue-600
              hover:from-purple-500 hover:to-blue-500
              text-white font-medium
              transition-all shadow-lg hover:shadow-purple-500/25
            "
          >
            <Crown size={18} />
            Upgrade Plan
            <ArrowUpRight size={16} />
          </a>

          {onOpenSettings && (
            <button
              onClick={() => {
                onClose();
                onOpenSettings('subscription');
              }}
              className="
                w-full px-4 py-2 rounded-lg
                bg-gray-700 hover:bg-gray-600
                text-gray-300 text-sm
                transition-colors
              "
            >
              View All Plans
            </button>
          )}

          <button
            onClick={onClose}
            className="
              w-full px-4 py-2 rounded-lg
              text-gray-400 hover:text-gray-300 text-sm
              transition-colors
            "
          >
            Maybe Later
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default TemplateUpgradeModal;
