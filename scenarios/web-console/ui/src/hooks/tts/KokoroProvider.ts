import type { TTSProvider, TTSSpeakOptions } from "./types";
import { synthesizeTTS } from "../../lib/api";

export class KokoroProvider implements TTSProvider {
  private audio: HTMLAudioElement | null = null;
  private objectUrl: string | null = null;
  private _isSpeaking = false;
  private abortController: AbortController | null = null;

  async speak(text: string, opts?: TTSSpeakOptions): Promise<void> {
    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;

    try {
      const blob = await synthesizeTTS(text, opts?.voice, opts?.rate, this.abortController.signal);
      // Check if stopped while fetching
      if (this.abortController?.signal.aborted) return;

      this.objectUrl = URL.createObjectURL(blob);
      const audio = new Audio(this.objectUrl);
      this.audio = audio;

      return new Promise<void>((resolve, reject) => {
        audio.onended = () => {
          this.cleanup();
          resolve();
        };
        audio.onerror = () => {
          this.cleanup();
          reject(new Error("Audio playback failed"));
        };
        audio.play().catch((err) => {
          this.cleanup();
          reject(err);
        });
      });
    } catch (err) {
      this.cleanup();
      throw err;
    }
  }

  stop(): void {
    this.abortController?.abort();
    this.abortController = null;
    if (this.audio) {
      this.audio.pause();
      this.audio.onended = null;
      this.audio.onerror = null;
      this.audio = null;
    }
    this.cleanup();
  }

  get isSpeaking(): boolean {
    return this._isSpeaking;
  }

  dispose(): void {
    this.stop();
  }

  private cleanup(): void {
    this._isSpeaking = false;
    if (this.objectUrl) {
      URL.revokeObjectURL(this.objectUrl);
      this.objectUrl = null;
    }
  }
}
