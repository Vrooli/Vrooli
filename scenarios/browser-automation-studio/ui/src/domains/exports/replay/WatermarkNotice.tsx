import { useState, useEffect } from 'react';
import { Crown, X, ArrowUpRight } from 'lucide-react';
import { useRequiresWatermark, useIsEntitlementsEnabled, useCurrentTier } from '@hooks/useEntitlement';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

// Session storage key for dismissal
const DISMISSED_KEY = 'bas.watermark_notice_dismissed';

interface WatermarkNoticeProps {
  /** Optional callback to open settings panel */
  onOpenSettings?: () => void;
}

/**
 * WatermarkNotice - Subtle banner shown to free/solo users on export pages
 * Informs users that their exports will include a watermark.
 * Dismissable per session.
 */
export function WatermarkNotice({ onOpenSettings }: WatermarkNoticeProps) {
  const [isDismissed, setIsDismissed] = useState(false);
  const requiresWatermark = useRequiresWatermark();
  const entitlementsEnabled = useIsEntitlementsEnabled();
  const currentTier = useCurrentTier();
  const { userEmail } = useEntitlementStore();

  // Check session storage on mount
  useEffect(() => {
    const dismissed = sessionStorage.getItem(DISMISSED_KEY);
    if (dismissed === 'true') {
      setIsDismissed(true);
    }
  }, []);

  // Don't show if:
  // - Entitlements are disabled
  // - User doesn't require watermark
  // - Already dismissed this session
  if (!entitlementsEnabled || !requiresWatermark || isDismissed) {
    return null;
  }

  const handleDismiss = () => {
    setIsDismissed(true);
    sessionStorage.setItem(DISMISSED_KEY, 'true');
  };

  const tierConfig = TIER_CONFIG[currentTier];
  const upgradeUrl = userEmail
    ? `${LANDING_PAGE_URL}/pricing?email=${encodeURIComponent(userEmail)}`
    : `${LANDING_PAGE_URL}/pricing`;

  return (
    <div className="bg-gradient-to-r from-purple-900/20 to-blue-900/20 border border-purple-500/30 rounded-lg px-4 py-3">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <Crown size={18} className="text-purple-400 flex-shrink-0" />
          <div className="text-sm">
            <span className="text-gray-300">
              Your exports include a{' '}
              <span className="text-purple-300 font-medium">Browser Automation Studio</span>{' '}
              watermark.
            </span>
            <span className="text-gray-400 ml-1">
              ({tierConfig.label} plan)
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          <a
            href={upgradeUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="
              flex items-center gap-1.5 px-3 py-1.5 rounded-lg
              bg-purple-600/30 hover:bg-purple-600/50
              text-purple-200 hover:text-purple-100
              text-xs font-medium
              transition-colors
            "
          >
            Remove Watermark
            <ArrowUpRight size={12} />
          </a>

          {onOpenSettings && (
            <button
              onClick={onOpenSettings}
              className="
                px-2 py-1.5 rounded-lg
                text-gray-400 hover:text-gray-300 hover:bg-gray-700
                text-xs
                transition-colors
              "
            >
              Settings
            </button>
          )}

          <button
            onClick={handleDismiss}
            className="p-1 text-gray-500 hover:text-gray-300 transition-colors"
            aria-label="Dismiss notice"
          >
            <X size={16} />
          </button>
        </div>
      </div>
    </div>
  );
}

export default WatermarkNotice;
