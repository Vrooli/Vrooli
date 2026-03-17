// DOC: docs/internal/SEAMS.md#tts-provider-seam
export type TTSBackend = "kokoro" | "browser" | "none";

export interface TTSProvider {
  /** Speak text. Returns a promise that resolves when playback finishes. */
  speak(text: string, opts?: TTSSpeakOptions): Promise<void>;
  /** Stop any current playback. */
  stop(): void;
  /** Whether this provider is currently speaking. */
  readonly isSpeaking: boolean;
  /** Release resources. */
  dispose(): void;
}

export interface TTSSpeakOptions {
  voice?: string;
  rate?: number;
  pitch?: number;
}

export interface TTSVoiceInfo {
  id: string;
  name: string;
}
