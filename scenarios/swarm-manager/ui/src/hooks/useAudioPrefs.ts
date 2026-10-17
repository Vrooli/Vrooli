// Swarm-manager-local audio preferences (auto-speak, voice, speed).
//
// The voice/speed/summarize knobs that are shared across audio-tools
// adopters live in audio-tools' own settings (see audio-integration's
// getTTSConfig / updateTTSConfig). The auto-speak toggle is purely
// scenario-local, so we persist it in localStorage rather than adding a
// proto field. The Audio settings tab reads/writes this store and
// audio-tools' shared store side-by-side.
//
// Future: when swarm-manager grows a richer client-side preferences
// store (Zustand persist, etc.), migrate this slice into it.

import { useCallback, useEffect, useSyncExternalStore } from "react";

const STORAGE_KEY = "swarm-manager:audio-prefs:v1";

export interface AudioPrefs {
  autoSpeak: boolean;
}

const DEFAULT_PREFS: AudioPrefs = {
  autoSpeak: false,
};

let cached: AudioPrefs | null = null;
const listeners = new Set<() => void>();

function readStorage(): AudioPrefs {
  if (typeof window === "undefined") return DEFAULT_PREFS;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_PREFS;
    const parsed = JSON.parse(raw) as Partial<AudioPrefs>;
    return { autoSpeak: parsed.autoSpeak ?? DEFAULT_PREFS.autoSpeak };
  } catch {
    return DEFAULT_PREFS;
  }
}

function getSnapshot(): AudioPrefs {
  if (cached === null) cached = readStorage();
  return cached;
}

function notify() {
  cached = null;
  listeners.forEach((fn) => fn());
}

function subscribe(fn: () => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

export function setAudioPrefs(patch: Partial<AudioPrefs>): void {
  const next = { ...getSnapshot(), ...patch };
  if (typeof window !== "undefined") {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    } catch {
      // Swallow quota / private-mode failures; in-memory cache still updates.
    }
  }
  notify();
}

export function useAudioPrefs(): [AudioPrefs, (patch: Partial<AudioPrefs>) => void] {
  const prefs = useSyncExternalStore(subscribe, getSnapshot, () => DEFAULT_PREFS);

  // Cross-tab sync.
  useEffect(() => {
    if (typeof window === "undefined") return;
    const handler = (ev: StorageEvent) => {
      if (ev.key === STORAGE_KEY) notify();
    };
    window.addEventListener("storage", handler);
    return () => window.removeEventListener("storage", handler);
  }, []);

  const update = useCallback((patch: Partial<AudioPrefs>) => setAudioPrefs(patch), []);
  return [prefs, update];
}
