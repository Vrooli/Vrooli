import type { TTSPlaybackCapabilities, TTSPlaybackProgressCallback, TTSPlaybackState, TTSProvider, TTSSpeakOptions } from "./types";
import * as ttsApi from "../../api/tts";
import type { TTSSynthesisMetrics } from "../../api/tts";

export type KokoroSynthesizeFn = (
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
) => Promise<Blob>;

export type KokoroSynthesizeWithMetricsFn = (
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
) => Promise<{ blob: Blob; metrics: TTSSynthesisMetrics }>;

export interface KokoroProviderOptions {
  /**
   * Override the synthesize implementation. Tests pass their own mock
   * here. Defaults to the web-console AudioRuntimeService same-origin
   * Connect client.
   *
   * When provided without `synthesizeWithMetrics`, calls are wrapped to
   * skip telemetry (no requestId is tracked).
   */
  synthesize?: KokoroSynthesizeFn;
  /**
   * Timing-aware synthesize override. When provided, takes precedence over
   * `synthesize` and enables play-start telemetry correlation. Production
   * defaults to `synthesizeTTSWithMetrics`.
   */
  synthesizeWithMetrics?: KokoroSynthesizeWithMetricsFn;
}

/**
 * TTS provider backed by the Kokoro synthesis API.
 *
 * Playback pipeline:
 *   MP3 blob → URL.createObjectURL() → HTMLAudioElement.play()
 *
 * A single reusable HTMLAudioElement is created in the constructor.
 * Blob URLs are revoked on stop / dispose / next speak call to prevent leaks.
 */
// Minimal valid silent WAV: 46 bytes, 16-bit mono, 8 kHz, 1 sample of silence.
// Used once per session to activate the reusable HTMLAudioElement inside a
// user gesture so subsequent programmatic play() calls bypass the browser
// autoplay policy. Must contain at least one sample of real PCM data —
// earlier 0-byte variants caused Chrome to fire MEDIA_ERR_DECODE on every
// `src=` assignment, storming the provider with spurious 'error' events.
const SILENT_WAV_DATA_URL =
  "data:audio/wav;base64,UklGRiYAAABXQVZFZm10IBAAAAABAAEAQB8AAIA+AAACABAAZGF0YQIAAAAAAA==";

// Bound on concurrent in-flight synths inside speakSequence. Allows N+1 to
// be ready by the time N finishes playing, without overloading audio-tools.
// Lives here (not in config) per plan section 8.
const SPEAK_SEQUENCE_CONCURRENCY = 2;

export class KokoroProvider implements TTSProvider {
  private _isSpeaking = false;
  private _isPaused = false;
  private abortController: AbortController | null = null;
  private audio: HTMLAudioElement;
  private blobUrl: string | null = null;
  private playbackResolve: (() => void) | null = null;
  private playbackReject: ((reason?: unknown) => void) | null = null;
  private progressCallback: TTSPlaybackProgressCallback | null = null;
  private unlocked = false;

  readonly capabilities: TTSPlaybackCapabilities = {
    canPause: true,
    canSeek: true,
    canAdjustSpeed: true,
    canAdjustVolume: true,
  };

  private readonly synthesizeWithMetrics: KokoroSynthesizeWithMetricsFn;

  constructor(options: KokoroProviderOptions = {}) {
    if (options.synthesizeWithMetrics) {
      this.synthesizeWithMetrics = options.synthesizeWithMetrics;
    } else if (options.synthesize) {
      const synth = options.synthesize;
      this.synthesizeWithMetrics = async (input, voice, speed, signal) => ({
        blob: await synth(input, voice, speed, signal),
        // Test-injected synthesize: no requestId, telemetry sentinel only.
        metrics: { requestId: "test-no-rid", synthStartMs: 0, totalChars: input.length },
      });
    } else {
      this.synthesizeWithMetrics = (input, voice, speed, signal) =>
        ttsApi.synthesizeTTSWithMetrics(input, voice, speed, signal);
    }
    this.audio = new Audio();
    this.audio.addEventListener("timeupdate", this.handleTimeUpdate);
    this.audio.addEventListener("ended", this.handleEnded);
    this.audio.addEventListener("error", this.handleError);
  }

  async unlock(force = false): Promise<boolean> {
    if (this.unlocked && !force) return true;
    // Never disturb an active playback by reassigning `src` — Chrome's
    // sticky activation must already exist if we're actively playing, and
    // a stray gesture-handler unlock call mid-playback would otherwise
    // interrupt the audio and spray MEDIA_ERR_DECODE events.
    if (this._isSpeaking && !this.audio.paused) {
      this.unlocked = true;
      return true;
    }
    try {
      this.audio.src = SILENT_WAV_DATA_URL;
      this.audio.muted = true;
      await this.audio.play();
      this.audio.pause();
      this.audio.removeAttribute("src");
      this.audio.load();
      this.audio.muted = false;
      this.unlocked = true;
      return true;
    } catch {
      this.audio.muted = false;
      return false;
    }
  }

  isUnlocked(): boolean {
    return this.unlocked;
  }

  async speak(text: string, opts?: TTSSpeakOptions): Promise<void> {
    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;
    this._isPaused = false;
    const signal = this.abortController.signal;

    try {
      const { blob, metrics } = await this.synthesizeWithMetrics(text, opts?.voice, opts?.rate, signal);
      this.throwIfAborted(signal);

      // Kokoro returns 0-byte audio for non-speakable input (e.g. "---",
      // lone punctuation). Skip silently instead of crashing playback.
      if (blob.size === 0) {
        this.cleanup();
        return;
      }
      await this.playBlobAndWait(blob, metrics);
      this.cleanup();
    } catch (err) {
      this.cleanup();
      throw err;
    }
  }

  /**
   * Pipelined sequence playback. Synthesizes up to SPEAK_SEQUENCE_CONCURRENCY
   * paragraphs in parallel so paragraph N+1 is ready (or almost ready) by
   * the time paragraph N finishes playing. Playback order is preserved.
   *
   * Each paragraph is played as its own track — unlike the prior version
   * which concatenated all blobs before starting playback (which forced
   * time-to-first-audio to scale with TOTAL synth duration).
   */
  async speakSequence(texts: string[], opts?: TTSSpeakOptions): Promise<void> {
    if (texts.length === 0) return;
    if (texts.length === 1 && texts[0]) return this.speak(texts[0], opts);

    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;
    this._isPaused = false;
    const signal = this.abortController.signal;

    // Kick synthesis up to CONCURRENCY at a time; build an array of promises
    // indexed by paragraph order so playback consumes them in sequence.
    const synths = new Array<Promise<{ blob: Blob; metrics: TTSSynthesisMetrics }>>(texts.length);
    let nextToKick = 0;
    let inFlight = 0;
    const kick = (): void => {
      while (inFlight < SPEAK_SEQUENCE_CONCURRENCY && nextToKick < texts.length) {
        const i = nextToKick++;
        const text = texts[i] ?? "";
        inFlight++;
        synths[i] = this.synthesizeWithMetrics(text, opts?.voice, opts?.rate, signal)
          .finally(() => { inFlight--; kick(); });
        // Surface synth rejections via the consumer await below; .catch here
        // would swallow them. The unhandled-rejection window is bounded by
        // the await in the consumer loop (always within a tick).
        synths[i].catch(() => undefined);
      }
    };
    kick();

    try {
      for (let i = 0; i < texts.length; i++) {
        const promise = synths[i];
        if (!promise) continue;
        const { blob, metrics } = await promise;
        this.throwIfAborted(signal);
        if (blob.size === 0) continue;
        await this.playBlobAndWait(blob, metrics);
        if (signal.aborted) throw this.createAbortError();
      }
      this.cleanup();
    } catch (err) {
      // Cancel any not-yet-awaited synths so they abort upstream.
      this.abortController?.abort();
      this.cleanup();
      throw err;
    }
  }

  /**
   * Play a pre-fetched audio blob (from the TTS cache) without synthesis.
   * Reuses the same HTMLAudioElement and blob lifecycle as speak().
   */
  async speakFromBlob(blob: Blob): Promise<void> {
    this.stop();
    this._isSpeaking = true;
    this._isPaused = false;

    if (blob.size === 0) {
      this.cleanup();
      return;
    }
    try {
      await this.playBlobAndWait(blob, null);
      this.cleanup();
    } catch (err) {
      this.cleanup();
      throw err;
    }
  }

  stop(): void {
    this.abortController?.abort();
    this.abortController = null;

    if (!this.audio.paused) {
      this.audio.pause();
    }
    this.audio.removeAttribute("src");
    this.audio.load(); // reset the element

    if (this.playbackReject) {
      this.playbackReject(this.createAbortError());
      this.playbackReject = null;
      this.playbackResolve = null;
    }

    this.revokeBlobUrl();
    this.cleanup();
  }

  get isSpeaking(): boolean {
    return this._isSpeaking;
  }

  dispose(): void {
    this.stop();
    this.audio.removeEventListener("timeupdate", this.handleTimeUpdate);
    this.audio.removeEventListener("ended", this.handleEnded);
    this.audio.removeEventListener("error", this.handleError);
  }

  pause(): void {
    if (this._isSpeaking && !this._isPaused) {
      this.audio.pause();
      this._isPaused = true;
    }
  }

  resume(): void {
    if (this._isSpeaking && this._isPaused) {
      void this.audio.play();
      this._isPaused = false;
    }
  }

  seek(seconds: number): void {
    if (this._isSpeaking && Number.isFinite(this.audio.duration)) {
      this.audio.currentTime = Math.max(0, Math.min(seconds, this.audio.duration));
    }
  }

  setPlaybackRate(rate: number): void {
    this.audio.playbackRate = rate;
  }

  setVolume(level: number): void {
    this.audio.volume = Math.max(0, Math.min(1, level));
  }

  getPlaybackState(): TTSPlaybackState {
    return {
      currentTime: this.audio.currentTime,
      duration: Number.isFinite(this.audio.duration) ? this.audio.duration : null,
      isPaused: this._isPaused,
      playbackRate: this.audio.playbackRate,
      volume: this.audio.volume,
      isMuted: false,
      capabilities: this.capabilities,
    };
  }

  onProgress(callback: TTSPlaybackProgressCallback | null): void {
    this.progressCallback = callback;
  }

  // --- Private helpers ---

  /**
   * Load a blob into the reusable HTMLAudioElement, start playback, and
   * resolve when the track ends. Emits a play-start timing event keyed to
   * the synthesis requestId (when metrics are available).
   */
  private playBlobAndWait(blob: Blob, metrics: TTSSynthesisMetrics | null): Promise<void> {
    this.revokeBlobUrl();
    this.blobUrl = URL.createObjectURL(blob);
    this.audio.src = this.blobUrl;
    this.audio.currentTime = 0;
    return new Promise<void>((resolve, reject) => {
      this.playbackResolve = resolve;
      this.playbackReject = reject;
      this.audio.play().then(() => {
        // Skip telemetry for the test-injected synthesize sentinel.
        if (metrics && metrics.requestId !== "test-no-rid") {
          try { ttsApi.reportTTSPlayStart(metrics); } catch { /* never block playback */ }
        }
      }).catch((err: unknown) => {
        this.playbackResolve = null;
        this.playbackReject = null;
        this.cleanup();
        reject(err instanceof Error ? err : new Error(String(err)));
      });
    });
  }

  private handleTimeUpdate = (): void => {
    if (this.progressCallback && Number.isFinite(this.audio.duration)) {
      this.progressCallback(this.audio.currentTime, this.audio.duration);
    }
  };

  private handleEnded = (): void => {
    // Per-track end: resolve the active promise but DO NOT cleanup() —
    // speakSequence chains multiple plays, and flipping _isSpeaking=false
    // between paragraphs would race external observers. The terminal
    // caller (speak/speakSequence/speakFromBlob) runs cleanup() itself.
    const resolve = this.playbackResolve;
    this.playbackResolve = null;
    this.playbackReject = null;
    resolve?.();
  };

  private handleError = (): void => {
    const reject = this.playbackReject;
    this.playbackResolve = null;
    this.playbackReject = null;
    this.cleanup();
    const mediaError = this.audio.error;
    reject?.(new Error(mediaError?.message ?? "Audio playback error"));
  };

  private cleanup(): void {
    this._isSpeaking = false;
    this._isPaused = false;
  }

  private revokeBlobUrl(): void {
    if (this.blobUrl) {
      URL.revokeObjectURL(this.blobUrl);
      this.blobUrl = null;
    }
  }

  private throwIfAborted(signal: AbortSignal): void {
    if (signal.aborted) {
      throw this.createAbortError();
    }
  }

  private createAbortError(): DOMException {
    return new DOMException("The operation was aborted.", "AbortError");
  }
}
