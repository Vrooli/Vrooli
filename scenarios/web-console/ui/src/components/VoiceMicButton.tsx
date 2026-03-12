import { Mic, Loader2, AlertCircle } from "lucide-react";
import { cn } from "../lib/classnames";

interface VoiceMicButtonProps {
  supported: boolean;
  isRecording: boolean;
  isTranscribing: boolean;
  error: string | null;
  onToggle: () => void;
}

export default function VoiceMicButton({
  supported,
  isRecording,
  isTranscribing,
  error,
  onToggle,
}: VoiceMicButtonProps) {
  if (!supported) return null;

  const hasError = error !== null && !isRecording && !isTranscribing;

  return (
    <div className="relative shrink-0">
      <button
        data-testid="voice-mic-btn"
        onPointerDown={(e) => e.preventDefault()}
        onClick={onToggle}
        className={cn(
          "relative shrink-0 rounded border px-1.5 py-1 text-xs font-medium transition active:bg-wc-accent-active touch-manipulation",
          isRecording
            ? "border-red-500 bg-red-500/20 text-red-400"
            : hasError
              ? "border-amber-500 bg-amber-500/10 text-amber-400"
              : "border-wc-default bg-wc-surface-input text-wc-text-secondary",
        )}
        title={
          isRecording
            ? "Recording... tap to stop"
            : isTranscribing
              ? "Transcribing..."
              : hasError
                ? `Voice error: ${error}`
                : "Tap to speak"
        }
      >
        {isTranscribing ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
        ) : hasError ? (
          <AlertCircle className="h-3.5 w-3.5" />
        ) : (
          <>
            {isRecording && (
              <span className="absolute -top-0.5 -right-0.5 h-2 w-2 rounded-full bg-red-500 animate-pulse" />
            )}
            <Mic className="h-3.5 w-3.5" />
          </>
        )}
      </button>
      {hasError && (
        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 w-48 z-50 rounded border border-amber-500/50 bg-wc-surface-raised px-2 py-1 text-[10px] text-amber-300 shadow-lg pointer-events-none">
          {error}
        </div>
      )}
    </div>
  );
}
