import { Check, X, Sparkles, Video, Image, Lock, type LucideIcon } from 'lucide-react';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';
import type { SubscriptionTier } from '@stores/entitlementStore';

interface FeatureItem {
  id: string;
  label: string;
  description: string;
  icon: LucideIcon;
  checkAccess: (status: { can_use_ai: boolean; can_use_recording: boolean; requires_watermark: boolean }) => boolean;
  requiredTier: SubscriptionTier;
}

const FEATURES: FeatureItem[] = [
  {
    id: 'ai',
    label: 'AI-Powered Features',
    description: 'Use AI to generate and edit workflows automatically',
    icon: Sparkles,
    checkAccess: (status) => status.can_use_ai,
    requiredTier: 'pro',
  },
  {
    id: 'recording',
    label: 'Live Recording',
    description: 'Record browser interactions to create workflows',
    icon: Video,
    checkAccess: (status) => status.can_use_recording,
    requiredTier: 'solo',
  },
  {
    id: 'watermark',
    label: 'Watermark-Free Exports',
    description: 'Export videos and replays without watermarks',
    icon: Image,
    checkAccess: (status) => !status.requires_watermark,
    requiredTier: 'pro',
  },
];

export function FeatureAccessList() {
  const { status } = useEntitlementStore();

  if (!status) {
    return null;
  }

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <Lock size={20} className="text-flow-accent" />
          Feature Access
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          Features available based on your subscription tier.
        </p>
      </div>

      <div className="space-y-3">
        {FEATURES.map((feature) => {
          const hasAccess = feature.checkAccess({
            can_use_ai: status.can_use_ai,
            can_use_recording: status.can_use_recording,
            requires_watermark: status.requires_watermark,
          });
          const Icon = feature.icon;
          const tierConfig = TIER_CONFIG[feature.requiredTier];

          return (
            <div
              key={feature.id}
              className={`
                flex items-center justify-between p-4 rounded-lg border
                ${hasAccess
                  ? 'bg-gray-800/50 border-gray-700'
                  : 'bg-gray-900/50 border-gray-800 opacity-75'
                }
              `}
            >
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded-lg ${hasAccess ? 'bg-flow-accent/20' : 'bg-gray-700/50'}`}>
                  <Icon size={20} className={hasAccess ? 'text-flow-accent' : 'text-gray-500'} />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className={`font-medium ${hasAccess ? 'text-surface' : 'text-gray-400'}`}>
                      {feature.label}
                    </span>
                    {!hasAccess && (
                      <span className={`text-xs px-1.5 py-0.5 rounded ${tierConfig.bgColor} ${tierConfig.color}`}>
                        {tierConfig.label}+
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-gray-500">{feature.description}</p>
                </div>
              </div>

              <div className={`p-1.5 rounded-full ${hasAccess ? 'bg-green-900/30' : 'bg-gray-800'}`}>
                {hasAccess ? (
                  <Check size={16} className="text-green-400" />
                ) : (
                  <X size={16} className="text-gray-500" />
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default FeatureAccessList;
