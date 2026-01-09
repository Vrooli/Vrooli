import { Lock, Crown, Sparkles, Video, Image, ArrowUpRight, X, type LucideIcon } from 'lucide-react';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';
import type { SubscriptionTier } from '@stores/entitlementStore';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import { TierBadge } from '@shared/ui';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

export type GatedFeature = 'ai' | 'recording' | 'watermark';

interface FeatureConfig {
  id: GatedFeature;
  title: string;
  description: string;
  icon: LucideIcon;
  requiredTier: SubscriptionTier;
  benefits: string[];
}

const FEATURE_CONFIGS: Record<GatedFeature, FeatureConfig> = {
  ai: {
    id: 'ai',
    title: 'AI-Powered Features',
    description: 'Use AI to automatically generate and edit workflows from natural language descriptions.',
    icon: Sparkles,
    requiredTier: 'pro',
    benefits: [
      'Generate workflows from text descriptions',
      'AI-assisted step editing',
      'Smart selector suggestions',
      'Natural language workflow modifications',
    ],
  },
  recording: {
    id: 'recording',
    title: 'Live Recording',
    description: 'Record your browser interactions and automatically convert them into workflows.',
    icon: Video,
    requiredTier: 'solo',
    benefits: [
      'Record browser sessions',
      'Automatic step detection',
      'Edit recorded workflows',
      'Save and replay recordings',
    ],
  },
  watermark: {
    id: 'watermark',
    title: 'Watermark-Free Exports',
    description: 'Export your replay videos and GIFs without the Browser Automation Studio watermark.',
    icon: Image,
    requiredTier: 'pro',
    benefits: [
      'Clean, professional exports',
      'No branding on your videos',
      'Perfect for client presentations',
      'Brand-ready content',
    ],
  },
};

interface FeatureGateModalProps {
  isOpen: boolean;
  onClose: () => void;
  feature: GatedFeature;
  onOpenSettings?: (tab?: string) => void;
}

export function FeatureGateModal({ isOpen, onClose, feature, onOpenSettings }: FeatureGateModalProps) {
  const { userEmail, status } = useEntitlementStore();
  const config = FEATURE_CONFIGS[feature];
  const tierConfig = TIER_CONFIG[config.requiredTier];
  const Icon = config.icon;

  const upgradeUrl = userEmail
    ? `${LANDING_PAGE_URL}/pricing?email=${encodeURIComponent(userEmail)}`
    : `${LANDING_PAGE_URL}/pricing`;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel={`${config.title} requires upgrade`}
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
          <div className={`p-3 rounded-full ${tierConfig.bgColor}`}>
            <Lock size={24} className={tierConfig.color} />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-surface">
              {config.title}
            </h2>
            <div className="flex items-center gap-2 mt-1">
              <span className="text-sm text-gray-400">Requires</span>
              <TierBadge tier={config.requiredTier} size="sm" />
              <span className="text-sm text-gray-400">or higher</span>
            </div>
          </div>
        </div>

        {/* Feature Description */}
        <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700 mb-4">
          <div className="flex items-start gap-3">
            <div className={`p-2 rounded-lg ${tierConfig.bgColor}`}>
              <Icon size={20} className={tierConfig.color} />
            </div>
            <p className="text-gray-300 text-sm">{config.description}</p>
          </div>
        </div>

        {/* Benefits List */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-400 mb-3">What you&apos;ll get:</h3>
          <ul className="space-y-2">
            {config.benefits.map((benefit, index) => (
              <li key={index} className="flex items-center gap-2 text-sm text-gray-300">
                <div className={`w-1.5 h-1.5 rounded-full ${tierConfig.bgColor.replace('bg-', 'bg-').replace('/30', '')}`} />
                {benefit}
              </li>
            ))}
          </ul>
        </div>

        {/* Current Tier Info */}
        {status && (
          <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30 border border-gray-700 mb-6">
            <span className="text-sm text-gray-400">Your current plan:</span>
            <TierBadge tier={status.tier} size="sm" />
          </div>
        )}

        {/* Actions */}
        <div className="space-y-3">
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
            Upgrade to {tierConfig.label}
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

export default FeatureGateModal;
