import type { TTSPlaybackCapabilities, TTSPlaybackProgressCallback, TTSPlaybackState, TTSProvider, TTSSpeakOptions } from "./types";

/**
 * TTS provider backed by the native Web Speech API (SpeechSynthesis).
 *
 * Supports pause/resume via speechSynthesis.pause()/resume().
 * Seeking and speed/volume adjustment are not available through this API.
 *
 * Known limitation: Chrome silently cancels paused utterances after ~15 s.
 * There is no workaround — this is a browser-level constraint.
 */
export class BrowserTTSProvider implements TTSProvider {
  private _isSpeaking = false;
  private _isPaused = false;

  /** See TTSProvider.onSettled — used by the playback registry to dispose an
   *  orphaned provider once it goes idle. */
  onSettled: (() => void) | null = null;

  private settle(): void {
    this._isSpeaking = false;
    this._isPaused = false;
    this.onSettled?.();
  }

  readonly capabilities: TTSPlaybackCapabilities = {
    canPause: true,
    canSeek: false,
    canAdjustSpeed: false,
    canAdjustVolume: false,
  };

  async speak(text: string, opts?: TTSSpeakOptions): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      window.speechSynthesis.cancel();
      const utterance = new SpeechSynthesisUtterance(text);

      if (opts?.voice) {
        const match = window.speechSynthesis.getVoices().find((v) => v.name === opts.voice);
        if (match) utterance.voice = match;
      }
      utterance.rate = opts?.rate ?? 1.0;
      utterance.pitch = opts?.pitch ?? 1.0;

      this._isSpeaking = true;
      this._isPaused = false;
      utterance.onend = () => {
        this.settle();
        resolve();
      };
      utterance.onerror = (e) => {
        this.settle();
        reject(new Error(e.error));
      };
      window.speechSynthesis.speak(utterance);
    });
  }

  stop(): void {
    window.speechSynthesis.cancel();
    this.settle();
  }

  get isSpeaking(): boolean {
    return this._isSpeaking;
  }

  dispose(): void {
    this.stop();
  }

  unlock(_force = false): Promise<boolean> {
    return Promise.resolve(true);
  }

  isUnlocked(): boolean {
    return true;
  }

  pause(): void {
    if (this._isSpeaking && !this._isPaused) {
      window.speechSynthesis.pause();
      this._isPaused = true;
    }
  }

  resume(): void {
    if (this._isSpeaking && this._isPaused) {
      window.speechSynthesis.resume();
      this._isPaused = false;
    }
  }

  /** SpeechSynthesis provides no timing info, so state reflects minimal data. */
  getPlaybackState(): TTSPlaybackState {
    return {
      currentTime: 0,
      duration: null,
      isPaused: this._isPaused,
      playbackRate: 1,
      volume: 1,
      isMuted: false,
      capabilities: this.capabilities,
    };
  }

  /** SpeechSynthesis has no progress events — this is a no-op. */
  onProgress(_callback: TTSPlaybackProgressCallback | null): void {
    // No-op: Web Speech API does not expose timing/progress data.
  }
}
