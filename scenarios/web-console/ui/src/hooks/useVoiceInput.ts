// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Voice Input Hook — Three-Provider Architecture
// ================================================
//
// This module implements voice-to-text transcription with automatic fallback:
//
// 1. VoiceStreamProvider (preferred) — WebSocket streaming to the Go backend,
//    which runs Whisper for server-side partial and final transcription.
//    Audio chunks are sent every STREAM_CHUNK_INTERVAL_MS (250ms); the backend
//    flushes to Whisper every 500ms (voiceStreamFlushInterval), meaning each
//    flush typically processes ~2 chunks.
//
// 2. WhisperProvider — HTTP batch transcription. Used when the backend supports
//    Whisper but not the streaming WebSocket endpoint. Collects all audio and
//    sends a single POST on stop.
//
// 3. WebSpeechProvider — Browser-native Web Speech API. Final fallback when
//    Whisper is entirely unavailable. Provides interim results but quality and
//    availability vary by browser.
//
// Provider selection happens on mount (via capability check) and can change
// at runtime if Whisper becomes unavailable (automatic downgrade after
// CAP_CHECK_FAIL_THRESHOLD consecutive failures).
//
// The hook also provides:
// - Audio level monitoring via Web Audio API (bandpass filter 80Hz–8kHz)
// - Voice Activity Detection (VAD) with adaptive noise floor
// - Dead-mic detection (warns after 2s of zero audio)

// Web Speech API type declarations (not included in all TS libs)
interface SpeechRecognitionResultItem {
  transcript: string;
  confidence: number;
}

interface SpeechRecognitionResult {
  readonly length: number;
  item(index: number): SpeechRecognitionResultItem;
  [index: number]: SpeechRecognitionResultItem;
  isFinal: boolean;
}

interface SpeechRecognitionResultList {
  readonly length: number;
  item(index: number): SpeechRecognitionResult;
  [index: number]: SpeechRecognitionResult;
}

interface _SpeechRecognitionEventInit extends EventInit {
  results: SpeechRecognitionResultList;
}

interface SpeechRecognitionEvent extends Event {
  readonly results: SpeechRecognitionResultList;
}

interface _SpeechRecognitionErrorEventInit extends EventInit {
  error: string;
  message?: string;
}

interface SpeechRecognitionErrorEvent extends Event {
  readonly error: string;
  readonly message: string;
}

interface SpeechRecognitionInstance extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}

interface SpeechRecognitionConstructor {
  new (): SpeechRecognitionInstance;
}

declare global {
  interface Window {
    SpeechRecognition?: SpeechRecognitionConstructor;
    webkitSpeechRecognition?: SpeechRecognitionConstructor;
  }
}

import { useState, useEffect, useRef, useCallback } from "react";
import { fetchCapabilities, transcribeAudioWithRetry, buildVoiceStreamWsUrl } from "../lib/api";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

const WHISPER_FAILED_SENTINEL = "__WHISPER_FAILED__";

/** Number of consecutive capability check failures before downgrading from Whisper. */
const CAP_CHECK_FAIL_THRESHOLD = 2;

/** 48kbps balances Whisper accuracy with minimal bandwidth (~6KB/s on localhost). */
export const AUDIO_BITRATE = 48_000;

/** How often MediaRecorder sends audio chunks to the WebSocket (ms). */
export const STREAM_CHUNK_INTERVAL_MS = 250;

/** Compute final transcription timeout: max(10s, 2× duration), capped at 60s. */
export function computeFinalTimeout(recordingDurationMs: number): number {
  return Math.min(60_000, Math.max(10_000, recordingDurationMs * 2));
}

/**
 * Build a bandpass filter chain targeting the speech band (80Hz–8kHz).
 * Returns an AnalyserNode (for level monitoring) and a filtered MediaStream
 * suitable for MediaRecorder input.
 */
export function createAudioFilterChain(
  ctx: AudioContext,
  source: MediaStreamAudioSourceNode,
): { analyser: AnalyserNode; filteredStream: MediaStream } {
  const highpass = ctx.createBiquadFilter();
  highpass.type = "highpass";
  highpass.frequency.value = 80;
  highpass.Q.value = 0.707; // Butterworth

  const lowpass = ctx.createBiquadFilter();
  lowpass.type = "lowpass";
  lowpass.frequency.value = 8000;
  lowpass.Q.value = 0.707;

  const destination = ctx.createMediaStreamDestination();
  const analyser = ctx.createAnalyser();
  analyser.fftSize = 128;

  // Chain: source → highpass → lowpass → destination
  //                                   └→ analyser (for level monitoring)
  source.connect(highpass);
  highpass.connect(lowpass);
  lowpass.connect(destination);
  lowpass.connect(analyser);

  return { analyser, filteredStream: destination.stream };
}

export type VoiceBackend = "whisper" | "web-speech" | "none";

export interface VoiceInputState {
  supported: boolean;
  backend: VoiceBackend;
  isRecording: boolean;
  isTranscribing: boolean;
  error: string | null;
  /** 0–1 audio level from the microphone while recording */
  audioLevel: number;
  /** Transient notice shown when falling back to a different backend. */
  fallbackNotice: string | null;
  /** Partial transcript from streaming transcription. */
  partialTranscript: string;
}

export interface StartRecordingOpts {
  /** When true, VAD will auto-stop recording after silence. */
  vadEnabled?: boolean;
}

interface TranscriptionProvider {
  start(): void | Promise<void>;
  stop(): void;
  dispose(): void;
  getStream(): MediaStream | null;
  onResult: ((text: string) => void) | null;
  onError: ((error: string) => void) | null;
  onPartial?: ((text: string) => void) | null;
}

class WhisperProvider implements TranscriptionProvider {
  private mediaRecorder: MediaRecorder | null = null;
  private chunks: Blob[] = [];
  private stream: MediaStream | null = null;
  language = "en";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;

  getStream(): MediaStream | null {
    return this.stream;
  }

  async start(): Promise<void> {
    try {
      this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      this.onError?.("Microphone access denied");
      return;
    }
    this.chunks = [];
    this.mediaRecorder = new MediaRecorder(this.stream, {
      mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
        ? "audio/webm;codecs=opus"
        : "audio/webm",
      audioBitsPerSecond: AUDIO_BITRATE,
    });
    this.mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) this.chunks.push(e.data);
    };
    this.mediaRecorder.onstop = async () => {
      this.stream?.getTracks().forEach((t) => t.stop());
      this.stream = null;

      const blob = new Blob(this.chunks, { type: "audio/webm" });
      this.chunks = [];
      if (blob.size === 0) return;
      try {
        const text = await transcribeAudioWithRetry(blob, 2, this.language);
        if (text.trim()) this.onResult?.(text.trim());
      } catch {
        this.onError?.(WHISPER_FAILED_SENTINEL);
      }
    };
    this.mediaRecorder.start();
  }

  stop(): void {
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
      // Mic release happens in onstop after final data is flushed.
    } else {
      this.stream?.getTracks().forEach((t) => t.stop());
      this.stream = null;
    }
  }

  dispose(): void {
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
  }
}

class WebSpeechProvider implements TranscriptionProvider {
  private recognition: SpeechRecognitionInstance | null = null;
  private micStream: MediaStream | null = null;
  private stopped = false;
  /** Tracks how many results have already been dispatched via onResult. */
  private processedResultCount = 0;
  lang = "en-US";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;

  getStream(): MediaStream | null {
    return this.micStream;
  }

  async start(): Promise<void> {
    const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
    if (!Ctor) {
      this.onError?.("Web Speech API not available");
      return;
    }
    // Acquire mic stream for audio level monitoring
    try {
      this.micStream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      this.onError?.("Microphone access denied");
      return;
    }
    this.stopped = false;
    this.processedResultCount = 0;
    this.recognition = new Ctor();
    this.recognition.continuous = true;
    this.recognition.interimResults = true;
    this.recognition.lang = this.lang;
    this.recognition.onresult = (event: SpeechRecognitionEvent) => {
      // event.results is cumulative — it contains ALL results from the start
      // of the session. Only process results we haven't dispatched yet.
      let newFinalText = "";
      let interimText = "";
      for (let i = this.processedResultCount; i < event.results.length; i++) {
        const result = event.results[i];
        if (result?.isFinal) {
          newFinalText += result[0]?.transcript ?? "";
          // Mark all results up to and including this one as processed.
          // We can't skip indices because the API guarantees results
          // finalize in order.
          this.processedResultCount = i + 1;
        } else {
          interimText += result?.[0]?.transcript ?? "";
        }
      }
      if (interimText) this.onPartial?.(interimText);
      if (newFinalText.trim()) this.onResult?.(newFinalText.trim());
    };
    this.recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      if (event.error !== "aborted") {
        this.onError?.(`Speech recognition error: ${event.error}`);
      }
    };
    this.recognition.onend = () => {
      // Browser may end continuous recognition spontaneously; restart unless
      // intentionally stopped. There is a brief gap (~100-500ms) during which
      // no audio is captured — this is an inherent browser limitation.
      // processedResultCount persists across restarts (it's an instance field,
      // not tied to the recognition instance), so previously finalized results
      // are correctly skipped after restart.
      if (!this.stopped && this.recognition) {
        try { this.recognition.start(); } catch { /* already started or disposed */ }
      }
    };
    this.recognition.start();
  }

  stop(): void {
    this.stopped = true;
    this.recognition?.stop();
    this.recognition = null;
    // Release mic so the browser indicator turns off
    this.micStream?.getTracks().forEach((t) => t.stop());
    this.micStream = null;
  }

  dispose(): void {
    this.stop();
  }
}

class VoiceStreamProvider implements TranscriptionProvider {
  private ws: WebSocket | null = null;
  private mediaRecorder: MediaRecorder | null = null;
  private stream: MediaStream | null = null;
  private finalReceived = false;
  private finalTimeout: ReturnType<typeof setTimeout> | null = null;
  private wsUrl = "";
  private reconnectAttempt = 0;
  private intentionallyStopped = false;
  private recordingStartTime = 0;
  /** Timestamp when stop() was called — used to measure stop→final latency. */
  private stopTime = 0;
  private pendingChunks: ArrayBuffer[] = [];
  /** All audio chunks collected for HTTP fallback if streaming fails entirely. */
  private allChunks: Blob[] = [];
  /** Running count of audio chunks sent via WebSocket. */
  private chunkCount = 0;
  /** Running total of bytes sent via WebSocket. */
  private totalBytesSent = 0;
  private static readonly MAX_RECONNECTS = 2;
  private static readonly RECONNECT_DELAYS = [1_000, 3_000];
  private firstPartialLogged = false;
  language = "en";
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;

  getStream(): MediaStream | null {
    return this.stream;
  }

  private setupWsHandlers(ws: WebSocket): void {
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as { type: string; text?: string };
        if (msg.type === "partial" && msg.text) {
          if (!this.firstPartialLogged) {
            const latency = Date.now() - this.recordingStartTime;
            console.info("[voice] First partial received, latency=%dms, text=%s",
              latency, msg.text.length > 60 ? msg.text.slice(0, 60) + "…" : msg.text);
            this.firstPartialLogged = true;
          }
          this.onPartial?.(msg.text);
        } else if (msg.type === "final") {
          this.finalReceived = true;
          if (this.finalTimeout) {
            clearTimeout(this.finalTimeout);
            this.finalTimeout = null;
          }
          const text = msg.text?.trim() ?? "";
          const stopToFinal = this.stopTime > 0 ? Date.now() - this.stopTime : 0;
          const totalDuration = Date.now() - this.recordingStartTime;
          console.info("[voice] Final received: %d chars, stopToFinal=%dms, total=%dms, chunks=%d, bytes=%d",
            text.length, stopToFinal, totalDuration, this.chunkCount, this.totalBytesSent);
          if (text) {
            this.onResult?.(text);
          }
        } else if (msg.type === "error") {
          this.onError?.(msg.text ?? "Streaming transcription failed");
        }
      } catch {
        // Ignore malformed messages
      }
    };

    ws.onerror = () => {
      console.warn("[voice] WebSocket error — will attempt HTTP fallback on close");
      // Don't emit error here; onclose fires after onerror and will handle fallback.
    };

    ws.onclose = () => {
      console.info("[voice] WebSocket closed, finalReceived:", this.finalReceived);
      if (this.finalReceived || this.intentionallyStopped) return;

      // Attempt reconnect with exponential backoff if recording is still active.
      if (this.reconnectAttempt < VoiceStreamProvider.MAX_RECONNECTS
          && this.mediaRecorder?.state === "recording") {
        const delay = VoiceStreamProvider.RECONNECT_DELAYS[this.reconnectAttempt] ?? 3_000;
        this.reconnectAttempt++;
        console.warn("[voice] WebSocket reconnect attempt %d/%d, delay=%dms, pendingChunks=%d",
          this.reconnectAttempt, VoiceStreamProvider.MAX_RECONNECTS, delay, this.pendingChunks.length);

        setTimeout(() => {
          const newWs = new WebSocket(this.wsUrl);
          const connTimeout = setTimeout(() => {
            newWs.close();
            // onclose will fire again — next attempt or final failure
          }, delay + 2_000);

          newWs.onopen = () => {
            clearTimeout(connTimeout);
            this.ws = newWs;
            this.setupWsHandlers(newWs);
            // Flush chunks buffered during reconnection
            for (const chunk of this.pendingChunks) {
              if (newWs.readyState === WebSocket.OPEN) newWs.send(chunk);
            }
            this.pendingChunks = [];
          };
          newWs.onerror = () => {
            clearTimeout(connTimeout);
          };
        }, delay);
        return;
      }

      // All reconnects exhausted — fall back to HTTP transcription
      this.attemptHttpFallback();
    };
  }

  /**
   * Fall back to HTTP transcription using all collected audio chunks.
   * Called when WebSocket streaming fails and all reconnects are exhausted.
   */
  private attemptHttpFallback(): void {
    if (this.allChunks.length === 0) {
      this.onError?.(WHISPER_FAILED_SENTINEL);
      return;
    }
    console.warn("[voice] Streaming failed — falling back to HTTP transcription");
    const blob = new Blob(this.allChunks, { type: "audio/webm" });
    this.allChunks = [];
    transcribeAudioWithRetry(blob, 2, this.language)
      .then((text) => {
        if (text.trim()) {
          this.finalReceived = true;
          this.onResult?.(text.trim());
        }
      })
      .catch(() => {
        this.onError?.(WHISPER_FAILED_SENTINEL);
      });
  }

  async start(): Promise<void> {
    try {
      this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      this.onError?.("Microphone access denied");
      return;
    }

    this.wsUrl = buildVoiceStreamWsUrl(this.language);
    this.ws = new WebSocket(this.wsUrl);
    this.finalReceived = false;
    this.firstPartialLogged = false;
    this.reconnectAttempt = 0;
    this.intentionallyStopped = false;
    this.recordingStartTime = Date.now();
    this.stopTime = 0;
    this.pendingChunks = [];
    this.allChunks = [];
    this.chunkCount = 0;
    this.totalBytesSent = 0;

    this.ws.onopen = () => {
      console.info("[voice] WebSocket connected");
      if (!this.stream) return;
      this.mediaRecorder = new MediaRecorder(this.stream, {
        mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
          ? "audio/webm;codecs=opus"
          : "audio/webm",
        audioBitsPerSecond: AUDIO_BITRATE,
      });
      this.mediaRecorder.ondataavailable = (e) => {
        if (e.data.size > 0) {
          this.chunkCount++;
          this.totalBytesSent += e.data.size;
          // Keep a copy for HTTP fallback in case streaming fails entirely.
          this.allChunks.push(e.data);
          e.data.arrayBuffer().then((buf) => {
            if (this.ws?.readyState === WebSocket.OPEN) {
              this.ws.send(buf);
            } else {
              // Buffer during reconnection so no audio is lost
              this.pendingChunks.push(buf);
            }
          });
        }
      };
      this.mediaRecorder.start(STREAM_CHUNK_INTERVAL_MS);
    };

    this.setupWsHandlers(this.ws);
  }

  stop(): void {
    this.intentionallyStopped = true;
    this.stopTime = Date.now();
    const recordingDuration = this.stopTime - this.recordingStartTime;
    console.info("[voice] Stop: recordingDuration=%dms, chunks=%d, bytes=%d",
      recordingDuration, this.chunkCount, this.totalBytesSent);

    if (this.mediaRecorder?.state === "recording") {
      // Defer mic release and "done" signal until after final data is flushed.
      this.mediaRecorder.onstop = () => {
        if (this.ws?.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify({ type: "done" }));
        }
        this.stream?.getTracks().forEach((t) => t.stop());
        this.stream = null;
      };
      this.mediaRecorder.stop();
    } else {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "done" }));
      }
      this.stream?.getTracks().forEach((t) => t.stop());
      this.stream = null;
    }

    if (!this.finalReceived) {
      const elapsed = Date.now() - this.recordingStartTime;
      const timeout = computeFinalTimeout(elapsed);
      this.finalTimeout = setTimeout(() => {
        if (!this.finalReceived) {
          this.ws?.close();
          this.onError?.("Streaming transcription timed out");
        }
      }, timeout);
    }
  }

  dispose(): void {
    this.intentionallyStopped = true;
    if (this.finalTimeout) {
      clearTimeout(this.finalTimeout);
      this.finalTimeout = null;
    }
    if (this.mediaRecorder?.state === "recording") {
      this.mediaRecorder.stop();
    }
    this.ws?.close();
    this.ws = null;
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
    this.pendingChunks = [];
  }
}

// --- VAD (Voice Activity Detection) ---

type VadState = "idle" | "calibrating" | "waitingForSpeech" | "speechDetected" | "watchingSilence";

const VAD_CALIBRATION_MS = 500;
/** Default silence timeout (ms). Configurable via workspace store `vadSilenceTimeoutMs`. */
export const VAD_DEFAULT_SILENCE_TIMEOUT_MS = 2000;
const VAD_NO_SPEECH_TIMEOUT_MS = 15_000;
const VAD_MIN_SILENCE_THRESHOLD = 0.02;
const VAD_MIN_SPEECH_THRESHOLD = 0.06;
const VAD_SLIDING_WINDOW_SIZE = 30;     // ~2s at 15Hz
const VAD_NOISE_FLOOR_DECAY_RATE = 0.5; // max floor decrease per second

export interface VadRefs {
  state: VadState;
  recordingStart: number;
  silenceStart: number;
  noiseFloorSamples: number[];
  speechThreshold: number;
  silenceThreshold: number;
  slidingWindow: number[];
  slidingWindowIdx: number;
  lastFloorUpdateTime: number;
}

export function createVadRefs(): VadRefs {
  return {
    state: "idle",
    recordingStart: 0,
    silenceStart: 0,
    noiseFloorSamples: [],
    speechThreshold: VAD_MIN_SPEECH_THRESHOLD,
    silenceThreshold: VAD_MIN_SILENCE_THRESHOLD,
    slidingWindow: [],
    slidingWindowIdx: 0,
    lastFloorUpdateTime: 0,
  };
}

/**
 * Compute an updated noise floor from a sliding window of RMS samples.
 * Uses the 25th percentile to ignore speech spikes. Rising noise is adopted
 * immediately; falling noise decays at most `decayRate` per second (hysteresis).
 */
export function computeSlidingNoiseFloor(
  samples: number[],
  currentFloor: number,
  elapsedSec: number,
  decayRate: number,
): number {
  if (samples.length === 0) return currentFloor;
  const sorted = [...samples].sort((a, b) => a - b);
  const pctIdx = Math.floor(sorted.length * 0.25);
  const percentile = sorted[pctIdx] ?? currentFloor;

  if (percentile >= currentFloor) {
    // Noise rose — adopt immediately
    return percentile;
  }
  // Noise fell — decay gradually
  const maxDrop = decayRate * elapsedSec;
  return Math.max(percentile, currentFloor - maxDrop);
}

/**
 * Run one VAD tick. Returns "stop" if recording should auto-stop,
 * "no-speech" if the no-speech timeout expired, or null to continue.
 *
 * Pure function — all inputs are explicit parameters with no external dependencies.
 */
export function vadTick(vad: VadRefs, rms: number, now: number, silenceTimeoutMs: number = VAD_DEFAULT_SILENCE_TIMEOUT_MS): "stop" | "no-speech" | null {
  if (vad.state === "idle") return null;

  if (vad.state === "calibrating") {
    vad.noiseFloorSamples.push(rms);
    if (now - vad.recordingStart >= VAD_CALIBRATION_MS) {
      // Compute adaptive thresholds from noise floor
      const avg = vad.noiseFloorSamples.reduce((a, b) => a + b, 0) / (vad.noiseFloorSamples.length || 1);
      vad.silenceThreshold = Math.max(VAD_MIN_SILENCE_THRESHOLD, avg * 1.5);
      vad.speechThreshold = Math.max(VAD_MIN_SPEECH_THRESHOLD, avg * 3);
      vad.state = "waitingForSpeech";
    }
    return null;
  }

  // --- Sliding window noise floor update (active in all post-calibration states) ---
  // Push RMS into circular buffer
  if (vad.slidingWindow.length < VAD_SLIDING_WINDOW_SIZE) {
    vad.slidingWindow.push(rms);
  } else {
    vad.slidingWindow[vad.slidingWindowIdx % VAD_SLIDING_WINDOW_SIZE] = rms;
  }
  vad.slidingWindowIdx++;

  // Recompute noise floor when buffer is full
  if (vad.slidingWindow.length >= VAD_SLIDING_WINDOW_SIZE) {
    const elapsed = vad.lastFloorUpdateTime > 0
      ? (now - vad.lastFloorUpdateTime) / 1000
      : 0;
    const currentFloor = vad.silenceThreshold / 1.5; // reverse from threshold
    const newFloor = computeSlidingNoiseFloor(
      vad.slidingWindow,
      currentFloor,
      elapsed,
      VAD_NOISE_FLOOR_DECAY_RATE,
    );
    vad.silenceThreshold = Math.max(VAD_MIN_SILENCE_THRESHOLD, newFloor * 1.5);
    vad.speechThreshold = Math.max(VAD_MIN_SPEECH_THRESHOLD, newFloor * 3);
    vad.lastFloorUpdateTime = now;
  }

  if (vad.state === "waitingForSpeech") {
    if (rms > vad.speechThreshold) {
      vad.state = "speechDetected";
      return null;
    }
    if (now - vad.recordingStart > VAD_NO_SPEECH_TIMEOUT_MS) {
      return "no-speech";
    }
    return null;
  }

  if (vad.state === "speechDetected") {
    if (rms < vad.silenceThreshold) {
      vad.state = "watchingSilence";
      vad.silenceStart = now;
    }
    return null;
  }

  if (vad.state === "watchingSilence") {
    if (rms > vad.speechThreshold) {
      vad.state = "speechDetected";
      return null;
    }
    if (now - vad.silenceStart >= silenceTimeoutMs) {
      return "stop";
    }
    return null;
  }

  return null;
}

// --- Hook ---

export function useVoiceInput(onTranscript: (text: string) => void) {
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  const voiceLanguage = useWorkspaceStore((s) => s.voiceLanguage);
  const vadSilenceTimeoutMs = useWorkspaceStore((s) => s.vadSilenceTimeoutMs);
  const [state, setState] = useState<VoiceInputState>({
    supported: false,
    backend: "none",
    isRecording: false,
    isTranscribing: false,
    error: null,
    audioLevel: 0,
    fallbackNotice: null,
    partialTranscript: "",
  });

  const providerRef = useRef<TranscriptionProvider | null>(null);
  const onTranscriptRef = useRef(onTranscript);
  onTranscriptRef.current = onTranscript;
  const backendRef = useRef<VoiceBackend>(state.backend);
  backendRef.current = state.backend;
  const streamingAvailableRef = useRef(false);

  // Audio level monitoring refs — AudioContext is reused across recording
  // sessions to avoid hitting the browser's 6-8 context limit.
  const audioCtxRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const rafRef = useRef<number>(0);
  const lastTickRef = useRef(0);
  const audioLevelRef = useRef(0);
  const levelSyncRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // VAD refs
  const vadRef = useRef<VadRefs>(createVadRefs());
  const vadActiveRef = useRef(false);
  // Ref to allow stopRecording to be called from inside the tick loop
  const stopRecordingRef = useRef<(() => void) | null>(null);
  // Track VAD state for diagnostic logging (only log on transitions)
  const prevVadStateRef = useRef<string>("idle");
  // Keep silence timeout in a ref so the rAF tick loop always reads the latest value.
  const vadSilenceTimeoutRef = useRef(vadSilenceTimeoutMs);
  vadSilenceTimeoutRef.current = vadSilenceTimeoutMs;
  // Timer to warn if no audio is detected after recording starts
  const noAudioTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  /** Guards against concurrent startRecording calls during async startup. */
  const startingRef = useRef(false);
  /** When true, stopRecording was called during startup — recording should abort after start completes. */
  const stopRequestedRef = useRef(false);

  const startLevelMonitor = useCallback((stream: MediaStream) => {
    try {
      // Reuse existing AudioContext to avoid hitting the browser's limit.
      let ctx = audioCtxRef.current;
      if (!ctx) {
        ctx = new AudioContext();
        audioCtxRef.current = ctx;
      }
      if (ctx.state === "suspended") {
        ctx.resume().catch(() => {});
      }

      const source = ctx.createMediaStreamSource(stream);
      const { analyser } = createAudioFilterChain(ctx, source);
      analyserRef.current = analyser;

      const data = new Uint8Array(analyser.frequencyBinCount);
      lastTickRef.current = 0;

      // Sync audioLevel ref → React state at 10 Hz (100ms).
      levelSyncRef.current = setInterval(() => {
        const l = audioLevelRef.current;
        setState((s) => (Math.abs(s.audioLevel - l) < 0.01 ? s : { ...s, audioLevel: l }));
      }, 100);

      const tick = () => {
        // Throttle to ~15 Hz — audio analysis doesn't need 60 fps.
        const now = performance.now();
        if (now - lastTickRef.current < 66) {
          rafRef.current = requestAnimationFrame(tick);
          return;
        }
        lastTickRef.current = now;

        analyser.getByteTimeDomainData(data);
        // Compute RMS level normalized to 0–1
        let sum = 0;
        for (let i = 0; i < data.length; i++) {
          const v = ((data[i] ?? 128) - 128) / 128;
          sum += v * v;
        }
        const rms = Math.sqrt(sum / data.length);
        // Scale up for visibility (RMS of speech is typically 0.05–0.3)
        audioLevelRef.current = Math.min(1, rms * 4);

        // VAD check
        if (vadActiveRef.current) {
          const prevState = vadRef.current.state;
          const result = vadTick(vadRef.current, rms, Date.now(), vadSilenceTimeoutRef.current);
          if (vadRef.current.state !== prevState) {
            console.debug("[voice] VAD:", prevState, "\u2192", vadRef.current.state,
              "rms=" + rms.toFixed(3), "speechThresh=" + vadRef.current.speechThreshold.toFixed(3));
            prevVadStateRef.current = vadRef.current.state;
          }
          if (result === "stop") {
            stopRecordingRef.current?.();
          } else if (result === "no-speech") {
            vadActiveRef.current = false;
            vadRef.current.state = "idle";
            stopRecordingRef.current?.();
            setState((s) => ({ ...s, error: "No speech detected" }));
          }
        }

        rafRef.current = requestAnimationFrame(tick);
      };
      rafRef.current = requestAnimationFrame(tick);
    } catch {
      // AudioContext not available — no level monitoring
    }
  }, []);

  const stopLevelMonitor = useCallback(() => {
    cancelAnimationFrame(rafRef.current);
    rafRef.current = 0;
    lastTickRef.current = 0;
    if (levelSyncRef.current) {
      clearInterval(levelSyncRef.current);
      levelSyncRef.current = null;
    }
    audioLevelRef.current = 0;
    // Keep AudioContext alive for reuse; only disconnect the analyser.
    analyserRef.current = null;
    setState((s) => (s.audioLevel === 0 ? s : { ...s, audioLevel: 0 }));
  }, []);

  // Pre-recording capability check debounce — declared before mount effect
  // so the effect can seed the timestamp and prevent a redundant blocking check.
  const lastCapCheckRef = useRef(0);

  // Optimistic mount: show the mic button immediately and check Whisper in
  // the background. WhisperProvider records audio locally (no backend needed
  // during recording), so the user can start speaking before the check resolves.
  useEffect(() => {
    if (!voiceEnabled) {
      setState((s) => ({ ...s, supported: false, backend: "none" }));
      return;
    }

    // Show button immediately — optimistic default assumes Whisper.
    setState((s) => ({ ...s, supported: true, backend: "whisper" }));
    // Seed the debounce timer so startRecording doesn't re-run the check.
    lastCapCheckRef.current = Date.now();

    let cancelled = false;
    (async () => {
      try {
        const caps = await fetchCapabilities();
        if (cancelled) return;
        const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
        if (whisper?.status === "available") {
          streamingAvailableRef.current = whisper.features?.includes("voice-streaming") ?? false;
          console.info("[voice] Backend confirmed: whisper, streaming=%s", streamingAvailableRef.current);
          // Already optimistically set to "whisper" — no state change needed.
          return;
        }
      } catch (err) {
        console.warn("[voice] Capabilities fetch failed on mount:", err);
      }

      if (cancelled) return;

      // Whisper unavailable — downgrade (but don't disrupt an in-progress recording)
      const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
      if (Ctor) {
        console.info("[voice] Backend: web-speech (Whisper unavailable)");
        setState((s) => s.isRecording ? s : { ...s, backend: "web-speech" });
        return;
      }

      console.info("[voice] Backend: none (no voice backend available)");
      setState((s) => s.isRecording
        ? { ...s, error: "Voice not available" }
        : { ...s, supported: false, backend: "none" });
    })();

    return () => {
      cancelled = true;
      providerRef.current?.dispose();
      providerRef.current = null;
      audioCtxRef.current?.close().catch(() => {});
      audioCtxRef.current = null;
      if (noAudioTimerRef.current) {
        clearTimeout(noAudioTimerRef.current);
        noAudioTimerRef.current = null;
      }
      startingRef.current = false;
    };
  }, [voiceEnabled]);
  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  /** Consecutive failed capability checks. Reset on success. */
  const capCheckFailCountRef = useRef(0);

  const startRecording = useCallback(async (opts?: StartRecordingOpts) => {
    if (state.isRecording || startingRef.current) return;
    startingRef.current = true;
    stopRequestedRef.current = false;

    // Pre-recording capability check (debounced to every 10s)
    const isWhisperOrFallback = state.backend === "whisper" || state.backend === "web-speech";
    if (isWhisperOrFallback && Date.now() - lastCapCheckRef.current > 10_000) {
      lastCapCheckRef.current = Date.now();
      try {
        const caps = await fetchCapabilities();
        const whisper = caps.capabilities.find((c) => c.id === "whisper-stt");
        if (whisper?.status !== "available") {
          capCheckFailCountRef.current++;
          console.warn(`[voice] Whisper unavailable (attempt ${capCheckFailCountRef.current}/${CAP_CHECK_FAIL_THRESHOLD})`);
          // Only downgrade after consecutive failures exceed threshold
          if (capCheckFailCountRef.current >= CAP_CHECK_FAIL_THRESHOLD && state.backend === "whisper") {
            const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
            if (Ctor) {
              providerRef.current?.dispose();
              providerRef.current = null;
              setState((s) => ({
                ...s,
                backend: "web-speech",
                fallbackNotice: "Whisper unavailable \u2014 using browser speech recognition",
              }));
              if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
              fallbackTimerRef.current = setTimeout(() => {
                setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
              }, 5000);
              // Continue — provider will be created below with new backend
            }
          }
        } else {
          // Whisper is available — reset failure counter and recover if needed
          console.info("[voice] Capability check: Whisper available");
          capCheckFailCountRef.current = 0;
          if (state.backend === "web-speech") {
            // Recover from previous fallback
            providerRef.current?.dispose();
            providerRef.current = null;
            streamingAvailableRef.current = whisper.features?.includes("voice-streaming") ?? false;
            setState((s) => ({
              ...s,
              backend: "whisper",
              fallbackNotice: null,
            }));
          }
        }
      } catch (err) {
        console.warn("[voice] Capabilities check failed:", err);
        // Network error — don't count as Whisper being unavailable,
        // proceed with current backend
      }
    }

    // Lazily create provider on first use (backendRef reflects any fallback changes)
    if (!providerRef.current) {
      if (backendRef.current === "whisper") {
        providerRef.current = streamingAvailableRef.current
          ? new VoiceStreamProvider()
          : new WhisperProvider();
        console.info("[voice] Provider:", streamingAvailableRef.current ? "VoiceStream" : "WhisperHTTP");
      } else if (backendRef.current === "web-speech") {
        providerRef.current = new WebSpeechProvider();
        console.info("[voice] Provider: WebSpeech");
      } else {
        return;
      }
    }

    // Set language from store. "auto" omits the language param so Whisper
    // auto-detects. BCP-47 tags like "en-US" map to Whisper's ISO-639-1
    // codes by taking the prefix before the hyphen.
    const provider = providerRef.current;
    const langCode = voiceLanguage === "auto" ? "" : (voiceLanguage.split("-")[0] ?? "en");
    if ("language" in provider) provider.language = langCode;
    if ("lang" in provider) {
      (provider as WebSpeechProvider).lang = voiceLanguage === "auto"
        ? (navigator.language || "en-US")
        : voiceLanguage;
    }
    provider.onResult = (text) => {
      console.info("[voice] Transcript:", text.length, "chars");
      if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
      vadActiveRef.current = false;
      vadRef.current.state = "idle";
      stopLevelMonitor();
      setState((s) => ({ ...s, isRecording: false, isTranscribing: false, error: null, audioLevel: 0, partialTranscript: "" }));
      onTranscriptRef.current(text);
    };
    provider.onError = (error) => {
      if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
      vadActiveRef.current = false;
      vadRef.current.state = "idle";
      stopLevelMonitor();

      // Whisper failed after retry — try falling back to Web Speech
      if (error === WHISPER_FAILED_SENTINEL) {
        const Ctor = window.SpeechRecognition ?? window.webkitSpeechRecognition;
        if (Ctor) {
          providerRef.current = new WebSpeechProvider();
          setState((s) => ({
            ...s,
            isRecording: false,
            isTranscribing: false,
            error: null,
            audioLevel: 0,
            backend: "web-speech",
            fallbackNotice: "Whisper unavailable \u2014 using browser speech recognition",
          }));
          if (fallbackTimerRef.current) clearTimeout(fallbackTimerRef.current);
          fallbackTimerRef.current = setTimeout(() => {
            setState((s) => (s.fallbackNotice ? { ...s, fallbackNotice: null } : s));
          }, 5000);
          return;
        }
        // No Web Speech either — show generic error
        setState((s) => ({
          ...s,
          isRecording: false,
          isTranscribing: false,
          error: "Transcription failed",
          audioLevel: 0,
        }));
        return;
      }

      setState((s) => ({ ...s, isRecording: false, isTranscribing: false, error, audioLevel: 0 }));
    };
    if (provider.onPartial !== undefined) {
      provider.onPartial = (text) => {
        setState((s) => ({ ...s, partialTranscript: text }));
      };
    }

    // Clear previous error but don't set isRecording yet — provider.start()
    // may trigger a permission prompt. We only show recording state after
    // the mic is actually acquired.
    setState((s) => ({ ...s, error: null }));
    await provider.start();

    // If start() failed (e.g. permission denied), onError already set state.
    // Check if the mic stream was acquired before entering recording state.
    const stream = provider.getStream();
    if (stream) {
      // Arm VAD if requested
      if (opts?.vadEnabled) {
        vadActiveRef.current = true;
        vadRef.current = createVadRefs();
        vadRef.current.state = "calibrating";
        vadRef.current.recordingStart = Date.now();
      }

      console.info("[voice] Recording started");
      setState((s) => ({ ...s, isRecording: true }));
      startLevelMonitor(stream);

      // Warn if no audio detected after 2s (catches dead/muted mics)
      if (noAudioTimerRef.current) clearTimeout(noAudioTimerRef.current);
      noAudioTimerRef.current = setTimeout(() => {
        if (audioLevelRef.current === 0) {
          setState((s) => s.isRecording ? { ...s, error: "No audio detected \u2014 check your microphone" } : s);
          console.warn("[voice] No audio detected after 2s");
        }
      }, 2000);

      // If stop was requested during async start, abort immediately.
      if (stopRequestedRef.current) {
        stopRequestedRef.current = false;
        startingRef.current = false;
        if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
        vadActiveRef.current = false;
        vadRef.current.state = "idle";
        stopLevelMonitor();
        setState((s) => ({
          ...s,
          isRecording: false,
          isTranscribing: false,
          audioLevel: 0,
          partialTranscript: "",
        }));
        provider.dispose();
        providerRef.current = null;
        return;
      }
    }
    startingRef.current = false;
  }, [state.isRecording, state.backend, voiceLanguage, startLevelMonitor, stopLevelMonitor]);

  const stopRecording = useCallback(() => {
    // If start is in progress, signal it to abort after completing
    if (startingRef.current) {
      stopRequestedRef.current = true;
      return;
    }

    const provider = providerRef.current;
    if (!provider || !state.isRecording) return;

    console.info("[voice] Recording stopped");
    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    stopLevelMonitor();
    setState((s) => ({
      ...s,
      isRecording: false,
      isTranscribing: state.backend === "whisper",
      audioLevel: 0,
      partialTranscript: "",
    }));
    provider.stop();
  }, [state.isRecording, state.backend, stopLevelMonitor]);

  const cancelTranscription = useCallback(() => {
    const provider = providerRef.current;
    if (!provider || !state.isTranscribing) return;

    console.info("[voice] Transcription cancelled");
    // Null out callbacks so in-flight HTTP/WS operations don't update state
    provider.onResult = null;
    provider.onError = null;
    if (provider.onPartial !== undefined) provider.onPartial = null;
    provider.dispose();
    providerRef.current = null;

    if (noAudioTimerRef.current) { clearTimeout(noAudioTimerRef.current); noAudioTimerRef.current = null; }
    vadActiveRef.current = false;
    vadRef.current.state = "idle";
    stopLevelMonitor();

    setState((s) => ({
      ...s,
      isRecording: false,
      isTranscribing: false,
      error: null,
      audioLevel: 0,
      partialTranscript: "",
    }));
  }, [state.isTranscribing, stopLevelMonitor]);

  // Keep the ref in sync so the tick loop can call stopRecording
  stopRecordingRef.current = stopRecording;

  return {
    ...state,
    startRecording,
    stopRecording,
    cancelTranscription,
  };
}
