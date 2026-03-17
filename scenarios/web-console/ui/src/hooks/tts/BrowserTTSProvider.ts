import type { TTSProvider, TTSSpeakOptions } from "./types";

export class BrowserTTSProvider implements TTSProvider {
  private _isSpeaking = false;

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
      utterance.onend = () => {
        this._isSpeaking = false;
        resolve();
      };
      utterance.onerror = (e) => {
        this._isSpeaking = false;
        reject(new Error((e as SpeechSynthesisErrorEvent).error ?? "Speech synthesis failed"));
      };
      window.speechSynthesis.speak(utterance);
    });
  }

  stop(): void {
    window.speechSynthesis.cancel();
    this._isSpeaking = false;
  }

  get isSpeaking(): boolean {
    return this._isSpeaking;
  }

  dispose(): void {
    this.stop();
  }
}
