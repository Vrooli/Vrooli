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

  /**
   * Attempt to unlock audio playback by running a silent play() from inside a
   * user-gesture call stack. Resolves `true` if the underlying media element
   * is now activated, `false` if the browser rejected the play (caller should
   * then show the enable-audio affordance). Must be safe to call multiple
   * times and must never throw.
   *
   * When `force` is true, re-plays the silent blob even if a previous unlock
   * already succeeded. Use this for explicit user actions (e.g. the
   * Enable-Audio banner click) where the browser may have since dropped the
   * element's activation and we need a fresh gesture-scoped play. Keystroke
   * / pointer preemptive unlocks should use the default (false) to avoid
   * thrashing the media element during typing.
   */
  unlock(force?: boolean): Promise<boolean>;
  /** Whether a prior unlock() call succeeded. */
  isUnlocked(): boolean;

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
  /**
   * Speak multiple texts as a single unified audio stream.
   * Providers that support this synthesize all segments up front, concatenate
   * the audio, and play it as one track — giving accurate total duration and
   * seek across the entire sequence.
   *
   * When absent, the caller falls back to sequential `speak()` calls.
   */
  speakSequence?(texts: string[], opts?: TTSSpeakOptions): Promise<void>;
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
