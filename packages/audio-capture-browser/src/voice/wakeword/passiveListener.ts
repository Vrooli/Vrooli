// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// PassiveListener — runs a lightweight background loop that listens for the
// wake word using VAD + ring buffer + MFCC/DTW, without streaming any audio
// to the backend.
//
// Lifecycle: start() → passive VAD loop → speech detected → capture audio →
//            extract MFCC → compare DTW → fire onWakeWordDetected or discard.

import { AudioRingBuffer, createPassiveCapturePipeline, downsample } from "../audioUtils";
import { acquireMicStream, releaseMicLease, type MicLease, type MicReleaseReason } from "../micOwnership";
import { createPassiveVadRefs, vadTick, type VadRefs } from "../vad";
import { WAKE_WORD_AUDIO_CONSTRAINTS, type WakeWordEngine, type WakeWordTemplate } from "./types";

/** Minimum speech duration (ms) before MFCC extraction. Filters clicks/coughs. */
const MIN_SPEECH_DURATION_MS = 400;
/** Cooldown (ms) after a failed match before next capture attempt. */
const MATCH_DEBOUNCE_MS = 500;
/** Maximum capture duration (ms) — hard cap to prevent runaway capture. */
const MAX_CAPTURE_MS = 3000;
/** Ring buffer capacity in seconds. */
const RING_BUFFER_SECONDS = 3;
/** Pre-roll before speech onset to capture attack transients (ms). */
const PRE_ROLL_MS = 300;
/** MFCC target sample rate. */
const MFCC_SAMPLE_RATE = 16000;
/** Throttle RAF tick to ~15Hz. */
const TICK_THROTTLE_MS = 66;

export interface PassiveListenerOpts {
  engine: WakeWordEngine;
  template: WakeWordTemplate;
  /** Fired when the wake word is detected. Receives the mic MediaStream for handoff. */
  onWakeWordDetected: (stream: MediaStream) => void;
  onError: (error: string) => void;
  /**
   * Fired when the mic lease is released for ANY reason — our own dispose, the
   * OS revoking the device, or page-lifecycle emergency cleanup (tab hidden /
   * pagehide). Lets the host drive passive UI state back to "not listening" so
   * the mic control never shows idle while a stream is (or was) live.
   */
  onMicReleased?: (reason: MicReleaseReason) => void;
  /** Optional shared AudioContext (reused to avoid browser limit). */
  audioContext?: AudioContext;
  /** Injectable browser-audio seams used by deterministic tests and embedders. */
  seams?: Partial<PassiveListenerSeams>;
}

export interface PassiveListenerSeams {
  createRingBuffer: (durationSeconds: number, sampleRate: number) => AudioRingBuffer;
  createCapturePipeline: typeof createPassiveCapturePipeline;
  downsample: typeof downsample;
  createVadRefs: typeof createPassiveVadRefs;
  vadTick: typeof vadTick;
}

export class PassiveListener {
  private engine: WakeWordEngine;
  private template: WakeWordTemplate;
  private onWakeWordDetected: (stream: MediaStream) => void;
  private onError: (error: string) => void;
  private onMicReleased?: (reason: MicReleaseReason) => void;
  private readonly seams: PassiveListenerSeams;

  private audioCtx: AudioContext | null = null;
  private ownAudioCtx = false; // whether we created the AudioContext
  private lease: MicLease | null = null;
  private stream: MediaStream | null = null;
  private ringBuffer: AudioRingBuffer | null = null;
  private vad: VadRefs | null = null;
  private analyser: AnalyserNode | null = null;
  private nodes: AudioNode[] = [];
  private rafId = 0;
  private running = false;
  private lastTickTime = 0;
  private speechStartMark = 0;
  private speechStartTime = 0;
  private lastFailedMatchTime = 0;
  private timeDomainData: Float32Array<ArrayBuffer> | null = null;

  constructor(opts: PassiveListenerOpts) {
    this.engine = opts.engine;
    this.template = opts.template;
    this.onWakeWordDetected = opts.onWakeWordDetected;
    this.onError = opts.onError;
    this.onMicReleased = opts.onMicReleased;
    this.seams = {
      createRingBuffer: (durationSeconds, sampleRate) => new AudioRingBuffer(durationSeconds, sampleRate),
      createCapturePipeline: createPassiveCapturePipeline,
      downsample,
      createVadRefs: createPassiveVadRefs,
      vadTick,
      ...opts.seams,
    };
    if (opts.audioContext) {
      this.audioCtx = opts.audioContext;
      this.ownAudioCtx = false;
    }
  }

  async start(): Promise<void> {
    if (this.running) return;
    this.running = true;

    // Acquire mic through the ownership registry — identical constraints to
    // enrollment + the settings test so the acoustic channel matches at
    // detection time. The lease's onRelease fires when the stream is released
    // for ANY reason (our dispose, OS revoke, page-hidden emergency cleanup),
    // making the listener inert and notifying the host.
    let lease: MicLease;
    try {
      lease = await acquireMicStream("passive-wake-word", { audio: WAKE_WORD_AUDIO_CONSTRAINTS }, {
        onRelease: (reason) => {
          if (this.lease === lease) {
            this.lease = null;
            this.stream = null;
          }
          this.stopLoop();
          this.onMicReleased?.(reason);
        },
      });
    } catch (err) {
      this.running = false;
      this.onError(`Passive listener failed to start: ${err}`);
      return;
    }

    // Disposed during the async acquire — do not leak the stream we just got.
    if (!this.running) {
      releaseMicLease(lease, "owner-replaced");
      return;
    }

    this.lease = lease;
    this.stream = lease.stream;

    try {
      // Reuse or create AudioContext
      if (!this.audioCtx) {
        this.audioCtx = new AudioContext();
        this.ownAudioCtx = true;
      }
      if (this.audioCtx.state === "suspended") {
        await this.audioCtx.resume();
      }

      const source = this.audioCtx.createMediaStreamSource(this.stream);
      this.ringBuffer = this.seams.createRingBuffer(RING_BUFFER_SECONDS, this.audioCtx.sampleRate);

      const pipeline = this.seams.createCapturePipeline(this.audioCtx, source, this.ringBuffer);
      this.analyser = pipeline.analyser;
      this.nodes = [source, ...pipeline.nodes];

      // Initialize VAD for passive mode
      this.vad = this.seams.createVadRefs();
      this.vad.state = "calibrating";
      this.vad.recordingStart = performance.now();

      // Allocate time-domain buffer for RMS calculation
      this.timeDomainData = new Float32Array(this.analyser.fftSize);

      // Start RAF tick loop
      this.lastTickTime = 0;
      this.speechStartMark = 0;
      this.speechStartTime = 0;
      this.lastFailedMatchTime = 0;
      this.tick();
    } catch (err) {
      // Setup failed AFTER getUserMedia succeeded. Release the acquired stream
      // so no mic track leaks (the historical passive-listener leak). The lease
      // onRelease nulls this.lease/this.stream and stops the loop.
      this.running = false;
      releaseMicLease(this.lease, "setup-error");
      this.onError(`Passive listener failed to start: ${err}`);
    }
  }

  /** Stop the RAF loop and tear down audio nodes; leaves the stream/lease alone. */
  private stopLoop(): void {
    this.running = false;
    if (this.rafId) {
      cancelAnimationFrame(this.rafId);
      this.rafId = 0;
    }
    this.cleanupAudioNodes();
  }

  stop(): void {
    this.stopLoop();
  }

  /** Full cleanup including mic stream and AudioContext. */
  dispose(reason: MicReleaseReason = "unmount"): void {
    this.stopLoop();
    if (this.lease) {
      // Releasing the lease stops tracks and runs onRelease (nulls stream/lease).
      releaseMicLease(this.lease, reason);
    } else {
      this.stream = null;
    }
    if (this.ownAudioCtx && this.audioCtx) {
      this.audioCtx.close().catch(() => {});
      this.audioCtx = null;
    }
  }

  /** Get the mic stream for handoff to VoiceStreamProvider. */
  getStream(): MediaStream | null {
    return this.stream;
  }

  /** Get the shared AudioContext for reuse by the active recording pipeline. */
  getAudioContext(): AudioContext | null {
    return this.audioCtx;
  }

  private cleanupAudioNodes(): void {
    for (const node of this.nodes) {
      try { node.disconnect(); } catch { /* already disconnected */ }
    }
    this.nodes = [];
    this.analyser = null;
  }

  private tick = (): void => {
    if (!this.running) return;
    this.rafId = requestAnimationFrame(this.tick);

    const now = performance.now();
    if (now - this.lastTickTime < TICK_THROTTLE_MS) return;
    this.lastTickTime = now;

    if (!this.analyser || !this.vad || !this.timeDomainData || !this.ringBuffer) return;

    // Compute RMS
    this.analyser.getFloatTimeDomainData(this.timeDomainData);
    let sum = 0;
    for (let i = 0; i < this.timeDomainData.length; i++) {
      const v = this.timeDomainData[i] ?? 0;
      sum += v * v;
    }
    const rms = Math.sqrt(sum / this.timeDomainData.length);

    // Run VAD tick
    const prevState = this.vad.state;
    const action = this.seams.vadTick(this.vad, rms, now);

    // Track speech onset
    if (prevState === "waitingForSpeech" && this.vad.state === "speechDetected") {
      // Check debounce
      if (now - this.lastFailedMatchTime < MATCH_DEBOUNCE_MS) {
        // Too soon after a failed match — reset and skip
        this.vad.state = "waitingForSpeech";
        return;
      }
      const preRollSamples = Math.round((PRE_ROLL_MS / 1000) * this.ringBuffer.sampleRate);
      this.speechStartMark = this.ringBuffer.mark() - preRollSamples;
      this.speechStartTime = now;
    }

    // Handle VAD actions
    if (action === "segment-boundary" || action === "stop") {
      this.handleCaptureComplete(now);
    } else if (this.vad.state === "speechDetected" && this.speechStartTime > 0) {
      // Hard cap on capture duration
      if (now - this.speechStartTime > MAX_CAPTURE_MS) {
        this.handleCaptureComplete(now);
      }
    }
  };

  private handleCaptureComplete(now: number): void {
    if (!this.vad || !this.ringBuffer) return;

    const speechDuration = now - this.speechStartTime;

    // Reset VAD for next detection cycle
    this.vad.state = "waitingForSpeech";
    this.vad.segmentBoundaryEmitted = false;

    // Skip if speech was too short (likely a click/cough)
    if (speechDuration < MIN_SPEECH_DURATION_MS) {
      this.speechStartMark = 0;
      this.speechStartTime = 0;
      return;
    }

    // Extract captured audio from ring buffer
    const captured = this.ringBuffer.extractSinceMark(Math.max(0, this.speechStartMark));
    this.speechStartMark = 0;
    this.speechStartTime = 0;

    if (captured.length === 0) return;

    // Downsample to MFCC target rate
    const downsampled = this.seams.downsample(captured, this.ringBuffer.sampleRate, MFCC_SAMPLE_RATE);

    // Extract MFCC features and compare
    const candidate = this.engine.extractFeatures(downsampled, MFCC_SAMPLE_RATE);
    const result = this.engine.compareBest(
      candidate,
      this.template.samples,
      this.template.threshold,
      this.template.calibration,
    );

    if (result.isMatch) {
      // Wake word detected! Stop the passive loop but keep the stream alive
      // for handoff to VoiceStreamProvider.
      this.running = false;
      if (this.rafId) {
        cancelAnimationFrame(this.rafId);
        this.rafId = 0;
      }
      this.cleanupAudioNodes();
      if (this.stream) {
        this.onWakeWordDetected(this.stream);
      }
    } else {
      // No match — record the time for debounce
      this.lastFailedMatchTime = performance.now();
    }
  }
}
