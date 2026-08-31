import type { TTSPlaybackCapabilities, TTSPlaybackProgressCallback, TTSPlaybackState, TTSProvider, TTSSpeakOptions } from "./types";
import type { TTSCacheControl, TTSSynthesisMetrics } from "./runtime";

export type KokoroSynthesizeFn = (
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
  cache?: TTSCacheControl,
) => Promise<Blob>;

export type KokoroSynthesizeWithMetricsFn = (
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
  cache?: TTSCacheControl,
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
  reportTTSPlayStart?: (metrics: TTSSynthesisMetrics) => void;
  /** Called when the server identifies the provider that synthesized audio. */
  onProviderResolved?: (providerId: string, providerTier?: string) => void;
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

/**
 * Non-fatal outcome for a single paragraph inside speakSequence when its
 * primary synth+play failed but the sequence continued: it was recovered by a
 * retry, recovered via the browser voice, or skipped after all recovery failed.
 * Surfaced through KokoroProvider.onParagraphOutcome for observability; it never
 * affects whether the remaining paragraphs play.
 */
export type TTSParagraphOutcome = "retried" | "fell-back" | "skipped";

/** True for an intentional stop/dispose abort (as opposed to a paragraph
 *  synth-reject or decode error, which must NOT halt the whole sequence). */
function isAbortError(err: unknown): boolean {
  return (
    err instanceof DOMException
      ? err.name === "AbortError"
      : typeof err === "object" && err !== null && (err as { name?: string }).name === "AbortError"
  );
}

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

  /**
   * Optional non-fatal per-paragraph outcome hook, fired when a paragraph in
   * speakSequence was retried, fell back to the browser voice, or was skipped.
   * Wired by the observability layer; it never blocks or halts playback.
   */
  onParagraphOutcome: ((info: { index: number; outcome: TTSParagraphOutcome; text: string }) => void) | null = null;

  /**
   * Fired whenever the provider settles to idle (a terminal speak call finished,
   * was stopped, or errored). Used by the playback registry to dispose an
   * orphaned provider once its tail completes. Non-fatal; never gates playback.
   */
  onSettled: (() => void) | null = null;

  readonly capabilities: TTSPlaybackCapabilities = {
    canPause: true,
    canSeek: true,
    canAdjustSpeed: true,
    canAdjustVolume: true,
  };

	private readonly synthesizeWithMetrics: KokoroSynthesizeWithMetricsFn;
	private readonly reportTTSPlayStart?: (metrics: TTSSynthesisMetrics) => void;
	private readonly onProviderResolved?: (providerId: string, providerTier?: string) => void;

	constructor(options: KokoroProviderOptions = {}) {
		this.reportTTSPlayStart = options.reportTTSPlayStart;
		this.onProviderResolved = options.onProviderResolved;
    if (options.synthesizeWithMetrics) {
      this.synthesizeWithMetrics = options.synthesizeWithMetrics;
    } else if (options.synthesize) {
      const synth = options.synthesize;
      this.synthesizeWithMetrics = async (input, voice, speed, signal, cache) => ({
        blob: await synth(input, voice, speed, signal, cache),
        // Test-injected synthesize: no requestId, telemetry sentinel only.
        metrics: { requestId: "test-no-rid", synthStartMs: 0, totalChars: input.length },
      });
    } else {
		this.synthesizeWithMetrics = async () => {
			throw new Error("Kokoro runtime is not configured; pass a TTSRuntime adapter");
		};
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

  /** Build server cache-control for a paragraph. Only meaningful when the
   *  caller supplied an eventId (Kokoro auto/replay path); undefined otherwise,
   *  so one-off / test speech is never cached. */
  private cacheControl(opts: TTSSpeakOptions | undefined, chunkIndex: number): TTSCacheControl | undefined {
    if (!opts?.eventId) return undefined;
    return { eventId: opts.eventId, version: opts.version, chunkIndex };
  }

  async speak(text: string, opts?: TTSSpeakOptions): Promise<void> {
    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;
    this._isPaused = false;
    const signal = this.abortController.signal;

    try {
      const { blob, metrics } = await this.synthesizeWithMetrics(text, opts?.voice, opts?.rate, signal, this.cacheControl(opts, 0));
      if (metrics.providerId) this.onProviderResolved?.(metrics.providerId, metrics.providerTier);
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
   *
   * TTS twin of the event-durability contract: each paragraph is a DURABLE
   * ORDERED UNIT. A single-paragraph fault is isolated on that unit
   * (playParagraphResilient: retry → browser-voice fallback → skip-with-notice)
   * and MUST NOT truncate the paragraphs after it — survivors synthesize and
   * play to completion, in order. Only a real abort (stop/dispose) halts the
   * tail. This is the playback analogue of a durable STT segment.
   * See scenarios/audio-tools/docs/domains/stt/streaming-pipeline.md#event-durability-contract.
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
        synths[i] = this.synthesizeWithMetrics(text, opts?.voice, opts?.rate, signal, this.cacheControl(opts, i))
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
        if (signal.aborted) throw this.createAbortError();
        const promise = synths[i];
        if (!promise) continue;
        // Per-paragraph resilience: a single paragraph's synth-reject or decode
        // error is isolated (retry once, else browser-voice fallback, else
        // skip-with-notice) and the sequence CONTINUES. Only a real abort
        // (stop/dispose) propagates and halts the tail.
        await this.playParagraphResilient(i, promise, texts[i] ?? "", opts, signal);
      }
      this.cleanup();
    } catch (err) {
      // Only a genuine abort reaches here now — paragraph failures are handled
      // inline. Cancel any not-yet-awaited synths so they abort upstream.
      this.abortController?.abort();
      this.cleanup();
      throw err;
    }
  }

  /**
   * Play one paragraph of a sequence with graceful degradation. Attempts, in
   * order: (1) the already-pipelined synth + play; (2) one fresh re-synth +
   * play; (3) the browser speech voice; else skip. Every step short-circuits on
   * a real abort (rethrown). A non-abort failure never halts the sequence — the
   * outcome is reported via onParagraphOutcome and playback moves on.
   */
  private async playParagraphResilient(
    index: number,
    synthPromise: Promise<{ blob: Blob; metrics: TTSSynthesisMetrics }>,
    text: string,
    opts: TTSSpeakOptions | undefined,
    signal: AbortSignal,
  ): Promise<void> {
    // Attempt 1: the in-flight pipelined synth.
    try {
      await this.synthAndPlay(synthPromise, signal);
      return;
    } catch (err) {
      if (isAbortError(err) || signal.aborted) throw err;
    }

    // Attempt 2: retry this paragraph once with a fresh synth.
    try {
      const retry = this.synthesizeWithMetrics(text, opts?.voice, opts?.rate, signal, this.cacheControl(opts, index));
      retry.catch(() => undefined); // bound the unhandled-rejection window
      await this.synthAndPlay(retry, signal);
      this.reportParagraphOutcome(index, "retried", text);
      return;
    } catch (err) {
      if (isAbortError(err) || signal.aborted) throw err;
    }

    // Attempt 3: per-paragraph browser-voice fallback.
    const spoke = await this.speakViaBrowser(text, opts, signal);
    if (signal.aborted) throw this.createAbortError();
    if (spoke) {
      this.reportParagraphOutcome(index, "fell-back", text);
      return;
    }

    // All recovery failed: skip THIS paragraph only, keep the sequence going.
    this.reportParagraphOutcome(index, "skipped", text);
  }

  /** Await a synth and play its blob, mapping abort to createAbortError. A
   *  0-byte blob (non-speakable input) counts as a clean, played paragraph. */
  private async synthAndPlay(
    synthPromise: Promise<{ blob: Blob; metrics: TTSSynthesisMetrics }>,
    signal: AbortSignal,
  ): Promise<void> {
    const { blob, metrics } = await synthPromise;
    if (metrics.providerId) this.onProviderResolved?.(metrics.providerId, metrics.providerTier);
    this.throwIfAborted(signal);
    if (blob.size === 0) return;
    await this.playBlobAndWait(blob, metrics);
    if (signal.aborted) throw this.createAbortError();
  }

  /**
   * Speak one paragraph with the browser SpeechSynthesis voice. Resolves true
   * when the utterance finished, false when the browser voice is unavailable or
   * errored. Honors the abort signal (cancels in-flight speech).
   */
  private speakViaBrowser(text: string, opts: TTSSpeakOptions | undefined, signal: AbortSignal): Promise<boolean> {
    const synth = typeof window !== "undefined" ? window.speechSynthesis : undefined;
    if (!synth || typeof SpeechSynthesisUtterance === "undefined" || !text.trim()) {
      return Promise.resolve(false);
    }
    if (signal.aborted) return Promise.resolve(false);
    return new Promise<boolean>((resolve) => {
      let settled = false;
      const finish = (ok: boolean): void => {
        if (settled) return;
        settled = true;
        signal.removeEventListener("abort", onAbort);
        resolve(ok);
      };
      const onAbort = (): void => {
        try { synth.cancel(); } catch { /* ignore */ }
        finish(false);
      };
      const utter = new SpeechSynthesisUtterance(text);
      if (opts?.rate) utter.rate = opts.rate;
      utter.onend = () => finish(true);
      utter.onerror = () => finish(false);
      signal.addEventListener("abort", onAbort, { once: true });
      try {
        synth.speak(utter);
      } catch {
        finish(false);
      }
    });
  }

  private reportParagraphOutcome(index: number, outcome: TTSParagraphOutcome, text: string): void {
    try {
      this.onParagraphOutcome?.({ index, outcome, text });
    } catch {
      /* observability must never break playback */
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

  /**
   * Play a sequence of pre-fetched cache blobs (one per paragraph) in order,
   * without any synthesis. The replay counterpart to speakSequence: a fully
   * cached message plays end-to-end from the byte cache. Each blob is its own
   * track (accurate per-paragraph boundaries); a real stop/dispose abort halts
   * the tail, mirroring speakSequence. Empty blobs (non-speakable paragraphs)
   * are skipped.
   */
  async speakFromBlobs(blobs: Blob[]): Promise<void> {
    if (blobs.length === 0) return;
    if (blobs.length === 1 && blobs[0]) return this.speakFromBlob(blobs[0]);

    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;
    this._isPaused = false;
    const signal = this.abortController.signal;

    try {
      for (const blob of blobs) {
        if (signal.aborted) throw this.createAbortError();
        if (!blob || blob.size === 0) continue;
        await this.playBlobAndWait(blob, null);
        if (signal.aborted) throw this.createAbortError();
      }
      this.cleanup();
    } catch (err) {
      this.abortController?.abort();
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
			try { this.reportTTSPlayStart?.(metrics); } catch { /* never block playback */ }
        }
      }).catch((err: unknown) => {
        this.playbackResolve = null;
        this.playbackReject = null;
        // Do NOT cleanup() here: speakSequence isolates a single paragraph's
        // play failure and continues, so flipping _isSpeaking=false mid-sequence
        // would corrupt the run. The terminal caller (speak / speakSequence /
        // speakFromBlob catch, or stop) owns cleanup.
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
    // Do NOT cleanup() here (see playBlobAndWait catch): a decode error on one
    // paragraph must not tear down a sequence that will retry / continue. The
    // terminal caller owns cleanup.
    const mediaError = this.audio.error;
    reject?.(new Error(mediaError?.message ?? "Audio playback error"));
  };

  private cleanup(): void {
    this._isSpeaking = false;
    this._isPaused = false;
    // Notify the playback registry (if any) that this provider is now idle, so
    // a handed-off/orphaned provider can be disposed once its tail completes.
    this.onSettled?.();
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
