// Tiny external store for the slice of audio-tools' StreamConfig that the
// host scenario needs to drive its mic UI. audio-tools' stt_stream_config
// (`vad_silence_ms`) is the single source of truth; this store caches the
// last-hydrated values so consumers don't have to re-fetch per render.
//
// Hydration is performed by useHydrateVoiceConfig (mounted at app root).
// Until hydration completes the store returns the documented fallbacks,
// which match the VAD_FALLBACK_* constants in hooks/voice/vad.ts.
//
// Uses React's built-in useSyncExternalStore so this file has no external
// state-library dependency — the audio-integration folder stays portable.

import { useSyncExternalStore } from "react";

import {
  VAD_FALLBACK_SEGMENT_SILENCE_MS,
  VAD_FALLBACK_SILENCE_TIMEOUT_MS,
} from "./voice";

export interface VoiceConfigState {
  hydrated: boolean;
  vadSilenceTimeoutMs: number;
  segmentSilenceMs: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
}

interface ServerStreamConfigSlice {
  vadSilenceMs: number;
  segmentSilenceMs: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
}

let state: VoiceConfigState = {
  hydrated: false,
  vadSilenceTimeoutMs: VAD_FALLBACK_SILENCE_TIMEOUT_MS,
  segmentSilenceMs: VAD_FALLBACK_SEGMENT_SILENCE_MS,
  persistentMode: false,
  wakeWordEnabled: false,
};

const subscribers = new Set<() => void>();

function emit(): void {
  for (const fn of subscribers) fn();
}

function subscribe(fn: () => void): () => void {
  subscribers.add(fn);
  return () => {
    subscribers.delete(fn);
  };
}

function getSnapshot(): VoiceConfigState {
  return state;
}

/**
 * Hydrate from an audio-tools StreamConfig payload. vad_silence_ms is
 * authoritative; falls back to the legacy segment_silence_ms only when
 * the server hasn't set the new field.
 */
export function setVoiceConfigFromServer(cfg: ServerStreamConfigSlice): void {
  const silenceMs = cfg.vadSilenceMs > 0 ? cfg.vadSilenceMs : cfg.segmentSilenceMs;
  state = {
    hydrated: true,
    vadSilenceTimeoutMs: silenceMs > 0 ? silenceMs : VAD_FALLBACK_SILENCE_TIMEOUT_MS,
    segmentSilenceMs: silenceMs > 0 ? silenceMs : VAD_FALLBACK_SEGMENT_SILENCE_MS,
    persistentMode: cfg.persistentMode,
    wakeWordEnabled: cfg.wakeWordEnabled,
  };
  emit();
}

/** Test-only: reset to fallback values between specs. */
export function _resetVoiceConfigForTesting(): void {
  state = {
    hydrated: false,
    vadSilenceTimeoutMs: VAD_FALLBACK_SILENCE_TIMEOUT_MS,
    segmentSilenceMs: VAD_FALLBACK_SEGMENT_SILENCE_MS,
    persistentMode: false,
    wakeWordEnabled: false,
  };
  emit();
}

/**
 * Hook returning the slice of voice config the caller selects.
 * Modeled on zustand's selector signature so existing call sites that
 * pattern-match `useVoiceConfigStore((s) => s.foo)` work unchanged.
 */
export function useVoiceConfigStore<T>(selector: (s: VoiceConfigState) => T): T {
  return useSyncExternalStore(
    subscribe,
    () => selector(getSnapshot()),
    () => selector(getSnapshot()),
  );
}

/** Direct accessor for non-React callers (e.g. hydration hook). */
useVoiceConfigStore.getState = (): VoiceConfigState & { setFromServer: typeof setVoiceConfigFromServer } => ({
  ...state,
  setFromServer: setVoiceConfigFromServer,
});
