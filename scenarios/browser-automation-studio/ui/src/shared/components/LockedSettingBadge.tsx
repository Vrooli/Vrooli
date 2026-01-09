/**
 * LockedSettingBadge - Shows a lock indicator with upgrade CTA for entitlement-gated features
 */

import { Lock, ArrowRight } from 'lucide-react';
import type { SubscriptionTier } from '@stores/entitlementStore';
import { TIER_CONFIG } from '@stores/entitlementStore';

export interface LockedSettingBadgeProps {
  /** The minimum tier required to unlock this feature */
  requiredTier: SubscriptionTier;
  /** Optional custom message */
  message?: string;
  /** Whether to show the upgrade link */
  showUpgradeLink?: boolean;
  /** Callback when upgrade is clicked (defaults to navigating to subscription tab) */
  onUpgradeClick?: () => void;
  /** Size variant */
  size?: 'sm' | 'md';
  /** Additional class names */
  className?: string;
}

export function LockedSettingBadge({
  requiredTier,
  message,
  showUpgradeLink = true,
  onUpgradeClick,
  size = 'md',
  className = '',
}: LockedSettingBadgeProps) {
  const tierConfig = TIER_CONFIG[requiredTier];
  const tierLabel = tierConfig?.label ?? requiredTier;

  const handleUpgradeClick = () => {
    if (onUpgradeClick) {
      onUpgradeClick();
    } else {
      // Default: Navigate to subscription tab in settings
      // Using hash navigation which is common in SPAs
      window.location.hash = 'settings/subscription';
    }
  };

  const sizeClasses = {
    sm: {
      container: 'px-2 py-1 gap-1.5',
      icon: 12,
      text: 'text-xs',
      arrow: 10,
    },
    md: {
      container: 'px-3 py-2 gap-2',
      icon: 14,
      text: 'text-sm',
      arrow: 12,
    },
  };

  const styles = sizeClasses[size];

  return (
    <div
      className={`
        inline-flex items-center rounded-lg
        bg-amber-950/30 border border-amber-500/30
        ${styles.container}
        ${className}
      `}
    >
      <Lock size={styles.icon} className="text-amber-400 flex-shrink-0" />
      <span className={`${styles.text} text-amber-200/90`}>
        {message ?? `Requires ${tierLabel}`}
      </span>
      {showUpgradeLink && (
        <button
          type="button"
          onClick={handleUpgradeClick}
          className={`
            ${styles.text} text-amber-400 hover:text-amber-300
            inline-flex items-center gap-1
            hover:underline transition-colors
            ml-1
          `}
        >
          Upgrade
          <ArrowRight size={styles.arrow} />
        </button>
      )}
    </div>
  );
}

/**
 * Inline locked indicator for toggle switches and small controls
 */
export function LockedIndicator({
  requiredTier,
  className = '',
}: {
  requiredTier: SubscriptionTier;
  className?: string;
}) {
  const tierConfig = TIER_CONFIG[requiredTier];
  const tierLabel = tierConfig?.label ?? requiredTier;

  return (
    <span
      className={`inline-flex items-center gap-1 text-xs text-amber-400/80 ${className}`}
      title={`Requires ${tierLabel} or higher`}
    >
      <Lock size={10} />
      <span>{tierLabel}+</span>
    </span>
  );
}

/**
 * Full-width locked overlay for sections
 */
export function LockedOverlay({
  requiredTier,
  featureName,
  description,
  onUpgradeClick,
  className = '',
}: {
  requiredTier: SubscriptionTier;
  featureName: string;
  description?: string;
  onUpgradeClick?: () => void;
  className?: string;
}) {
  const tierConfig = TIER_CONFIG[requiredTier];
  const tierLabel = tierConfig?.label ?? requiredTier;

  const handleUpgradeClick = () => {
    if (onUpgradeClick) {
      onUpgradeClick();
    } else {
      window.location.hash = 'settings/subscription';
    }
  };

  return (
    <div
      className={`
        relative rounded-xl border border-amber-500/20 bg-amber-950/10
        p-6 text-center
        ${className}
      `}
    >
      <div className="flex flex-col items-center gap-3">
        <div className="p-3 rounded-full bg-amber-500/10 border border-amber-500/20">
          <Lock size={24} className="text-amber-400" />
        </div>
        <div>
          <h4 className="text-sm font-medium text-amber-200">
            {featureName} requires {tierLabel}
          </h4>
          {description && (
            <p className="text-xs text-amber-200/60 mt-1 max-w-xs mx-auto">
              {description}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={handleUpgradeClick}
          className="
            inline-flex items-center gap-2 px-4 py-2
            text-sm font-medium text-amber-900
            bg-gradient-to-r from-amber-400 to-amber-500
            hover:from-amber-300 hover:to-amber-400
            rounded-lg transition-all
            shadow-lg shadow-amber-500/20
          "
        >
          Upgrade to {tierLabel}
          <ArrowRight size={14} />
        </button>
      </div>
    </div>
  );
}

export default LockedSettingBadge;
