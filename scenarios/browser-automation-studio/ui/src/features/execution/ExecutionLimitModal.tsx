import { AlertTriangle, Crown, Zap, ArrowUpRight, X } from 'lucide-react';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import { UsageMeter } from '@shared/ui';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

interface ExecutionLimitModalProps {
  isOpen: boolean;
  onClose: () => void;
  onOpenSettings?: () => void;
}

export function ExecutionLimitModal({ isOpen, onClose, onOpenSettings }: ExecutionLimitModalProps) {
  const { status, userEmail } = useEntitlementStore();

  if (!status) {
    return null;
  }

  const tierConfig = TIER_CONFIG[status.tier];
  const upgradeUrl = userEmail
    ? `${LANDING_PAGE_URL}/pricing?email=${encodeURIComponent(userEmail)}`
    : `${LANDING_PAGE_URL}/pricing`;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabel="Execution limit reached"
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
          <div className="p-3 rounded-full bg-red-900/30">
            <AlertTriangle size={24} className="text-red-400" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-surface">
              Monthly Limit Reached
            </h2>
            <p className="text-sm text-gray-400">
              You&apos;ve used all your workflow executions for this month
            </p>
          </div>
        </div>

        {/* Usage Meter */}
        <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700 mb-4">
          <UsageMeter
            used={status.monthly_used}
            limit={status.monthly_limit}
            label="Monthly Executions"
            size="lg"
          />
        </div>

        {/* Current Plan Info */}
        <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30 border border-gray-700 mb-6">
          <div className="flex items-center gap-2">
            <Zap size={16} className={tierConfig.color} />
            <span className="text-sm text-gray-300">Current Plan:</span>
          </div>
          <span className={`font-medium ${tierConfig.color}`}>
            {tierConfig.label} ({status.monthly_limit} executions/month)
          </span>
        </div>

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
            Upgrade to Get More Executions
            <ArrowUpRight size={16} />
          </a>

          {onOpenSettings && (
            <button
              onClick={() => {
                onClose();
                onOpenSettings();
              }}
              className="
                w-full px-4 py-2 rounded-lg
                bg-gray-700 hover:bg-gray-600
                text-gray-300 text-sm
                transition-colors
              "
            >
              Manage Subscription Settings
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

        {/* Help Text */}
        <p className="text-xs text-gray-500 text-center mt-4">
          Your execution limit resets at the beginning of each billing period.
        </p>
      </div>
    </ResponsiveDialog>
  );
}

export default ExecutionLimitModal;
