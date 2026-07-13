// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Shared types, constants, and interfaces for the voice input system.

import type { StreamTurnDiagnostic } from "@vrooli/audio-capture-browser";

/** Sentinel error value indicating Whisper transcription failed after retries. */
export const WHISPER_FAILED_SENTINEL = "__WHISPER_FAILED__";

/** Number of consecutive capability check failures before downgrading from Whisper. */
export const CAP_CHECK_FAIL_THRESHOLD = 2;

/** 48kbps balances Whisper accuracy with minimal bandwidth (~6KB/s on localhost). */
export const AUDIO_BITRATE = 48_000;

/** How often MediaRecorder sends audio chunks to the WebSocket (ms). */
export const STREAM_CHUNK_INTERVAL_MS = 250;

/** Compute final transcription timeout: max(10s, 2x duration), capped at 60s. */
export function computeFinalTimeout(recordingDurationMs: number): number {
  return Math.min(60_000, Math.max(10_000, recordingDurationMs * 2));
}

/**
 * Map a `getUserMedia` rejection to an honest, user-actionable message — and
 * always log the raw cause.
 *
 * The browser rejects `getUserMedia` with a `DOMException` whose `name` is the
 * real cause. A bare `catch` that unconditionally reports "Microphone access
 * denied" is wrong for the common NON-permission failures and actively
 * misleads: it sends the user to browser permission settings that are already
 * correct. That is exactly the "Settings mic test works, but the main mic
 * button says access denied" report — the same origin/permission is granted
 * (the test proves it), so the button's failure is really `NotReadableError`
 * (device held by another capture — passive wake-word listener, low-latency
 * pre-warm, or another tab/app) or `NotFoundError` (no input device), NOT a
 * denial. Classify by `name` so the message matches reality, and log the raw
 * `name`/`message` so the true cause is never swallowed again.
 */
export function classifyMicError(err: unknown): string {
  // Secure-context gate — the #1 "works on my computer, not my phone" cause.
  //
  // getUserMedia (on navigator.mediaDevices) is exposed ONLY in a secure context:
  // HTTPS, or a loopback origin (localhost / 127.0.0.1) which browsers treat as
  // secure. A desktop hitting http://localhost works; a phone reaching the same
  // self-hosted machine over a plain http:// LAN address (e.g. 192.168.x.x) is
  // NOT a secure context, so on iOS/Safari `navigator.mediaDevices` is undefined
  // and the call throws a plain TypeError — not a permission DOMException.
  // Reporting "access denied" there is doubly wrong: it isn't a denial, and it
  // sends the user to permission settings that can never fix it. Detect the
  // missing secure context first and point at the actual fix (use HTTPS).
  // Feature-detect via a local — this is detection, not acquisition, and the
  // audio-boundary guard reserves the getUserMedia access literal for the mic
  // ownership registry.
  const mediaDevices = typeof navigator !== "undefined" ? navigator.mediaDevices : undefined;
  const insecure =
    (typeof window !== "undefined" && window.isSecureContext === false) ||
    !mediaDevices ||
    typeof mediaDevices.getUserMedia !== "function";
  if (insecure) {
    console.warn(
      "[voice] getUserMedia unavailable: isSecureContext=%s hasMediaDevices=%s",
      typeof window !== "undefined" ? window.isSecureContext : "(no window)",
      typeof navigator !== "undefined" && !!navigator.mediaDevices,
    );
    return "Microphone needs a secure (HTTPS) connection. Open this site over HTTPS — on a phone, a plain http:// address (like a LAN IP) blocks the mic on iOS/Safari.";
  }

  const name =
    err instanceof DOMException
      ? err.name
      : typeof err === "object" && err !== null && "name" in err
        ? String((err as { name: unknown }).name)
        : "";
  console.warn(
    "[voice] getUserMedia failed: name=%s message=%s",
    name || "(unknown)",
    err instanceof Error ? err.message : String(err),
  );
  switch (name) {
    case "NotAllowedError":
    case "SecurityError":
      return "Microphone access denied";
    case "NotReadableError":
    case "AbortError":
      return "Microphone is busy or held by another app/tab — close the other user and try again";
    case "NotFoundError":
    case "OverconstrainedError":
      return "No usable microphone found — check your input device";
    default:
      return "Could not start the microphone — try again";
  }
}

export type VoiceBackend = "whisper" | "web-speech" | "none";

/** Explicit state machine replacing the old isRecording/isTranscribing boolean combo.
 *  "listening" is the persistent voice mode equivalent of "recording" — the mic
 *  stays active until the user taps it again. Passive wake-word listening is
 *  represented by `passiveListeningActive`, not a workflow state. */
export type VoiceState = "idle" | "preparing" | "recording" | "listening" | "transcribing";

/** The voice input mode — one-shot records a single utterance, persistent
 *  stays active with segment-boundary detection until manually stopped. */
export type VoiceMode = "one-shot" | "persistent";

export type VoiceActivityPhase = "idle" | "calibrating" | "waiting-for-speech" | "speech" | "silence";

export interface VoiceActivitySnapshot {
  phase: VoiceActivityPhase;
  audioLevel: number;
  rms: number;
  speechThreshold: number;
  silenceThreshold: number;
  silenceElapsedMs: number;
  silenceTimeoutMs: number;
  autoStopProgress: number;
  autoStopVisible: boolean;
}

/** Tracks a single speech segment within a persistent voice session. */
export interface VoiceSegment {
  text: string;
  isFinal: boolean;
}

/**
 * Snapshot of the last completed recording turn's audio, retained by the
 * provider until explicitly disposed. The hook retrieves this via
 * `TranscriptionProvider.getLastTurnAudio()` when a rejection occurs, so
 * the user can retry transcription with the speaker-verification filter
 * bypassed without re-recording.
 *
 */
export interface LastTurnAudio {
  blob: Blob;
  mimeType: string;
  durationMs: number;
  capturedAt: number;
}

/**
 * Why a turn surfaced a retryable banner:
 *   - "speaker-rejected": speaker verification rejected the audio. Retry
 *     re-transcribes with the verification filter bypassed.
 *   - "empty-transcript": the turn completed but no text was delivered (an
 *     empty `final`, an empty HTTP-fallback result, or a streaming session
 *     that dropped mid-turn and lost context). The audio is intact, so retry
 *     re-transcribes the full retained turn via the plain HTTP batch path —
 *     which often recovers a message the streaming path silently lost.
 */
export type VoiceRejectionCause = "speaker-rejected" | "empty-transcript";

/**
 * A retryable/explanatory notice surfaced to the UI when a turn did not
 * deliver usable text.
 *
 * `retryable` is emitted when the provider retained the turn's audio — the UI
 * can offer a one-tap retry (see `cause` for what the retry does). `explanatory`
 * is emitted when the provider cannot retain audio (e.g. `WebSpeechProvider`,
 * which does not hold the raw bytes); the UI shows the reason but hides the
 * retry action.
 *
 * Discriminated union enforces at compile time that every consumer handles
 * both kinds.
 */
export type VoiceRejection =
  | {
      kind: "retryable";
      /** What produced this banner — drives both the copy and the retry path. */
      cause: VoiceRejectionCause;
      id: string;
      blob: Blob;
      mimeType: string;
      durationMs: number;
      score: number;
      threshold: number;
      createdAt: number;
      status: "idle" | "retrying" | "failed";
      errorMessage?: string;
    }
  | {
      kind: "explanatory";
      id: string;
      reason: string;
      score: number;
      threshold: number;
      createdAt: number;
    };

export interface VoiceInputState {
  supported: boolean;
  backend: VoiceBackend;
  voiceState: VoiceState;
  error: string | null;
  /** 0-1 audio level from the microphone while recording */
  audioLevel: number;
  /** UI-safe VAD snapshot derived from the same sample as audioLevel. */
  voiceActivity: VoiceActivitySnapshot;
  /** Transient notice shown when falling back to a different backend. */
  fallbackNotice: string | null;
  /** Partial transcript from streaming transcription. */
  partialTranscript: string;
  /** Active voice mode for the current session. */
  voiceMode: VoiceMode;
  /** Accumulated segment texts during persistent mode. */
  segments: VoiceSegment[];
  /** Current command suggestion awaiting user confirmation, or null. */
  commandSuggestion: CommandSuggestion | null;
  /** Whether a wake word template is configured and detection is available. */
  wakeWordConfigured: boolean;
  /**
   * Whether the passive wake-word listener currently holds the microphone.
   * Drives the mic control's honest "passively listening" presentation — the
   * UI must never report ordinary idle/off while this is true. Released (set
   * false) by user exit, active recording takeover, or page-hidden lifecycle
   * cleanup.
   */
  passiveListeningActive: boolean;
  /**
   * True when the mic ownership registry holds a live lease that the workflow
   * should NOT be holding (UI idle/off while the OS mic is still on — the
   * iOS-PWA "stuck indicator" violation). The hook self-heals by releasing the
   * orphaned lease; this flag drives the user-facing "release microphone"
   * recovery affordance for the window where a mismatch is observed. Hardware
   * truth comes from the registry, never from `voiceState`.
   */
  staleLiveMicLease: boolean;
  /**
   * The most recent speaker-verification rejection that still has user-visible
   * state (banner open, retry available). Single slot — a new rejection
   * replaces the previous one. Cleared by `dismissRejection()`, successful
   * retry, or the retention TTL.
   */
  rejectedAudio: VoiceRejection | null;
  /** Whether speaker verification is enabled and configured for the current session. */
  speakerVerificationEnabled: boolean;
  speakerProfileConfigured: boolean;
  /** Latest metadata-only streaming record, retained after a failed turn for safe export. */
  turnDiagnostic: StreamTurnDiagnostic | null;
}

/** A voice command detected from a segment-final transcript. */
export interface CommandSuggestion {
  id: string;
  commandId: string;
  description: string;
  confidence: number;
  rawText: string;
  timestamp: number;
  /** Parsed arguments from the command (e.g., { number: 3 } for "switch tab 3"). */
  args: Record<string, unknown>;
}

export interface StartRecordingOpts {
  /** When true, VAD will auto-stop recording after silence. */
  vadEnabled?: boolean;
}

export interface TranscriptionProvider {
  /** Start recording. The provider acquires and owns its own mic stream. */
  start(): void | Promise<void>;
  stop(): void;
  dispose(): void;
  getStream(): MediaStream | null;
  /**
   * Retrieve the most recent completed turn's audio, or null if none is
   * retained. Providers that cannot produce a blob (e.g. Web Speech API)
   * always return null. The blob stays valid until `disposeLastTurn()` or
   * the next `start()` (which auto-disposes).
   */
  getLastTurnAudio(): LastTurnAudio | null;
  /** Drop the retained turn's audio; subsequent `getLastTurnAudio()` returns null. */
  disposeLastTurn(): void;
  /**
   * Arm in-flight audio drop. After calling, the provider must:
   *  - drop any subsequent encoder chunk instead of forwarding to the transport
   *  - on `stop()`, signal end-of-utterance immediately and skip tail retention
   * Used by the auto-stop path so words spoken after the server-VAD verdict
   * do not leak into the transcription. Reset on the next `start()`.
   */
  dropTail(): void;
  /**
   * Delivers the final transcript for a turn. CONTRACT: a provider must call
   * `onResult` exactly once per completed turn whose audio was not entirely
   * dropped — **including with an empty string** when transcription produced
   * no text. This is what lets the host distinguish "turn ended, nothing came
   * back" (a recoverable silent loss) from "turn still in flight". A provider
   * that returns early on empty text leaves the UI wedged in "transcribing".
   */
  onResult: ((text: string) => void) | null;
  onError: ((error: string) => void) | null;
  onPartial?: ((text: string) => void) | null;
  onStatus?: ((status: { code: string; message: string }) => void) | null;
  onDiagnostic?: ((diagnostic: StreamTurnDiagnostic) => void) | null;
  getDiagnostic?(): StreamTurnDiagnostic;
  exportDiagnostic?(): string;
}
