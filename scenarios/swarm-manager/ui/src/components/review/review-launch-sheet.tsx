/**
 * ReviewLaunchSheet — BottomSheet for choosing review type.
 *
 * Presents two options:
 * - Full Review: triggers complete finalization (restart → health → GCT → evidence)
 * - Gather Evidence Only: spawns the review agent without re-running finalization
 *
 * DOC: docs/internal/SEAMS.md#review-launch-sheet
 */

import { AlertTriangle, Check, Info, Loader2, RefreshCw, Search } from "lucide-react";
import { BottomSheet } from "../ui/bottom-sheet";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";

export interface ReviewLaunchSheetProps {
  isOpen: boolean;
  onClose: () => void;
  onFullReview: () => void;
  onGatherEvidence: () => void;
  isTriggering: boolean;
  isTriggeringEvidence: boolean;
  /** Whether a prior finalization exists (enables "Gather Evidence Only"). */
  hasExistingFinalization: boolean;
  /** Whether the review agent policy is enabled. */
  reviewAgentEnabled: boolean;
  /** Error message from the last trigger attempt, if any. */
  triggerError?: string | null;
}

function OptionCard({
  icon,
  title,
  description,
  badge,
  estimate,
  disabled,
  loading,
  onClick,
  testId,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  badge?: React.ReactNode;
  estimate: string;
  disabled?: boolean;
  loading?: boolean;
  onClick: () => void;
  testId: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || loading}
      data-testid={testId}
      className={cn(
        "flex w-full items-start gap-3 rounded-lg border px-4 py-3 text-left transition-colors",
        disabled
          ? "cursor-not-allowed border-slate-700/50 bg-slate-800/30 opacity-50"
          : "border-slate-700 bg-slate-800/50 hover:border-slate-600 hover:bg-slate-800",
      )}
    >
      <div className="mt-0.5 shrink-0 text-slate-400">
        {loading ? <Loader2 className="h-5 w-5 animate-spin" /> : icon}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium text-slate-200">{title}</p>
          {badge}
        </div>
        <p className="mt-0.5 text-xs leading-relaxed text-slate-400">{description}</p>
        <p className="mt-1 text-[11px] text-slate-500">{estimate}</p>
      </div>
    </button>
  );
}

function AgentStatusBar({ enabled }: { enabled: boolean }) {
  return (
    <div
      className={cn(
        "flex items-start gap-2 rounded-md border px-3 py-2",
        enabled
          ? "border-emerald-500/20 bg-emerald-500/5"
          : "border-amber-500/20 bg-amber-500/5",
      )}
      data-testid="review-agent-status-bar"
    >
      {enabled ? (
        <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-400" />
      ) : (
        <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-400" />
      )}
      <div className="min-w-0">
        <p className={cn("text-xs font-medium", enabled ? "text-emerald-300" : "text-amber-300")}>
          Review agent {enabled ? "enabled" : "disabled"}
        </p>
        <p className="mt-0.5 text-[11px] text-slate-400">
          {enabled
            ? "Both options will spawn an AI agent to gather evidence."
            : "Enable the review agent in Settings to gather evidence automatically."}
        </p>
      </div>
    </div>
  );
}

export function ReviewLaunchSheet({
  isOpen,
  onClose,
  onFullReview,
  onGatherEvidence,
  isTriggering,
  isTriggeringEvidence,
  hasExistingFinalization,
  reviewAgentEnabled,
  triggerError,
}: ReviewLaunchSheetProps) {
  const anyLoading = isTriggering || isTriggeringEvidence;

  return (
    <BottomSheet
      isOpen={isOpen}
      onClose={onClose}
      title="Run Review"
      description="Choose the type of review to run for this execution."
      data-testid={selectors.review.launchSheet}
    >
      <div className="space-y-3">
        <AgentStatusBar enabled={reviewAgentEnabled} />

        <OptionCard
          icon={<RefreshCw className="h-5 w-5" />}
          title="Full Review"
          description="Restarts affected scenarios, runs health checks, and code review (GCT). If the review agent is enabled, also gathers evidence."
          estimate={reviewAgentEnabled ? "Estimated: 5-15 minutes" : "Estimated: 2-5 minutes"}
          onClick={onFullReview}
          loading={isTriggering}
          disabled={anyLoading}
          testId={selectors.review.launchSheetFullReview}
        />

        <OptionCard
          icon={<Search className="h-5 w-5" />}
          title="Gather Evidence Only"
          description="Spawns the review agent to gather screenshots, test results, and other verification artifacts. Skips scenario restarts and health checks."
          estimate="Estimated: 3-10 minutes"
          onClick={onGatherEvidence}
          loading={isTriggeringEvidence}
          disabled={anyLoading || !hasExistingFinalization || !reviewAgentEnabled}
          testId={selectors.review.launchSheetGatherEvidence}
        />

        {!hasExistingFinalization && (
          <p className="text-center text-[11px] text-slate-500">
            Run a full review first before gathering additional evidence.
          </p>
        )}
        {hasExistingFinalization && !reviewAgentEnabled && (
          <p className="text-center text-[11px] text-slate-500">
            Enable the review agent in Settings to gather evidence.
          </p>
        )}

        {triggerError && (
          <div
            className="flex items-start gap-2 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2"
            data-testid="review-trigger-error"
          >
            <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-red-400" />
            <div className="min-w-0">
              <p className="text-xs font-medium text-red-300">Review failed</p>
              <p className="mt-0.5 text-[11px] leading-relaxed text-red-300/80">{triggerError}</p>
            </div>
          </div>
        )}
      </div>
    </BottomSheet>
  );
}
