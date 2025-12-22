import { Check, X, Sparkles, Video, Image, Lock, type LucideIcon } from 'lucide-react';
import { useEntitlementStore, TIER_CONFIG, type FeatureAccessSummary } from '@stores/entitlementStore';

const FEATURE_ICONS: Record<string, LucideIcon> = {
  ai: Sparkles,
  recording: Video,
  'watermark-free': Image,
};

const buildFallbackFeatures = (status: {
  can_use_ai: boolean;
  can_use_recording: boolean;
  requires_watermark: boolean;
}): FeatureAccessSummary[] => [
  {
    id: 'ai',
    label: 'AI-Powered Features',
    description: 'Use AI to generate and edit workflows automatically',
    required_tier: 'pro',
    has_access: status.can_use_ai,
  },
  {
    id: 'recording',
    label: 'Live Recording',
    description: 'Record browser interactions to create workflows',
    required_tier: 'solo',
    has_access: status.can_use_recording,
  },
  {
    id: 'watermark-free',
    label: 'Watermark-Free Exports',
    description: 'Export videos and replays without watermarks',
    required_tier: 'pro',
    has_access: !status.requires_watermark,
  },
];

export function FeatureAccessList() {
  const { status } = useEntitlementStore();

  if (!status) {
    return null;
  }

  const features = status.feature_access?.length
    ? status.feature_access
    : buildFallbackFeatures({
        can_use_ai: status.can_use_ai,
        can_use_recording: status.can_use_recording,
        requires_watermark: status.requires_watermark,
      });

  const includedFeatures = features.filter((feature) => feature.has_access);
  const lockedFeatures = features.filter((feature) => !feature.has_access);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <Lock size={20} className="text-flow-accent" />
          Entitlement Access
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          See what your current subscription unlocks and what upgrades provide.
        </p>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <div className="space-y-3 rounded-xl border border-gray-800 bg-gray-900/40 p-4">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-semibold text-surface">Included</h4>
            <span className="text-xs text-gray-400">{includedFeatures.length}</span>
          </div>
          {includedFeatures.length === 0 ? (
            <div className="text-sm text-gray-500">No entitlements detected.</div>
          ) : (
            includedFeatures.map((feature) => {
              const Icon = FEATURE_ICONS[feature.id] ?? Lock;
              return (
                <div
                  key={feature.id}
                  className="flex items-center justify-between gap-3 rounded-lg border border-gray-800 bg-gray-900/60 p-3"
                >
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-flow-accent/15 p-2">
                      <Icon size={18} className="text-flow-accent" />
                    </div>
                    <div>
                      <div className="font-medium text-surface">{feature.label}</div>
                      <p className="text-xs text-gray-500">{feature.description}</p>
                    </div>
                  </div>
                  <div className="rounded-full bg-green-900/30 p-1.5">
                    <Check size={16} className="text-green-400" />
                  </div>
                </div>
              );
            })
          )}
        </div>

        <div className="space-y-3 rounded-xl border border-gray-800 bg-gray-900/40 p-4">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-semibold text-surface">Locked</h4>
            <span className="text-xs text-gray-400">{lockedFeatures.length}</span>
          </div>
          {lockedFeatures.length === 0 ? (
            <div className="text-sm text-gray-500">You have access to everything listed.</div>
          ) : (
            lockedFeatures.map((feature) => {
              const Icon = FEATURE_ICONS[feature.id] ?? Lock;
              const tierConfig = feature.required_tier ? TIER_CONFIG[feature.required_tier] : undefined;
              return (
                <div
                  key={feature.id}
                  className="flex items-center justify-between gap-3 rounded-lg border border-gray-800 bg-gray-900/30 p-3"
                >
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-gray-800/70 p-2">
                      <Icon size={18} className="text-gray-500" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-300">{feature.label}</span>
                        {tierConfig && (
                          <span className={`text-[10px] px-1.5 py-0.5 rounded ${tierConfig.bgColor} ${tierConfig.color}`}>
                            {tierConfig.label}+
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-gray-500">{feature.description}</p>
                    </div>
                  </div>
                  <div className="rounded-full bg-gray-800 p-1.5">
                    <X size={16} className="text-gray-500" />
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}

export default FeatureAccessList;
