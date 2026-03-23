import type { TTSPlaybackCapabilities, TTSPlaybackProgressCallback, TTSPlaybackState, TTSProvider, TTSSpeakOptions } from "./types";
import { synthesizeTTS } from "../../lib/api";

/**
 * TTS provider backed by the Kokoro synthesis API.
 *
 * Playback pipeline:
 *   MP3 blob → URL.createObjectURL() → HTMLAudioElement.play()
 *
 * A single reusable HTMLAudioElement is created in the constructor.
 * Blob URLs are revoked on stop / dispose / next speak call to prevent leaks.
 */
export class KokoroProvider implements TTSProvider {
  private _isSpeaking = false;
  private _isPaused = false;
  private abortController: AbortController | null = null;
  private audio: HTMLAudioElement;
  private blobUrl: string | null = null;
  private playbackResolve: (() => void) | null = null;
  private playbackReject: ((reason?: unknown) => void) | null = null;
  private progressCallback: TTSPlaybackProgressCallback | null = null;

  readonly capabilities: TTSPlaybackCapabilities = {
    canPause: true,
    canSeek: true,
    canAdjustSpeed: true,
    canAdjustVolume: true,
  };

  constructor() {
    this.audio = new Audio();
    this.audio.addEventListener("timeupdate", this.handleTimeUpdate);
    this.audio.addEventListener("ended", this.handleEnded);
    this.audio.addEventListener("error", this.handleError);
  }

  async speak(text: string, opts?: TTSSpeakOptions): Promise<void> {
    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;
    this._isPaused = false;
    const signal = this.abortController.signal;

    try {
      const blob = await synthesizeTTS(text, opts?.voice, opts?.rate, signal);
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
        const blob = await synthesizeTTS(text, opts?.voice, opts?.rate, signal);
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
