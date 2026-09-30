import { useCallback, useSyncExternalStore } from "react";
import type { TTSPlaybackState } from "../../audio-integration";

// DOC: docs/reference/tts-api.md#playback-transport

/**
 * A pane's live playback position and settings. Each pane publishes its
 * provider's state when the provider reports a change (timeupdate, pause,
 * rate, volume, mute); nothing here runs on a timer.
 */
export type PlaybackTransport = TTSPlaybackState;

type TransportListener = (sessionId: string, transport: PlaybackTransport | null) => void;

const transports = new Map<string, PlaybackTransport>();
const listeners = new Set<TransportListener>();

/** Record a pane's transport (null when the pane goes away) and notify subscribers. */
export function publishTransport(sessionId: string, transport: PlaybackTransport | null): void {
  if (transport) transports.set(sessionId, transport);
  else if (!transports.delete(sessionId)) return;
  for (const listener of listeners) listener(sessionId, transport);
}

export function subscribeTransport(listener: TransportListener): () => void {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}

export function getTransport(sessionId: string | null): PlaybackTransport | null {
  return sessionId ? transports.get(sessionId) ?? null : null;
}

function useSessionSubscription(sessionId: string | null) {
  return useCallback((onChange: () => void) => subscribeTransport((changed) => {
    if (changed === sessionId) onChange();
  }), [sessionId]);
}

/** The pane's transport; the caller re-renders on each provider event. */
export function usePlaybackTransport(sessionId: string | null): PlaybackTransport | null {
  const subscribe = useSessionSubscription(sessionId);
  return useSyncExternalStore(subscribe, () => getTransport(sessionId));
}

/** Whether the pane's audio is paused; the caller re-renders only when that flips. */
export function usePlaybackPaused(sessionId: string | null): boolean {
  const subscribe = useSessionSubscription(sessionId);
  return useSyncExternalStore(subscribe, () => getTransport(sessionId)?.isPaused ?? false);
}
