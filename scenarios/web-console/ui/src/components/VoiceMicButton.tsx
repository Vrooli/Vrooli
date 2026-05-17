import { memo, useRef, useLayoutEffect, useState, useCallback, useEffect } from "react";
import { createPortal } from "react-dom";
import { Mic, Loader2, AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../hooks/useVoiceInput";
import type { ServerVadStateSnapshot } from "../audio-integration";
import { VAD_AUTO_STOP_VISUAL_GRACE_MS, useServerVadStateStore } from "../audio-integration";

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
  /** Interaction-only flag: starting voice input should stop active TTS first. */
  isTtsSpeaking?: boolean;
  onStart: (opts?: StartRecordingOpts) => void;
  onStop: () => void;
  onCancel?: () => void;
  /** Exit passive wake word mode. */
  onExitPassive?: () => void;
  /** Stop active TTS before starting voice input. Does not affect presentation. */
  onTtsStop?: () => void;
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
  isTtsSpeaking = false,
  onStart,
  onStop,
  onCancel,
  onExitPassive,
  onTtsStop,
  className: wrapperClassName,
  buttonClassName,
}: VoiceMicButtonProps) {
  const { t } = useTranslation();
  /** True when the mic is actively capturing (either one-shot or persistent). */
  const isMicActive = isRecording || isListening;
  const [buttonEl, setButtonEl] = useState<HTMLButtonElement | null>(null);
  const pressStartRef = useRef(0);
  /** Tracks the intent of the current pointer interaction to avoid stale-closure races. */
  const pressIntentRef = useRef<"start" | "stop" | "cancel" | "none">("none");

  /** Timestamp (ms) when isTranscribing last became true — used for grace period. */
  const transcribingAtRef = useRef(0);
  const prevTranscribingRef = useRef(false);
  if (isTranscribing && !prevTranscribingRef.current) {
    transcribingAtRef.current = Date.now();
  }
  prevTranscribingRef.current = isTranscribing;

  const handlePointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    // Block interaction while preparing to prevent double-tap issues
    if (isPreparing) return;
    pressStartRef.current = Date.now();
    if (isTranscribing) {
      // Grace period: if we just entered transcribing (e.g. VAD auto-stopped),
      // ignore the tap so the user doesn't accidentally cancel the transcript.
      const inGracePeriod = Date.now() - transcribingAtRef.current < TRANSCRIBING_GRACE_MS;
      pressIntentRef.current = onCancel && !inGracePeriod ? "cancel" : "none";
    } else if (isPassive) {
      pressIntentRef.current = "stop"; // Will call onExitPassive
    } else if (isMicActive) {
      pressIntentRef.current = "stop";
    } else {
      // Stop TTS if it's playing, then start recording
      if (isTtsSpeaking) onTtsStop?.();
      pressIntentRef.current = "start";
      onStart({ vadEnabled: true });
    }
  }, [isPreparing, isMicActive, isPassive, isTranscribing, isTtsSpeaking, onStart, onCancel, onTtsStop]);

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
      // Long press release -- push-to-talk: stop recording (one-shot only)
      onStop();
    }
    // Short press on "start" -- tap-to-toggle: keep recording
  }, [isPreparing, isPassive, isListening, onStop, onCancel, onExitPassive]);

  // Subscribe to the server VAD-state store. Prop override (used by tests)
  // wins over the live store snapshot.
  const serverVadFromStore = useServerVadStateStore((s) => s);
  const serverVad: ServerVadStateSnapshot | undefined = serverVadProp ?? serverVadFromStore;

  // Server-snapshot interpolation tick. When a fresh server vad-state is in
  // play (mic is recording, snapshot age <250 ms), drive a RAF loop so the
  // ring fills smoothly between server ticks. The render derives the actual
  // value each frame — we just trigger re-renders here.
  const [, setRafTick] = useState(0);
  const serverVadFresh = serverVad
    && serverVad.receivedAt > 0
    && isRecording
    && (typeof performance !== "undefined" ? performance.now() : Date.now()) - serverVad.receivedAt < 250;
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

  if (!supported) return null;

  const isIdle = !isMicActive && !isPassive && !isTranscribing && !isPreparing;
  const hasError = error !== null && isIdle;
  const liveAudioLevel = voiceActivity?.audioLevel ?? audioLevel;

  // Prefer the server-emitted silence clock when fresh; fall back to the
  // client VAD's autoStopProgress when stale (>250 ms since last tick) or
  // when no server tick has arrived this stream. Formula matches plan §7
  // step 12 with the VAD_AUTO_STOP_VISUAL_GRACE_MS gate applied to the
  // server-derived elapsed too, so a 1-frame silence blip doesn't flash.
  const nowPerf = typeof performance !== "undefined" ? performance.now() : Date.now();
  const fresh = !!serverVad
    && serverVad.receivedAt > 0
    && (nowPerf - serverVad.receivedAt) < 250;
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

  return (
    <div className={cn("relative shrink-0", wrapperClassName)}>
      <button
        ref={setButtonEl}
        data-testid="voice-mic-btn"
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        className={cn(
          "relative shrink-0 rounded border px-1.5 py-1 text-xs font-medium transition active:bg-wc-accent-active touch-manipulation overflow-hidden",
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
                    : "border-wc-default bg-wc-surface-input text-wc-text-secondary",
        )}
        title={
          isPreparing
            ? t(strings.voiceMicButton.preparing)
            : isPassive
              ? t(strings.voiceMicButton.passiveListening)
              : isListening
                ? t(strings.voiceMicButton.listening)
              : isRecording
                ? t(strings.voiceMicButton.recording)
                : isTranscribing
                  ? t(strings.voiceMicButton.transcribing)
                  : hasError
                    ? t(strings.voiceMicButton.error, { error })
                    : backend
                      ? t(strings.voiceMicButton.tapToSpeakWithBackend, {
                          backend: backend === "whisper"
                            ? t(strings.voiceMicButton.backendWhisper)
                            : t(strings.voiceMicButton.backendBrowser),
                        })
                      : t(strings.voiceMicButton.tapToSpeak)
        }
      >
        {/* Audio level fill -- rises from bottom. Cyan for listening, red for recording. */}
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
          <Mic className="h-3.5 w-3.5 animate-pulse relative" />
        ) : isPassive ? (
          <Mic className="h-3.5 w-3.5 animate-[breathe_3s_ease-in-out_infinite] relative opacity-60" />
        ) : isListening ? (
          <Mic className="h-3.5 w-3.5 animate-pulse relative" />
        ) : isTranscribing ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin relative" />
        ) : hasError ? (
          <AlertCircle className="h-3.5 w-3.5 relative" />
        ) : (
          <Mic className="h-3.5 w-3.5 relative" />
        )}
      </button>
      {hasError && buttonEl && (
        <ErrorTooltip anchor={buttonEl} text={error as string} />
      )}
      {isMicActive && partialTranscript && buttonEl && (
        <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-1 max-w-[200px] rounded border border-wc-default bg-wc-surface-raised px-2 py-1 text-[10px] text-wc-text-secondary shadow-lg pointer-events-none whitespace-nowrap overflow-hidden text-ellipsis">
          {partialTranscript}
        </div>
      )}
    </div>
  );
}

/**
 * Memoized so it doesn't re-render on every MobileToolbar textarea keystroke —
 * none of its props depend on the typed input. Without this, typing in the
 * mobile input felt laggy because each keystroke re-walked this subtree
 * (including the conditional ErrorTooltip portal).
 */
const VoiceMicButton = memo(VoiceMicButtonInner);
export default VoiceMicButton;
