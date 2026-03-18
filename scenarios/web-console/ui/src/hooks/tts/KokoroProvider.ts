import type { TTSProvider, TTSSpeakOptions } from "./types";
import { synthesizeTTS } from "../../lib/api";

export class KokoroProvider implements TTSProvider {
  private _isSpeaking = false;
  private abortController: AbortController | null = null;
  private audioContext: AudioContext | null = null;
  private source: AudioBufferSourceNode | null = null;
  private playbackReject: ((reason?: unknown) => void) | null = null;

  async speak(text: string, opts?: TTSSpeakOptions): Promise<void> {
    this.stop();
    this.abortController = new AbortController();
    this._isSpeaking = true;
    const signal = this.abortController.signal;

    try {
      const blob = await synthesizeTTS(text, opts?.voice, opts?.rate, signal);
      this.throwIfAborted(signal);

      const audioBytes = await blob.arrayBuffer();
      this.throwIfAborted(signal);

      // Kokoro returns 0-byte audio for non-speakable input (e.g. "---",
      // lone punctuation). Skip silently instead of crashing decodeAudioData.
      if (audioBytes.byteLength === 0) {
        this.cleanup();
        return;
      }

      const context = this.getAudioContext();
      if (context.state === "suspended") {
        await context.resume();
      }
      this.throwIfAborted(signal);

      const audioBuffer = await context.decodeAudioData(audioBytes.slice(0));
      this.throwIfAborted(signal);

      return await new Promise<void>((resolve, reject) => {
        const source = context.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(context.destination);
        this.source = source;
        this.playbackReject = reject;
        source.onended = () => {
          if (this.source !== source) return;
          this.source = null;
          this.playbackReject = null;
          this.cleanup();
          resolve();
        };
        try {
          source.start(0);
        } catch (err) {
          source.onended = null;
          this.source = null;
          this.playbackReject = null;
          source.disconnect();
          this.cleanup();
          reject(err);
        }
      });
    } catch (err) {
      this.cleanup();
      throw err;
    }
  }

  stop(): void {
    this.abortController?.abort();
    this.abortController = null;
    if (this.source) {
      const source = this.source;
      this.source = null;
      source.onended = null;
      try {
        source.stop(0);
      } catch {
        // Ignore stop errors for already-ended sources.
      }
      source.disconnect();
    }
    if (this.playbackReject) {
      this.playbackReject(this.createAbortError());
      this.playbackReject = null;
    }
    this.cleanup();
  }

  get isSpeaking(): boolean {
    return this._isSpeaking;
  }

  dispose(): void {
    this.stop();
    if (this.audioContext) {
      void this.audioContext.close();
      this.audioContext = null;
    }
  }

  private cleanup(): void {
    this._isSpeaking = false;
  }

  private getAudioContext(): AudioContext {
    if (!this.audioContext) {
      const AudioContextCtor = window.AudioContext
        ?? (window as Window & typeof globalThis & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!AudioContextCtor) {
        throw new Error("Web Audio is not supported in this browser");
      }
      this.audioContext = new AudioContextCtor();
    }
    return this.audioContext;
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
