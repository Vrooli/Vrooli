import { useCallback, useState } from "react";
import { Loader2, Volume2, X } from "lucide-react";

interface EnableAudioBannerProps {
  /**
   * Called when the user clicks "Enable voice". Returns `true` if unlock
   * succeeded (caller should then replay any pending event). Returns
   * `false` if the browser still rejected playback.
   */
  onEnable: () => Promise<boolean>;
  /** Called when the user dismisses without enabling. */
  onDismiss: () => void;
}

/**
 * Persistent affordance shown when auto-TTS playback was rejected by the
 * browser's autoplay policy and no qualifying user gesture has unlocked the
 * audio element yet. Clicking "Enable voice" runs a silent play() from the
 * click's gesture stack, activating the media element for the rest of the
 * session.
 */
export default function EnableAudioBanner({ onEnable, onDismiss }: EnableAudioBannerProps) {
  const [enabling, setEnabling] = useState(false);

  const handleEnable = useCallback(async () => {
    if (enabling) return;
    setEnabling(true);
    try {
      await onEnable();
    } finally {
      setEnabling(false);
    }
  }, [enabling, onEnable]);

  const handleDismiss = useCallback(() => {
    if (enabling) return;
    onDismiss();
  }, [enabling, onDismiss]);

  return (
    <div
      data-testid="enable-audio-banner"
      className="flex items-start gap-2 border-b border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200"
      role="status"
    >
      <Volume2 className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
      <div className="flex-1 min-w-0">
        <div className="font-medium">Voice playback is blocked</div>
        <div className="mt-0.5 break-words text-amber-200/80">
          Your browser requires a click before it will play audio in this tab.
        </div>
      </div>
      <button
        type="button"
        data-testid="enable-audio-banner-enable"
        onClick={handleEnable}
        disabled={enabling}
        className="shrink-0 inline-flex items-center gap-1 rounded border border-amber-400/40 bg-amber-500/20 px-2 py-1 font-medium text-amber-100 transition active:bg-amber-500/30 disabled:cursor-not-allowed disabled:opacity-60"
        title="Enable voice for this session"
        aria-label="Enable voice"
      >
        {enabling ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
        ) : (
          <Volume2 className="h-3.5 w-3.5" aria-hidden />
        )}
        <span>{enabling ? "Enabling…" : "Enable voice"}</span>
      </button>
      <button
        type="button"
        data-testid="enable-audio-banner-dismiss"
        onClick={handleDismiss}
        disabled={enabling}
        className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1 text-wc-text-secondary transition active:bg-wc-accent-active disabled:cursor-not-allowed disabled:opacity-60"
        title="Dismiss"
        aria-label="Dismiss"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
