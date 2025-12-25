import { useState, useEffect } from 'react';
import {
  Sparkles,
  AlertTriangle,
  Eye,
  EyeOff,
  CheckCircle,
  XCircle,
  ExternalLink,
  Zap,
  Crown,
  TrendingUp,
} from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import {
  useEntitlementStore,
  useAICredits,
  useCanUseAI,
  useCurrentTier,
  TIER_CONFIG,
  type SubscriptionTier,
} from '@stores/entitlementStore';
import { SettingSection } from './shared';

// AI Credits tier configuration
const AI_CREDITS_BY_TIER: Record<SubscriptionTier, { limit: number; label: string }> = {
  free: { limit: 0, label: 'No AI access' },
  solo: { limit: 50, label: '50 credits/month' },
  pro: { limit: 500, label: '500 credits/month' },
  studio: { limit: 2000, label: '2,000 credits/month' },
  business: { limit: -1, label: 'Unlimited' },
};

function AIAccessStatusCard() {
  const canUseAI = useCanUseAI();
  const currentTier = useCurrentTier();
  const aiCredits = useAICredits();
  const status = useEntitlementStore((state) => state.status);
  const tierConfig = TIER_CONFIG[currentTier];

  if (!status?.entitlements_enabled) {
    return (
      <div className="bg-green-500/10 border border-green-500/30 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <CheckCircle size={20} className="text-green-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-green-200 font-medium">AI Features Enabled (Development Mode)</p>
            <p className="text-xs text-green-300/80 mt-1">
              Entitlements are disabled. All AI features are available without restrictions.
            </p>
          </div>
        </div>
      </div>
    );
  }

  if (!aiCredits.hasAccess) {
    return (
      <div className="bg-gray-700/50 border border-gray-600 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <XCircle size={20} className="text-gray-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-gray-200 font-medium">AI Features Not Available</p>
            <p className="text-xs text-gray-400 mt-1">
              Your current plan ({tierConfig.label}) doesn't include AI features.
              Upgrade to Solo or higher to unlock AI-powered workflow generation.
            </p>
          </div>
        </div>
      </div>
    );
  }

  if (canUseAI) {
    return (
      <div className={`${tierConfig.bgColor} border ${tierConfig.borderColor} rounded-lg p-4`}>
        <div className="flex items-start gap-3">
          <CheckCircle size={20} className={`${tierConfig.color} flex-shrink-0 mt-0.5`} />
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <p className={`text-sm ${tierConfig.color} font-medium`}>AI Features Active</p>
              <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${tierConfig.bgColor} ${tierConfig.color} border ${tierConfig.borderColor}`}>
                {tierConfig.label}
              </span>
            </div>
            <p className="text-xs text-gray-400 mt-1">
              {aiCredits.isUnlimited
                ? 'You have unlimited AI credits with your plan.'
                : `You have ${aiCredits.remaining} of ${aiCredits.limit} credits remaining this month.`}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-4">
      <div className="flex items-start gap-3">
        <AlertTriangle size={20} className="text-amber-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="text-sm text-amber-200 font-medium">AI Credits Exhausted</p>
          <p className="text-xs text-amber-300/80 mt-1">
            You've used all your AI credits for this month. Credits reset on {aiCredits.resetDate}.
          </p>
        </div>
      </div>
    </div>
  );
}

function AIUsageMeter() {
  const aiCredits = useAICredits();
  const status = useEntitlementStore((state) => state.status);

  if (!status?.entitlements_enabled || !aiCredits.hasAccess || aiCredits.isUnlimited) {
    return null;
  }

  const getProgressColor = () => {
    if (aiCredits.percentUsed >= 90) return 'bg-red-500';
    if (aiCredits.percentUsed >= 75) return 'bg-amber-500';
    return 'bg-blue-500';
  };

  return (
    <SettingSection title="AI Credits Usage" tooltip="Track your monthly AI credits consumption.">
      <div className="space-y-4">
        {/* Progress bar */}
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span className="text-gray-400">
              {aiCredits.used} / {aiCredits.limit} credits used
            </span>
            <span className="text-gray-400">
              {aiCredits.remaining} remaining
            </span>
          </div>
          <div className="h-3 bg-gray-700 rounded-full overflow-hidden">
            <div
              className={`h-full ${getProgressColor()} transition-all duration-300`}
              style={{ width: `${aiCredits.percentUsed}%` }}
            />
          </div>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-gray-800/50 rounded-lg p-3">
            <div className="text-xs text-gray-500 mb-1">Requests This Month</div>
            <div className="text-lg font-semibold text-surface">{aiCredits.requestsCount}</div>
          </div>
          <div className="bg-gray-800/50 rounded-lg p-3">
            <div className="text-xs text-gray-500 mb-1">Credits Reset</div>
            <div className="text-lg font-semibold text-surface">
              {aiCredits.resetDate ? new Date(aiCredits.resetDate).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) : 'N/A'}
            </div>
          </div>
        </div>

        {/* Warning messages */}
        {aiCredits.percentUsed >= 90 && aiCredits.remaining > 0 && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3">
            <p className="text-xs text-red-300">
              You're running low on AI credits. Consider upgrading your plan for more credits.
            </p>
          </div>
        )}
        {aiCredits.remaining === 0 && (
          <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-3">
            <p className="text-xs text-amber-300">
              You've used all your AI credits for this month. Credits will reset on {aiCredits.resetDate}.
            </p>
          </div>
        )}
      </div>
    </SettingSection>
  );
}

function OpenRouterKeySection() {
  const { apiKeys, setApiKey } = useSettingsStore();
  const [showKey, setShowKey] = useState(false);

  return (
    <SettingSection
      title="OpenRouter API Key (Optional)"
      tooltip="Override the default AI service with your own OpenRouter key."
    >
      <div className="space-y-4">
        <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <Zap size={20} className="text-blue-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm text-blue-200 font-medium">Bring Your Own Key (BYOK)</p>
              <p className="text-xs text-blue-300/80 mt-1">
                Optionally use your own OpenRouter key to access additional models or use your own credits.
                Configure BYOK keys directly in your{' '}
                <a
                  href="https://openrouter.ai/settings/keys"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300 underline inline-flex items-center gap-1"
                >
                  OpenRouter dashboard <ExternalLink size={12} />
                </a>
              </p>
            </div>
          </div>
        </div>

        <div>
          <label className="text-sm text-gray-300 block mb-2">OpenRouter API Key</label>
          <div className="relative">
            <input
              type={showKey ? 'text' : 'password'}
              value={apiKeys.openrouterApiKey}
              onChange={(e) => setApiKey('openrouterApiKey', e.target.value)}
              placeholder="sk-or-v1-..."
              className="w-full px-4 py-3 pr-12 bg-gray-800 border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
            <button
              type="button"
              onClick={() => setShowKey(!showKey)}
              className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-gray-300 transition-colors"
              aria-label={showKey ? 'Hide API key' : 'Show API key'}
            >
              {showKey ? <EyeOff size={18} /> : <Eye size={18} />}
            </button>
          </div>
          <p className="text-xs text-gray-500 mt-2">
            Leave empty to use your plan's included AI credits.
          </p>
        </div>

        <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle size={20} className="text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm text-amber-200 font-medium">Security Notice</p>
              <p className="text-xs text-amber-300/80 mt-1">
                API keys are stored locally in your browser. Never share your API keys with others.
              </p>
            </div>
          </div>
        </div>
      </div>
    </SettingSection>
  );
}

function AITierComparison() {
  const currentTier = useCurrentTier();
  const tiers: SubscriptionTier[] = ['free', 'solo', 'pro', 'studio', 'business'];

  return (
    <SettingSection
      title="AI Credits by Plan"
      tooltip="Compare AI credits across different subscription tiers."
    >
      <div className="space-y-2">
        {tiers.map((tier) => {
          const config = TIER_CONFIG[tier];
          const credits = AI_CREDITS_BY_TIER[tier];
          const isCurrent = tier === currentTier;

          return (
            <div
              key={tier}
              className={`flex items-center justify-between p-3 rounded-lg border transition-colors ${
                isCurrent
                  ? `${config.bgColor} ${config.borderColor}`
                  : 'bg-gray-800/30 border-gray-700 hover:bg-gray-800/50'
              }`}
            >
              <div className="flex items-center gap-3">
                {tier === 'business' ? (
                  <Crown size={18} className={config.color} />
                ) : tier === 'studio' ? (
                  <Sparkles size={18} className={config.color} />
                ) : tier === 'pro' ? (
                  <TrendingUp size={18} className={config.color} />
                ) : (
                  <div className={`w-4 h-4 rounded-full ${isCurrent ? config.bgColor : 'bg-gray-600'} border ${config.borderColor}`} />
                )}
                <span className={`font-medium ${isCurrent ? config.color : 'text-gray-300'}`}>
                  {config.label}
                </span>
                {isCurrent && (
                  <span className="px-2 py-0.5 text-xs rounded-full bg-gray-700 text-gray-300">
                    Current
                  </span>
                )}
              </div>
              <span className={`text-sm ${isCurrent ? config.color : 'text-gray-400'}`}>
                {credits.label}
              </span>
            </div>
          );
        })}
      </div>
    </SettingSection>
  );
}

export function AISettingsSection() {
  const { fetchStatus } = useEntitlementStore();

  // Fetch entitlement status on mount
  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-6">
        <Sparkles size={24} className="text-purple-400" />
        <div>
          <h2 className="text-lg font-semibold text-surface">AI Settings</h2>
          <p className="text-sm text-gray-400">Manage AI features and credits</p>
        </div>
      </div>

      <AIAccessStatusCard />
      <AIUsageMeter />
      <OpenRouterKeySection />
      <AITierComparison />
    </div>
  );
}

export default AISettingsSection;
