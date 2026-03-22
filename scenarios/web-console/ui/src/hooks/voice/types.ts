// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Shared types, constants, and interfaces for the voice input system.

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

export type VoiceBackend = "whisper" | "web-speech" | "none";

/** Explicit state machine replacing the old isRecording/isTranscribing boolean combo.
 *  "listening" is the persistent voice mode equivalent of "recording" — the mic
 *  stays active until the user taps it again. */
export type VoiceState = "idle" | "preparing" | "recording" | "listening" | "transcribing";

/** The voice input mode — one-shot records a single utterance, persistent
 *  stays active with segment-boundary detection until manually stopped. */
export type VoiceMode = "one-shot" | "persistent";

/** Tracks a single speech segment within a persistent voice session. */
export interface VoiceSegment {
  text: string;
  isFinal: boolean;
}

export interface VoiceInputState {
  supported: boolean;
  backend: VoiceBackend;
  voiceState: VoiceState;
  error: string | null;
  /** 0-1 audio level from the microphone while recording */
  audioLevel: number;
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
  start(): void | Promise<void>;
  stop(): void;
  dispose(): void;
  getStream(): MediaStream | null;
  onResult: ((text: string) => void) | null;
  onError: ((error: string) => void) | null;
  onPartial?: ((text: string) => void) | null;
}
