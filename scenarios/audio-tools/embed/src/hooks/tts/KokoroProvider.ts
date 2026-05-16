import type { TTSPlaybackCapabilities, TTSPlaybackProgressCallback, TTSPlaybackState, TTSProvider, TTSSpeakOptions } from "./types";
import * as ttsApi from "../../api/tts";

export type KokoroSynthesizeFn = (
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
) => Promise<Blob>;

export interface KokoroProviderOptions {
  /**
   * Override the synthesize implementation. Consumer test environments
   * that can't intercept the embed package barrel (vi.mock on
   * "@audio-tools/embed" doesn't reach embed-internal relative imports)
   * should pass their own mock here. Defaults to the live audio-tools
   * Connect client wired to window.__AUDIO_TOOLS_URL__.
   */
  synthesize?: KokoroSynthesizeFn;
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

  private readonly synthesize: KokoroSynthesizeFn;

  constructor(options: KokoroProviderOptions = {}) {
    this.synthesize = options.synthesize ?? ((input, voice, speed, signal) =>
      ttsApi.synthesizeTTS(input, voice, speed, signal));
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
      // Set a silent source and play within the caller's gesture stack.
      // After the silent play resolves we clean the element up (pause,
      // clear src, reload) so the next speakFromBlob sets a fresh state
      // without leftover format/source metadata confusing the decoder.
      // Chrome's autoplay eligibility is stored on HTMLMediaElement and
      // survives a load() reset — see the separate `unlocked` flag.
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
      const blob = await this.synthesize(text, opts?.voice, opts?.rate, signal);
      this.throwIfAborted(signal);

      // Kokoro returns 0-byte audio for non-speakable input (e.g. "---",
      // lone punctuation). Skip silently instead of crashing playback.
      if (blob.size === 0) {
        this.cleanup();
        return;
      }

      this.revokeBlobUrl();
      this.blobUrl = URL.createObjectURL(blob);
      this.audio.src = this.blobUrl;
      this.audio.currentTime = 0;

      return await new Promise<void>((resolve, reject) => {
        this.playbackResolve = resolve;
        this.playbackReject = reject;
        this.audio.play().catch((err) => {
          this.playbackResolve = null;
          this.playbackReject = null;
          this.cleanup();
          reject(err);
        });
      });
    } catch (err) {
      this.cleanup();
      throw err;
    }
  }

  /**
   * Synthesize all texts, concatenate the resulting MP3 blobs, and play
   * the combined audio as a single track. This gives accurate total
   * duration and full seek/scrub across the entire sequence.
   *
   * MP3 frames are self-contained, so byte concatenation produces a
   * valid stream without re-encoding.
   */
  async speakSequence(texts: string[], opts?: TTSSpeakOptions): Promise<void> {
    if (texts.length === 0) return;
    if (texts.length === 1 && texts[0]) return this.speak(texts[0], opts);

    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;
    this._isPaused = false;
    const signal = this.abortController.signal;

    try {
      // Synthesize all segments sequentially (preserves order, respects abort)
      const blobs: Blob[] = [];
      for (const text of texts) {
        const blob = await this.synthesize(text, opts?.voice, opts?.rate, signal);
        this.throwIfAborted(signal);
        if (blob.size > 0) blobs.push(blob);
      }

      // All segments were non-speakable (e.g. "---")
      if (blobs.length === 0) {
        this.cleanup();
        return;
      }

      // Concatenate MP3 blobs into a single blob
      const combined = new Blob(blobs, { type: blobs[0]?.type ?? "audio/mpeg" });

      this.revokeBlobUrl();
      this.blobUrl = URL.createObjectURL(combined);
      this.audio.src = this.blobUrl;
      this.audio.currentTime = 0;

      return await new Promise<void>((resolve, reject) => {
        this.playbackResolve = resolve;
        this.playbackReject = reject;
        this.audio.play().catch((err) => {
          this.playbackResolve = null;
          this.playbackReject = null;
          this.cleanup();
          reject(err);
        });
      });
    } catch (err) {
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

    this.revokeBlobUrl();
    this.blobUrl = URL.createObjectURL(blob);
    this.audio.src = this.blobUrl;
    this.audio.currentTime = 0;

    return new Promise<void>((resolve, reject) => {
      this.playbackResolve = resolve;
      this.playbackReject = reject;
      this.audio.play().catch((err) => {
        this.playbackResolve = null;
        this.playbackReject = null;
        this.cleanup();
        reject(err);
      });
    });
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

  private handleTimeUpdate = (): void => {
    if (this.progressCallback && Number.isFinite(this.audio.duration)) {
      this.progressCallback(this.audio.currentTime, this.audio.duration);
    }
  };

  private handleEnded = (): void => {
    const resolve = this.playbackResolve;
    this.playbackResolve = null;
    this.playbackReject = null;
    this.cleanup();
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
