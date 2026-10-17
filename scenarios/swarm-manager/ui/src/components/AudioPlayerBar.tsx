import { Loader2, Square, Volume2 } from "lucide-react";

export interface AudioPlayerBarProps {
  isSpeaking: boolean;
  loading?: boolean;
  label?: string;
  onStop: () => void;
  className?: string;
}

/** Compact playback state surface shared by message and session hosts. */
export function AudioPlayerBar({ isSpeaking, loading = false, label = "Voice response", onStop, className }: AudioPlayerBarProps) {
  if (!isSpeaking && !loading) return null;
  return (
    <div data-testid="audio-player-bar" data-audio-state="player" role="status" className={["flex items-center gap-2 rounded border border-cyan-500/30 bg-cyan-500/10 px-3 py-2 text-xs text-cyan-100", className ?? ""].join(" ").trim()}>
      {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" /> : <Volume2 className="h-3.5 w-3.5" aria-hidden="true" />}
      <span className="min-w-0 flex-1 truncate">{loading ? `Loading ${label.toLowerCase()}…` : label}</span>
      {isSpeaking && <button type="button" onClick={onStop} className="inline-flex items-center gap-1 rounded border border-cyan-400/40 px-2 py-1 text-cyan-100" aria-label="Stop voice playback"><Square className="h-3 w-3" aria-hidden="true" /> Stop</button>}
    </div>
  );
}

export default AudioPlayerBar;
