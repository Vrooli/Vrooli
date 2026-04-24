// DOC: docs/plans/stt-voice-filter-retry-implementation-plan.md §7 point 5
//
// VoiceRejectionBanner — persistent notice shown when speaker verification
// has rejected the user's last turn. Replaces the earlier auto-dismissing
// sky-blue notice. The banner stays up until the user either:
//   1. Clicks "Transcribe anyway" and the server successfully transcribes
//      the retained audio without the verification filter, or
//   2. Clicks "Dismiss", or
//   3. The 5-minute retention TTL (owned by `useVoiceInput`) fires.
//
// Two rejection kinds are rendered differently:
//   - "retryable": we have the audio blob — show Transcribe-anyway + Dismiss.
//   - "explanatory": audio was not retained (e.g. Web Speech API) — show
//     Dismiss only with a short reason string.
//
// Status transitions inside a `retryable` rejection:
//   idle → retrying → (settled: banner closes) | failed → idle/retrying

import { useCallback } from "react";
import { AlertTriangle, X, RotateCcw, Loader2 } from "lucide-react";
import type { VoiceRejection } from "../hooks/voice/types";

interface VoiceRejectionBannerProps {
  rejection: VoiceRejection;
  /** Trigger a "Transcribe anyway" retry. Ignored for explanatory kind. */
  onRetry: () => void;
  /** Dismiss the banner and release any retained audio. */
  onDismiss: () => void;
}

/** Format a duration like 12_500 → "12.5s". */
function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  const rem = Math.round(seconds - minutes * 60);
  return `${minutes}m${rem.toString().padStart(2, "0")}s`;
}

export default function VoiceRejectionBanner({
  rejection,
  onRetry,
  onDismiss,
}: VoiceRejectionBannerProps) {
  const handleRetry = useCallback(() => {
    onRetry();
  }, [onRetry]);

  const handleDismiss = useCallback(() => {
    onDismiss();
  }, [onDismiss]);

  if (rejection.kind === "explanatory") {
    return (
      <div
        data-testid="voice-rejection-banner"
        data-kind="explanatory"
        className="flex items-start gap-2 border-b border-sky-500/30 bg-sky-500/10 px-3 py-2 text-xs text-sky-200"
        role="status"
      >
        <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
        <div className="flex-1">
          <div className="font-medium">Speech didn&apos;t match your voice</div>
          <div className="mt-0.5 text-sky-200/80">
            {rejection.reason} (score {rejection.score.toFixed(2)} &lt; threshold {rejection.threshold.toFixed(2)})
          </div>
        </div>
        <button
          data-testid="voice-rejection-dismiss"
          onClick={handleDismiss}
          className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1 text-wc-text-secondary transition active:bg-wc-accent-active"
          title="Dismiss"
          aria-label="Dismiss rejection notice"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
    );
  }

  const isRetrying = rejection.status === "retrying";
  const isFailed = rejection.status === "failed";
  const primaryLabel = isFailed ? "Retry" : "Transcribe anyway";

  return (
    <div
      data-testid="voice-rejection-banner"
      data-kind="retryable"
      data-status={rejection.status}
      className="flex items-start gap-2 border-b border-sky-500/30 bg-sky-500/10 px-3 py-2 text-xs text-sky-200"
      role="status"
    >
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
      <div className="flex-1 min-w-0">
        <div className="font-medium">Speech didn&apos;t match your voice</div>
        <div className="mt-0.5 text-sky-200/80">
          Score {rejection.score.toFixed(2)} &lt; threshold {rejection.threshold.toFixed(2)}
          {" · "}
          {formatDurationMs(rejection.durationMs)} retained
        </div>
        {isFailed && rejection.errorMessage ? (
          <div
            data-testid="voice-rejection-error"
            className="mt-1 text-rose-300"
          >
            {rejection.errorMessage}
          </div>
        ) : null}
      </div>
      <button
        data-testid="voice-rejection-retry"
        onClick={handleRetry}
        disabled={isRetrying}
        className="shrink-0 inline-flex items-center gap-1 rounded border border-sky-400/40 bg-sky-500/20 px-2 py-1 font-medium text-sky-100 transition active:bg-sky-500/30 disabled:cursor-not-allowed disabled:opacity-60"
        title="Transcribe the audio without speaker verification"
      >
        {isRetrying ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
        ) : (
          <RotateCcw className="h-3.5 w-3.5" aria-hidden />
        )}
        <span>{isRetrying ? "Transcribing…" : primaryLabel}</span>
      </button>
      <button
        data-testid="voice-rejection-dismiss"
        onClick={handleDismiss}
        disabled={isRetrying}
        className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1 text-wc-text-secondary transition active:bg-wc-accent-active disabled:cursor-not-allowed disabled:opacity-60"
        title="Dismiss"
        aria-label="Dismiss rejection notice"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
