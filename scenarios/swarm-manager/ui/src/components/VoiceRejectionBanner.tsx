import { AlertTriangle, X } from "lucide-react";
import type { VoiceRejection } from "../audio-integration";

export interface VoiceRejectionBannerProps {
  rejection: VoiceRejection;
  onRetry?: () => void;
  onDismiss: () => void;
}

/** Recoverable voice-turn outcome; it never turns a lost turn into silence. */
export function VoiceRejectionBanner({ rejection, onRetry, onDismiss }: VoiceRejectionBannerProps) {
  const retryable = rejection.kind === "retryable";
  const message = retryable
    ? rejection.cause === "empty-transcript" ? "No transcript arrived for that turn." : "Speaker verification rejected that turn."
    : rejection.reason;
  return (
    <div data-testid="voice-rejection-banner" data-audio-state="rejection" role="alert" className="flex items-center gap-2 rounded border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-100">
      <AlertTriangle className="h-3.5 w-3.5 shrink-0 text-amber-300" aria-hidden="true" />
      <span className="min-w-0 flex-1">{message}</span>
      {retryable && onRetry && <button type="button" onClick={onRetry} disabled={rejection.status === "retrying"} className="rounded border border-amber-400/40 px-2 py-1">Retry</button>}
      <button type="button" onClick={onDismiss} className="rounded p-1 text-amber-200" aria-label="Dismiss voice notice"><X className="h-3.5 w-3.5" aria-hidden="true" /></button>
    </div>
  );
}

export default VoiceRejectionBanner;
