import { useCallback, useState } from "react";
import { Loader2, Volume2, X } from "lucide-react";

export interface EnableAudioBannerProps {
  onEnable: () => Promise<boolean>;
  onDismiss: () => void;
}

/** User-gesture affordance shown after browser autoplay blocks TTS. */
export function EnableAudioBanner({ onEnable, onDismiss }: EnableAudioBannerProps) {
  const [enabling, setEnabling] = useState(false);
  const handleEnable = useCallback(async () => {
    if (enabling) return;
    setEnabling(true);
    try { await onEnable(); } finally { setEnabling(false); }
  }, [enabling, onEnable]);

  return (
    <div data-testid="enable-audio-banner" data-audio-state="enable-audio" role="status" className="flex items-start gap-2 border-b border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
      <Volume2 className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <div className="font-medium">Enable voice playback</div>
        <div className="mt-0.5 text-amber-200/80">Your browser blocked automatic audio. Enable it to hear responses.</div>
      </div>
      <button type="button" onClick={() => void handleEnable()} disabled={enabling} className="inline-flex shrink-0 items-center gap-1 rounded border border-amber-400/40 bg-amber-500/20 px-2 py-1 font-medium text-amber-100 disabled:opacity-60" aria-label="Enable voice">
        {enabling ? <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" /> : <Volume2 className="h-3.5 w-3.5" aria-hidden="true" />}
        <span>{enabling ? "Enabling…" : "Enable voice"}</span>
      </button>
      <button type="button" onClick={onDismiss} disabled={enabling} className="shrink-0 rounded border border-slate-600 bg-slate-800 p-1 text-slate-300 disabled:opacity-60" aria-label="Dismiss audio notice">
        <X className="h-3.5 w-3.5" aria-hidden="true" />
      </button>
    </div>
  );
}

export default EnableAudioBanner;
