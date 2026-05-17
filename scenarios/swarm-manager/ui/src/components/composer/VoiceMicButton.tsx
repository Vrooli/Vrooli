// VoiceMicButton — swarm-manager's port of web-console's VoiceMicButton.
// Pure presentation: takes VAD-driven activity snapshot + recording state
// and renders the mic button with audio-level fill + auto-stop ring. The
// host (MicButton.tsx) owns useVoiceCore and feeds these props in.

import { memo, useRef, useLayoutEffect, useState, useCallback, useEffect } from "react";
import { createPortal } from "react-dom";
import { Mic, Loader2, AlertCircle, X } from "lucide-react";

import { cn } from "../../lib/utils";
import type { StartRecordingOpts, VoiceActivitySnapshot, ServerVadStateSnapshot } from "../../audio-integration";
import { VAD_AUTO_STOP_VISUAL_GRACE_MS, useServerVadStateStore, SERVER_VAD_STALE_MS } from "../../audio-integration";

/** Hold duration (ms) that distinguishes tap-to-toggle from push-to-talk. */
const LONG_PRESS_MS = 300;

/**
 * Grace period (ms) after entering "transcribing" state during which taps are
 * treated as no-ops instead of cancels.  This prevents the race where VAD
 * auto-stops recording at the same instant the user taps to stop — without the
 * guard the tap would land on the new "transcribing" state and discard the
 * pending transcript.
 */
const TRANSCRIBING_GRACE_MS = 400;
const AUTO_STOP_RING_CIRCUMFERENCE = 2 * Math.PI * 18;

interface VoiceMicButtonProps {
  supported: boolean;
  isPreparing: boolean;
  isRecording: boolean;
  /** True when persistent voice mode is active (distinct from one-shot recording). */
  isListening?: boolean;
  /** True when passive wake word listening is active (mic open, no streaming). */
  isPassive?: boolean;
  isTranscribing: boolean;
  error: string | null;
  /** 0-1 audio level for live mic visualization */
  audioLevel?: number;
  /** VAD-derived snapshot used for the auto-stop countdown visualization. */
  voiceActivity?: VoiceActivitySnapshot;
  /**
   * Latest server-emitted VAD-state snapshot. When fresh (<250 ms since
   * receivedAt) the ring renders server-derived silence progress with light
   * interpolation between ticks; stale snapshots fall back to voiceActivity.
   * See plan: server-driven-mic-ring-streamvadstate-event.md.
   */
  serverVad?: ServerVadStateSnapshot;
  /** Live partial transcript from streaming transcription. */
  partialTranscript?: string;
  /** Active voice backend, shown in tooltip for diagnostics. */
  backend?: string;
  onStart: (opts?: StartRecordingOpts) => void;
  onStop: () => void;
  onCancel?: () => void;
  /** Exit passive wake word mode. */
  onExitPassive?: () => void;
  /** Extra classes for the outer wrapper. */
  className?: string;
  /** Extra classes for the inner button element. */
  buttonClassName?: string;
  /** Optional test id forwarded to the button. */
  testId?: string;
}

/** Fixed-position tooltip rendered via portal so it can't be clipped by overflow parents. */
function ErrorTooltip({ anchor, text, onDismiss }: { anchor: HTMLElement; text: string; onDismiss: () => void }) {
  const [style, setStyle] = useState<React.CSSProperties>({ visibility: "hidden" });
  const tooltipRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const rect = anchor.getBoundingClientRect();
    const el = tooltipRef.current;
    if (!el) return;

    const tooltipRect = el.getBoundingClientRect();
    const pad = 4;

    let top: number;
    if (rect.top - tooltipRect.height - pad >= pad) {
      top = rect.top - tooltipRect.height - pad;
    } else {
      top = rect.bottom + pad;
    }

    let left = rect.left + rect.width / 2 - tooltipRect.width / 2;
    left = Math.max(pad, Math.min(left, window.innerWidth - tooltipRect.width - pad));

    setStyle({ position: "fixed", top, left, visibility: "visible" });
  }, [anchor, text]);

  return createPortal(
    <div
      ref={tooltipRef}
      className="z-[9999] flex w-52 items-start gap-1.5 rounded border border-amber-500/50 bg-slate-800 px-2 py-1 text-[10px] text-amber-300 shadow-lg"
      style={style}
      role="status"
    >
      <span className="min-w-0 flex-1 break-words">{text}</span>
      <button
        type="button"
        onClick={onDismiss}
        className="-mr-1 rounded p-0.5 text-amber-200 transition hover:bg-amber-500/15 hover:text-amber-100"
        aria-label="Dismiss voice input error"
      >
        <X className="h-3 w-3" />
      </button>
    </div>,
    document.body,
  );
}

function VoiceMicButtonInner({
  supported,
  isPreparing,
  isRecording,
  isListening = false,
  isPassive = false,
  isTranscribing,
  error,
  audioLevel = 0,
  voiceActivity,
  serverVad: serverVadProp,
  partialTranscript,
  backend,
  onStart,
  onStop,
  onCancel,
  onExitPassive,
  className: wrapperClassName,
  buttonClassName,
  testId,
}: VoiceMicButtonProps) {
  const isMicActive = isRecording || isListening;
  const [buttonEl, setButtonEl] = useState<HTMLButtonElement | null>(null);
  const [dismissedError, setDismissedError] = useState<string | null>(null);
  const pressStartRef = useRef(0);
  const pressIntentRef = useRef<"start" | "stop" | "cancel" | "none">("none");

  const transcribingAtRef = useRef(0);
  const prevTranscribingRef = useRef(false);
  if (isTranscribing && !prevTranscribingRef.current) {
    transcribingAtRef.current = Date.now();
  }
  prevTranscribingRef.current = isTranscribing;

  const handlePointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    if (isPreparing) return;
    pressStartRef.current = Date.now();
    if (isTranscribing) {
      const inGracePeriod = Date.now() - transcribingAtRef.current < TRANSCRIBING_GRACE_MS;
      pressIntentRef.current = onCancel && !inGracePeriod ? "cancel" : "none";
    } else if (isPassive) {
      pressIntentRef.current = "stop";
    } else if (isMicActive) {
      pressIntentRef.current = "stop";
    } else {
      pressIntentRef.current = "start";
      onStart({ vadEnabled: true });
    }
  }, [isPreparing, isMicActive, isPassive, isTranscribing, onStart, onCancel]);

  const handlePointerUp = useCallback(() => {
    if (isPreparing) return;
    const intent = pressIntentRef.current;
    pressIntentRef.current = "none";
    if (intent === "cancel") {
      onCancel?.();
    } else if (intent === "stop" && isPassive) {
      onExitPassive?.();
    } else if (intent === "stop") {
      onStop();
    } else if (intent === "start" && !isListening && Date.now() - pressStartRef.current >= LONG_PRESS_MS) {
      onStop();
    }
  }, [isPreparing, isPassive, isListening, onStop, onCancel, onExitPassive]);

  // Subscribe to the server VAD-state store. Prop override (used by tests
  // and by MicButton.tsx wrapper) wins over the live store snapshot.
  const serverVadFromStore = useServerVadStateStore((s) => s);
  const serverVad: ServerVadStateSnapshot | undefined = serverVadProp ?? serverVadFromStore;

  // Server-snapshot interpolation tick. Drives a RAF loop while a fresh
  // server vad-state is in play so the ring fills smoothly between server
  // ticks. The render derives the actual value each frame.
  const [, setRafTick] = useState(0);
  const serverVadFresh = serverVad
    && serverVad.receivedAt > 0
    && isRecording
    && (typeof performance !== "undefined" ? performance.now() : Date.now()) - serverVad.receivedAt < SERVER_VAD_STALE_MS;
  useEffect(() => {
    if (!serverVadFresh) return;
    let raf = 0;
    const tick = () => {
      setRafTick((n) => (n + 1) & 0x3fffffff);
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [serverVadFresh]);

  const isIdle = !isMicActive && !isPassive && !isTranscribing && !isPreparing;
  const hasError = error !== null && isIdle;
  const showErrorTooltip = hasError && error !== dismissedError;
  const liveAudioLevel = voiceActivity?.audioLevel ?? audioLevel;

  useEffect(() => {
    if (!hasError && dismissedError !== null) {
      setDismissedError(null);
    }
  }, [dismissedError, hasError]);

  if (!supported) return null;

  // Prefer the server-emitted silence clock when fresh; fall back to the
  // client VAD's autoStopProgress when stale (>250 ms) or absent. Formula
  // matches plan §7 step 12.
  const nowPerf = typeof performance !== "undefined" ? performance.now() : Date.now();
  const fresh = !!serverVad
    && serverVad.receivedAt > 0
    && (nowPerf - serverVad.receivedAt) < SERVER_VAD_STALE_MS;
  let autoStopProgress: number;
  let showAutoStopRing: boolean;
  if (fresh && serverVad!.silenceTimeoutMs > 0) {
    const interpolated = Math.min(
      serverVad!.silenceElapsedMs + (nowPerf - serverVad!.receivedAt),
      serverVad!.silenceTimeoutMs,
    );
    autoStopProgress = Math.max(0, Math.min(1, interpolated / serverVad!.silenceTimeoutMs));
    const autoStopVisible = !serverVad!.voiced
      && interpolated >= VAD_AUTO_STOP_VISUAL_GRACE_MS;
    showAutoStopRing = isRecording && autoStopVisible;
  } else {
    autoStopProgress = Math.max(0, Math.min(1, voiceActivity?.autoStopProgress ?? 0));
    showAutoStopRing = isRecording
      && voiceActivity?.phase === "silence"
      && voiceActivity.autoStopVisible
      && voiceActivity.silenceTimeoutMs > 0;
  }

  const title = isPreparing
    ? "Preparing microphone…"
    : isPassive
      ? "Passive listening — tap to exit"
      : isListening
        ? "Listening — tap to stop"
        : isRecording
          ? "Recording — tap to stop"
          : isTranscribing
            ? "Transcribing…"
            : hasError
              ? `Voice input error: ${error}`
              : backend
                ? `Tap to speak (${backend})`
                : "Tap to speak";

  return (
    <div className={cn("relative shrink-0", wrapperClassName)}>
      <button
        ref={setButtonEl}
        type="button"
        data-testid={testId ?? "voice-mic-btn"}
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        aria-pressed={isMicActive}
        className={cn(
          "relative shrink-0 overflow-hidden rounded border px-1.5 py-1 text-xs font-medium transition active:opacity-80 touch-manipulation",
          buttonClassName,
          isPreparing
            ? "border-yellow-500/50 bg-yellow-500/10 text-yellow-400"
            : isPassive
              ? "border-indigo-500/30 bg-indigo-500/5 text-indigo-400"
              : isListening
                ? "border-cyan-500 bg-cyan-500/20 text-cyan-400"
                : isRecording
                  ? "border-red-500 bg-red-500/20 text-red-400"
                  : isTranscribing
                    ? "border-blue-500 bg-blue-500/20 text-blue-400"
                    : hasError
                      ? "border-amber-500 bg-amber-500/10 text-amber-400"
                      : "border-slate-700 bg-slate-800/60 text-slate-400",
        )}
        title={title}
      >
        {isMicActive && (
          <span
            className={cn(
              "absolute inset-x-0 bottom-0 rounded-[inherit] transition-[height] duration-75",
              isListening ? "bg-cyan-500/30" : "bg-red-500/30",
            )}
            style={{ height: `${Math.round(liveAudioLevel * 100)}%` }}
          />
        )}
        {showAutoStopRing && (
          <svg
            aria-hidden="true"
            data-testid="voice-auto-stop-ring"
            className="pointer-events-none absolute left-1/2 top-1/2 z-10 aspect-square h-[calc(100%-4px)] min-h-6 max-h-8 -translate-x-1/2 -translate-y-1/2 -rotate-90 overflow-visible"
            viewBox="0 0 44 44"
          >
            <circle
              cx="22"
              cy="22"
              r="18"
              fill="none"
              stroke="currentColor"
              strokeWidth="3"
              className="text-amber-300/80"
              strokeDasharray={AUTO_STOP_RING_CIRCUMFERENCE}
              strokeDashoffset={AUTO_STOP_RING_CIRCUMFERENCE * (1 - autoStopProgress)}
              strokeLinecap="round"
            />
          </svg>
        )}
        {isPreparing ? (
          <Mic className="relative h-3.5 w-3.5 animate-pulse" />
        ) : isPassive ? (
          <Mic className="relative h-3.5 w-3.5 opacity-60" />
        ) : isListening ? (
          <Mic className="relative h-3.5 w-3.5 animate-pulse" />
        ) : isTranscribing ? (
          <Loader2 className="relative h-3.5 w-3.5 animate-spin" />
        ) : hasError ? (
          <AlertCircle className="relative h-3.5 w-3.5" />
        ) : (
          <Mic className="relative h-3.5 w-3.5" />
        )}
        <span className="sr-only">{title}</span>
      </button>
      {showErrorTooltip && buttonEl && (
        <ErrorTooltip anchor={buttonEl} text={error as string} onDismiss={() => setDismissedError(error)} />
      )}
      {isMicActive && partialTranscript && buttonEl && (
        <div className="pointer-events-none absolute bottom-full left-1/2 mb-1 max-w-[200px] -translate-x-1/2 overflow-hidden text-ellipsis whitespace-nowrap rounded border border-slate-700 bg-slate-800 px-2 py-1 text-[10px] text-slate-300 shadow-lg">
          {partialTranscript}
        </div>
      )}
    </div>
  );
}

const VoiceMicButton = memo(VoiceMicButtonInner);
export default VoiceMicButton;
