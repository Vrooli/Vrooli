import { useRef, useLayoutEffect, useState, useCallback } from "react";
import { createPortal } from "react-dom";
import { Mic, Loader2, AlertCircle } from "lucide-react";
import { cn } from "../lib/classnames";
import type { StartRecordingOpts } from "../hooks/useVoiceInput";

/** Hold duration (ms) that distinguishes tap-to-toggle from push-to-talk. */
const LONG_PRESS_MS = 300;

interface VoiceMicButtonProps {
  supported: boolean;
  isRecording: boolean;
  isTranscribing: boolean;
  error: string | null;
  /** 0–1 audio level for live mic visualization */
  audioLevel?: number;
  /** Live partial transcript from streaming transcription. */
  partialTranscript?: string;
  /** Active voice backend, shown in tooltip for diagnostics. */
  backend?: string;
  onStart: (opts?: StartRecordingOpts) => void;
  onStop: () => void;
  /** Extra classes for the outer wrapper (e.g. to control height from a grid parent). */
  className?: string;
  /** Extra classes for the inner button element. */
  buttonClassName?: string;
}

/** Fixed-position tooltip rendered via portal so it can't be clipped by overflow parents. */
function ErrorTooltip({ anchor, text }: { anchor: HTMLElement; text: string }) {
  const [style, setStyle] = useState<React.CSSProperties>({ visibility: "hidden" });
  const tooltipRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const rect = anchor.getBoundingClientRect();
    const el = tooltipRef.current;
    if (!el) return;

    const tooltipRect = el.getBoundingClientRect();
    const pad = 4;

    // Try above the button first, fall back to below if no room
    let top: number;
    if (rect.top - tooltipRect.height - pad >= pad) {
      top = rect.top - tooltipRect.height - pad;
    } else {
      top = rect.bottom + pad;
    }

    // Center horizontally on the button, clamped to viewport
    let left = rect.left + rect.width / 2 - tooltipRect.width / 2;
    left = Math.max(pad, Math.min(left, window.innerWidth - tooltipRect.width - pad));

    setStyle({ position: "fixed", top, left, visibility: "visible" });
  }, [anchor, text]);

  return createPortal(
    <div
      ref={tooltipRef}
      className="z-[9999] w-48 rounded border border-amber-500/50 bg-wc-surface-raised px-2 py-1 text-[10px] text-amber-300 shadow-lg pointer-events-none"
      style={style}
    >
      {text}
    </div>,
    document.body,
  );
}

export default function VoiceMicButton({
  supported,
  isRecording,
  isTranscribing,
  error,
  audioLevel = 0,
  partialTranscript,
  backend,
  onStart,
  onStop,
  className: wrapperClassName,
  buttonClassName,
}: VoiceMicButtonProps) {
  const btnRef = useRef<HTMLButtonElement>(null);
  const pressStartRef = useRef(0);
  /** Tracks the intent of the current pointer interaction to avoid stale-closure races. */
  const pressIntentRef = useRef<"start" | "stop" | "none">("none");

  const handlePointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    pressStartRef.current = Date.now();
    if (isTranscribing) {
      pressIntentRef.current = "none";
    } else if (isRecording) {
      pressIntentRef.current = "stop";
    } else {
      pressIntentRef.current = "start";
      onStart({ vadEnabled: true });
    }
  }, [isRecording, isTranscribing, onStart]);

  const handlePointerUp = useCallback(() => {
    const intent = pressIntentRef.current;
    pressIntentRef.current = "none";
    if (intent === "stop") {
      onStop();
    } else if (intent === "start" && Date.now() - pressStartRef.current >= LONG_PRESS_MS) {
      // Long press release — push-to-talk: stop recording
      onStop();
    }
    // Short press on "start" — tap-to-toggle: keep recording
  }, [onStop]);

  if (!supported) return null;

  const hasError = error !== null && !isRecording && !isTranscribing;

  return (
    <div className={cn("relative shrink-0", wrapperClassName)}>
      <button
        ref={btnRef}
        data-testid="voice-mic-btn"
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        className={cn(
          "relative shrink-0 rounded border px-1.5 py-1 text-xs font-medium transition active:bg-wc-accent-active touch-manipulation overflow-hidden",
          buttonClassName,
          isRecording
            ? "border-red-500 bg-red-500/20 text-red-400"
            : isTranscribing
              ? "border-blue-500 bg-blue-500/20 text-blue-400"
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
                : `Tap to speak${backend ? ` (${backend === "whisper" ? "Whisper" : "Browser"})` : ""}`
        }
      >
        {/* Audio level fill — rises from bottom */}
        {isRecording && (
          <span
            className="absolute inset-x-0 bottom-0 bg-red-500/30 rounded-[inherit] transition-[height] duration-75"
            style={{ height: `${Math.round(audioLevel * 100)}%` }}
          />
        )}
        {isTranscribing ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin relative" />
        ) : hasError ? (
          <AlertCircle className="h-3.5 w-3.5 relative" />
        ) : (
          <Mic className="h-3.5 w-3.5 relative" />
        )}
      </button>
      {hasError && btnRef.current && (
        <ErrorTooltip anchor={btnRef.current} text={error as string} />
      )}
      {isRecording && partialTranscript && btnRef.current && (
        <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-1 max-w-[200px] rounded border border-wc-default bg-wc-surface-raised px-2 py-1 text-[10px] text-wc-text-secondary shadow-lg pointer-events-none whitespace-nowrap overflow-hidden text-ellipsis">
          {partialTranscript}
        </div>
      )}
    </div>
  );
}
