// DOC: docs/internal/SEAMS.md#tts-provider-seam
export type TTSBackend = "kokoro" | "browser" | "none";

/** Describes which rich-playback features a provider supports. */
export interface TTSPlaybackCapabilities {
  canPause: boolean;
  canSeek: boolean;
  canAdjustSpeed: boolean;
  canAdjustVolume: boolean;
}

/** Snapshot of the current playback position and settings. */
export interface TTSPlaybackState {
  currentTime: number;
  duration: number | null;
  isPaused: boolean;
  playbackRate: number;
  volume: number;
  capabilities: TTSPlaybackCapabilities;
}

/** Callback invoked during playback with the current time and total duration. */
export type TTSPlaybackProgressCallback = (time: number, duration: number) => void;

export interface TTSProvider {
  /** Speak text. Returns a promise that resolves when playback finishes. */
  speak(text: string, opts?: TTSSpeakOptions): Promise<void>;
  /** Stop any current playback. */
  stop(): void;
  /** Whether this provider is currently speaking (includes paused state). */
  readonly isSpeaking: boolean;
  /** Release resources. */
  dispose(): void;
  /** Which rich-playback features this provider supports. */
  readonly capabilities: TTSPlaybackCapabilities;

  // --- Optional rich-playback methods ---

  /** Pause current playback. No-op if the provider does not support pausing. */
  pause?(): void;
  /** Resume paused playback. No-op if not paused. */
  resume?(): void;
  /** Seek to a position in seconds. No-op if the provider cannot seek. */
  seek?(seconds: number): void;
  /** Change the playback speed. No-op if the provider cannot adjust speed. */
  setPlaybackRate?(rate: number): void;
  /** Change the volume (0–1). No-op if the provider cannot adjust volume. */
  setVolume?(level: number): void;
  /** Return a snapshot of the current playback state. */
  getPlaybackState?(): TTSPlaybackState;
  /** Register a callback that fires during playback (~4 Hz from timeupdate). */
  onProgress?(callback: TTSPlaybackProgressCallback | null): void;
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
