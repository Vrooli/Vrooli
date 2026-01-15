/**
 * EntitlementErrorCard Component
 *
 * Renders a professional inline error card for AI entitlement errors.
 * Two variants:
 * - Tier error (AI_NOT_AVAILABLE): User's plan doesn't include AI features
 * - Credit error (INSUFFICIENT_CREDITS): User has exhausted their AI credits
 */

import { formatDistanceToNow } from 'date-fns';
import { Lock, AlertTriangle, ArrowUpRight, RefreshCw } from 'lucide-react';
import { TierBadge } from '@shared/ui/TierBadge';
import { UsageMeter } from '@shared/ui/UsageMeter';
import type { SubscriptionTier } from '@stores/entitlementStore';
import type { EntitlementErrorCode, EntitlementErrorDetails } from './types';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

export interface EntitlementErrorCardProps {
  errorCode: EntitlementErrorCode;
  details?: EntitlementErrorDetails;
}

/**
 * Format the reset date into a human-readable string.
 * Returns: "Resets in X days (Jan 29, 2026)" or fallback message
 */
function formatResetDate(resetDateStr?: string): string {
  if (!resetDateStr) {
    return 'Resets at the start of your billing cycle';
  }

  try {
    const resetDate = new Date(resetDateStr);
    if (Number.isNaN(resetDate.getTime())) {
      return 'Resets at the start of your billing cycle';
    }

    const distance = formatDistanceToNow(resetDate, { addSuffix: false });
    const formattedDate = resetDate.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });

    return `Resets in ${distance} (${formattedDate})`;
  } catch {
    return 'Resets at the start of your billing cycle';
  }
}

/**
 * Get the upgrade URL with email pre-filled if available.
 */
function getUpgradeUrl(): string {
  // Try to get user email from localStorage or entitlement store
  let email = '';
  try {
    const stored = localStorage.getItem('entitlement-user-email');
    if (stored) {
      email = stored;
    }
  } catch {
    // localStorage not available
  }

  const baseUrl = `${LANDING_PAGE_URL}/pricing`;
  return email ? `${baseUrl}?email=${encodeURIComponent(email)}` : baseUrl;
}

function TierErrorCard({ details }: { details?: EntitlementErrorDetails }) {
  const tier = (details?.tier as SubscriptionTier) || 'free';

  return (
    <div className="mb-4 mx-2">
      <div className="rounded-xl border border-amber-500/40 bg-gradient-to-br from-amber-950/40 to-orange-950/30 overflow-hidden shadow-lg">
        {/* Header */}
        <div className="flex items-center gap-2 px-4 py-3 bg-amber-900/30 border-b border-amber-500/30">
          <div className="p-1.5 rounded-lg bg-amber-500/20">
            <Lock size={16} className="text-amber-400" />
          </div>
          <h3 className="text-sm font-semibold text-amber-200">
            AI Features Not Available
          </h3>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          <p className="text-sm text-gray-300">
            Your current plan doesn't include AI-powered browser automation.
            Upgrade to Pro or higher to unlock this feature.
          </p>

          {/* Current plan display */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400">Current Plan:</span>
            <TierBadge tier={tier} size="sm" />
          </div>

          {/* Upgrade CTA */}
          <a
            href={getUpgradeUrl()}
            target="_blank"
            rel="noopener noreferrer"
            className="
              flex items-center justify-center gap-2 w-full px-4 py-2.5 rounded-lg
              bg-gradient-to-r from-purple-600 to-blue-600
              hover:from-purple-500 hover:to-blue-500
              text-white text-sm font-medium
              transition-all shadow-md hover:shadow-purple-500/25
            "
          >
            Upgrade to Pro
            <ArrowUpRight size={16} />
          </a>
        </div>
      </div>
    </div>
  );
}

function CreditErrorCard({ details }: { details?: EntitlementErrorDetails }) {
  const creditsUsed = details?.creditsUsed ?? details?.creditsLimit ?? 0;
  const creditsLimit = details?.creditsLimit ?? 0;
  const resetDateStr = details?.resetDate;
  const tier = (details?.tier as SubscriptionTier) || 'free';

  // For display purposes, show 100% if we have a limit
  const displayUsed = creditsLimit > 0 ? creditsLimit : creditsUsed;

  return (
    <div className="mb-4 mx-2">
      <div className="rounded-xl border border-red-500/40 bg-gradient-to-br from-red-950/40 to-rose-950/30 overflow-hidden shadow-lg">
        {/* Header */}
        <div className="flex items-center gap-2 px-4 py-3 bg-red-900/30 border-b border-red-500/30">
          <div className="p-1.5 rounded-lg bg-red-500/20">
            <AlertTriangle size={16} className="text-red-400" />
          </div>
          <h3 className="text-sm font-semibold text-red-200">
            AI Credits Exhausted
          </h3>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          <p className="text-sm text-gray-300">
            You've used all your AI credits for this month.
          </p>

          {/* Usage meter */}
          {creditsLimit > 0 && (
            <div className="space-y-1">
              <UsageMeter
                used={displayUsed}
                limit={creditsLimit}
                label="AI Credits"
                size="sm"
              />
            </div>
          )}

          {/* Reset date */}
          <div className="flex items-center gap-2 text-xs text-gray-400">
            <RefreshCw size={12} className="text-gray-500" />
            <span>{formatResetDate(resetDateStr)}</span>
          </div>

          {/* Current plan display */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400">Current Plan:</span>
            <TierBadge tier={tier} size="sm" />
          </div>

          {/* Upgrade CTA */}
          <a
            href={getUpgradeUrl()}
            target="_blank"
            rel="noopener noreferrer"
            className="
              flex items-center justify-center gap-2 w-full px-4 py-2.5 rounded-lg
              bg-gradient-to-r from-purple-600 to-blue-600
              hover:from-purple-500 hover:to-blue-500
              text-white text-sm font-medium
              transition-all shadow-md hover:shadow-purple-500/25
            "
          >
            Upgrade for More Credits
            <ArrowUpRight size={16} />
          </a>
        </div>
      </div>
    </div>
  );
}

export function EntitlementErrorCard({ errorCode, details }: EntitlementErrorCardProps) {
  if (errorCode === 'AI_NOT_AVAILABLE') {
    return <TierErrorCard details={details} />;
  }

  if (errorCode === 'INSUFFICIENT_CREDITS') {
    return <CreditErrorCard details={details} />;
  }

  // Fallback (should not happen)
  return null;
}

export default EntitlementErrorCard;
